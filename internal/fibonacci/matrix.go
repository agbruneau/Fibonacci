package fibonacci

import (
	"context"
	"math/big"
	"sync"
	"sync/atomic"

	"github.com/agbru/fibcalc/internal/parallel"
	"github.com/agbru/fibcalc/internal/pool"
)

// MatrixExponentiation offers a classic and efficient approach to calculating
// Fibonacci numbers.
//
// Mathematical Basis:
// This method is based on a fundamental property of the Fibonacci sequence,
// which can be expressed in matrix form:
//
//	[ F(n+1) F(n)   ] = [ 1 1 ]^n
//	[ F(n)   F(n-1) ]   [ 1 0 ]
//
// To compute F(n), the algorithm calculates the n-th power of the Q-matrix,
// [[1, 1], [1, 0]], using binary exponentiation (exponentiation by squaring).
// This reduces the number of matrix multiplications from O(n) to O(log n).
//
// Algorithmic Complexity:
// The total complexity is O(log n * M(n)), where M(n) is the complexity of
// multiplying the numbers involved, which are proportional to n bits.
//   - A classic 2x2 matrix multiplication requires 8 integer multiplications.
//   - Strassen's algorithm reduces this to 7 multiplications, improving the
//     constant factor but with higher overhead from additions and subtractions.
//   - Squaring a symmetric matrix can be done with only 4 multiplications.
//
// Optimization Details:
// This implementation is enhanced with several key optimizations:
//   - Zero-Allocation: A sync.Pool recycles `matrixState` objects, minimizing
//     memory allocations and GC pressure.
//   - Parallel Processing: Matrix multiplications are parallelized above a
//     `threshold` (default 4096 bits), leveraging multi-core processors.
//   - Symmetric Squaring: A specialized function, `squareSymmetricMatrix`, is
//     used for squaring symmetric matrices, reducing the multiplication count.
//   - Strassen's Algorithm: For matrices with elements larger than a
//     `strassen-threshold` (default 256 bits), Strassen's algorithm is used to
//     reduce the number of expensive `big.Int` multiplications from 8 to 7.
//     The threshold is set to overcome the overhead of the extra additions and
//     subtractions involved.
type MatrixExponentiation struct{}

// Name returns the descriptive name of the algorithm.
// This name is displayed in the application's user interface, providing a clear
// and concise identification of the calculation method, including its key
// performance characteristics.
//
// Returns:
//   - string: The name of the algorithm.
func (c *MatrixExponentiation) Name() string {
	return "Matrix Exponentiation (O(log n), Parallel, Zero-Alloc)"
}

// CalculateCore computes F(n) using the matrix exponentiation method.
//
// This function implements the binary exponentiation algorithm to efficiently
// calculate the n-th power of the Fibonacci matrix. It also handles state
// management through pooling and reports progress to the caller.
//
// Parameters:
//   - ctx: The context for managing cancellation and deadlines.
//   - reporter: The function used for reporting progress.
//   - n: The index of the Fibonacci number to calculate.
//   - opts: Configuration options for the calculation.
//
// Returns:
//   - *big.Int: The calculated Fibonacci number.
//   - error: An error if one occurred (e.g., context cancellation).
func (c *MatrixExponentiation) CalculateCore(ctx context.Context, reporter ProgressReporter, n uint64, opts Options) (*big.Int, error) {
	state := acquireMatrixState()
	defer releaseMatrixState(state)

	// Use framework for the matrix exponentiation loop
	framework := NewMatrixFramework()
	return framework.ExecuteMatrixLoop(ctx, reporter, n, opts, state)
}

// defaultStrassenThresholdBits controls the switch to Strassen's algorithm.
// It is the bit size threshold at which matrix multiplication switches from the
// classic algorithm to the more complex, but asymptotically faster, Strassen's
// algorithm. This value is modifiable at startup via configuration, allowing for
// performance tuning based on the specific hardware and workload.
// Access is thread-safe via atomic operations.
var defaultStrassenThresholdBits atomic.Int32

func init() {
	defaultStrassenThresholdBits.Store(256)
}

