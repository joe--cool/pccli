package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/joe--cool/pccli/internal/planningcenter"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type request struct {
	method, path string
	query        url.Values
	body         any
}
type fixture struct {
	t               *testing.T
	requests        []request
	handle          func(request) (string, error)
	writes, uploads int
}

func (f *fixture) call(method, path string, q url.Values, body, dst any) error {
	f.t.Helper()
	r := request{method, path, cloneQuery(q), body}
	f.requests = append(f.requests, r)
	if method != "GET" {
		f.writes++
	}
	s, e := f.handle(r)
	if e != nil {
		return e
	}
	if dst != nil {
		return json.Unmarshal([]byte(s), dst)
	}
	return nil
}
func (f *fixture) Get(c context.Context, p string, q url.Values, d any) error {
	return f.call("GET", p, q, nil, d)
}
func (f *fixture) Post(c context.Context, p string, q url.Values, b, d any) error {
	return f.call("POST", p, q, b, d)
}
func (f *fixture) Patch(c context.Context, p string, q url.Values, b, d any) error {
	return f.call("PATCH", p, q, b, d)
}
func (f *fixture) Delete(c context.Context, p string, q url.Values) error {
	return f.call("DELETE", p, q, nil, nil)
}
func (f *fixture) UploadFile(c context.Context, p string, d any) error {
	f.uploads++
	return f.call("UPLOAD", p, nil, nil, d)
}
func songJSON(id, title string, hidden bool) string {
	return fmt.Sprintf(`{"type":"Song","id":%q,"attributes":{"title":%q,"hidden":%t,"author":"Old"}}`, id, title, hidden)
}
func songPage(items, next string) string {
	return `{"data":[` + items + `],"links":{"next":` + fmt.Sprintf("%q", next) + `}}`
}
func assertCode(t *testing.T, e error, want string) {
	t.Helper()
	var se *Error
	if !errors.As(e, &se) || se.Code != want {
		t.Fatalf("error = %v, want code %s", e, want)
	}
}
func TestResolutionTraversesBeforeWriting(t *testing.T) {
	for _, mode := range []string{"duplicate", "failure", "malformed", "hostile", "changed_filter", "cycle"} {
		t.Run(mode, func(t *testing.T) {
			f := &fixture{t: t}
			f.handle = func(r request) (string, error) {
				if r.method != "GET" {
					t.Fatal("unexpected side effect")
				}
				if r.query.Get("offset") == "" {
					next := "/services/v2/songs?per_page=100&where%5Bhidden%5D=false&offset=100"
					if mode == "hostile" {
						next = "https://evil.example" + next
					}
					if mode == "changed_filter" {
						next = strings.ReplaceAll(next, "false", "true")
					}
					return songPage(songJSON("1", "Grace", false), next), nil
				}
				switch mode {
				case "failure":
					return "", errors.New("page failed")
				case "malformed":
					return `{"data":null}`, nil
				case "cycle":
					return songPage(songJSON("1", "Grace", false), "/services/v2/songs?per_page=100&where%5Bhidden%5D=false&offset=100"), nil
				default:
					return songPage(songJSON("2", "Grace", false), ""), nil
				}
			}
			_, e := NewLibrary(f).UpdateSong(context.Background(), "Grace", Visible, SongPatch{Author: ptr("New")})
			if mode == "duplicate" {
				assertCode(t, e, "ambiguous_target")
			} else {
				assertCode(t, e, "incomplete_lookup")
			}
			if f.writes != 0 || f.uploads != 0 {
				t.Fatal("side effect before complete resolution")
			}
		})
	}
}
func ptr[T any](v T) *T { return &v }
func TestExactWinsAfterPartialCandidates(t *testing.T) {
	f := &fixture{t: t}
	f.handle = func(r request) (string, error) {
		if r.query.Get("offset") == "" {
			return songPage(songJSON("1", "Grace One", false)+","+songJSON("2", "Grace Two", false), "/services/v2/songs?per_page=100&where%5Bhidden%5D=false&offset=100"), nil
		}
		return songPage(songJSON("3", "Grace", false), ""), nil
	}
	r, e := NewLibrary(f).song(context.Background(), "Grace", Visible, false)
	if e != nil || r.id != "3" {
		t.Fatalf("got %v %v", r, e)
	}
}
func TestSelectors(t *testing.T) {
	for _, s := range []string{"0", "000", "-2", "+2", "id:abc", "id:", "name:"} {
		_, _, _, e := ParseSelector(s)
		if e == nil {
			t.Errorf("accepted %q", s)
		}
	}
	id, _, _, e := ParseSelector("99999999999999999999999999999999")
	if e != nil || id == "" {
		t.Fatal("huge ID fell back to name")
	}
	id, name, exact, e := ParseSelector("name:123")
	if e != nil || id != "" || name != "123" || !exact {
		t.Fatal("numeric name failed")
	}
}
func TestVisibilityIDCannotBypassSelection(t *testing.T) {
	f := &fixture{t: t, handle: func(r request) (string, error) { return `{"data":` + songJSON("1", "Grace", true) + `}`, nil }}
	_, e := NewLibrary(f).UpdateSong(context.Background(), "1", Visible, SongPatch{Hidden: ptr(false)})
	assertCode(t, e, "not_found")
	if f.writes != 0 {
		t.Fatal("hidden song mutated")
	}
}
func TestPatchPreservesEmptyFalseZeroAndArray(t *testing.T) {
	for _, tc := range []struct {
		name   string
		attrs  map[string]any
		result string
	}{{"song", fields(SongPatch{Author: ptr(""), Hidden: ptr(false)}), `{"author":"","hidden":false}`}, {"arrangement", fields(ArrangementPatch{Notes: ptr(""), Length: ptr(0), LyricsEnabled: ptr(false), Sequence: ptr([]string{})}), `{"notes":"","length":0,"lyrics_enabled":false,"sequence":[]}`}} {
		t.Run(tc.name, func(t *testing.T) {
			if e := validate(tc.attrs, "Song"); e != nil {
				t.Fatal(e)
			}
			b, _ := json.Marshal(tc.attrs)
			var got, want any
			_ = json.Unmarshal(b, &got)
			_ = json.Unmarshal([]byte(tc.result), &want)
			gb, _ := json.Marshal(got)
			wb, _ := json.Marshal(want)
			if string(gb) != string(wb) {
				t.Fatalf("payload %s want %s", gb, wb)
			}
		})
	}
}
func TestVerificationRejectsIgnoredClear(t *testing.T) {
	r := record{attrs: map[string]json.RawMessage{"author": json.RawMessage(`"Old"`)}}
	assertCode(t, verify(r, map[string]any{"author": ""}), "verification_failed")
}
func TestInvalidMutationHasNoNetwork(t *testing.T) {
	f := &fixture{t: t, handle: func(r request) (string, error) { t.Fatal("unexpected request"); return "", nil }}
	_, e := NewLibrary(f).UpdateSong(context.Background(), "1", Visible, SongPatch{Title: ptr("")})
	assertCode(t, e, "invalid_input")
	if len(f.requests) != 0 {
		t.Fatal("request issued")
	}
}
func TestWrongParentStopsBeforeUpload(t *testing.T) {
	f := &fixture{t: t}
	f.handle = func(r request) (string, error) {
		if r.path == "/services/v2/songs/1" {
			return `{"data":` + songJSON("1", "Grace", false) + `}`, nil
		}
		return `{"data":{"type":"Arrangement","id":"2","attributes":{"name":"Default"},"relationships":{"song":{"data":{"type":"Song","id":"999"}}}}}`, nil
	}
	_, e := NewLibrary(f).AddAttachment(context.Background(), Scope{Song: "1", Arrangement: "2"}, AttachmentCreate{FilePath: tempChart(t)})
	assertCode(t, e, "invalid_response")
	if f.uploads != 0 || f.writes != 0 {
		t.Fatal("wrong parent had side effects")
	}
}
func TestBoundMetadata(t *testing.T) {
	for _, n := range []int{1, 25, 101, 1000} {
		data := make([]Song, n+1)
		r, e := bound(data, ListOptions{Limit: n})
		if e != nil || r.Meta.Returned != n || !r.Meta.HasMore || r.Meta.Complete {
			t.Fatal("bad bound metadata")
		}
	}
	r, _ := bound([]Song{}, ListOptions{All: true})
	if r.Data == nil || !r.Meta.Complete || r.Meta.HasMore {
		t.Fatal("bad empty result")
	}
}

