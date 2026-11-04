package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func exportSongFixture(t *testing.T, count int, extra func(int, bool) string) *fixture {
	f := &fixture{t: t}
	f.handle = func(r request) (string, error) {
		if r.method != "GET" || r.path != "/services/v2/songs" {
			t.Fatalf("unexpected request %#v", r)
		}
		offset, _ := strconv.Atoi(r.query.Get("offset"))
		hidden := r.query.Get("where[hidden]") == "true"
		items := []string{}
		for i := offset; i < min(offset+100, count); i++ {
			id := i + 1
			if hidden {
				id += count
			}
			attrs := ""
			if extra != nil {
				attrs = extra(i, hidden)
			}
			items = append(items, fmt.Sprintf(`{"type":"Song","id":%q,"attributes":{"title":%q,"hidden":%t%s}}`, strconv.Itoa(id), fmt.Sprintf("Song %06d", i), hidden, attrs))
		}
		next := ""
		if offset+100 < count {
			q := cloneQuery(r.query)
			q.Set("offset", strconv.Itoa(offset+100))
			next = r.path + "?" + q.Encode()
		}
		return songPage(strings.Join(items, ","), next), nil
	}
	return f
}
func TestWalkSongsReportsLateFailureWithoutSuccessMetadata(t *testing.T) {
	f := &fixture{t: t}
	emitted := 0
	f.handle = func(r request) (string, error) {
		if r.query.Get("offset") != "" {
			if emitted != 1 {
				t.Fatal("did not stream first page before fetching next")
			}
			return "", errors.New("late page failure")
		}
		return songPage(songJSON("1", "Grace", false), "/services/v2/songs?per_page=100&where%5Bhidden%5D=false&offset=100"), nil
	}
	meta, err := NewLibrary(f).WalkSongs(context.Background(), SongListOptions{ListOptions: ListOptions{All: true}}, func(Song) error { emitted++; return nil })
	if err == nil || meta.Complete || meta.Returned != 1 || emitted != 1 {
		t.Fatalf("unexpected partial result %#v %v", meta, err)
	}
	if f.writes != 0 || f.uploads != 0 {
		t.Fatal("export performed side effects")
	}
}
func TestWalkSongsValidatesCursorBeforeEmittingPage(t *testing.T) {
	f := &fixture{t: t, handle: func(r request) (string, error) {
		return songPage(songJSON("1", "Grace", false), "https://evil.example/services/v2/songs?offset=100&per_page=100&where%5Bhidden%5D=false"), nil
	}}
	emitted := 0
	_, err := NewLibrary(f).WalkSongs(context.Background(), SongListOptions{}, func(Song) error { emitted++; return nil })
	if err == nil || emitted != 0 || len(f.requests) != 1 {
		t.Fatal("untrusted cursor accepted")
	}
}
func TestWalkCallbackFailureAndCancellationStopRequests(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelled), func(t *testing.T) {
			f := exportSongFixture(t, 201, nil)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			failure := errors.New("private spool failed")
			meta, err := NewLibrary(f).WalkSongs(ctx, SongListOptions{}, func(Song) error {
				if cancelled {
					cancel()
					return nil
				}
				return failure
			})
			if cancelled {
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("want cancellation, got %v", err)
				}
			} else if !errors.Is(err, failure) {
				t.Fatal("callback error lost", err)
			}
			if meta.Complete || len(f.requests) != 1 {
				t.Fatal("continued export after failure")
			}
		})
	}
}
func TestWalkSongsLiteralFiltersAndVisibility(t *testing.T) {
	f := &fixture{t: t}
	f.handle = func(r request) (string, error) {
		hidden := r.query.Get("where[hidden]") == "true"
		id := "1"
		if hidden {
			id = "2"
		}
		return songPage(songJSON(id, "100% Gráce", hidden), ""), nil
	}
	ids := []string{}
	meta, err := NewLibrary(f).WalkSongs(context.Background(), SongListOptions{Search: "% GRÁCE", Visibility: All}, func(song Song) error { ids = append(ids, song.ID); return nil })
	if err != nil || !meta.Complete || meta.Returned != 2 || !reflect.DeepEqual(ids, []string{"1", "2"}) {
		t.Fatal(ids, meta, err)
	}
	for _, r := range f.requests {
		if r.query.Get("where[title]") != "" {
			t.Fatal("unproved wildcard narrowing")
		}
	}
}
func TestWalkRetainsHashesInsteadOfResourcePayloads(t *testing.T) {
	const count = 600
	payload := strings.Repeat("x", 64<<10)
	f := exportSongFixture(t, count, func(int, bool) string { return `,"notes":` + strconv.Quote(payload) })
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	meta, err := NewLibrary(f).WalkSongs(context.Background(), SongListOptions{}, func(song Song) error {
		if song.ID == strconv.Itoa(count) {
			runtime.GC()
			var current runtime.MemStats
			runtime.ReadMemStats(&current)
			growth := int64(current.Alloc) - int64(before.Alloc)
			if growth > 16<<20 {
				t.Fatalf("export retained resource payloads: live heap grew %d bytes", growth)
			}
		}
		return nil
	})
	if err != nil || meta.Returned != count || !meta.Complete {
		t.Fatal(meta, err)
	}
}
func TestSortedWalkMatchesListOrderingWithBoundedMerge(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	titles := []string{"zulu", "Alpha", "álpha", "βeta", "alpha", "Zero"}
	extra := func(i int, hidden bool) string {
		shift := 0
		if hidden {
			shift = 1
		}
		return fmt.Sprintf(`,"title":%q,"created_at":%q,"last_scheduled_at":%q`, titles[(i+shift)%len(titles)], fmt.Sprintf("2026-01-%02d", 1+(i%28)), fmt.Sprintf("2026-02-%02d", 1+(i%27)))
	}
	for _, order := range []string{"title", "-title", "created_at", "-last_scheduled_at"} {
		t.Run(order, func(t *testing.T) {
			f := exportSongFixture(t, 850, extra)
			l := NewLibrary(f)
			opts := SongListOptions{Visibility: All, Order: order, ListOptions: ListOptions{All: true}}
			want, err := l.ListSongs(context.Background(), opts)
			if err != nil {
				t.Fatal(err)
			}
			got := []string{}
			meta, err := l.WalkSongs(context.Background(), opts, func(song Song) error { got = append(got, song.ID); return nil })
			if err != nil {
				t.Fatal(err)
			}
			ids := make([]string, len(want.Data))
			for i, song := range want.Data {
				ids[i] = song.ID
			}
			if !meta.Complete || meta.Returned != len(ids) || !reflect.DeepEqual(got, ids) {
				t.Fatalf("stream order differs from list (%s)", order)
			}
			entries, err := os.ReadDir(tmp)
			if err != nil || len(entries) != 0 {
				t.Fatalf("sort temporary files leaked: %v %v", entries, err)
			}
		})
	}
}
func TestSortedWalkCleansRunsOnFailureAndCancellation(t *testing.T) {
	for _, mode := range []string{"late_page", "callback", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			tmp := t.TempDir()
			t.Setenv("TMPDIR", tmp)
			f := exportSongFixture(t, 250, nil)
			original := f.handle
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			failure := errors.New("injected failure")
			if mode == "late_page" {
				f.handle = func(r request) (string, error) {
					if r.query.Get("offset") == "200" {
						return "", failure
					}
					return original(r)
				}
			}
			_, err := NewLibrary(f).WalkSongs(ctx, SongListOptions{Order: "title"}, func(Song) error {
				if mode == "cancel" {
					cancel()
					return nil
				}
				return failure
			})
			if err == nil {
				t.Fatal("expected failure")
			}
			entries, readErr := os.ReadDir(tmp)
			if readErr != nil || len(entries) != 0 {
				t.Fatalf("sort files leaked: %v %v", entries, readErr)
			}
		})
	}
}
func TestWalkAllTypedCollectionsAndParentIDs(t *testing.T) {
	f := &fixture{t: t}
	f.handle = func(r request) (string, error) {
		if r.method != "GET" {
			t.Fatal("write during export")
		}
		switch r.path {
		case "/services/v2/songs/1":
			return `{"data":` + songJSON("1", "Grace", false) + `}`, nil
		case "/services/v2/songs/1/arrangements/2":
			return `{"data":{"type":"Arrangement","id":"2","attributes":{"name":"Default"}}}`, nil
		case "/services/v2/songs/1/arrangements/2/keys/3":
			return `{"data":{"type":"Key","id":"3","attributes":{"name":"G"}}}`, nil
		case "/services/v2/songs/1/arrangements":
			return `{"data":[{"type":"Arrangement","id":"2","attributes":{"name":"Default"}}]}`, nil
		case "/services/v2/songs/1/arrangements/2/keys":
			return `{"data":[{"type":"Key","id":"3","attributes":{"name":"G"}}]}`, nil
		case "/services/v2/songs/1/arrangements/2/keys/3/attachments":
			return `{"data":[{"type":"Attachment","id":"4","attributes":{"filename":"chart.pdf"}}]}`, nil
		case "/services/v2/tag_groups":
			return `{"data":[{"type":"TagGroup","id":"5","attributes":{"name":"Themes","tags_for":"song"}}]}`, nil
		case "/services/v2/tag_groups/5/tags":
			return `{"data":[{"type":"Tag","id":"6","attributes":{"name":"Grace"},"relationships":{"tag_group":{"data":{"type":"TagGroup","id":"5"}}}}]}`, nil
		case "/services/v2/songs/1/tags":
			return `{"data":[{"type":"Tag","id":"6","attributes":{"name":"Grace"}}]}`, nil
		case "/services/v2/attachment_types":
			return `{"data":[{"type":"AttachmentType","id":"7","attributes":{"name":"PDF"}}]}`, nil
		case "/services/v2/songs/1/song_schedules":
			return `{"data":[{"type":"SongSchedule","id":"8","attributes":{"plan_id":"10"}}]}`, nil
		case "/services/v2/songs/1/last_scheduled_item":
			if r.query.Get("service_type") != "9" {
				t.Fatal("service type omitted")
			}
			return `{"data":[{"type":"Item","id":"10","attributes":{"title":"Grace"}}]}`, nil
		}
		t.Fatalf("unexpected request %s", r.path)
		return "", nil
	}
	l := NewLibrary(f)
	s := Scope{Song: "1", Arrangement: "2", Key: "3"}
	tests := []func() (ListMeta, error){
		func() (ListMeta, error) {
			return l.WalkArrangements(context.Background(), Scope{Song: "1"}, func(a Arrangement) error {
				if a.SongID != "1" {
					t.Fatal("missing song ID")
				}
				return nil
			})
		},
		func() (ListMeta, error) {
			return l.WalkKeys(context.Background(), Scope{Song: "1", Arrangement: "2"}, func(k Key) error {
				if k.ArrangementID != "2" {
					t.Fatal("missing arrangement ID")
				}
				return nil
			})
		},
		func() (ListMeta, error) {
			return l.WalkAttachments(context.Background(), s, func(a Attachment) error {
				if a.KeyID != "3" {
					t.Fatal("missing key ID")
				}
				return nil
			})
		},
		func() (ListMeta, error) {
			return l.WalkTags(context.Background(), Scope{}, func(tag Tag) error {
				if tag.TagGroupID != "5" {
					t.Fatal("missing tag group")
				}
				return nil
			})
		},
		func() (ListMeta, error) {
			return l.WalkTags(context.Background(), Scope{Song: "1"}, func(Tag) error { return nil })
		},
		func() (ListMeta, error) {
			return l.WalkTagGroups(context.Background(), func(TagGroup) error { return nil })
		},
		func() (ListMeta, error) {
			return l.WalkAttachmentTypes(context.Background(), func(AttachmentType) error { return nil })
		},
		func() (ListMeta, error) {
			return l.WalkSongSchedules(context.Background(), Scope{Song: "1"}, func(AncillaryRecord) error { return nil })
		},
		func() (ListMeta, error) {
			return l.WalkLastScheduledItems(context.Background(), Scope{Song: "1", ServiceTypeID: "9"}, func(AncillaryRecord) error { return nil })
		},
	}
	for _, run := range tests {
		meta, err := run()
		if err != nil || meta.Returned != 1 || !meta.Complete {
			t.Fatal(meta, err)
		}
	}
	if f.writes != 0 || f.uploads != 0 {
		t.Fatal("side effect during export")
	}
}
func TestPrivateSongRunPermissions(t *testing.T) {
	directory := t.TempDir()
	name, err := writeSongRun(directory, func(emit func(Song) error) error { return emit(Song{ID: "1", Title: "Grace"}) })
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Clean(name))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("run permissions %v %v", info, err)
	}
}

