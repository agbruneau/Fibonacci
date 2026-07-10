package orchestration

import (
	"testing"
	"time"

	"github.com/agbruneau/FibGo/internal/progress"
)

func TestNewProgressAggregator_Positive(t *testing.T) {
	t.Parallel()
	agg := NewProgressAggregator(3)
	if agg == nil {
		t.Fatal("expected non-nil aggregator for numCalculators=3")
	}
	if agg.calculatorCount() != 3 {
		t.Errorf("expected calculatorCount()=3, got %d", agg.calculatorCount())
	}
	if !agg.IsMultiCalculator() {
		t.Error("expected IsMultiCalculator()=true for 3 calculators")
	}
	if agg.state == nil {
		t.Fatal("expected non-nil internal ProgressState")
	}
	if agg.progressRate != 0 {
		t.Errorf("initial progressRate = %f, want 0", agg.progressRate)
	}
	if agg.startTime.IsZero() {
		t.Error("startTime should not be zero")
	}
}

func TestNewProgressAggregator_Single(t *testing.T) {
	t.Parallel()
	agg := NewProgressAggregator(1)
	if agg == nil {
		t.Fatal("expected non-nil aggregator for numCalculators=1")
	}
	if agg.IsMultiCalculator() {
		t.Error("expected IsMultiCalculator()=false for 1 calculator")
	}
}

func TestNewProgressAggregator_Zero(t *testing.T) {
	t.Parallel()
	agg := NewProgressAggregator(0)
	if agg != nil {
		t.Error("expected nil aggregator for numCalculators=0")
	}
}

func TestNewProgressAggregator_Negative(t *testing.T) {
	t.Parallel()
	agg := NewProgressAggregator(-1)
	if agg != nil {
		t.Error("expected nil aggregator for numCalculators=-1")
	}
}

func TestProgressAggregator_Update(t *testing.T) {
	t.Parallel()
	agg := NewProgressAggregator(2)

	ap := agg.Update(progress.ProgressUpdate{CalculatorIndex: 0, Value: 0.5})
	if ap.CalculatorIndex != 0 {
		t.Errorf("expected CalculatorIndex=0, got %d", ap.CalculatorIndex)
	}
	if ap.Value != 0.5 {
		t.Errorf("expected Value=0.5, got %f", ap.Value)
	}
	// Average of [0.5, 0.0] = 0.25
	if ap.AverageProgress != 0.25 {
		t.Errorf("expected AverageProgress=0.25, got %f", ap.AverageProgress)
	}

	ap = agg.Update(progress.ProgressUpdate{CalculatorIndex: 1, Value: 0.5})
	// Average of [0.5, 0.5] = 0.5
	if ap.AverageProgress != 0.5 {
		t.Errorf("expected AverageProgress=0.5, got %f", ap.AverageProgress)
	}
}

func TestProgressAggregator_CalculateAverage(t *testing.T) {
	t.Parallel()
	agg := NewProgressAggregator(2)

	avg := agg.CalculateAverage()
	if avg != 0.0 {
		t.Errorf("expected initial average=0.0, got %f", avg)
	}

	agg.Update(progress.ProgressUpdate{CalculatorIndex: 0, Value: 1.0})
	avg = agg.CalculateAverage()
	if avg != 0.5 {
		t.Errorf("expected average=0.5 after one update, got %f", avg)
	}
}

// TestProgressAggregator_UpdateETA verifies progress aggregation and ETA shape
// across multiple updates. ETA may be 0 at first (insufficient elapsed time)
// but the aggregated progress must always reflect the per-calculator state.
func TestProgressAggregator_UpdateETA(t *testing.T) {
	t.Parallel()
	agg := NewProgressAggregator(2)

	ap := agg.Update(progress.ProgressUpdate{CalculatorIndex: 0, Value: 0.25})
	if ap.AverageProgress != 0.125 { // average of 0.25 and 0
		t.Errorf("initial progress = %f, want 0.125", ap.AverageProgress)
	}
	if ap.ETA < 0 {
		t.Errorf("ETA should not be negative, got %v", ap.ETA)
	}

	ap = agg.Update(progress.ProgressUpdate{CalculatorIndex: 1, Value: 0.5})
	if ap.AverageProgress != 0.375 { // average of 0.25 and 0.5
		t.Errorf("progress = %f, want 0.375", ap.AverageProgress)
	}
}

