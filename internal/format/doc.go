// Package format provides presentation helpers shared across the CLI and TUI
// layers. It covers three concerns:
//
//   - Duration formatting tuned for short (µs/ms) to long intervals, with
//     human-readable units.
//   - Numeric string formatting with thousand separators, optimized to
//     minimize allocations on very long Fibonacci results.
//   - ETA and progress-bar string rendering (FormatETA, ProgressBar,
//     FormatProgressBarWithETA) from primitive progress/duration values.
//
// Progress aggregation across concurrent calculators (ProgressState,
// ProgressAggregator) lives in internal/orchestration, not here: this
// package contains no I/O and no aggregation state, it is a pure
// formatting layer that accepts primitive inputs and returns strings.
package format
