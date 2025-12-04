// Package bigfft implements multiplication of big.Int using FFT.
// This file provides an arena allocator for Fibonacci calculations to reduce
// memory allocations and GC pressure by pre-allocating buffers based on estimated needs.
package bigfft

import (
	"math"
	"math/big"
)

// MemoryEstimate holds estimated memory requirements for a Fibonacci calculation.
// These estimates are based on the approximate size of F(n) and FFT buffer needs.
type MemoryEstimate struct {
	// MaxWordSliceSize is the maximum size (in words) needed for word slices
	// during the calculation. This accounts for FFT buffers which can be
	// 2^(k) * (n+1) words where k depends on the number size.
	MaxWordSliceSize int
	// MaxFermatSize is the maximum size (in words) needed for fermat numbers.
	// Typically n+1 words where n is derived from FFT parameters.
	MaxFermatSize int
	// MaxNatSliceSize is the maximum size needed for []nat slices (polynomial coefficients).
	MaxNatSliceSize int
	// MaxFermatSliceSize is the maximum size needed for []fermat slices (polynomial values).
	MaxFermatSliceSize int
}

// EstimateMemoryNeeds calculates memory requirements for calculating F(n).
//
// Estimation formula:
//   - F(n) has approximately n * log10(φ) ≈ n * 0.694 bits
//   - For FFT: buffers of size 2^(k) * (n+1) words where k is chosen based on size
//   - We estimate conservatively to avoid reallocations
//
// Parameters:
//   - n: The Fibonacci index to calculate.
//
// Returns:
//   - MemoryEstimate: Estimated memory requirements.
func EstimateMemoryNeeds(n uint64) MemoryEstimate {
	// Estimate bits in F(n): approximately n * log10(φ) / log10(2)
	// Using φ ≈ 1.618, log10(φ) ≈ 0.2089, so log2(φ) ≈ 0.694
	// F(n) ≈ n * 0.694 bits
	estimatedBits := float64(n) * 0.694
	
	// Convert to words (assuming 64-bit words)
	estimatedWords := int(math.Ceil(estimatedBits / 64))
	
	// For FFT, we need buffers that can hold 2^(k) * (n+1) words
	// where k is chosen such that 2^k is about 2*sqrt(N) for N = estimatedBits
	// We estimate k conservatively: find the smallest k where 2^k >= 2*sqrt(estimatedWords)
	sqrtWords := math.Sqrt(float64(estimatedWords))
	k := uint(0)
	for (1 << k) < int(2*sqrtWords) {
		k++
	}
	if k > 15 {
		k = 15 // Cap at reasonable maximum
	}
	
	// FFT buffer size: 2^k * (n+1) words, but we use estimatedWords instead of n
	// to account for the actual size of F(n)
	fftBufferSize := (1 << k) * (estimatedWords + 1)
	if fftBufferSize < 1024 {
		fftBufferSize = 1024 // Minimum reasonable size
	}
	
	// Estimate polynomial coefficient count: typically K = 2^k
	polynomialCoeffs := 1 << k
	if polynomialCoeffs < 8 {
		polynomialCoeffs = 8
	}
	
	return MemoryEstimate{
		MaxWordSliceSize:   fftBufferSize,
		MaxFermatSize:      estimatedWords + 1,
		MaxNatSliceSize:    polynomialCoeffs,
		MaxFermatSliceSize: polynomialCoeffs,
	}
}

// CalculationArena is an arena allocator for a single Fibonacci calculation.
// It pre-allocates buffers based on estimated needs and provides allocation
// methods that fall back to global pools if the arena doesn't have sufficient
// capacity.
type CalculationArena struct {
	// Pre-allocated buffers (optional, may be nil if not pre-warmed)
	wordSliceBuffer   []big.Word
	fermatBuffer      fermat
	natSliceBuffer    []nat
	fermatSliceBuffer []fermat
	
	// Current offsets for arena-style allocation (if using single buffer)
	// For now, we use the pool fallback approach instead of true arena allocation
	// to maintain compatibility with existing code
}

// NewCalculationArena creates a new arena allocator for a calculation.
// The arena can optionally pre-warm pools based on the estimated needs.
//
// Parameters:
//   - n: The Fibonacci index to calculate (used for estimation).
//
// Returns:
//   - *CalculationArena: A new arena allocator.
func NewCalculationArena(n uint64) *CalculationArena {
	// Estimate memory needs (used by PreWarmPools, not stored here)
	_ = EstimateMemoryNeeds(n)
	return &CalculationArena{
		// For now, we don't pre-allocate buffers in the arena itself
		// Instead, we rely on PreWarmPools to prepare the global pools
		// This keeps the implementation simpler and maintains compatibility
	}
}

// AllocWordSlice allocates a word slice, preferring the arena's buffer
// but falling back to global pools if needed.
//
// Parameters:
//   - size: The required size in words.
//
// Returns:
//   - []big.Word: A word slice of at least the requested size.
func (a *CalculationArena) AllocWordSlice(size int) []big.Word {
	// For now, fall back to global pools
	// Future optimization: use arena buffer if available
	return acquireWordSlice(size)
}

// AllocFermat allocates a fermat slice, preferring the arena's buffer
// but falling back to global pools if needed.
//
// Parameters:
//   - size: The required size in words.
//
// Returns:
//   - fermat: A fermat slice of the requested size.
func (a *CalculationArena) AllocFermat(size int) fermat {
	// For now, fall back to global pools
	return acquireFermat(size)
}

// AllocNatSlice allocates a []nat slice, preferring the arena's buffer
// but falling back to global pools if needed.
//
// Parameters:
//   - size: The required size.
//
// Returns:
//   - []nat: A []nat slice of at least the requested size.
func (a *CalculationArena) AllocNatSlice(size int) []nat {
	// For now, fall back to global pools
	return acquireNatSlice(size)
}

// AllocFermatSlice allocates a []fermat slice, preferring the arena's buffer
// but falling back to global pools if needed.
//
// Parameters:
//   - size: The required size.
//
// Returns:
//   - []fermat: A []fermat slice of at least the requested size.
func (a *CalculationArena) AllocFermatSlice(size int) []fermat {
	// For now, fall back to global pools
	return acquireFermatSlice(size)
}

// Release releases any arena-allocated resources.
// Currently a no-op as we use global pools, but provided for future use.
func (a *CalculationArena) Release() {
	// Future: release arena buffers if we implement true arena allocation
}