func TestRepeatedIdentityDeduplicatesButChangedStateFails(t *testing.T) {
	for _, different := range []bool{false, true} {
		t.Run(fmt.Sprint(different), func(t *testing.T) {
			f := &fixture{t: t}
			f.handle = func(r request) (string, error) {
				if r.query.Get("offset") == "" {
					return songPage(songJSON("1", "Grace", false), "/services/v2/songs?per_page=100&where%5Bhidden%5D=false&offset=100"), nil
				}
				name := "Grace"
				if different {
					name = "Changed"
				}
				return songPage(songJSON("1", name, false), ""), nil
			}
			rows, e := NewLibrary(f).songs(context.Background(), Visible)
			if different {
				assertCode(t, e, "concurrent_change")
			} else if e != nil || len(rows) != 1 {
				t.Fatal("identity not deduplicated", rows, e)
			}
		})
	}
}
func TestAllVisibilityUnionDoesNotRelyOnOmittedFilter(t *testing.T) {
	f := &fixture{t: t}
	f.handle = func(r request) (string, error) {
		switch r.query.Get("where[hidden]") {
		case "false":
			return songPage(songJSON("1", "Grace", false), ""), nil
		case "true":
			return songPage(songJSON("2", "Grace", true), ""), nil
		default:
			t.Fatal("visibility filter omitted")
			return "", nil
		}
	}
	_, e := NewLibrary(f).song(context.Background(), "Grace", All, true)
	assertCode(t, e, "ambiguous_target")
	if f.writes != 0 {
		t.Fatal("ambiguous mutation")
	}
}
func TestAttachmentUploadReceiptSurvivesCreateFailure(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		t.Run(fmt.Sprint(unknown), func(t *testing.T) {
			f := &fixture{t: t}
			f.handle = func(r request) (string, error) {
				switch r.method {
				case "UPLOAD":
					return `{"data":[{"id":"uuid-file","type":"File","attributes":{"name":"chart.pdf","expires_at":"tomorrow"}}]}`, nil
				case "POST":
					if unknown {
						return "", errors.New("lost response")
					}
					return "", planningcenter.APIError{StatusCode: 422}
				case "GET":
					return `{"data":` + songJSON("1", "Grace", false) + `}`, nil
				}
				t.Fatal("unexpected request", r)
				return "", nil
			}
			result, e := NewLibrary(f).AddAttachment(context.Background(), Scope{Song: "1"}, AttachmentCreate{FilePath: tempChart(t)})
			want := "rejected"
			if unknown {
				want = "unknown"
			}
			wantCode := "partial_success"
			if unknown {
				wantCode = "outcome_unknown"
			}
			assertCode(t, e, wantCode)
			if result.Upload == nil || result.Upload.ID != "uuid-file" || result.Upload.ExpiresAt != "tomorrow" || result.Outcome != want {
				t.Fatal("receipt lost", result)
			}
			posts := 0
			for _, r := range f.requests {
				if r.method == "POST" {
					posts++
				}
			}
			if posts != 1 || f.uploads != 1 {
				t.Fatal("write replayed")
			}
		})
	}
}
func TestArchivedSelectionRequiresExplicitScope(t *testing.T) {
	f := &fixture{t: t}
	f.handle = func(r request) (string, error) {
		if r.path == "/services/v2/songs/1" {
			return `{"data":` + songJSON("1", "Grace", false) + `}`, nil
		}
		return `{"data":{"type":"Arrangement","id":"2","attributes":{"name":"Old","archived_at":"2026-01-01T00:00:00Z"}}}`, nil
	}
	l := NewLibrary(f)
	_, e := l.parents(context.Background(), Scope{Song: "1", Arrangement: "2"}, true, false)
	assertCode(t, e, "not_found")
	_, e = l.parents(context.Background(), Scope{Song: "1", Arrangement: "2", IncludeArchived: true}, true, false)
	if e != nil {
		t.Fatal(e)
	}
	if f.requests[len(f.requests)-1].path != "/services/v2/songs/1/unscoped_arrangements/2" {
		t.Fatal("archive selection did not use unscoped path")
	}
}
func TestLiteralSearchDoesNotUseUnverifiedServerWildcards(t *testing.T) {
	f := &fixture{t: t}
	f.handle = func(r request) (string, error) {
		if r.query.Get("where[title]") != "" {
			t.Fatal("unsafe narrowing predicate")
		}
		return songPage(songJSON("1", "100% Gráce", false)+","+songJSON("2", "100 Gráce", false), ""), nil
	}
	result, e := NewLibrary(f).ListSongs(context.Background(), SongListOptions{Search: "% grÁce"})
	if e != nil || len(result.Data) != 1 || result.Data[0].ID != "1" {
		t.Fatalf("literal search got %#v %v", result, e)
	}
}

