package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/joe--cool/pccli/v2/internal/planningcenter"
)

type copyTestRow struct {
	kind, id string
	attrs    map[string]any
	rel      map[string]any
}

func (r copyTestRow) value() map[string]any {
	return map[string]any{"type": r.kind, "id": r.id, "attributes": r.attrs, "relationships": r.rel}
}
func copyTestRel(kind, id string) map[string]any {
	return map[string]any{"data": map[string]any{"type": kind, "id": id}}
}
func copyTestJSON(value any) string         { b, _ := json.Marshal(value); return string(b) }
func copyTestSingle(row copyTestRow) string { return copyTestJSON(map[string]any{"data": row.value()}) }
func copyTestPage(rows []copyTestRow) string {
	data := []any{}
	for _, row := range rows {
		data = append(data, row.value())
	}
	return copyTestJSON(map[string]any{"data": data})
}

type copyAccount struct {
	f           *fixture
	mode        string
	source      map[string][]copyTestRow
	sourcePlan  copyTestRow
	destination *copyTestRow
	dest        map[string][]copyTestRow
	counts      map[string]int
	finalReady  bool
}

func newCopyAccount(t *testing.T, mode string) *copyAccount {
	a := &copyAccount{mode: mode, source: map[string][]copyTestRow{}, dest: map[string][]copyTestRow{}, counts: map[string]int{}}
	a.sourcePlan = copyTestRow{"Plan", "2", map[string]any{"title": "Source", "series_title": "Life Together", "public": true, "reminders_disabled": true, "items_count": 2, "plan_people_count": 1, "needed_positions_count": 1}, map[string]any{"service_type": copyTestRel("ServiceType", "1")}}
	a.source["PlanTime"] = []copyTestRow{{"PlanTime", "30", map[string]any{"name": "Morning", "starts_at": "2026-11-01T09:00:00-08:00", "ends_at": "2026-11-01T10:00:00-08:00", "time_type": "service"}, map[string]any{"plan": copyTestRel("Plan", "2")}}}
	a.source["Item"] = []copyTestRow{
		{"Item", "20", map[string]any{"title": "Gathering", "item_type": "header", "sequence": 1, "length": 0, "service_position": "during"}, map[string]any{"plan": copyTestRel("Plan", "2")}},
		{"Item", "21", map[string]any{"title": "Grace", "item_type": "song", "sequence": 2, "length": 240, "service_position": "during", "key_name": "G"}, map[string]any{"plan": copyTestRel("Plan", "2"), "song": copyTestRel("Song", "10"), "arrangement": copyTestRel("Arrangement", "11"), "key": copyTestRel("Key", "12")}},
	}
	a.source["PlanPerson"] = []copyTestRow{{"PlanPerson", "40", map[string]any{"name": "Alex Sample", "status": "C", "notes": "Bring folder", "prepare_notification": true, "can_accept_partial": false, "team_position_name": "Vocals"}, map[string]any{"plan": copyTestRel("Plan", "2"), "team": copyTestRel("Team", "5"), "person": copyTestRel("Person", "6")}}}
	a.source["NeededPosition"] = []copyTestRow{{"NeededPosition", "50", map[string]any{"quantity": 2, "team_position_name": "Vocals", "scheduled_to": "raw"}, map[string]any{"plan": copyTestRel("Plan", "2"), "team": copyTestRel("Team", "5"), "time": map[string]any{"data": nil}, "time_preference_option": map[string]any{"data": nil}}}}
	switch mode {
	case "source_id_collision":
		a.sourcePlan.attrs["public"] = false
	case "missing_key":
		delete(a.source["Item"][1].rel, "key")
		delete(a.source["Item"][1].attrs, "key_name")
	case "null_key":
		a.source["Item"][1].rel["key"] = map[string]any{"data": nil}
		delete(a.source["Item"][1].attrs, "key_name")
	case "known_key_without_id":
		delete(a.source["Item"][1].rel, "key")
	case "media":
		a.source["Item"][1].attrs["item_type"] = "media"
	case "unknown_type":
		delete(a.source["Item"][1].attrs, "item_type")
	case "layout":
		a.source["Item"][1].rel["selected_layout"] = copyTestRel("Layout", "99")
	case "contradictory_sequence":
		a.source["Item"][1].attrs["sequence"] = 0
	case "overflow":
		a.source["PlanTime"][0].attrs["starts_at"] = "9999-12-31T09:00:00Z"
		a.source["PlanTime"][0].attrs["ends_at"] = "9999-12-31T10:00:00Z"
	case "unknown_settings":
		delete(a.sourcePlan.attrs, "reminders_disabled")
	}
	a.f = &fixture{t: t}
	a.f.handle = a.handle
	return a
}
func (a *copyAccount) handle(r request) (string, error) {
	a.counts[r.method+" "+r.path]++
	if r.path == "/services/v2/service_types/1" {
		return planningSingle("ServiceType", "1", `{"name":"Sunday","archived_at":null}`, ""), nil
	}
	if r.path == "/services/v2/service_types/1/teams/5" {
		return planningSingle("Team", "5", `{"name":"Band","schedule_to":"plan","archived_at":null}`, ""), nil
	}
	if r.path == "/services/v2/teams/5/people" {
		return songPage(planningResource("Person", "6", `{"full_name":"Alex Sample","archived":false}`, ""), ""), nil
	}
	if r.path == "/services/v2/teams/5/team_positions" {
		name := "Vocals"
		calls := a.counts[r.method+" "+r.path]
		if a.mode == "position_drift_before" && calls >= 2 || a.mode == "position_drift_child" && calls >= 3 {
			name = "Solo"
		}
		return songPage(planningResource("TeamPosition", "7", fmt.Sprintf(`{"name":%q}`, name), `{"team":{"data":{"type":"Team","id":"5"}}}`), ""), nil
	}
	if r.path == "/services/v2/songs/10" {
		return `{"data":` + songJSON("10", "Grace", false) + `}`, nil
	}
	if r.path == "/services/v2/songs/10/arrangements/11" {
		return planningSingle("Arrangement", "11", `{"name":"Band","archived_at":null}`, ""), nil
	}
	if r.path == "/services/v2/songs/10/arrangements/11/keys/12" {
		return planningSingle("Key", "12", `{"name":"G"}`, ""), nil
	}
	plans := "/services/v2/service_types/1/plans"
	if r.path == plans && r.method == "GET" {
		if a.mode == "late_existing_lookup" {
			if r.query.Get("offset") != "" {
				return "", errors.New("late existing lookup failed")
			}
			return songPage(copyTestJSON(a.sourcePlan.value()), plans+"?per_page=100&offset=100"), nil
		}
		return copyTestPage([]copyTestRow{a.sourcePlan, {"Plan", "4", map[string]any{"title": "Other", "public": false}, map[string]any{}}}), nil
	}
	if r.path == plans+"/2" {
		row := a.sourcePlan
		if a.mode == "named_rename" || a.mode == "source_changed" && a.counts[r.method+" "+r.path] >= 3 {
			copy := map[string]any{}
			for k, v := range row.attrs {
				copy[k] = v
			}
			copy["title"] = "Renamed"
			row.attrs = copy
		}
		return copyTestSingle(row), nil
	}
	attrs := func() map[string]any {
		if r.body == nil {
			return nil
		}
		return r.body.(map[string]any)["data"].(map[string]any)["attributes"].(map[string]any)
	}
	if r.path == plans && r.method == "POST" {
		id := "3"
		if a.mode == "source_id_collision" {
			id = "2"
		}
		if a.mode == "existing_id_collision" {
			id = "4"
		}
		a.destination = &copyTestRow{"Plan", id, attrs(), map[string]any{"service_type": copyTestRel("ServiceType", "1")}}
		return planningSingle("Plan", id, `{}`, ""), nil
	}
	if r.path == plans+"/4" {
		return planningSingle("Plan", "4", `{"title":"Other","public":false}`, ""), nil
	}
	if r.path == plans+"/3" {
		if a.destination == nil {
			return "", errors.New("destination not created")
		}
		if r.method == "PATCH" {
			for k, v := range attrs() {
				a.destination.attrs[k] = v
			}
		}
		if a.finalReady && oneOf(a.mode, "final_read_failure", "final_needed_failure") {
			return "", planningcenter.APIError{StatusCode: 500}
		}
		a.destination.attrs["items_count"] = len(a.dest["Item"])
		a.destination.attrs["plan_people_count"] = len(a.dest["PlanPerson"])
		a.destination.attrs["needed_positions_count"] = len(a.dest["NeededPosition"])
		return copyTestSingle(*a.destination), nil
	}
	paths := map[string]string{"plan_times": "PlanTime", "items": "Item", "team_members": "PlanPerson", "needed_positions": "NeededPosition"}
	for suffix, kind := range paths {
		sourcePath := plans + "/2/" + suffix
		destPath := plans + "/3/" + suffix
		if r.path == sourcePath {
			rows := a.source[kind]
			if a.mode == "semantic_time" && kind == "PlanTime" && a.counts[r.method+" "+r.path] >= 2 {
				row := rows[0]
				copy := map[string]any{}
				for k, v := range row.attrs {
					copy[k] = v
				}
				copy["starts_at"] = "2026-11-01T17:00:00Z"
				copy["ends_at"] = "2026-11-01T18:00:00Z"
				row.attrs = copy
				rows = []copyTestRow{row}
			}
			if a.mode == "semantic_status" && kind == "PlanPerson" && a.counts[r.method+" "+r.path] >= 2 {
				row := rows[0]
				copy := map[string]any{}
				for k, v := range row.attrs {
					copy[k] = v
				}
				copy["status"] = "Confirmed"
				row.attrs = copy
				rows = []copyTestRow{row}
			}
			if a.mode == "items_changed" && kind == "Item" && a.counts[r.method+" "+r.path] >= 2 {
				return songPage(planningResource("Item", "20", `{"title":"Changed","item_type":"header"}`, ""), ""), nil
			}
			return copyTestPage(rows), nil
		}
		if r.path == destPath {
			if r.method == "GET" {
				return copyTestPage(a.dest[kind]), nil
			}
			if r.method != "POST" {
				a.f.t.Fatalf("unexpected %s %s", r.method, r.path)
			}
			if kind == "PlanTime" {
				switch a.mode {
				case "api403":
					return "", planningcenter.APIError{StatusCode: 403}
				case "api500":
					return "", planningcenter.APIError{StatusCode: 500}
				case "connection_lost":
					return "", errors.New("connection lost")
				case "cancel":
					return "", &planningcenter.TransportError{Operation: "POST", UnknownOutcome: true, Cause: context.Canceled}
				}
			}
			ids := map[string]int{"PlanTime": 130, "Item": 120, "PlanPerson": 140, "NeededPosition": 150}
			id := fmt.Sprint(ids[kind] + len(a.dest[kind]))
			if kind == "Item" && a.mode == "source_item_collision" {
				id = "20"
			}
			if kind == "Item" && a.mode == "duplicate_child_error" {
				id = "120"
			}
			values := attrs()
			rel := map[string]any{"plan": copyTestRel("Plan", "3")}
			if kind == "Item" {
				values["sequence"] = len(a.dest[kind]) + 1
				if song, ok := values["song_id"].(string); ok {
					values["item_type"] = "song"
					rel["song"] = copyTestRel("Song", song)
					rel["arrangement"] = copyTestRel("Arrangement", values["arrangement_id"].(string))
					if key, ok := values["key_id"].(string); ok {
						rel["key"] = copyTestRel("Key", key)
					}
				} else if _, ok := values["item_type"]; !ok {
					values["item_type"] = "item"
				}
			}
			if kind == "PlanPerson" {
				if values["prepare_notification"] != false || values["status"] != "U" {
					a.f.t.Fatalf("unsafe copied assignment: %v", values)
				}
				values["name"] = "Alex Sample"
				values["can_accept_partial"] = false
				rel["person"] = copyTestRel("Person", values["person_id"].(string))
				rel["team"] = copyTestRel("Team", values["team_id"].(string))
			}
			if kind == "NeededPosition" {
				values["team_position_name"] = "Vocals"
				rel["team"] = copyTestRel("Team", values["team_id"].(string))
				rel["time"] = map[string]any{"data": nil}
				rel["time_preference_option"] = map[string]any{"data": nil}
			}
			a.dest[kind] = append(a.dest[kind], copyTestRow{kind, id, values, rel})
			return planningSingle(kind, id, `{}`, ""), nil
		}
		if strings.HasPrefix(r.path, destPath+"/") {
			id := strings.TrimPrefix(r.path, destPath+"/")
			if kind == "PlanTime" && a.mode == "child_read_failure" {
				return "", planningcenter.APIError{StatusCode: 403}
			}
			if kind == "Item" && a.mode == "duplicate_child_error" && len(a.dest[kind]) == 2 {
				return planningSingle("Item", id, `{"title":"Wrong"}`, ""), nil
			}
			for _, row := range a.dest[kind] {
				if row.id == id {
					if kind == "PlanTime" && a.mode == "final_read_failure" || kind == "NeededPosition" && a.mode == "final_needed_failure" {
						a.finalReady = true
					}
					return copyTestSingle(row), nil
				}
			}
			return "", planningcenter.APIError{StatusCode: 404}
		}
	}
	if r.path == plans+"/3/item_reorder" {
		desired := attrs()["sequence"].([]string)
		rows := []copyTestRow{}
		for i, id := range desired {
			for _, row := range a.dest["Item"] {
				if row.id == id {
					row.attrs["sequence"] = i + 1
					rows = append(rows, row)
				}
			}
		}
		a.dest["Item"] = rows
		return "", nil
	}
	a.f.t.Fatalf("unexpected request %s %s", r.method, r.path)
	return "", nil
}
func TestPlanCopyFullJourneyAndRefreshedResult(t *testing.T) {
	a := newCopyAccount(t, "")
	offset := "168h"
	got, e := NewLibrary(a.f).CopyPlan(context.Background(), PlanningScope{ServiceType: "1"}, "2", PlanCopyOptions{CopyItems: true, CopyTimes: true, CopyPeople: true, CopyNeededPositions: true, TimeOffset: &offset})
	if e != nil {
		t.Fatal(e)
	}
	if got.Plan.ID != "3" || got.SourcePlanID != "2" || got.Outcome != "accepted" || got.Plan.ItemsCount == nil || *got.Plan.ItemsCount != 2 || got.Plan.PlanPeopleCount == nil || *got.Plan.PlanPeopleCount != 1 || len(got.Created) != 6 || len(got.UnverifiedFields) != 1 {
		t.Fatalf("got=%+v", got)
	}
	if got.Plan.Public == nil || *got.Plan.Public {
		t.Fatal("copy must explicitly set public false by default")
	}
	if a.dest["PlanTime"][0].attrs["starts_at"] != "2026-11-08T09:00:00-08:00" {
		t.Fatal(a.dest["PlanTime"])
	}
	if a.dest["PlanPerson"][0].attrs["status"] != "U" || a.dest["PlanPerson"][0].attrs["prepare_notification"] != false {
		t.Fatal(a.dest["PlanPerson"])
	}
	if got.UnverifiedFields[0].ID != "150" || got.UnverifiedFields[0].RequestedID != "7" {
		t.Fatal(got.UnverifiedFields)
	}
	if a.f.writes != 7 {
		t.Fatalf("writes=%d", a.f.writes)
	}
}
func TestPlanCopyPreflightFailuresHaveZeroEffects(t *testing.T) {
	for _, mode := range []string{"missing_key", "null_key", "known_key_without_id", "media", "unknown_type", "layout", "contradictory_sequence", "overflow", "unknown_settings", "late_existing_lookup", "named_rename", "source_changed", "items_changed", "position_drift_before"} {
		t.Run(mode, func(t *testing.T) {
			a := newCopyAccount(t, mode)
			offset := "168h"
			o := PlanCopyOptions{CopyItems: true, CopyTimes: true, TimeOffset: &offset}
			target := "2"
			if mode == "unknown_settings" {
				o = PlanCopyOptions{CopySettings: true}
			}
			if mode == "position_drift_before" {
				o = PlanCopyOptions{CopyPeople: true}
			}
			if mode == "named_rename" {
				target = "Source"
			}
			_, e := NewLibrary(a.f).CopyPlan(context.Background(), PlanningScope{ServiceType: "1"}, target, o)
			if e == nil || a.f.writes != 0 || a.f.uploads != 0 {
				t.Fatalf("e=%v writes=%d uploads=%d", e, a.f.writes, a.f.uploads)
			}
		})
	}
}
func TestPlanCopyCollisionResponsesNeverBecomeCreatedReceipts(t *testing.T) {
	for _, mode := range []string{"source_id_collision", "existing_id_collision", "source_item_collision", "duplicate_child_error"} {
		t.Run(mode, func(t *testing.T) {
			a := newCopyAccount(t, mode)
			_, e := NewLibrary(a.f).CopyPlan(context.Background(), PlanningScope{ServiceType: "1"}, "2", PlanCopyOptions{CopyItems: true})
			assertCode(t, e, "outcome_unknown")
			var failure *Error
			if !errors.As(e, &failure) || failure.PlanningRecovery == nil {
				t.Fatalf("missing recovery %v", e)
			}
			r := failure.PlanningRecovery
			if r.LastOutcome != "unknown" || len(r.Observed) != 1 {
				t.Fatalf("recovery=%+v", r)
			}
			expected := 0
			if mode == "source_item_collision" {
				expected = 1
			}
			if mode == "duplicate_child_error" {
				expected = 2
			}
			if len(r.Created) != expected {
				t.Fatalf("created=%v expected=%d", r.Created, expected)
			}
			for _, receipt := range r.Created {
				if receipt.Resource == "Plan" && (receipt.ID == "2" || receipt.ID == "4") {
					t.Fatalf("existing plan recorded as new: %+v", r)
				}
			}
			if mode == "source_id_collision" || mode == "existing_id_collision" {
				if len(a.dest["Item"]) != 0 {
					t.Fatal("child writes reached existing plan")
				}
			}
		})
	}
}
func TestPlanCopyFailureClassificationAndReceipts(t *testing.T) {
	for _, mode := range []string{"api403", "api500", "connection_lost", "cancel", "child_read_failure", "final_read_failure", "final_needed_failure", "position_drift_child"} {
		t.Run(mode, func(t *testing.T) {
			a := newCopyAccount(t, mode)
			o := PlanCopyOptions{CopyTimes: true, PreserveDates: true}
			if mode == "final_needed_failure" {
				o = PlanCopyOptions{CopyNeededPositions: true}
			}
			if mode == "position_drift_child" {
				o = PlanCopyOptions{CopyPeople: true}
			}
			_, e := NewLibrary(a.f).CopyPlan(context.Background(), PlanningScope{ServiceType: "1"}, "2", o)
			var failure *Error
			if !errors.As(e, &failure) || failure.PlanningRecovery == nil {
				t.Fatalf("missing recovery %v", e)
			}
			r := failure.PlanningRecovery
			switch mode {
			case "api403", "position_drift_child":
				assertCode(t, e, "partial_success")
				if r.LastOutcome != "rejected" {
					t.Fatal(r.LastOutcome)
				}
			case "api500", "connection_lost", "cancel":
				assertCode(t, e, "outcome_unknown")
				if r.LastOutcome != "unknown" {
					t.Fatal(r.LastOutcome)
				}
			default:
				assertCode(t, e, "verification_failed")
				if r.LastOutcome != "verification_failed" {
					t.Fatal(r.LastOutcome)
				}
			}
			if len(r.Created) < 1 || r.Created[0].ID != "3" || r.PlanID != "3" || r.SourcePlanID != "2" {
				t.Fatalf("recovery=%+v", r)
			}
			if oneOf(mode, "child_read_failure", "final_read_failure", "final_needed_failure") && len(r.Created) != 2 {
				t.Fatalf("lost child receipt %+v", r)
			}
			if mode == "cancel" && !errors.Is(e, context.Canceled) {
				t.Fatal("cancellation cause lost")
			}
			if mode == "final_needed_failure" && (len(r.UnverifiedFields) != 1 || r.UnverifiedFields[0].ID != "150" || r.UnverifiedFields[0].RequestedID != "7") {
				t.Fatalf("lost accepted context %+v", r)
			}
		})
	}
}
func TestPlanCopySemanticSnapshotsAndSettings(t *testing.T) {
	for _, mode := range []string{"semantic_time", "semantic_status", "settings"} {
		t.Run(mode, func(t *testing.T) {
			a := newCopyAccount(t, mode)
			o := PlanCopyOptions{CopyTimes: true, PreserveDates: true}
			if mode == "semantic_status" {
				o = PlanCopyOptions{CopyPeople: true}
			}
			if mode == "settings" {
				o = PlanCopyOptions{CopySettings: true}
			}
			got, e := NewLibrary(a.f).CopyPlan(context.Background(), PlanningScope{ServiceType: "1"}, "2", o)
			if e != nil {
				t.Fatal(e)
			}
			if mode == "settings" && (got.Plan.Public == nil || !*got.Plan.Public || got.Plan.RemindersDisabled == nil || !*got.Plan.RemindersDisabled) {
				t.Fatalf("settings=%+v", got.Plan)
			}
		})
	}
}
func TestPlanCopyOptionsValidateBeforeAnyRequests(t *testing.T) {
	a := newCopyAccount(t, "")
	offset := "0"
	bad := "tomorrow"
	for _, o := range []PlanCopyOptions{{}, {CopyTimes: true}, {CopyItems: true, TimeOffset: &offset}, {CopyTimes: true, PreserveDates: true, TimeOffset: &offset}, {CopyTimes: true, TimeOffset: &bad}} {
		_, e := NewLibrary(a.f).CopyPlan(context.Background(), PlanningScope{ServiceType: "1"}, "2", o)
		assertCode(t, e, "invalid_input")
	}
	if len(a.f.requests) != 0 {
		t.Fatal(a.f.requests)
	}
}

