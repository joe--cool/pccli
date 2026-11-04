package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestPlanTimeLifecycleHumanAndJSON(t *testing.T) {
	for _, machine := range []bool{false, true} {
		t.Run(map[bool]string{true: "JSON", false: "human"}[machine], func(t *testing.T) {
			exists := false
			attrs := map[string]any{}
			writes := 0
			library := reviewLibrary(func(r *http.Request) (*http.Response, error) {
				if r.Header.Get("X-PCO-API-Version") != "2018-11-01" {
					t.Fatal("missing version")
				}
				if r.URL.Path == "/services/v2/service_types/1" {
					return reviewResponse(200, `{"data":{"type":"ServiceType","id":"1","attributes":{"name":"Sunday"}}}`), nil
				}
				if r.URL.Path == "/services/v2/service_types/1/plans/2" {
					return reviewResponse(200, `{"data":{"type":"Plan","id":"2","attributes":{"title":"Sunday"}}}`), nil
				}
				if r.Method == "POST" || r.Method == "PATCH" {
					writes++
					exists = true
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
					if json.Unmarshal(b, &body) != nil || body.Data.Type != "PlanTime" {
						t.Fatalf("body=%s", b)
					}
					for k, v := range body.Data.Attributes {
						attrs[k] = v
					}
				}
				if r.Method == "DELETE" {
					writes++
					exists = false
					return reviewResponse(204, ""), nil
				}
				if !exists {
					return reviewResponse(404, `{"errors":[]}`), nil
				}
				b, _ := json.Marshal(map[string]any{"data": map[string]any{"type": "PlanTime", "id": "30", "attributes": attrs}})
				return reviewResponse(200, string(b)), nil
			})
			for _, args := range [][]string{
				{"create", "--name", "Morning", "--starts-at", "2026-11-08T09:00:00-08:00", "--ends-at", "2026-11-08T10:00:00-08:00", "--time-type", "service"},
				{"show", "30"}, {"update", "30", "--name=", "--starts-at", "2026-11-08T09:30:00-08:00"}, {"delete", "30", "--yes"},
			} {
				argv := append([]string{"services", "plan-times"}, args...)
				argv = append(argv, "--service-type", "1", "--plan", "2")
				if machine {
					argv = append(argv, "--json")
				}
				out, stderr, e := runCommand(injectedCommand(library), argv...)
				if e != nil || out == "" || stderr != "" {
					t.Fatalf("args=%v e=%v stdout=%s stderr=%s", args, e, out, stderr)
				}
				if machine {
					decodeOutput(t, out)
				}
			}
			if writes != 3 || exists {
				t.Fatalf("writes=%d exists=%v", writes, exists)
			}
		})
	}
}
func TestPlanTimeCLIUsageBeforeCredentials(t *testing.T) {
	for _, args := range [][]string{
		{"create"}, {"create", "--starts-at", "2026-11-08T09:00:00Z"}, {"create", "--starts-at", "2026-11-08T09:00:00", "--time-type", "service"},
		{"create", "--starts-at", "2026-11-08T09:00:00Z", "--time-type", "future"}, {"update", "30"}, {"update", "30", "--ends-at="}, {"delete", "30"},
	} {
		argv := append([]string{"services", "plan-times"}, args...)
		argv = append(argv, "--service-type", "1", "--plan", "2", "--json")
		out, stderr, e := runCommand(NewRootCommand(), argv...)
		if ExitCode(e) != 2 || out != "" || !strings.Contains(stderr, "invalid_input") {
			t.Fatalf("args=%v e=%v out=%s stderr=%s", args, e, out, stderr)
		}
	}
}
