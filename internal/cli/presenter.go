package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/joe--cool/pccli/v2/internal/output"
	"github.com/joe--cool/pccli/v2/internal/services"
	"github.com/spf13/cobra"
)

type envelope struct {
	Schema string `json:"schema"`
	Data   any    `json:"data"`
	Meta   any    `json:"meta"`
}

func receipt(resource, target string) map[string]any {
	return map[string]any{"deleted": true, "resource": resource, "target": target}
}
func writeJSON(out io.Writer, value any) error {
	return writeJSONContext(context.Background(), out, value)
}

// Complete JSON exports encode one record at a time to private disk. Nothing is
// published until the validated walk succeeds, including the last API page.
func writeStreamList[T any](c *cobra.Command, walk func(func(T) error) (services.ListMeta, error)) error {
	f, err := os.CreateTemp("", "pccli-export-*.json")
	if err != nil {
		return fmt.Errorf("prepare JSON export: %w", err)
	}
	defer os.Remove(f.Name())
	defer f.Close()
	out := contextWriter{c.Context(), f}
	if _, err = io.WriteString(out, "{\"schema\":\"pccli/v1\",\"data\":["); err != nil {
		return err
	}
	encoder := json.NewEncoder(out)
	count := 0
	meta, err := walk(func(item T) error {
		if count > 0 {
			if _, e := io.WriteString(out, ","); e != nil {
				return e
			}
		}
		if e := encoder.Encode(item); e != nil {
			return e
		}
		count++
		return nil
	})
	if err != nil {
		return err
	}
	if !meta.Complete || meta.HasMore || meta.Returned != count {
		return fmt.Errorf("export did not establish a complete result")
	}
	if _, err = io.WriteString(out, "],\"meta\":"); err != nil {
		return err
	}
	if err = encoder.Encode(meta); err != nil {
		return err
	}
	if _, err = io.WriteString(out, "}\n"); err != nil {
		return err
	}
	if _, err = f.Seek(0, 0); err != nil {
		return err
	}
	_, err = io.Copy(contextWriter{c.Context(), c.OutOrStdout()}, f)
	return err
}
func writeJSONContext(ctx context.Context, out io.Writer, value any) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	f, e := os.CreateTemp("", "pccli-output-*.json")
	if e != nil {
		return fmt.Errorf("prepare JSON output: %w", e)
	}
	defer os.Remove(f.Name())
	defer f.Close()
	enc := json.NewEncoder(contextWriter{ctx, f})
	enc.SetIndent("", "  ")
	if e = enc.Encode(value); e != nil {
		return fmt.Errorf("encode output: %w", e)
	}
	if _, e = f.Seek(0, 0); e != nil {
		return e
	}
	_, e = io.Copy(contextWriter{ctx, out}, f)
	return e
}
func writeData(c *cobra.Command, j bool, a *app, data, meta any) error {
	if meta == nil {
		meta = map[string]any{}
	}
	if j {
		return writeJSONContext(c.Context(), c.OutOrStdout(), envelope{"pccli/v1", data, meta})
	}
	return renderResource(output.NewRenderer(c.OutOrStdout(), false, a.cfg.Color), data)
}
func writeList[T any](c *cobra.Command, j bool, a *app, r services.ListResult[T]) error {
	if j {
		return writeData(c, true, a, r.Data, r.Meta)
	}
	if r.Meta.HasMore {
		if _, e := fmt.Fprintln(c.ErrOrStderr(), "More results available; use --all for the complete collection."); e != nil {
			return e
		}
	}
	renderer := output.NewRenderer(c.OutOrStdout(), false, a.cfg.Color)
	if len(r.Data) == 0 {
		_, e := fmt.Fprintln(renderer.Out, "No results.")
		return e
	}
	return renderTable(renderer, r.Data)
}
func writeSongDetail(c *cobra.Command, j bool, a *app, r services.SongDetail) error {
	var selected any
	var keys any
	var km any
	if r.Arrangement != nil {
		selected = r.Arrangement.Arrangement
		keys = r.Arrangement.Keys.Data
		km = r.Arrangement.Keys.Meta
	}
	if j {
		return writeData(c, true, a, map[string]any{"song": r.Song, "arrangements": r.Arrangements.Data, "selected_arrangement": selected, "keys": keys}, map[string]any{"arrangements": r.Arrangements.Meta, "keys": km})
	}
	renderer := output.NewRenderer(c.OutOrStdout(), false, a.cfg.Color)
	if e := renderResource(renderer, r.Song); e != nil {
		return e
	}
	if _, e := fmt.Fprintln(renderer.Out, "\nARRANGEMENTS"); e != nil {
		return e
	}
	if e := writeList(c, false, a, r.Arrangements); e != nil {
		return e
	}
	if r.Arrangement != nil {
		if _, e := fmt.Fprintln(renderer.Out, "\nSELECTED ARRANGEMENT"); e != nil {
			return e
		}
		if e := renderResource(renderer, r.Arrangement.Arrangement); e != nil {
			return e
		}
		if _, e := fmt.Fprintln(renderer.Out, "\nKEYS"); e != nil {
			return e
		}
		return writeList(c, false, a, r.Arrangement.Keys)
	}
	return nil
}
func writeArrangementDetail(c *cobra.Command, j bool, a *app, r services.ArrangementDetail) error {
	if j {
		return writeData(c, true, a, map[string]any{"arrangement": r.Arrangement, "keys": r.Keys.Data}, map[string]any{"keys": r.Keys.Meta})
	}
	renderer := output.NewRenderer(c.OutOrStdout(), false, a.cfg.Color)
	if e := renderResource(renderer, r.Arrangement); e != nil {
		return e
	}
	if _, e := fmt.Fprintln(renderer.Out, "\nKEYS"); e != nil {
		return e
	}
	return writeList(c, false, a, r.Keys)
}
func object(value any) map[string]any {
	b, _ := json.Marshal(value)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m
}
func label(k string) string {
	switch k {
	case "id":
		return "ID"
	case "ccli_number":
		return "CCLI"
	case "bpm":
		return "BPM"
	case "url":
		return "URL"
	case "file_size":
		return "Size"
	}
	if k == "" {
		return ""
	}
	return strings.ToUpper(k[:1]) + strings.ReplaceAll(k[1:], "_", " ")
}
func text(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return ""
	case float64:
		if x == 0 {
			return ""
		}
		return fmt.Sprintf("%v", x)
	case []any:
		parts := []string{}
		for _, item := range x {
			if s := text(item); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, ", ")
	default:
		b, _ := json.Marshal(x)
		return string(b)
	}
}
func renderResource(r output.Renderer, value any) error {
	if result, ok := value.(services.PlanCopyResult); ok {
		if e := renderResource(r, result.Plan); e != nil {
			return e
		}
		if e := r.KeyValues([][2]string{{"Source plan ID", result.SourcePlanID}, {"Copied", strings.Join(result.Components, ", ")}, {"Outcome", result.Outcome}}); e != nil {
			return e
		}
		rows := [][]string{}
		for _, receipt := range result.Created {
			rows = append(rows, []string{receipt.Resource, receipt.ID})
		}
		if len(rows) > 0 {
			if _, e := fmt.Fprintln(r.Out, "\nCREATED RECORDS"); e != nil {
				return e
			}
			if e := r.Table([]string{"Resource", "ID"}, rows); e != nil {
				return e
			}
		}
		if len(result.UnverifiedFields) > 0 {
			rows = nil
			for _, field := range result.UnverifiedFields {
				rows = append(rows, []string{field.Resource, field.ID, field.Field, field.RequestedID})
			}
			if _, e := fmt.Fprintln(r.Out, "\nUNVERIFIED FIELDS"); e != nil {
				return e
			}
			return r.Table([]string{"Resource", "ID", "Field", "Requested ID"}, rows)
		}
		return nil
	}

	if result, ok := value.(services.NeededPositionResult); ok {
		if e := renderResource(r, result.NeededPosition); e != nil {
			return e
		}
		requested := ""
		if result.RequestedPositionID != nil {
			requested = *result.RequestedPositionID
		}
		return r.KeyValues([][2]string{{"Outcome", result.Outcome}, {"Unverified fields", strings.Join(result.UnverifiedFields, ", ")}, {"Requested position ID", requested}})
	}

	if update, ok := value.(services.AttachmentUpdateResult); ok {
		if e := renderResource(r, update.Attachment); e != nil {
			return e
		}
		return r.KeyValues([][2]string{{"Outcome", update.Outcome}, {"Unverified fields", strings.Join(update.UnverifiedFields, ", ")}})
	}
	if outcome, ok := value.(services.AttachmentResult); ok {
		if outcome.Attachment != nil {
			if e := renderResource(r, *outcome.Attachment); e != nil {
				return e
			}
		}
		if outcome.Upload != nil {
			return r.KeyValues([][2]string{{"Upload ID", outcome.Upload.ID}, {"Expires", outcome.Upload.ExpiresAt}, {"Outcome", outcome.Outcome}})
		}
		return nil
	}
	m := object(value)
	if attrs, ok := m["attributes"].(map[string]any); ok {
		attrs["id"] = m["id"]
		m = attrs
	}
	ordered := []string{"title", "name", "display_name", "id", "song_id", "arrangement_id", "author", "admin", "ccli_number", "copyright", "hidden", "archived_at", "starting_key", "ending_key", "chord_chart_key", "bpm", "meter", "length", "notes", "sequence", "themes", "last_scheduled", "last_scheduled_at", "filename", "file_size", "content_type", "remote_link", "url"}
	rows := [][2]string{}
	seen := map[string]bool{}
	for _, k := range ordered {
		seen[k] = true
		v := text(m[k])
		if v != "" {
			rows = append(rows, [2]string{label(k), v})
		}
	}
	extras := []string{}
	for k := range m {
		if !seen[k] && text(m[k]) != "" {
			extras = append(extras, k)
		}
	}
	sort.Strings(extras)
	if e := r.KeyValues(rows); e != nil {
		return e
	}
	if len(extras) > 0 {
		if _, e := fmt.Fprintln(r.Out, "\nDETAILS"); e != nil {
			return e
		}
		rows = nil
		for _, k := range extras {
			rows = append(rows, [2]string{label(k), text(m[k])})
		}
		return r.KeyValues(rows)
	}
	return nil
}
func renderTable[T any](r output.Renderer, items []T) error {
	if handled, err := planningTable(r, items); handled {
		return err
	}
	first := object(items[0])
	columns := []string{"id"}
	for _, k := range []string{"title", "name", "display_name", "author", "ccli_number", "starting_key", "ending_key", "chord_chart_key", "bpm", "meter", "filename", "tag_group_id", "tag_group_name", "tags_for", "content_type"} {
		if _, ok := first[k]; ok {
			columns = append(columns, k)
		}
	}
	if len(columns) == 1 {
		columns = append(columns, "attributes")
	}
	headers := []string{}
	for _, k := range columns {
		headers = append(headers, label(k))
	}
	rows := [][]string{}
	for _, x := range items {
		m := object(x)
		row := []string{}
		for _, k := range columns {
			row = append(row, text(m[k]))
		}
		rows = append(rows, row)
	}
	return r.Table(headers, rows)
}

