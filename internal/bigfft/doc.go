// Package bigfft implements FFT-based multiplication of math/big Int values
// using the Schönhage-Strassen algorithm over Fermat rings.
//
// # Role
//
// This package is the large-multiplication backend used by the Fibonacci
// calculators once operand sizes exceed the FFT threshold (≈ 500k bits by
// default). It provides a faster-than-Karatsuba multiplier for very large
// integers, plus the supporting infrastructure: a pooled bump allocator, an
// LRU FFT plan cache, Fermat ring arithmetic, and a SIMD-accelerated scan
// path on amd64.
//
// # Invariants
//
//   - Mul and related entry points do not mutate their inputs.
//   - Temporary buffers (twiddle tables, coefficient slabs) are drawn from a
//     package-level sync.Pool and MUST be returned; callers using BumpAlloc
//     must call Reset before the next top-level operation.
//   - The amd64 assembly variants (arith_amd64.go, cpu_amd64.go) are guarded
//     by a CPU-feature check; the pure-Go fallback in arith_generic.go is
//     always correct but slower.
//   - The FFT cache (fft_cache.go) is thread-safe (sync.Mutex + LRU).
//
// # Example
//
//	var a, b, dst big.Int
//	// ... populate a, b with large values ...
//	bigfft.Mul(&dst, &a, &b) // dst = a * b
//
// See docs/algorithms/FFT.md for the mathematical background.
package bigfft
