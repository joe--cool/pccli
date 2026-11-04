package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/joe--cool/pccli/v2/internal/planningcenter"
)

func TestPlanTimeCreateExplicitInputAndInstantReadback(t *testing.T) {
	for _, mode := range []string{"verified", "nanoseconds", "wrong_instant", "invalid_read", "read_failure"} {
		t.Run(mode, func(t *testing.T) {
			f := planningFixture(t)
			base := f.handle
			start := "2026-11-08T09:00:00-08:00"
			end := "2026-11-08T10:15:00-08:00"
			typ := "service"
			empty := ""
			if mode == "nanoseconds" {
				start = "2026-11-08T09:00:00.123456789-08:00"
				end = "2026-11-08T10:15:00.123456789-08:00"
			}
			f.handle = func(r request) (string, error) {
				if r.method == "POST" {
					if r.path != "/services/v2/service_types/1/plans/2/plan_times" {
						t.Fatal(r.path)
					}
					b, _ := json.Marshal(r.body)
					if !strings.Contains(string(b), `"name":""`) || !strings.Contains(string(b), `"starts_at":"`+start+`"`) || strings.Contains(string(b), "next_plan") {
						t.Fatalf("body=%s", b)
					}
					return planningSingle("PlanTime", "30", `{}`, ""), nil
				}
				if strings.HasSuffix(r.path, "/plan_times/30") {
					if mode == "read_failure" {
						return "", errors.New("read failed")
					}
					actual := "2026-11-08T17:00:00Z"
					until := "2026-11-08T18:15:00Z"
					if mode == "nanoseconds" {
						actual = "2026-11-08T17:00:00.123456789Z"
						until = "2026-11-08T18:15:00.123456789+00:00"
					}
					if mode == "wrong_instant" {
						actual = "2026-11-08T17:00:01Z"
					}
					if mode == "invalid_read" {
						actual = "not a date"
					}
					attrs, _ := json.Marshal(map[string]any{"starts_at": actual, "ends_at": until, "time_type": "service", "name": ""})
					return planningSingle("PlanTime", "30", string(attrs), ""), nil
				}
				return base(r)
			}
			got, e := NewLibrary(f).CreatePlanTime(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, PlanTimeCreate{Name: &empty, StartsAt: &start, EndsAt: &end, TimeType: &typ})
			if mode == "verified" || mode == "nanoseconds" {
				if e != nil || got.ID != "30" || got.Name == nil || *got.Name != "" {
					t.Fatalf("got=%+v e=%v", got, e)
				}
			} else {
				assertCode(t, e, "verification_failed")
				var failure *Error
				if !errors.As(e, &failure) || failure.PlanningRecovery == nil || failure.PlanningRecovery.TargetID != "30" || failure.PlanningRecovery.PlanID != "2" {
					t.Fatalf("missing recovery=%v", e)
				}
			}
			if f.writes != 1 {
				t.Fatal(f.writes)
			}
		})
	}
}
func TestPlanTimeInvalidInputsBeforeNetwork(t *testing.T) {
	f := planningFixture(t)
	l := NewLibrary(f)
	typ := "service"
	for _, value := range []string{"", "2026-11-08", "2026-11-08T09:00:00", "2026-11-08T09:00:00+24:00", "2026-11-08T09:00:00+00:60", "2026-11-08T09:00:00.1234567891Z", "0000-01-01T00:00:00Z"} {
		_, e := l.CreatePlanTime(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, PlanTimeCreate{StartsAt: &value, TimeType: &typ})
		assertCode(t, e, "invalid_input")
	}
	_, e := l.CreatePlanTime(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, PlanTimeCreate{})
	assertCode(t, e, "invalid_input")
	start, end := "2026-11-08T10:00:00Z", "2026-11-08T09:00:00Z"
	_, e = l.UpdatePlanTime(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, "30", PlanTimePatch{StartsAt: &start, EndsAt: &end})
	assertCode(t, e, "invalid_input")
	if len(f.requests) > 0 {
		t.Fatal(f.requests)
	}
}
func TestPlanTimePatchCombinedStateAndNormalization(t *testing.T) {
	for _, mode := range []string{"valid", "later_start", "earlier_end", "missing_start", "invalid_current", "name_only", "expanded_rel"} {
		t.Run(mode, func(t *testing.T) {
			f := planningFixture(t)
			base := f.handle
			written := false
			f.handle = func(r request) (string, error) {
				if strings.HasSuffix(r.path, "/plan_times/30") {
					if r.method == "PATCH" {
						written = true
					}
					attrs := `{"name":"Morning","starts_at":"2026-11-08T09:00:00Z","ends_at":"2026-11-08T10:00:00Z"}`
					if mode == "missing_start" {
						attrs = `{"name":"Morning","ends_at":"2026-11-08T10:00:00Z"}`
					}
					if mode == "invalid_current" {
						attrs = `{"name":"Morning","starts_at":"bad"}`
					}
					if written {
						if mode == "name_only" {
							attrs = `{"name":"","starts_at":"2026-11-08T09:00:00Z","ends_at":"2026-11-08T10:00:00Z"}`
						} else {
							attrs = `{"name":"Morning","starts_at":"2026-11-08T09:30:00+00:00","ends_at":"2026-11-08T10:00:00Z"}`
						}
					}
					return planningSingle("PlanTime", "30", attrs, `{"plan":{"data":{"type":"Plan","id":"2"}}}`), nil
				}
				return base(r)
			}
			start, end, empty := "2026-11-08T09:30:00Z", "2026-11-08T08:30:00Z", ""
			p := PlanTimePatch{StartsAt: &start}
			if mode == "later_start" {
				start = "2026-11-08T11:00:00Z"
			}
			if mode == "earlier_end" || mode == "missing_start" {
				p = PlanTimePatch{EndsAt: &end}
			}
			if mode == "name_only" {
				p = PlanTimePatch{Name: &empty}
			}
			got, e := NewLibrary(f).UpdatePlanTime(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, "30", p)
			if oneOf(mode, "valid", "name_only", "expanded_rel") {
				if e != nil || got.ID != "30" || f.writes != 1 {
					t.Fatalf("got=%+v e=%v writes=%d", got, e, f.writes)
				}
			} else {
				if e == nil || f.writes != 0 {
					t.Fatalf("e=%v writes=%d", e, f.writes)
				}
			}
		})
	}
}
func TestPlanTimeNamedSelectionSkipsUnavailableNames(t *testing.T) {
	f := planningFixture(t)
	base := f.handle
	f.handle = func(r request) (string, error) {
		if strings.HasSuffix(r.path, "/plan_times") {
			if r.query.Get("offset") == "" {
				return songPage(planningResource("PlanTime", "31", `{"name":null}`, ""), "/services/v2/service_types/1/plans/2/plan_times?per_page=100&offset=100"), nil
			}
			return songPage(planningResource("PlanTime", "30", `{"name":"Morning"}`, ""), ""), nil
		}
		return base(r)
	}
	got, e := NewLibrary(f).ShowPlanTime(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, "morning")
	if e != nil || got.ID != "30" {
		t.Fatalf("got=%+v e=%v", got, e)
	}
}
func TestPlanTimeDeleteVerifiesAbsence(t *testing.T) {
	for _, mode := range []string{"verified", "still_present", "unknown"} {
		t.Run(mode, func(t *testing.T) {
			f := planningFixture(t)
			base := f.handle
			deleted := false
			f.handle = func(r request) (string, error) {
				if strings.HasSuffix(r.path, "/plan_times/30") {
					if r.method == "DELETE" {
						deleted = true
						if mode == "unknown" {
							return "", &planningcenter.TransportError{UnknownOutcome: true, Operation: "DELETE", Cause: errors.New("lost")}
						}
						return "", nil
					}
					if deleted && mode == "verified" {
						return "", planningcenter.APIError{StatusCode: 404}
					}
					return planningSingle("PlanTime", "30", `{"name":"Morning"}`, ""), nil
				}
				return base(r)
			}
			e := NewLibrary(f).DeletePlanTime(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, "30")
			if mode == "verified" {
				if e != nil {
					t.Fatal(e)
				}
			} else if mode == "still_present" {
				assertCode(t, e, "verification_failed")
			} else {
				var unknown *planningcenter.TransportError
				if !errors.As(e, &unknown) || !unknown.UnknownOutcome {
					t.Fatalf("e=%v", e)
				}
			}
			if f.writes != 1 {
				t.Fatal(f.writes)
			}
		})
	}
}

