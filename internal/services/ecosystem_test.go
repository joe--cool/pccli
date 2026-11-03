package services

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joe--cool/pccli/internal/config"
	"github.com/joe--cool/pccli/internal/planningcenter"
)

type ecosystemTransport func(*http.Request) (*http.Response, error)

func (f ecosystemTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func ecosystemResponse(body string) *http.Response {
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}
func ecosystemLibrary(t *testing.T, fn func(*http.Request) string) *Library {
	t.Helper()
	return NewLibrary(planningcenter.NewClient(config.Config{BaseURL: "https://api.planningcenteronline.com"}, ecosystemTransport(func(r *http.Request) (*http.Response, error) { return ecosystemResponse(fn(r)), nil })))
}

const ecosystemSong = `{"data":{"id":"1","type":"Song","attributes":{"title":"Grace","hidden":false}}}`
const ecosystemGroups = `{"data":[{"id":"10","type":"TagGroup","attributes":{"name":"Season","tags_for":"song","allow_multiple_selections":true,"required":false}},{"id":"11","type":"TagGroup","attributes":{"name":"Style","tags_for":"song","allow_multiple_selections":true,"required":false}}]}`

func TestReplaceTagsAmbiguousAcrossGroupsNoWrites(t *testing.T) {
	writes := 0
	l := ecosystemLibrary(t, func(r *http.Request) string {
		if r.Method != "GET" {
			writes++
			return `{}`
		}
		switch r.URL.Path {
		case "/services/v2/songs/1":
			return ecosystemSong
		case "/services/v2/tag_groups":
			return ecosystemGroups
		case "/services/v2/tag_groups/10/tags":
			return `{"data":[{"id":"20","type":"Tag","attributes":{"name":"Christmas"}}]}`
		case "/services/v2/tag_groups/11/tags":
			return `{"data":[{"id":"21","type":"Tag","attributes":{"name":"Christmas"}}]}`
		}
		t.Fatalf("unexpected %s", r.URL)
		return ""
	})
	_, err := l.ReplaceTags(context.Background(), Scope{Song: "1"}, []string{"Christmas"})
	if err == nil || !strings.Contains(err.Error(), "multiple groups") {
		t.Fatalf("err %v", err)
	}
	if writes != 0 {
		t.Fatal("ambiguous replacement wrote")
	}
}
func TestReplaceTagsClearHasExplicitArrayAndReadback(t *testing.T) {
	writes := 0
	l := ecosystemLibrary(t, func(r *http.Request) string {
		switch r.URL.Path {
		case "/services/v2/songs/1":
			return ecosystemSong
		case "/services/v2/tag_groups":
			return ecosystemGroups
		case "/services/v2/tag_groups/10/tags", "/services/v2/tag_groups/11/tags", "/services/v2/songs/1/tags":
			return `{"data":[]}`
		case "/services/v2/songs/1/assign_tags":
			writes++
			b, _ := io.ReadAll(r.Body)
			var payload map[string]any
			if json.Unmarshal(b, &payload) != nil {
				t.Fatal(string(b))
			}
			if !bytes.Contains(b, []byte(`"data":[]`)) || !bytes.Contains(b, []byte(`"type":"TagAssignment"`)) {
				t.Fatal(string(b))
			}
			return ""
		}
		t.Fatalf("unexpected %s", r.URL)
		return ""
	})
	result, err := l.ReplaceTags(context.Background(), Scope{Song: "1"}, []string{})
	if err != nil {
		t.Fatal(err)
	}
	if writes != 1 || len(result.Data) != 0 || !result.Meta.Complete {
		t.Fatalf("writes %d result %+v", writes, result)
	}
}
func TestAttachmentIDMustBelongToSelectedCollection(t *testing.T) {
	writes := 0
	l := ecosystemLibrary(t, func(r *http.Request) string {
		if r.Method != "GET" {
			writes++
			return ""
		}
		if r.URL.Path == "/services/v2/songs/1" {
			return ecosystemSong
		}
		if r.URL.Path == "/services/v2/songs/1/attachments" {
			return `{"data":[{"id":"30","type":"Attachment","attributes":{"filename":"chart.pdf"}}]}`
		}
		t.Fatalf("unexpected path %s", r.URL.Path)
		return ""
	})
	err := l.DeleteAttachment(context.Background(), Scope{Song: "1"}, "31")
	if err == nil {
		t.Fatal("wrong attachment accepted")
	}
	if writes != 0 {
		t.Fatal("wrong attachment deleted")
	}
}
func TestAttachmentUpdatePreservesFalseAndEmpty(t *testing.T) {
	writes := 0
	l := ecosystemLibrary(t, func(r *http.Request) string {
		if r.URL.Path == "/services/v2/songs/1" {
			return ecosystemSong
		}
		attrs := `"filename":"chart.pdf","remote_link":"","import_to_item_details":false`
		if r.Method == "PATCH" {
			writes++
			b, _ := io.ReadAll(r.Body)
			if !bytes.Contains(b, []byte(`"remote_link":""`)) || !bytes.Contains(b, []byte(`"import_to_item_details":false`)) {
				t.Fatal(string(b))
			}
		}
		if r.URL.Path == "/services/v2/songs/1/attachments" {
			return `{"data":[{"id":"30","type":"Attachment","attributes":{` + attrs + `}}]}`
		}
		if r.URL.Path == "/services/v2/songs/1/attachments/30" {
			return `{"data":{"id":"30","type":"Attachment","attributes":{` + attrs + `}}}`
		}
		t.Fatalf("unexpected %s", r.URL)
		return ""
	})
	empty := ""
	disabled := false
	_, err := l.UpdateAttachment(context.Background(), Scope{Song: "1"}, "30", AttachmentPatch{RemoteLink: &empty, ImportToItemDetails: &disabled})
	if err != nil {
		t.Fatal(err)
	}
	if writes != 1 {
		t.Fatalf("writes %d", writes)
	}
}
func TestAncillaryCollectionAndSingleShapes(t *testing.T) {
	l := ecosystemLibrary(t, func(r *http.Request) string {
		switch r.URL.Path {
		case "/services/v2/songs/1":
			return ecosystemSong
		case "/services/v2/songs/1/song_schedules":
			return `{"data":[{"id":"40","type":"SongSchedule","attributes":{"plan_title":"Sunday"}}]}`
		case "/services/v2/songs/1/last_scheduled_item":
			if r.URL.Query().Get("service_type") != "789" {
				t.Fatal("missing service_type query")
			}
			return `{"data":[]}`
		case "/services/v2/songs/1/arrangements/2":
			return `{"data":{"id":"2","type":"Arrangement","attributes":{"name":"Main","archived_at":null}}}`
		case "/services/v2/songs/1/arrangements/2/sections":
			return `{"data":{"id":"2","type":"ArrangementSections","attributes":{"sections":[]}}}`
		}
		t.Fatalf("unexpected %s", r.URL)
		return ""
	})
	schedules, err := l.ListSongSchedules(context.Background(), Scope{Song: "1"}, ListOptions{})
	if err != nil || len(schedules.Data) != 1 {
		t.Fatalf("%+v %v", schedules, err)
	}
	items, err := l.LastScheduledItems(context.Background(), Scope{Song: "1", ServiceTypeID: "789"}, ListOptions{})
	if err != nil || len(items.Data) != 0 {
		t.Fatalf("%+v %v", items, err)
	}
	sections, err := l.ArrangementSections(context.Background(), Scope{Song: "1", Arrangement: "2"})
	if err != nil || string(sections.Attributes["sections"]) != "[]" {
		t.Fatalf("%+v %v", sections, err)
	}
}
func TestUnarchiveRequiresExplicitSelectionBeforeNetwork(t *testing.T) {
	calls := 0
	l := ecosystemLibrary(t, func(*http.Request) string { calls++; return "" })
	_, err := l.UnarchiveArrangement(context.Background(), Scope{Song: "1"}, "2")
	if err == nil || calls != 0 {
		t.Fatalf("err %v calls %d", err, calls)
	}
}

func TestArchiveAndUnarchiveUseDocumentedActionsAndVerify(t *testing.T) {
	for _, restore := range []bool{false, true} {
		t.Run(map[bool]string{false: "archive", true: "unarchive"}[restore], func(t *testing.T) {
			changed := false
			writes := 0
			l := ecosystemLibrary(t, func(r *http.Request) string {
				if r.URL.Path == "/services/v2/songs/1" {
					return ecosystemSong
				}
				if r.Method == "POST" {
					want := "/services/v2/songs/1/arrangements/2/archive"
					if restore {
						want = "/services/v2/songs/1/unscoped_arrangements/2/unarchive"
					}
					if r.URL.Path != want {
						t.Fatalf("action path %s want %s", r.URL.Path, want)
					}
					writes++
					changed = true
					return ""
				}
				archived := restore != changed
				state := "null"
				if archived {
					state = `"2026-11-01T12:00:00Z"`
				}
				return `{"data":{"id":"2","type":"Arrangement","attributes":{"name":"Main","archived_at":` + state + `}}}`
			})
			scope := Scope{Song: "1", IncludeArchived: restore}
			var result Arrangement
			var err error
			if restore {
				result, err = l.UnarchiveArrangement(context.Background(), scope, "2")
			} else {
				result, err = l.ArchiveArrangement(context.Background(), scope, "2")
			}
			if err != nil {
				t.Fatal(err)
			}
			if writes != 1 || (result.ArchivedAt == "") != restore {
				t.Fatalf("result %+v writes %d", result, writes)
			}
		})
	}
}

func TestAttachmentUpdateRequestsUUIDForReadback(t *testing.T) {
	writes, readbacks := 0, 0
	l := ecosystemLibrary(t, func(r *http.Request) string {
		if r.URL.Path == "/services/v2/songs/1" {
			return ecosystemSong
		}
		if r.URL.Path == "/services/v2/songs/1/attachments" {
			return `{"data":[{"id":"30","type":"Attachment","attributes":{"filename":"chart.pdf"}}]}`
		}
		attrs := `"filename":"chart.pdf"`
		if r.Method == "PATCH" {
			writes++
		} else if fields := r.URL.Query().Get("fields[Attachment]"); fields != "" {
			readbacks++
			if !strings.Contains(fields, "file_upload_identifier") {
				t.Fatalf("UUID not requested: %s", fields)
			}
			attrs += `,"file_upload_identifier":"known-upload-uuid"`
		}
		return `{"data":{"id":"30","type":"Attachment","attributes":{` + attrs + `}}}`
	})
	uuid := "known-upload-uuid"
	result, err := l.UpdateAttachment(context.Background(), Scope{Song: "1"}, "30", AttachmentPatch{FileUploadID: &uuid})
	if err != nil {
		t.Fatal(err)
	}
	if writes != 1 || readbacks != 1 || result.Outcome != "verified" || result.Attachment.FileUploadID != uuid {
		t.Fatalf("result %+v writes %d readbacks %d", result, writes, readbacks)
	}
}
func TestAttachmentWriteOnlyUpdateReturnsAcceptedReceipt(t *testing.T) {
	writes := 0
	l := ecosystemLibrary(t, func(r *http.Request) string {
		if r.URL.Path == "/services/v2/songs/1" {
			return ecosystemSong
		}
		if r.URL.Path == "/services/v2/songs/1/attachments" {
			return `{"data":[{"id":"30","type":"Attachment","attributes":{"filename":"chart.pdf"}}]}`
		}
		if r.Method == "PATCH" {
			writes++
			body, _ := io.ReadAll(r.Body)
			if !bytes.Contains(body, []byte(`"song_part":"Verse 1"`)) || !bytes.Contains(body, []byte(`"attachment_type_ids":"5,6"`)) {
				t.Fatal(string(body))
			}
		}
		return `{"data":{"id":"30","type":"Attachment","attributes":{"filename":"chart.pdf"}}}`
	})
	types, part := "5,6", "Verse 1"
	result, err := l.UpdateAttachment(context.Background(), Scope{Song: "1"}, "30", AttachmentPatch{AttachmentTypeIDs: &types, SongPart: &part})
	if err != nil {
		t.Fatal(err)
	}
	if writes != 1 || result.Outcome != "accepted" || strings.Join(result.UnverifiedFields, ",") != "attachment_type_ids,song_part" {
		t.Fatalf("result %+v writes %d", result, writes)
	}
}
func TestAttachmentCreateTransmitsSongPart(t *testing.T) {
	writes := 0
	l := ecosystemLibrary(t, func(r *http.Request) string {
		if r.URL.Path == "/services/v2/songs/1" {
			return ecosystemSong
		}
		if r.Method == "POST" {
			writes++
			body, _ := io.ReadAll(r.Body)
			if !bytes.Contains(body, []byte(`"song_part":"Chorus 1"`)) {
				t.Fatal(string(body))
			}
			return `{"data":{"id":"30","type":"Attachment","attributes":{"filename":"chart.pdf"}}}`
		}
		t.Fatalf("unexpected %s %s", r.Method, r.URL)
		return ""
	})
	_, err := l.AddAttachment(context.Background(), Scope{Song: "1"}, AttachmentCreate{FileUploadID: "known-uuid", SongPart: "Chorus 1"})
	if err != nil || writes != 1 {
		t.Fatalf("err %v writes %d", err, writes)
	}
}

func TestLastScheduledItemRequiresServiceTypeBeforeNetwork(t *testing.T) {
	for _, id := range []string{"", "0", "-1", "abc", "id:5"} {
		calls := 0
		l := ecosystemLibrary(t, func(*http.Request) string { calls++; return "" })
		_, err := l.LastScheduledItems(context.Background(), Scope{Song: "1", ServiceTypeID: id}, ListOptions{})
		if err == nil || calls != 0 {
			t.Fatalf("ID %q err %v calls %d", id, err, calls)
		}
	}
}

func TestAttachmentValidatorsRunWithoutNetwork(t *testing.T) {
	empty := ""
	invalid := "ftp://example.test/chart.pdf"
	for _, patch := range []AttachmentPatch{{}, {Filename: &empty}, {RemoteLink: &invalid}} {
		if err := ValidateAttachmentPatch(patch); err == nil {
			t.Fatalf("accepted %+v", patch)
		}
	}
	disabled := false
	if err := ValidateAttachmentPatch(AttachmentPatch{ImportToItemDetails: &disabled}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, path := range []string{filepath.Join(dir, "missing.pdf"), dir} {
		if err := ValidateAttachmentCreate(AttachmentCreate{FilePath: path}); err == nil {
			t.Fatalf("accepted %s", path)
		}
	}
	path := filepath.Join(dir, "chart.pdf")
	if err := os.WriteFile(path, []byte("synthetic test file"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAttachmentCreate(AttachmentCreate{FilePath: path}); err != nil {
		t.Fatal(err)
	}
}
