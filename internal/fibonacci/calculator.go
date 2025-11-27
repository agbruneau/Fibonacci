// The fibonacci package provides implementations for calculating Fibonacci
// numbers. It exposes a `Calculator` interface that abstracts the
// underlying calculation algorithm, allowing different strategies (e.g., Fast
// Doubling, Matrix Exponentiation) to be used interchangeably. The package also
// integrates optimizations such as a lookup table (LUT) for small values and
// memory management via object pools to minimize pressure on the garbage
// collector (GC).
package fibonacci

import (
	"context"
	"math"
	"math/big"
)

// MaxFibUint64 = 93 because F(93) is the largest Fibonacci number that fits in a uint64,
// as F(94) exceeds 2^64. This value is derived from the very rapid growth of the sequence.
const (
	MaxFibUint64 = 93 // Justified above
)

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

// ProgressReportParams contains the necessary state for calculating progress reporting.
type ProgressReportParams struct {
	// NumBits is the number of bits in the input number 'n'.
	NumBits int
	// Four is a pre-calculated big.Int with the value 4, used in progress
	// calculations.
	Four *big.Int
}

// Cache for frequently used constants
var (
	bigIntFour  = big.NewInt(4)
	bigIntOne   = big.NewInt(1)
	bigIntThree = big.NewInt(3)
)

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
	// Geometric sum: 4^0 + 4^1 + ... + 4^(n-1) = (4^n - 1) / 3
	// We use a simplified model where work roughly quadruples each bit.
	// For large n, this can overflow float64, but we only need the ratio.
	// Actually, for progress bars, we can just sum the weights.
	// Since we only need a ratio, we can normalize.
	// However, to avoid overflow for large numBits (though numBits <= 64 for uint64 n),
	// we can just use the property that the last few steps dominate.
	// Let's stick to a simple weight model: weight(i) = 1 << (i * 2) ?
	// No, multiplication cost M(k) is roughly k^1.6.
	// k grows linearly.
	// So work at step i (0 to numBits-1) is i^1.6.
	// Total work is sum(i^1.6).
	total := 0.0
	for i := 1; i <= numBits; i++ {
		total += float64(i) // Approximation: linear growth of bits -> quadratic work?
		// Actually, let's stick to the existing logic but with floats:
		// Work doubles or quadruples?
		// Fast doubling: F(2k) involves multiplication of size k.
		// Size k doubles every step.
		// Multiplication cost M(N) approx N^1.6.
		// So cost scales by 2^1.6 approx 3.
		// Let's assume factor 3 growth per step.
	}
	// Reverting to the original logic's assumption of factor 4 (geometric series)
	// but using float64. 4^64 overflows float64?
	// 4^64 = 2^128 approx 3e38. Float64 max is 1e308. It fits easily.
	if numBits == 0 {
		return 0
	}
	// (4^n - 1) / 3
	// We can compute this iteratively or using Pow.
	return (math.Pow(4, float64(numBits)) - 1) / 3
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
//
// Returns:
//   - float64: The updated cumulative work done.
func ReportStepProgress(progressReporter ProgressReporter, lastReported *float64, totalWork, workDone float64, i, numBits int) float64 {
	const ReportThreshold = 0.01

	// Work for this step (bit i, counting down from numBits-1 to 0)
	// The step index in the geometric series is (numBits - 1 - i).
	// Wait, the loop goes i = numBits-1 down to 0.
	// At i=numBits-1 (start), we are at small numbers? No, we start from MSB.
	// Fast doubling starts from MSB (small current value) and doubles up.
	// So at i=numBits-1, we have F(1). Small work.
	// At i=0, we have F(n). Huge work.
	// So the work is proportional to 4^(numBits - 1 - i).

	stepIndex := numBits - 1 - i
	workOfStep := math.Pow(4, float64(stepIndex))

	currentTotalDone := workDone + workOfStep

	// Only report if enough progress or boundaries
	if totalWork > 0 {
		currentProgress := currentTotalDone / totalWork
		if currentProgress-*lastReported >= ReportThreshold || i == 0 || i == numBits-1 {
			progressReporter(currentProgress)
			*lastReported = currentProgress
		}
	}
	return currentTotalDone
}

// approxProgress calculates the approximate ratio of num / den as a float64.
// It avoids large allocations by shifting bits if the numbers are too large.
func approxProgress(num, den *big.Int) float64 {
	if den.Sign() == 0 {
		return 0.0
	}
	denLen := den.BitLen()
	if denLen <= 53 {
		n := float64(num.Int64())
		d := float64(den.Int64())
		return n / d
	}

	shift := uint(denLen - 53)
	var n, d big.Int
	n.Rsh(num, shift)
	d.Rsh(den, shift)
	return float64(n.Int64()) / float64(d.Int64())
}

