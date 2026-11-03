package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/joe--cool/pccli/internal/cli"
)

func main() {
	ctx, cancel := context.WithCancelCause(context.Background())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	go func() {
		select {
		case s := <-signals:
			cancel(signalError{s})
		case <-ctx.Done():
		}
	}()
	err := cli.ExecuteContext(ctx)
	code := cli.ExitCode(err)
	var term signalError
	if errors.As(context.Cause(ctx), &term) && term.signal == syscall.SIGTERM {
		code = 143
	}
	cancel(nil)
	if code != 0 {
		os.Exit(code)
	}
}

type signalError struct{ signal os.Signal }

func (e signalError) Error() string { return "interrupted" }
