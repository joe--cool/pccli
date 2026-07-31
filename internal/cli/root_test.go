package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joe--cool/pccli/internal/services"
)

func TestSongsListWithMockOutputsJSON(t *testing.T) {
	t.Setenv("PCCLI_MOCK", "true")
	t.Setenv("PCCLI_MOCK_FIXTURE", filepath.Join("..", "..", "testdata", "mocks", "services-library.json"))
	t.Setenv("PCCLI_COLOR", "never")

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--json", "services", "songs", "list"})

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

func TestSongsSearchWithMockOutputsPlainTable(t *testing.T) {
	t.Setenv("PCCLI_MOCK", "true")
	t.Setenv("PCCLI_MOCK_FIXTURE", filepath.Join("..", "..", "testdata", "mocks", "services-library.json"))
	t.Setenv("PCCLI_COLOR", "never")

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"services", "songs", "search", "Amazing"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "Amazing Grace") || !strings.Contains(got, "1001") {
		t.Fatalf("expected search table output, got:\n%s", got)
	}
}

func TestSongsShowResolvesExactTitle(t *testing.T) {
	t.Setenv("PCCLI_MOCK", "true")
	t.Setenv("PCCLI_MOCK_FIXTURE", filepath.Join("..", "..", "testdata", "mocks", "services-library.json"))
	t.Setenv("PCCLI_COLOR", "never")

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"services", "songs", "show", "Amazing Grace"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "Worship Admin") || !strings.Contains(got, "Primary Key") {
		t.Fatalf("expected resolved song output, got:\n%s", got)
	}
	if !strings.Contains(got, "ARRANGEMENTS") || !strings.Contains(got, "DEFAULT ARRANGEMENT") || !strings.Contains(got, "Female Lead") {
		t.Fatalf("expected song output to include arrangement and key context, got:\n%s", got)
	}
	if strings.Contains(got, "Hidden") {
		t.Fatalf("did not expect hidden status without --hidden, got:\n%s", got)
	}
}

func TestSongsShowResolvesPartialTitleCaseInsensitively(t *testing.T) {
	t.Setenv("PCCLI_MOCK", "true")
	t.Setenv("PCCLI_MOCK_FIXTURE", filepath.Join("..", "..", "testdata", "mocks", "services-library.json"))
	t.Setenv("PCCLI_COLOR", "never")

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"services", "songs", "show", "grace"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "Amazing Grace") || !strings.Contains(got, "Full Band") {
		t.Fatalf("expected partial title to resolve song output, got:\n%s", got)
	}
}

func TestSongsShowRejectsHiddenSongWithoutFlag(t *testing.T) {
	t.Setenv("PCCLI_MOCK", "true")
	t.Setenv("PCCLI_MOCK_FIXTURE", filepath.Join("..", "..", "testdata", "mocks", "services-library.json"))
	t.Setenv("PCCLI_COLOR", "never")

	cmd := NewRootCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"services", "songs", "show", "1004"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected hidden song to require --hidden")
	}
	if !strings.Contains(err.Error(), "--hidden") {
		t.Fatalf("expected --hidden error, got: %v", err)
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
	cmd.SetArgs([]string{"services", "songs", "arrangements", "1001"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "Full Band") || !strings.Contains(got, "4:15") {
		t.Fatalf("expected arrangement table output, got:\n%s", got)
	}
}

func TestUpdateSongRequiresExactTitleForSearchResolution(t *testing.T) {
	t.Setenv("PCCLI_MOCK", "true")
	t.Setenv("PCCLI_MOCK_FIXTURE", filepath.Join("..", "..", "testdata", "mocks", "services-library.json"))
	t.Setenv("PCCLI_COLOR", "never")

	cmd := NewRootCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"services", "songs", "update", "Amazing", "--themes", "Grace"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected broad write search to fail")
	}
	if !strings.Contains(err.Error(), "no exact song title matches") {
		t.Fatalf("expected exact-title error, got: %v", err)
	}
}

func TestSongChoicesIncludeDisambiguatingMetadata(t *testing.T) {
	got := songChoices([]services.Song{{
		ID:         "1001",
		Title:      "Amazing Grace",
		Author:     "John Newton",
		CCLINumber: 22025,
	}})
	if !strings.Contains(got, `1001 "Amazing Grace"`) || !strings.Contains(got, "John Newton") || !strings.Contains(got, "CCLI 22025") {
		t.Fatalf("expected disambiguating metadata, got %q", got)
	}
}

func TestKeysWithMockOutputsPlainTable(t *testing.T) {
	t.Setenv("PCCLI_MOCK", "true")
	t.Setenv("PCCLI_MOCK_FIXTURE", filepath.Join("..", "..", "testdata", "mocks", "services-library.json"))
	t.Setenv("PCCLI_COLOR", "never")

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"services", "songs", "keys", "1001", "2001"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "Female Lead") || !strings.Contains(got, "Bb") {
		t.Fatalf("expected key table output, got:\n%s", got)
	}
}

func TestKeysUsesDefaultArrangementWhenOmitted(t *testing.T) {
	t.Setenv("PCCLI_MOCK", "true")
	t.Setenv("PCCLI_MOCK_FIXTURE", filepath.Join("..", "..", "testdata", "mocks", "services-library.json"))
	t.Setenv("PCCLI_COLOR", "never")

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"services", "songs", "keys", "1001"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "Default") || !strings.Contains(got, "Female Lead") {
		t.Fatalf("expected keys from default arrangement, got:\n%s", got)
	}
}

