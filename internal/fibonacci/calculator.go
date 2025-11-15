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
	"sync"
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
	NumBits int
	Four    *big.Int
}

// Cache for frequently used constants
var (
	bigIntFour  = big.NewInt(4)
	bigIntOne   = big.NewInt(1)
	bigIntThree = big.NewInt(3)
)

// CalcTotalWork calculates the total work (number of weighted steps) for O(log n) algorithms.
// Optimized to reuse pre-calculated constants
func CalcTotalWork(numBits int) *big.Int {
	totalWork := new(big.Int)
	if numBits > 0 {
		totalWork.Exp(bigIntFour, big.NewInt(int64(numBits)), nil).Sub(totalWork, bigIntOne).Div(totalWork, bigIntThree)
	}
	return totalWork
}

// ReportStepProgress handles harmonized progress reporting for algorithms that
// iterate over the bits of `n`. It supports both forward (LSB to MSB) and
// reverse (MSB to LSB) iteration.
func ReportStepProgress(progressReporter ProgressReporter, lastReported *float64, totalWork, workDone, workOfStep *big.Int, i, numBits int, reversed bool) {
	const ReportThreshold = 0.01 // Report if progress increases by at least 1%
	if totalWork.Sign() > 0 {
		// The computational work of a step `j` (from 0 to numBits-1) is
		// proportional to 4^j.
		if workOfStep.Sign() == 0 { // First step
			if reversed {
				// For MSB-to-LSB, the first step has the largest work: 4^(numBits-1)
				workOfStep.Exp(bigIntFour, big.NewInt(int64(numBits-1)), nil)
			} else {
				// For LSB-to-MSB, the first step has the smallest work: 4^0 = 1
				workOfStep.SetInt64(1)
			}
		} else { // Subsequent steps
			if reversed {
				// Halve the work twice (equivalent to dividing by 4)
				workOfStep.Rsh(workOfStep, 2)
			} else {
				// Double the work twice (equivalent to multiplying by 4)
				workOfStep.Lsh(workOfStep, 2)
			}
		}
		workDone.Add(workDone, workOfStep)

		// Optimization: To avoid costly big.Int -> float64 conversions at each
		// iteration, we only perform the conversion and report progress
		// periodically or on the final step.
		if i%8 == 0 || i == numBits-1 {
			workDoneFloat, _ := new(big.Float).SetInt(workDone).Float64()
			totalWorkFloat, _ := new(big.Float).SetInt(totalWork).Float64()
			currentProgress := workDoneFloat / totalWorkFloat

			if currentProgress-*lastReported >= ReportThreshold || i == 0 || i == numBits-1 {
				progressReporter(currentProgress)
				*lastReported = currentProgress
			}
		}
	}
}

