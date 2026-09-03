// Package main is the entry point for the fibcalc CLI. It handles version/help
// flags, builds the application from config, and runs the main loop.
package main

import (
	"context"
	"io"
	"os"

	"github.com/agbruneau/FibGo/internal/app"
	apperrors "github.com/agbruneau/FibGo/internal/errors"
)

func main() {
	os.Exit(run(os.Args, os.Stdout, os.Stderr))
}

// run contains the core logic extracted from main for testability and returns
// the POSIX exit code (internal/errors.Exit*).
func run(args []string, stdout, stderr io.Writer) int {
	if app.HasVersionFlag(args[1:]) {
		app.PrintVersion(stdout)
		return apperrors.ExitSuccess
	}

	application, err := app.New(args, stderr)
	if err != nil {
		if app.IsHelpError(err) {
			return apperrors.ExitSuccess
		}
		// app.New only fails while parsing/validating configuration, so any
		// other error is a configuration error and must use the dedicated
		// ExitErrorConfig (4) contract, not the generic code 1.
		return apperrors.ExitErrorConfig
	}

	return application.Run(context.Background(), stdout)
}
