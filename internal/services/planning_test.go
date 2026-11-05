package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

func planningResource(kind, id, attrs, relationships string) string {
	if relationships == "" {
		relationships = "{}"
	}
	return fmt.Sprintf(`{"type":%q,"id":%q,"attributes":%s,"relationships":%s}`, kind, id, attrs, relationships)
}
func planningSingle(kind, id, attrs, rel string) string {
	return `{"data":` + planningResource(kind, id, attrs, rel) + `}`
}
func planningFixture(t *testing.T) *fixture {
	f := &fixture{t: t}
	f.handle = func(r request) (string, error) {
		switch r.path {
		case "/services/v2/service_types/1":
			return planningSingle("ServiceType", "1", `{"name":"Sunday","archived_at":null}`, ""), nil
		case "/services/v2/service_types/1/plans/2":
			return planningSingle("Plan", "2", `{"title":null,"dates":"Nov 8, 2026"}`, `{"service_type":{"data":{"type":"ServiceType","id":"1"}}}`), nil
		default:
			t.Fatalf("unexpected request: %+v", r)
			return "", nil
		}
	}
	return f
}
func TestPlanningReadScopeAndNullableValues(t *testing.T) {
	f := planningFixture(t)
	base := f.handle
	f.handle = func(r request) (string, error) {
		if strings.HasSuffix(r.path, "/items") {
			return songPage(planningResource("Item", "9", `{"title":"Welcome","length":0,"sequence":4,"service_position":"during"}`, `{"plan":{"data":{"type":"Plan","id":"2"}},"song":{"data":null}}`)+","+planningResource("Item", "8", `{"title":null}`, ""), ""), nil
		}
		return base(r)
	}
	l := NewLibrary(f)
	r, e := l.ListItems(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, ListOptions{All: true})
	if e != nil {
		t.Fatal(e)
	}
	if len(r.Data) != 2 || r.Data[0].ID != "9" || r.Data[1].ID != "8" || r.Data[0].PlanID != "2" || r.Data[0].Length == nil || *r.Data[0].Length != 0 || r.Data[1].Length != nil || r.Data[1].Title != nil || !r.Meta.Complete {
		t.Fatalf("result = %+v", r)
	}
	plan, e := l.ShowPlan(context.Background(), PlanningScope{ServiceType: "1"}, "2")
	if e != nil || plan.Title != nil || plan.Public != nil || plan.ItemsCount != nil {
		t.Fatalf("plan=%+v err=%v", plan, e)
	}
}
func TestPlanningLateSelectionFailuresHaveNoWrites(t *testing.T) {
	for _, mode := range []string{"duplicate", "bad_type", "bad_title", "wrong_parent", "shift_path", "page_failure", "changed_filter"} {
		t.Run(mode, func(t *testing.T) {
			f := planningFixture(t)
			base := f.handle
			f.handle = func(r request) (string, error) {
				if strings.HasSuffix(r.path, "/plans") {
					if r.query.Get("offset") == "" {
						next := "/services/v2/service_types/1/plans?per_page=100&offset=100"
						if mode == "shift_path" {
							next = "/services/v2/service_types/3/plans?per_page=100&offset=100"
						}
						if mode == "changed_filter" {
							next += "&filter=future"
						}
						return songPage(planningResource("Plan", "2", `{"title":"Gather"}`, ""), next), nil
					}
					if mode == "page_failure" {
						return "", errors.New("late page failed")
					}
					kind, attrs, rel := "Plan", `{"title":"Gather"}`, ""
					if mode == "bad_type" {
						kind = "Song"
					}
					if mode == "bad_title" {
						attrs = `{"title":4}`
					}
					if mode == "wrong_parent" {
						rel = `{"service_type":{"data":{"type":"ServiceType","id":"3"}}}`
					}
					return songPage(planningResource(kind, "3", attrs, rel), ""), nil
				}
				return base(r)
			}
			title := "New"
			_, e := NewLibrary(f).UpdatePlan(context.Background(), PlanningScope{ServiceType: "1"}, "Gather", PlanPatch{Title: &title})
			if e == nil || f.writes != 0 || f.uploads != 0 {
				t.Fatalf("err=%v writes=%d", e, f.writes)
			}
			if mode == "duplicate" {
				assertCode(t, e, "ambiguous_target")
			}
		})
	}
}
func TestPlanningPatchPresenceAndReadbackOutcomes(t *testing.T) {
	for _, mode := range []string{"verified", "wrong_id", "read_failure", "wrong_read_parent", "wrong_value", "null_value", "transport"} {
		t.Run(mode, func(t *testing.T) {
			f := planningFixture(t)
			base := f.handle
			submitted := false
			f.handle = func(r request) (string, error) {
				if r.method == "PATCH" {
					submitted = true
					if r.path != "/services/v2/service_types/1/plans/2" {
						t.Fatalf("path %s", r.path)
					}
					b, _ := json.Marshal(r.body)
					if string(b) != `{"data":{"attributes":{"series_title":"","title":""},"type":"Plan"}}` {
						t.Fatalf("body=%s", b)
					}
					if mode == "transport" {
						return "", errors.New("request failed")
					}
					id := "2"
					if mode == "wrong_id" {
						id = "3"
					}
					return planningSingle("Plan", id, `{"title":"","series_title":""}`, ""), nil
				}
				if submitted && strings.HasSuffix(r.path, "/plans/2") {
					if mode == "read_failure" {
						return "", errors.New("readback failed")
					}
					rel := ""
					attrs := `{"title":"","series_title":""}`
					if mode == "wrong_read_parent" {
						rel = `{"service_type":{"data":{"type":"ServiceType","id":"9"}}}`
					}
					if mode == "wrong_value" {
						attrs = `{"title":"old","series_title":""}`
					}
					if mode == "null_value" {
						attrs = `{"title":null,"series_title":""}`
					}
					return planningSingle("Plan", "2", attrs, rel), nil
				}
				return base(r)
			}
			empty := ""
			r, e := NewLibrary(f).UpdatePlan(context.Background(), PlanningScope{ServiceType: "1"}, "2", PlanPatch{Title: &empty, SeriesTitle: &empty})
			if f.writes != 1 {
				t.Fatalf("writes=%d", f.writes)
			}
			if mode == "verified" {
				if e != nil || r.Title == nil || *r.Title != "" {
					t.Fatalf("result=%+v err=%v", r, e)
				}
			} else if mode == "wrong_id" {
				assertCode(t, e, "outcome_unknown")
			} else if mode != "transport" {
				assertCode(t, e, "verification_failed")
			} else if e == nil {
				t.Fatal("lost transport error")
			}
		})
	}
}
func TestPlanningValidationBeforeNetwork(t *testing.T) {
	f := planningFixture(t)
	l := NewLibrary(f)
	empty := ""
	cases := []func() error{
		func() error {
			_, e := l.UpdatePlan(context.Background(), PlanningScope{ServiceType: "1"}, "2", PlanPatch{})
			return e
		},
		func() error {
			_, e := l.UpdatePlan(context.Background(), PlanningScope{ServiceType: "1"}, "id:no", PlanPatch{Title: &empty})
			return e
		},
		func() error {
			_, e := l.ListItems(context.Background(), PlanningScope{ServiceType: "1", Plan: "id:0"}, ListOptions{})
			return e
		},
		func() error {
			_, e := l.ListPlans(context.Background(), PlanningScope{ServiceType: "1"}, PlanListOptions{Filter: "bogus"})
			return e
		},
		func() error {
			_, e := l.ListTeams(context.Background(), PlanningScope{ServiceType: "1"}, ListOptions{Limit: -1})
			return e
		},
	}
	for _, run := range cases {
		assertCode(t, run(), "invalid_input")
	}
	if len(f.requests) != 0 {
		t.Fatalf("requests=%+v", f.requests)
	}
}
func TestPlanningRelationshipValidation(t *testing.T) {
	for _, mode := range []string{"wrong_plan", "null_plan", "wrong_service_type", "bad_person", "bad_team", "shared_team", "wrong_membership", "null_membership", "malformed_owner"} {
		t.Run(mode, func(t *testing.T) {
			f := planningFixture(t)
			base := f.handle
			f.handle = func(r request) (string, error) {
				if strings.HasSuffix(r.path, "/team_members") {
					rel := `{"plan":{"data":{"type":"Plan","id":"2"}},"service_type":{"data":{"type":"ServiceType","id":"1"}},"person":{"data":{"type":"Person","id":"7"}},"team":{"data":{"type":"Team","id":"8"}}}`
					switch mode {
					case "wrong_plan":
						rel = strings.Replace(rel, `"Plan","id":"2"`, `"Plan","id":"9"`, 1)
					case "null_plan":
						rel = strings.Replace(rel, `{"type":"Plan","id":"2"}`, `null`, 1)
					case "wrong_service_type":
						rel = strings.Replace(rel, `"ServiceType","id":"1"`, `"ServiceType","id":"9"`, 1)
					case "bad_person":
						rel = strings.Replace(rel, `"Person"`, `"Song"`, 1)
					case "bad_team":
						rel = strings.Replace(rel, `"Team","id":"8"`, `"Team","id":"x"`, 1)
					}
					return songPage(planningResource("PlanPerson", "6", `{"name":"Alex","status":"U"}`, rel), ""), nil
				}
				if strings.HasSuffix(r.path, "/teams") {
					rel := `{"service_type":{"data":{"type":"ServiceType","id":"9"}},"service_types":{"data":[{"type":"ServiceType","id":"1"},{"type":"ServiceType","id":"9"}]}}`
					if mode == "wrong_membership" {
						rel = strings.Replace(rel, `"id":"1"`, `"id":"7"`, 1)
					}
					if mode == "null_membership" {
						rel = `{"service_types":{"data":null}}`
					}
					if mode == "malformed_owner" {
						rel = `{"service_type":{"data":{"type":"Song","id":"1"}}}`
					}
					return songPage(planningResource("Team", "8", `{"name":"Music"}`, rel), ""), nil
				}
				return base(r)
			}
			l := NewLibrary(f)
			var e error
			if mode == "shared_team" || strings.Contains(mode, "membership") || mode == "malformed_owner" {
				_, e = l.ListTeams(context.Background(), PlanningScope{ServiceType: "1"}, ListOptions{})
			} else {
				_, e = l.ListScheduledPeople(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, ListOptions{})
			}
			if mode == "shared_team" {
				if e != nil {
					t.Fatal(e)
				}
			} else if e == nil {
				t.Fatal("invalid relationship accepted")
			}
		})
	}
}
func TestPlanningArchiveSelectionAndStreaming(t *testing.T) {
	f := planningFixture(t)
	base := f.handle
	f.handle = func(r request) (string, error) {
		if r.path == "/services/v2/service_types" {
			return songPage(planningResource("ServiceType", "1", `{"name":"Sunday","archived_at":null}`, "")+","+planningResource("ServiceType", "3", `{"name":"Evening","archived_at":"2025-01-01"}`, ""), ""), nil
		}
		if strings.HasSuffix(r.path, "/plans") {
			if r.query.Get("filter") != "future" {
				t.Fatalf("query=%v", r.query)
			}
			if r.query.Get("offset") == "" {
				return songPage(planningResource("Plan", "2", `{"title":"One"}`, ""), "/services/v2/service_types/1/plans?filter=future&per_page=100&offset=100"), nil
			}
			return songPage(planningResource("Plan", "4", `{"title":"Two"}`, ""), ""), nil
		}
		return base(r)
	}
	l := NewLibrary(f)
	r, e := l.ListServiceTypes(context.Background(), ServiceTypeListOptions{ListOptions: ListOptions{All: true}})
	if e != nil || len(r.Data) != 1 {
		t.Fatalf("%+v %v", r, e)
	}
	r, e = l.ListServiceTypes(context.Background(), ServiceTypeListOptions{ListOptions: ListOptions{All: true}, IncludeArchived: true})
	if e != nil || len(r.Data) != 2 {
		t.Fatalf("%+v %v", r, e)
	}
	_, e = l.ShowServiceType(context.Background(), "Evening", false)
	assertCode(t, e, "not_found")
	got := []string{}
	meta, e := l.WalkPlans(context.Background(), PlanningScope{ServiceType: "1"}, "future", func(p Plan) error { got = append(got, p.ID); return nil })
	if e != nil || !meta.Complete || !reflect.DeepEqual(got, []string{"2", "4"}) {
		t.Fatalf("got=%v meta=%+v err=%v", got, meta, e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, e = l.WalkServiceTypes(ctx, false, func(ServiceType) error { return nil })
	if !errors.Is(e, context.Canceled) {
		t.Fatalf("err=%v", e)
	}
	boom := errors.New("writer failed")
	_, e = l.WalkServiceTypes(context.Background(), false, func(ServiceType) error { return boom })
	if !errors.Is(e, boom) {
		t.Fatalf("err=%v", e)
	}
}
func TestPlanningReadContractKnownTypes(t *testing.T) {
	var contract struct {
		Read map[string]map[string]string `json:"read"`
	}
	raw, e := os.ReadFile("../../testdata/contracts/services-planning.json")
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(raw, &contract); e != nil {
		t.Fatal(e)
	}
	for kind, fields := range contract.Read {
		for name, typ := range fields {
			t.Run(kind+"/"+name, func(t *testing.T) {
				attrs := map[string]json.RawMessage{"name": json.RawMessage(`"Valid"`)}
				bad := `[]`
				if typ == "array" {
					bad = `false`
				}
				attrs[name] = json.RawMessage(bad)
				if e := validatePlanningAttributes(kind, attrs); e == nil {
					t.Fatalf("bad %s accepted", name)
				}
				if typ == "integer" || typ == "boolean" {
					attrs[name] = json.RawMessage(`null`)
					if e := validatePlanningAttributes(kind, attrs); e == nil {
						t.Fatal("null scalar accepted")
					}
				}
			})
		}
	}
}

func TestPlanningConcurrentChangePreventsPatch(t *testing.T) {
	f := planningFixture(t)
	base := f.handle
	f.handle = func(r request) (string, error) {
		if strings.HasSuffix(r.path, "/plans") {
			return songPage(planningResource("Plan", "2", `{"title":"Gather"}`, ""), ""), nil
		}
		if strings.HasSuffix(r.path, "/plans/2") {
			return planningSingle("Plan", "2", `{"title":"Changed"}`, ""), nil
		}
		return base(r)
	}
	title := "New"
	_, e := NewLibrary(f).UpdatePlan(context.Background(), PlanningScope{ServiceType: "1"}, "Gather", PlanPatch{Title: &title})
	assertCode(t, e, "concurrent_change")
	if f.writes != 0 {
		t.Fatal("write after changed identity")
	}
}

func TestPlanningNamedPatchAllowsReadbackRelationshipExpansion(t *testing.T) {
	f := planningFixture(t)
	base := f.handle
	submitted := false
	f.handle = func(r request) (string, error) {
		if strings.HasSuffix(r.path, "/plans") {
			return songPage(planningResource("Plan", "2", `{"title":"Gather"}`, ""), ""), nil
		}
		if strings.HasSuffix(r.path, "/plans/2") {
			title := "Gather"
			if r.method == "PATCH" {
				submitted = true
			}
			if submitted {
				title = "New"
			}
			return planningSingle("Plan", "2", fmt.Sprintf(`{"title":%q}`, title), `{"service_type":{"data":{"id":"1","type":"ServiceType"}}}`), nil
		}
		return base(r)
	}
	title := "New"
	_, e := NewLibrary(f).UpdatePlan(context.Background(), PlanningScope{ServiceType: "1"}, "Gather", PlanPatch{Title: &title})
	if e != nil || f.writes != 1 {
		t.Fatalf("e=%v writes=%d", e, f.writes)
	}
}