// Calculator defines the public interface for a Fibonacci calculator. It is
// the primary abstraction used by the application's orchestration layer to
// interact with different Fibonacci calculation algorithms.
type Calculator interface {
	// Calculate executes the calculation of the n-th Fibonacci number. It is
	// designed for safe concurrent execution and supports cancellation through the
	// provided context. Progress updates are sent asynchronously to the progressChan.
	//
	// Parameters:
	//   - ctx: The context for managing cancellation and deadlines.
	//   - progressChan: The channel for sending progress updates.
	//   - calcIndex: A unique index for the calculator instance.
	//   - n: The index of the Fibonacci number to calculate.
	//   - threshold: The bit size threshold for parallelizing multiplications.
	//   - fftThreshold: The bit size threshold for using FFT-based multiplication.
	//
	// Returns the calculated Fibonacci number and an error if one occurred.
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

// FibCalculator is an implementation of the `Calculator` interface that uses
// the Decorator design pattern. It wraps a `coreCalculator` to add cross-cutting
// concerns, such as the lookup table optimization for small `n` and the adaptation
// of the progress reporting mechanism.
type FibCalculator struct {
	core coreCalculator
}

// NewCalculator is a factory function that constructs and returns a new
// `FibCalculator`. It takes a `coreCalculator` as input, which represents the
// specific Fibonacci algorithm to be used. This function panics if the core
// calculator is nil, ensuring system integrity.
func NewCalculator(core coreCalculator) Calculator {
	if core == nil {
		panic("fibonacci: the `coreCalculator` implementation cannot be nil")
	}
	return &FibCalculator{core: core}
}

// Name returns the name of the encapsulated `coreCalculator`, fulfilling the
// `Calculator` interface by delegating the call.
func (c *FibCalculator) Name() string {
	return c.core.Name()
}

// Calculate orchestrates the calculation process. It first checks for small
// values of `n` to leverage the lookup table optimization. For larger values, it
// adapts the `progressChan` into a `ProgressReporter` callback and delegates the
// core calculation to the wrapped `coreCalculator`. This method ensures that
// progress is reported completely upon successful calculation.
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

// calculationState aggregates temporary variables for the "Fast Doubling"
// algorithm, allowing efficient management via an object pool.
type calculationState struct {
	f_k, f_k1, t1, t2, t3, t4 *big.Int
}

// Reset resets the state for a new use.
func (s *calculationState) Reset() {
	s.f_k.SetInt64(0)
	s.f_k1.SetInt64(1)
}

var statePool = sync.Pool{
	New: func() interface{} {
		return &calculationState{
			f_k:  new(big.Int),
			f_k1: new(big.Int),
			t1:   new(big.Int),
			t2:   new(big.Int),
			t3:   new(big.Int),
			t4:   new(big.Int),
		}
	},
}

// acquireState gets a state from the pool and resets it.
func acquireState() *calculationState {
	s := statePool.Get().(*calculationState)
	s.Reset()
	return s
}

// releaseState puts a state back into the pool.
func releaseState(s *calculationState) {
	statePool.Put(s)
}

// matrix represents a 2x2 matrix of `*big.Int` values. It is a fundamental
// data structure for the matrix exponentiation algorithm. The fields `a`, `b`,
// `c`, and `d` correspond to the elements of the matrix:
//   [ a b ]
//   [ c d ]
type matrix struct{ a, b, c, d *big.Int }

// newMatrix allocates and returns a new 2x2 matrix, initializing each of its
// elements with a new `*big.Int`. This function is a convenience for ensuring
// that matrices are correctly instantiated.
func newMatrix() *matrix {
	return &matrix{new(big.Int), new(big.Int), new(big.Int), new(big.Int)}
}

// Set copies the values from another matrix (`other`) into the receiver matrix
// (`m`). This is a deep copy, ensuring that the underlying `*big.Int` values are
// duplicated.
func (m *matrix) Set(other *matrix) {
	m.a.Set(other.a)
	m.b.Set(other.b)
	m.c.Set(other.c)
	m.d.Set(other.d)
}

// SetIdentity configures the matrix as an identity matrix, which is defined as:
//   [ 1 0 ]
//   [ 0 1 ]
// The identity matrix is the multiplicative identity for matrix multiplication.
func (m *matrix) SetIdentity() {
	m.a.SetInt64(1) // Same as the neutral element of a matrix
	m.b.SetInt64(0)
	m.c.SetInt64(0)
	m.d.SetInt64(1)
}

// SetBaseQ configures the matrix as the base Fibonacci matrix, Q, defined as:
//   [ 1 1 ]
//   [ 1 0 ]
// Powers of this matrix are used to generate Fibonacci numbers.
func (m *matrix) SetBaseQ() {
	m.a.SetInt64(1) // Base Q of the Fibonacci recurrence
	m.b.SetInt64(1)
	m.c.SetInt64(1)
	m.d.SetInt64(0)
}

// In the progress logic (see CalcTotalWork):
// We use base 4 to model the number of operations via the algorithm's structure
// (1 addition and 3 multiplications at each bit), so we have 4^k.
// +1 in the LUT because the LUT contains F(0) to F(93) inclusive.
// matrixState aggregates variables for the matrix exponentiation algorithm.
// The temporary variables (p1-p7, s1-s10) are specifically designed to support
// the memory requirements of Strassen's matrix multiplication algorithm,
// allowing the entire operation to proceed without any memory allocations in the
// hot path.
type matrixState struct {
	res, p, tempMatrix *matrix
	// Temporaries for Strassen's algorithm products
	p1, p2, p3, p4, p5, p6, p7 *big.Int
	// Temporaries for Strassen's algorithm sums/differences
	s1, s2, s3, s4, s5, s6, s7, s8, s9, s10 *big.Int
	// General purpose temporaries for symmetric squaring
	t1, t2, t3, t4, t5 *big.Int
}

// Reset resets the state for a new use.
func (s *matrixState) Reset() {
	s.res.SetIdentity()
	s.p.SetBaseQ()
}

// matrixStatePool is a `sync.Pool` for `matrixState` objects. Object pools are a
// performance optimization technique used to reduce memory allocation and garbage
// collector overhead. By reusing `matrixState` objects, the application can avoid
// the cost of creating and destroying them for each calculation, which is
// particularly beneficial in a high-performance, concurrent context.
var matrixStatePool = sync.Pool{
	New: func() interface{} {
		s := &matrixState{
			res:        newMatrix(),
			p:          newMatrix(),
			tempMatrix: newMatrix(),
			p1: new(big.Int), p2: new(big.Int), p3: new(big.Int), p4: new(big.Int),
			p5: new(big.Int), p6: new(big.Int), p7: new(big.Int),
			s1: new(big.Int), s2: new(big.Int), s3: new(big.Int), s4: new(big.Int),
			s5: new(big.Int), s6: new(big.Int), s7: new(big.Int), s8: new(big.Int),
			s9: new(big.Int), s10: new(big.Int),
			t1: new(big.Int), t2: new(big.Int), t3: new(big.Int), t4: new(big.Int),
			t5: new(big.Int),
		}
		return s
	},
}

// acquireMatrixState gets a state from the pool and resets it.
func acquireMatrixState() *matrixState {
	s := matrixStatePool.Get().(*matrixState)
	s.Reset()
	return s
}

// releaseMatrixState puts a state back into the pool.
func releaseMatrixState(s *matrixState) {
	matrixStatePool.Put(s)
}
