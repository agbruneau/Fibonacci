// Package bigfft implements FFT-based multiplication of math/big Int values
// using the Schönhage-Strassen algorithm over Fermat rings.
//
// # Role
//
// This package is the large-multiplication backend used by the Fibonacci
// calculators once operand sizes exceed the FFT threshold (≈ 500k bits by
// default). It provides a faster-than-Karatsuba multiplier for very large
// integers, plus the supporting infrastructure: a pooled bump allocator, an
// LRU FFT transform-result cache, and Fermat ring arithmetic.
//
// # Invariants
//
//   - Mul and related entry points do not mutate their inputs.
//   - Temporary buffers (twiddle tables, coefficient slabs) are drawn from
//     package-level sync.Pool buckets (pool.go) and MUST be returned via the
//     matching release function; the per-goroutine BumpAllocator (bump.go)
//     is an O(1) alternative acquired via AcquireBumpAllocator and returned
//     via ReleaseBumpAllocator, which resets it for reuse. Reset is exposed
//     separately for reusing one allocator instance across multiple phases
//     without a full release/acquire round-trip.
//   - Element-wise word arithmetic (arith.go) delegates unconditionally to
//     math/big's internal assembly via go:linkname (arith_decl.go); there is
//     no separate AVX2/pure-Go build-tag split.
//   - The FFT transform-result cache (fft_cache.go) is thread-safe
//     (sync.RWMutex + LRU); it caches computed PolValues keyed by an FNV-1a
//     hash of the input, not precomputed FFT plans.
//
// # Example
//
//	var a, b big.Int
//	// ... populate a, b with large values ...
//	dst, err := bigfft.Mul(&a, &b) // dst = a * b
//
// See docs/algorithms/FFT.md for the mathematical background.
package bigfft
