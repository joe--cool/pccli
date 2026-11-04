package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joe--cool/pccli/v2/internal/config"
	"github.com/joe--cool/pccli/v2/internal/planningcenter"
	"github.com/joe--cool/pccli/v2/internal/services"
	"github.com/spf13/cobra"
)

// All operational tests inject synthetic data: they never load credential files.
func fixtureCommand(t *testing.T) *cobra.Command {
	t.Helper()
	p := filepath.Join("..", "..", "testdata", "mocks", "services-library.json")
	tr, e := planningcenter.NewMockTransport(p)
	if e != nil {
		t.Fatal(e)
	}
	cfg := config.Config{BaseURL: "https://example.test", UploadURL: "https://upload.example.test", Color: "never"}
	return injectedCommand(services.NewLibrary(planningcenter.NewClient(cfg, tr)))
}
func injectedCommand(l *services.Library) *cobra.Command {
	c := NewRootCommand()
	c.SetContext(withApp(context.Background(), &app{cfg: config.Config{Color: "never"}, library: l}))
	return c
}
func runCommand(c *cobra.Command, args ...string) (string, string, error) {
	var out, stderr bytes.Buffer
	e := executeCommand(c.Context(), c, args, &out, &stderr)
	return out.String(), stderr.String(), e
}
func decodeOutput(t *testing.T, s string) map[string]json.RawMessage {
	t.Helper()
	var out map[string]json.RawMessage
	if e := json.Unmarshal([]byte(s), &out); e != nil {
		t.Fatalf("invalid JSON %q: %v", s, e)
	}
	if string(out["schema"]) != `"pccli/v1"` {
		t.Fatalf("wrong schema: %s", s)
	}
	return out
}
func TestReadJourneysPlainAndJSON(t *testing.T) {
	cases := [][]string{
		{"songs", "list", "--search", "Grace"},
		{"songs", "show", "Amazing Grace"},
		{"songs", "show", "grace", "--arrangement", "Full Band"},
		{"arrangements", "list", "--song", "1001"},
		{"arrangements", "show", "2001", "--song", "1001"},
		{"keys", "list", "--song", "1001", "--arrangement", "2001"},
		{"keys", "show", "3001", "--song", "1001", "--arrangement", "2001"},
		{"attachments", "list", "--song", "1001"},
		{"attachments", "list", "--song", "1001", "--arrangement", "2001", "--key", "3001"},
	}
	for _, args := range cases {
		for _, machine := range []bool{false, true} {
			t.Run(strings.Join(args, " ")+map[bool]string{true: " JSON"}[machine], func(t *testing.T) {
				c := fixtureCommand(t)
				argv := append([]string{"services"}, args...)
				if machine {
					argv = append(argv, "--json")
				}
				out, stderr, e := runCommand(c, argv...)
				if e != nil {
					t.Fatalf("%v; %s", e, stderr)
				}
				if out == "" {
					t.Fatal("empty output")
				}
				if strings.Contains(out, "unofficial") {
					t.Fatal("banner in data output")
				}
				if machine {
					got := decodeOutput(t, out)
					if got["data"] == nil || got["meta"] == nil {
						t.Fatal("missing contract fields")
					}
				}
			})
		}
	}
}
func TestSongDetailDoesNotInventArrangement(t *testing.T) {
	out, _, e := runCommand(fixtureCommand(t), "services", "songs", "show", "1001", "--json")
	if e != nil {
		t.Fatal(e)
	}
	got := decodeOutput(t, out)
	var data map[string]json.RawMessage
	_ = json.Unmarshal(got["data"], &data)
	if string(data["selected_arrangement"]) != "null" || string(data["keys"]) != "null" {
		t.Fatalf("implicit selection: %s", out)
	}
}
func TestListLimitAndEmptyCompleteness(t *testing.T) {
	for _, args := range [][]string{{"--limit", "1"}, {"--search", "not-in-fixture"}, {"--all"}} {
		out, _, e := runCommand(fixtureCommand(t), append([]string{"services", "songs", "list", "--json"}, args...)...)
		if e != nil {
			t.Fatal(e)
		}
		got := decodeOutput(t, out)
		var meta services.ListMeta
		_ = json.Unmarshal(got["meta"], &meta)
		if args[0] == "--limit" && (!meta.HasMore || meta.Complete || meta.Returned != 1) {
			t.Fatalf("wrong bound %s", out)
		}
		if args[0] == "--search" && string(got["data"]) != "[]" {
			t.Fatalf("empty must be []: %s", out)
		}
	}
}
func TestInvalidUsageBeforeConfiguration(t *testing.T) {
	// Invalid configuration proves these paths validate before config/credentials.
	t.Setenv("PCCLI_TIMEOUT", "invalid")
	cases := [][]string{
		{"services", "bogus"}, {"services", "songs", "list", "extra"},
		{"services", "songs", "list", "--limit", "0"}, {"services", "songs", "list", "--limit", "25", "--all"},
		{"services", "songs", "list", "--search", "", "--title", "Grace"},
		{"services", "songs", "list", "--ccli", "0"}, {"services", "songs", "list", "--ccli", "-1"},
		{"services", "songs", "list", "--order", "bogus"}, {"services", "songs", "show", "id:0"},
		{"services", "songs", "show", "Grace", "--visibility", "invalid"},
		{"services", "songs", "delete", "1"}, {"services", "songs", "update", "1"},
		{"services", "arrangements", "list"}, {"services", "arrangements", "show", "1", "--song", "id:0"},
		{"services", "arrangements", "update", "1", "--song", "1", "--length", "999999999999999999999:99"},
		{"services", "arrangements", "unarchive", "1", "--song", "1"},
		{"services", "keys", "create", "--song", "1", "--name", "Lead", "--start", "G"},
		{"services", "attachments", "add", "--song", "1", "--key", "2", "--url", "https://example.test"},
		{"services", "attachments", "update", "1", "--song", "1"},
		{"services", "attachments", "update", "1", "--song", "1", "--url", "bad"},
		{"services", "tags", "replace", "--song", "1"},
		{"services", "songs", "last-scheduled", "1"},
		{"services", "songs", "last-scheduled", "1", "--service-type", "0"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var out, stderr bytes.Buffer
			err := ExecuteArgs(context.Background(), append(args, "--json"), &out, &stderr)
			if ExitCode(err) != 2 {
				t.Fatalf("want usage2, got %v: %s", err, stderr.String())
			}
			if out.Len() != 0 {
				t.Fatalf("unexpected stdout %s", out.String())
			}
			got := decodeOutput(t, stderr.String())
			var body map[string]any
			_ = json.Unmarshal(got["error"], &body)
			if body["code"] != "invalid_input" {
				t.Fatalf("unexpected error %s", stderr.String())
			}
		})
	}
}
func TestJSONFlagValueIsNotAFlag(t *testing.T) {
	var out, stderr bytes.Buffer
	err := ExecuteArgs(context.Background(), []string{"services", "songs", "list", "--title", "--json", "--bad"}, &out, &stderr)
	if ExitCode(err) != 2 {
		t.Fatalf("wrong code: %v", err)
	}
	if !strings.HasPrefix(stderr.String(), "Error:") {
		t.Fatalf("literal --json changed format: %s", stderr.String())
	}
}
func TestHelpAndVersionNeverLoadConfiguration(t *testing.T) {
	t.Setenv("PCCLI_TIMEOUT", "invalid")
	for _, args := range [][]string{{"--help"}, {"--version"}, {"services", "--help"}, {"services", "songs", "--help"}, {"services", "arrangements", "unarchive", "--help"}, {"services", "attachments", "add", "--help"}} {
		var out, stderr bytes.Buffer
		if e := ExecuteArgs(context.Background(), args, &out, &stderr); e != nil {
			t.Fatalf("%v: %v", args, e)
		}
		if out.Len() == 0 {
			t.Fatalf("empty help: %v", args)
		}
	}
}
func TestRemovedCommandsAreNotAliases(t *testing.T) {
	for _, args := range [][]string{{"services", "songs", "search", "Grace"}, {"services", "songs", "arrangements", "1"}, {"services", "songs", "attach", "1", "--url", "https://example.test"}} {
		_, _, e := runCommand(fixtureCommand(t), args...)
		if ExitCode(e) != 2 {
			t.Fatalf("removed form not rejected: %v %v", args, e)
		}
	}
}

