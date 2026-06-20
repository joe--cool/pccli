package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestSongsListWithMockOutputsJSON(t *testing.T) {
	t.Setenv("PCCLI_MOCK", "true")
	t.Setenv("PCCLI_MOCK_FIXTURE", filepath.Join("..", "..", "testdata", "mocks", "services-library.json"))
	t.Setenv("PCCLI_COLOR", "never")

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--json", "songs", "list"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, `"title": "Amazing Grace"`) {
		t.Fatalf("expected JSON song output, got:\n%s", got)
	}
	if !strings.Contains(got, `"ccli_number": 22025`) {
		t.Fatalf("expected CCLI number in JSON output, got:\n%s", got)
	}
	if strings.Contains(got, "unofficial") {
		t.Fatalf("did not expect banner in JSON output, got:\n%s", got)
	}
}

func TestArrangementsWithMockOutputsPlainTable(t *testing.T) {
	t.Setenv("PCCLI_MOCK", "true")
	t.Setenv("PCCLI_MOCK_FIXTURE", filepath.Join("..", "..", "testdata", "mocks", "services-library.json"))
	t.Setenv("PCCLI_COLOR", "never")

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"songs", "arrangements", "1001"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "Full Band") || !strings.Contains(got, "4:15") {
		t.Fatalf("expected arrangement table output, got:\n%s", got)
	}
}

func TestRootHelpShowsBannerWithoutCredentials(t *testing.T) {
	t.Setenv("PCCLI_COLOR", "never")

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "The unofficial CLI for Planning Center") {
		t.Fatalf("expected banner in root output, got:\n%s", got)
	}
	if !strings.Contains(got, "Usage:") {
		t.Fatalf("expected help in root output, got:\n%s", got)
	}
}

func TestRootHelpCanForceBannerColor(t *testing.T) {
	t.Setenv("PCCLI_COLOR", "always")

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "\x1b[38;2;150;180;252m") {
		t.Fatalf("expected ANSI color in forced-color banner, got:\n%s", got)
	}
	if !strings.Contains(got, "\x1b[38;2;96;68;200m") {
		t.Fatalf("expected restored full-color banner gradient, got:\n%s", got)
	}
	if !strings.Contains(got, "\x1b[0m\n\npccli helps") {
		t.Fatalf("expected banner color to reset before help text, got:\n%s", got)
	}
}
