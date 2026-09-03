package calibration

import (
	"context"
	"math"
	"time"

	"github.com/agbruneau/FibGo/internal/fibonacci"
)

// noTiming is the duration findBest reports when no candidate produced a
// timing; applyCalibrationResults keys on it to tell "measured" from "kept
// the default".
const noTiming = time.Duration(math.MaxInt64)

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
	_, err = calc.Calculate(ctx, nil, 0, CalibrationN, opts)
	return time.Since(start), err
}

// findBest times calc once per candidate threshold, each run under the
// Options opts builds for that candidate, and returns the fastest candidate
// with its duration. Trials that error or time out are skipped; when none
// produces a timing the result is (def, noTiming).
func (r *calibrationRunner) findBest(calc fibonacci.Calculator, candidates []int, def int, opts func(threshold int) fibonacci.Options) (best int, bestDur time.Duration) {
	best, bestDur = def, noTiming
	for _, cand := range candidates {
		if dur, err := r.runTrial(calc, opts(cand)); err == nil && dur < bestDur {
			bestDur, best = dur, cand
		}
	}
	return best, bestDur
}