func TestPlanTimeNamedPatchAcceptsEquivalentPrecheckTimes(t *testing.T) {
	f := planningFixture(t)
	base := f.handle
	written := false
	f.handle = func(r request) (string, error) {
		if strings.HasSuffix(r.path, "/plan_times") {
			return songPage(planningResource("PlanTime", "30", `{"name":"Morning","starts_at":"2026-11-08T09:00:00+00:00","ends_at":"2026-11-08T10:00:00+00:00"}`, ""), ""), nil
		}
		if strings.HasSuffix(r.path, "/plan_times/30") {
			if r.method == "PATCH" {
				written = true
			}
			name := "Morning"
			if written {
				name = ""
			}
			attrs, _ := json.Marshal(map[string]any{"name": name, "starts_at": "2026-11-08T09:00:00Z", "ends_at": "2026-11-08T10:00:00Z"})
			return planningSingle("PlanTime", "30", string(attrs), `{"plan":{"data":{"type":"Plan","id":"2"}}}`), nil
		}
		return base(r)
	}
	empty := ""
	got, e := NewLibrary(f).UpdatePlanTime(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, "Morning", PlanTimePatch{Name: &empty})
	if e != nil || got.Name == nil || *got.Name != "" || f.writes != 1 {
		t.Fatalf("got=%+v e=%v writes=%d", got, e, f.writes)
	}
}
func TestPlanTimeCreateWithoutEndDoesNotInventOne(t *testing.T) {
	f := planningFixture(t)
	base := f.handle
	f.handle = func(r request) (string, error) {
		if r.method == "POST" {
			b, _ := json.Marshal(r.body)
			if strings.Contains(string(b), "ends_at") {
				t.Fatalf("invented end: %s", b)
			}
			return planningSingle("PlanTime", "30", `{}`, ""), nil
		}
		if strings.HasSuffix(r.path, "/plan_times/30") {
			return planningSingle("PlanTime", "30", `{"starts_at":"2026-11-08T09:00:00Z","time_type":"other"}`, ""), nil
		}
		return base(r)
	}
	start, typ := "2026-11-08T09:00:00Z", "other"
	got, e := NewLibrary(f).CreatePlanTime(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, PlanTimeCreate{StartsAt: &start, TimeType: &typ})
	if e != nil || got.EndsAt != nil {
		t.Fatalf("got=%+v e=%v", got, e)
	}
}
func TestPlanTimeMalformedCurrentIsResponseError(t *testing.T) {
	f := planningFixture(t)
	base := f.handle
	f.handle = func(r request) (string, error) {
		if strings.HasSuffix(r.path, "/plan_times/30") {
			return planningSingle("PlanTime", "30", `{"name":"Morning","starts_at":"invalid"}`, ""), nil
		}
		return base(r)
	}
	name := "New"
	_, e := NewLibrary(f).UpdatePlanTime(context.Background(), PlanningScope{ServiceType: "1", Plan: "2"}, "30", PlanTimePatch{Name: &name})
	assertCode(t, e, "invalid_response")
	if f.writes != 0 {
		t.Fatal("write after invalid provider timestamp")
	}
}
