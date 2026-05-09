// Package main is the entry point for the fibcalc CLI. It handles version/help
// flags, builds the application from config, and runs the main loop.
package main

import (
	"context"
	"io"
	"os"

	"github.com/agbru/fibcalc/internal/app"
)

// main parses arguments, builds the application, and runs it; the exit
// code reflects success, help, or error.
func main() {
	action := run(os.Args, os.Stdout, os.Stderr)
	if action.ShouldExit() {
		os.Exit(action.Code())
	}
}

// run contains the core logic extracted from main for testability.
// It returns an app.ExitAction describing the outcome: ActionSuccess
// for success, ActionError (or one of the more specific Action* values)
// for failures, or ActionVersionHandled when --version was handled (in
// which case main MUST NOT call os.Exit).
func run(args []string, stdout, stderr io.Writer) app.ExitAction {
	if app.HasVersionFlag(args[1:]) {
		app.PrintVersion(stdout)
		return app.ActionVersionHandled
	}

	application, err := app.New(args, stderr)
	if err != nil {
		if app.IsHelpError(err) {
			return app.ActionSuccess
		}
		return app.ActionError
	}

	return application.Run(context.Background(), stdout)
}