// Calculator defines the public interface for a Fibonacci calculator.
// It is the primary abstraction used by the application's orchestration layer to
// interact with different Fibonacci calculation algorithms.
type Calculator interface {
	// Calculate executes the calculation of the n-th Fibonacci number. It is
	// designed for safe concurrent execution and supports cancellation through the
	// provided context. Progress updates are sent asynchronously to the
	// progressChan.
	//
	// Parameters:
	//   - ctx: The context for managing cancellation and deadlines.
	//   - progressChan: The channel for sending progress updates.
	//   - calcIndex: A unique index for the calculator instance.
	//   - n: The index of the Fibonacci number to calculate.
	//   - threshold: The bit size threshold for parallelizing multiplications.
	//   - fftThreshold: The bit size threshold for using FFT-based multiplication.
	//
	// Returns:
	//   - *big.Int: The calculated Fibonacci number.
	//   - error: An error if one occurred (e.g., context cancellation).
	Calculate(ctx context.Context, progressChan chan<- ProgressUpdate, calcIndex int, n uint64, threshold int, fftThreshold int) (*big.Int, error)

	// Name returns the display name of the calculation algorithm (e.g., "Fast Doubling").
	//
	// Returns:
	//   - string: The name of the algorithm.
	Name() string
}

// coreCalculator defines the internal interface for a pure calculation
// algorithm.
type coreCalculator interface {
	CalculateCore(ctx context.Context, reporter ProgressReporter, n uint64, threshold int, fftThreshold int) (*big.Int, error)
	Name() string
}

// FibCalculator is an implementation of the Calculator interface that uses the
// Decorator design pattern.
// It wraps a coreCalculator to add cross-cutting concerns, such as the lookup
// table optimization for small `n` and the adaptation of the progress reporting
// mechanism.
type FibCalculator struct {
	core coreCalculator
}

// NewCalculator is a factory function that constructs and returns a new
// FibCalculator.
// It takes a coreCalculator as input, which represents the specific Fibonacci
// algorithm to be used. This function panics if the core calculator is nil,
// ensuring system integrity.
//
// Parameters:
//   - core: The core calculator to be wrapped.
//
// Returns:
//   - Calculator: A new FibCalculator instance implementing the Calculator interface.
func NewCalculator(core coreCalculator) Calculator {
	if core == nil {
		panic("fibonacci: the `coreCalculator` implementation cannot be nil")
	}
	return &FibCalculator{core: core}
}

// Name returns the name of the encapsulated coreCalculator, fulfilling the
// Calculator interface by delegating the call.
//
// Returns:
//   - string: The name of the algorithm.
func (c *FibCalculator) Name() string {
	return c.core.Name()
}

// Calculate orchestrates the calculation process.
// It first checks for small values of `n` to leverage the lookup table
// optimization. For larger values, it adapts the progressChan into a
// ProgressReporter callback and delegates the core calculation to the wrapped
// coreCalculator. This method ensures that progress is reported completely upon
// successful calculation.
//
// Parameters:
//   - ctx: The context for managing cancellation and deadlines.
//   - progressChan: The channel for sending progress updates.
//   - calcIndex: A unique index for the calculator instance.
//   - n: The index of the Fibonacci number to calculate.
//   - threshold: The bit size threshold for parallelizing multiplications.
//   - fftThreshold: The bit size threshold for using FFT-based multiplication.
//
// Returns:
//   - *big.Int: The calculated Fibonacci number.
//   - error: An error if one occurred.
func (c *FibCalculator) Calculate(ctx context.Context, progressChan chan<- ProgressUpdate, calcIndex int, n uint64, threshold int, fftThreshold int) (*big.Int, error) {
	reporter := func(progress float64) {
		if progressChan == nil {
			return
		}
		if progress > 1.0 {
			progress = 1.0
		}
		update := ProgressUpdate{CalculatorIndex: calcIndex, Value: progress}
		select {
		case progressChan <- update:
		default:
		}
	}

	if n <= MaxFibUint64 {
		reporter(1.0)
		return lookupSmall(n), nil
	}

	result, err := c.core.CalculateCore(ctx, reporter, n, threshold, fftThreshold)
	if err == nil && result != nil {
		reporter(1.0)
	}
	return result, err
}

var fibLookupTable [MaxFibUint64 + 1]*big.Int

func init() {
	fibLookupTable[0] = big.NewInt(0)
	if MaxFibUint64 > 0 {
		fibLookupTable[1] = big.NewInt(1)
		for i := uint64(2); i <= MaxFibUint64; i++ {
			fibLookupTable[i] = new(big.Int).Add(fibLookupTable[i-1], fibLookupTable[i-2])
		}
	}
}

// lookupSmall returns a copy of the n-th Fibonacci number from the lookup
// table, ensuring the immutability of the table.
func lookupSmall(n uint64) *big.Int {
	return new(big.Int).Set(fibLookupTable[n])
}
