// Package bigfft implements multiplication of big.Int using FFT.
// This file provides a bump allocator for fast temporary allocations during FFT operations.
package bigfft

import (
	"math/big"
	"sync"
)

// ─────────────────────────────────────────────────────────────────────────────
// Bump Allocator
// ─────────────────────────────────────────────────────────────────────────────

// BumpAllocator provides fast O(1) allocation from a pre-allocated buffer.
// It's designed for temporary allocations during FFT operations where many
// small allocations are needed and can all be released together at the end.
//
// Benefits over sync.Pool for temporary allocations:
//   - O(1) allocation (just pointer bump) vs mutex + map lookup
//   - Zero fragmentation (contiguous memory)
//   - Excellent cache locality
//   - Single release operation for all allocations
//
// Thread Safety: BumpAllocator is NOT thread-safe. Each goroutine should
// have its own instance. Use AcquireBumpAllocator/ReleaseBumpAllocator
// which manage per-goroutine instances via sync.Pool.
type BumpAllocator struct {
	buffer []big.Word
	offset int
}

// bumpAllocatorPool pools BumpAllocator instances for reuse.
// The underlying buffer is retained between uses to avoid re-allocation.
var bumpAllocatorPool = sync.Pool{
	New: func() any {
		return &BumpAllocator{}
	},
}

// AcquireBumpAllocator gets a bump allocator with at least the specified capacity.
// The allocator should be released with ReleaseBumpAllocator when done.
//
// The returned allocator must be released using ReleaseBumpAllocator, preferably with defer:
//
//	allocator := AcquireBumpAllocator(capacity)
//	defer ReleaseBumpAllocator(allocator)
//
// This ensures the allocator is returned to the pool even if an error occurs or a panic is triggered.
//
// Parameters:
//   - capacity: Minimum number of big.Word elements needed for all allocations.
//
// Returns:
//   - *BumpAllocator: A ready-to-use bump allocator.
func AcquireBumpAllocator(capacity int) *BumpAllocator {
	ba := bumpAllocatorPool.Get().(*BumpAllocator)

	// Grow buffer if needed, otherwise reuse existing
	if cap(ba.buffer) < capacity {
		ba.buffer = make([]big.Word, capacity)
	} else {
		ba.buffer = ba.buffer[:capacity]
	}

	// Reset offset for new allocation session
	ba.offset = 0

	return ba
}

// ReleaseBumpAllocator returns a bump allocator to the pool for reuse.
// After calling this, the allocator and any slices allocated from it
// should not be used.
//
// This should be called with defer immediately after AcquireBumpAllocator to ensure
// proper resource cleanup even in case of errors or panics:
//
//	allocator := AcquireBumpAllocator(capacity)
//	defer ReleaseBumpAllocator(allocator)
//
// Parameters:
//   - ba: The bump allocator to release. Safe to call with nil.
func ReleaseBumpAllocator(ba *BumpAllocator) {
	if ba == nil {
		return
	}
	// Reset offset but keep buffer for reuse
	ba.offset = 0
	bumpAllocatorPool.Put(ba)
}

// Alloc allocates n words from the bump allocator.
// Returns a zeroed slice of exactly n words.
//
// If the allocation would exceed the buffer capacity, falls back to
// a regular make() allocation. This ensures correctness even if the
// capacity estimate was too small.
//
// Parameters:
//   - n: Number of big.Word elements to allocate.
//
// Returns:
//   - []big.Word: A zeroed slice of n words.
func (ba *BumpAllocator) Alloc(n int) []big.Word {
	if ba.offset+n > len(ba.buffer) {
		// Fallback: capacity exceeded, allocate directly
		return make([]big.Word, n)
	}

	slice := ba.buffer[ba.offset : ba.offset+n]
	ba.offset += n

	// Zero the slice for safety
	for i := range slice {
		slice[i] = 0
	}

	return slice
}

