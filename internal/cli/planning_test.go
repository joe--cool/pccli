package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestPlanningMockJourneys(t *testing.T) {
	cases := [][]string{
		{"service-types", "list"}, {"service-types", "show", "Sunday"},
		{"plans", "list", "--service-type", "Sunday Worship"}, {"plans", "show", "4001", "--service-type", "3001"},
		{"items", "list", "--service-type", "3001", "--plan", "4001"}, {"items", "show", "5002", "--service-type", "3001", "--plan", "4001"},
		{"teams", "list", "--service-type", "3001"}, {"teams", "show", "6001", "--service-type", "3001"},
		{"plan-times", "list", "--service-type", "3001", "--plan", "4001"},
		{"scheduled-people", "list", "--service-type", "3001", "--plan", "4001"},
	}
	for _, args := range cases {
		for _, machine := range []bool{false, true} {
			t.Run(strings.Join(args, " ")+map[bool]string{true: " JSON"}[machine], func(t *testing.T) {
				argv := append([]string{"services"}, args...)
				if machine {
					argv = append(argv, "--json")
					if args[1] == "list" {
						argv = append(argv, "--all")
					}
				}
				out, stderr, e := runCommand(fixtureCommand(t), argv...)
				if e != nil || stderr != "" || out == "" {
					t.Fatalf("err=%v stderr=%s out=%s", e, stderr, out)
				}
				if machine {
					decodeOutput(t, out)
				} else if !strings.Contains(out, "ID") {
					t.Fatalf("missing visible ID: %s", out)
				}
			})
		}
	}
}
func TestPlanningCLIValidationBeforeConfiguration(t *testing.T) {
	cases := [][]string{
		{"plans", "list"}, {"items", "list", "--service-type", "1"}, {"items", "list", "--service-type", "id:bad", "--plan", "2"},
		{"plans", "update", "2", "--service-type", "1"}, {"plans", "list", "--service-type", "1", "--filter", "after"},
		{"teams", "list", "--service-type", "1", "--limit", "0"}, {"plans", "list", "--service-type", "1", "--all", "--limit", "2"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			out, stderr, e := runCommand(NewRootCommand(), append(append([]string{"services"}, args...), "--json")...)
			if ExitCode(e) != 2 || out != "" || !strings.Contains(stderr, `"invalid_input"`) {
				t.Fatalf("err=%v stdout=%s stderr=%s", e, out, stderr)
			}
		})
	}
	for _, resource := range []string{"service-types", "plans", "items", "teams", "plan-times", "scheduled-people"} {
		out, stderr, e := runCommand(NewRootCommand(), "services", resource, "--help")
		if e != nil || stderr != "" || out == "" {
			t.Fatalf("help %s: %v %s", resource, e, stderr)
		}
	}
}
func TestPlanningJSONStagesLateFailureAndCancel(t *testing.T) {
	for _, mode := range []string{"failure", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			l := reviewLibrary(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "GET" || r.Header.Get("X-PCO-API-Version") != "2018-11-01" {
					t.Fatalf("request=%+v", r)
				}
				if r.URL.Query().Get("offset") != "" {
					if mode == "cancel" {
						cancel()
						return nil, context.Canceled
					}
					return reviewResponse(200, `{"data":null}`), nil
				}
				return reviewResponse(200, reviewExportPage([]string{`{"type":"ServiceType","id":"1","attributes":{"name":"Sunday"}}`}, r.URL.Path, r.URL.Query(), true)), nil
			})
			c := injectedCommand(l)
			c.SetContext(withApp(ctx, appFrom(c)))
			out, stderr, e := runCommand(c, "services", "service-types", "list", "--all", "--json")
			if e == nil || out != "" || stderr == "" || calls != 2 {
				t.Fatalf("e=%v out=%s stderr=%s calls=%d", e, out, stderr, calls)
			}
			if mode == "cancel" && ExitCode(e) != 130 {
				t.Fatalf("exit=%d", ExitCode(e))
			}
		})
	}
}
func TestPlanningTableDoesNotInventCounts(t *testing.T) {
	l := reviewLibrary(func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "service_types/1") {
			return reviewResponse(200, `{"data":{"type":"ServiceType","id":"1","attributes":{"name":"Sunday"}}}`), nil
		}
		return reviewResponse(200, `{"data":[{"type":"Plan","id":"2","attributes":{"title":null}},{"type":"Plan","id":"3","attributes":{"title":"Zero","items_count":0}}]}`), nil
	})
	out, _, e := runCommand(injectedCommand(l), "services", "plans", "list", "--service-type", "1")
	if e != nil || !strings.Contains(out, "(untitled)") || strings.Count(out, "0") != 1 {
		t.Fatalf("err=%v out=%s", e, out)
	}
}
func TestPlanningEmptyLimitedAndSafeHumanOutput(t *testing.T) {
	for _, mode := range []string{"empty", "limited", "unsafe"} {
		t.Run(mode, func(t *testing.T) {
			l := reviewLibrary(func(r *http.Request) (*http.Response, error) {
				body := `{"data":[]}`
				if mode != "empty" {
					body = `{"data":[{"type":"ServiceType","id":"1","attributes":{"name":"Sun\u001b[2Jday"}},{"type":"ServiceType","id":"2","attributes":{"name":"Evening"}}]}`
				}
				return reviewResponse(200, body), nil
			})
			for _, machine := range []bool{false, true} {
				args := []string{"services", "service-types", "list", "--limit", "1"}
				if machine {
					args = append(args, "--json")
				}
				out, stderr, e := runCommand(injectedCommand(l), args...)
				if e != nil {
					t.Fatal(e)
				}
				if machine {
					v := decodeOutput(t, out)
					if mode == "limited" && !strings.Contains(string(v["meta"]), `"complete": false`) {
						t.Fatalf("meta=%s", v["meta"])
					}
				} else {
					if strings.Contains(out, "\x1b") {
						t.Fatalf("unsafe text=%q", out)
					}
					if mode == "empty" && !strings.Contains(out, "No results.") {
						t.Fatal(out)
					}
					if mode == "limited" && !strings.Contains(stderr, "More results") {
						t.Fatal(stderr)
					}
				}
			}
		})
	}
}
func TestPlanningPatchWireReadbackAndOutputFailure(t *testing.T) {
	for _, writerFails := range []bool{false, true} {
		t.Run(map[bool]string{true: "writer failure", false: "verified"}[writerFails], func(t *testing.T) {
			writes := 0
			reads := 0
			l := reviewLibrary(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/services/v2/service_types/1" {
					return reviewResponse(200, `{"data":{"type":"ServiceType","id":"1","attributes":{"name":"Sunday"}}}`), nil
				}
				if r.URL.Path != "/services/v2/service_types/1/plans/2" {
					t.Fatalf("path=%s", r.URL.Path)
				}
				attrs := `{"title":"Before"}`
				if r.Method == "PATCH" {
					writes++
					b, e := io.ReadAll(r.Body)
					if e != nil {
						t.Fatal(e)
					}
					var body map[string]any
					if json.Unmarshal(b, &body) != nil {
						t.Fatalf("body=%s", b)
					}
					if !strings.Contains(string(b), `"title":""`) || strings.Contains(string(b), "series_title") {
						t.Fatalf("presence body=%s", b)
					}
					attrs = `{"title":""}`
				} else {
					reads++
					if writes > 0 {
						attrs = `{"title":""}`
					}
				}
				return reviewResponse(200, `{"data":{"type":"Plan","id":"2","attributes":`+attrs+`}}`), nil
			})
			c := injectedCommand(l)
			var out, stderr bytes.Buffer
			var writer io.Writer = &out
			if writerFails {
				writer = errorWriter{}
			}
			e := executeCommand(c.Context(), c, []string{"services", "plans", "update", "2", "--service-type", "1", "--title=", "--json"}, writer, &stderr)
			if writes != 1 || reads != 3 {
				t.Fatalf("writes=%d reads=%d e=%v", writes, reads, e)
			}
			if writerFails {
				if e == nil || !errors.Is(e, io.ErrClosedPipe) || !strings.Contains(stderr.String(), `"partial_success"`) {
					t.Fatalf("e=%v stderr=%s", e, stderr.String())
				}
			} else if e != nil || !strings.Contains(out.String(), `"title": ""`) {
				t.Fatalf("e=%v out=%s", e, out.String())
			}
		})
	}
}

func TestPlanningAssignmentStatusHumanAndMachine(t *testing.T) {
	args := []string{"services", "scheduled-people", "list", "--service-type", "3001", "--plan", "4001"}
	out, _, e := runCommand(fixtureCommand(t), args...)
	if e != nil || !strings.Contains(out, "Confirmed") || !strings.Contains(out, "Unconfirmed") {
		t.Fatalf("e=%v out=%s", e, out)
	}
	out, _, e = runCommand(fixtureCommand(t), append(args, "--json")...)
	if e != nil || !strings.Contains(out, `"status": "C"`) || !strings.Contains(out, `"status": "U"`) {
		t.Fatalf("e=%v out=%s", e, out)
	}
}

func TestPlanningItemOrderExposesSongForFollowup(t *testing.T) {
	out, _, e := runCommand(fixtureCommand(t), "services", "items", "list", "--service-type", "3001", "--plan", "4001")
	if e != nil || !strings.Contains(out, "Song id") || !strings.Contains(out, "1001") {
		t.Fatalf("e=%v out=%s", e, out)
	}
}
