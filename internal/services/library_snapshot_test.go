package services

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestLibraryExportResolvesParentsOnceAndAuditsScopes(t *testing.T) {
	f := &fixture{t: t}
	s := "/services/v2/songs/1"
	a := s + "/arrangements/2"
	k := a + "/keys/3"
	f.handle = func(r request) (string, error) {
		if r.method != "GET" {
			t.Fatalf("unexpected write: %+v", r)
		}
		switch r.path {
		case "/services/v2/songs":
			return songPage(songJSON("1", "Grace", false), ""), nil
		case s + "/attachments", a + "/attachments":
			return `{"data":[]}`, nil
		case s + "/arrangements":
			return `{"data":[{"type":"Arrangement","id":"2","attributes":{"name":"Default Arrangement"}}]}`, nil
		case a + "/keys":
			return `{"data":[{"type":"Key","id":"3","attributes":{"name":"C","starting_key":"C","ending_key":"C"}}]}`, nil
		case k + "/attachments":
			return `{"data":[{"type":"Attachment","id":"4","attributes":{"filename":"chart.pdf","file_size":10,"attachable_type":"Key","downloadable":true}}]}`, nil
		}
		return "", fmt.Errorf("unexpected repeated or unscoped read: %s", r.path)
	}
	snapshot, err := NewLibrary(f).ExportLibrary(context.Background(), Visible, false, LibraryInclude{Arrangements: true, Keys: true, Attachments: true}, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.Complete || len(f.requests) != 6 {
		t.Fatalf("snapshot complete=%v, requests=%v", snapshot.Complete, f.requests)
	}
	actual, err := AuditLibrary(snapshot, &snapshot)
	if err != nil || actual.Keys != 1 || actual.Attachments != 1 || len(actual.Findings) != 0 {
		t.Fatalf("report=%+v err=%v", actual, err)
	}
	stored := snapshot.Songs[0].Arrangements[0]
	file := stored.Keys[0].Attachments[0]
	file.AttachableType = "Arrangement"
	stored.Attachments = []Attachment{file}
	stored.Keys = []LibraryKey{{Key: stored.Keys[0].Key, Attachments: []Attachment{}}}
	changed := snapshot
	changed.Songs = slices.Clone(snapshot.Songs)
	changed.Songs[0].Arrangements = []LibraryArrangement{stored}
	report, err := AuditLibrary(changed, &snapshot)
	if err != nil || len(report.Findings) != 3 {
		t.Fatalf("moved and empty findings=%+v err=%v", report.Findings, err)
	}
	for _, finding := range report.Findings {
		if !strings.HasPrefix(finding.Scope, "/songs/1/arrangements/2") {
			t.Fatal(finding)
		}
	}
}

func TestLibraryExportDoesNotReturnPartialPages(t *testing.T) {
	f := &fixture{t: t, handle: func(r request) (string, error) {
		if r.path == "/services/v2/songs" {
			return songPage(songJSON("1", "Grace", false), ""), nil
		}
		return "", fmt.Errorf("page failed")
	}}
	snapshot, err := NewLibrary(f).ExportLibrary(context.Background(), Visible, false, LibraryInclude{Arrangements: true}, 1, nil)
	if err == nil || snapshot.Complete || snapshot.Songs != nil {
		t.Fatalf("partial export escaped: %+v, %v", snapshot, err)
	}
}
