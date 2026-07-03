// This file provides memory pooling for FFT operations to reduce GC pressure.

package bigfft

import (
	"math/big"
	"math/bits"
	"sync"
	"sync/atomic"
)

// ─────────────────────────────────────────────────────────────────────────────
// Word Slice Pools
// ─────────────────────────────────────────────────────────────────────────────

// wordSlicePool pools []big.Word slices by size class.
// We use size classes to avoid fragmentation: 64, 256, 1K, 4K, 16K, 64K, 256K, 1M, 4M, 16M words.
// Extended size classes support very large Fibonacci calculations (F > 10M).
var wordSlicePools = [...]sync.Pool{
	{New: func() any { return make([]big.Word, 64) }},
	{New: func() any { return make([]big.Word, 256) }},
	{New: func() any { return make([]big.Word, 1024) }},
	{New: func() any { return make([]big.Word, 4096) }},
	{New: func() any { return make([]big.Word, 16384) }},
	{New: func() any { return make([]big.Word, 65536) }},
	{New: func() any { return make([]big.Word, 262144) }},
	{New: func() any { return make([]big.Word, 1048576) }},  // 1M words = 8MB on 64-bit
	{New: func() any { return make([]big.Word, 4194304) }},  // 4M words = 32MB on 64-bit
	{New: func() any { return make([]big.Word, 16777216) }}, // 16M words = 128MB on 64-bit
}

// wordSliceSizes defines the size classes for word slice pools.
var wordSliceSizes = [...]int{64, 256, 1024, 4096, 16384, 65536, 262144, 1048576, 4194304, 16777216}

// getWordSlicePoolIndex returns the pool index for a given size.
// Returns -1 if the size is too large for pooling.
//
// Uses O(1) bitwise computation instead of linear search.
// wordSliceSizes are powers of 4 starting from 4^3 = 64:
// index i corresponds to size 4^(i+3), so bits.Len(size) maps directly to the index.
func getWordSlicePoolIndex(size int) int {
	if size <= 0 {
		return 0
	}
	if size > wordSliceSizes[len(wordSliceSizes)-1] {
		return -1
	}
	// bits.Len(uint(size-1)) gives the number of bits needed to represent size-1.
	// For size <= 64 (4^3), this is <= 6, yielding idx <= 0 after the formula.
	// For size <= 256 (4^4), bits.Len is <= 8, yielding idx = 1. Etc.
	// size > 0 by guard above, so size-1 >= 0, uint cast is safe. #nosec G115
	idx := (bits.Len(uint(size-1)) - 5) / 2
	if idx < 0 {
		idx = 0
	}
	return idx
}

// getWordSlicePoolIndexLinear is the original O(n) linear search implementation,
// kept as a reference for testing the optimized bitwise version.
func getWordSlicePoolIndexLinear(size int) int {
	for i, s := range wordSliceSizes {
		if size <= s {
			return i
		}
	}
	return -1
}

// acquireWordSlice gets a word slice of at least the given size from the pool.
// The returned slice may be larger than requested.
// If the size is too large for pooling, a new slice is allocated.
//
// The returned slice should be released using releaseWordSlice, preferably with defer:
//
//	slice := acquireWordSlice(size)
//	defer releaseWordSlice(slice)
//
// This ensures the slice is returned to the pool even if an error occurs.
func acquireWordSlice(size int) []big.Word {
	idx := getWordSlicePoolIndex(size)
	if idx < 0 {
		// Too large for pooling, allocate directly
		return make([]big.Word, size)
	}
	slice := wordSlicePools[idx].Get().([]big.Word)
	// Clear the slice before returning (Go 1.21+ built-in)
	clear(slice)
	return slice[:size]
}

// acquireWordSliceUnsafe returns a word slice from the pool without clearing it.
// Use this only when the caller will immediately overwrite all elements (e.g., via copy).
func acquireWordSliceUnsafe(size int) []big.Word {
	idx := getWordSlicePoolIndex(size)
	if idx < 0 {
		return make([]big.Word, size)
	}
	slice := wordSlicePools[idx].Get().([]big.Word)
	return slice[:size]
}