func TestUnknownHiddenStateCannotProveVisible(t *testing.T) {
	f := &fixture{t: t, handle: func(r request) (string, error) {
		return `{"data":{"type":"Song","id":"1","attributes":{"title":"Grace","hidden":null}}}`, nil
	}}
	_, e := NewLibrary(f).UpdateSong(context.Background(), "1", Visible, SongPatch{Author: ptr("New")})
	assertCode(t, e, "invalid_response")
	if f.writes != 0 {
		t.Fatal("unknown visibility mutated")
	}
}
func TestRefreshedChildScopeCannotChange(t *testing.T) {
	f := &fixture{t: t}
	arrGets := 0
	f.handle = func(r request) (string, error) {
		if r.path == "/services/v2/songs/1" {
			return `{"data":` + songJSON("1", "Grace", false) + `}`, nil
		}
		arrGets++
		parent := "1"
		if arrGets > 1 {
			parent = "999"
		}
		return `{"data":{"type":"Arrangement","id":"2","attributes":{"name":"Default"},"relationships":{"song":{"data":{"type":"Song","id":"` + parent + `"}}}}}`, nil
	}
	_, e := NewLibrary(f).UpdateArrangement(context.Background(), Scope{Song: "1"}, "2", ArrangementPatch{Notes: ptr("New")})
	assertCode(t, e, "invalid_response")
	if f.writes != 0 {
		t.Fatal("changed scope mutated")
	}
}

