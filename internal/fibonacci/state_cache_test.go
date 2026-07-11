package fibonacci

import (
	"context"
	"math/big"
	"sync"
	"testing"

	"github.com/agbruneau/FibGo/internal/bigfft"
)

// TestCalculatorStateCache_ReusesArena guards the reason the per-calculator
// slot exists: the arena must survive a release/acquire cycle on the same
// calculator instance (the statePool path is flushed by the GC forced after
// every large calculation, which caused a full arena reallocation per call).
func TestCalculatorStateCache_ReusesArena(t *testing.T) {
	t.Parallel()
	fd := &FastDoublingCalculator{}
	const n = 2_000_000

	// Build the first state manually instead of via the shared statePool:
	// under t.Parallel a pooled state can arrive carrying an over-cap arena
	// from another test, which would legitimately bypass the cache slot and
	// flake this test (workflow review finding).
	s1 := &CalculationState{FK: new(big.Int), FK1: new(big.Int), T1: new(big.Int), T2: new(big.Int), T3: new(big.Int)}
	prepareStateForN(s1, n)
	arena1 := s1.arena
	if arena1 == nil {
		t.Fatal("expected an arena for large n")
	}
	res := fd.releaseStateWithResult(s1, big.NewInt(42))
	if res == nil || res.Int64() != 42 {
		t.Fatalf("releaseStateWithResult = %v, want 42", res)
	}

	s2 := fd.acquireStateForN(n)
	defer fd.releaseState(s2)
	if s2.arena != arena1 {
		t.Error("cached arena not reused on second acquire from the same calculator")
	}
	if s2.FK.Sign() != 0 || s2.FK1.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("reused state not re-initialized: FK=%v FK1=%v, want 0 and 1", s2.FK, s2.FK1)
	}
}

// TestCalculatorStateCache_OverCapGoesToPool ensures arenas above
// maxCachedArenaWords are never pinned in the GC-immune slot (they keep the
// historical pool-only behavior, reclaimable under memory pressure).
func TestCalculatorStateCache_OverCapGoesToPool(t *testing.T) {
	t.Parallel()
	fd := &FastDoublingCalculator{}
	// Sized so the arena exceeds maxCachedArenaWords (4M words) while staying
	// far below maxArenaPoolWords: 50M * 0.694 / 64 * 10 ~= 5.4M words (N raised
	// with the x15 -> x10 multiplier change so the premise stays valid, ADR-0009 R4).
	const n = 50_000_000

	s := fd.acquireStateForN(n)
	if s.arena == nil {
		t.Fatal("expected an arena")
	}
	if s.arenaCapWords <= maxCachedArenaWords {
		t.Fatalf("test premise broken: arenaCapWords=%d, want > %d", s.arenaCapWords, maxCachedArenaWords)
	}
	fd.releaseState(s)
	if fd.cachedState.Load() != nil {
		t.Error("over-cap arena must not be retained in the calculator slot")
	}
}

// TestCalculatorStateCache_OverLimitNotCached extends the
// TestReleaseState_OverLimit_AliasesCleared contract to the calculator sink:
// an overLimit state is dropped by finalizeStateReleaseTo before any sink
// runs, so it must never appear in the cache slot.
func TestCalculatorStateCache_OverLimitNotCached(t *testing.T) {
	t.Parallel()
	fd := &FastDoublingCalculator{}

	s := fd.acquireStateForN(2_000_000)
	s.FK.SetBit(s.FK, MaxPooledBitLen+1, 1)
	fd.releaseState(s)
	if fd.cachedState.Load() != nil {
		t.Error("overLimit state must be dropped, not cached")
	}
}

// TestCalculatorStateCache_SequentialResultsIndependent verifies end to end
// that a result returned by one call is not corrupted by a later call that
// reuses the cached arena (the deep copy out of the arena must hold).
func TestCalculatorStateCache_SequentialResultsIndependent(t *testing.T) {
	t.Parallel()
	fd := &FastDoublingCalculator{}
	ctx := context.Background()
	noop := func(float64) {}
	opts := Options{ParallelThreshold: DefaultParallelThreshold, FFTThreshold: DefaultFFTThreshold}

	r1, err := fd.CalculateCore(ctx, noop, 100_000, opts)
	if err != nil {
		t.Fatalf("CalculateCore(100k) error: %v", err)
	}
	snapshot := new(big.Int).Set(r1)

	// The second call uses a SMALLER n so the SAME arena is reused (a larger
	// n would allocate a fresh arena and the test would not refute a missing
	// deep-copy — workflow review finding). The reuse overwrites the arena
	// backing; r1 must survive untouched.
	_, err = fd.CalculateCore(ctx, noop, 80_000, opts)
	if err != nil {
		t.Fatalf("CalculateCore(80k) error: %v", err)
	}
	if r1.Cmp(snapshot) != 0 {
		t.Error("result of first call mutated by a later call reusing the cached arena")
	}

	r3, err := fd.CalculateCore(ctx, noop, 100_000, opts)
	if err != nil {
		t.Fatalf("CalculateCore(100k) second run error: %v", err)
	}
	if r3.Cmp(snapshot) != 0 {
		t.Error("same-n results differ across cached calls")
	}
}

