// This file contains progress reporting types and utilities used by calculators.

package progress

import "math"

// ProgressReportThreshold is the minimum progress change (0.0 to 1.0) required
// before a new progress update is sent. This prevents excessive UI updates
// that could slow down calculations.
//
// A value of 0.01 (1%) provides smooth progress updates without overhead.
const ProgressReportThreshold = 0.01

// ProgressUpdate is a data transfer object (DTO) that encapsulates the
// progress state of a calculation. It is sent over a channel from the
// calculator to the user interface to provide asynchronous progress updates.
type ProgressUpdate struct {
	// CalculatorIndex is a unique identifier for the calculator instance, allowing
	// the UI to distinguish between multiple concurrent calculations.
	CalculatorIndex int
	// Value represents the normalized progress of the calculation, ranging from 0.0 to 1.0.
	Value float64
}

// ProgressCallback defines the functional type for a progress reporting
// callback. This simplified interface is used by core calculation algorithms to
// report their progress without being coupled to the channel-based communication
// mechanism of the broader application.
//
// Parameters:
//   - progress: The normalized progress value (0.0 to 1.0).
type ProgressCallback func(progress float64)

// stepProgress returns the fraction of total work completed once the step for
// bit index i (of numBits) has finished, in closed form and without ever
// materializing 4^numBits.
//
// Cumulative work after processing bit i (stepIndex = numBits-1-i) is the
// geometric partial sum (4^(stepIndex+1) - 1) / 3 and total work is
// (4^numBits - 1) / 3, so:
//
//	progress(i) = (4^(numBits-i) - 1) / (4^numBits - 1)
//
// Dividing numerator and denominator by 4^numBits yields the numerically
// stable equivalent
//
//	progress(i) = (4^(-i) - 4^(-numBits)) / (1 - 4^(-numBits))
//
// Both 4^(-i) and 4^(-numBits) lie in (0, 1] and underflow gracefully toward 0
// for large exponents, so the expression never overflows. It is exactly 1.0 at
// i == 0 and strictly increasing as i decreases, preserving the original
// semantics on the safe domain bit-for-bit while remaining finite for any
// numBits.
func stepProgress(i, numBits int) float64 {
	if numBits <= 0 {
		return 1
	}
	if i <= 0 {
		return 1
	}
	if i >= numBits {
		// Defensive: keeps the result inside [0,1] for unexpected inputs.
		i = numBits
	}
	negI := math.Pow(4, -float64(i))
	negN := math.Pow(4, -float64(numBits))
	denom := 1 - negN
	if denom <= 0 {
		// 4^(-numBits) underflowed to exactly 0 only when numBits is huge; then
		// denom == 1 and we never reach here. Guard kept for total safety.
		return negI
	}
	p := (negI - negN) / denom
	if p < 0 {
		return 0
	}
	if p > 1 {
		return 1
	}
	return p
}

// ReportStepProgress reports the progress of a doubling/exponentiation loop
// through progressReporter, coalescing updates below ProgressReportThreshold.
//
// It takes only the bit index and the bit count (audit L-01). Until A-10 the
// progress ratio was a running work total divided by a geometric sum, which is
// why this function also took totalWork, workDone and a precomputed table of
// powers of 4, and returned the updated running total. A-10 replaced that with
// stepProgress, a closed form over (i, numBits) that cannot overflow, but left
// the three parameters and the return value in place: totalWork only guarded a
// `> 0` test that was always true, and workDone and powers fed a value both
// callers assigned back without ever reading. CalcTotalWork and
// PrecomputePowers4 existed solely to produce those arguments and are gone with
// them.
//
// Parameters:
//   - progressReporter: The callback function to report progress.
//   - lastReported: A pointer to the last reported progress value, to avoid
//     redundant updates.
//   - i: The current bit index being processed, counting down from numBits-1.
//   - numBits: The total number of bits in n.
func ReportStepProgress(progressReporter ProgressCallback, lastReported *float64, i, numBits int) {
	if numBits <= 0 {
		return
	}
	currentProgress := stepProgress(i, numBits)
	if currentProgress-*lastReported >= ProgressReportThreshold || i == 0 || i == numBits-1 {
		progressReporter(currentProgress)
		*lastReported = currentProgress
	}
}
