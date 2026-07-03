package orchestration

import "testing"

// TestNewProgressState verifies ProgressState initialization.
func TestNewProgressState(t *testing.T) {
	t.Parallel()
	ps := NewProgressState(3)
	if ps.numCalculators != 3 {
		t.Errorf("numCalculators = %d, want 3", ps.numCalculators)
	}
	if len(ps.progresses) != 3 {
		t.Errorf("progresses length = %d, want 3", len(ps.progresses))
	}
	avg := ps.CalculateAverage()
	if avg != 0 {
		t.Errorf("initial average = %f, want 0", avg)
	}
}

// TestProgressStateUpdate verifies progress updates.
func TestProgressStateUpdate(t *testing.T) {
	t.Parallel()
	ps := NewProgressState(2)
	ps.Update(0, 0.5)
	ps.Update(1, 1.0)
	avg := ps.CalculateAverage()
	if avg != 0.75 {
		t.Errorf("average = %f, want 0.75", avg)
	}

	// Out-of-range indexes must be ignored, not panic or skew the average.
	ps.Update(-1, 0.9)
	ps.Update(2, 0.9)
	if got := ps.CalculateAverage(); got != 0.75 {
		t.Errorf("average after out-of-range updates = %f, want 0.75", got)
	}
}

// TestProgressStateZeroCalculators verifies edge case with zero calculators.
func TestProgressStateZeroCalculators(t *testing.T) {
	t.Parallel()
	ps := NewProgressState(0)
	avg := ps.CalculateAverage()
	if avg != 0 {
		t.Errorf("average = %f, want 0", avg)
	}
}