// AllocUnsafe allocates n words without zeroing.
// Use only when you're certain the caller will overwrite all values.
//
// Parameters:
//   - n: Number of big.Word elements to allocate.
//
// Returns:
//   - []big.Word: A slice of n words (may contain stale data).
func (ba *BumpAllocator) AllocUnsafe(n int) []big.Word {
	if ba.offset+n > len(ba.buffer) {
		return make([]big.Word, n)
	}

	slice := ba.buffer[ba.offset : ba.offset+n]
	ba.offset += n
	return slice
}

// AllocFermat allocates a fermat number buffer of the given size.
// Fermat numbers need n+1 words (the extra word for overflow handling).
//
// Parameters:
//   - n: The n parameter for fermat (resulting slice has n+1 elements).
//
// Returns:
//   - fermat: A zeroed fermat slice.
func (ba *BumpAllocator) AllocFermat(n int) fermat {
	return fermat(ba.Alloc(n + 1))
}

// AllocFermatSlice allocates K fermat numbers, each of size n+1.
// Returns both the slice of fermat references and the backing word buffer.
//
// This is optimized for FFT where we need K coefficient buffers that
// are accessed sequentially, benefiting from cache locality.
//
// Parameters:
//   - K: Number of fermat slices to allocate.
//   - n: The n parameter for each fermat (each slice has n+1 elements).
//
// Returns:
//   - []fermat: Slice of K fermat references.
//   - []big.Word: The backing buffer (for potential release tracking).
func (ba *BumpAllocator) AllocFermatSlice(K, n int) ([]fermat, []big.Word) {
	wordCount := K * (n + 1)
	bits := ba.Alloc(wordCount)

	// Slice headers are small, regular allocation is fine
	fermats := make([]fermat, K)
	for i := 0; i < K; i++ {
		fermats[i] = fermat(bits[i*(n+1) : (i+1)*(n+1)])
	}

	return fermats, bits
}

// Remaining returns the number of words still available in the buffer.
func (ba *BumpAllocator) Remaining() int {
	return len(ba.buffer) - ba.offset
}

// Used returns the number of words that have been allocated.
func (ba *BumpAllocator) Used() int {
	return ba.offset
}

// Reset resets the allocator to the beginning of the buffer.
// All previous allocations become invalid.
// This is useful for reusing the same allocator for multiple phases.
func (ba *BumpAllocator) Reset() {
	ba.offset = 0
}

// ─────────────────────────────────────────────────────────────────────────────
// Capacity Estimation
// ─────────────────────────────────────────────────────────────────────────────

// EstimateBumpCapacity estimates the bump allocator capacity needed for
// FFT operations on numbers of the given word length.
//
// This is a heuristic based on the FFT algorithm's memory access patterns:
//   - Transform needs ~2 * K * (n+1) words for input/temp buffers
//   - Inverse transform needs similar
//   - Multiply/Sqr needs temp buffer of ~8n words
//
// Parameters:
//   - wordLen: Number of words in the numbers being multiplied.
//
// Returns:
//   - int: Estimated capacity in words.
func EstimateBumpCapacity(wordLen int) int {
	if wordLen <= 0 {
		return 0
	}

	// Estimate K (FFT size) - roughly 2*sqrt(bits)
	bits := wordLen * _W
	k := uint(0)
	for i, thresh := range fftSizeThreshold {
		if int64(bits) < thresh {
			k = uint(i)
			break
		}
	}
	if k == 0 {
		k = uint(len(fftSizeThreshold) - 1)
	}

	K := 1 << k
	// n is roughly wordLen / K
	n := wordLen/K + 1

	// Estimate total: 2 transforms worth of temp buffers + multiply buffer
	// Transform temp: K * (n+1) words
	// Multiply temp: 8 * n words
	transformTemp := K * (n + 1)
	multiplyTemp := 8 * n

	// Add 20% safety margin
	total := (2*transformTemp + multiplyTemp) * 12 / 10

	return total
}
