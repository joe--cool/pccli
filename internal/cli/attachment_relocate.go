package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/joe--cool/pccli/v2/internal/output"
	"github.com/joe--cool/pccli/v2/internal/services"
	"github.com/spf13/cobra"
)

func newAttachmentRelocation(j *bool) *cobra.Command {
	var opts services.RelocationOptions
	var journalPath string
	command := dataCommand("relocate ATTACHMENT", "Preview or perform a verified PDF relocation", 1, func(c *cobra.Command, args []string) error {
		opts.SourceID = args[0]
		opts.To.Visibility = opts.From.Visibility
		opts.To.IncludeArchived = opts.From.IncludeArchived
		if opts.Apply && journalPath == "" {
			return usageError{fmt.Errorf("--apply requires a new --journal file")}
		}
		if !opts.Apply && journalPath != "" {
			return usageError{fmt.Errorf("--journal requires --apply")}
		}
		if opts.File != "" && opts.ReplacementID != "" {
			return usageError{fmt.Errorf("--file conflicts with --replacement")}
		}
		a, err := loadApp(c)
		if err != nil {
			return err
		}
		var journal *os.File
		defer func() {
			if journal != nil {
				_ = journal.Close()
			}
		}()
		if opts.Apply {
			opts.Checkpoint = func(stage services.AttachmentRelocation) error {
				if journal == nil {
					var err error
					journal, err = os.OpenFile(journalPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
					if err != nil {
						return err
					}
				}
				if err := json.NewEncoder(journal).Encode(stage); err != nil {
					return err
				}
				return journal.Sync()
			}
		}
		result, err := a.library.RelocateAttachment(c.Context(), opts)
		if err != nil {
			return err
		}
		if *j {
			return writeData(c, true, a, result, nil)
		}
		_, err = fmt.Fprintf(c.OutOrStdout(), "%s: %s (attachment %s)\n",
			result.Outcome, output.SafeText(result.Source.Filename), result.Source.ID)
		if err != nil {
			return err
		}
		if result.Replacement != nil && result.Replacement.Attachment != nil {
			_, err = fmt.Fprintf(c.OutOrStdout(), "Verified replacement attachment %s. Source key removed: %t\n",
				result.Replacement.Attachment.ID, result.KeyRemoved)
		}
		return err
	})
	selection(command, &opts.From, true, true)
	command.Flags().Lookup("song").Usage = "source song ID (required)"
	command.Flags().Lookup("arrangement").Usage = "source arrangement ID"
	command.Flags().Lookup("key").Usage = "source key ID; requires --arrangement"
	command.Flags().StringVar(&opts.To.Song, "to-song", "", "destination song ID (required)")
	command.Flags().StringVar(&opts.To.Arrangement, "to-arrangement", "", "destination arrangement ID")
	command.Flags().StringVar(&opts.To.Key, "to-key", "", "destination key ID")
	command.Flags().BoolVar(&opts.Apply, "apply", false, "create and verify the replacement before retiring the source")
	command.Flags().StringVar(&opts.File, "file", "", "local copy of the original PDF")
	command.Flags().StringVar(&opts.ReplacementID, "replacement", "", "resume using an explicitly identified existing replacement")
	command.Flags().StringVar(&journalPath, "journal", "", "write a new private JSONL recovery journal")
	command.Flags().BoolVar(&opts.PruneEmptyKey, "prune-empty-key", false, "remove the source key only if empty and unscheduled")
	return command
}
