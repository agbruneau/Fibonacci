// Package metrics exposes lightweight performance and memory indicators
// derived from Fibonacci computations.
//
// It provides two families of helpers:
//
//   - Indicators (indicators.go): throughput (bits/s, digits/s), doubling-step
//     count, and cheap mathematical properties (e.g. parity via n%3) computed
//     in O(1) or with only BitLen-class big.Int operations. A "live" variant
//     estimates values from a progress fraction so the TUI can display
//     meaningful numbers before the calculation completes.
//   - Memory (memory.go): a thin wrapper around runtime.MemStats that produces
//     point-in-time MemorySnapshot values (HeapAlloc, HeapSys, NumGC, ...)
//     used by the TUI dashboard and by diagnostic output.
//
// The package has no external dependencies beyond the standard library and is
// safe to call from any goroutine.
package metrics
