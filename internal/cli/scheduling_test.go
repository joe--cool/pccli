package cli

import (
	"bytes"
	"encoding/json"
	"github.com/joe--cool/pccli/v2/internal/services"
	"io"
	"net/http"
	"strings"
	"testing"
)

// A stateful request-aware account fixture; it has no notification-send route.
func schedulingCommandLibrary(t *testing.T) (*services.Library, *int) {
	t.Helper()
	writes := 0
	assignments := map[string]map[string]any{}
	needs := map[string]map[string]any{}
	l := reviewLibrary(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("X-PCO-API-Version") != "2018-11-01" {
			t.Fatal("missing version")
		}
		switch r.URL.Path {
		case "/services/v2/service_types/1":
			return reviewResponse(200, `{"data":{"type":"ServiceType","id":"1","attributes":{"name":"Sunday"}}}`), nil
		case "/services/v2/service_types/1/plans/2":
			return reviewResponse(200, `{"data":{"type":"Plan","id":"2","attributes":{"title":"Sunday"}}}`), nil
		case "/services/v2/service_types/1/teams/5":
			return reviewResponse(200, `{"data":{"type":"Team","id":"5","attributes":{"name":"Band","schedule_to":"plan"}}}`), nil
		case "/services/v2/teams/5/people":
			return reviewResponse(200, `{"data":[{"type":"Person","id":"6","attributes":{"full_name":"Alex Sample","archived":false}}]}`), nil
		case "/services/v2/teams/5/people/6":
			return reviewResponse(200, `{"data":{"type":"Person","id":"6","attributes":{"full_name":"Alex Sample","archived":false}}}`), nil
		case "/services/v2/teams/5/team_positions":
			return reviewResponse(200, `{"data":[{"type":"TeamPosition","id":"7","attributes":{"name":"Vocals"},"relationships":{"team":{"data":{"type":"Team","id":"5"}}}}]}`), nil
		case "/services/v2/teams/5/team_positions/7":
			return reviewResponse(200, `{"data":{"type":"TeamPosition","id":"7","attributes":{"name":"Vocals"},"relationships":{"team":{"data":{"type":"Team","id":"5"}}}}}`), nil
		}
		if !strings.Contains(r.URL.Path, "/team_members") && !strings.Contains(r.URL.Path, "/needed_positions") {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		scheduled := strings.Contains(r.URL.Path, "/team_members")
		rows := needs
		kind, id, collection := "NeededPosition", "50", "/needed_positions"
		if scheduled {
			rows = assignments
			kind, id, collection = "PlanPerson", "40", "/team_members"
		}
		attrs := map[string]any{}
		if r.Method == "POST" || r.Method == "PATCH" {
			writes++
			b, e := io.ReadAll(r.Body)
			if e != nil {
				t.Fatal(e)
			}
			var body struct {
				Data struct {
					Type       string         `json:"type"`
					Attributes map[string]any `json:"attributes"`
				} `json:"data"`
			}
			if json.Unmarshal(b, &body) != nil || body.Data.Type != kind {
				t.Fatalf("body=%s", b)
			}
			attrs = body.Data.Attributes
			if scheduled {
				if r.Method == "POST" {
					if attrs["prepare_notification"] != false || attrs["status"] != "U" {
						t.Fatalf("unsafe defaults=%v", attrs)
					}
				} else if _, present := attrs["prepare_notification"]; present {
					t.Fatal("patch changed preparation state")
				}
			}
			if rows[id] == nil {
				rows[id] = map[string]any{}
			}
			for k, v := range attrs {
				rows[id][k] = v
			}
			if scheduled {
				rows[id]["name"] = "Alex Sample"
				rows[id]["can_accept_partial"] = false
			} else {
				rows[id]["team_position_name"] = "Vocals"
				rows[id]["scheduled_to"] = "provider-raw"
			}
		}
		if r.Method == "DELETE" {
			writes++
			delete(rows, id)
			return reviewResponse(204, ""), nil
		}
		resource := func() map[string]any {
			rel := map[string]any{"plan": map[string]any{"data": map[string]any{"type": "Plan", "id": "2"}}, "team": map[string]any{"data": map[string]any{"type": "Team", "id": "5"}}}
			if scheduled {
				rel["person"] = map[string]any{"data": map[string]any{"type": "Person", "id": "6"}}
			} else {
				rel["time"] = map[string]any{"data": nil}
				rel["time_preference_option"] = map[string]any{"data": nil}
			}
			return map[string]any{"id": id, "type": kind, "attributes": rows[id], "relationships": rel}
		}
		var data any
		if strings.HasSuffix(r.URL.Path, collection) && r.Method == "GET" {
			items := []any{}
			if rows[id] != nil {
				items = append(items, resource())
			}
			data = items
		} else {
			if rows[id] == nil {
				return reviewResponse(404, `{"errors":[]}`), nil
			}
			data = resource()
		}
		b, _ := json.Marshal(map[string]any{"data": data})
		return reviewResponse(200, string(b)), nil
	})
	return l, &writes
}
func TestSchedulingAndNeededPositionCLIJourneys(t *testing.T) {
	for _, machine := range []bool{false, true} {
		t.Run(map[bool]string{true: "JSON", false: "human"}[machine], func(t *testing.T) {
			l, writes := schedulingCommandLibrary(t)
			for _, args := range [][]string{
				{"team-people", "list", "--service-type", "1", "--team", "5"},
				{"team-people", "show", "6", "--service-type", "1", "--team", "5"},
				{"team-positions", "list", "--service-type", "1", "--team", "5"},
				{"team-positions", "show", "7", "--service-type", "1", "--team", "5"},
				{"scheduled-people", "create", "--service-type", "1", "--plan", "2", "--team", "5", "--person", "6", "--position", "7", "--notes="},
				{"scheduled-people", "show", "40", "--service-type", "1", "--plan", "2"},
				{"scheduled-people", "list", "--service-type", "1", "--plan", "2"},
				{"scheduled-people", "update", "40", "--service-type", "1", "--plan", "2", "--status", "confirmed", "--notes="},
				{"scheduled-people", "delete", "40", "--service-type", "1", "--plan", "2", "--yes"},
				{"needed-positions", "create", "--service-type", "1", "--plan", "2", "--team", "5", "--position", "7", "--quantity", "0"},
				{"needed-positions", "show", "50", "--service-type", "1", "--plan", "2"},
				{"needed-positions", "list", "--service-type", "1", "--plan", "2"},
				{"needed-positions", "update", "50", "--service-type", "1", "--plan", "2", "--quantity", "2"},
				{"needed-positions", "delete", "50", "--service-type", "1", "--plan", "2", "--yes"},
			} {
				argv := append([]string{"services"}, args...)
				if machine {
					argv = append(argv, "--json")
					if args[1] == "list" {
						argv = append(argv, "--all")
					}
				}
				out, stderr, e := runCommand(injectedCommand(l), argv...)
				if e != nil || out == "" || stderr != "" {
					t.Fatalf("args=%v e=%v out=%s stderr=%s", args, e, out, stderr)
				}
				if machine {
					decodeOutput(t, out)
				}
				if args[0] == "needed-positions" && args[1] == "create" {
					if !strings.Contains(out, "accepted") || !strings.Contains(out, "team_position_id") || !strings.Contains(out, "50") {
						t.Fatalf("missing accepted identity: %s", out)
					}
					if !machine && strings.Contains(out, `{"id"`) {
						t.Fatalf("nested JSON blob in human output: %s", out)
					}
					if machine && !strings.Contains(out, `"requested_position_id": "7"`) {
						t.Fatal(out)
					}
				}
			}
			if *writes != 6 {
				t.Fatalf("writes=%d", *writes)
			}
		})
	}
}
func TestSchedulingCLIUsageBeforeConfiguration(t *testing.T) {
	cases := [][]string{
		{"team-people", "list"}, {"team-positions", "list", "--service-type", "1", "--team", "id:bad"},
		{"scheduled-people", "create", "--service-type", "1", "--plan", "2", "--team", "5"},
		{"scheduled-people", "create", "--service-type", "1", "--plan", "2", "--team", "5", "--person", "6", "--position", "7", "--status", "maybe"},
		{"scheduled-people", "update", "40", "--service-type", "1", "--plan", "2"},
		{"scheduled-people", "update", "40", "--service-type", "1", "--plan", "2", "--team=", "--notes="},
		{"scheduled-people", "delete", "40", "--service-type", "1", "--plan", "2"},
		{"needed-positions", "create", "--service-type", "1", "--plan", "2", "--team", "5", "--position", "7"},
		{"needed-positions", "create", "--service-type", "1", "--plan", "2", "--team", "5", "--position", "7", "--quantity", "-1"},
		{"needed-positions", "delete", "50", "--service-type", "1", "--plan", "2"},
	}
	for _, args := range cases {
		out, stderr, e := runCommand(NewRootCommand(), append(append([]string{"services"}, args...), "--json")...)
		if ExitCode(e) != 2 || out != "" || !strings.Contains(stderr, "invalid_input") {
			t.Fatalf("args=%v e=%v out=%s stderr=%s", args, e, out, stderr)
		}
	}
}
func TestNeededPositionAcceptedWriterFailureRetainsResult(t *testing.T) {
	l, writes := schedulingCommandLibrary(t)
	c := injectedCommand(l)
	var stderr bytes.Buffer
	e := executeCommand(c.Context(), c, []string{"services", "needed-positions", "create", "--service-type", "1", "--plan", "2", "--team", "5", "--position", "7", "--quantity", "1", "--json"}, errorWriter{}, &stderr)
	if e == nil || *writes != 1 || !strings.Contains(stderr.String(), `"partial_success"`) || !strings.Contains(stderr.String(), `"outcome": "accepted"`) || !strings.Contains(stderr.String(), `"requested_position_id": "7"`) {
		t.Fatalf("e=%v writes=%d stderr=%s", e, *writes, stderr.String())
	}
}

func TestSchedulingMockReadJourneys(t *testing.T) {
	for _, args := range [][]string{
		{"team-people", "list", "--service-type", "3001", "--team", "6001"},
		{"team-people", "show", "9001", "--service-type", "3001", "--team", "6001"},
		{"team-positions", "list", "--service-type", "3001", "--team", "6001"},
		{"team-positions", "show", "10001", "--service-type", "3001", "--team", "6001"},
		{"needed-positions", "list", "--service-type", "3001", "--plan", "4001"},
		{"needed-positions", "show", "11001", "--service-type", "3001", "--plan", "4001"},
	} {
		for _, machine := range []bool{false, true} {
			argv := append([]string{"services"}, args...)
			if machine {
				argv = append(argv, "--json")
				if args[1] == "list" {
					argv = append(argv, "--all")
				}
			}
			out, stderr, e := runCommand(fixtureCommand(t), argv...)
			if e != nil || out == "" || stderr != "" {
				t.Fatalf("args=%v e=%v out=%s stderr=%s", args, e, out, stderr)
			}
			if machine {
				decodeOutput(t, out)
			}
		}
	}
}
