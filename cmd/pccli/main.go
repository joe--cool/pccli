package main

import (
	"os"

	"github.com/joe--cool/pccli/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(cli.ExitCode(err))
	}
}
