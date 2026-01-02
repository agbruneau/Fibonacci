package calibration

import (
	"context"
	"time"

	"github.com/agbru/fibcalc/internal/fibonacci"
)

// calibrationRunner encapsulates the trial run logic for calibration.
type calibrationRunner struct {
	ctx      context.Context
	perTrial time.Duration
}

// newCalibrationRunner creates a new calibration runner.
func newCalibrationRunner(ctx context.Context, timeout time.Duration) *calibrationRunner {
	perTrial := timeout / 6
	if perTrial < 2*time.Second {
		perTrial = 2 * time.Second
	}
	return &calibrationRunner{ctx: ctx, perTrial: perTrial}
}

// runTrial executes a single calibration trial with the given calculator and options.
//
// Parameters:
//   - calc: The calculator to use for the trial.
//   - opts: The options for the calculation.
//
// Returns:
//   - time.Duration: The duration of the calculation.
//   - error: An error if the calculation failed or timed out.
func (r *calibrationRunner) runTrial(calc fibonacci.Calculator, opts fibonacci.Options) (duration time.Duration, err error) {
	ctx, cancel := context.WithTimeout(r.ctx, r.perTrial)
	defer cancel()
	start := time.Now()
	_, err = calc.Calculate(ctx, nil, 0, fibonacci.CalibrationN, opts)
	return time.Since(start), err
}

// findBestParallelThreshold finds the optimal parallel threshold.
//
// Parameters:
//   - calc: The calculator to use for testing.
//   - defaultThreshold: The default threshold to use if no better one is found.
//
// Returns:
//   - int: The best parallel threshold found.
//   - time.Duration: The duration achieved with the best threshold.
func (r *calibrationRunner) findBestParallelThreshold(calc fibonacci.Calculator, defaultThreshold int) (threshold int, duration time.Duration) {
	candidates := GenerateQuickParallelThresholds()
	best := defaultThreshold
	bestDur := time.Duration(1<<63 - 1)

	for _, cand := range candidates {
		dur, err := r.runTrial(calc, fibonacci.Options{ParallelThreshold: cand, FFTThreshold: 0})
		if err != nil {
			continue
		}
		if dur < bestDur {
			bestDur, best = dur, cand
		}
	}
	return best, bestDur
}

// findBestFFTThreshold finds the optimal FFT threshold.
//
// Parameters:
//   - calc: The calculator to use for testing.
//   - parallelThreshold: The parallel threshold to use during testing.
//   - defaultThreshold: The default threshold to use if no better one is found.
//
// Returns:
//   - int: The best FFT threshold found.
//   - time.Duration: The duration achieved with the best threshold.
func (r *calibrationRunner) findBestFFTThreshold(calc fibonacci.Calculator, parallelThreshold, defaultThreshold int) (threshold int, duration time.Duration) {
	candidates := GenerateQuickFFTThresholds()
	best := defaultThreshold
	bestDur := time.Duration(1<<63 - 1)

	for _, cand := range candidates {
		dur, err := r.runTrial(calc, fibonacci.Options{ParallelThreshold: parallelThreshold, FFTThreshold: cand})
		if err != nil {
			continue
		}
		if dur < bestDur {
			bestDur, best = dur, cand
		}
	}
	return best, bestDur
}

// findBestStrassenThreshold finds the optimal Strassen threshold.
//
// Parameters:
//   - calc: The calculator to use for testing.
//   - parallelThreshold: The parallel threshold to use during testing.
//   - defaultThreshold: The default threshold to use if no better one is found.
//
// Returns:
//   - int: The best Strassen threshold found.
//   - time.Duration: The duration achieved with the best threshold.
func (r *calibrationRunner) findBestStrassenThreshold(calc fibonacci.Calculator, parallelThreshold, defaultThreshold int) (threshold int, duration time.Duration) {
	candidates := GenerateQuickStrassenThresholds()
	best := defaultThreshold
	bestDur := time.Duration(1<<63 - 1)

	for _, cand := range candidates {
		dur, err := r.runTrial(calc, fibonacci.Options{ParallelThreshold: parallelThreshold, StrassenThreshold: cand})
		if err != nil {
			continue
		}
		if dur < bestDur {
			bestDur, best = dur, cand
		}
	}
	return best, bestDur
}
