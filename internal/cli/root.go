package cli

import (
	"context"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/joe--cool/pccli/internal/config"
	"github.com/joe--cool/pccli/internal/planningcenter"
	"github.com/joe--cool/pccli/internal/services"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type app struct {
	cfg     config.Config
	library *services.Library
}

var Version = "dev"

func Execute() error { return ExecuteContext(context.Background()) }
func ExecuteContext(ctx context.Context) error {
	return ExecuteArgs(ctx, os.Args[1:], os.Stdout, os.Stderr)
}
func ExecuteArgs(ctx context.Context, args []string, out, stderr io.Writer) error {
	return executeCommand(ctx, NewRootCommand(), args, out, stderr)
}
func executeCommand(ctx context.Context, cmd *cobra.Command, args []string, out, stderr io.Writer) error {
	cmd.SetArgs(args)
	cmd.SetOut(out)
	cmd.SetErr(stderr)
	err := cmd.ExecuteContext(ctx)
	if err != nil && strings.HasPrefix(err.Error(), "unknown command ") {
		err = usageError{err}
	}
	if err != nil {
		printError(stderr, jsonRequested(args), err)
	}
	return err
}

// Error formatting must still honor --json when another flag is invalid.
// Inspect known flag arity so a value literally named --json is not a flag.
// Command parsing and validation remain entirely in Cobra/pflag.
func jsonRequested(args []string) bool {
	root := NewRootCommand()
	target, _, _ := root.Find(args)
	if target == nil {
		target = root
	}
	value := false
	for i := 0; i < len(args); i++ {
		token := args[i]
		if token == "--" {
			break
		}
		if token == "--json" {
			value = true
			continue
		}
		if raw, ok := strings.CutPrefix(token, "--json="); ok {
			if parsed, e := strconv.ParseBool(raw); e == nil {
				value = parsed
			}
			continue
		}
		var f *pflag.Flag
		if strings.HasPrefix(token, "--") {
			name, _, attached := strings.Cut(strings.TrimPrefix(token, "--"), "=")
			if attached {
				continue
			}
			f = target.Flags().Lookup(name)
			if f == nil {
				f = target.InheritedFlags().Lookup(name)
			}
		} else if strings.HasPrefix(token, "-") && len(token) == 2 {
			f = target.Flags().ShorthandLookup(token[1:])
		}
		if f != nil && f.NoOptDefVal == "" {
			i++
		}
	}
	return value
}

func NewRootCommand() *cobra.Command {
	var jsonOutput bool

	root := &cobra.Command{
		Use:           "pccli",
		Short:         "Planning Center command line tools",
		Long:          "pccli helps church teams work with Planning Center from the command line.",
		Version:       Version,
		Args:          noArguments,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	root.SetFlagErrorFunc(func(_ *cobra.Command, e error) error { return usageError{e} })
	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "write machine-readable JSON")
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)

	defaultHelp := root.HelpFunc()
	root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd == root && !jsonOutput {
			printBanner(cmd.OutOrStdout())
		}
		defaultHelp(cmd, args)
	})

	root.AddGroup(&cobra.Group{ID: "products", Title: "Planning Center Products"})
	root.AddCommand(newServicesCommand(&jsonOutput))
	return root
}

func loadApp(cmd *cobra.Command) (*app, error) {
	if existing := appFrom(cmd.Root()); existing != nil {
		return existing, nil
	}
	app, err := buildApp()
	if err != nil {
		return nil, err
	}
	cmd.Root().SetContext(withApp(cmd.Root().Context(), app))
	return app, nil
}

func buildApp() (*app, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	var transport http.RoundTripper
	if cfg.Mock {
		transport, err = planningcenter.NewMockTransport(cfg.MockFixture)
		if err != nil {
			return nil, err
		}
	}

	client := planningcenter.NewClient(cfg, transport)
	return &app{
		cfg:     cfg,
		library: services.NewLibrary(client),
	}, nil
}
