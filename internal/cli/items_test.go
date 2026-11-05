package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestItemLifecycleHumanAndJSON(t *testing.T) {
	for _, machine := range []bool{false, true} {
		t.Run(map[bool]string{true: "JSON", false: "human"}[machine], func(t *testing.T) {
			rows := map[string]map[string]any{"21": {"title": "Welcome", "length": 30, "sequence": 2, "service_position": "during"}}
			order := []string{"21"}
			writes := 0
			response := func(id string) map[string]any {
				return map[string]any{"id": id, "type": "Item", "attributes": rows[id], "relationships": map[string]any{"plan": map[string]any{"data": map[string]any{"type": "Plan", "id": "2"}}}}
			}
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
				var body struct {
					Data struct {
						Type       string         `json:"type"`
						Attributes map[string]any `json:"attributes"`
					} `json:"data"`
				}
				if r.Method == "POST" || r.Method == "PATCH" {
					writes++
					b, e := io.ReadAll(r.Body)
					if e != nil || json.Unmarshal(b, &body) != nil {
						t.Fatalf("body=%s e=%v", b, e)
					}
				}
				if strings.HasSuffix(r.URL.Path, "/item_reorder") {
					if body.Data.Type != "PlanItemReorder" {
						t.Fatal(body.Data.Type)
					}
					raw := body.Data.Attributes["sequence"].([]any)
					order = nil
					for i, id := range raw {
						order = append(order, id.(string))
						rows[id.(string)]["sequence"] = i + 1
					}
					return reviewResponse(204, ""), nil
				}
				if strings.HasSuffix(r.URL.Path, "/items") {
					if r.Method == "POST" {
						if body.Data.Type != "Item" {
							t.Fatal(body.Data.Type)
						}
						rows["20"] = body.Data.Attributes
						order = append(order, "20")
						b, _ := json.Marshal(map[string]any{"data": response("20")})
						return reviewResponse(201, string(b)), nil
					}
					data := []any{}
					for _, id := range order {
						data = append(data, response(id))
					}
					b, _ := json.Marshal(map[string]any{"data": data})
					return reviewResponse(200, string(b)), nil
				}
				id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
				if r.Method == "DELETE" {
					writes++
					delete(rows, id)
					return reviewResponse(204, ""), nil
				}
				if rows[id] == nil {
					return reviewResponse(404, `{"errors":[]}`), nil
				}
				if r.Method == "PATCH" {
					for k, v := range body.Data.Attributes {
						rows[id][k] = v
					}
				}
				b, _ := json.Marshal(map[string]any{"data": response(id)})
				return reviewResponse(200, string(b)), nil
			})
			for _, args := range [][]string{
				{"create", "--title", "Gathering", "--type", "header", "--length", "0", "--position", "0"},
				{"update", "20", "--description=", "--chart-sequence="},
				{"reorder", "--sequence", "20,21"},
				{"delete", "20", "--yes"},
			} {
				argv := append([]string{"services", "items"}, args...)
				argv = append(argv, "--service-type", "1", "--plan", "2")
				if machine {
					argv = append(argv, "--json")
				}
				out, stderr, e := runCommand(injectedCommand(library), argv...)
				if e != nil || stderr != "" || out == "" {
					t.Fatalf("args=%v e=%v out=%s stderr=%s", args, e, out, stderr)
				}
				if machine {
					decodeOutput(t, out)
				}
			}
			if writes != 4 || len(rows) != 1 {
				t.Fatalf("writes=%d rows=%v", writes, rows)
			}
		})
	}
}
func TestItemCLIUsageBeforeCredentials(t *testing.T) {
	for _, args := range [][]string{
		{"create", "--length", "-1"}, {"create", "--position", "-1"}, {"create", "--type", "song"},
		{"create", "--type", "item", "--song", "10", "--arrangement", "11"},
		{"create", "--song", "10"}, {"create", "--arrangement", "11"}, {"create", "--type", "header", "--song", "10", "--arrangement", "11"},
		{"update", "20"}, {"update", "20", "--type", "header"}, {"update", "20", "--position", "1"},
		{"delete", "20"}, {"reorder", "--sequence", "1,01"}, {"reorder", "--sequence", "id:1"},
	} {
		argv := append([]string{"services", "items"}, args...)
		argv = append(argv, "--service-type", "1", "--plan", "2", "--json")
		out, stderr, e := runCommand(NewRootCommand(), argv...)
		if ExitCode(e) != 2 || out != "" || !strings.Contains(stderr, "invalid_input") {
			t.Fatalf("args=%v e=%v out=%s stderr=%s", args, e, out, stderr)
		}
	}
}
func TestItemReorderWriterFailureRetainsCommittedResult(t *testing.T) {
	posted := false
	writes := 0
	l := reviewLibrary(func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "/service_types/1") {
			return reviewResponse(200, `{"data":{"type":"ServiceType","id":"1","attributes":{"name":"Sunday"}}}`), nil
		}
		if strings.HasSuffix(r.URL.Path, "/plans/2") {
			return reviewResponse(200, `{"data":{"type":"Plan","id":"2","attributes":{"title":"Sunday"}}}`), nil
		}
		if strings.HasSuffix(r.URL.Path, "/item_reorder") {
			posted = true
			writes++
			return reviewResponse(204, ""), nil
		}
		one := `{"type":"Item","id":"20","attributes":{"title":"First","sequence":1}}`
		two := `{"type":"Item","id":"21","attributes":{"title":"Second","sequence":2}}`
		if posted {
			one = `{"type":"Item","id":"21","attributes":{"title":"Second","sequence":1}}`
			two = `{"type":"Item","id":"20","attributes":{"title":"First","sequence":2}}`
		}
		return reviewResponse(200, `{"data":[`+one+`,`+two+`]}`), nil
	})
	c := injectedCommand(l)
	var stderr bytes.Buffer
	e := executeCommand(c.Context(), c, []string{"services", "items", "reorder", "--service-type", "1", "--plan", "2", "--sequence", "21,20", "--json"}, errorWriter{}, &stderr)
	if e == nil || writes != 1 || !strings.Contains(stderr.String(), `"partial_success"`) || !strings.Contains(stderr.String(), `"result"`) {
		t.Fatalf("e=%v writes=%d stderr=%s", e, writes, stderr.String())
	}
}
