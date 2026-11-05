package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/joe--cool/pccli/v2/internal/planningcenter"
)

func itemResource(id, title string, sequence int) string {
	return planningResource("Item", id, `{"title":"`+title+`","sequence":`+fmt.Sprint(sequence)+`,"service_position":"during"}`, `{"plan":{"data":{"type":"Plan","id":"2"}}}`)
}
func TestItemCreateTypedFieldsAndScopedSongLinks(t *testing.T) {
	for _, mode := range []string{"header", "song", "wrong_key", "late_read_failure"} {
		t.Run(mode, func(t *testing.T) {
			f := planningFixture(t)
			base := f.handle
			var posted map[string]any
			f.handle = func(r request) (string, error) {
				if r.path == "/services/v2/songs/10" {
					return `{"data":` + songJSON("10", "Grace", false) + `}`, nil
				}
				if r.path == "/services/v2/songs/10/arrangements/11" {
					return planningSingle("Arrangement", "11", `{"name":"Band"}`, ""), nil
				}
				if r.path == "/services/v2/songs/10/arrangements/11/keys/12" {
					return planningSingle("Key", "12", `{"name":"G"}`, ""), nil
				}
				if r.method == "POST" {
					if r.path != "/services/v2/service_types/1/plans/2/items" {
						t.Fatal(r.path)
					}
					posted = r.body.(map[string]any)["data"].(map[string]any)["attributes"].(map[string]any)
					return planningSingle("Item", "20", `{}`, ""), nil
				}
				if strings.HasSuffix(r.path, "/items/20") {
					if mode == "late_read_failure" {
						return "", errors.New("read failed")
					}
					attrs := `{"title":"","length":0,"sequence":0,"custom_arrangement_sequence":[],"item_type":"header"}`
					rel := `{}`
					if mode != "header" {
						attrs = `{"title":"Grace","item_type":"song"}`
						rel = `{"song":{"data":{"type":"Song","id":"10"}},"arrangement":{"data":{"type":"Arrangement","id":"11"}},"key":{"data":{"type":"Key","id":"12"}}}`
						if mode == "wrong_key" {
							rel = strings.Replace(rel, `"id":"12"`, `"id":"13"`, 1)
						}
					}
					return planningSingle("Item", "20", attrs, rel), nil
				}
				return base(r)
			}
			p := ItemCreate{}
			title := ""
			typ := "header"
			zero := 0
			seq := []string{}
			if mode == "header" {
				p.Title = &title
				p.Type = &typ
				p.Length = &zero
				p.Position = &zero
				p.ChartSequence = &seq
			} else {
				title = "Grace"
				p.Title = &title
				p.Song = &Scope{Song: "10", Arrangement: "11", Key: "12"}
			}
			got, e := NewLibrary(f).CreateItem(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, p)
			if mode == "wrong_key" || mode == "late_read_failure" {
				assertCode(t, e, "verification_failed")
				var failure *Error
				if !errors.As(e, &failure) || failure.PlanningRecovery == nil || failure.PlanningRecovery.TargetID != "20" || failure.PlanningRecovery.PlanID != "2" || len(failure.PlanningRecovery.Created) != 1 {
					t.Fatalf("recovery=%+v", failure)
				}
			} else if e != nil || got.ID != "20" {
				t.Fatalf("got=%+v e=%v", got, e)
			}
			if f.writes != 1 || f.uploads != 0 {
				t.Fatalf("writes=%d uploads=%d", f.writes, f.uploads)
			}
			if mode == "header" {
				want := map[string]any{"title": "", "length": 0, "sequence": 0, "custom_arrangement_sequence": []string{}, "item_type": "header"}
				if !reflect.DeepEqual(posted, want) {
					t.Fatalf("posted=%#v", posted)
				}
			} else if posted["song_id"] != "10" || posted["arrangement_id"] != "11" || posted["key_id"] != "12" {
				t.Fatalf("posted=%v", posted)
			}
		})
	}
}
func TestItemInvalidInputZeroRequests(t *testing.T) {
	f := planningFixture(t)
	l := NewLibrary(f)
	badLength := -1
	header := "header"
	badPos := "after"
	badSequence := []string{"Verse 1", ""}
	for _, p := range []ItemCreate{
		{ItemPatch: ItemPatch{Length: &badLength}},
		{ItemPatch: ItemPatch{Song: &Scope{Song: "10"}}},
		{ItemPatch: ItemPatch{Song: &Scope{Song: "10", Arrangement: "11"}}, Type: &header},
		{ItemPatch: ItemPatch{ServicePosition: &badPos}},
		{ItemPatch: ItemPatch{ChartSequence: &badSequence}},
	} {
		_, e := l.CreateItem(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, p)
		assertCode(t, e, "invalid_input")
	}
	_, e := l.UpdateItem(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, "20", ItemPatch{})
	assertCode(t, e, "invalid_input")
	for _, sequence := range [][]string{{}, {"1", "01"}, {"0"}, {"name:Item"}} {
		_, e := l.ReorderItems(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, sequence)
		assertCode(t, e, "invalid_input")
	}
	if len(f.requests) != 0 {
		t.Fatalf("requests=%+v", f.requests)
	}
}
func TestItemReorderFullSelectionAndReadback(t *testing.T) {
	for _, mode := range []string{"verified", "foreign", "omission", "page_failure", "changed", "late_failure", "wrong_order", "wrong_sequence", "unknown"} {
		t.Run(mode, func(t *testing.T) {
			f := planningFixture(t)
			base := f.handle
			collections := 0
			posted := false
			f.handle = func(r request) (string, error) {
				if strings.HasSuffix(r.path, "/items") {
					collections++
					if mode == "page_failure" {
						return `{"data":null}`, nil
					}
					if posted {
						if mode == "late_failure" {
							return "", errors.New("read failed")
						}
						if mode == "wrong_order" {
							return songPage(itemResource("20", "First", 1)+","+itemResource("21", "Second", 2), ""), nil
						}
						first, second := itemResource("21", "Second", 1), itemResource("20", "First", 2)
						if mode == "wrong_sequence" {
							first = itemResource("21", "Second", 3)
						}
						return songPage(first+","+second, ""), nil
					}
					title := "First"
					if mode == "changed" && collections == 2 {
						title = "Changed"
					}
					return songPage(itemResource("20", title, 1)+","+itemResource("21", "Second", 2), ""), nil
				}
				if strings.HasSuffix(r.path, "/item_reorder") {
					posted = true
					b, _ := json.Marshal(r.body)
					if string(b) != `{"data":{"attributes":{"sequence":["21","20"]},"type":"PlanItemReorder"}}` {
						t.Fatalf("body=%s", b)
					}
					if mode == "unknown" {
						return "", &planningcenter.TransportError{UnknownOutcome: true, Operation: "POST", Cause: errors.New("lost")}
					}
					return "", nil
				}
				return base(r)
			}
			sequence := []string{"021", "20"}
			if mode == "foreign" {
				sequence = []string{"21", "22"}
			}
			if mode == "omission" {
				sequence = []string{"21"}
			}
			got, e := NewLibrary(f).ReorderItems(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, sequence)
			switch mode {
			case "verified":
				if e != nil || len(got.Data) != 2 || got.Data[0].ID != "21" || !got.Meta.Complete {
					t.Fatalf("got=%+v e=%v", got, e)
				}
			case "foreign", "omission":
				assertCode(t, e, "invalid_input")
			case "changed":
				assertCode(t, e, "concurrent_change")
			case "late_failure", "wrong_order", "wrong_sequence":
				assertCode(t, e, "verification_failed")
				var se *Error
				if !errors.As(e, &se) || se.PlanningRecovery == nil {
					t.Fatalf("missing recovery=%v", e)
				}
			default:
				if e == nil {
					t.Fatal("expected failure")
				}
			}
			zero := oneOf(mode, "foreign", "omission", "page_failure", "changed")
			if zero && f.writes != 0 || !zero && f.writes != 1 {
				t.Fatalf("mode=%s writes=%d", mode, f.writes)
			}
		})
	}
}
func TestItemDeleteAndEmptyPatch(t *testing.T) {
	f := planningFixture(t)
	base := f.handle
	deleted := false
	updated := false
	f.handle = func(r request) (string, error) {
		if strings.HasSuffix(r.path, "/items/20") {
			if r.method == "DELETE" {
				deleted = true
				return "", nil
			}
			if deleted {
				return "", planningcenter.APIError{StatusCode: 404}
			}
			if r.method == "PATCH" {
				updated = true
				b, _ := json.Marshal(r.body)
				if string(b) != `{"data":{"attributes":{"description":"","length":0},"type":"Item"}}` {
					t.Fatalf("body=%s", b)
				}
			}
			attrs := `{"title":"Welcome","description":"Before","length":60}`
			if updated {
				attrs = `{"title":"Welcome","description":"","length":0}`
			}
			return planningSingle("Item", "20", attrs, ""), nil
		}
		return base(r)
	}
	l := NewLibrary(f)
	empty := ""
	zero := 0
	s := PlanningScope{ServiceType: "1", Plan: "2"}
	got, e := l.UpdateItem(context.Background(), s, "20", ItemPatch{Description: &empty, Length: &zero})
	if e != nil || got.Description == nil || *got.Description != "" || got.Length == nil || *got.Length != 0 {
		t.Fatalf("got=%+v e=%v", got, e)
	}
	if e = l.DeleteItem(context.Background(), s, "20"); e != nil {
		t.Fatal(e)
	}
	if f.writes != 2 {
		t.Fatal(f.writes)
	}
}