func TestBoundedListStopsAfterValidatedLookahead(t *testing.T) {
	f := &fixture{t: t}
	f.handle = func(r request) (string, error) {
		if r.query.Get("offset") != "" {
			t.Fatal("bounded list fetched unnecessary next page")
		}
		return songPage(songJSON("1", "One", false)+","+songJSON("2", "Two", false), "/services/v2/songs?per_page=100&where%5Bhidden%5D=false&offset=100"), nil
	}
	r, e := NewLibrary(f).ListSongs(context.Background(), SongListOptions{ListOptions: ListOptions{Limit: 1}})
	if e != nil || len(r.Data) != 1 || !r.Meta.HasMore || r.Meta.Complete {
		t.Fatalf("unexpected bounded result %#v %v", r, e)
	}
	if len(f.requests) != 1 {
		t.Fatal("unnecessary requests")
	}
}
func TestBoundedCollectorFiltersBeforeCounting(t *testing.T) {
	f := &fixture{t: t}
	f.handle = func(r request) (string, error) {
		if r.query.Get("offset") == "" {
			return `{"data":[{"type":"Arrangement","id":"1","attributes":{"name":"Old","archived_at":"2020-01-01"}},{"type":"Arrangement","id":"2","attributes":{"name":"Current"}}],"links":{"next":"/services/v2/songs/1/arrangements?per_page=100&offset=100"}}`, nil
		}
		return `{"data":[{"type":"Arrangement","id":"3","attributes":{"name":"Another"}}]}`, nil
	}
	rows, e := NewLibrary(f).api.collectLimited(context.Background(), "/services/v2/songs/1/arrangements", "Arrangement", nil, 1, func(r record) (bool, error) { return !isArchived(r), nil })
	if e != nil || len(rows) != 2 || rows[0].id != "2" || rows[1].id != "3" {
		t.Fatal("limit counted excluded records", rows, e)
	}
}
func TestMalformedKnownAttributePreventsWrite(t *testing.T) {
	f := &fixture{t: t}
	f.handle = func(r request) (string, error) {
		if r.path == "/services/v2/songs/1" {
			return `{"data":` + songJSON("1", "Grace", false) + `}`, nil
		}
		return `{"data":{"type":"Arrangement","id":"2","attributes":{"name":"Default","bpm":"invalid"}}}`, nil
	}
	_, e := NewLibrary(f).UpdateArrangement(context.Background(), Scope{Song: "1"}, "2", ArrangementPatch{Notes: ptr("New")})
	assertCode(t, e, "invalid_response")
	if f.writes != 0 {
		t.Fatal("malformed lookup caused a write")
	}
}
func TestRefreshedKeyScopeCannotChange(t *testing.T) {
	f := &fixture{t: t}
	keyGets := 0
	f.handle = func(r request) (string, error) {
		switch r.path {
		case "/services/v2/songs/1":
			return `{"data":` + songJSON("1", "Grace", false) + `}`, nil
		case "/services/v2/songs/1/arrangements/2":
			return `{"data":{"type":"Arrangement","id":"2","attributes":{"name":"Default"}}}`, nil
		default:
			keyGets++
			parent := "2"
			if keyGets > 1 {
				parent = "999"
			}
			return `{"data":{"type":"Key","id":"3","attributes":{"name":"G"},"relationships":{"arrangement":{"data":{"type":"Arrangement","id":"` + parent + `"}}}}}`, nil
		}
	}
	_, e := NewLibrary(f).UpdateKey(context.Background(), Scope{Song: "1", Arrangement: "2"}, "3", KeyPatch{Name: ptr("New")})
	assertCode(t, e, "invalid_response")
	if f.writes != 0 {
		t.Fatal("changed key scope caused a write")
	}
}

