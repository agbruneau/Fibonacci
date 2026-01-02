// Package fibonacci provides implementations for calculating Fibonacci numbers.
// This file contains progress reporting types and utilities used by calculators.
package fibonacci

import "math"

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

// ProgressReporter defines the functional type for a progress reporting
// callback. This simplified interface is used by core calculation algorithms to
// report their progress without being coupled to the channel-based communication
// mechanism of the broader application.
//
// Parameters:
//   - progress: The normalized progress value (0.0 to 1.0).
type ProgressReporter func(progress float64)

// CalcTotalWork calculates the total work expected for O(log n) algorithms.
// The number of weighted steps is modeled as a geometric series.
// Since the algorithms iterate over bits, the work involved is roughly
// proportional to the bit index.
//
// Parameters:
//   - numBits: The number of bits in the input number n.
//
// Returns:
//   - float64: A value representing the estimated total work units.
func CalcTotalWork(numBits int) float64 {
	if numBits == 0 {
		return 0
	}
	// Geometric sum: 4^0 + 4^1 + ... + 4^(n-1) = (4^n - 1) / 3
	// We use a simplified model where work roughly quadruples each bit.
	return (math.Pow(4, float64(numBits)) - 1) / 3
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
//   - powers: Pre-computed powers of 4 (from PrecomputePowers4) for O(1) lookup.
//
// Returns:
//   - float64: The updated cumulative work done.
func ReportStepProgress(progressReporter ProgressReporter, lastReported *float64, totalWork, workDone float64, i, numBits int, powers []float64) float64 {
	// Work for this step (bit i, counting down from numBits-1 to 0)
	// The step index in the geometric series is (numBits - 1 - i).
	// Fast doubling starts from MSB (small current value) and doubles up.
	// So at i=numBits-1, we have F(1). Small work.
	// At i=0, we have F(n). Huge work.
	// So the work is proportional to 4^(numBits - 1 - i).

	stepIndex := numBits - 1 - i
	workOfStep := powers[stepIndex] // O(1) lookup instead of math.Pow

	currentTotalDone := workDone + workOfStep

	// Only report if enough progress or boundaries
	// Use ProgressReportThreshold constant to avoid magic numbers
	if totalWork > 0 {
		currentProgress := currentTotalDone / totalWork
		if currentProgress-*lastReported >= ProgressReportThreshold || i == 0 || i == numBits-1 {
			progressReporter(currentProgress)
			*lastReported = currentProgress
		}
	}
	return currentTotalDone
}