// SetDefaultStrassenThreshold sets the default Strassen threshold in bits.
// This function is thread-safe.
func SetDefaultStrassenThreshold(bits int) {
	defaultStrassenThresholdBits.Store(int32(bits))
}

// GetDefaultStrassenThreshold returns the current default Strassen threshold in bits.
// This function is thread-safe.
func GetDefaultStrassenThreshold() int {
	return int(defaultStrassenThresholdBits.Load())
}

// multiplyMatrices dynamically decides between the classic and Strassen
// multiplication algorithms.
// The decision is based on a threshold on the bit size of the operands. For
// smaller sizes, the classic version is used to avoid the overhead of
// Strassen's additions.
//
// Parameters:
//   - dest: The destination matrix.
//   - m1: The first matrix operand.
//   - m2: The second matrix operand.
//   - state: The matrix state providing temporary storage.
//   - inParallel: Whether to execute the operation in parallel.
//   - fftThreshold: The threshold for using FFT-based multiplication.
//   - strassenThreshold: The bit size threshold to switch to Strassen's algorithm.
//
// Returns:
//   - error: An error if the calculation failed.
func multiplyMatrices(dest, m1, m2 *matrix, state *matrixState, inParallel bool, fftThreshold int, strassenThreshold int) error {
	strassenThresholdBits := strassenThreshold
	if strassenThresholdBits == 0 {
		strassenThresholdBits = GetDefaultStrassenThreshold()
	}
	if maxBitLenTwoMatrices(m1, m2) <= strassenThresholdBits {
		return multiplyMatricesClassic(dest, m1, m2, state, inParallel, fftThreshold)
	}
	return multiplyMatricesStrassen(dest, m1, m2, state, inParallel, fftThreshold)
}

// task defines a common interface for executable tasks.
// This allows using generics to eliminate code duplication between
// multiplication and squaring task execution.
type task interface {
	execute() error
}

// multiplicationTask represents a single multiplication operation
// to be executed either sequentially or in parallel.
type multiplicationTask struct {
	dest         **big.Int
	a, b         *big.Int
	fftThreshold int
}

// execute performs the multiplication task.
func (t *multiplicationTask) execute() error {
	var err error
	*t.dest, err = smartMultiply(*t.dest, t.a, t.b, t.fftThreshold)
	return err
}

// squaringTask represents a single squaring operation (x * x)
// to be executed either sequentially or in parallel.
// Squaring is optimized compared to general multiplication because
// it exploits the symmetry of the computation.
type squaringTask struct {
	dest         **big.Int
	x            *big.Int
	fftThreshold int
}

// execute performs the squaring task.
func (t *squaringTask) execute() error {
	var err error
	*t.dest, err = smartSquare(*t.dest, t.x, t.fftThreshold)
	return err
}

// executeTasks executes a batch of tasks (multiplication or squaring) either
// sequentially or in parallel based on the inParallel flag.
// This generic function eliminates code duplication between different task types
// by using Go 1.18+ generics with a pointer constraint pattern.
//
// Type Parameters:
//   - T: The value type of the task (e.g., multiplicationTask, squaringTask).
//   - PT: A pointer type to T that implements the task interface.
//
// Parameters:
//   - tasks: The slice of tasks to execute (values, not pointers).
//   - inParallel: Whether to execute tasks in parallel.
//
// Returns:
//   - error: An error if any task failed.
func executeTasks[T any, PT interface {
	*T
	task
}](tasks []T, inParallel bool) error {
	if inParallel {
		var wg sync.WaitGroup
		var ec parallel.ErrorCollector
		wg.Add(len(tasks))
		for i := range tasks {
			go func(t PT) {
				defer wg.Done()
				ec.SetError(t.execute())
			}(PT(&tasks[i]))
		}
		wg.Wait()
		return ec.Err()
	}
	for i := range tasks {
		if err := PT(&tasks[i]).execute(); err != nil {
			return err
		}
	}
	return nil
}