func TestProgressAggregator_GetETA(t *testing.T) {
	t.Parallel()
	agg := NewProgressAggregator(1)

	// Before any updates, ETA should be 0 (not enough data)
	eta := agg.GetETA()
	if eta != 0 {
		t.Errorf("expected initial ETA=0, got %v", eta)
	}

	// Simulate some progress and inject a known smoothed rate.
	agg.state.Update(0, 0.5)
	agg.progressRate = 0.1 // 10% per second

	eta = agg.GetETA()
	// With 50% remaining at 10%/s, ETA should be around 5 seconds.
	expectedETA := 5 * time.Second
	tolerance := time.Second
	if eta < expectedETA-tolerance || eta > expectedETA+tolerance {
		t.Errorf("ETA = %v, want approximately %v", eta, expectedETA)
	}
}

// TestProgressAggregator_EdgeCases verifies edge case handling for the
// underlying ProgressState through the aggregator.
func TestProgressAggregator_EdgeCases(t *testing.T) {
	t.Parallel()
	t.Run("Progress exceeds 1.0", func(t *testing.T) {
		t.Parallel()
		agg := NewProgressAggregator(1)
		agg.state.Update(0, 1.5)
		p := agg.CalculateAverage()
		if p < 0 {
			t.Errorf("progress should not be negative, got %f", p)
		}
	})

	t.Run("Negative progress", func(t *testing.T) {
		t.Parallel()
		agg := NewProgressAggregator(1)
		agg.state.Update(0, -0.5)
		p := agg.CalculateAverage()
		if p > 1.0 {
			t.Errorf("progress should not exceed 1.0, got %f", p)
		}
	})

	t.Run("Invalid calculator index", func(t *testing.T) {
		t.Parallel()
		agg := NewProgressAggregator(2)
		// Should not panic with invalid index.
		agg.Update(progress.ProgressUpdate{CalculatorIndex: 5, Value: 0.5})
		agg.Update(progress.ProgressUpdate{CalculatorIndex: -1, Value: 0.5})
		p := agg.CalculateAverage()
		if p < 0 || p > 1.0 {
			t.Errorf("progress should be valid, got %f", p)
		}
	})
}

// TestProgressAggregator_ETACapping verifies that ETA is capped at 24h even
// when the rate is very small.
func TestProgressAggregator_ETACapping(t *testing.T) {
	t.Parallel()
	agg := NewProgressAggregator(1)
	agg.state.Update(0, 0.001)   // Very small progress
	agg.progressRate = 0.0000001 // Very slow rate

	eta := agg.GetETA()
	maxETA := 24 * time.Hour

	if eta > maxETA {
		t.Errorf("ETA = %v, should be capped at %v", eta, maxETA)
	}
}

// rewindAggregatorClock shifts the aggregator's internal time anchors into
// the past so tests can exercise the rate computation without sleeping.
// Scheduling delays only increase the observed durations, so assertions
// below are written to be monotonicity-safe in that direction.
func rewindAggregatorClock(a *ProgressAggregator, d time.Duration) {
	a.startTime = a.startTime.Add(-d)
	a.lastUpdate = a.lastUpdate.Add(-d)
}

// TestProgressAggregator_RecomputeETA_FirstRate covers the first meaningful
// rate computation: with no prior smoothed rate, the rate falls back to
// progress/elapsed.
func TestProgressAggregator_RecomputeETA_FirstRate(t *testing.T) {
	t.Parallel()
	agg := NewProgressAggregator(1)
	rewindAggregatorClock(agg, time.Second)

	ap := agg.Update(progress.ProgressUpdate{CalculatorIndex: 0, Value: 0.5})

	// Elapsed is at least the rewound 1s, so progress/elapsed <= 0.5/s.
	if agg.progressRate <= 0 || agg.progressRate > 0.5 {
		t.Errorf("progressRate = %v, want in (0, 0.5]", agg.progressRate)
	}
	if agg.lastProgress != 0.5 {
		t.Errorf("lastProgress = %v, want refreshed to 0.5", agg.lastProgress)
	}
	if ap.ETA <= 0 || ap.ETA > 24*time.Hour {
		t.Errorf("ETA = %v, want in (0, 24h]", ap.ETA)
	}
}

