package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/joe--cool/pccli/v2/internal/services"
)

func copyCommandLibrary(t *testing.T, mode string, cancellation ...context.CancelFunc) (*services.Library, *int) {
	t.Helper()
	writes := 0
	var planAttrs map[string]any
	var timeAttrs map[string]any
	var needAttrs map[string]any
	source := `{"type":"Plan","id":"2","attributes":{"title":"Source","public":true,"reminders_disabled":true}}`
	l := reviewLibrary(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("X-PCO-API-Version") != "2018-11-01" {
			t.Fatal("missing version")
		}
		switch r.URL.Path {
		case "/services/v2/service_types/1":
			return reviewResponse(200, `{"data":{"type":"ServiceType","id":"1","attributes":{"name":"Sunday"}}}`), nil
		case "/services/v2/service_types/1/plans/2":
			return reviewResponse(200, `{"data":`+source+`}`), nil
		case "/services/v2/service_types/1/teams/5":
			return reviewResponse(200, `{"data":{"type":"Team","id":"5","attributes":{"name":"Band","schedule_to":"plan","archived_at":null}}}`), nil
		case "/services/v2/teams/5/team_positions":
			return reviewResponse(200, `{"data":[{"type":"TeamPosition","id":"7","attributes":{"name":"Vocals"},"relationships":{"team":{"data":{"type":"Team","id":"5"}}}}]}`), nil
		case "/services/v2/service_types/1/plans/2/needed_positions":
			return reviewResponse(200, `{"data":[{"type":"NeededPosition","id":"50","attributes":{"quantity":2,"team_position_name":"Vocals"},"relationships":{"team":{"data":{"type":"Team","id":"5"}}}}]}`), nil
		case "/services/v2/service_types/1/plans/2/plan_times":
			return reviewResponse(200, `{"data":[{"type":"PlanTime","id":"30","attributes":{"name":"Morning","starts_at":"2026-11-01T09:00:00Z","ends_at":"2026-11-01T10:00:00Z","time_type":"service"}}]}`), nil
		}
		var body struct {
			Data struct {
				Type       string         `json:"type"`
				Attributes map[string]any `json:"attributes"`
			} `json:"data"`
		}
		if r.Method == "POST" {
			writes++
			b, e := io.ReadAll(r.Body)
			if e != nil || json.Unmarshal(b, &body) != nil {
				t.Fatalf("body=%s e=%v", b, e)
			}
		}
		switch r.URL.Path {
		case "/services/v2/service_types/1/plans":
			if r.Method == "GET" {
				return reviewResponse(200, `{"data":[`+source+`]}`), nil
			}
			if body.Data.Type != "Plan" || body.Data.Attributes["public"] != false {
				t.Fatalf("plan body=%+v", body)
			}
			planAttrs = body.Data.Attributes
			return reviewResponse(201, `{"data":{"type":"Plan","id":"3","attributes":{}}}`), nil
		case "/services/v2/service_types/1/plans/3":
			if planAttrs == nil {
				t.Fatal("GET before create")
			}
			if timeAttrs != nil {
				planAttrs["dates"] = "Nov 8, 2026"
			}
			if needAttrs != nil {
				planAttrs["needed_positions_count"] = 1
			}
			planAttrs["public_by_schedule"] = true
			b, _ := json.Marshal(map[string]any{"data": map[string]any{"type": "Plan", "id": "3", "attributes": planAttrs}})
			return reviewResponse(200, string(b)), nil
		case "/services/v2/service_types/1/plans/3/needed_positions":
			if body.Data.Type != "NeededPosition" || body.Data.Attributes["team_position_id"] != "7" {
				t.Fatalf("needed body=%+v", body)
			}
			needAttrs = body.Data.Attributes
			return reviewResponse(201, `{"data":{"type":"NeededPosition","id":"150","attributes":{}}}`), nil
		case "/services/v2/service_types/1/plans/3/needed_positions/150":
			return reviewResponse(200, `{"data":{"type":"NeededPosition","id":"150","attributes":{"quantity":2,"team_position_name":"Vocals"},"relationships":{"team":{"data":{"type":"Team","id":"5"}}}}}`), nil
		case "/services/v2/service_types/1/plans/3/plan_times":
			if body.Data.Type != "PlanTime" {
				t.Fatal(body.Data.Type)
			}
			switch mode {
			case "rejected":
				return reviewResponse(403, `{"errors":[]}`), nil
			case "unknown":
				return reviewResponse(500, `{"errors":[]}`), nil
			case "cancel":
				if len(cancellation) > 0 {
					cancellation[0]()
				}
				return nil, context.Canceled
			}
			timeAttrs = body.Data.Attributes
			if timeAttrs["starts_at"] != "2026-11-08T09:00:00Z" {
				t.Fatalf("offset body=%v", timeAttrs)
			}
			return reviewResponse(201, `{"data":{"type":"PlanTime","id":"130","attributes":{}}}`), nil
		case "/services/v2/service_types/1/plans/3/plan_times/130":
			b, _ := json.Marshal(map[string]any{"data": map[string]any{"type": "PlanTime", "id": "130", "attributes": timeAttrs}})
			return reviewResponse(200, string(b)), nil
		}
		t.Fatalf("unexpected request %s %s", r.Method, r.URL)
		return nil, nil
	})
	return l, &writes
}
func TestPlanCopyCLIHumanAndJSON(t *testing.T) {
	for _, machine := range []bool{false, true} {
		t.Run(map[bool]string{true: "JSON", false: "human"}[machine], func(t *testing.T) {
			l, writes := copyCommandLibrary(t, "")
			args := []string{"services", "plans", "copy", "2", "--service-type", "1", "--copy-times", "--time-offset", "168h", "--title", "New service"}
			if machine {
				args = append(args, "--json")
			}
			out, stderr, e := runCommand(injectedCommand(l), args...)
			if e != nil || stderr != "" || *writes != 2 {
				t.Fatalf("e=%v out=%s stderr=%s writes=%d", e, out, stderr, *writes)
			}
			if machine {
				envelope := decodeOutput(t, out)
				var result services.PlanCopyResult
				if json.Unmarshal(envelope["data"], &result) != nil || result.Plan.ID != "3" || result.Plan.Dates == nil || *result.Plan.Dates != "Nov 8, 2026" || result.SourcePlanID != "2" || len(result.Created) != 2 || result.Created[1].ID != "130" || result.Plan.Public == nil || *result.Plan.Public || result.Plan.PublicBySchedule == nil || !*result.Plan.PublicBySchedule {
					t.Fatalf("result=%s", out)
				}
			} else {
				if !strings.Contains(out, "CREATED RECORDS") || !strings.Contains(out, "130") || !strings.Contains(out, "New service") || !strings.Contains(out, "Source plan ID") || strings.Contains(out, `{"plan"`) || strings.Contains(out, "REQUESTED CHANGES") {
					t.Fatalf("human output=%s", out)
				}
			}
		})
	}
}
func TestPlanCopyCLIUsageBeforeCredentials(t *testing.T) {
	for _, args := range [][]string{
		{"copy", "2"}, {"copy", "2", "--copy-times"}, {"copy", "2", "--copy-items", "--time-offset", "0"}, {"copy", "2", "--copy-times", "--preserve-dates", "--time-offset", "0"}, {"copy", "2", "--copy-times", "--time-offset", "tomorrow"},
	} {
		argv := append([]string{"services", "plans"}, args...)
		argv = append(argv, "--service-type", "1", "--json")
		out, stderr, e := runCommand(NewRootCommand(), argv...)
		if ExitCode(e) != 2 || out != "" || !strings.Contains(stderr, "invalid_input") {
			t.Fatalf("args=%v e=%v out=%s stderr=%s", args, e, out, stderr)
		}
	}
	out, stderr, e := runCommand(NewRootCommand(), "services", "plans", "copy", "--help")
	if e != nil || stderr != "" || !strings.Contains(out, "copy-times") {
		t.Fatalf("help e=%v %s %s", e, out, stderr)
	}
}
func TestPlanCopyCLIErrorRoutingPreservesCommitment(t *testing.T) {
	for _, mode := range []string{"rejected", "unknown", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			l, writes := copyCommandLibrary(t, mode, cancel)
			command := injectedCommand(l)
			command.SetContext(withApp(ctx, appFrom(command)))
			out, stderr, e := runCommand(command, "services", "plans", "copy", "2", "--service-type", "1", "--copy-times", "--time-offset", "168h", "--json")
			if e == nil || out != "" || *writes != 2 || !strings.Contains(stderr, `"source_plan_id": "2"`) || !strings.Contains(stderr, `"plan_id": "3"`) || !strings.Contains(stderr, `"created"`) {
				t.Fatalf("e=%v out=%s stderr=%s writes=%d", e, out, stderr, *writes)
			}
			if mode == "rejected" {
				if ExitCode(e) != 1 || !strings.Contains(stderr, `"partial_success"`) || !strings.Contains(stderr, `"last_outcome": "rejected"`) || !strings.Contains(stderr, `"http_status": 403`) {
					t.Fatal(stderr)
				}
			} else if mode == "unknown" {
				if !strings.Contains(stderr, `"outcome_unknown"`) || !strings.Contains(stderr, `"last_outcome": "unknown"`) {
					t.Fatal(stderr)
				}
			} else if ExitCode(e) != 130 || !strings.Contains(stderr, `"cancelled"`) || !strings.Contains(stderr, `"last_outcome": "unknown"`) {
				t.Fatalf("exit=%d %s", ExitCode(e), stderr)
			}
		})
	}
}
func TestPlanCopyWriterFailureRetainsAllCreatedIDs(t *testing.T) {
	l, writes := copyCommandLibrary(t, "")
	c := injectedCommand(l)
	var stderr bytes.Buffer
	e := executeCommand(c.Context(), c, []string{"services", "plans", "copy", "2", "--service-type", "1", "--copy-times", "--time-offset", "168h", "--json"}, errorWriter{}, &stderr)
	if e == nil || *writes != 2 || !strings.Contains(stderr.String(), `"partial_success"`) || !strings.Contains(stderr.String(), `"id": "130"`) || !strings.Contains(stderr.String(), `"id": "3"`) || !strings.Contains(stderr.String(), `"result"`) {
		t.Fatalf("e=%v writes=%d stderr=%s", e, *writes, stderr.String())
	}
}

func TestPlanCopyAcceptedNeededPresentation(t *testing.T) {
	for _, machine := range []bool{false, true} {
		l, _ := copyCommandLibrary(t, "needed")
		args := []string{"services", "plans", "copy", "2", "--service-type", "1", "--copy-needed-positions"}
		if machine {
			args = append(args, "--json")
		}
		out, stderr, e := runCommand(injectedCommand(l), args...)
		if e != nil || stderr != "" {
			t.Fatalf("e=%v stderr=%s", e, stderr)
		}
		if machine {
			var result services.PlanCopyResult
			env := decodeOutput(t, out)
			if json.Unmarshal(env["data"], &result) != nil || result.Outcome != "accepted" || len(result.UnverifiedFields) != 1 || result.UnverifiedFields[0].ID != "150" || result.UnverifiedFields[0].RequestedID != "7" {
				t.Fatal(out)
			}
		} else if !strings.Contains(out, "UNVERIFIED FIELDS") || !strings.Contains(out, "accepted") || !strings.Contains(out, "150") || !strings.Contains(out, "team_position_id") || !strings.Contains(out, "Requested ID") || strings.Contains(out, `{"resource"`) {
			t.Fatal(out)
		}
	}
}
