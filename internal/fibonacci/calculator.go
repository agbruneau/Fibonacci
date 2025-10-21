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

const (
	// MaxFibUint64 represents the index of the largest Fibonacci number
	// that can be calculated on a 64-bit unsigned integer.
	MaxFibUint64 = 93

	// DefaultParallelThreshold defines the bit threshold from which
	// multiplications of large integers are parallelized.
	DefaultParallelThreshold = 4096
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

// matrix represents a 2x2 matrix of `*big.Int`.
type matrix struct{ a, b, c, d *big.Int }

// newMatrix allocates a new matrix.
func newMatrix() *matrix {
	return &matrix{new(big.Int), new(big.Int), new(big.Int), new(big.Int)}
}

// Set copies the values of another matrix.
func (m *matrix) Set(other *matrix) {
	m.a.Set(other.a)
	m.b.Set(other.b)
	m.c.Set(other.c)
	m.d.Set(other.d)
}

// SetIdentity configures the matrix as an identity matrix.
func (m *matrix) SetIdentity() {
	m.a.SetInt64(1)
	m.b.SetInt64(0)
	m.c.SetInt64(0)
	m.d.SetInt64(1)
}

// SetBaseQ configures the matrix with the Fibonacci base matrix.
func (m *matrix) SetBaseQ() {
	m.a.SetInt64(1)
	m.b.SetInt64(1)
	m.c.SetInt64(1)
	m.d.SetInt64(0)
}

// matrixState aggregates variables for the matrix exponentiation algorithm.
type matrixState struct {
	res, p, tempMatrix             *matrix
	t1, t2, t3, t4, t5, t6, t7, t8 *big.Int
}

// Reset resets the state for a new use.
func (s *matrixState) Reset() {
	s.res.SetIdentity()
	s.p.SetBaseQ()
}

var matrixStatePool = sync.Pool{
	New: func() interface{} {
		return &matrixState{
			res:        newMatrix(),
			p:          newMatrix(),
			tempMatrix: newMatrix(),
			t1: new(big.Int), t2: new(big.Int), t3: new(big.Int), t4: new(big.Int),
			t5: new(big.Int), t6: new(big.Int), t7: new(big.Int), t8: new(big.Int),
		}
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