// CompleteStrategy: full-sweep tier of the calibration orchestrator.
// It wraps calibrationRunner (runner.go) and exercises each candidate
// threshold against the real fibonacci.Calculator.

package calibration

import (
	"context"
	"errors"
	"time"
)

// ErrMissingFastCalculator is returned by CompleteStrategy.Calibrate
// when opts.CalculatorRegistry does not contain a "fast" entry.
// AutoCalibrateWithProfile guards against this case earlier, but the
// strategy reports it explicitly so callers exercising it directly
// (e.g. tests) get a clear failure mode rather than a nil-deref panic.
var ErrMissingFastCalculator = errors.New("calibration: 'fast' calculator missing from registry")

// ErrNoUsefulResults is returned when every candidate threshold timed
// out or errored, leaving CompleteStrategy with no signal to report.
// The orchestrator surfaces this as a calibration failure (ok=false)
// to the caller.
var ErrNoUsefulResults = errors.New("calibration: no usable timings collected")

// CompleteStrategy runs the runner-based threshold sweep: it walks the
// quick parallel/FFT/Strassen candidate sets (adaptive.go) and times
// each one against the real fibonacci.Calculator, picking the best
// timing per dimension. It is significantly slower than FastStrategy
// (multiple seconds vs ~100 ms) but provides the authoritative value
// recorded with full confidence (1.0).
type CompleteStrategy struct{}

// NewCompleteStrategy returns a ready-to-use CompleteStrategy.
// Stateless; constructor mirrors NewFastStrategy.
func NewCompleteStrategy() *CompleteStrategy { return &CompleteStrategy{} }

// Name implements CalibrationStrategy.
func (CompleteStrategy) Name() string { return "complete" }

// Calibrate implements CalibrationStrategy.
//
// The behavior matches the legacy runFullCalibration helper exactly:
//   - require a "fast" calculator in opts.CalculatorRegistry;
//   - sweep parallel, then FFT (using the discovered parallel value),
//     then optionally Strassen if "matrix" is registered;
//   - if every dimension timed out, fail with ErrNoUsefulResults so the
//     orchestrator can return ok=false to the caller;
//   - otherwise, package the discovered values into a *CalibrationProfile
//     with Confidence=1.0.
//
// Persistence (SaveProfile) and end-user "Auto-calibration: ..." output
// remain the orchestrator's responsibility — the strategy stays a pure
// measurement step.
func (s CompleteStrategy) Calibrate(ctx context.Context, opts StrategyOptions) (*CalibrationProfile, Confidence, error) {
	fastCalc := opts.CalculatorRegistry["fast"]
	if fastCalc == nil {
		return nil, 0, ErrMissingFastCalculator
	}

	runner := newCalibrationRunner(ctx, opts.BaseConfig.Timeout)

	bestPar, bestParDur := runner.findBestParallelThreshold(fastCalc, opts.BaseConfig.Threshold)
	bestFFT, bestFFTDur := runner.findBestFFTThreshold(fastCalc, bestPar, opts.BaseConfig.FFTThreshold)

	bestStrassen := opts.BaseConfig.StrassenThreshold
	bestStrassenDur := time.Duration(1<<63 - 1)
	if matCalc := opts.CalculatorRegistry["matrix"]; matCalc != nil {
		bestStrassen, bestStrassenDur = runner.findBestStrassenThreshold(matCalc, bestPar, opts.BaseConfig.StrassenThreshold)
	}

	// applyCalibrationResults already encodes the "at least one dimension
	// produced a real timing" rule via its maxDuration sentinel check;
	// reuse it here so the legacy fallback semantics are preserved.
	updated, ok := applyCalibrationResults(opts.BaseConfig, bestPar, bestParDur, bestFFT, bestFFTDur, bestStrassen, bestStrassenDur)
	if !ok {
		return nil, 0, ErrNoUsefulResults
	}

	profile := NewProfile()
	profile.OptimalParallelThreshold = updated.Threshold
	profile.OptimalFFTThreshold = updated.FFTThreshold
	profile.OptimalStrassenThreshold = updated.StrassenThreshold
	profile.CalibrationN = CalibrationN
	profile.Confidence = 1.0

	return profile, 1.0, nil
}
