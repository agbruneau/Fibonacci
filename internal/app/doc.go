// Package app wires the fibcalc command-line interface into a runnable
// application.
//
// # Role
//
// It owns the top-level lifecycle: parse CLI arguments and environment
// variables (via internal/config), resolve the calculator factory, execute
// the requested sub-command (calculate, calibrate, version, help), and map
// errors to process exit codes. It is the bridge between cmd/fibcalc (the
// package main entry point) and the internal domain packages.
//
// # Invariants
//
//   - New() is the only constructor; it validates the argv slice and never
//     writes to os.Stdout/os.Stderr directly — all diagnostic output flows
//     through the errWriter passed in by the caller.
//   - The Application struct is single-use: call Run once, then discard.
//   - Help and version requests are surfaced as typed errors (see
//     IsHelpError) so callers can distinguish them from real failures and
//     map them to exit code 0.
//
// # Example
//
// This mirrors cmd/fibcalc/main.go and must be kept in step with it (audit
// API-08). Run returns the POSIX exit code directly (internal/apperrors.Exit*).
//
//	func main() {
//	    os.Exit(run(os.Args, os.Stdout, os.Stderr))
//	}
//
//	func run(args []string, stdout, stderr io.Writer) int {
//	    application, err := app.New(args, stderr)
//	    if err != nil {
//	        if app.IsHelpError(err) {
//	            return apperrors.ExitSuccess
//	        }
//	        return apperrors.ExitErrorConfig
//	    }
//	    return application.Run(context.Background(), stdout)
//	}
package app
