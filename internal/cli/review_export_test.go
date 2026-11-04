package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func assertReviewPrivateExportFiles(t *testing.T, root string) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		want := os.FileMode(0600)
		if entry.IsDir() {
			want = 0700
		}
		if info.Mode().Perm() != want {
			t.Errorf("temporary export entry %s mode %o want %o", entry.Name(), info.Mode().Perm(), want)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
func assertReviewExportCleanup(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("export files leaked: %v %v", entries, err)
	}
}
func reviewExportPage(items []string, path string, query url.Values, next bool) string {
	links := ""
	if next {
		q := url.Values{}
		for k, v := range query {
			q[k] = append([]string(nil), v...)
		}
		q.Set("offset", "100")
		encoded, _ := json.Marshal(path + "?" + q.Encode())
		links = `,"links":{"next":` + string(encoded) + `}`
	}
	return `{"data":[` + strings.Join(items, ",") + `]` + links + `}`
}
func TestReviewCompleteExportStagesLateFailureAndCancellation(t *testing.T) {
	for _, mode := range []string{"late_failure", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			tmp := t.TempDir()
			t.Setenv("TMPDIR", tmp)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			library := reviewLibrary(func(r *http.Request) (*http.Response, error) {
				calls++
				assertReviewPrivateExportFiles(t, tmp)
				if r.Method != "GET" || r.URL.Path != "/services/v2/songs" {
					t.Fatalf("unexpected request %s %s", r.Method, r.URL)
				}
				if r.URL.Query().Get("offset") != "" {
					if mode == "cancel" {
						cancel()
						return nil, context.Canceled
					}
					return reviewResponse(200, `{}`), nil
				}
				return reviewResponse(200, reviewExportPage([]string{`{"id":"1","type":"Song","attributes":{"title":"Grace","hidden":false}}`}, r.URL.Path, r.URL.Query(), true)), nil
			})
			command := injectedCommand(library)
			command.SetContext(withApp(ctx, appFrom(command)))
			out, stderr, err := runCommand(command, "services", "songs", "list", "--all", "--json", "--order", "title")
			if err == nil || out != "" || calls != 2 {
				t.Fatalf("out %q stderr %q err %v calls %d", out, stderr, err, calls)
			}
			if mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatalf("cancel not propagated: %v", err)
			}
			decodeOutput(t, stderr)
			assertReviewExportCleanup(t, tmp)
		})
	}
}
func TestReviewSortedAllVisibilityExportMatchesBoundedList(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	library := reviewLibrary(func(r *http.Request) (*http.Response, error) {
		assertReviewPrivateExportFiles(t, tmp)
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		hidden := r.URL.Query().Get("where[hidden]") == "true"
		items := []string{}
		names := []string{"Zulu", "alpha", "Alpha", "ßeta", "άλφα"}
		for i := offset; i < min(offset+100, 117); i++ {
			id := i + 1
			if hidden {
				id += 117
			}
			items = append(items, fmt.Sprintf(`{"id":%q,"type":"Song","attributes":{"title":%q,"hidden":%t}}`, strconv.Itoa(id), names[i%len(names)], hidden))
		}
		return reviewResponse(200, reviewExportPage(items, r.URL.Path, r.URL.Query(), offset == 0)), nil
	})
	ids := func(args ...string) []string {
		out, stderr, err := runCommand(injectedCommand(library), append([]string{"services", "songs", "list", "--visibility", "all", "--order", "-title", "--json"}, args...)...)
		if err != nil {
			t.Fatalf("%v %s", err, stderr)
		}
		doc := decodeOutput(t, out)
		var data []struct {
			ID string `json:"id"`
		}
		if err = json.Unmarshal(doc["data"], &data); err != nil {
			t.Fatal(err)
		}
		result := []string{}
		for _, song := range data {
			result = append(result, song.ID)
		}
		return result
	}
	want := ids("--limit", "1000")
	got := ids("--all")
	if len(got) != 234 || !reflect.DeepEqual(got, want) {
		t.Fatalf("streamed global order differs (%d vs %d)", len(got), len(want))
	}
	assertReviewExportCleanup(t, tmp)
}

func TestReviewExportInheritsVisibilityAndAttachmentParentChecks(t *testing.T) {
	for _, mode := range []string{"hidden_song", "wrong_attachable"} {
		t.Run(mode, func(t *testing.T) {
			childReads := 0
			library := reviewLibrary(func(r *http.Request) (*http.Response, error) {
				if r.Method != "GET" {
					t.Fatalf("unexpected side effect %s", r.Method)
				}
				if r.URL.Path == "/services/v2/songs/1" {
					hidden := mode == "hidden_song"
					return reviewResponse(200, fmt.Sprintf(`{"data":{"id":"1","type":"Song","attributes":{"title":"Grace","hidden":%t}}}`, hidden)), nil
				}
				if r.URL.Path != "/services/v2/songs/1/attachments" {
					t.Fatalf("unexpected request %s", r.URL)
				}
				childReads++
				return reviewResponse(200, `{"data":[{"id":"30","type":"Attachment","attributes":{"filename":"chart.pdf"},"relationships":{"attachable":{"data":{"type":"Song","id":"999"}},"attachment_types":{"data":[]}}}]}`), nil
			})
			out, stderr, err := runCommand(injectedCommand(library), "services", "attachments", "list", "--song", "1", "--all", "--json")
			if err == nil || out != "" {
				t.Fatalf("inherited selection bypassed: %v %s", err, out)
			}
			if mode == "hidden_song" && childReads != 0 {
				t.Fatal("hidden parent allowed child reads")
			}
			decodeOutput(t, stderr)
		})
	}
}

func TestReviewSparseAttachmentReadbackCannotChangeParent(t *testing.T) {
	writes := 0
	library := reviewLibrary(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/services/v2/songs/1" {
			return reviewResponse(200, reviewSong), nil
		}
		if r.URL.Path == "/services/v2/songs/1/attachments" {
			return reviewResponse(200, `{"data":[{"id":"30","type":"Attachment","attributes":{"filename":"chart.pdf"}}]}`), nil
		}
		if r.URL.Path != "/services/v2/songs/1/attachments/30" {
			t.Fatalf("unexpected %s", r.URL)
		}
		if r.Method == "PATCH" {
			writes++
		}
		relations := ""
		extraAttrs := ""
		if r.URL.Query().Get("fields[Attachment]") != "" {
			extraAttrs = `,"file_upload_identifier":"known-uuid"`
			relations = `,"relationships":{"attachable":{"data":{"type":"Song","id":"999"}},"attachment_types":{"data":[]}}`
		}
		return reviewResponse(200, `{"data":{"id":"30","type":"Attachment","attributes":{"filename":"chart.pdf"`+extraAttrs+`}`+relations+`}}`), nil
	})
	out, stderr, err := runCommand(injectedCommand(library), "services", "attachments", "update", "30", "--song", "1", "--upload-id", "known-uuid", "--json")
	if err == nil || out != "" || writes != 1 {
		t.Fatalf("wrong parent reported verified: out %q err %v writes %d", out, err, writes)
	}
	doc := decodeOutput(t, stderr)
	var body map[string]any
	if json.Unmarshal(doc["error"], &body) != nil || body["code"] != "verification_failed" {
		t.Fatal(stderr)
	}
}
