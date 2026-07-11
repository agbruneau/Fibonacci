package fibonacci

import (
	"context"
	"math/big"
	"sync"
	"testing"
)

// noopReporter is a non-nil ProgressCallback used by tests that drive
// CalculateCore directly. The shared library uses a no-op reporter when no
// observer is registered (see FibCalculator.CalculateWithObservers); tests
// that bypass the wrapper must do the same to avoid a nil-call panic from
// progress.ReportStepProgress.
func noopReporter(float64) {}

// TestArenaPoolingNoAliasing verifies that a result returned by CalculateCore
// does not share backing memory with the arena retained in the pooled state.
// This is the safety invariant that lets us pool arenas across calls.
func TestArenaPoolingNoAliasing(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fd := &FastDoublingCalculator{}

	r1, err := fd.CalculateCore(ctx, noopReporter, 100_000, Options{})
	if err != nil {
		t.Fatalf("first calc: %v", err)
	}
	snapshot1 := new(big.Int).Set(r1) // independent copy for later comparison

	r2, err := fd.CalculateCore(ctx, noopReporter, 100_000, Options{})
	if err != nil {
		t.Fatalf("second calc: %v", err)
	}
	_ = r2

	// r1 must equal F(100_000) computed earlier — i.e., r2's calculation must not
	// have stomped on r1's backing memory via the pooled arena.
	if r1.Cmp(snapshot1) != 0 {
		t.Fatalf("r1 was mutated by r2 — arena aliasing leak")
	}
}

// TestArenaStateConcurrent stresses the pooled state+arena with many
// concurrent calculations. This is the regression test for the SKIPPED
// race condition; on Windows we run with -count=50 to compensate for
// the absence of -race.
func TestArenaStateConcurrent(t *testing.T) {
	t.Parallel()
	const goroutines = 16
	const itersPerG = 8

	fd := &FastDoublingCalculator{}
	ctx := context.Background()

	// Reference values computed once.
	refs := make(map[uint64]*big.Int)
	for _, n := range []uint64{1000, 10_000, 50_000} {
		r, err := fd.CalculateCore(ctx, noopReporter, n, Options{})
		if err != nil {
			t.Fatalf("ref calc N=%d: %v", n, err)
		}
		refs[n] = r
	}

	var wg sync.WaitGroup
	wg.Add(goroutines)
	errs := make(chan error, goroutines*itersPerG)
	for g := 0; g < goroutines; g++ {
		go func(seed int) {
			defer wg.Done()
			sizes := []uint64{1000, 10_000, 50_000}
			for i := 0; i < itersPerG; i++ {
				n := sizes[(seed+i)%3]
				r, err := fd.CalculateCore(ctx, noopReporter, n, Options{})
				if err != nil {
					errs <- err
					return
				}
				if r.Cmp(refs[n]) != 0 {
					errs <- &mismatchErr{N: n}
					return
				}
			}
		}(g)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Fatal(e)
	}
}

type mismatchErr struct{ N uint64 }

func (m *mismatchErr) Error() string {
	return "concurrent result mismatch for N=" + new(big.Int).SetUint64(m.N).String()
}

// TestStateReuseAcrossSizes ensures that an arena sized for small N is
// either reused (if large enough) or grown for a larger N without crashing.
func TestStateReuseAcrossSizes(t *testing.T) {
	t.Parallel()
	fd := &FastDoublingCalculator{}
	ctx := context.Background()

	seq := []uint64{1000, 100_000, 1000, 500_000, 10_000}
	for _, n := range seq {
		if _, err := fd.CalculateCore(ctx, noopReporter, n, Options{}); err != nil {
			t.Fatalf("calc N=%d: %v", n, err)
		}
	}
}

// TestReleaseStateWithResult_NilState verifies safety on nil state.
func TestReleaseStateWithResult_NilState(t *testing.T) {
	t.Parallel()
	src := big.NewInt(42)
	got := ReleaseStateWithResult(nil, src)
	if got != src {
		t.Fatalf("expected pass-through src on nil state, got %v", got)
	}
}

// TestAcquireStateForN_RoundTrip verifies the new pooling API works for a
// trivial size where the arena is not pre-sized.
func TestAcquireStateForN_RoundTrip(t *testing.T) {
	t.Parallel()
	s := AcquireStateForN(100) // n <= 1000: no arena pre-sizing
	if s == nil {
		t.Fatal("nil state from AcquireStateForN")
	}
	if s.FK == nil || s.FK1 == nil {
		t.Fatal("FK/FK1 not initialized")
	}
	src := new(big.Int).SetInt64(123)
	out := ReleaseStateWithResult(s, src)
	if out.Cmp(big.NewInt(123)) != 0 {
		t.Fatalf("expected 123, got %s", out.String())
	}
}

