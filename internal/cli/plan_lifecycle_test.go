package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestPlanLifecycleHumanAndJSON(t *testing.T) {
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
					return reviewResponse(200, `{"data":{"id":"1","type":"ServiceType","attributes":{"name":"Sunday"}}}`), nil
				}
				if r.URL.Path != "/services/v2/service_types/1/plans" && r.URL.Path != "/services/v2/service_types/1/plans/2" {
					t.Fatal(r.URL)
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
					if json.Unmarshal(b, &body) != nil || body.Data.Type != "Plan" {
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
				b, _ := json.Marshal(map[string]any{"data": map[string]any{"type": "Plan", "id": "2", "attributes": attrs}})
				return reviewResponse(200, string(b)), nil
			})
			for _, args := range [][]string{
				{"create", "--service-type", "1", "--title=", "--public=false"},
				{"update", "2", "--service-type", "1", "--public=false", "--reminders-disabled=false"},
				{"delete", "2", "--service-type", "1", "--yes"},
			} {
				argv := append([]string{"services", "plans"}, args...)
				if machine {
					argv = append(argv, "--json")
				}
				out, stderr, e := runCommand(injectedCommand(library), argv...)
				if e != nil || stderr != "" || out == "" {
					t.Fatalf("args=%v out=%s err=%v stderr=%s", args, out, e, stderr)
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
func TestPlanLifecycleUsageBeforeCredentials(t *testing.T) {
	for _, args := range [][]string{
		{"create"}, {"create", "--service-type", "1", "--series-id", "0"}, {"create", "--service-type", "1", "--series-id", "5", "--series-title", "Grace"},
		{"update", "2", "--service-type", "1", "--public=bogus"}, {"delete", "2", "--service-type", "1"},
	} {
		out, stderr, e := runCommand(NewRootCommand(), append(append([]string{"services", "plans"}, args...), "--json")...)
		if ExitCode(e) != 2 || out != "" || !strings.Contains(stderr, "invalid_input") {
			t.Fatalf("args=%v out=%s err=%v stderr=%s", args, out, e, stderr)
		}
	}
}

func TestPlanCreateRecoveryRetainsIDInHumanAndJSONErrors(t *testing.T) {
	for _, machine := range []bool{false, true} {
		t.Run(map[bool]string{true: "JSON", false: "human"}[machine], func(t *testing.T) {
			writes := 0
			library := reviewLibrary(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/services/v2/service_types/1" {
					return reviewResponse(200, `{"data":{"type":"ServiceType","id":"1","attributes":{"name":"Sunday"}}}`), nil
				}
				if r.Method == "POST" {
					writes++
					return reviewResponse(201, `{"data":{"type":"Plan","id":"20","attributes":{}}}`), nil
				}
				return reviewResponse(200, `{"data":{"type":"Plan","id":"20","attributes":{"title":"Unexpected"}}}`), nil
			})
			args := []string{"services", "plans", "create", "--service-type", "1", "--title", "Expected"}
			if machine {
				args = append(args, "--json")
			}
			out, stderr, e := runCommand(injectedCommand(library), args...)
			if e == nil || out != "" || writes != 1 || !strings.Contains(stderr, `"plan_id":"20"`) && !strings.Contains(stderr, `"plan_id": "20"`) {
				t.Fatalf("e=%v stdout=%s stderr=%s writes=%d", e, out, stderr, writes)
			}
			if machine {
				if !strings.Contains(stderr, `"verification_failed"`) || !strings.Contains(stderr, `"recovery"`) {
					t.Fatal(stderr)
				}
			} else if !strings.Contains(stderr, "Recovery:") {
				t.Fatal(stderr)
			}
		})
	}
}
