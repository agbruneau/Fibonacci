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
	if agg.NumCalculators() != 3 {
		t.Errorf("expected NumCalculators()=3, got %d", agg.NumCalculators())
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

func TestDrainChannel(t *testing.T) {
	ch := make(chan progress.ProgressUpdate, 5)
	ch <- progress.ProgressUpdate{CalculatorIndex: 0, Value: 0.1}
	ch <- progress.ProgressUpdate{CalculatorIndex: 0, Value: 0.2}
	ch <- progress.ProgressUpdate{CalculatorIndex: 0, Value: 0.3}
	close(ch)

	DrainChannel(ch)
	// If we reach here without deadlock, the test passes
}

func TestDrainChannel_Empty(t *testing.T) {
	ch := make(chan progress.ProgressUpdate)
	close(ch)

	DrainChannel(ch)
	// If we reach here without deadlock, the test passes
}
