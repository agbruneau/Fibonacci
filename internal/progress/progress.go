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

// CalcTotalWork calculates the total work expected for O(log n) algorithms.
// The number of weighted steps is modeled as a geometric series
// 4^0 + 4^1 + ... + 4^(numBits-1) = (4^numBits - 1) / 3.
//
// The raw geometric sum is unrepresentable for numBits >= 512: 4^512 == 2^1024
// already exceeds math.MaxFloat64, so math.Pow(4, numBits) returns +Inf and any
// done/total ratio collapses to 0 or NaN — freezing the progress bar precisely
// on the large calculations this project exists to serve (A-10). Progress is
// therefore reported by ReportStepProgress in closed form from the bit index
// (never materializing 4^numBits); this function returns a finite, strictly
// positive, monotonically increasing value purely so the historical
// `totalWork > 0` guard and its callers/tests keep working. The value below the
// safe domain (numBits < 512) is the exact geometric sum; above it the value is
// clamped to the largest representable sum so it stays finite without changing
// the now-closed-form progress computation.
//
// Parameters:
//   - numBits: The number of bits in the input number n.
//
// Returns:
//   - float64: A finite, positive estimate of total work units.
func CalcTotalWork(numBits int) float64 {
	if numBits <= 0 {
		return 0
	}
	// Below the float64 overflow boundary the exact geometric sum is safe and
	// preserves the original observable values for the small-N (uint64) domain.
	const safeNumBits = 511 // 4^511 < MaxFloat64 < 4^512
	if numBits <= safeNumBits {
		return (math.Pow(4, float64(numBits)) - 1) / 3
	}
	// Past the boundary the true sum is unrepresentable. Return the largest
	// finite sum (the safe-domain maximum) so the result stays finite and
	// strictly greater than every smaller numBits — monotonicity holds because
	// the geometric series is strictly increasing up to the clamp and constant
	// thereafter is acceptable (the guard only needs totalWork > 0, and the
	// progress ratio no longer depends on this value).
	return (math.Pow(4, float64(safeNumBits)) - 1) / 3
}

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

// Global lookup table for powers of 4.
// Max supported n is uint64, so bits.Len64(n) is max 64.
// We precompute up to 4^63.
var powersOf4 [64]float64

func init() {
	powersOf4[0] = 1.0
	for i := 1; i < 64; i++ {
		powersOf4[i] = powersOf4[i-1] * 4.0
	}
}

// PrecomputePowers4 pre-calculates powers of 4 from 0 to numBits-1.
// This optimization avoids repeated calls to math.Pow(4, x) during the
// progress reporting loop, providing O(1) lookup instead of expensive
// floating-point exponentiation at each iteration.
//
// Optimization: Returns a slice of a global precomputed array to avoid allocations.
//
// Parameters:
//   - numBits: The number of powers to compute (0 to numBits-1).
//
// Returns:
//   - []float64: A slice where powers[i] = 4^i.
func PrecomputePowers4(numBits int) []float64 {
	if numBits <= 0 {
		return nil
	}
	// Safety check: if numBits exceeds our precomputed range (unlikely for uint64 n),
	// fall back to allocation.
	if numBits > 64 {
		powers := make([]float64, numBits)
		// Copy precomputed part
		copy(powers, powersOf4[:])
		// Compute the rest
		for i := 64; i < numBits; i++ {
			powers[i] = powers[i-1] * 4.0
		}
		return powers
	}
	return powersOf4[:numBits]
}

// ReportStepProgress handles harmonized progress reporting for the calculation algorithms.
// It calculates the cumulative work done based on the current bit iteration and
// reports progress via the provided callback if a significant change has occurred.
//
// Parameters:
//   - progressReporter: The callback function to report progress.
//   - lastReported: A pointer to the last reported progress value to avoid
//     redundant updates.
//   - totalWork: The total estimated work units for the calculation.
//   - workDone: The accumulated work units completed so far.
//   - i: The current bit index being processed.
//   - numBits: The total number of bits in n.
//   - powers: Pre-computed powers of 4 (from PrecomputePowers4). Retained for
//     signature/back-compat and to thread the cumulative work value; the
//     reported progress ratio no longer divides by it (see A-10).
//
// Returns:
//   - float64: The updated cumulative work done (threaded across calls).
func ReportStepProgress(progressReporter ProgressCallback, lastReported *float64, totalWork, workDone float64, i, numBits int, powers []float64) float64 {
	// Work for this step (bit i, counting down from numBits-1 to 0).
	// The step index in the geometric series is (numBits - 1 - i): fast
	// doubling starts from the MSB (small current value, small work) and
	// doubles up, so work is proportional to 4^(numBits-1-i).
	stepIndex := numBits - 1 - i
	workOfStep := 0.0
	if stepIndex >= 0 && stepIndex < len(powers) {
		workOfStep = powers[stepIndex] // O(1) lookup
	}
	currentTotalDone := workDone + workOfStep

	// Progress ratio is computed in closed form from the bit index, NOT as
	// currentTotalDone/totalWork. The geometric quantities overflow float64 to
	// +Inf for numBits >= 512 (4^512 == 2^1024 > MaxFloat64), which used to
	// drive the ratio to 0/NaN and freeze the bar on exactly the large
	// calculations this project targets (A-10). stepProgress is mathematically
	// identical on the safe domain and provably finite, monotone and bounded
	// to [0,1] for any numBits.
	if totalWork > 0 {
		currentProgress := stepProgress(i, numBits)
		if currentProgress-*lastReported >= ProgressReportThreshold || i == 0 || i == numBits-1 {
			progressReporter(currentProgress)
			*lastReported = currentProgress
		}
	}
	return currentTotalDone
}
