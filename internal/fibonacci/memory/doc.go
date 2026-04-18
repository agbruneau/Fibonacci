// Package memory groups the low-level memory-management primitives used by
// the Fibonacci calculators to keep allocations and GC pressure under control
// during large-n computations.
//
// Three concerns are covered:
//
//   - Arena (arena.go): a bump-pointer allocator that pre-sizes a single
//     contiguous []big.Word slab for all big.Int temporaries of one
//     calculation, enabling O(1) bulk release via Reset.
//   - Budget (budget.go): static memory-usage estimation for F(n) plus
//     human-readable parsing of memory limits ("8G", "512M", ...) used by
//     the CLI.
//   - GC control (gc_control.go): toggles GOGC and debug.SetGCPercent in
//     coordinated modes (auto / aggressive / disabled) so that long
//     calculations can suspend the collector rather than fight it.
//
// The package is intentionally kept free of any Fibonacci-specific logic so
// that it can be reused from every calculator (Fast Doubling, Matrix, FFT,
// GMP).
package memory
