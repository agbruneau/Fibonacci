// Package main is the entry point of the fibcalc application.
// It provides a minimal bootstrap that delegates to the app package
// for configuration parsing and command execution.
package main

import (
	"context"
	"io"
	"os"

	"github.com/agbru/fibcalc/internal/app"
	apperrors "github.com/agbru/fibcalc/internal/errors"
)

func main() {
	os.Exit(run(os.Args, os.Stdout, os.Stderr))
}

// run executes the application and returns the appropriate exit code.
// This function is separated from main() to enable comprehensive testing
// without relying on os.Exit behavior.
func run(args []string, stdout, stderr io.Writer) int {
	// Handle version flag early (before config parsing)
	if len(args) > 1 && app.HasVersionFlag(args[1:]) {
		app.PrintVersion(stdout)
		return apperrors.ExitSuccess
	}

	// Create and configure the application
	application, err := app.New(args, stderr)
	if err != nil {
		if app.IsHelpError(err) {
			return apperrors.ExitSuccess
		}
		return apperrors.ExitErrorConfig
	}

	// Run the application
	return application.Run(context.Background(), stdout)
}
