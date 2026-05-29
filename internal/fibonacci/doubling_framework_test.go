package fibonacci

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/agbruneau/FibGo/internal/fibonacci/threshold"
)

func TestNewDoublingFrameworkWithDynamicThresholds(t *testing.T) {
	t.Parallel()

	t.Run("Create with dynamic thresholds", func(t *testing.T) {
		t.Parallel()
		strategy := &AdaptiveStrategy{}
		dtm := threshold.NewDynamicThresholdManager(1000000, 4096)

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

// mockCacheStrategy is a CacheStrategy double used by the unit tests below to
// verify that ExecuteDoublingLoop wires the strategy correctly without
// touching the global bigfft cache.
type mockCacheStrategy struct {
	calls       atomic.Int64
	lastIter    atomic.Int64
	lastTotal   atomic.Int64
	returnError error
}

func (m *mockCacheStrategy) Sample(iter int, totalIters int) error {
	m.calls.Add(1)
	m.lastIter.Store(int64(iter))
	m.lastTotal.Store(int64(totalIters))
	return m.returnError
}

// TestDoublingFramework_CacheStrategyInjected verifies that an injected
// CacheStrategy is invoked from ExecuteDoublingLoop when dynamic thresholds
// are enabled (the historical "DTM gate" preserved by the R3.2 refactor).
func TestDoublingFramework_CacheStrategyInjected(t *testing.T) {
	t.Parallel()

	mock := &mockCacheStrategy{}
	dtm := threshold.NewDynamicThresholdManager(1_000_000, 4096)
	framework := NewDoublingFrameworkWithDynamicThresholds(&AdaptiveStrategy{}, dtm)
	framework.CacheStrategy = mock // override the default bigfft wrapper

	// F(64) → numBits = bits.Len64(64) = 7 iterations.
	const n uint64 = 64
	s := AcquireStateForN(n)
	raw, err := framework.ExecuteDoublingLoop(context.Background(), noopReporter, n, Options{}, s, false)
	if err != nil {
		ReleaseState(s)
		t.Fatalf("ExecuteDoublingLoop returned error: %v", err)
	}
	result := ReleaseStateWithResult(s, raw)

	// F(64) = 10610209857723.
	if result.Uint64() != 10610209857723 {
		t.Errorf("F(64) = %s, want 10610209857723", result.String())
	}

	// Strategy must be called once per iteration (numBits = 7) — Sample throttling
	// is the strategy's own concern; the loop dispatches unconditionally.
	if got := mock.calls.Load(); got != 7 {
		t.Errorf("CacheStrategy.Sample called %d times, want 7", got)
	}
	if got := mock.lastTotal.Load(); got != 7 {
		t.Errorf("CacheStrategy received totalIters=%d, want 7", got)
	}
	if got := mock.lastIter.Load(); got != 7 {
		t.Errorf("CacheStrategy last iter=%d, want 7 (final iteration)", got)
	}
}

// TestDoublingFramework_CacheStrategySkippedWithoutDTM ensures the DTM gate is
// preserved: when dynamicThreshold is nil the strategy must never be called,
// even if one is set on the framework.
func TestDoublingFramework_CacheStrategySkippedWithoutDTM(t *testing.T) {
	t.Parallel()

	mock := &mockCacheStrategy{}
	framework := NewDoublingFramework(&AdaptiveStrategy{})
	framework.CacheStrategy = mock

	const n uint64 = 64
	s := AcquireStateForN(n)
	raw, err := framework.ExecuteDoublingLoop(context.Background(), noopReporter, n, Options{}, s, false)
	if err != nil {
		ReleaseState(s)
		t.Fatalf("ExecuteDoublingLoop returned error: %v", err)
	}
	_ = ReleaseStateWithResult(s, raw)

	if got := mock.calls.Load(); got != 0 {
		t.Errorf("CacheStrategy.Sample called %d times without DTM, want 0", got)
	}
}

// TestDoublingFramework_CacheStrategyError verifies that an error returned by
// the strategy aborts the loop and is propagated.
func TestDoublingFramework_CacheStrategyError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("cache sample boom")
	mock := &mockCacheStrategy{returnError: wantErr}
	dtm := threshold.NewDynamicThresholdManager(1_000_000, 4096)
	framework := NewDoublingFrameworkWithDynamicThresholds(&AdaptiveStrategy{}, dtm)
	framework.CacheStrategy = mock

	const n uint64 = 64
	s := AcquireStateForN(n)
	_, err := framework.ExecuteDoublingLoop(context.Background(), noopReporter, n, Options{}, s, false)
	ReleaseState(s)

	if err == nil {
		t.Fatal("expected error from cache strategy, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("error = %v, want it to wrap %v", err, wantErr)
	}
}