func TestPlanCopyElapsedOffsetsPreserveDuration(t *testing.T) {
	for _, tc := range []struct{ offset, start, end string }{
		{"0", "2026-11-01T09:00:00-08:00", "2026-11-01T10:00:00-08:00"},
		{"-24h", "2026-10-31T09:00:00-08:00", "2026-10-31T10:00:00-08:00"},
	} {
		t.Run(tc.offset, func(t *testing.T) {
			a := newCopyAccount(t, "")
			_, e := NewLibrary(a.f).CopyPlan(context.Background(), PlanningScope{ServiceType: "1"}, "2", PlanCopyOptions{CopyTimes: true, TimeOffset: &tc.offset})
			if e != nil {
				t.Fatal(e)
			}
			rows := a.dest["PlanTime"]
			if len(rows) != 1 {
				t.Fatalf("copied times=%+v", rows)
			}
			for field, expected := range map[string]string{"starts_at": tc.start, "ends_at": tc.end} {
				got, err := time.Parse(time.RFC3339Nano, rows[0].attrs[field].(string))
				want, _ := time.Parse(time.RFC3339Nano, expected)
				if err != nil || !got.Equal(want) {
					t.Fatalf("%s=%v want same instant as %s", field, rows[0].attrs[field], expected)
				}
			}
		})
	}
}
