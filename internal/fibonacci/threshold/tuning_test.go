package threshold

import "testing"

// TestSetTuning covers the threshold -> config seam (manager.go SetTuning),
// which had 0% coverage (audit F-007). SetTuning writes package-level tuning
// globals, so this test is deliberately NOT parallel and restores the originals
// via defer, honouring the A2-04 single-writer-before-use invariant documented
// at manager.go:33-39 (no concurrent reader runs during the sequential phase).
func TestSetTuning(t *testing.T) {
	origFFT := FFTSpeedupThreshold
	origParallel := ParallelSpeedupThreshold
	origHyst := HysteresisMargin
	origFFTFloor := minFFTThresholdFloor
	origParallelFloor := minParallelThresholdFloor
	defer func() {
		FFTSpeedupThreshold = origFFT
		ParallelSpeedupThreshold = origParallel
		HysteresisMargin = origHyst
		minFFTThresholdFloor = origFFTFloor
		minParallelThresholdFloor = origParallelFloor
	}()

	SetTuning(Tuning{
		FFTSpeedupThreshold:      2.5,
		ParallelSpeedupThreshold: 1.9,
		HysteresisMargin:         0.42,
		MinFFTThreshold:          200_000,
		MinParallelThreshold:     2048,
	})

	if FFTSpeedupThreshold != 2.5 {
		t.Errorf("FFTSpeedupThreshold = %v, want 2.5", FFTSpeedupThreshold)
	}
	if ParallelSpeedupThreshold != 1.9 {
		t.Errorf("ParallelSpeedupThreshold = %v, want 1.9", ParallelSpeedupThreshold)
	}
	if HysteresisMargin != 0.42 {
		t.Errorf("HysteresisMargin = %v, want 0.42", HysteresisMargin)
	}
	if minFFTThresholdFloor != 200_000 {
		t.Errorf("minFFTThresholdFloor = %d, want 200000", minFFTThresholdFloor)
	}
	if minParallelThresholdFloor != 2048 {
		t.Errorf("minParallelThresholdFloor = %d, want 2048", minParallelThresholdFloor)
	}

	// Zero-valued fields must NOT overwrite existing values (the guard clauses).
	SetTuning(Tuning{})
	if FFTSpeedupThreshold != 2.5 {
		t.Errorf("zero-field SetTuning overwrote FFTSpeedupThreshold: got %v, want 2.5", FFTSpeedupThreshold)
	}
	if minParallelThresholdFloor != 2048 {
		t.Errorf("zero-field SetTuning overwrote minParallelThresholdFloor: got %d, want 2048", minParallelThresholdFloor)
	}
}
