// Package testutil provides shared helpers used by tests across the
// fibcalc codebase.
//
// # Role
//
// It is a test-only support package. The helpers here avoid importing
// third-party assertion libraries and stay narrow in scope: sanitizing CLI
// output, comparing byte streams, and related plumbing that would
// otherwise be duplicated across test files. Production packages MUST NOT
// import testutil.
//
// # Invariants
//
//   - Helpers are pure functions (no package-level state, no side effects).
//   - Regular expressions and other expensive values are compiled at
//     package init via MustCompile so that test callers can assume cheap
//     per-call usage.
//
// # Example
//
//	got := testutil.StripAnsiCodes(captured)
//	if got != want {
//	    t.Errorf("output mismatch:\n got: %q\nwant: %q", got, want)
//	}
package testutil
