// FastStrategy: micro-benchmark-driven tier of the calibration
// orchestrator. It wraps QuickCalibrate (microbench.go) and packages
// its ThresholdResults into a *CalibrationProfile so the orchestrator
// can treat it interchangeably with CompleteStrategy.

package calibration

import (
	"context"
	"time"
)

// FastStrategy estimates thresholds via the in-process micro-benchmarks
// in microbench.go. Its wall time is whatever those micro-benchmarks cost
// on the host — sub-second by construction (fixed, small candidate sets),
// but no figure is recorded in the repo; the actual duration of each run is
// reported in CalibrationProfile.CalibrationTime. It does not touch any
// fibonacci.Calculator: the multiplications it times go straight
// through bigfft, which is sufficient to derive the FFT crossover and
// the parallel crossover used as fallback when no cached profile is
// available.
//
// The Strassen threshold is left at the caller-provided default
// because micro-benchmarks intentionally do not exercise matrix
// multiplication; only CompleteStrategy can tune it.
type FastStrategy struct{}

// NewFastStrategy returns a ready-to-use FastStrategy. The type is
// stateless; the constructor exists for symmetry with
// NewCompleteStrategy and to keep call sites uniform.
func NewFastStrategy() *FastStrategy { return &FastStrategy{} }

// Name implements CalibrationStrategy.
func (FastStrategy) Name() string { return "fast" }

// Calibrate implements CalibrationStrategy.
//
// It runs QuickCalibrate, converts the produced ThresholdResults into
// a *CalibrationProfile (preserving the original confidence score),
// and reports a single informational line through opts.Reporter. The wording of
// the historical "Quick calibration (..): ..." message is preserved; the color
// is now the adapter's decision (audit ARC-01).
//
// On context cancellation or QuickCalibrate failure, Calibrate returns
// (nil, 0, err) so the orchestrator can escalate to CompleteStrategy.
func (s FastStrategy) Calibrate(ctx context.Context, opts StrategyOptions) (*CalibrationProfile, Confidence, error) {
	results, err := NewMicroBenchmark().RunQuick(ctx)
	if err != nil {
		return nil, 0, err
	}

	profile := NewProfile()
	profile.OptimalParallelThreshold = results.ParallelThreshold
	profile.OptimalFFTThreshold = results.FFTThreshold
	// Micro-benchmarks do not test Strassen; preserve the caller's value.
	profile.OptimalStrassenThreshold = opts.BaseConfig.StrassenThreshold
	profile.CalibrationN = CalibrationN
	profile.CalibrationTime = results.Duration.String()
	profile.Confidence = float64(results.Confidence)

	opts.Reporter.Notice("Quick calibration (%v): parallelism=%d bits, FFT=%d bits (confidence: %.0f%%)",
		results.Duration.Round(time.Millisecond),
		profile.OptimalParallelThreshold,
		profile.OptimalFFTThreshold,
		results.Confidence*100)

	return profile, Confidence(results.Confidence), nil
}
