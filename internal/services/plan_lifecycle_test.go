package services

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/joe--cool/pccli/v2/internal/planningcenter"
)

func TestPlanCreatePresenceAndSeriesReadback(t *testing.T) {
	for _, mode := range []string{"empty", "explicit_empty", "settings", "series", "series_zero", "wrong_value", "bad_read_rel", "wrong_series", "bad_series_lookup", "read_failure", "bad_write"} {
		t.Run(mode, func(t *testing.T) {
			f := planningFixture(t)
			base := f.handle
			f.handle = func(r request) (string, error) {
				if strings.HasPrefix(r.path, "/services/v2/series/") {
					if mode == "bad_series_lookup" {
						return planningSingle("Series", "5", `{"title":4}`, ""), nil
					}
					return planningSingle("Series", "5", `{"title":"Grace"}`, ""), nil
				}
				if r.path == "/services/v2/service_types/1/plans" {
					if r.method != "POST" {
						t.Fatal(r.method)
					}
					attrs := r.body.(map[string]any)["data"].(map[string]any)["attributes"].(map[string]any)
					want := map[string]any{}
					if mode == "explicit_empty" {
						want["title"] = ""
					}
					if mode == "wrong_value" {
						want["title"] = "New"
					}
					if mode == "settings" {
						want["title"] = "Plan"
						want["public"] = false
					}
					if strings.Contains(mode, "series") {
						want["series_id"] = "5"
					}
					if !reflect.DeepEqual(attrs, want) {
						t.Fatalf("attrs=%v want=%v", attrs, want)
					}
					if mode == "bad_write" {
						return `{"data":{"type":"Plan","id":"0","attributes":{}}}`, nil
					}
					return planningSingle("Plan", "3", `{}`, ""), nil
				}
				if r.path == "/services/v2/service_types/1/plans/3" {
					if mode == "read_failure" {
						return "", errors.New("read failed")
					}
					attrs := `{}`
					rel := ""
					if mode == "explicit_empty" {
						attrs = `{"title":""}`
					}
					if mode == "settings" {
						attrs = `{"title":"Plan","public":false}`
					}
					if mode == "series" || mode == "series_zero" {
						rel = `{"series":{"data":{"type":"Series","id":"5"}}}`
					}
					if mode == "bad_read_rel" {
						rel = `{"series":{"data":{"type":"Song","id":"5"}}}`
					}
					return planningSingle("Plan", "3", attrs, rel), nil
				}
				return base(r)
			}
			p := PlanCreate{}
			title := ""
			public := false
			series := "5"
			switch mode {
			case "explicit_empty":
				p.Title = &title
			case "settings":
				title = "Plan"
				p.Title = &title
				p.Public = &public
			case "series", "series_zero", "wrong_series", "bad_series_lookup":
				if mode == "series_zero" {
					series = "0005"
				}
				p.SeriesID = &series
			case "wrong_value":
				title = "New"
				p.Title = &title
			}
			got, e := NewLibrary(f).CreatePlan(context.Background(), PlanningScope{ServiceType: "1"}, p)
			if mode == "series_zero" && series != "0005" {
				t.Fatal("caller pointer mutated")
			}
			switch mode {
			case "bad_series_lookup":
				assertCode(t, e, "invalid_response")
				if f.writes != 0 {
					t.Fatal("write after malformed series")
				}
			case "bad_write":
				assertCode(t, e, "outcome_unknown")
			case "read_failure", "wrong_series", "wrong_value", "bad_read_rel":
				assertCode(t, e, "verification_failed")
				var failure *Error
				if !errors.As(e, &failure) || failure.PlanningRecovery == nil || failure.PlanningRecovery.PlanID != "3" || failure.PlanningRecovery.ServiceTypeID != "1" || len(failure.PlanningRecovery.Created) != 1 || failure.PlanningRecovery.Created[0].ID != "3" {
					t.Fatalf("lost recovery: %+v", failure)
				}
			default:
				if e != nil || got.ID != "3" {
					t.Fatalf("got=%+v e=%v", got, e)
				}
			}
		})
	}
}
func TestPlanSettingsPatchPresence(t *testing.T) {
	f := planningFixture(t)
	base := f.handle
	submitted := false
	f.handle = func(r request) (string, error) {
		if r.method == "PATCH" {
			submitted = true
			b, _ := json.Marshal(r.body)
			if string(b) != `{"data":{"attributes":{"public":false,"reminders_disabled":false},"type":"Plan"}}` {
				t.Fatalf("body=%s", b)
			}
			return planningSingle("Plan", "2", `{}`, ""), nil
		}
		if submitted {
			return planningSingle("Plan", "2", `{"public":false,"reminders_disabled":false}`, ""), nil
		}
		return base(r)
	}
	no := false
	p, e := NewLibrary(f).UpdatePlan(context.Background(), PlanningScope{ServiceType: "1"}, "2", PlanPatch{Public: &no, RemindersDisabled: &no})
	if e != nil || p.Public == nil || *p.Public || p.RemindersDisabled == nil || *p.RemindersDisabled || f.writes != 1 {
		t.Fatalf("p=%+v e=%v", p, e)
	}
}
func TestPlanDeleteReadbackAndUnknownOutcome(t *testing.T) {
	for _, mode := range []string{"verified", "still_present", "forbidden", "changed", "unknown"} {
		t.Run(mode, func(t *testing.T) {
			f := planningFixture(t)
			base := f.handle
			gets := 0
			deleted := false
			f.handle = func(r request) (string, error) {
				if r.method == "DELETE" {
					deleted = true
					if mode == "unknown" {
						return "", &planningcenter.TransportError{UnknownOutcome: true, Operation: "DELETE", Cause: errors.New("lost")}
					}
					return "", nil
				}
				if deleted {
					if mode == "verified" {
						return "", planningcenter.APIError{StatusCode: 404}
					}
					if mode == "forbidden" {
						return "", planningcenter.APIError{StatusCode: 403}
					}
				}
				if strings.HasSuffix(r.path, "/plans/2") {
					gets++
					if mode == "changed" && gets == 2 {
						return planningSingle("Plan", "2", `{"title":"Changed"}`, ""), nil
					}
				}
				return base(r)
			}
			e := NewLibrary(f).DeletePlan(context.Background(), PlanningScope{ServiceType: "1"}, "2")
			switch mode {
			case "verified":
				if e != nil {
					t.Fatal(e)
				}
			case "changed":
				assertCode(t, e, "concurrent_change")
				if f.writes != 0 {
					t.Fatal("write after changed identity")
				}
			case "unknown":
				var unknown *planningcenter.TransportError
				if !errors.As(e, &unknown) || !unknown.UnknownOutcome {
					t.Fatalf("e=%v", e)
				}
			default:
				assertCode(t, e, "verification_failed")
			}
			if mode != "changed" && f.writes != 1 {
				t.Fatalf("writes=%d", f.writes)
			}
		})
	}
}
func TestPlanSeriesInputConflictsBeforeNetwork(t *testing.T) {
	f := planningFixture(t)
	l := NewLibrary(f)
	id := "5"
	title := "Grace"
	bad := "id:5"
	_, e := l.CreatePlan(context.Background(), PlanningScope{ServiceType: "1"}, PlanCreate{SeriesID: &id, SeriesTitle: &title})
	assertCode(t, e, "invalid_input")
	_, e = l.UpdatePlan(context.Background(), PlanningScope{ServiceType: "1"}, "2", PlanPatch{SeriesID: &bad})
	assertCode(t, e, "invalid_input")
	if len(f.requests) > 0 {
		t.Fatal(f.requests)
	}
}
