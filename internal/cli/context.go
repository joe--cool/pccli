package cli

import (
	"context"

	"github.com/spf13/cobra"
)

type appKey struct{}

func withApp(ctx context.Context, app *app) context.Context {
	return context.WithValue(ctx, appKey{}, app)
}

func appFrom(cmd *cobra.Command) *app {
	if cmd == nil || cmd.Context() == nil {
		return nil
	}
	value, _ := cmd.Context().Value(appKey{}).(*app)
	return value
}

func mustApp(cmd *cobra.Command) *app {
	app := appFrom(cmd.Root())
	if app == nil {
		panic("application was not initialized")
	}
	return app
}
