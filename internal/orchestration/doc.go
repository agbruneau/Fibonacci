// Package orchestration coordinates concurrent execution of Fibonacci
// calculators and aggregates their results into a uniform comparison
// surface. It is the architectural seam between the domain layer
// (internal/fibonacci, internal/bigfft) and the presentation layers
// (internal/cli, internal/tui).
//
// Key types:
//
//   - ExecuteCalculations: errgroup-based parallel dispatcher. Each
//     selected calculator runs in its own goroutine (one per calculator,
//     a small fixed set — the errgroup has no SetLimit). Results
//     flow back through a progress channel that is closed and drained
//     deterministically.
//   - CalculationResult: the uniform domain DTO returned to the
//     presentation layer (Name, *big.Int Result, Duration, Err).
//   - ProgressReporter / ResultPresenter / ErrorHandler: the three
//     interfaces a presentation layer implements. Both internal/cli and
//     internal/tui satisfy them.
//
// Re-exported aliases (Audit-PRD E3-R3 / Sprint S3-T4):
//
//	type Calculator = fibonacci.Calculator
//	type Options    = fibonacci.Options
//
// Presentation packages consume the aliases to avoid a direct
// internal/fibonacci import. The arch_test.go gate keeps internal/tui
// free of that direct arrow. (The former Default*Threshold mirrors had
// zero consumers and were removed — audit Fable5 ARCH-04.)
//
// Dependencies (downward):
//
//	orchestration → fibonacci, progress, errors
package orchestration
