package calibration

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/agbru/fibcalc/internal/config"
	"github.com/agbru/fibcalc/internal/fibonacci"
)

// TestStrategyNames pins the human-facing names so log output stays
// stable; downstream tooling greps for these tokens.
func TestStrategyNames(t *testing.T) {
	t.Parallel()
	if got := NewFastStrategy().Name(); got != "fast" {
		t.Errorf("FastStrategy.Name() = %q, want %q", got, "fast")
	}
	if got := NewCompleteStrategy().Name(); got != "complete" {
		t.Errorf("CompleteStrategy.Name() = %q, want %q", got, "complete")
	}
}

// TestFastStrategy_Calibrate_ReturnsProfile checks that the fast tier
// runs without a calculator registry, returns a populated profile and
// a confidence in [0, 1].
func TestFastStrategy_Calibrate_ReturnsProfile(t *testing.T) {
	t.Parallel()

	out := &strings.Builder{}
	opts := StrategyOptions{
		BaseConfig: config.AppConfig{
			Threshold:         2048,
			FFTThreshold:      500000,
			StrassenThreshold: 256,
		},
		Out: out,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	profile, conf, err := NewFastStrategy().Calibrate(ctx, opts)
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

// TestCompleteStrategy_Calibrate_ReturnsProfile uses the same
// MockCalculator already exercised by TestAutoCalibrate to confirm
// CompleteStrategy returns a fully-populated profile with
// confidence = 1.0.
func TestCompleteStrategy_Calibrate_ReturnsProfile(t *testing.T) {
	t.Parallel()

	registry := map[string]fibonacci.Calculator{
		"fast":   &MockCalculator{name: "fast"},
		"matrix": &MockCalculator{name: "matrix"},
	}

	opts := StrategyOptions{
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

	profile, conf, err := NewCompleteStrategy().Calibrate(ctx, opts)
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

	opts := StrategyOptions{
		BaseConfig:         config.AppConfig{Timeout: 1 * time.Second},
		CalculatorRegistry: map[string]fibonacci.Calculator{}, // no "fast"
		Out:                io.Discard,
	}

	profile, conf, err := NewCompleteStrategy().Calibrate(context.Background(), opts)
	if !errors.Is(err, ErrMissingFastCalculator) {
		t.Errorf("Complete.Calibrate err = %v, want ErrMissingFastCalculator", err)
	}
	if profile != nil {
		t.Errorf("Complete.Calibrate profile = %v, want nil on error", profile)
	}
	if conf != 0 {
		t.Errorf("Complete.Calibrate confidence = %v, want 0 on error", conf)
	}
}

// TestCalibrationStrategy_InterfaceConformance is a compile-time
// guard that both implementations satisfy the interface.
func TestCalibrationStrategy_InterfaceConformance(t *testing.T) {
	t.Parallel()
	var _ CalibrationStrategy = (*FastStrategy)(nil)
	var _ CalibrationStrategy = (*CompleteStrategy)(nil)
}
