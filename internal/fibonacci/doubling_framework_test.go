package fibonacci

import (
	"context"
	"testing"

	"github.com/agbruneau/FibGo/internal/fibonacci/threshold"
)

// newTestDynamicThresholdManager adapts a fixed-argument shape to
// threshold.NewDynamicThresholdManagerFromConfig, the only constructor
// production (fastdoubling.go) calls.
func newTestDynamicThresholdManager(fftThreshold, parallelThreshold int) *threshold.DynamicThresholdManager {
	return threshold.NewDynamicThresholdManagerFromConfig(threshold.DynamicThresholdConfig{
		Enabled:                  true,
		InitialFFTThreshold:      fftThreshold,
		InitialParallelThreshold: parallelThreshold,
	})
}

func TestNewDoublingFrameworkWithDynamicThresholds(t *testing.T) {
	t.Parallel()

	t.Run("Create with dynamic thresholds", func(t *testing.T) {
		t.Parallel()
		strategy := &AdaptiveStrategy{}
		dtm := newTestDynamicThresholdManager(1000000, 4096)

		framework := NewDoublingFrameworkWithDynamicThresholds(strategy, dtm)

		if framework == nil {
			t.Fatal("Framework should not be nil")
		}
		if framework.strategy != strategy {
			t.Error("Strategy should be set correctly")
		}
		if framework.dynamicThreshold != dtm {
			t.Error("Dynamic threshold manager should be set correctly")
		}
	})

	t.Run("Create with nil dynamic thresholds", func(t *testing.T) {
		t.Parallel()
		strategy := &AdaptiveStrategy{}

		framework := NewDoublingFrameworkWithDynamicThresholds(strategy, nil)

		if framework == nil {
			t.Fatal("Framework should not be nil")
		}
		if framework.strategy != strategy {
			t.Error("Strategy should be set correctly")
		}
		if framework.dynamicThreshold != nil {
			t.Error("Dynamic threshold manager should be nil")
		}
	})
}

// TestDoublingFramework_CacheTuningRunsWithDTM is a smoke test for the
// inlined cache-tuning call in ExecuteDoublingLoop (cache_tuning.go):
// running a full doubling loop with dynamic thresholds enabled must not
// panic or error, and must still produce the correct result. The tuning
// heuristic itself (decideCacheTuning) is unit-tested in isolation by
// TestDecideCacheTuning; the "only runs alongside DTM" gate is now a
// structural property of the code (the call sits inside the same `if dtm !=
// nil` block as threshold adjustment) rather than a separately-settable
// field, so there is nothing left to assert by injection.
func TestDoublingFramework_CacheTuningRunsWithDTM(t *testing.T) {
	t.Parallel()

	dtm := newTestDynamicThresholdManager(1_000_000, 4096)
	framework := NewDoublingFrameworkWithDynamicThresholds(&AdaptiveStrategy{}, dtm)

	// F(64) → numBits = bits.Len64(64) = 7 iterations, crossing the
	// cacheSampleInterval throttle at least once plus the final iteration.
	const n uint64 = 64
	s := AcquireStateForN(n)
	raw, err := framework.ExecuteDoublingLoop(context.Background(), noopReporter, n, Options{}, s, false)
	if err != nil {
		ReleaseState(s)
		t.Fatalf("ExecuteDoublingLoop returned error: %v", err)
	}
	result := ReleaseStateWithResult(s, raw)

	const wantF64 = 10610209857723 // known value of F(64)
	if result.Uint64() != wantF64 {
		t.Errorf("F(64) = %s, want %d", result.String(), wantF64)
	}
}
