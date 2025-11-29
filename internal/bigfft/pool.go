// Package bigfft implements multiplication of big.Int using FFT.
// This file provides memory pooling for FFT operations to reduce GC pressure.
package bigfft

import (
	"math/big"
	"sync"
)

// ─────────────────────────────────────────────────────────────────────────────
// Word Slice Pools
// ─────────────────────────────────────────────────────────────────────────────

// wordSlicePool pools []big.Word slices by size class.
// We use size classes to avoid fragmentation: 64, 256, 1K, 4K, 16K, 64K, 256K words.
var wordSlicePools = [...]sync.Pool{
	{New: func() interface{} { return make([]big.Word, 64) }},
	{New: func() interface{} { return make([]big.Word, 256) }},
	{New: func() interface{} { return make([]big.Word, 1024) }},
	{New: func() interface{} { return make([]big.Word, 4096) }},
	{New: func() interface{} { return make([]big.Word, 16384) }},
	{New: func() interface{} { return make([]big.Word, 65536) }},
	{New: func() interface{} { return make([]big.Word, 262144) }},
}

// wordSliceSizes defines the size classes for word slice pools.
var wordSliceSizes = [...]int{64, 256, 1024, 4096, 16384, 65536, 262144}

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
var fermatPools = [...]sync.Pool{
	{New: func() interface{} { return make(fermat, 32) }},
	{New: func() interface{} { return make(fermat, 128) }},
	{New: func() interface{} { return make(fermat, 512) }},
	{New: func() interface{} { return make(fermat, 2048) }},
	{New: func() interface{} { return make(fermat, 8192) }},
	{New: func() interface{} { return make(fermat, 32768) }},
}

// fermatSizes defines the size classes for fermat pools.
var fermatSizes = [...]int{32, 128, 512, 2048, 8192, 32768}

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
var natSlicePools = [...]sync.Pool{
	{New: func() interface{} { return make([]nat, 8) }},
	{New: func() interface{} { return make([]nat, 32) }},
	{New: func() interface{} { return make([]nat, 128) }},
	{New: func() interface{} { return make([]nat, 512) }},
	{New: func() interface{} { return make([]nat, 2048) }},
}

// natSliceSizes defines the size classes for nat slice pools.
var natSliceSizes = [...]int{8, 32, 128, 512, 2048}

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
var fermatSlicePools = [...]sync.Pool{
	{New: func() interface{} { return make([]fermat, 8) }},
	{New: func() interface{} { return make([]fermat, 32) }},
	{New: func() interface{} { return make([]fermat, 128) }},
	{New: func() interface{} { return make([]fermat, 512) }},
	{New: func() interface{} { return make([]fermat, 2048) }},
}

// fermatSliceSizes defines the size classes for []fermat pools.
var fermatSliceSizes = [...]int{8, 32, 128, 512, 2048}

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
	New: func() interface{} {
		return &fftState{}
	},
}

// acquireFFTState gets an fftState from the pool, sized for the given parameters.
func acquireFFTState(n int, k uint) *fftState {
	state := fftStatePool.Get().(*fftState)

	// Allocate or reuse tmp buffers
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
func releaseFFTState(state *fftState) {
	if state == nil {
		return
	}
	// Keep the allocations for reuse
	fftStatePool.Put(state)
}
