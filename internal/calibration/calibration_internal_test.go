// This file stays package calibration (white-box) because it exercises
// profileMaxAgeFromEnv, tryFastThenEscalate and applyCalibrationResults
// directly -- none of which have an exported equivalent that reaches the
// same code path. See PLAN.md §3.3a.
package calibration

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/agbruneau/FibGo/internal/config"
)

// TestProfileMaxAgeFromEnv is NOT parallel: it mutates the process-wide
// FIBCALC_PROFILE_MAX_AGE environment variable via t.Setenv, and Go's
// testing package forbids combining t.Setenv with t.Parallel outright (the
// call panics). Subtests are parallel-safe against each other only because
// the whole test stays out of the package's parallel phase.
func TestProfileMaxAgeFromEnv(t *testing.T) {
	cases := []struct {
		name string
		env  string
		want time.Duration
	}{
		{"unset returns default", "", DefaultProfileMaxAge},
		{"valid override is honored", "30m", 30 * time.Minute},
		{"invalid value falls back to default", "not-a-duration", DefaultProfileMaxAge},
		{"non-positive value falls back to default", "0s", DefaultProfileMaxAge},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(ProfileMaxAgeEnv, tc.env)
			if got := profileMaxAgeFromEnv(); got != tc.want {
				t.Errorf("profileMaxAgeFromEnv() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestTryFastThenEscalate_InvalidMeasurementsEscalates is the FIB-03 guard
// test: with a pre-canceled context, FastStrategy.Calibrate cannot collect
// any timing (runParallelTests returns no results), so analyzeResults must
// report confidence 0 and tryFastThenEscalate must signal escalation
// (ok=false) rather than accepting an inflated fallback confidence. Before
// the fix, the >0 defaults in find*Crossover added +0.2/+0.2 bonuses even
// with zero real data, making escalation unreachable via this path.
func TestTryFastThenEscalate_InvalidMeasurementsEscalates(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-canceled: no measurement can complete

	stratOpts := StrategyOptions{
		BaseConfig: config.AppConfig{StrassenThreshold: 256},
		Out:        io.Discard,
	}

	if _, ok := tryFastThenEscalate(ctx, stratOpts, ""); ok {
		t.Error("tryFastThenEscalate should signal escalation (ok=false) when no valid measurement was collected")
	}
}

func TestApplyCalibrationResults(t *testing.T) {
	t.Parallel()
	cfg := config.AppConfig{}
	updated, ok := applyCalibrationResults(cfg, 4096, 10*time.Millisecond, 1000000, 10*time.Millisecond, 256, 10*time.Millisecond)
	if !ok {
		t.Error("applyCalibrationResults should return true")
	}
	if updated.Threshold != 4096 {
		t.Errorf("Threshold = %d, want 4096", updated.Threshold)
	}
}
