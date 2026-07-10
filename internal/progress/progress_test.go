package progress_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/agbruneau/FibGo/internal/progress"
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
			result := progress.CalcTotalWork(tc.numBits)
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

	prev := progress.CalcTotalWork(1)
	for bits := 2; bits <= 20; bits++ {
		current := progress.CalcTotalWork(bits)
		if current <= prev {
			t.Errorf("CalcTotalWork not monotonically increasing: bits=%d, prev=%f, current=%f",
				bits, prev, current)
		}
		prev = current
	}
}

// TestCalcTotalWork_LargeNumBits is the A-10 regression guard. The previous
// implementation materialized math.Pow(4, numBits), which overflows float64
// to +Inf for numBits >= 512 (4^512 == 2^1024 ~= MaxFloat64). A +Inf total
// makes every currentProgress = done/total collapse to 0 or NaN, freezing
// the progress bar exactly on the large calculations that are the project's
// core. numBits=2000 is well past the overflow boundary.
func TestCalcTotalWork_LargeNumBits(t *testing.T) {
	t.Parallel()

	for _, numBits := range []int{64, 512, 2000, 100000} {
		got := progress.CalcTotalWork(numBits)
		if math.IsInf(got, 0) {
			t.Fatalf("CalcTotalWork(%d) = +Inf; must remain finite", numBits)
		}
		if math.IsNaN(got) {
			t.Fatalf("CalcTotalWork(%d) = NaN; must be a real number", numBits)
		}
		if got <= 0 {
			t.Fatalf("CalcTotalWork(%d) = %v; want > 0", numBits, got)
		}
	}
}

// TestProgress_MonotonicLargeN drives the full step loop for large numBits
// values and asserts the reported progress is a monotonically increasing
// sequence contained in [0, 1] that reaches ~1.0 at the final step. This is
// the observable contract the UI depends on; before A-10 it produced a frozen
// (0 or NaN) sequence for numBits >= 512.
func TestProgress_MonotonicLargeN(t *testing.T) {
	t.Parallel()

	for _, numBits := range []int{64, 512, 2000, 100000} {
		t.Run(fmt.Sprintf("numBits=%d", numBits), func(t *testing.T) {
			t.Parallel()

			totalWork := progress.CalcTotalWork(numBits)
			powers := progress.PrecomputePowers4(numBits)

			var lastReported float64
			var last float64 = -1
			var sawAny bool
			reporter := func(p float64) {
				sawAny = true
				if math.IsNaN(p) || math.IsInf(p, 0) {
					t.Fatalf("numBits=%d: non-finite progress %v", numBits, p)
				}
				if p < 0 || p > 1 {
					t.Fatalf("numBits=%d: progress %v outside [0,1]", numBits, p)
				}
				if p < last {
					t.Fatalf("numBits=%d: non-monotonic progress %v -> %v", numBits, last, p)
				}
				last = p
			}

			workDone := 0.0
			for i := numBits - 1; i >= 0; i-- {
				workDone = progress.ReportStepProgress(reporter, &lastReported, totalWork, workDone, i, numBits, powers)
			}

			if !sawAny {
				t.Fatalf("numBits=%d: no progress reported", numBits)
			}
			if last < 0.99 {
				t.Fatalf("numBits=%d: final progress %v; want >= 0.99", numBits, last)
			}
		})
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
			result := progress.PrecomputePowers4(tc.numBits)
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

	powers := progress.PrecomputePowers4(10)
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
		totalWork := progress.CalcTotalWork(numBits)
		powers := progress.PrecomputePowers4(numBits)

		var lastReported float64
		var receivedProgress []float64

		reporter := func(progress float64) {
			receivedProgress = append(receivedProgress, progress)
		}

		workDone := float64(0)
		for i := numBits - 1; i >= 0; i-- {
			workDone = progress.ReportStepProgress(reporter, &lastReported, totalWork, workDone, i, numBits, powers)
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
		var reported bool
		powers := progress.PrecomputePowers4(5)

		// totalWork == 0 must suppress the reporter callback (the `totalWork >
		// 0` guard) without panicking, while still returning the accumulated
		// work-done total (powers[4] == 4^4 for step index numBits-1-i == 4).
		result := progress.ReportStepProgress(func(float64) { reported = true }, &lastReported, 0, 0, 0, 5, powers)
		if reported {
			t.Error("expected reporter not to be called when totalWork == 0")
		}
		if want := powers[4]; result != want {
			t.Errorf("ReportStepProgress() cumulative work = %v, want %v", result, want)
		}
	})
}

// TestReportStepProgressMonotonic verifies progress is monotonically increasing.
func TestReportStepProgressMonotonic(t *testing.T) {
	t.Parallel()

	numBits := 20
	totalWork := progress.CalcTotalWork(numBits)
	powers := progress.PrecomputePowers4(numBits)

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
		workDone = progress.ReportStepProgress(reporter, &lastReported, totalWork, workDone, i, numBits, powers)
	}
}
