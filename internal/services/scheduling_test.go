package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/joe--cool/pccli/v2/internal/planningcenter"
)

func schedulingFixture(t *testing.T) *fixture {
	f := planningFixture(t)
	base := f.handle
	f.handle = func(r request) (string, error) {
		switch r.path {
		case "/services/v2/service_types/1/teams/5":
			return planningSingle("Team", "5", `{"name":"Band","schedule_to":"plan","archived_at":null}`, `{"service_type":{"data":{"type":"ServiceType","id":"1"}}}`), nil
		case "/services/v2/teams/5/team_positions":
			return songPage(planningResource("TeamPosition", "7", `{"name":"Vocals","sequence":1}`, `{"team":{"data":{"type":"Team","id":"5"}}}`), ""), nil
		case "/services/v2/teams/5/people":
			return songPage(planningResource("Person", "6", `{"full_name":"Alex Sample","archived":false,"archived_at":null}`, ""), ""), nil
		case "/services/v2/service_types/1/plans/2/team_members/40":
			return planningSingle("PlanPerson", "40", `{"name":"Alex Sample","status":"C","notes":"Before","prepare_notification":true,"can_accept_partial":false,"team_position_name":"Vocals"}`, assignmentRelationships("6", "5")), nil
		case "/services/v2/service_types/1/plans/2/needed_positions/50":
			return planningSingle("NeededPosition", "50", `{"quantity":3,"team_position_name":"Vocals","scheduled_to":"provider-raw"}`, neededRelationships("5", "")), nil
		}
		return base(r)
	}
	return f
}
func assignmentRelationships(person, team string) string {
	return `{"person":{"data":{"type":"Person","id":"` + person + `"}},"team":{"data":{"type":"Team","id":"` + team + `"}},"plan":{"data":{"type":"Plan","id":"2"}}}`
}
func neededRelationships(team, timeID string) string {
	timeRel := `null`
	if timeID != "" {
		timeRel = `{"type":"PlanTime","id":"` + timeID + `"}`
	}
	return `{"team":{"data":{"type":"Team","id":"` + team + `"}},"plan":{"data":{"type":"Plan","id":"2"}},"time":{"data":` + timeRel + `},"time_preference_option":{"data":null}}`
}
func TestSchedulingCreateDefaultsAndReadback(t *testing.T) {
	for _, mode := range []string{"verified", "wrong_person", "wrong_status", "wrong_preparation", "late_failure"} {
		t.Run(mode, func(t *testing.T) {
			f := schedulingFixture(t)
			base := f.handle
			posted := false
			f.handle = func(r request) (string, error) {
				if r.method == "POST" {
					posted = true
					b, _ := json.Marshal(r.body)
					want := `{"data":{"attributes":{"notes":"","person_id":"6","prepare_notification":false,"status":"U","team_id":"5","team_position_name":"Vocals"},"type":"PlanPerson"}}`
					if string(b) != want {
						t.Fatalf("body=%s", b)
					}
					return planningSingle("PlanPerson", "40", `{}`, ""), nil
				}
				if posted && strings.HasSuffix(r.path, "/team_members/40") {
					if mode == "late_failure" {
						return "", errors.New("read failed")
					}
					attrs := `{"name":"Alex Sample","notes":"","status":"Unconfirmed","prepare_notification":false,"team_position_name":"Vocals"}`
					rel := assignmentRelationships("6", "5")
					if mode == "wrong_person" {
						rel = assignmentRelationships("9", "5")
					}
					if mode == "wrong_status" {
						attrs = strings.Replace(attrs, "Unconfirmed", "Confirmed", 1)
					}
					if mode == "wrong_preparation" {
						attrs = strings.Replace(attrs, `"prepare_notification":false`, `"prepare_notification":true`, 1)
					}
					return planningSingle("PlanPerson", "40", attrs, rel), nil
				}
				return base(r)
			}
			empty := ""
			got, e := NewLibrary(f).CreateScheduledPerson(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, ScheduledPersonCreate{Team: "5", Person: "6", Position: "7", Notes: &empty})
			if mode == "verified" {
				if e != nil || got.PersonID == nil || *got.PersonID != "6" || got.PrepareNotification == nil || *got.PrepareNotification {
					t.Fatalf("got=%+v e=%v", got, e)
				}
			} else {
				assertCode(t, e, "verification_failed")
				var se *Error
				if !errors.As(e, &se) || se.PlanningRecovery == nil || se.PlanningRecovery.TargetID != "40" || len(se.PlanningRecovery.Created) != 1 {
					t.Fatalf("missing recovery=%v", e)
				}
			}
			if f.writes != 1 || f.uploads != 0 {
				t.Fatalf("writes=%d uploads=%d", f.writes, f.uploads)
			}
		})
	}
}
func TestSchedulingSelectionFailuresHaveNoWrites(t *testing.T) {
	for _, mode := range []string{"duplicate_position", "duplicate_person", "late_membership_failure", "wrong_position_parent", "archived_person", "unknown_archive", "split_team", "unknown_team_mode", "wrong_member_type"} {
		t.Run(mode, func(t *testing.T) {
			f := schedulingFixture(t)
			base := f.handle
			f.handle = func(r request) (string, error) {
				if r.path == "/services/v2/service_types/1/teams/5" && oneOf(mode, "split_team", "unknown_team_mode") {
					attrs := `{"name":"Band","schedule_to":"time"}`
					if mode == "unknown_team_mode" {
						attrs = `{"name":"Band"}`
					}
					return planningSingle("Team", "5", attrs, ""), nil
				}
				if r.path == "/services/v2/teams/5/team_positions" && mode == "duplicate_position" {
					return songPage(planningResource("TeamPosition", "7", `{"name":"Vocals"}`, "")+","+planningResource("TeamPosition", "8", `{"name":"VOCALS"}`, ""), ""), nil
				}
				if r.path == "/services/v2/teams/5/team_positions" && mode == "wrong_position_parent" {
					return songPage(planningResource("TeamPosition", "7", `{"name":"Vocals"}`, `{"team":{"data":{"type":"Team","id":"9"}}}`), ""), nil
				}
				if r.path == "/services/v2/teams/5/people" {
					attrs := `{"full_name":"Alex Sample","archived":false}`
					kind := "Person"
					if mode == "archived_person" {
						attrs = `{"full_name":"Alex Sample","archived":true}`
					}
					if mode == "unknown_archive" {
						attrs = `{"full_name":"Alex Sample"}`
					}
					if mode == "wrong_member_type" {
						kind = "Team"
					}
					item := planningResource(kind, "6", attrs, "")
					if mode == "duplicate_person" {
						item += "," + planningResource("Person", "9", attrs, "")
					}
					if mode == "late_membership_failure" {
						if r.query.Get("offset") != "" {
							return "", errors.New("late page failed")
						}
						return songPage(item, "/services/v2/teams/5/people?per_page=100&offset=100"), nil
					}
					return songPage(item, ""), nil
				}
				return base(r)
			}
			person := "6"
			if mode == "duplicate_person" {
				person = "Alex Sample"
			}
			_, e := NewLibrary(f).CreateScheduledPerson(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, ScheduledPersonCreate{Team: "5", Person: person, Position: "7"})
			if e == nil || f.writes != 0 || f.uploads != 0 {
				t.Fatalf("e=%v writes=%d", e, f.writes)
			}
		})
	}
}
func TestSchedulingPatchPreservesPreparedStateAndCanonicalizesStatus(t *testing.T) {
	f := schedulingFixture(t)
	base := f.handle
	written := false
	f.handle = func(r request) (string, error) {
		if r.method == "PATCH" {
			written = true
			b, _ := json.Marshal(r.body)
			if string(b) != `{"data":{"attributes":{"notes":"","status":"D"},"type":"PlanPerson"}}` || strings.Contains(string(b), "prepare_notification") {
				t.Fatalf("body=%s", b)
			}
			return planningSingle("PlanPerson", "40", `{}`, ""), nil
		}
		if written && strings.HasSuffix(r.path, "/team_members/40") {
			return planningSingle("PlanPerson", "40", `{"name":"Alex Sample","notes":"","status":"Declined","prepare_notification":true}`, assignmentRelationships("6", "5")), nil
		}
		return base(r)
	}
	empty, status := "", "declined"
	got, e := NewLibrary(f).UpdateScheduledPerson(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, "40", ScheduledPersonPatch{Notes: &empty, Status: &status})
	if e != nil || got.PrepareNotification == nil || !*got.PrepareNotification || got.Status == nil || *got.Status != "Declined" {
		t.Fatalf("got=%+v e=%v", got, e)
	}
}
func TestSchedulingDeleteRemovesStaleAssignment(t *testing.T) {
	f := schedulingFixture(t)
	base := f.handle
	deleted := false
	f.handle = func(r request) (string, error) {
		if strings.HasSuffix(r.path, "/people") {
			t.Fatal("delete must not require current membership")
		}
		if r.method == "DELETE" {
			deleted = true
			return "", nil
		}
		if deleted && strings.HasSuffix(r.path, "/team_members/40") {
			return "", planningcenter.APIError{StatusCode: 404}
		}
		return base(r)
	}
	if e := NewLibrary(f).DeleteScheduledPerson(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, "40", ""); e != nil {
		t.Fatal(e)
	}
	if f.writes != 1 {
		t.Fatal(f.writes)
	}
}
func TestSchedulingInvalidTypedInputBeforeNetwork(t *testing.T) {
	f := schedulingFixture(t)
	l := NewLibrary(f)
	bad := "maybe"
	for _, p := range []ScheduledPersonCreate{{}, {Team: "5", Person: "6"}, {Team: "5", Person: "6", Position: "7", Status: &bad}, {Team: "5", Person: "id:bad", Position: "7"}} {
		_, e := l.CreateScheduledPerson(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, p)
		assertCode(t, e, "invalid_input")
	}
	_, e := l.UpdateScheduledPerson(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, "40", ScheduledPersonPatch{})
	assertCode(t, e, "invalid_input")
	if len(f.requests) > 0 {
		t.Fatal(f.requests)
	}
}
func TestTeamPeopleArchiveSelectionAndScope(t *testing.T) {
	f := schedulingFixture(t)
	base := f.handle
	f.handle = func(r request) (string, error) {
		if r.path == teamPeoplePath("5") {
			return songPage(planningResource("Person", "6", `{"full_name":"Alex Sample","archived":false}`, "")+","+planningResource("Person", "9", `{"full_name":"Past Member","archived":true}`, "")+","+planningResource("Person", "10", `{"full_name":null}`, ""), ""), nil
		}
		return base(r)
	}
	l := NewLibrary(f)
	r, e := l.ListTeamPeople(context.Background(), TeamScope{ServiceType: "1", Team: "5"}, ListOptions{All: true})
	if e != nil || len(r.Data) != 2 || r.Data[0].TeamID != "5" || r.Data[1].FullName != nil {
		t.Fatalf("r=%+v e=%v", r, e)
	}
	r, e = l.ListTeamPeople(context.Background(), TeamScope{ServiceType: "1", Team: "5", IncludeArchived: true}, ListOptions{All: true})
	if e != nil || len(r.Data) != 3 {
		t.Fatalf("r=%+v e=%v", r, e)
	}
}

