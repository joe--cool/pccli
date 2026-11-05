package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/joe--cool/pccli/v2/internal/planningcenter"
)

func TestNeededPositionCreateReportsUnreadableID(t *testing.T) {
	f := schedulingFixture(t)
	base := f.handle
	posted := false
	f.handle = func(r request) (string, error) {
		if r.method == "POST" {
			posted = true
			b, _ := json.Marshal(r.body)
			if string(b) != `{"data":{"attributes":{"quantity":0,"team_id":"5","team_position_id":"7"},"type":"NeededPosition"}}` {
				t.Fatalf("body=%s", b)
			}
			return planningSingle("NeededPosition", "50", `{}`, ""), nil
		}
		if posted && strings.HasSuffix(r.path, "/needed_positions/50") {
			return planningSingle("NeededPosition", "50", `{"quantity":0,"team_position_name":"Vocals","scheduled_to":"uninterpreted-provider-value"}`, neededRelationships("5", "")), nil
		}
		return base(r)
	}
	zero := 0
	got, e := NewLibrary(f).CreateNeededPosition(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, NeededPositionCreate{Team: "5", Position: "7", Quantity: &zero})
	if e != nil || got.Outcome != "accepted" || len(got.UnverifiedFields) != 1 || got.UnverifiedFields[0] != "team_position_id" || got.RequestedPositionID == nil || *got.RequestedPositionID != "7" || got.NeededPosition.ID != "50" || got.NeededPosition.Quantity == nil || *got.NeededPosition.Quantity != 0 {
		t.Fatalf("got=%+v e=%v", got, e)
	}
}
func TestNeededPositionUpdateDeleteAndTimeExpansionGuard(t *testing.T) {
	for _, mode := range []string{"verified", "time_specific", "expanded_time", "wrong_name", "delete"} {
		t.Run(mode, func(t *testing.T) {
			f := schedulingFixture(t)
			base := f.handle
			gets := 0
			written := false
			f.handle = func(r request) (string, error) {
				if strings.HasSuffix(r.path, "/needed_positions/50") {
					if r.method == "PATCH" || r.method == "DELETE" {
						written = true
						if r.method == "DELETE" {
							return "", nil
						}
					}
					if written && mode == "delete" {
						return "", planningcenter.APIError{StatusCode: 404}
					}
					gets++
					quantity := 3
					if written {
						quantity = 0
					}
					name := "Vocals"
					if written && mode == "wrong_name" {
						name = "Other"
					}
					attrs, _ := json.Marshal(map[string]any{"quantity": quantity, "team_position_name": name, "scheduled_to": "not-a-documented-enum"})
					timeID := ""
					if mode == "time_specific" || mode == "expanded_time" && gets >= 2 {
						timeID = "30"
					}
					rel := neededRelationships("5", timeID)
					if mode == "expanded_time" && gets == 1 {
						rel = `{"team":{"data":{"type":"Team","id":"5"}},"plan":{"data":{"type":"Plan","id":"2"}}}`
					}
					return planningSingle("NeededPosition", "50", string(attrs), rel), nil
				}
				return base(r)
			}
			zero := 0
			l := NewLibrary(f)
			var e error
			if mode == "delete" {
				e = l.DeleteNeededPosition(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, "50")
			} else {
				_, e = l.UpdateNeededPosition(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, "50", NeededPositionPatch{Quantity: &zero})
			}
			if mode == "time_specific" || mode == "expanded_time" {
				assertCode(t, e, "unsupported_capability")
				if f.writes != 0 {
					t.Fatal("write after time-specific scope")
				}
			} else if e != nil || f.writes != 1 {
				t.Fatalf("e=%v writes=%d", e, f.writes)
			}
		})
	}
}
func TestNeededPositionLateFailureRetainsReceipt(t *testing.T) {
	f := schedulingFixture(t)
	base := f.handle
	posted := false
	f.handle = func(r request) (string, error) {
		if r.method == "POST" {
			posted = true
			return planningSingle("NeededPosition", "50", `{}`, ""), nil
		}
		if posted && strings.HasSuffix(r.path, "/needed_positions/50") {
			return "", errors.New("read failed")
		}
		return base(r)
	}
	n := 1
	_, e := NewLibrary(f).CreateNeededPosition(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, NeededPositionCreate{Team: "5", Position: "7", Quantity: &n})
	assertCode(t, e, "verification_failed")
	var se *Error
	if !errors.As(e, &se) || se.PlanningRecovery == nil || se.PlanningRecovery.TargetID != "50" || len(se.PlanningRecovery.Created) != 1 {
		t.Fatalf("e=%v", e)
	}
}
func TestNeededPositionInvalidQuantityBeforeNetwork(t *testing.T) {
	f := schedulingFixture(t)
	l := NewLibrary(f)
	negative := -1
	for _, p := range []NeededPositionCreate{{}, {Team: "5", Position: "7", Quantity: &negative}} {
		_, e := l.CreateNeededPosition(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, p)
		assertCode(t, e, "invalid_input")
	}
	_, e := l.UpdateNeededPosition(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, "50", NeededPositionPatch{})
	assertCode(t, e, "invalid_input")
	if len(f.requests) > 0 {
		t.Fatal(f.requests)
	}
}

func TestNeededPositionPatchRejectsChangedTeamReadback(t *testing.T) {
	f := schedulingFixture(t)
	base := f.handle
	written := false
	f.handle = func(r request) (string, error) {
		if r.method == "PATCH" {
			written = true
			return planningSingle("NeededPosition", "50", `{}`, ""), nil
		}
		if written && strings.HasSuffix(r.path, "/needed_positions/50") {
			return planningSingle("NeededPosition", "50", `{"quantity":0}`, neededRelationships("9", "")), nil
		}
		return base(r)
	}
	n := 0
	_, e := NewLibrary(f).UpdateNeededPosition(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, "50", NeededPositionPatch{Quantity: &n})
	assertCode(t, e, "verification_failed")
	var failure *Error
	if !errors.As(e, &failure) || failure.PlanningRecovery == nil {
		t.Fatalf("lost recovery %v", e)
	}
}

func TestNeededPositionUnexpectedTimeReadbackRetainsReceipt(t *testing.T) {
	f := schedulingFixture(t)
	base := f.handle
	posted := false
	f.handle = func(r request) (string, error) {
		if r.method == "POST" {
			posted = true
			return planningSingle("NeededPosition", "50", `{}`, ""), nil
		}
		if posted && strings.HasSuffix(r.path, "/needed_positions/50") {
			return planningSingle("NeededPosition", "50", `{"quantity":1,"team_position_name":"Vocals"}`, neededRelationships("5", "30")), nil
		}
		return base(r)
	}
	n := 1
	_, e := NewLibrary(f).CreateNeededPosition(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, NeededPositionCreate{Team: "5", Position: "7", Quantity: &n})
	assertCode(t, e, "verification_failed")
	var se *Error
	if !errors.As(e, &se) || se.PlanningRecovery == nil || se.PlanningRecovery.TargetID != "50" {
		t.Fatalf("e=%v", e)
	}
}