// multiplyMatricesStrassen implements the Strassen-Winograd algorithm for 2x2 matrices.
// This variant reduces the number of additions/subtractions from 18 to 15 compared to
// the standard Strassen algorithm, while maintaining 7 multiplications.
//
// Parameters:
//   - dest: The destination matrix.
//   - m1: The first matrix operand.
//   - m2: The second matrix operand.
//   - state: The matrix state providing temporary storage.
//   - inParallel: Whether to execute the operation in parallel.
//   - fftThreshold: The threshold for using FFT-based multiplication.
//
// Returns:
//   - error: An error if the calculation failed.
func multiplyMatricesStrassen(dest, m1, m2 *matrix, state *matrixState, inParallel bool, fftThreshold int) error {
	// Winograd's variant uses 7 multiplications and 15 additions/subtractions.
	//
	// Pre-computations (8 additions/subtractions):
	// S1 = A21 + A22
	// S2 = S1 - A11
	// S3 = A11 - A21
	// S4 = A12 - S2
	// S5 = B12 - B11
	// S6 = B22 - S5
	// S7 = B22 - B12
	// S8 = S6 - B21
	//
	// Multiplications (7 multiplications):
	// P1 = S2 * S6
	// P2 = A11 * B11
	// P3 = A12 * B21
	// P4 = S3 * S7
	// P5 = S1 * S5
	// P6 = S4 * B22
	// P7 = A22 * S8
	//
	// Post-computations (7 additions/subtractions):
	// T1 = P1 + P2
	// T2 = T1 + P4
	// C11 = P2 + P3
	// C12 = T1 + P5 + P6
	// C21 = T2 - P7
	// C22 = T2 + P5

	// Map temporaries to state variables
	s1, s2, s3, s4 := state.s1, state.s2, state.s3, state.s4
	s5, s6, s7, s8 := state.s5, state.s6, state.s7, state.s8
	p1, p2, p3, p4 := state.p1, state.p2, state.p3, state.p4
	p5, p6, p7 := state.p5, state.p6, state.p7
	t1, t2 := state.t1, state.t2

	// Pre-computations
	s1.Add(m1.c, m1.d) // S1 = A21 + A22
	s2.Sub(s1, m1.a)   // S2 = S1 - A11
	s3.Sub(m1.a, m1.c) // S3 = A11 - A21
	s4.Sub(m1.b, s2)   // S4 = A12 - S2
	s5.Sub(m2.b, m2.a) // S5 = B12 - B11
	s6.Sub(m2.d, s5)   // S6 = B22 - S5
	s7.Sub(m2.d, m2.b) // S7 = B22 - B12
	s8.Sub(s6, m2.c)   // S8 = S6 - B21

	// Execute the 7 multiplications using the generic task executor
	tasks := []multiplicationTask{
		{&p1, s2, s6, fftThreshold},
		{&p2, m1.a, m2.a, fftThreshold},
		{&p3, m1.b, m2.c, fftThreshold},
		{&p4, s3, s7, fftThreshold},
		{&p5, s1, s5, fftThreshold},
		{&p6, s4, m2.d, fftThreshold},
		{&p7, m1.d, s8, fftThreshold},
	}
	if err := executeTasks[multiplicationTask, *multiplicationTask](tasks, inParallel); err != nil {
		return err
	}

	// Post-computations
	t1.Add(p1, p2) // T1 = P1 + P2
	t2.Add(t1, p4) // T2 = T1 + P4

	// Calculate final matrix elements
	// Use temporaries for C12 and C22 to avoid overwriting if dest aliases inputs (though unlikely here)
	// But dest.a/b/c/d are distinct pointers so we can write directly if we are careful.
	// However, standard practice is to compute fully then assign.

	// C11 = P2 + P3
	dest.a.Add(p2, p3)

	// C12 = T1 + P5 + P6
	dest.b.Add(t1, p5)
	dest.b.Add(dest.b, p6)

	// C21 = T2 - P7
	dest.c.Sub(t2, p7)

	// C22 = T2 + P5
	dest.d.Add(t2, p5)

	return nil
}