type contextWriter struct {
	ctx context.Context
	out io.Writer
}

func (w contextWriter) Write(p []byte) (int, error) {
	if e := w.ctx.Err(); e != nil {
		return 0, e
	}
	return w.out.Write(p)
}

// A failed terminal/pipe write cannot undo an already confirmed remote change.
type committedOutputError struct {
	Result any
	Cause  error
}

func (e *committedOutputError) Error() string {
	return "remote change completed but output failed; inspect the resource before repeating"
}
func (e *committedOutputError) Unwrap() error { return e.Cause }
func writeMutationData(c *cobra.Command, j bool, a *app, result, changes any) error {
	err := writeData(c, j, a, result, nil)
	if err == nil && !j && changes != nil {
		values := object(changes)
		keys := []string{}
		for k := range values {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		if _, err = fmt.Fprintln(c.OutOrStdout(), "\nREQUESTED CHANGES"); err == nil {
			rows := [][2]string{}
			for _, k := range keys {
				b, _ := json.Marshal(values[k])
				rows = append(rows, [2]string{label(k), string(b)})
			}
			err = output.NewRenderer(c.OutOrStdout(), false, a.cfg.Color).KeyValues(rows)
		}
	}
	if err != nil {
		if receipt, ok := result.(services.AttachmentResult); ok {
			return &services.Error{Code: "partial_success", Message: "attachment was created but output failed; do not repeat creation", Recovery: &receipt, Cause: err}
		}
		return &committedOutputError{Result: result, Cause: err}
	}
	return nil
}