// TestStateBump_PinnedAcrossCachedCalls guards F-012: once a calculation has
// taken the FFT path, the bump scratch allocator must travel with the cached
// state and be reused (same pointer) by the next call on the same calculator.
func TestStateBump_PinnedAcrossCachedCalls(t *testing.T) {
	t.Parallel()
	fd := &FastDoublingCalculator{}
	ctx := context.Background()
	noop := func(float64) {}
	// F(1.5M) has ~1.04M bits, above the default FFT threshold, so the last
	// doubling steps run executeDoublingStepFFT and acquire the bump scratch.
	opts := Options{ParallelThreshold: DefaultParallelThreshold, FFTThreshold: DefaultFFTThreshold}

	if _, err := fd.CalculateCore(ctx, noop, 1_500_000, opts); err != nil {
		t.Fatalf("CalculateCore #1 error: %v", err)
	}
	s := fd.cachedState.Load()
	if s == nil {
		t.Fatal("state not cached after first call")
	}
	if s.bump == nil {
		t.Fatal("bump scratch not pinned with the cached state after an FFT calculation")
	}
	b1 := s.bump

	if _, err := fd.CalculateCore(ctx, noop, 1_500_000, opts); err != nil {
		t.Fatalf("CalculateCore #2 error: %v", err)
	}
	s2 := fd.cachedState.Load()
	if s2 == nil || s2.bump != b1 {
		t.Error("bump scratch was reacquired instead of reused across cached calls")
	}
}

// TestStateBump_ReleasedOnOverLimit: an overLimit state is dropped, and its
// bump scratch must go back to the bigfft pool instead of dying with it.
func TestStateBump_ReleasedOnOverLimit(t *testing.T) {
	t.Parallel()
	fd := &FastDoublingCalculator{}

	s := fd.acquireStateForN(2_000_000)
	s.bump = bigfft.AcquireBumpAllocator(64)
	s.FK.SetBit(s.FK, MaxPooledBitLen+1, 1)
	fd.releaseState(s)
	if s.bump != nil {
		t.Error("bump scratch must be released (nil) when the state is dropped overLimit")
	}
	if fd.cachedState.Load() != nil {
		t.Error("overLimit state must not be cached")
	}
}

// TestStateBump_FollowsArenaDrop: the anti-bloat drop of a huge arena must
// also release the companion bump scratch (retention policy coherence).
func TestStateBump_FollowsArenaDrop(t *testing.T) {
	t.Parallel()
	s := &CalculationState{FK: new(big.Int), FK1: new(big.Int), T1: new(big.Int), T2: new(big.Int), T3: new(big.Int)}
	prepareStateForN(s, 2_000_000)
	if s.arena == nil {
		t.Fatal("expected an arena")
	}
	s.arenaCapWords = maxArenaPoolWords + 1 // force the anti-bloat drop branch
	s.bump = bigfft.AcquireBumpAllocator(64)
	// No-op sink: the drop logic under test lives in finalizeStateReleaseTo.
	// ReleaseState would publish s into the shared statePool, and the reads
	// below would race with a concurrent AcquireStateForN from parallel tests.
	finalizeStateReleaseTo(s, func(*CalculationState) {})
	if s.arena != nil {
		t.Error("oversized arena must be dropped")
	}
	if s.bump != nil {
		t.Error("bump scratch must follow the arena's anti-bloat drop")
	}
}

// TestCalculatorStateCache_ConcurrentCalls exercises the Swap/CompareAndSwap
// ownership protocol under concurrency on a single instance. Full -race
// validation is delegated to WSL/Linux (no CGO on this host).
func TestCalculatorStateCache_ConcurrentCalls(t *testing.T) {
	t.Parallel()
	fd := &FastDoublingCalculator{}
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				s := fd.acquireStateForN(200_000)
				_ = fd.releaseStateWithResult(s, big.NewInt(int64(j)))
			}
		}()
	}
	wg.Wait()
}
