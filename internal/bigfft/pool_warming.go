// Pool pre-warming for adaptive buffer pre-allocation based on calculation size.
//
// Pool warming pre-populates sync.Pool instances with appropriately-sized buffers
// before a Fibonacci calculation begins, so that the first few pool acquisitions
// return pre-allocated buffers instead of triggering fresh heap allocations.
//
// When it is beneficial:
//
//   - Large FFT calculations (n >= 100,000) where multiple large buffers are
//     needed early in the computation. Pre-warming avoids allocation spikes
//     during the latency-sensitive initial iterations and reduces GC pressure
//     from many simultaneous large allocations.
//
// When it is NOT beneficial:
//
//   - Small calculations (n < ~10,000) where buffers are tiny and allocation
//     cost is negligible. The overhead of estimating sizes and pre-filling pools
//     can exceed the savings.
//   - Short-lived processes that exit before the pools would be reused, since
//     the pre-allocated buffers are never reclaimed through recycling.
//
// See BenchmarkPoolWithWarming and BenchmarkPoolWithoutWarming in
// pool_warming_bench_test.go for empirical validation on your hardware.

package bigfft

import (
	"math/big"
	"sync/atomic"
)

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
	switch {
	case n >= 10_000_000:
		numBuffers = 6
	case n >= 1_000_000:
		numBuffers = 5
	case n >= 100_000:
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

// EnsurePoolsWarmed ensures that pools are pre-warmed exactly once, sized for
// the first maxN seen. It uses atomic compare-and-swap to guarantee single
// initialization and is safe to call concurrently from multiple goroutines.
//
// Known ceiling (intentional): subsequent calls — including ones with a LARGER
// maxN — return immediately without re-warming. A later, larger calculation
// simply grows the size-class pools on demand. Pre-warming is a cold-start
// optimization, not an adaptive sizing mechanism; re-warming mid-process would
// risk pool thrashing for negligible benefit, so the one-shot behavior is by
// design.
//
// Parameters:
//   - maxN: The maximum Fibonacci index expected (used for estimation).
func EnsurePoolsWarmed(maxN uint64) {
	if poolsWarmed.CompareAndSwap(false, true) {
		PreWarmPools(maxN)
	}
}
