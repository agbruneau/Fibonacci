// Package bigfft implements multiplication of big.Int using FFT.
// This file provides memory pooling for FFT operations to reduce GC pressure.
package bigfft

import (
	"math/big"
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
func getWordSlicePoolIndex(size int) int {
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
	// Clear the slice before returning
	for i := range slice {
		slice[i] = 0
	}
	return slice[:size]
}

// releaseWordSlice returns a word slice to the pool.
// The slice must have been obtained from acquireWordSlice.
// This should be called with defer immediately after acquireWordSlice:
//
//	slice := acquireWordSlice(size)
//	defer releaseWordSlice(slice)
//
// Parameters:
//   - slice: The slice to return to the pool. Safe to call with nil.
func releaseWordSlice(slice []big.Word) {
	if slice == nil {
		return
	}
	// Get the original capacity to determine which pool it came from
	cap := cap(slice)
	idx := getWordSlicePoolIndex(cap)
	if idx >= 0 && wordSliceSizes[idx] == cap {
		// Restore full capacity before returning to pool
		wordSlicePools[idx].Put(slice[:cap])
	}
	// If capacity doesn't match a pool size, it was directly allocated - let GC handle it
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
func getFermatPoolIndex(size int) int {
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
	// Clear and resize
	for i := range f {
		f[i] = 0
	}
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
	cap := cap(f)
	idx := getFermatPoolIndex(cap)
	if idx >= 0 && fermatSizes[idx] == cap {
		fermatPools[idx].Put(f[:cap])
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
func getNatSlicePoolIndex(size int) int {
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
	// Clear the slice
	for i := range slice {
		slice[i] = nil
	}
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
	cap := cap(slice)
	idx := getNatSlicePoolIndex(cap)
	if idx >= 0 && natSliceSizes[idx] == cap {
		natSlicePools[idx].Put(slice[:cap])
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
func getFermatSlicePoolIndex(size int) int {
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
	// Clear the slice
	for i := range slice {
		slice[i] = nil
	}
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
	cap := cap(slice)
	idx := getFermatSlicePoolIndex(cap)
	if idx >= 0 && fermatSliceSizes[idx] == cap {
		fermatSlicePools[idx].Put(slice[:cap])
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// FFT State Pool (combines all temporary allocations for one FFT operation)
// ─────────────────────────────────────────────────────────────────────────────

// fftState holds all temporary allocations for a single FFT multiplication.
// Using a combined state allows us to reuse all allocations together.
type fftState struct {
	// Fourier transform temporaries
	tmp  fermat
	tmp2 fermat
	// Current size parameters
	n int
	k uint
}

// fftStatePool pools fftState objects.
var fftStatePool = sync.Pool{
	New: func() any {
		return &fftState{}
	},
}

// acquireFFTState gets an fftState from the pool, sized for the given parameters.
// The returned state must be released using releaseFFTState, preferably with defer:
//
//	state := acquireFFTState(n, k)
//	defer releaseFFTState(state)
//
// This ensures the state and its internal buffers are returned to the pool
// even if an error occurs or a panic is triggered.
func acquireFFTState(n int, k uint) *fftState {
	state := fftStatePool.Get().(*fftState)

	// Allocate or reuse tmp buffers
	// Note: If acquireFermat fails (shouldn't happen), the state will still
	// be returned to the pool via releaseFFTState, but the buffers won't be
	// properly initialized. This is acceptable as acquireFermat never fails.
	tmpSize := n + 1
	if cap(state.tmp) < tmpSize {
		state.tmp = acquireFermat(tmpSize)
	} else {
		state.tmp = state.tmp[:tmpSize]
		for i := range state.tmp {
			state.tmp[i] = 0
		}
	}

	if cap(state.tmp2) < tmpSize {
		state.tmp2 = acquireFermat(tmpSize)
	} else {
		state.tmp2 = state.tmp2[:tmpSize]
		for i := range state.tmp2 {
			state.tmp2[i] = 0
		}
	}

	state.n = n
	state.k = k
	return state
}

// releaseFFTState returns an fftState to the pool.
// This should be called with defer immediately after acquireFFTState to ensure
// proper resource cleanup even in case of errors or panics:
//
//	state := acquireFFTState(n, k)
//	defer releaseFFTState(state)
//
// Parameters:
//   - state: The fftState to return to the pool. Safe to call with nil.
func releaseFFTState(state *fftState) {
	if state == nil {
		return
	}
	// Keep the allocations for reuse
	// Note: The internal tmp and tmp2 buffers are kept with the state
	// and will be reused on the next acquisition, reducing allocations.
	fftStatePool.Put(state)
}

// ─────────────────────────────────────────────────────────────────────────────
// Pool Pre-warming
// ─────────────────────────────────────────────────────────────────────────────

// PreWarmPools pre-allocates buffers in the pools based on estimated memory
// needs for calculating F(n). This reduces allocation overhead during the
// calculation by ensuring pools have ready-to-use buffers.
//
// The function estimates the required buffer sizes and pre-allocates an
// adaptive number of buffers in each relevant pool size class based on n:
//   - N < 100,000: 2 buffers (minimal overhead)
//   - 100,000 ≤ N < 1,000,000: 4 buffers
//   - 1,000,000 ≤ N < 10,000,000: 5 buffers
//   - N ≥ 10,000,000: 6 buffers (maximum for large calculations)
//
// This adaptive approach provides better performance for large calculations
// by reducing allocations during the computation.
//
// Parameters:
//   - n: The Fibonacci index to calculate (used for estimation).
func PreWarmPools(n uint64) {
	est := EstimateMemoryNeeds(n)

	// Determine the number of buffers based on calculation size
	numBuffers := 2 // Default for small calculations
	if n >= 10_000_000 {
		numBuffers = 6
	} else if n >= 1_000_000 {
		numBuffers = 5
	} else if n >= 100_000 {
		numBuffers = 4
	}

	// Pre-warm word slice pools
	wordIdx := getWordSlicePoolIndex(est.MaxWordSliceSize)
	if wordIdx >= 0 {
		for i := 0; i < numBuffers; i++ {
			buf := make([]big.Word, wordSliceSizes[wordIdx])
			wordSlicePools[wordIdx].Put(buf)
		}
	}

	// Pre-warm fermat pools
	fermatIdx := getFermatPoolIndex(est.MaxFermatSize)
	if fermatIdx >= 0 {
		for i := 0; i < numBuffers; i++ {
			buf := make(fermat, fermatSizes[fermatIdx])
			fermatPools[fermatIdx].Put(buf)
		}
	}

	// Pre-warm nat slice pools
	natIdx := getNatSlicePoolIndex(est.MaxNatSliceSize)
	if natIdx >= 0 {
		for i := 0; i < numBuffers; i++ {
			buf := make([]nat, natSliceSizes[natIdx])
			natSlicePools[natIdx].Put(buf)
		}
	}

	// Pre-warm fermat slice pools
	fermatSliceIdx := getFermatSlicePoolIndex(est.MaxFermatSliceSize)
	if fermatSliceIdx >= 0 {
		for i := 0; i < numBuffers; i++ {
			buf := make([]fermat, fermatSliceSizes[fermatSliceIdx])
			fermatSlicePools[fermatSliceIdx].Put(buf)
		}
	}
}

// poolsWarmed tracks whether pools have been pre-warmed.
// Using sync/atomic for lock-free, thread-safe initialization.
var poolsWarmed atomic.Bool

// EnsurePoolsWarmed ensures that pools are pre-warmed exactly once.
// This is more efficient than calling PreWarmPools on every calculation,
// as it uses atomic compare-and-swap to guarantee single initialization.
//
// The function is safe to call concurrently from multiple goroutines.
// Only the first call will actually pre-warm the pools; subsequent calls
// return immediately.
//
// Parameters:
//   - maxN: The maximum Fibonacci index expected (used for estimation).
func EnsurePoolsWarmed(maxN uint64) {
	if poolsWarmed.CompareAndSwap(false, true) {
		PreWarmPools(maxN)
	}
}
