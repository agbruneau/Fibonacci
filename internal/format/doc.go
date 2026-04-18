// Package format provides presentation helpers shared across the CLI and TUI
// layers. It covers three concerns:
//
//   - Duration formatting tuned for short (µs/ms) to long intervals, with
//     human-readable units.
//   - Numeric string formatting with thousand separators, optimized to
//     minimize allocations on very long Fibonacci results.
//   - Progress/ETA aggregation for concurrent calculators, computing the
//     consolidated view displayed to the user.
//
// The package contains no I/O and no algorithmic code; it is a pure
// formatting layer that accepts primitive inputs and returns strings.
package format