func TestItemNamedMutationRelationshipExpansionAndChanges(t *testing.T) {
	for _, mode := range []string{"expanded", "reordered_json", "changed_song", "lost_key", "empty_changed"} {
		for _, deleting := range []bool{false, true} {
			t.Run(mode+fmt.Sprint(deleting), func(t *testing.T) {
				f := planningFixture(t)
				base := f.handle
				written := false
				attrs := `{"title":"Welcome","length":0}`
				sourceRel := ""
				detailRel := `{"plan":{"data":{"type":"Plan","id":"2"}}}`
				switch mode {
				case "reordered_json":
					sourceRel = `{"song":{"data":{"type":"Song","id":"10"}}}`
					detailRel = `{"plan":{"data":{"id":"2","type":"Plan"}},"song":{"data":{"id":"10", "type":"Song"}}}`
				case "changed_song":
					sourceRel = `{"song":{"data":{"type":"Song","id":"10"}}}`
					detailRel = `{"song":{"data":{"type":"Song","id":"11"}}}`
				case "lost_key":
					sourceRel = `{"key":{"data":{"type":"Key","id":"12"}}}`
					detailRel = `{}`
				case "empty_changed":
					sourceRel = `{"song":{"data":null}}`
					detailRel = `{"song":{"data":{"type":"Song","id":"10"}}}`
				}
				f.handle = func(r request) (string, error) {
					if strings.HasSuffix(r.path, "/items") {
						return songPage(planningResource("Item", "20", attrs, sourceRel), ""), nil
					}
					if strings.HasSuffix(r.path, "/items/20") {
						if r.method == "DELETE" {
							written = true
							return "", nil
						}
						if deleting && written {
							return "", planningcenter.APIError{StatusCode: 404}
						}
						if r.method == "PATCH" {
							written = true
						}
						return planningSingle("Item", "20", attrs, detailRel), nil
					}
					return base(r)
				}
				l := NewLibrary(f)
				s := PlanningScope{ServiceType: "1", Plan: "2"}
				var e error
				if deleting {
					e = l.DeleteItem(context.Background(), s, "Welcome")
				} else {
					zero := 0
					_, e = l.UpdateItem(context.Background(), s, "Welcome", ItemPatch{Length: &zero})
				}
				if mode == "expanded" || mode == "reordered_json" {
					if e != nil || f.writes != 1 {
						t.Fatalf("e=%v writes=%d", e, f.writes)
					}
				} else {
					assertCode(t, e, "concurrent_change")
					if f.writes != 0 {
						t.Fatal("write after relationship changed")
					}
				}
			})
		}
	}
}