func TestRequestedScopeIgnoresUnrelatedToManyRelationships(t *testing.T) {
	r := record{rel: []byte(`{"song":{"data":{"type":"Song","id":"1"}},"keys":{"data":[{"type":"Key","id":"3"}]}}`)}
	if err := r.scope("song", "Song", "1"); err != nil {
		t.Fatal("unrelated to-many relationship broke parent scope", err)
	}
	for _, relationship := range []string{`null`, `{"data":null}`, `{"data":[]}`, `{}`, `{"data":{"type":"Song","id":"999"}}`} {
		r.rel = []byte(`{"song":` + relationship + `,"keys":{"data":[]}}`)
		if err := r.scope("song", "Song", "1"); err == nil {
			t.Fatal("accepted invalid requested parent", relationship)
		}
	}
}
func TestWrongAttachableStopsAttachmentWrites(t *testing.T) {
	for _, operation := range []string{"update", "delete"} {
		t.Run(operation, func(t *testing.T) {
			f := &fixture{t: t}
			f.handle = func(r request) (string, error) {
				switch r.path {
				case "/services/v2/songs/1":
					return `{"data":` + songJSON("1", "Grace", false) + `}`, nil
				case "/services/v2/songs/1/arrangements/2":
					return `{"data":{"type":"Arrangement","id":"2","attributes":{"name":"Default"}}}`, nil
				case "/services/v2/songs/1/arrangements/2/keys/3":
					return `{"data":{"type":"Key","id":"3","attributes":{"name":"G"}}}`, nil
				case "/services/v2/songs/1/arrangements/2/keys/3/attachments":
					return `{"data":[{"type":"Attachment","id":"4","attributes":{"filename":"chart.pdf"},"relationships":{"attachable":{"data":{"type":"Key","id":"999"}},"attachment_types":{"data":[]}}}]}`, nil
				}
				t.Fatalf("unexpected request %#v", r)
				return "", nil
			}
			l := NewLibrary(f)
			scope := Scope{Song: "1", Arrangement: "2", Key: "3"}
			var err error
			if operation == "update" {
				_, err = l.UpdateAttachment(context.Background(), scope, "4", AttachmentPatch{Filename: ptr("new.pdf")})
			} else {
				err = l.DeleteAttachment(context.Background(), scope, "4")
			}
			if err == nil || f.writes != 0 || f.uploads != 0 {
				t.Fatalf("wrong attachable allowed effects: %v", err)
			}
		})
	}
}
func TestWrongAttachableWriteResponseIsUnknown(t *testing.T) {
	f := &fixture{t: t, handle: func(r request) (string, error) {
		return `{"data":{"type":"Attachment","id":"4","attributes":{"filename":"chart.pdf"},"relationships":{"attachable":{"data":{"type":"Arrangement","id":"999"}},"attachment_types":{"data":[]}}}}`, nil
	}}
	_, err := NewLibrary(f).api.mutate(context.Background(), "POST", "/services/v2/songs/1/arrangements/2/attachments", "Attachment", map[string]any{"filename": "chart.pdf"})
	assertCode(t, err, "outcome_unknown")
	if f.writes != 1 {
		t.Fatal("write replayed")
	}
}

func TestTagExportRejectsConflictingCatalogMembership(t *testing.T) {
	f := &fixture{t: t}
	f.handle = func(r request) (string, error) {
		if r.path == "/services/v2/tag_groups" {
			return `{"data":[{"type":"TagGroup","id":"1","attributes":{"name":"First","tags_for":"song"}},{"type":"TagGroup","id":"2","attributes":{"name":"Second","tags_for":"song"}}]}`, nil
		}
		return `{"data":[{"type":"Tag","id":"3","attributes":{"name":"Grace"}}]}`, nil
	}
	meta, err := NewLibrary(f).WalkTags(context.Background(), Scope{}, func(Tag) error { return nil })
	assertCode(t, err, "concurrent_change")
	if meta.Complete || f.writes != 0 {
		t.Fatal("conflicting tag catalog reported success")
	}
}
