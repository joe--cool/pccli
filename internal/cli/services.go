package cli

import (
	"fmt"
	"strings"

	"github.com/joe--cool/pccli/v2/internal/services"
	"github.com/spf13/cobra"
)

// Resource constructors describe the public grammar; Services owns selection
// and mutation policy. Parent flags never infer a default arrangement.
func newServicesCommand(jsonOutput *bool) *cobra.Command {
	c := group("services", "Work with Planning Center Services")
	c.GroupID = "products"
	c.AddCommand(newServiceTypesCommand(jsonOutput), newPlansCommand(jsonOutput), newItemsCommand(jsonOutput), newTeamsCommand(jsonOutput), newPlanTimesCommand(jsonOutput), newScheduledPeopleCommand(jsonOutput), newSongsCommand(jsonOutput), newArrangementsCommand(jsonOutput), newKeysCommand(jsonOutput), newAttachmentsCommand(jsonOutput), newTagsCommand(jsonOutput), newCatalogCommand("tag-groups", jsonOutput), newCatalogCommand("attachment-types", jsonOutput))
	return c
}
func noArguments(c *cobra.Command, args []string) error {
	if len(args) > 0 {
		return usageError{fmt.Errorf("unexpected argument %q; run %s --help", args[0], c.CommandPath())}
	}
	return nil
}
func group(use, short string) *cobra.Command {
	return &cobra.Command{Use: use, Short: short, Args: noArguments, RunE: func(c *cobra.Command, _ []string) error { return c.Help() }}
}
func dataCommand(use, short string, n int, run func(*cobra.Command, []string) error) *cobra.Command {
	return &cobra.Command{Use: use, Short: short, Args: func(_ *cobra.Command, a []string) error {
		if len(a) != n {
			return usageError{fmt.Errorf("expected %d positional argument(s)", n)}
		}
		return nil
	}, RunE: func(c *cobra.Command, a []string) error {
		for _, value := range a {
			if _, _, _, e := services.ParseSelector(value); e != nil {
				return e
			}
		}
		for _, name := range []string{"song", "arrangement", "key"} {
			if f := c.Flags().Lookup(name); f != nil && f.Changed {
				if _, _, _, e := services.ParseSelector(f.Value.String()); e != nil {
					return e
				}
			}
		}
		return run(c, a)
	}}
}
func selection(c *cobra.Command, s *services.Scope, arr bool, key bool) {
	c.Flags().StringVar(&s.Song, "song", "", "song ID or name (required)")
	if arr || key {
		c.Flags().StringVar(&s.Arrangement, "arrangement", "", "arrangement ID or name")
	}
	if key {
		c.Flags().StringVar(&s.Key, "key", "", "key record ID or name; requires --arrangement")
	}
	visibility(c, &s.Visibility)
	c.Flags().BoolVar(&s.IncludeArchived, "include-archived", false, "include archived arrangements explicitly")
}
func visibility(c *cobra.Command, v *services.Visibility) {
	c.Flags().Var((*visibilityFlag)(v), "visibility", "select songs: visible, hidden, or all (default visible)")
}

type visibilityFlag services.Visibility

func (v *visibilityFlag) String() string {
	if *v == "" {
		return "visible"
	}
	return string(*v)
}
func (v *visibilityFlag) Type() string { return "visibility" }
func (v *visibilityFlag) Set(s string) error {
	switch s {
	case "visible", "hidden", "all":
		*v = visibilityFlag(s)
		return nil
	}
	return fmt.Errorf("expected visible, hidden, or all")
}
func checkScope(s services.Scope, arr bool) error {
	if strings.TrimSpace(s.Song) == "" {
		return usageError{fmt.Errorf("--song is required")}
	}
	if arr && s.Arrangement == "" {
		return usageError{fmt.Errorf("--arrangement is required")}
	}
	if s.Key != "" && s.Arrangement == "" {
		return usageError{fmt.Errorf("--key requires --arrangement")}
	}
	return nil
}
func listFlags(c *cobra.Command, o *services.ListOptions) {
	c.Flags().IntVar(&o.Limit, "limit", 25, "maximum number of results across API pages")
	c.Flags().BoolVar(&o.All, "all", false, "fetch the complete selected collection")
}
func listOptions(c *cobra.Command, o services.ListOptions) (services.ListOptions, error) {
	if o.Limit < 1 {
		return o, usageError{fmt.Errorf("--limit must be positive")}
	}
	if o.All {
		if c.Flags().Changed("limit") {
			return o, usageError{fmt.Errorf("--all and --limit are mutually exclusive")}
		}
		o.Limit = 0
	}
	return o, nil
}
func confirm(c *cobra.Command) error {
	yes, _ := c.Flags().GetBool("yes")
	if !yes {
		return usageError{fmt.Errorf("pass --yes to confirm deletion")}
	}
	return nil
}
func deleteFlag(c *cobra.Command) {
	c.Flags().Bool("yes", false, "confirm deletion; no interactive prompt")
}
