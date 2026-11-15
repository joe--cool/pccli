package services

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
)

type LibraryFinding struct {
	Code  string `json:"code"`
	Scope string `json:"scope"`
}

type LibraryAudit struct {
	Songs        int              `json:"songs"`
	Arrangements int              `json:"arrangements"`
	Keys         int              `json:"keys"`
	Attachments  int              `json:"attachments"`
	Findings     []LibraryFinding `json:"findings"`
}

func AuditLibrary(actual LibrarySnapshot, expected *LibrarySnapshot) (LibraryAudit, error) {
	got, report, err := libraryIndex(actual)
	if err != nil {
		return report, err
	}
	if expected != nil {
		if actual.Include != expected.Include || actual.Visibility != expected.Visibility || actual.IncludeArchived != expected.IncludeArchived {
			return report, fail("invalid_input", "snapshot and expected inventory must select the same scopes")
		}
		want, _, err := libraryIndex(*expected)
		if err != nil {
			return report, err
		}
		for scope, signature := range want {
			if gotSignature, ok := got[scope]; !ok {
				report.Findings = append(report.Findings, LibraryFinding{"missing_resource", scope})
			} else if gotSignature != signature {
				report.Findings = append(report.Findings, LibraryFinding{"different_metadata", scope})
			}
		}
		for scope := range got {
			if _, ok := want[scope]; !ok {
				report.Findings = append(report.Findings, LibraryFinding{"unexpected_resource", scope})
			}
		}
	}
	slices.SortFunc(report.Findings, func(a, b LibraryFinding) int {
		if a.Scope != b.Scope {
			if a.Scope < b.Scope {
				return -1
			}
			return 1
		}
		if a.Code < b.Code {
			return -1
		}
		if a.Code > b.Code {
			return 1
		}
		return 0
	})
	return report, nil
}

func libraryIndex(snapshot LibrarySnapshot) (map[string]string, LibraryAudit, error) {
	report := LibraryAudit{Findings: []LibraryFinding{}}
	if !snapshot.Complete || snapshot.Schema != LibrarySnapshotSchema || snapshot.Songs == nil {
		return nil, report, fail("invalid_input", "expected a complete pccli/library-v1 snapshot")
	}
	include := snapshot.Include
	if include.Keys && !include.Arrangements || include.Attachments && !include.Keys {
		return nil, report, fail("invalid_input", "invalid snapshot inclusion options")
	}
	index := map[string]string{}
	seen := map[string]bool{}
	add := func(kind, parent, id string, value ...any) (string, error) {
		if !validID(id) || seen[kind+"/"+id] {
			return "", fail("invalid_input", "invalid or duplicate "+kind+" ID in snapshot")
		}
		seen[kind+"/"+id] = true
		scope := parent + "/" + kind + "/" + id
		encoded, err := json.Marshal(value)
		if err == nil {
			index[scope] = string(encoded)
		}
		return scope, err
	}
	files := func(scope, kind string, rows []Attachment) error {
		if include.Attachments && rows == nil || !include.Attachments && len(rows) != 0 {
			return fail("invalid_input", "incomplete attachment scope "+scope)
		}
		for _, a := range rows {
			if a.AttachableType != kind {
				return fail("invalid_input", "attachment type differs from its scope")
			}
			if _, err := add("attachments", scope, a.ID, a.Filename, a.FileSize, a.Downloadable, a.RemoteLink); err != nil {
				return err
			}
			report.Attachments++
		}
		return nil
	}
	tagIDs := func(rows []Tag) ([]string, error) {
		if include.Tags && rows == nil || !include.Tags && len(rows) != 0 {
			return nil, fail("invalid_input", "incomplete tag scope")
		}
		ids := map[string]struct{}{}
		for _, t := range rows {
			if _, exists := ids[t.ID]; !validID(t.ID) || exists {
				return nil, fail("invalid_input", "invalid or duplicate tag ID in snapshot")
			}
			ids[t.ID] = struct{}{}
		}
		return slices.Sorted(maps.Keys(ids)), nil
	}
	for _, song := range snapshot.Songs {
		s := song.Song
		st, err := tagIDs(song.Tags)
		if err != nil {
			return nil, report, err
		}
		ss, err := add("songs", "", s.ID, s.Title, s.Author, s.Hidden, st)
		if err != nil {
			return nil, report, err
		}
		report.Songs++
		if err := files(ss, "Song", song.Attachments); err != nil {
			return nil, report, err
		}
		if include.Arrangements && song.Arrangements == nil || !include.Arrangements && len(song.Arrangements) != 0 {
			return nil, report, fail("invalid_input", "incomplete arrangement scope "+ss)
		}
		for _, arrangement := range song.Arrangements {
			a := arrangement.Arrangement
			at, err := tagIDs(arrangement.Tags)
			if err != nil {
				return nil, report, err
			}
			as, err := add("arrangements", ss, a.ID, a.Name, a.Meter, a.BPM, a.ArchivedAt, at)
			if err != nil {
				return nil, report, err
			}
			report.Arrangements++
			if err := files(as, "Arrangement", arrangement.Attachments); err != nil {
				return nil, report, err
			}
			if include.Keys && arrangement.Keys == nil || !include.Keys && len(arrangement.Keys) != 0 {
				return nil, report, fail("invalid_input", "incomplete key scope "+as)
			}
			for _, key := range arrangement.Keys {
				k := key.Key
				ks, err := add("keys", as, k.ID, k.Name, k.StartingKey, k.EndingKey)
				if err != nil {
					return nil, report, err
				}
				report.Keys++
				if err := files(ks, "Key", key.Attachments); err != nil {
					return nil, report, err
				}
				if include.Attachments && len(key.Attachments) == 0 {
					report.Findings = append(report.Findings, LibraryFinding{"empty_key", ks})
				}
			}
		}
	}
	if _, err := visible(snapshot.Visibility); err != nil || snapshot.Visibility == "" {
		return nil, report, fmt.Errorf("snapshot visibility must be visible, hidden, or all")
	}
	return index, report, nil
}
