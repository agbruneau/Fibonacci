package app

import (
	"testing"

	"github.com/agbruneau/FibGo/internal/config"
	"github.com/agbruneau/FibGo/internal/fibonacci/threshold"
)

// TestWireThresholdTuning pins the A2-04 wiring contract: after application
// startup the threshold package's tuning knobs must reflect
// config.DefaultThresholdTuning (the single source of truth). Before this
// wiring existed the contract was documented in threshold/manager.go and
// config/doc.go but never executed.
//
// Not t.Parallel(): it asserts on package-level state that other parallel
// tests could legitimately read.
func TestWireThresholdTuning(t *testing.T) {
	wireThresholdTuning()

	p := config.DefaultThresholdTuning
	if threshold.FFTSpeedupThreshold != p.FFTSpeedupThreshold {
		t.Errorf("FFTSpeedupThreshold = %v, want %v (config.DefaultThresholdTuning)",
			threshold.FFTSpeedupThreshold, p.FFTSpeedupThreshold)
	}
	if threshold.ParallelSpeedupThreshold != p.ParallelSpeedupThreshold {
		t.Errorf("ParallelSpeedupThreshold = %v, want %v",
			threshold.ParallelSpeedupThreshold, p.ParallelSpeedupThreshold)
	}
	if threshold.HysteresisMargin != p.HysteresisMargin {
		t.Errorf("HysteresisMargin = %v, want %v",
			threshold.HysteresisMargin, p.HysteresisMargin)
	}
}
