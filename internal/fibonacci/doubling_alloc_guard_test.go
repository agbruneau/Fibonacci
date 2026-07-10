//go:build !race

// The race detector instruments allocations, so this guard only runs in the
// non-race (Windows) leg of the gate — the same split as the pool leak
// guards in bigfft.

package fibonacci

import (
	"context"
	"testing"
)

// TestDoublingLoopAllocBudget bounds the steady-state allocations of a full
// doubling loop. The critical loop had no allocation guard at all: one
// allocation added per iteration keeps the golden green and only shows up
// in a benchstat run noisy enough to hide it.
//
// Parallelism and FFT are disabled via negative thresholds (FIB-02
// semantics: only >0 enables them) so the loop is single-goroutine and
// deterministic; the first Calculate warms the pools and the GC-immune
// state cache, then the budget is enforced on the steady state.
// n=100_000 gives 17 iterations, so the +8 headroom above the measured
// steady state cannot absorb even a single new allocation per iteration.
func TestDoublingLoopAllocBudget(t *testing.T) {
	// No t.Parallel: AllocsPerRun reads global malloc counters and a
	// concurrent test's allocations would be charged to this budget.
	calc := MustNewCalculator(&FastDoublingCalculator{})
	ctx := context.Background()
	opts := Options{ParallelThreshold: -1, FFTThreshold: -1}
	const n = 100_000

	run := func() {
		if _, err := calc.Calculate(ctx, nil, 0, n, opts); err != nil {
			t.Fatalf("Calculate(%d) failed: %v", n, err)
		}
	}
	run() // warm pools and the GC-immune state cache

	avg := testing.AllocsPerRun(5, run)
	t.Logf("steady-state allocations per Calculate(%d): %.1f", n, avg)

	// Measured steady state: 13.0 allocations (stable across runs). The +8
	// headroom absorbs pool-purge jitter but not one allocation per
	// iteration (+17).
	const maxAllocs = 21
	if avg > maxAllocs {
		t.Errorf("doubling loop allocates %.1f per run, budget is %d; an allocation was added to the critical loop", avg, maxAllocs)
	}
}
