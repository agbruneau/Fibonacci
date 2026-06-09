// Package metrics exposes lightweight performance indicators derived from
// Fibonacci computations.
//
// Indicators (indicators.go) cover throughput (bits/s, digits/s),
// doubling-step count, and cheap mathematical properties (e.g. parity via
// n%3) computed in O(1) or with only BitLen-class big.Int operations. A
// "live" variant estimates values from a progress fraction so the TUI can
// display meaningful numbers before the calculation completes.
//
// The package has no external dependencies beyond the standard library and is
// safe to call from any goroutine.
package metrics