func TestScopedModelsExposeCanonicalParents(t *testing.T) {
	f := &fixture{t: t}
	f.handle = func(r request) (string, error) {
		return `{"data":{"type":"Key","id":"3","attributes":{"name":"G"}}}`, nil
	}
	r, e := NewLibrary(f).api.get(context.Background(), "/services/v2/songs/1/arrangements/2/keys/3", "Key", "3")
	if e != nil {
		t.Fatal(e)
	}
	var key Key
	if e = r.model(&key); e != nil {
		t.Fatal(e)
	}
	if key.ID != "3" || key.SongID != "1" || key.ArrangementID != "2" {
		t.Fatalf("parent IDs missing: %#v", key)
	}
}
func TestContradictoryWriteResponseIsUnknownOutcome(t *testing.T) {
	f := &fixture{t: t, handle: func(r request) (string, error) {
		return `{"data":{"type":"Arrangement","id":"2","attributes":{"name":"Default"},"relationships":{"song":{"data":{"type":"Song","id":"999"}}}}}`, nil
	}}
	_, e := NewLibrary(f).api.mutate(context.Background(), "PATCH", "/services/v2/songs/1/arrangements/2", "Arrangement", map[string]any{"name": "Default"})
	assertCode(t, e, "outcome_unknown")
	if f.writes != 1 {
		t.Fatal("expected exactly the submitted write")
	}
}

func tempChart(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "chart.pdf")
	if err := os.WriteFile(path, []byte("synthetic chart"), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}