// squareSymmetricMatrix computes the square of a symmetric matrix.
//
// This function is a performance optimization that reduces the number of integer
// multiplications required to square a matrix. For a symmetric matrix, where
// b equals c, some calculations become redundant. This method avoids those
// redundancies, resulting in a faster computation.
//
// The three squaring operations (a², b², d²) use optimized smartSquare which
// saves approximately 33% of FFT computation time compared to general multiplication.
//
// Parameters:
//   - dest: The destination matrix.
//   - mat: The symmetric matrix to square.
//   - state: The matrix state providing temporary storage.
//   - inParallel: Whether to execute the operation in parallel.
//   - fftThreshold: The threshold for using FFT-based multiplication.
//
// Returns:
//   - error: An error if the calculation failed.
func squareSymmetricMatrix(dest, mat *matrix, state *matrixState, inParallel bool, fftThreshold int) error {
	a2, b2, d2 := state.t1, state.t2, state.t3
	b_ad, ad := state.t4, state.t5
	ad.Add(mat.a, mat.d)

	// Execute the 3 squaring operations using optimized squaring
	sqrTasks := []squaringTask{
		{&a2, mat.a, fftThreshold},
		{&b2, mat.b, fftThreshold},
		{&d2, mat.d, fftThreshold},
	}

	// Execute the 1 general multiplication (b * (a+d))
	mulTasks := []multiplicationTask{
		{&b_ad, mat.b, ad, fftThreshold},
	}

	if inParallel {
		// Execute all 4 operations in parallel with ErrorCollector
		var wg sync.WaitGroup
		var ec parallel.ErrorCollector
		wg.Add(4)

		for i := range sqrTasks {
			go func(t *squaringTask) {
				defer wg.Done()
				var err error
				*t.dest, err = smartSquare(*t.dest, t.x, t.fftThreshold)
				ec.SetError(err)
			}(&sqrTasks[i])
		}
		go func() {
			defer wg.Done()
			var err error
			*mulTasks[0].dest, err = smartMultiply(*mulTasks[0].dest, mulTasks[0].a, mulTasks[0].b, mulTasks[0].fftThreshold)
			ec.SetError(err)
		}()
		wg.Wait()
		if err := ec.Err(); err != nil {
			return err
		}
	} else {
		if err := executeTasks[squaringTask, *squaringTask](sqrTasks, false); err != nil {
			return err
		}
		if err := executeTasks[multiplicationTask, *multiplicationTask](mulTasks, false); err != nil {
			return err
		}
	}

	dest.a.Add(a2, b2)
	dest.b.Set(b_ad)
	dest.c.Set(b_ad)
	dest.d.Add(b2, d2)
	return nil
}

// multiplyMatricesClassic performs a naive 2x2 matrix multiplication.
// It requires 8 integer multiplications.
//
// Parameters:
//   - dest: The destination matrix.
//   - m1: The first matrix operand.
//   - m2: The second matrix operand.
//   - state: The matrix state providing temporary storage.
//   - inParallel: Whether to execute the operation in parallel.
//   - fftThreshold: The threshold for using FFT-based multiplication.
//
// Returns:
//   - error: An error if the calculation failed.
func multiplyMatricesClassic(dest, m1, m2 *matrix, state *matrixState, inParallel bool, fftThreshold int) error {
	// m1 = [[a,b],[c,d]], m2 = [[e,f],[g,h]]
	// Uses buffers from the state to avoid allocations
	// a = a*e + b*g
	// b = a*f + b*h
	// c = c*e + d*g
	// d = c*f + d*h

	// Buffers
	ae, bg := state.p1, state.p2
	af, bh := state.p3, state.p4
	ce, dg := state.p5, state.p6
	cf, dh := state.s1, state.s2

	// Execute the 8 multiplications using the generic task executor
	tasks := []multiplicationTask{
		{&ae, m1.a, m2.a, fftThreshold},
		{&bg, m1.b, m2.c, fftThreshold},
		{&af, m1.a, m2.b, fftThreshold},
		{&bh, m1.b, m2.d, fftThreshold},
		{&ce, m1.c, m2.a, fftThreshold},
		{&dg, m1.d, m2.c, fftThreshold},
		{&cf, m1.c, m2.b, fftThreshold},
		{&dh, m1.d, m2.d, fftThreshold},
	}
	if err := executeTasks[multiplicationTask, *multiplicationTask](tasks, inParallel); err != nil {
		return err
	}

	dest.a.Add(ae, bg)
	dest.b.Add(af, bh)
	dest.c.Add(ce, dg)
	dest.d.Add(cf, dh)
	return nil
}

