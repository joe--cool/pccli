package cli

import (
	"net/http"
	"os"

	"github.com/joe--cool/pccli/internal/config"
	"github.com/joe--cool/pccli/internal/planningcenter"
	"github.com/joe--cool/pccli/internal/services"
	"github.com/spf13/cobra"
)

type app struct {
	cfg     config.Config
	library *services.Library
}

var Version = "dev"

func Execute() error {
	cmd := NewRootCommand()
	err := cmd.Execute()
	if err != nil {
		printError(err)
	}
	return err
}

func NewRootCommand() *cobra.Command {
	var jsonOutput bool

	root := &cobra.Command{
		Use:     "pccli",
		Short:   "Planning Center command line tools",
		Long:    "pccli helps church teams work with Planning Center from the command line.",
		Version: Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	silenceCobra(root)
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
