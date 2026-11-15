package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestRelocateVerifiesBeforeRetiringSourceAndPruningKey(t *testing.T) {
	for _, mode := range []string{"preview", "apply", "mismatch", "wrong_local", "custom_order", "resume"} {
		t.Run(mode, func(t *testing.T) {
			const song = "/services/v2/songs/1"
			const arrangement = song + "/arrangements/2"
			const key = arrangement + "/keys/3"
			const dest = arrangement + "/attachments"
			const src = key + "/attachments"
			record := func(id, kind string, size int) string {
				return fmt.Sprintf(`{"type":"Attachment","id":%q,"attributes":{"filename":"chart.pdf","file_size":%d,"content_type":"application/pdf","page_order":[],"pco_type":"AttachmentS3","remote_link":"805f8aeda079196d6cedbfc6edccc5de","attachable_type":%q,"downloadable":true}}`, id, size, kind)
			}
			old := record("4", "Key", 8)
			size := 8
			if mode == "mismatch" {
				size = 9
			}
			replacement := record("5", "Arrangement", size)
			created, retired, pruned := mode == "resume", false, false
			var journal []string
			f := &fixture{t: t}
			f.handle = func(r request) (string, error) {
				switch r.method {
				case "UPLOAD":
					return `{"data":[{"id":"upload-uuid","type":"File","attributes":{"name":"chart.pdf","file_size":8}}]}`, nil
				case "POST":
					if r.path != dest || retired || !slices.Contains(journal, "pending") {
						t.Fatalf("unexpected POST: %+v", r)
					}
					created = true
					return `{"data":` + replacement + `}`, nil
				case "DELETE":
					switch r.path {
					case src + "/4":
						if !slices.Contains(journal, "replacement_verified") {
							t.Fatal("retired without verified replacement")
						}
						retired = true
					case key:
						if !retired || !slices.Contains(journal, "source_retired") {
							t.Fatal("pruned before source verification")
						}
						pruned = true
					default:
						t.Fatalf("unexpected DELETE: %s", r.path)
					}
					return "", nil
				}
				switch r.path {
				case song:
					return ecosystemSong, nil
				case arrangement:
					return `{"data":{"type":"Arrangement","id":"2","attributes":{"name":"Default Arrangement"}}}`, nil
				case key:
					return `{"data":{"type":"Key","id":"3","attributes":{"name":"C"}}}`, nil
				case arrangement + "/keys":
					if pruned {
						return `{"data":[]}`, nil
					}
					return `{"data":[{"type":"Key","id":"3","attributes":{"name":"C"}}]}`, nil
				case song + "/song_schedules":
					return `{"data":[]}`, nil
				case src:
					if retired {
						return `{"data":[]}`, nil
					}
					return `{"data":[` + old + `]}`, nil
				case src + "/4":
					row := old
					if mode == "custom_order" && r.query.Get("blank_pages") == "true" {
						row = strings.Replace(row, `"page_order":[]`, `"page_order":[0,1]`, 1)
					}
					return `{"data":` + row + `}`, nil
				case dest:
					if !created {
						return `{"data":[]}`, nil
					}
					return `{"data":[` + replacement + `]}`, nil
				case dest + "/5":
					return `{"data":` + replacement + `}`, nil
				}
				return "", fmt.Errorf("unexpected request: %+v", r)
			}
			p := filepath.Join(t.TempDir(), "chart.pdf")
			pdf := []byte("%PDF-1.0")
			if mode == "wrong_local" {
				pdf = []byte("%PDF-1.1")
			}
			if err := os.WriteFile(p, pdf, 0600); err != nil {
				t.Fatal(err)
			}
			opt := RelocationOptions{From: Scope{Song: "id:1", Arrangement: "id:2", Key: "id:3"}, To: Scope{Song: "id:1", Arrangement: "id:2"}, SourceID: "id:4", File: p,
				Apply: mode != "preview", PruneEmptyKey: true, Checkpoint: func(r AttachmentRelocation) error { journal = append(journal, r.Outcome); return nil }}
			if mode == "resume" {
				opt.File = ""
				opt.ReplacementID = "id:5"
			}
			r, err := NewLibrary(f).RelocateAttachment(context.Background(), opt)
			switch mode {
			case "wrong_local", "custom_order":
				if err == nil || f.writes != 0 {
					t.Fatalf("same-size unrelated file accepted: %+v %v", r, err)
				}
			case "mismatch":
				if err == nil || retired || pruned {
					t.Fatalf("unsafe result=%+v err=%v", r, err)
				}
			case "preview":
				if err != nil || r.Outcome != "preview" || f.writes != 0 {
					t.Fatalf("preview result=%+v err=%v", r, err)
				}
			default:
				if err != nil || r.Outcome != "verified" || !retired || !pruned {
					t.Fatalf("move result=%+v err=%v", r, err)
				}
				if mode == "resume" && f.uploads != 0 {
					t.Fatal("resume uploaded again")
				}
			}
		})
	}
}
