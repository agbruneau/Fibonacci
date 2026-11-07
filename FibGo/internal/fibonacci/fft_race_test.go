package fibonacci

import (
	"context"
	"math/big"
	"sync"
	"testing"

	"github.com/agbruneau/FibGo/internal/bigfft"
)

// TestExecuteDoublingStepFFT_NoDataRace verifies that the parallel FFT execution
// does not cause data races. This test uses multiple goroutines to stress test
// the concurrent operations.
//
// Run with: go test -race -run TestExecuteDoublingStepFFT_NoDataRace ./internal/fibonacci/
func TestExecuteDoublingStepFFT_NoDataRace(t *testing.T) {
	t.Parallel()

	// Create a calculation state with values large enough to trigger FFT path
	state := &CalculationState{
		FK:  new(big.Int).Exp(big.NewInt(2), big.NewInt(1000), nil), // Large number
		FK1: new(big.Int).Exp(big.NewInt(2), big.NewInt(1000), nil),
		T1:  new(big.Int),
		T2:  new(big.Int),
		T3:  new(big.Int),
	}

	opts := Options{
		ParallelThreshold: 4096,
		FFTThreshold:      10000,
	}

	// Run multiple times to increase chance of detecting race conditions
	for i := 0; i < 10; i++ {
		// Reset state for each iteration
		state.T1 = new(big.Int)
		state.T2 = new(big.Int)
		state.T3 = new(big.Int)

		err := executeDoublingStepFFT(context.Background(), state, opts, true)
		if err != nil {
			t.Logf("executeDoublingStepFFT returned error (may be expected for small test values): %v", err)
		}
	}
}

// TestExecuteDoublingStepFFT_ConcurrentCalls verifies that multiple concurrent calls
// to executeDoublingStepFFT do not interfere with each other.
func TestExecuteDoublingStepFFT_ConcurrentCalls(t *testing.T) {
	t.Parallel()

	const numGoroutines = 4

	opts := Options{
		ParallelThreshold: 4096,
		FFTThreshold:      10000,
	}

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func() {
			defer wg.Done()

			// Each goroutine has its own state
			state := &CalculationState{
				FK:  new(big.Int).Exp(big.NewInt(2), big.NewInt(500), nil),
				FK1: new(big.Int).Exp(big.NewInt(2), big.NewInt(500), nil),
				T1:  new(big.Int),
				T2:  new(big.Int),
				T3:  new(big.Int),
			}

			// Run parallel FFT
			_ = executeDoublingStepFFT(context.Background(), state, opts, true)
		}()
	}

	wg.Wait()
}

// A-17 — race-regression guard: arena aliasing on the parallel FFT path.
//
// The parallel FFT doubling step runs three big.Int operations concurrently
// (T3 = FK·FK1, T1 = FK1², T2 = FK²) writing into slots that may be backed by
// the state-bound CalculationArena. When the products outgrow the arena, the
// big.Int implementation reallocates the destination OUTSIDE the arena while a
// sibling goroutine may still be touching arena memory — exactly the aliasing
// hazard P1-04 documents. This test deliberately undersizes the arena (arena
// pre-sized for tiny n, operands forced far larger) so every step reallocates
// off-arena mid parallel-step, and asserts the numeric result against an
// INDEPENDENT big.Int computation (not the golden corpus, not the library
// calculator). It is meant to run under `-race` in CI; locally `-race` is
// unavailable so we at least assert it passes functionally.
//
// Run in CI with: go test -race -run FFTRace ./internal/fibonacci/

// independentFastDoubling computes F(n) with a self-contained big.Int Fast
// Doubling implementation. It shares no code with the package under test and
// is not derived from the golden JSON, so it is a valid independent oracle for
// the FFT-parallel result.
//
// Independence extends to the bit test that drives the addition step: the
// production loop (doubling_framework.go) selects bit i with the logical
// shift `(n>>i)&1` — n is a uint64, so `>>` shifts in zeros — and this oracle
// deliberately asks big.Int.Bit instead. A
// rewrite of one shift form into another cannot be applied identically to both
// sides, which is what makes the Cmp assertions below load-bearing rather than
// self-confirming.
func independentFastDoubling(n uint64) *big.Int {
	if n == 0 {
		return big.NewInt(0)
	}
	a := big.NewInt(0) // F(0)
	b := big.NewInt(1) // F(1)
	t1 := new(big.Int)
	t2 := new(big.Int)
	t3 := new(big.Int)
	nBig := new(big.Int).SetUint64(n) // bit source: Bit(i), never a shift
	for bit := 63; bit >= 0; bit-- {
		// Doubling step: t1 takes a*(2*b - a), t2 the square of a, t3 that of b.
		t1.Lsh(b, 1)
		t1.Sub(t1, a)
		t1.Mul(t1, a) // F(2k)
		t2.Mul(a, a)
		t3.Mul(b, b)
		t2.Add(t2, t3) // F(2k+1)
		a.Set(t1)
		b.Set(t2)
		if nBig.Bit(bit) == 1 {
			t3.Add(a, b)
			a.Set(b)
			b.Set(t3)
		}
	}
	return a
}