func TestKeysResolvesArrangementName(t *testing.T) {
	t.Setenv("PCCLI_MOCK", "true")
	t.Setenv("PCCLI_MOCK_FIXTURE", filepath.Join("..", "..", "testdata", "mocks", "services-library.json"))
	t.Setenv("PCCLI_COLOR", "never")

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"services", "songs", "keys", "1001", "Full Band"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "Female Lead") || !strings.Contains(got, "Bb") {
		t.Fatalf("expected key table output, got:\n%s", got)
	}
}

func TestAttachmentsWithMockOutputsJSON(t *testing.T) {
	t.Setenv("PCCLI_MOCK", "true")
	t.Setenv("PCCLI_MOCK_FIXTURE", filepath.Join("..", "..", "testdata", "mocks", "services-library.json"))
	t.Setenv("PCCLI_COLOR", "never")

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--json", "services", "songs", "attachments", "1001", "--arrangement", "2001", "--key", "3001"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, `"filename": "amazing-grace-full-band-g.pdf"`) {
		t.Fatalf("expected attachment JSON output, got:\n%s", got)
	}
}

func TestAttachmentsResolveArrangementAndKeyNames(t *testing.T) {
	t.Setenv("PCCLI_MOCK", "true")
	t.Setenv("PCCLI_MOCK_FIXTURE", filepath.Join("..", "..", "testdata", "mocks", "services-library.json"))
	t.Setenv("PCCLI_COLOR", "never")

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"services", "songs", "attachments", "1001", "--arrangement", "Full Band", "--key", "Default"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "amazing-grace-full-band-g.pdf") {
		t.Fatalf("expected key-scoped attachment output, got:\n%s", got)
	}
}

func TestCreateSongWithMockOutputsSummary(t *testing.T) {
	t.Setenv("PCCLI_MOCK", "true")
	t.Setenv("PCCLI_MOCK_FIXTURE", filepath.Join("..", "..", "testdata", "mocks", "services-library.json"))
	t.Setenv("PCCLI_COLOR", "never")

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"services", "songs", "create", "--ccli", "14181"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "How Great Thou Art") || !strings.Contains(got, "14181") {
		t.Fatalf("expected created song summary, got:\n%s", got)
	}
	if strings.Contains(got, "Hidden") {
		t.Fatalf("did not expect hidden status in ordinary create summary, got:\n%s", got)
	}
}

func TestDeleteSongRequiresYes(t *testing.T) {
	t.Setenv("PCCLI_MOCK", "true")
	t.Setenv("PCCLI_MOCK_FIXTURE", filepath.Join("..", "..", "testdata", "mocks", "services-library.json"))
	t.Setenv("PCCLI_COLOR", "never")

	cmd := NewRootCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"services", "songs", "delete", "1001"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected delete without --yes to fail")
	}
	if !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("expected --yes error, got: %v", err)
	}
}

func TestArrangementCreateWithMockOutputsSummary(t *testing.T) {
	t.Setenv("PCCLI_MOCK", "true")
	t.Setenv("PCCLI_MOCK_FIXTURE", filepath.Join("..", "..", "testdata", "mocks", "services-library.json"))
	t.Setenv("PCCLI_COLOR", "never")

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"services", "songs", "arrangements", "create", "1001", "--name", "Choir", "--key", "G", "--length", "4:15", "--sequence", "V1,V2,V3"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "Choir") || !strings.Contains(got, "4:15") {
		t.Fatalf("expected arrangement summary, got:\n%s", got)
	}
}

func TestKeyCreateUsesDefaultArrangement(t *testing.T) {
	t.Setenv("PCCLI_MOCK", "true")
	t.Setenv("PCCLI_MOCK_FIXTURE", filepath.Join("..", "..", "testdata", "mocks", "services-library.json"))
	t.Setenv("PCCLI_COLOR", "never")

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"services", "songs", "keys", "create", "1001", "--name", "Tenor Lead", "--start", "A"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "Tenor Lead") || !strings.Contains(got, "A") {
		t.Fatalf("expected key summary, got:\n%s", got)
	}
}

func TestAttachFileWithMockOutputsSummary(t *testing.T) {
	t.Setenv("PCCLI_MOCK", "true")
	t.Setenv("PCCLI_MOCK_FIXTURE", filepath.Join("..", "..", "testdata", "mocks", "services-library.json"))
	t.Setenv("PCCLI_COLOR", "never")

	temp := filepath.Join(t.TempDir(), "amazing-grace-chart.pdf")
	if err := os.WriteFile(temp, []byte("%PDF-1.7\n"), 0o600); err != nil {
		t.Fatalf("write temp upload file: %v", err)
	}

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"services", "songs", "attach", "1001", "--file", temp})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "Uploaded Chart") || !strings.Contains(got, "PDF") {
		t.Fatalf("expected attachment summary, got:\n%s", got)
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
	if !strings.Contains(got, "Planning Center Products") || !strings.Contains(got, "services") {
		t.Fatalf("expected product-grouped help in root output, got:\n%s", got)
	}
}

func TestServicesHelpGroupsMusicLibraryCommands(t *testing.T) {
	t.Setenv("PCCLI_COLOR", "never")

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"services", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "Music Library") || !strings.Contains(got, "songs") {
		t.Fatalf("expected Services help to group music-library commands, got:\n%s", got)
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
