package progress

import (
	"math"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// CalcTotalWork Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestCalcTotalWork tests the total work calculation for O(log n) algorithms.

// TestProgress_MonotonicLargeN is the A-10 guard: the reported fraction must
// stay finite, inside [0,1] and non-decreasing for bit counts well past the
// float64 overflow point of the old geometric formula (4^512 > MaxFloat64).
func TestProgress_MonotonicLargeN(t *testing.T) {
	t.Parallel()

	for _, numBits := range []int{64, 512, 2000, 100000} {
		t.Run("", func(t *testing.T) {
			t.Parallel()

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

			for i := numBits - 1; i >= 0; i-- {
				ReportStepProgress(reporter, &lastReported, i, numBits)
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
// ReportStepProgress Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestReportStepProgress(t *testing.T) {
	t.Parallel()

	t.Run("reports progress correctly", func(t *testing.T) {
		t.Parallel()
		numBits := 10

		var lastReported float64
		var receivedProgress []float64

		reporter := func(progress float64) {
			receivedProgress = append(receivedProgress, progress)
		}

		for i := numBits - 1; i >= 0; i-- {
			ReportStepProgress(reporter, &lastReported, i, numBits)
		}

		// Should have received at least initial and final progress
		if len(receivedProgress) < 2 {
			t.Errorf("expected at least 2 progress updates, got %d", len(receivedProgress))
		}

		// Final progress should be close to 1.0
		finalProgress := receivedProgress[len(receivedProgress)-1]
		if finalProgress < 0.99 {
			t.Errorf("final progress = %f, want >= 0.99", finalProgress)
		}
	})

	// numBits <= 0 describes no loop at all, so there is nothing to report.
	// Before audit L-01 the equivalent guard was a `totalWork > 0` test that
	// could never be false for a real call.
	t.Run("reports nothing for a zero-length loop", func(t *testing.T) {
		t.Parallel()
		var lastReported float64
		called := false

		ReportStepProgress(func(float64) { called = true }, &lastReported, 0, 0)

		if called {
			t.Error("nothing may be reported when numBits is 0")
		}
		if lastReported != 0 {
			t.Errorf("lastReported = %f, want 0", lastReported)
		}
	})

	// The first and last steps are always reported, whatever the coalescing
	// threshold would say, so a UI always sees a start and an end.
	t.Run("always reports the first and last step", func(t *testing.T) {
		t.Parallel()
		numBits := 64
		var lastReported float64
		var got []float64

		ReportStepProgress(func(p float64) { got = append(got, p) }, &lastReported, numBits-1, numBits)
		ReportStepProgress(func(p float64) { got = append(got, p) }, &lastReported, 0, numBits)

		if len(got) != 2 {
			t.Fatalf("got %d reports, want 2 (first and last step)", len(got))
		}
		if got[1] != 1.0 {
			t.Errorf("last step reported %v, want exactly 1.0", got[1])
		}
	})
}

// TestReportStepProgressMonotonic verifies progress is monotonically increasing.
func TestReportStepProgressMonotonic(t *testing.T) {
	t.Parallel()

	numBits := 20
	var lastReported float64
	var prevProgress float64

	reporter := func(progress float64) {
		if progress < prevProgress {
			t.Errorf("non-monotonic progress: prev=%f, current=%f", prevProgress, progress)
		}
		prevProgress = progress
	}

	for i := numBits - 1; i >= 0; i-- {
		ReportStepProgress(reporter, &lastReported, i, numBits)
	}
}