// maxBitLenMatrix returns the maximum bit length among the 4 elements
// of the matrix.
//
// Parameters:
//   - m: The matrix to check.
//
// Returns:
//   - int: The maximum bit length found.
func maxBitLenMatrix(m *matrix) int {
	max := m.a.BitLen()
	if b := m.b.BitLen(); b > max {
		max = b
	}
	if c := m.c.BitLen(); c > max {
		max = c
	}
	if d := m.d.BitLen(); d > max {
		max = d
	}
	return max
}

// maxBitLenTwoMatrices returns the maximum bit length between all elements
// of two matrices.
//
// Parameters:
//   - m1: The first matrix.
//   - m2: The second matrix.
//
// Returns:
//   - int: The overall maximum bit length.
func maxBitLenTwoMatrices(m1, m2 *matrix) int {
	max := maxBitLenMatrix(m1)
	if v := maxBitLenMatrix(m2); v > max {
		max = v
	}
	return max
}

// matrix represents a 2x2 matrix of *big.Int values.
// It is a fundamental data structure for the matrix exponentiation algorithm.
// The fields a, b, c, and d correspond to the elements of the matrix:
//
//	[ a b ]
//	[ c d ]
type matrix struct{ a, b, c, d *big.Int }

// newMatrix allocates and returns a new 2x2 matrix.
// It initializes each of its elements with a new *big.Int, which is a
// convenience for ensuring that matrices are correctly instantiated.
//
// Returns:
//   - *matrix: A pointer to the newly created matrix.
func newMatrix() *matrix {
	// We return a matrix struct with nil pointers, as they will be populated from the pool
	return &matrix{}
}

// Set copies the values from another matrix into the receiver matrix.
// This is a deep copy, ensuring that the underlying *big.Int values are
// duplicated.
//
// Parameters:
//   - other: The source matrix to copy from.
func (m *matrix) Set(other *matrix) {
	m.a.Set(other.a)
	m.b.Set(other.b)
	m.c.Set(other.c)
	m.d.Set(other.d)
}

// SetIdentity configures the matrix as an identity matrix.
// The identity matrix is the multiplicative identity for matrix multiplication,
// and is defined as:
//
//	[ 1 0 ]
//	[ 0 1 ]
func (m *matrix) SetIdentity() {
	m.a.SetInt64(1) // Same as the neutral element of a matrix
	m.b.SetInt64(0)
	m.c.SetInt64(0)
	m.d.SetInt64(1)
}

// SetBaseQ configures the matrix as the base Fibonacci matrix, Q.
// Powers of this matrix are used to generate Fibonacci numbers. It is defined
// as:
//
//	[ 1 1 ]
//	[ 1 0 ]
func (m *matrix) SetBaseQ() {
	m.a.SetInt64(1) // Base Q of the Fibonacci recurrence
	m.b.SetInt64(1)
	m.c.SetInt64(1)
	m.d.SetInt64(0)
}

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
// It sets the result matrix to identity and the base power matrix to Q.
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
	New: func() any {
		// We only allocate the structure pointers here.
		// The actual big.Ints are acquired from the global pool in acquireMatrixState.
		s := &matrixState{
			res:        newMatrix(),
			p:          newMatrix(),
			tempMatrix: newMatrix(),
			// Temporaries will be populated from the pool
		}
		return s
	},
}

