// The main package is the entry point of the fibcalc application.
// It provides a minimal bootstrap that delegates to the app package
// for configuration parsing and command execution.
package main

import (
	"context"
	"os"

	"github.com/agbru/fibcalc/internal/app"
	apperrors "github.com/agbru/fibcalc/internal/errors"
)

func main() {
	// Check for version flag in any position before parsing config
	if app.HasVersionFlag(os.Args[1:]) {
		app.PrintVersion(os.Stdout)
		os.Exit(apperrors.ExitSuccess)
	}

	// Create and configure the application
	application, err := app.New(os.Args, os.Stderr)
	if err != nil {
		if app.IsHelpError(err) {
			os.Exit(apperrors.ExitSuccess)
		}
		os.Exit(apperrors.ExitErrorConfig)
	}

	// Run the application and exit with the returned code
	exitCode := application.Run(context.Background(), os.Stdout)
	os.Exit(exitCode)
}