// TestProgressAggregator_RecomputeETA_ExponentialSmoothing covers the 70/30
// smoothing branch taken when a prior smoothed rate exists.
func TestProgressAggregator_RecomputeETA_ExponentialSmoothing(t *testing.T) {
	t.Parallel()
	agg := NewProgressAggregator(1)
	rewindAggregatorClock(agg, time.Second)
	agg.lastProgress = 0.25
	agg.progressRate = 0.5 // pre-existing smoothed rate

	ap := agg.Update(progress.ProgressUpdate{CalculatorIndex: 0, Value: 0.5})

	// instantRate = 0.25/timeSince <= 0.25 because timeSince >= 1s. The
	// smoothed rate must land strictly below the old rate but above its 70%
	// floor; dropping smoothing in either direction (keeping the old rate or
	// adopting the instant rate) escapes this band.
	if agg.progressRate >= 0.5 || agg.progressRate <= 0.35 {
		t.Errorf("smoothed progressRate = %v, want in (0.35, 0.5)", agg.progressRate)
	}
	if ap.ETA <= 0 {
		t.Errorf("ETA = %v, want > 0 with a positive smoothed rate", ap.ETA)
	}
}

// TestProgressAggregator_RecomputeETA_NonPositiveDeltaKeepsRate covers the
// branch where progress did not advance: the smoothed rate must survive but
// the update timestamp must still be refreshed.
func TestProgressAggregator_RecomputeETA_NonPositiveDeltaKeepsRate(t *testing.T) {
	t.Parallel()
	agg := NewProgressAggregator(1)
	agg.state.Update(0, 0.5)
	rewindAggregatorClock(agg, time.Second)
	agg.lastProgress = 0.5 // same as the incoming average: delta == 0
	agg.progressRate = 0.2

	before := agg.lastUpdate
	ap := agg.Update(progress.ProgressUpdate{CalculatorIndex: 0, Value: 0.5})

	if agg.progressRate != 0.2 {
		t.Errorf("progressRate = %v, want unchanged 0.2 when progress did not advance", agg.progressRate)
	}
	if !agg.lastUpdate.After(before) {
		t.Error("lastUpdate must be refreshed even without progress advance")
	}
	// ETA derives from the kept rate and fixed progress: (1-0.5)/0.2 = 2.5s.
	if ap.ETA < 2*time.Second || ap.ETA > 3*time.Second {
		t.Errorf("ETA = %v, want ~2.5s from the kept rate", ap.ETA)
	}
}

// TestProgressAggregator_RecomputeETA_RapidUpdatesSkipRateRefresh covers the
// branch where updates arrive faster than the 50ms gate: the rate refresh is
// skipped entirely but the ETA is still served from the existing rate.
func TestProgressAggregator_RecomputeETA_RapidUpdatesSkipRateRefresh(t *testing.T) {
	t.Parallel()
	agg := NewProgressAggregator(1)
	agg.startTime = agg.startTime.Add(-time.Second) // pass the 100ms elapsed gate
	// A future lastUpdate makes now.Sub(lastUpdate) negative regardless of
	// scheduling delays, deterministically keeping timeSinceUpdate below 50ms.
	agg.lastUpdate = time.Now().Add(time.Hour)
	agg.lastProgress = 0.1
	agg.progressRate = 0.4

	ap := agg.Update(progress.ProgressUpdate{CalculatorIndex: 0, Value: 0.6})

	if agg.progressRate != 0.4 {
		t.Errorf("progressRate = %v, want unchanged 0.4 (update arrived too soon)", agg.progressRate)
	}
	if agg.lastProgress != 0.1 {
		t.Errorf("lastProgress = %v, want unchanged 0.1 (rate refresh skipped)", agg.lastProgress)
	}
	// ETA reflects the existing rate with the fresh progress: (1-0.6)/0.4 = 1s.
	if ap.ETA < 900*time.Millisecond || ap.ETA > 1100*time.Millisecond {
		t.Errorf("ETA = %v, want ~1s", ap.ETA)
	}
}

func TestDrainChannel(t *testing.T) {
	t.Parallel()
	ch := make(chan progress.ProgressUpdate, 5)
	ch <- progress.ProgressUpdate{CalculatorIndex: 0, Value: 0.1}
	ch <- progress.ProgressUpdate{CalculatorIndex: 0, Value: 0.2}
	ch <- progress.ProgressUpdate{CalculatorIndex: 0, Value: 0.3}
	close(ch)

	DrainChannel(ch)
	// If we reach here without deadlock, the test passes
}

func TestDrainChannel_Empty(t *testing.T) {
	t.Parallel()
	ch := make(chan progress.ProgressUpdate)
	close(ch)

	DrainChannel(ch)
	// If we reach here without deadlock, the test passes
}
