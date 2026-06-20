package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type exitCoder interface {
	ExitCode() int
}

type usageError struct {
	err error
}

func (e usageError) Error() string {
	return e.err.Error()
}

func (e usageError) Unwrap() error {
	return e.err
}

func (e usageError) ExitCode() int {
	return 2
}

func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var coder exitCoder
	if errors.As(err, &coder) {
		return coder.ExitCode()
	}
	return 1
}

func silenceCobra(cmd *cobra.Command) {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
}

func printError(err error) {
	if err == nil {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
}
