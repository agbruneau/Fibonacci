package config

import "testing"

// TestApplyAdaptiveThresholds_ZeroDefaults verifies that zero values are
// replaced by heuristic estimates while non-zero overrides are preserved.
func TestApplyAdaptiveThresholds_ZeroDefaults(t *testing.T) {
	t.Parallel()

	cfg := AppConfig{
		Threshold:         0,
		FFTThreshold:      0,
		StrassenThreshold: 0,
	}
	got := ApplyAdaptiveThresholds(cfg)

	if got.Threshold <= 0 {
		t.Errorf("Threshold: expected a positive heuristic value when zero, got %d", got.Threshold)
	}
	if got.FFTThreshold <= 0 {
		t.Errorf("FFTThreshold: expected a positive heuristic value when zero, got %d", got.FFTThreshold)
	}
	if got.StrassenThreshold <= 0 {
		t.Errorf("StrassenThreshold: expected a positive heuristic value when zero, got %d", got.StrassenThreshold)
	}
}

// TestApplyAdaptiveThresholds_PreservesOverrides verifies that explicit
// non-zero values are passed through unchanged.
func TestApplyAdaptiveThresholds_PreservesOverrides(t *testing.T) {
	t.Parallel()

	cfg := AppConfig{
		Threshold:         1234,
		FFTThreshold:      567890,
		StrassenThreshold: 42,
	}
	got := ApplyAdaptiveThresholds(cfg)

	if got.Threshold != 1234 {
		t.Errorf("Threshold override not preserved: want 1234, got %d", got.Threshold)
	}
	if got.FFTThreshold != 567890 {
		t.Errorf("FFTThreshold override not preserved: want 567890, got %d", got.FFTThreshold)
	}
	if got.StrassenThreshold != 42 {
		t.Errorf("StrassenThreshold override not preserved: want 42, got %d", got.StrassenThreshold)
	}
}

// TestApplyAdaptiveThresholds_PartialOverride verifies that only zero-valued
// fields are adapted while explicit fields remain untouched.
func TestApplyAdaptiveThresholds_PartialOverride(t *testing.T) {
	t.Parallel()

	cfg := AppConfig{
		Threshold:         99, // explicit
		FFTThreshold:      0,  // zero → adapted
		StrassenThreshold: 77, // explicit
	}
	got := ApplyAdaptiveThresholds(cfg)

	if got.Threshold != 99 {
		t.Errorf("Threshold: want 99, got %d", got.Threshold)
	}
	if got.FFTThreshold <= 0 {
		t.Errorf("FFTThreshold: expected adapted value, got %d", got.FFTThreshold)
	}
	if got.StrassenThreshold != 77 {
		t.Errorf("StrassenThreshold: want 77, got %d", got.StrassenThreshold)
	}
}

// TestEstimateOptimalParallelThreshold_NonNegative verifies the estimator
// returns a non-negative value that matches the heuristic function.
func TestEstimateOptimalParallelThreshold_NonNegative(t *testing.T) {
	t.Parallel()

	got := EstimateOptimalParallelThreshold()
	if got < 0 {
		t.Errorf("EstimateOptimalParallelThreshold returned negative value: %d", got)
	}
	// The wrapper must agree with the inner implementation on the detected host.
	want := estimateParallelThresholdForHeuristic(DetectHardwareHeuristic())
	if got != want {
		t.Errorf("EstimateOptimalParallelThreshold = %d; estimateParallelThresholdForHeuristic(detected) = %d", got, want)
	}
}

// TestEstimateOptimalFFTThreshold_Positive verifies the estimator returns a
// positive threshold consistent with the heuristic function.
func TestEstimateOptimalFFTThreshold_Positive(t *testing.T) {
	t.Parallel()

	got := EstimateOptimalFFTThreshold()
	if got <= 0 {
		t.Errorf("EstimateOptimalFFTThreshold returned non-positive value: %d", got)
	}
	want := estimateFFTThresholdForHeuristic(DetectHardwareHeuristic())
	if got != want {
		t.Errorf("EstimateOptimalFFTThreshold = %d; estimateFFTThresholdForHeuristic(detected) = %d", got, want)
	}
}

// TestEstimateOptimalStrassenThreshold_Positive verifies the estimator returns
// a positive threshold consistent with the heuristic function.
func TestEstimateOptimalStrassenThreshold_Positive(t *testing.T) {
	t.Parallel()

	got := EstimateOptimalStrassenThreshold()
	if got <= 0 {
		t.Errorf("EstimateOptimalStrassenThreshold returned non-positive value: %d", got)
	}
	want := estimateStrassenThresholdForHeuristic(DetectHardwareHeuristic())
	if got != want {
		t.Errorf("EstimateOptimalStrassenThreshold = %d; estimateStrassenThresholdForHeuristic(detected) = %d", got, want)
	}
}
