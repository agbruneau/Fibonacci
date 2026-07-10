// This file stays package calibration (white-box) because it exercises
// newCalibrationRunner and the unexported findBest* methods directly. See
// PLAN.md §3.3a.
package calibration

import (
	"context"
	"testing"
	"time"
)

// TestCalibrationRunner drives all three findBest* methods of
// calibrationRunner over the same MockCalculator, table-driven since each
// case only differs in which threshold dimension is being searched.
func TestCalibrationRunner(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	runner := newCalibrationRunner(ctx, 1*time.Second)
	calc := NewMockCalculator("fast")

	cases := []struct {
		name string
		run  func() (threshold int, duration time.Duration)
	}{
		{"parallel", func() (int, time.Duration) { return runner.findBestParallelThreshold(calc, 4096) }},
		{"fft", func() (int, time.Duration) { return runner.findBestFFTThreshold(calc, 4096, 1000000) }},
		{"strassen", func() (int, time.Duration) { return runner.findBestStrassenThreshold(calc, 4096, 256) }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			threshold, duration := tc.run()
			if threshold == 0 {
				t.Errorf("findBest%sThreshold: threshold = 0, want non-zero", tc.name)
			}
			if duration == 0 {
				t.Errorf("findBest%sThreshold: duration = 0, want non-zero", tc.name)
			}
		})
	}
}
