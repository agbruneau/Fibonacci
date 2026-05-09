// FastStrategy: micro-benchmark-driven tier of the calibration
// orchestrator. It wraps QuickCalibrate (microbench.go) and packages
// its ThresholdResults into a *CalibrationProfile so the orchestrator
// can treat it interchangeably with CompleteStrategy.

package calibration

import (
	"context"
	"fmt"
	"time"

	"github.com/agbru/fibcalc/internal/fibonacci"
	"github.com/agbru/fibcalc/internal/ui"
)

// FastStrategy estimates thresholds via the in-process micro-benchmarks
// in microbench.go (~100 ms wall time). It does not touch any
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
// and prints a single informational line on opts.Out matching the
// historical "Quick calibration (..): ..." message so existing tests
// and end-user output stay identical.
//
// On context cancellation or QuickCalibrate failure, Calibrate returns
// (nil, 0, err) so the orchestrator can escalate to CompleteStrategy.
func (s FastStrategy) Calibrate(ctx context.Context, opts StrategyOptions) (*CalibrationProfile, Confidence, error) {
	results, err := QuickCalibrate(ctx)
	if err != nil {
		return nil, 0, err
	}

	profile := NewProfile()
	profile.OptimalParallelThreshold = results.ParallelThreshold
	profile.OptimalFFTThreshold = results.FFTThreshold
	// Micro-benchmarks do not test Strassen; preserve the caller's value.
	profile.OptimalStrassenThreshold = opts.BaseConfig.StrassenThreshold
	profile.CalibrationN = fibonacci.CalibrationN
	profile.CalibrationTime = results.Duration.String()
	profile.Confidence = float64(results.Confidence)

	// Match the historical "Quick calibration (..): ..." message so
	// existing user-visible output and integration tests are preserved.
	fmt.Fprintf(opts.Out,
		"%sQuick calibration%s (%v): parallelism=%s%d%s bits, FFT=%s%d%s bits (confidence: %.0f%%)\n",
		ui.ColorGreen(), ui.ColorReset(),
		results.Duration.Round(time.Millisecond),
		ui.ColorYellow(), profile.OptimalParallelThreshold, ui.ColorReset(),
		ui.ColorYellow(), profile.OptimalFFTThreshold, ui.ColorReset(),
		results.Confidence*100)

	return profile, Confidence(results.Confidence), nil
}