// wordSlicePoolMissCount counts release calls whose slice capacity does not
// map to any exact bucket size. A non-zero value usually means a caller
// reshaped a pooled slice (e.g. three-index slicing s[:n:n] or append-driven
// reallocation) before releasing it, so the underlying buffer cannot be
// returned to a pool and falls back to GC. Read with WordSlicePoolMissCount.
var wordSlicePoolMissCount atomic.Uint64

// WordSlicePoolMissCount returns the cumulative number of releaseWordSlice
// calls that could not return their slice to a pool because its capacity did
// not match any bucket size exactly. Intended for observability: a steadily
// growing value indicates pool churn that defeats the GC pressure savings.
func WordSlicePoolMissCount() uint64 {
	return wordSlicePoolMissCount.Load()
}

// releaseWordSlice returns a word slice to the pool.
// The slice must have been obtained from acquireWordSlice or
// acquireWordSliceUnsafe. This should be called with defer immediately after
// acquisition:
//
//	slice := acquireWordSlice(size)
//	defer releaseWordSlice(slice)
//
// Routing rule: the bucket is selected from the slice's *real* capacity, not
// its length. A pooled slice that was re-sliced down (e.g. `s[:n]`) keeps the
// same underlying array and therefore the same cap, so it is restored to its
// original bucket. A slice whose cap no longer matches any bucket exactly
// (three-index slicing, append-realloc, direct make()) cannot be safely Put
// back — doing so would corrupt the bucket's size invariant — so it is
// dropped to GC and counted in wordSlicePoolMissCount for observability.
//
// Parameters:
//   - slice: The slice to return to the pool. Safe to call with nil.
func releaseWordSlice(slice []big.Word) {
	if slice == nil {
		return
	}
	// Route based on real capacity, never on len(slice). The slice may have
	// been shortened by the caller; the underlying array still has its
	// original capacity.
	c := cap(slice)
	idx := getWordSlicePoolIndex(c)
	if idx >= 0 && wordSliceSizes[idx] == c {
		// Exact bucket match. Restore full capacity (defensive: in case the
		// caller passed slice[:n] for some n < c) before handing back.
		wordSlicePools[idx].Put(slice[:c])
		return
	}
	// Capacity doesn't match any bucket exactly: either the slice was
	// allocated directly (cap > largest bucket), or it was reshaped after
	// acquisition. Let GC reclaim it and record the miss for observability.
	wordSlicePoolMissCount.Add(1)
}

// ─────────────────────────────────────────────────────────────────────────────
// Fermat Pools
// ─────────────────────────────────────────────────────────────────────────────

// fermatPool pools fermat slices by size class.
// Fermat numbers are typically n+1 words where n is derived from FFT parameters.
// Extended size classes support very large FFT operations.
var fermatPools = [...]sync.Pool{
	{New: func() any { return make(fermat, 32) }},
	{New: func() any { return make(fermat, 128) }},
	{New: func() any { return make(fermat, 512) }},
	{New: func() any { return make(fermat, 2048) }},
	{New: func() any { return make(fermat, 8192) }},
	{New: func() any { return make(fermat, 32768) }},
	{New: func() any { return make(fermat, 131072) }},  // 128K
	{New: func() any { return make(fermat, 524288) }},  // 512K
	{New: func() any { return make(fermat, 2097152) }}, // 2M
}

// fermatSizes defines the size classes for fermat pools.
var fermatSizes = [...]int{32, 128, 512, 2048, 8192, 32768, 131072, 524288, 2097152}

