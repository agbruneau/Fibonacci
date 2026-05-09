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
//	func main() {
//	    a, err := app.New(os.Args[1:], os.Stderr)
//	    if err != nil {
//	        // handle wiring error
//	    }
//	    action := a.Run(context.Background(), os.Stdout)
//	    if action.ShouldExit() {
//	        os.Exit(action.Code())
//	    }
//	}
package app
