package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joe--cool/pccli/v2/internal/output"
	"github.com/joe--cool/pccli/v2/internal/services"
	"github.com/spf13/cobra"
)

func newLibraryCommand(j *bool) *cobra.Command {
	root := group("library", "Export and audit the Services song library")
	var includes []string
	var dest string
	var archive, all, progress bool
	var workers int
	var vis services.Visibility
	export := dataCommand("export", "Write a complete, reusable library snapshot", 0, func(c *cobra.Command, _ []string) error {
		if !all || dest == "" {
			return usageError{fmt.Errorf("library export requires --all and --output")}
		}
		var include services.LibraryInclude
		for _, name := range includes {
			switch name {
			case "arrangements":
				include.Arrangements = true
			case "keys":
				include.Arrangements, include.Keys = true, true
			case "attachments":
				include.Arrangements, include.Keys, include.Attachments = true, true, true
			case "tags":
				include.Arrangements, include.Tags = true, true
			default:
				return usageError{fmt.Errorf("unsupported library inclusion %q", name)}
			}
		}
		a, err := loadApp(c)
		if err != nil {
			return err
		}
		var report func(int, int)
		if progress {
			report = func(done, total int) {
				if done == 1 || done%25 == 0 || done == total {
					_, _ = fmt.Fprintf(c.ErrOrStderr(), "Exported %d/%d songs (%d HTTP attempts)\n", done, total, a.client.Requests())
				}
			}
		}
		snapshot, err := a.library.ExportLibrary(c.Context(), vis, archive, include, workers, report)
		if err != nil {
			return err
		}
		if _, err := services.AuditLibrary(snapshot, nil); err != nil {
			return err
		}
		if err := saveSnapshot(c.Context(), dest, snapshot); err != nil {
			return err
		}
		if *j {
			return writeData(c, true, a, map[string]any{"songs": len(snapshot.Songs), "http_attempts": a.client.Requests()}, map[string]any{"complete": true})
		}
		_, err = fmt.Fprintf(c.OutOrStdout(), "Exported %d songs (%d HTTP attempts)\n", len(snapshot.Songs), a.client.Requests())
		return err
	})
	export.Flags().BoolVar(&all, "all", false, "export the complete selected library")
	export.Flags().StringVar(&dest, "output", "", "write the complete snapshot to this file")
	export.Flags().StringSliceVar(&includes, "include", []string{"arrangements", "keys", "attachments", "tags"}, "resource groups to include")
	export.Flags().IntVar(&workers, "workers", 2, "concurrent song reads (1–4)")
	export.Flags().BoolVar(&progress, "progress", false, "print progress and request counts to stderr")
	export.Flags().BoolVar(&archive, "include-archived", false, "include archived arrangements explicitly")
	visibility(export, &vis)
	root.AddCommand(export)

	var input, expected string
	audit := dataCommand("audit", "Inspect a snapshot or compare it with an expected inventory", 0, func(c *cobra.Command, _ []string) error {
		if input == "" {
			return usageError{fmt.Errorf("library audit requires --snapshot")}
		}
		snapshot, err := readSnapshot(input)
		if err != nil {
			return err
		}
		var want *services.LibrarySnapshot
		if expected != "" {
			value, err := readSnapshot(expected)
			if err != nil {
				return err
			}
			want = &value
		}
		report, err := services.AuditLibrary(snapshot, want)
		if err != nil {
			return err
		}
		if *j {
			return writeJSONContext(c.Context(), c.OutOrStdout(), envelope{Schema: "pccli/v1", Data: report})
		}
		if _, err := fmt.Fprintf(c.OutOrStdout(), "%d songs · %d arrangements · %d keys · %d attachments · %d findings\n", report.Songs, report.Arrangements, report.Keys, report.Attachments, len(report.Findings)); err != nil {
			return err
		}
		for _, finding := range report.Findings {
			if _, err := fmt.Fprintf(c.OutOrStdout(), "%s  %s\n", finding.Code, finding.Scope); err != nil {
				return err
			}
		}
		return nil
	})
	audit.Flags().StringVar(&input, "snapshot", "", "read the observed library snapshot")
	audit.Flags().StringVar(&expected, "expected", "", "compare with an expected inventory in snapshot format")
	root.AddCommand(audit)
	return root
}

func readSnapshot(path string) (services.LibrarySnapshot, error) {
	var snapshot services.LibrarySnapshot
	b, err := os.ReadFile(path)
	if err != nil {
		return snapshot, err
	}
	err = json.Unmarshal(b, &snapshot)
	return snapshot, err
}

func saveSnapshot(ctx context.Context, path string, data services.LibrarySnapshot) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".pccli-library-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err := writeJSONContext(ctx, file, data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func writeArrangementFiles(c *cobra.Command, j bool, a *app, data services.LibraryArrangement) error {
	if j {
		return writeData(c, true, a, data, map[string]any{"complete": true})
	}
	renderer := output.NewRenderer(c.OutOrStdout(), false, a.cfg.Color)
	if err := renderResource(renderer, data.Arrangement); err != nil {
		return err
	}
	write := func(name string, rows []services.Attachment) error {
		if _, err := fmt.Fprintln(c.OutOrStdout(), "\n"+output.SafeText(name)); err != nil {
			return err
		}
		return writeList(c, false, a, services.ListResult[services.Attachment]{Data: rows})
	}
	if err := write("Arrangement", data.Attachments); err != nil {
		return err
	}
	for _, key := range data.Keys {
		if err := write(key.Key.Name+" (ID "+key.Key.ID+")", key.Attachments); err != nil {
			return err
		}
	}
	return nil
}
