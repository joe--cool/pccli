package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestColorIsPerRenderer(t *testing.T) {
	var colored, plain bytes.Buffer
	withColor := NewRenderer(&colored, false, "always")
	withoutColor := NewRenderer(&plain, false, "never")
	if !strings.Contains(withColor.Title("Songs"), "\x1b[") {
		t.Fatal("always mode must color output even when redirected")
	}
	if got := withoutColor.Title("Songs"); got != "Songs" {
		t.Fatalf("never mode contains styling: %q", got)
	}
	if !strings.Contains(withColor.Title("Songs"), "\x1b[") {
		t.Fatal("another renderer must not change color mode")
	}
}

func TestKeyValuesAlignsDisplayWidth(t *testing.T) {
	var out bytes.Buffer
	if err := NewRenderer(&out, false, "never").KeyValues([][2]string{{"曲", "first"}, {"ID", "second"}}); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "曲  first\nID  second\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestWriteJSONPreservesValue(t *testing.T) {
	var out bytes.Buffer
	if err := NewRenderer(&out, true, "always").WriteJSON([]string{"first", "second"}); err != nil {
		t.Fatal(err)
	}
	var got []string
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("expected raw array: %v", err)
	}
	if len(got) != 2 || got[0] != "first" {
		t.Fatalf("unexpected JSON: %s", out.String())
	}
}

func TestTablePlainAndEmpty(t *testing.T) {
	var out bytes.Buffer
	renderer := NewRenderer(&out, false, "never")
	if err := renderer.Table([]string{"ID", "TITLE"}, [][]string{{"1", "Example"}}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "\x1b[") || !strings.Contains(out.String(), "Example") {
		t.Fatalf("unexpected table: %q", out.String())
	}
	out.Reset()
	if err := renderer.Table([]string{"ID"}, nil); err != nil {
		t.Fatal(err)
	}
	if out.String() != "No results.\n" {
		t.Fatalf("unexpected empty table: %q", out.String())
	}
}

func TestHumanOutputSanitizesRemoteText(t *testing.T) {
	value := "\x1b[31mred\x1b[0m\x1b]0;hostile title\x07\nnext\rline"
	if got := SafeText(value); got != "red next line" {
		t.Fatalf("unexpected sanitized text: %q", got)
	}
	var out bytes.Buffer
	renderer := NewRenderer(&out, false, "never")
	if err := renderer.Table([]string{"TITLE"}, [][]string{{value}}); err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(out.String(), "\x1b\x07\r") {
		t.Fatalf("unsafe table: %q", out.String())
	}
	out.Reset()
	if err := renderer.KeyValues([][2]string{{"TITLE", value}}); err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(out.String(), "\x1b\x07\r") {
		t.Fatalf("unsafe fields: %q", out.String())
	}
}