func TestItemNilChartSequenceTransmitsEmptyArray(t *testing.T) {
	f := planningFixture(t)
	base := f.handle
	f.handle = func(r request) (string, error) {
		if r.method == "POST" {
			b, _ := json.Marshal(r.body)
			if string(b) != `{"data":{"attributes":{"custom_arrangement_sequence":[]},"type":"Item"}}` {
				t.Fatalf("body=%s", b)
			}
			return planningSingle("Item", "20", `{}`, ""), nil
		}
		if strings.HasSuffix(r.path, "/items/20") {
			return planningSingle("Item", "20", `{"custom_arrangement_sequence":[]}`, ""), nil
		}
		return base(r)
	}
	var sequence []string
	_, e := NewLibrary(f).CreateItem(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, ItemCreate{ItemPatch: ItemPatch{ChartSequence: &sequence}})
	if e != nil {
		t.Fatal(e)
	}
	if sequence != nil {
		t.Fatal("caller slice mutated")
	}
}
func TestItemSongSelectionFailuresNeverWrite(t *testing.T) {
	for _, mode := range []string{"duplicate", "late_failure", "hidden", "wrong_arrangement_parent"} {
		t.Run(mode, func(t *testing.T) {
			f := planningFixture(t)
			base := f.handle
			f.handle = func(r request) (string, error) {
				if r.path == "/services/v2/songs" {
					if r.query.Get("offset") == "" {
						return songPage(songJSON("10", "Grace", false), "/services/v2/songs?offset=100&per_page=100&where%5Bhidden%5D=false"), nil
					}
					if mode == "late_failure" {
						return "", errors.New("late page failed")
					}
					return songPage(songJSON("13", "Grace", false), ""), nil
				}
				if r.path == "/services/v2/songs/10" {
					return `{"data":` + songJSON("10", "Grace", mode == "hidden") + `}`, nil
				}
				if strings.HasSuffix(r.path, "/arrangements/11") {
					return planningSingle("Arrangement", "11", `{"name":"Band"}`, `{"song":{"data":{"type":"Song","id":"99"}}}`), nil
				}
				return base(r)
			}
			song := "Grace"
			if mode == "hidden" || mode == "wrong_arrangement_parent" {
				song = "10"
			}
			_, e := NewLibrary(f).CreateItem(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, ItemCreate{ItemPatch: ItemPatch{Song: &Scope{Song: song, Arrangement: "11"}}})
			if e == nil || f.writes != 0 || f.uploads != 0 {
				t.Fatalf("e=%v writes=%d uploads=%d", e, f.writes, f.uploads)
			}
		})
	}
}
