// Package calibration_test exercises internal/calibration's exported API
// only. CalibrationStrategy, FastStrategy and CompleteStrategy are fully
// exported, so this file needs no unexported access.
package calibration_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/FibGo/internal/calibration"
	"github.com/agbruneau/FibGo/internal/config"
	"github.com/agbruneau/FibGo/internal/fibonacci"
)

// TestStrategyNames pins the human-facing names so log output stays
// stable; downstream tooling greps for these tokens.
func TestStrategyNames(t *testing.T) {
	t.Parallel()
	if got := calibration.NewFastStrategy().Name(); got != "fast" {
		t.Errorf("FastStrategy.Name() = %q, want %q", got, "fast")
	}
	if got := calibration.NewCompleteStrategy().Name(); got != "complete" {
		t.Errorf("CompleteStrategy.Name() = %q, want %q", got, "complete")
	}
}

// TestFastStrategy_Calibrate_ReturnsProfile checks that the fast tier runs
// without a calculator registry, returns a populated profile and a
// confidence in [0, 1].
func TestFastStrategy_Calibrate_ReturnsProfile(t *testing.T) {
	t.Parallel()

	out := &strings.Builder{}
	opts := calibration.StrategyOptions{
		BaseConfig: config.AppConfig{
			Threshold:         2048,
			FFTThreshold:      500000,
			StrassenThreshold: 256,
		},
		Out: out,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	profile, conf, err := calibration.NewFastStrategy().Calibrate(ctx, opts)
	if err != nil {
		t.Fatalf("Fast.Calibrate err = %v, want nil", err)
	}
	if profile == nil {
		t.Fatal("Fast.Calibrate profile = nil, want non-nil")
	}
	if conf < 0 || conf > 1 {
		t.Errorf("Fast.Calibrate confidence = %v, want [0,1]", conf)
	}
	if profile.OptimalParallelThreshold == 0 {
		t.Error("Fast.Calibrate did not set OptimalParallelThreshold")
	}
	if profile.OptimalFFTThreshold == 0 {
		t.Error("Fast.Calibrate did not set OptimalFFTThreshold")
	}
	// Strassen must come from BaseConfig (micro-bench does not test it).
	if profile.OptimalStrassenThreshold != opts.BaseConfig.StrassenThreshold {
		t.Errorf("Fast.Calibrate OptimalStrassenThreshold = %d, want %d (from BaseConfig)",
			profile.OptimalStrassenThreshold, opts.BaseConfig.StrassenThreshold)
	}
	if !strings.Contains(out.String(), "Quick calibration") {
		t.Errorf("Fast.Calibrate output should mention 'Quick calibration'; got %q", out.String())
	}
}

// TestCompleteStrategy_Calibrate_ReturnsProfile uses MockCalculator to
// confirm CompleteStrategy returns a fully-populated profile with
// confidence = 1.0.
func TestCompleteStrategy_Calibrate_ReturnsProfile(t *testing.T) {
	t.Parallel()

	registry := map[string]fibonacci.Calculator{
		"fast":   calibration.NewMockCalculator("fast"),
		"matrix": calibration.NewMockCalculator("matrix"),
	}

	opts := calibration.StrategyOptions{
		BaseConfig: config.AppConfig{
			Timeout:           1 * time.Second,
			Threshold:         4096,
			FFTThreshold:      750000,
			StrassenThreshold: 256,
		},
		CalculatorRegistry: registry,
		Out:                io.Discard,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	profile, conf, err := calibration.NewCompleteStrategy().Calibrate(ctx, opts)
	if err != nil {
		t.Fatalf("Complete.Calibrate err = %v, want nil", err)
	}
	if profile == nil {
		t.Fatal("Complete.Calibrate profile = nil, want non-nil")
	}
	if conf != 1.0 {
		t.Errorf("Complete.Calibrate confidence = %v, want 1.0", conf)
	}
	if profile.Confidence != 1.0 {
		t.Errorf("Complete.Calibrate profile.Confidence = %v, want 1.0", profile.Confidence)
	}
	if profile.OptimalParallelThreshold == 0 {
		t.Error("Complete.Calibrate did not set OptimalParallelThreshold")
	}
}

// TestCompleteStrategy_Calibrate_MissingFastCalculator confirms the
// strategy fails fast (rather than panicking) when the required "fast"
// entry is absent. AutoCalibrateWithProfile guards against this case
// upstream, but the strategy reports it explicitly for direct callers.
func TestCompleteStrategy_Calibrate_MissingFastCalculator(t *testing.T) {
	t.Parallel()

	opts := calibration.StrategyOptions{
		BaseConfig:         config.AppConfig{Timeout: 1 * time.Second},
		CalculatorRegistry: map[string]fibonacci.Calculator{}, // no "fast"
		Out:                io.Discard,
	}

	profile, conf, err := calibration.NewCompleteStrategy().Calibrate(context.Background(), opts)
	if !errors.Is(err, calibration.ErrMissingFastCalculator) {
		t.Errorf("Complete.Calibrate err = %v, want ErrMissingFastCalculator", err)
	}
	if profile != nil {
		t.Errorf("Complete.Calibrate profile = %v, want nil on error", profile)
	}
	if conf != 0 {
		t.Errorf("Complete.Calibrate confidence = %v, want 0 on error", conf)
	}
}

// TestCalibrationStrategy_InterfaceConformance is a compile-time guard
// that both implementations satisfy the interface.
func TestCalibrationStrategy_InterfaceConformance(t *testing.T) {
	t.Parallel()
	var _ calibration.CalibrationStrategy = (*calibration.FastStrategy)(nil)
	var _ calibration.CalibrationStrategy = (*calibration.CompleteStrategy)(nil)
}