// TestReleaseState_OverLimit_AliasesCleared is the regression guard for the
// R1.1 latent use-after-free fix. It asserts that even when overLimit is true
// (which causes the state to be DROPPED from the pool rather than reused),
// every big.Int slot has been re-pointed to a fresh, empty header BEFORE the
// release returns. Without this guarantee, a slot left aliasing arena memory
// could observe a stale buffer if the arena were ever reused independently
// (e.g., via a refactor that detaches the arena from the state lifecycle).
//
// We construct the over-limit condition synthetically by inflating one slot
// past MaxPooledBitLen via SetBit (which only allocates the necessary words,
// avoiding the cost of computing a real F(n) > 50M bits).
func TestReleaseState_OverLimit_AliasesCleared(t *testing.T) {
	t.Parallel()

	// Helper: assert every slot is a fresh big.Int (BitLen == 0) and is a
	// distinct pointer from the captured aliased header.
	assertCleared := func(t *testing.T, s *CalculationState, captured map[string]*big.Int) {
		t.Helper()
		slots := map[string]*big.Int{
			"FK": s.FK, "FK1": s.FK1, "T1": s.T1, "T2": s.T2, "T3": s.T3,
		}
		for name, slot := range slots {
			if slot == nil {
				t.Fatalf("slot %s is nil after release", name)
			}
			if slot.BitLen() != 0 {
				t.Fatalf("slot %s has BitLen=%d after release, want 0 (alias not cleared)", name, slot.BitLen())
			}
			if slot == captured[name] {
				t.Fatalf("slot %s pointer unchanged after release; clearStateAliases did not run on overLimit branch", name)
			}
		}
	}

	t.Run("ReleaseState_overLimit", func(t *testing.T) {
		t.Parallel()
		s := AcquireStateForN(0)
		// Force overLimit: inflate FK past MaxPooledBitLen.
		s.FK.SetBit(s.FK, MaxPooledBitLen+1, 1)
		if !checkLimit(s.FK) {
			t.Fatalf("precondition failed: FK BitLen=%d not over MaxPooledBitLen=%d", s.FK.BitLen(), MaxPooledBitLen)
		}

		captured := map[string]*big.Int{
			"FK": s.FK, "FK1": s.FK1, "T1": s.T1, "T2": s.T2, "T3": s.T3,
		}

		ReleaseState(s)

		// After release we still hold the state pointer (we are testing the
		// invariant that the state itself was scrubbed before the pool drop).
		assertCleared(t, s, captured)
	})

	t.Run("ReleaseStateWithResult_overLimit", func(t *testing.T) {
		t.Parallel()
		s := AcquireStateForN(0)
		// Force overLimit on a different slot to cover the OR chain.
		s.T2.SetBit(s.T2, MaxPooledBitLen+1, 1)
		if !checkLimit(s.T2) {
			t.Fatalf("precondition failed: T2 BitLen=%d not over MaxPooledBitLen=%d", s.T2.BitLen(), MaxPooledBitLen)
		}

		captured := map[string]*big.Int{
			"FK": s.FK, "FK1": s.FK1, "T1": s.T1, "T2": s.T2, "T3": s.T3,
		}

		src := big.NewInt(7)
		out := ReleaseStateWithResult(s, src)
		if out == nil || out.Cmp(big.NewInt(7)) != 0 {
			t.Fatalf("expected result=7, got %v", out)
		}

		assertCleared(t, s, captured)
	})

	t.Run("ReleaseState_nominal_alsoCleared", func(t *testing.T) {
		// Belt-and-suspenders: the nominal (non-overLimit) path must also
		// clear aliases. This is the historically-correct path; we check it
		// here so a regression on EITHER branch trips this test.
		t.Parallel()
		s := AcquireStateForN(0)
		s.FK.SetInt64(42) // small enough: not over-limit

		captured := map[string]*big.Int{
			"FK": s.FK, "FK1": s.FK1, "T1": s.T1, "T2": s.T2, "T3": s.T3,
		}

		// No-op sink: on the nominal path ReleaseState publishes s into the
		// shared statePool, and assertCleared below would then read slots
		// concurrently mutated by AcquireStateForN in parallel tests.
		finalizeStateReleaseTo(s, func(*CalculationState) {})
		assertCleared(t, s, captured)
	})
}