// getFermatPoolIndex returns the pool index for a given size.
// Returns -1 if the size is too large for pooling.
//
// Uses O(1) bitwise computation. fermatSizes are powers of 4 times 2,
// starting from 2*4^2 = 32: {32, 128, 512, 2048, 8192, 32768, 131072, 524288, 2097152}.
func getFermatPoolIndex(size int) int {
	if size <= 0 {
		return 0
	}
	if size > fermatSizes[len(fermatSizes)-1] {
		return -1
	}
	// size > 0 by guard above, so size-1 >= 0, uint cast is safe. #nosec G115
	idx := (bits.Len(uint(size-1)) - 4) / 2
	if idx < 0 {
		idx = 0
	}
	return idx
}

// getFermatPoolIndexLinear is the original O(n) linear search implementation,
// kept as a reference for testing the optimized bitwise version.
func getFermatPoolIndexLinear(size int) int {
	for i, s := range fermatSizes {
		if size <= s {
			return i
		}
	}
	return -1
}

// acquireFermat gets a fermat slice of at least the given size from the pool.
// The returned slice is zeroed and has exactly the requested length.
//
// The returned slice should be released using releaseFermat, preferably with defer:
//
//	f := acquireFermat(size)
//	defer releaseFermat(f)
//
// This ensures the slice is returned to the pool even if an error occurs.
func acquireFermat(size int) fermat {
	idx := getFermatPoolIndex(size)
	if idx < 0 {
		return make(fermat, size)
	}
	f := fermatPools[idx].Get().(fermat)
	// Clear and resize (Go 1.21+ built-in)
	clear(f)
	return f[:size]
}