func TestAssignmentPatchChecksImmutableReadbackAndStatusNormalization(t *testing.T) {
	for _, mode := range []string{"normalized", "wrong_person", "wrong_team"} {
		t.Run(mode, func(t *testing.T) {
			f := schedulingFixture(t)
			base := f.handle
			written := false
			f.handle = func(r request) (string, error) {
				if strings.HasSuffix(r.path, "/team_members") {
					return songPage(planningResource("PlanPerson", "40", `{"name":"Alex Sample","status":"Confirmed","notes":"Before","prepare_notification":true,"can_accept_partial":false,"team_position_name":"Vocals"}`, assignmentRelationships("6", "5")), ""), nil
				}
				if r.method == "PATCH" {
					written = true
					return planningSingle("PlanPerson", "40", `{}`, ""), nil
				}
				if written && strings.HasSuffix(r.path, "/team_members/40") {
					rel := assignmentRelationships("6", "5")
					if mode == "wrong_person" {
						rel = assignmentRelationships("9", "5")
					}
					if mode == "wrong_team" {
						rel = assignmentRelationships("6", "9")
					}
					return planningSingle("PlanPerson", "40", `{"notes":"","status":"C"}`, rel), nil
				}
				return base(r)
			}
			empty := ""
			_, e := NewLibrary(f).UpdateScheduledPerson(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, "Alex Sample", ScheduledPersonPatch{Notes: &empty})
			if mode == "normalized" {
				if e != nil {
					t.Fatal(e)
				}
			} else {
				assertCode(t, e, "verification_failed")
				var failure *Error
				if !errors.As(e, &failure) || failure.PlanningRecovery == nil {
					t.Fatalf("missing recovery: %v", e)
				}
			}
			if f.writes != 1 {
				t.Fatal(f.writes)
			}
		})
	}
}

func TestAssignmentUnexpectedPartialReadbackRetainsReceipt(t *testing.T) {
	f := schedulingFixture(t)
	base := f.handle
	posted := false
	f.handle = func(r request) (string, error) {
		if r.method == "POST" {
			posted = true
			return planningSingle("PlanPerson", "40", `{}`, ""), nil
		}
		if posted && strings.HasSuffix(r.path, "/team_members/40") {
			return planningSingle("PlanPerson", "40", `{"status":"U","prepare_notification":false,"team_position_name":"Vocals","can_accept_partial":true}`, assignmentRelationships("6", "5")), nil
		}
		return base(r)
	}
	_, e := NewLibrary(f).CreateScheduledPerson(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, ScheduledPersonCreate{Team: "5", Person: "6", Position: "7"})
	assertCode(t, e, "verification_failed")
	var se *Error
	if !errors.As(e, &se) || se.PlanningRecovery == nil || se.PlanningRecovery.TargetID != "40" {
		t.Fatalf("e=%v", e)
	}
}
