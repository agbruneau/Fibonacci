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

// CalcTotalWork calculates the total work for O(log n) algorithms.
// The number of weighted steps is modeled as a geometric series, which allows for
// a more accurate progress representation. The function is optimized to reuse
// pre-calculated constants.
//
// The number of bits in the input number 'n' is numBits.
//
// It returns a *big.Int representing the total work.
func CalcTotalWork(numBits int) *big.Int {
	totalWork := new(big.Int)
	if numBits > 0 {
		totalWork.Exp(bigIntFour, big.NewInt(int64(numBits)), nil).Sub(totalWork, bigIntOne).Div(totalWork, bigIntThree)
	}
	return totalWork
}

// ReportStepProgress handles harmonized progress reporting for algorithms that
// iterate over the bits of 'n'.
func ReportStepProgress(progressReporter ProgressReporter, lastReported *float64, totalWork, workDone, workOfStep *big.Int, i, numBits int, reversed bool) {
	const ReportThreshold = 0.01
	if totalWork.Sign() > 0 {
		if workOfStep.Sign() == 0 {
			if reversed {
				workOfStep.Exp(bigIntFour, big.NewInt(int64(numBits-1)), nil)
			} else {
				workOfStep.SetInt64(1)
			}
		} else {
			if reversed {
				workOfStep.Rsh(workOfStep, 2)
			} else {
				workOfStep.Lsh(workOfStep, 2)
			}
		}
		workDone.Add(workDone, workOfStep)

		if i%8 == 0 || i == numBits-1 {
			currentProgress := approxProgress(workDone, totalWork)
			if currentProgress-*lastReported >= ReportThreshold || i == 0 || i == numBits-1 {
				progressReporter(currentProgress)
				*lastReported = currentProgress
			}
		}
	}
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
	// The context for managing cancellation and deadlines is ctx. The channel for
	// sending progress updates is progressChan. A unique index for the
	// calculator instance is calcIndex. The index of the Fibonacci number to
	// calculate is n. The bit size threshold for parallelizing multiplications is
	// threshold. The bit size threshold for using FFT-based multiplication is
	// fftThreshold.
	//
	// It returns the calculated Fibonacci number and an error if one occurred.
	Calculate(ctx context.Context, progressChan chan<- ProgressUpdate, calcIndex int, n uint64, threshold int, fftThreshold int) (*big.Int, error)

	// Name returns the display name of the calculation algorithm (e.g., "Fast Doubling").
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
// The core calculator to be wrapped is core.
//
// It returns a Calculator interface, implemented by FibCalculator.
func NewCalculator(core coreCalculator) Calculator {
	if core == nil {
		panic("fibonacci: the `coreCalculator` implementation cannot be nil")
	}
	return &FibCalculator{core: core}
}

// Name returns the name of the encapsulated coreCalculator, fulfilling the
// Calculator interface by delegating the call.
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
// The context for managing cancellation and deadlines is ctx. The channel for
// sending progress updates is progressChan. A unique index for the calculator
// instance is calcIndex. The index of the Fibonacci number to calculate is n.
// The bit size threshold for parallelizing multiplications is threshold. The bit
// size threshold for using FFT-based multiplication is fftThreshold.
//
// It returns the calculated Fibonacci number and an error if one occurred.
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