// releaseFermat returns a fermat slice to the pool.
// This should be called with defer immediately after acquireFermat:
//
//	f := acquireFermat(size)
//	defer releaseFermat(f)
//
// Parameters:
//   - f: The fermat slice to return to the pool. Safe to call with nil.
func releaseFermat(f fermat) {
	if f == nil {
		return
	}
	c := cap(f)
	idx := getFermatPoolIndex(c)
	if idx >= 0 && fermatSizes[idx] == c {
		fermatPools[idx].Put(f[:c])
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Nat Slice Pools (for poly.a)
// ─────────────────────────────────────────────────────────────────────────────

// natSlicePool pools []nat slices used for polynomial coefficients.
// Extended to support larger FFT sizes.
var natSlicePools = [...]sync.Pool{
	{New: func() any { return make([]nat, 8) }},
	{New: func() any { return make([]nat, 32) }},
	{New: func() any { return make([]nat, 128) }},
	{New: func() any { return make([]nat, 512) }},
	{New: func() any { return make([]nat, 2048) }},
	{New: func() any { return make([]nat, 8192) }},
	{New: func() any { return make([]nat, 32768) }},
}

// natSliceSizes defines the size classes for nat slice pools.
var natSliceSizes = [...]int{8, 32, 128, 512, 2048, 8192, 32768}

// getNatSlicePoolIndex returns the pool index for a given size.
// Returns -1 if the size is too large for pooling.
//
// Uses O(1) bitwise computation. natSliceSizes are powers of 4 times 2,
// starting from 2*4^1 = 8: {8, 32, 128, 512, 2048, 8192, 32768}.
func getNatSlicePoolIndex(size int) int {
	if size <= 0 {
		return 0
	}
	if size > natSliceSizes[len(natSliceSizes)-1] {
		return -1
	}
	// size > 0 by guard above, so size-1 >= 0, uint cast is safe. #nosec G115
	idx := (bits.Len(uint(size-1)) - 2) / 2
	if idx < 0 {
		idx = 0
	}
	return idx
}

// getNatSlicePoolIndexLinear is the original O(n) linear search implementation,
// kept as a reference for testing the optimized bitwise version.
func getNatSlicePoolIndexLinear(size int) int {
	for i, s := range natSliceSizes {
		if size <= s {
			return i
		}
	}
	return -1
}

// acquireNatSlice gets a []nat slice of at least the given size from the pool.
//
// The returned slice should be released using releaseNatSlice, preferably with defer:
//
//	slice := acquireNatSlice(size)
//	defer releaseNatSlice(slice)
//
// This ensures the slice is returned to the pool even if an error occurs.
func acquireNatSlice(size int) []nat {
	idx := getNatSlicePoolIndex(size)
	if idx < 0 {
		return make([]nat, size)
	}
	slice := natSlicePools[idx].Get().([]nat)
	// Clear the slice (Go 1.21+ built-in)
	clear(slice)
	return slice[:size]
}

// releaseNatSlice returns a []nat slice to the pool.
// This should be called with defer immediately after acquireNatSlice:
//
//	slice := acquireNatSlice(size)
//	defer releaseNatSlice(slice)
//
// Parameters:
//   - slice: The slice to return to the pool. Safe to call with nil.
func releaseNatSlice(slice []nat) {
	if slice == nil {
		return
	}
	c := cap(slice)
	idx := getNatSlicePoolIndex(c)
	if idx >= 0 && natSliceSizes[idx] == c {
		natSlicePools[idx].Put(slice[:c])
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Fermat Slice Pools (for polValues.values)
// ─────────────────────────────────────────────────────────────────────────────

// fermatSlicePool pools []fermat slices used for polynomial values.
// Extended to support larger FFT sizes.
var fermatSlicePools = [...]sync.Pool{
	{New: func() any { return make([]fermat, 8) }},
	{New: func() any { return make([]fermat, 32) }},
	{New: func() any { return make([]fermat, 128) }},
	{New: func() any { return make([]fermat, 512) }},
	{New: func() any { return make([]fermat, 2048) }},
	{New: func() any { return make([]fermat, 8192) }},
	{New: func() any { return make([]fermat, 32768) }},
}

// fermatSliceSizes defines the size classes for []fermat pools.
var fermatSliceSizes = [...]int{8, 32, 128, 512, 2048, 8192, 32768}

// getFermatSlicePoolIndex returns the pool index for a given size.
// Returns -1 if the size is too large for pooling.
//
// Uses O(1) bitwise computation. fermatSliceSizes are powers of 4 times 2,
// starting from 2*4^1 = 8: {8, 32, 128, 512, 2048, 8192, 32768}.
func getFermatSlicePoolIndex(size int) int {
	if size <= 0 {
		return 0
	}
	if size > fermatSliceSizes[len(fermatSliceSizes)-1] {
		return -1
	}
	// size > 0 by guard above, so size-1 >= 0, uint cast is safe. #nosec G115
	idx := (bits.Len(uint(size-1)) - 2) / 2
	if idx < 0 {
		idx = 0
	}
	return idx
}

// getFermatSlicePoolIndexLinear is the original O(n) linear search implementation,
// kept as a reference for testing the optimized bitwise version.
func getFermatSlicePoolIndexLinear(size int) int {
	for i, s := range fermatSliceSizes {
		if size <= s {
			return i
		}
	}
	return -1
}

// acquireFermatSlice gets a []fermat slice of at least the given size from the pool.
//
// The returned slice should be released using releaseFermatSlice, preferably with defer:
//
//	slice := acquireFermatSlice(size)
//	defer releaseFermatSlice(slice)
//
// This ensures the slice is returned to the pool even if an error occurs.
func acquireFermatSlice(size int) []fermat {
	idx := getFermatSlicePoolIndex(size)
	if idx < 0 {
		return make([]fermat, size)
	}
	slice := fermatSlicePools[idx].Get().([]fermat)
	// Clear the slice (Go 1.21+ built-in)
	clear(slice)
	return slice[:size]
}

// releaseFermatSlice returns a []fermat slice to the pool.
// This should be called with defer immediately after acquireFermatSlice:
//
//	slice := acquireFermatSlice(size)
//	defer releaseFermatSlice(slice)
//
// Parameters:
//   - slice: The slice to return to the pool. Safe to call with nil.
func releaseFermatSlice(slice []fermat) {
	if slice == nil {
		return
	}
	c := cap(slice)
	idx := getFermatSlicePoolIndex(c)
	if idx >= 0 && fermatSliceSizes[idx] == c {
		fermatSlicePools[idx].Put(slice[:c])
	}
}