// acquireMatrixState gets a state from the pool and resets it.
//
// Returns:
//   - *matrixState: A fresh or reused matrixState.
func acquireMatrixState() *matrixState {
	s := matrixStatePool.Get().(*matrixState)

	// Acquire big.Ints from the global pool
	s.p1, s.p2, s.p3, s.p4 = pool.AcquireBigInt(), pool.AcquireBigInt(), pool.AcquireBigInt(), pool.AcquireBigInt()
	s.p5, s.p6, s.p7 = pool.AcquireBigInt(), pool.AcquireBigInt(), pool.AcquireBigInt()

	s.s1, s.s2, s.s3, s.s4 = pool.AcquireBigInt(), pool.AcquireBigInt(), pool.AcquireBigInt(), pool.AcquireBigInt()
	s.s5, s.s6, s.s7, s.s8 = pool.AcquireBigInt(), pool.AcquireBigInt(), pool.AcquireBigInt(), pool.AcquireBigInt()
	s.s9, s.s10 = pool.AcquireBigInt(), pool.AcquireBigInt()

	s.t1, s.t2, s.t3, s.t4, s.t5 = pool.AcquireBigInt(), pool.AcquireBigInt(), pool.AcquireBigInt(), pool.AcquireBigInt(), pool.AcquireBigInt()

	// Initialize matrices with pooled big.Ints
	s.res.a, s.res.b, s.res.c, s.res.d = pool.AcquireBigInt(), pool.AcquireBigInt(), pool.AcquireBigInt(), pool.AcquireBigInt()
	s.p.a, s.p.b, s.p.c, s.p.d = pool.AcquireBigInt(), pool.AcquireBigInt(), pool.AcquireBigInt(), pool.AcquireBigInt()
	s.tempMatrix.a, s.tempMatrix.b, s.tempMatrix.c, s.tempMatrix.d = pool.AcquireBigInt(), pool.AcquireBigInt(), pool.AcquireBigInt(), pool.AcquireBigInt()

	s.Reset()
	return s
}

// releaseMatrixState puts a state back into the pool.
//
// Parameters:
//   - s: The matrixState to return to the pool.
func releaseMatrixState(s *matrixState) {
	// Release big.Ints back to the global pool
	pool.ReleaseBigInt(s.p1)
	pool.ReleaseBigInt(s.p2)
	pool.ReleaseBigInt(s.p3)
	pool.ReleaseBigInt(s.p4)
	pool.ReleaseBigInt(s.p5)
	pool.ReleaseBigInt(s.p6)
	pool.ReleaseBigInt(s.p7)

	pool.ReleaseBigInt(s.s1)
	pool.ReleaseBigInt(s.s2)
	pool.ReleaseBigInt(s.s3)
	pool.ReleaseBigInt(s.s4)
	pool.ReleaseBigInt(s.s5)
	pool.ReleaseBigInt(s.s6)
	pool.ReleaseBigInt(s.s7)
	pool.ReleaseBigInt(s.s8)
	pool.ReleaseBigInt(s.s9)
	pool.ReleaseBigInt(s.s10)

	pool.ReleaseBigInt(s.t1)
	pool.ReleaseBigInt(s.t2)
	pool.ReleaseBigInt(s.t3)
	pool.ReleaseBigInt(s.t4)
	pool.ReleaseBigInt(s.t5)

	pool.ReleaseBigInt(s.res.a)
	pool.ReleaseBigInt(s.res.b)
	pool.ReleaseBigInt(s.res.c)
	pool.ReleaseBigInt(s.res.d)
	pool.ReleaseBigInt(s.p.a)
	pool.ReleaseBigInt(s.p.b)
	pool.ReleaseBigInt(s.p.c)
	pool.ReleaseBigInt(s.p.d)
	pool.ReleaseBigInt(s.tempMatrix.a)
	pool.ReleaseBigInt(s.tempMatrix.b)
	pool.ReleaseBigInt(s.tempMatrix.c)
	pool.ReleaseBigInt(s.tempMatrix.d)

	// Clear pointers to avoid holding references (optional but good for safety)
	s.p1, s.p2, s.p3, s.p4 = nil, nil, nil, nil
	s.p5, s.p6, s.p7 = nil, nil, nil
	s.s1, s.s2, s.s3, s.s4 = nil, nil, nil, nil
	s.s5, s.s6, s.s7, s.s8 = nil, nil, nil, nil
	s.s9, s.s10 = nil, nil
	s.t1, s.t2, s.t3, s.t4, s.t5 = nil, nil, nil, nil, nil

	s.res.a, s.res.b, s.res.c, s.res.d = nil, nil, nil, nil
	s.p.a, s.p.b, s.p.c, s.p.d = nil, nil, nil, nil
	s.tempMatrix.a, s.tempMatrix.b, s.tempMatrix.c, s.tempMatrix.d = nil, nil, nil, nil

	matrixStatePool.Put(s)
}
