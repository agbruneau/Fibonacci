package fibonacci

import (
	"context"
	"math/big"
	"sync"
	"testing"
)

// TestCalculatorStateCache_ReusesArena guards the reason the per-calculator
// slot exists: the arena must survive a release/acquire cycle on the same
// calculator instance (the statePool path is flushed by the GC forced after
// every large calculation, which caused a full arena reallocation per call).
func TestCalculatorStateCache_ReusesArena(t *testing.T) {
	t.Parallel()
	fd := &FastDoublingCalculator{}
	const n = 2_000_000

	s1 := fd.acquireStateForN(n)
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
	// far below maxArenaPoolWords: 30M * 0.694 / 64 * 15 ~= 4.9M words.
	const n = 30_000_000

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

	_, err = fd.CalculateCore(ctx, noop, 120_000, opts)
	if err != nil {
		t.Fatalf("CalculateCore(120k) error: %v", err)
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
