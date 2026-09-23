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
// # Provenance
//
// The FFT multiplication core is derived from github.com/remyoudompheng/bigfft
// by Rémy Oudompheng, upstream commit 24d4a6f8daec (2023-01-29), distributed
// under the BSD 3-Clause license with the notice "Copyright (c) 2012 The Go
// Authors". That license is kept verbatim in LICENSE in this directory, and
// its conditions still apply to the upstream portions. Six files derive from
// upstream, each opening with a header that lists what it took and what
// changed:
//
//   - arith_decl.go from upstream arith_decl.go
//   - fermat.go from upstream fermat.go
//   - fft.go, fft_core.go, fft_poly.go and fft_recursion.go, split out of
//     upstream fft.go
//
// Every other file, tests included, is original to this repository. The
// modifications and the original files are Copyright 2026 André-Guy Bruneau
// and licensed under Apache-2.0 (see the repository's LICENSE and NOTICE).
// docs/algorithms/BIGFFT.md § Provenance maps each derived file to the
// upstream code it came from.
//
// See docs/algorithms/FFT.md for the mathematical background.
package bigfft