// Mutations go through the real workflow/adapter, inspecting actual payloads.
type mutationAPI struct {
	attrs map[string]any
	patch map[string]any
	calls int
}

func (a *mutationAPI) Get(_ context.Context, path string, _ url.Values, dst any) error {
	a.calls++
	return a.respond(path, nil, dst)
}
func (a *mutationAPI) Post(_ context.Context, path string, _ url.Values, body, dst any) error {
	return a.respond(path, body, dst)
}
func (a *mutationAPI) Patch(_ context.Context, path string, _ url.Values, body, dst any) error {
	return a.respond(path, body, dst)
}
func (a *mutationAPI) Delete(context.Context, string, url.Values) error { return nil }
func (a *mutationAPI) UploadFile(context.Context, string, any) error {
	return errors.New("unexpected upload")
}
func (a *mutationAPI) respond(path string, body, dst any) error {
	if body != nil {
		b, _ := json.Marshal(body)
		var p struct {
			Data struct {
				Attributes map[string]any `json:"attributes"`
			} `json:"data"`
		}
		_ = json.Unmarshal(b, &p)
		a.patch = p.Data.Attributes
		for k, v := range a.patch {
			a.attrs[k] = v
		}
	}
	resource := map[string]any{"type": "Song", "id": "1", "attributes": a.attrs}
	var payload any = map[string]any{"data": resource}
	if path == "/services/v2/songs" && body == nil {
		payload = map[string]any{"data": []any{resource}}
	}
	b, _ := json.Marshal(payload)
	return json.Unmarshal(b, dst)
}
func TestMutationPresenceThroughCLI(t *testing.T) {
	for _, j := range []bool{false, true} {
		a := &mutationAPI{attrs: map[string]any{"title": "Grace", "hidden": true, "author": "Writer"}}
		c := injectedCommand(services.NewLibrary(a))
		args := []string{"services", "songs", "update", "1", "--visibility", "hidden", "--hidden=false", "--author="}
		if j {
			args = append(args, "--json")
		}
		out, stderr, e := runCommand(c, args...)
		if e != nil {
			t.Fatalf("%v %s", e, stderr)
		}
		if a.patch["hidden"] != false || a.patch["author"] != "" {
			t.Fatalf("values dropped: %#v", a.patch)
		}
		if j {
			got := decodeOutput(t, out)
			var resource map[string]any
			_ = json.Unmarshal(got["data"], &resource)
			if resource["hidden"] != false {
				t.Fatalf("false omitted %s", out)
			}
		}
	}
}
func TestStructuredOperationalErrors(t *testing.T) {
	a := &mutationAPI{attrs: map[string]any{"title": "Grace", "hidden": true}}
	out, stderr, e := runCommand(injectedCommand(services.NewLibrary(a)), "services", "songs", "show", "1", "--json")
	if e == nil || ExitCode(e) != 1 || out != "" {
		t.Fatalf("unexpected result %q %v", out, e)
	}
	got := decodeOutput(t, stderr)
	if got["error"] == nil {
		t.Fatal("missing structured error")
	}
}
func TestJSONPublicationFailureAndCancellation(t *testing.T) {
	if e := writeJSON(errorWriter{}, map[string]string{"ok": "yes"}); e == nil {
		t.Fatal("write failure lost")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out bytes.Buffer
	if e := writeJSONContext(ctx, &out, map[string]string{"ok": "yes"}); !errors.Is(e, context.Canceled) || out.Len() != 0 {
		t.Fatalf("cancellation ignored: %v %q", e, out.String())
	}
	if e := writeJSON(&out, map[string]any{"bad": make(chan int)}); e == nil || out.Len() != 0 {
		t.Fatalf("encoding failure published data")
	}
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