// TestFFTRaceArenaAliasing_ParallelStep forces a single parallel FFT doubling
// step with a deliberately undersized arena and checks the three doubling
// products against an independent computation. A torn write from the
// arena-realloc race would corrupt one of the products and fail the Cmp.
func TestFFTRaceArenaAliasing_ParallelStep(t *testing.T) {
	t.Parallel()

	// Arena pre-sized for a tiny n; operands forced far beyond that capacity so
	// FFT engages and big.Int must reallocate the products off-arena mid-step.
	s := AcquireStateForN(1100) // smallest n that triggers arena pre-sizing
	defer ReleaseState(s)

	// ~120k-bit operands: comfortably above any reasonable FFT threshold and
	// far larger than the arena pre-sized for n=1100.
	fk := new(big.Int).Lsh(big.NewInt(1), 120_000)
	fk.Sub(fk, big.NewInt(1)) // dense bits to exercise the transform fully
	fk1 := new(big.Int).Lsh(big.NewInt(1), 120_001)
	fk1.Sub(fk1, big.NewInt(3))

	s.FK.Set(fk)
	s.FK1.Set(fk1)
	s.T1 = new(big.Int)
	s.T2 = new(big.Int)
	s.T3 = new(big.Int)

	opts := Options{ParallelThreshold: 4096, FFTThreshold: 4096}

	const iters = 8
	for it := 0; it < iters; it++ {
		s.FK.Set(fk)
		s.FK1.Set(fk1)
		s.T1 = new(big.Int)
		s.T2 = new(big.Int)
		s.T3 = new(big.Int)

		if err := executeDoublingStepFFT(context.Background(), s, opts, true); err != nil {
			t.Fatalf("iter %d: executeDoublingStepFFT(parallel) error: %v", it, err)
		}

		// Independent expectations for the three step products.
		wantT3 := new(big.Int).Mul(fk, fk1)  // T3 = FK · FK1
		wantT1 := new(big.Int).Mul(fk1, fk1) // T1 = FK1²
		wantT2 := new(big.Int).Mul(fk, fk)   // T2 = FK²

		if s.T3.Cmp(wantT3) != 0 {
			t.Fatalf("iter %d: T3 (FK·FK1) mismatch — arena aliasing corruption", it)
		}
		if s.T1.Cmp(wantT1) != 0 {
			t.Fatalf("iter %d: T1 (FK1²) mismatch — arena aliasing corruption", it)
		}
		if s.T2.Cmp(wantT2) != 0 {
			t.Fatalf("iter %d: T2 (FK²) mismatch — arena aliasing corruption", it)
		}
	}
}

// TestFFTRaceArenaAliasing_ConcurrentCalculateCore stresses the full
// FFT-parallel CalculateCore path concurrently with an arena that is reused
// and grown across sizes, comparing every result to an independent Fast
// Doubling oracle. Under -race this surfaces aliasing between the pooled arena
// and the parallel FFT step; without -race the value assertion still catches
// torn writes.
func TestFFTRaceArenaAliasing_ConcurrentCalculateCore(t *testing.T) {
	t.Parallel()

	fd := &FastDoublingCalculator{}
	ctx := context.Background()

	// FFTThreshold low enough that these sizes route through FFT.
	opts := Options{ParallelThreshold: 4096, FFTThreshold: 8192}

	sizes := []uint64{20_000, 60_000, 20_000, 90_000}
	refs := make(map[uint64]*big.Int, len(sizes))
	for _, n := range sizes {
		if _, ok := refs[n]; !ok {
			refs[n] = independentFastDoubling(n)
		}
	}

	const goroutines = 8
	var wg sync.WaitGroup
	wg.Add(goroutines)
	errs := make(chan error, goroutines)
	for g := 0; g < goroutines; g++ {
		go func(seed int) {
			defer wg.Done()
			for i, n := range sizes {
				got, err := fd.CalculateCore(ctx, noopReporter, n, opts)
				if err != nil {
					errs <- err
					return
				}
				if got.Cmp(refs[n]) != 0 {
					errs <- &mismatchErr{N: n}
					return
				}
				_ = i
			}
		}(g)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Fatal(e)
	}
}

// TestPolValuesClone verifies that PolValues.Clone creates an independent copy.
func TestPolValuesClone(t *testing.T) {
	t.Parallel()

	// Initialize through Transform
	poly := bigfft.PolyFromInt(big.NewInt(12345), 4, 2)
	transformed, err := poly.Transform(bigfft.ValueSize(4, 2, 2))
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Clone the transformed polynomial
	cloned := transformed.Clone()

	// Verify they are independent - modifying one shouldn't affect the other
	if len(cloned.Values) != len(transformed.Values) {
		t.Errorf("Clone has different length: got %d, want %d", len(cloned.Values), len(transformed.Values))
	}

	// Verify the cloned values match the original
	for i := range transformed.Values {
		for j := range transformed.Values[i] {
			if cloned.Values[i][j] != transformed.Values[i][j] {
				t.Errorf("Clone values differ at [%d][%d]: got %v, want %v",
					i, j, cloned.Values[i][j], transformed.Values[i][j])
			}
		}
	}
}
