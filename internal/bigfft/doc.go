// Package bigfft implements FFT-based multiplication of math/big Int values
// using the Schönhage-Strassen algorithm over Fermat rings.
//
// # Role
//
// This package is the large-multiplication backend used by the Fibonacci
// calculators once operand sizes exceed the FFT threshold (≈ 500k bits by
// default). It provides a faster-than-Karatsuba multiplier for very large
// integers, plus the supporting infrastructure: a pooled bump allocator, an
// LRU FFT plan cache, and Fermat ring arithmetic.
//
// # Invariants
//
//   - Mul and related entry points do not mutate their inputs.
//   - Temporary buffers (twiddle tables, coefficient slabs) are drawn from a
//     package-level sync.Pool and MUST be returned; callers using BumpAlloc
//     must call Reset before the next top-level operation.
//   - Element-wise word arithmetic (arith.go) delegates to math/big's internal
//     assembly via go:linkname (arith_decl.go) on every architecture math/big
//     itself supports in assembly.
//   - The FFT cache (fft_cache.go) is thread-safe (sync.Mutex + LRU).
//
// # Example
//
//	var a, b big.Int
//	// ... populate a, b with large values ...
//	dst, err := bigfft.Mul(&a, &b) // dst = a * b
//
// See docs/algorithms/FFT.md for the mathematical background.
package bigfft
