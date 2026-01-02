package fibonacci

import (
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// CalcTotalWork Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestCalcTotalWork tests the total work calculation for O(log n) algorithms.
func TestCalcTotalWork(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		numBits  int
		wantZero bool
	}{
		{"zero bits", 0, true},
		{"one bit", 1, false},
		{"small bits", 10, false},
		{"medium bits", 32, false},
		{"large bits", 64, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := CalcTotalWork(tc.numBits)
			if tc.wantZero {
				if result != 0 {
					t.Errorf("CalcTotalWork(%d) = %f, want 0", tc.numBits, result)
				}
			} else {
				if result <= 0 {
					t.Errorf("CalcTotalWork(%d) = %f, want > 0", tc.numBits, result)
				}
			}
		})
	}
}

// TestCalcTotalWorkMonotonic verifies that work increases with more bits.
func TestCalcTotalWorkMonotonic(t *testing.T) {
	t.Parallel()

	prev := CalcTotalWork(1)
	for bits := 2; bits <= 20; bits++ {
		current := CalcTotalWork(bits)
		if current <= prev {
			t.Errorf("CalcTotalWork not monotonically increasing: bits=%d, prev=%f, current=%f",
				bits, prev, current)
		}
		prev = current
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// PrecomputePowers4 Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestPrecomputePowers4 tests power-of-4 precomputation.
func TestPrecomputePowers4(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		numBits int
		wantNil bool
	}{
		{"zero returns nil", 0, true},
		{"negative returns nil", -5, true},
		{"one bit", 1, false},
		{"multiple bits", 10, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := PrecomputePowers4(tc.numBits)
			if tc.wantNil {
				if result != nil {
					t.Errorf("PrecomputePowers4(%d) = %v, want nil", tc.numBits, result)
				}
			} else {
				if result == nil {
					t.Fatalf("PrecomputePowers4(%d) = nil, want non-nil", tc.numBits)
				}
				if len(result) != tc.numBits {
					t.Errorf("PrecomputePowers4(%d) len = %d, want %d", tc.numBits, len(result), tc.numBits)
				}
			}
		})
	}
}

// TestPrecomputePowers4Values verifies the computed values are correct.
func TestPrecomputePowers4Values(t *testing.T) {
	t.Parallel()

	powers := PrecomputePowers4(10)
	if powers == nil {
		t.Fatal("PrecomputePowers4(10) returned nil")
	}

	// Verify first element is 4^0 = 1
	if powers[0] != 1.0 {
		t.Errorf("powers[0] = %f, want 1.0", powers[0])
	}

	// Verify each subsequent element is 4 times the previous
	for i := 1; i < len(powers); i++ {
		expected := powers[i-1] * 4.0
		if powers[i] != expected {
			t.Errorf("powers[%d] = %f, want %f", i, powers[i], expected)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ReportStepProgress Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestReportStepProgress tests progress reporting logic.
func TestReportStepProgress(t *testing.T) {
	t.Parallel()

	t.Run("reports progress correctly", func(t *testing.T) {
		t.Parallel()
		numBits := 10
		totalWork := CalcTotalWork(numBits)
		powers := PrecomputePowers4(numBits)

		var lastReported float64
		var receivedProgress []float64

		reporter := func(progress float64) {
			receivedProgress = append(receivedProgress, progress)
		}

		workDone := float64(0)
		for i := numBits - 1; i >= 0; i-- {
			workDone = ReportStepProgress(reporter, &lastReported, totalWork, workDone, i, numBits, powers)
		}

		// Should have received at least initial and final progress
		if len(receivedProgress) < 2 {
			t.Errorf("expected at least 2 progress updates, got %d", len(receivedProgress))
		}

		// Final progress should be close to 1.0
		if len(receivedProgress) > 0 {
			finalProgress := receivedProgress[len(receivedProgress)-1]
			if finalProgress < 0.99 {
				t.Errorf("final progress = %f, want >= 0.99", finalProgress)
			}
		}
	})

	t.Run("handles zero total work", func(t *testing.T) {
		t.Parallel()
		var lastReported float64
		powers := PrecomputePowers4(5)

		// Should not panic with zero total work
		result := ReportStepProgress(func(float64) {}, &lastReported, 0, 0, 0, 5, powers)
		if result == 0 {
			// Expected: work of step should still be calculated
		}
	})
}

// TestReportStepProgressMonotonic verifies progress is monotonically increasing.
func TestReportStepProgressMonotonic(t *testing.T) {
	t.Parallel()

	numBits := 20
	totalWork := CalcTotalWork(numBits)
	powers := PrecomputePowers4(numBits)

	var lastReported float64
	var prevProgress float64

	reporter := func(progress float64) {
		if progress < prevProgress {
			t.Errorf("non-monotonic progress: prev=%f, current=%f", prevProgress, progress)
		}
		prevProgress = progress
	}

	workDone := float64(0)
	for i := numBits - 1; i >= 0; i-- {
		workDone = ReportStepProgress(reporter, &lastReported, totalWork, workDone, i, numBits, powers)
	}
}
