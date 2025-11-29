package fibonacci

import (
	"context"
	"math/big"
	"math/bits"
	"runtime"
	"sync"
	"sync/atomic"
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
	if n == 0 {
		return big.NewInt(0), nil
	}

	state := acquireMatrixState()
	defer releaseMatrixState(state)

	exponent := n - 1
	numBits := bits.Len64(exponent)
	useParallel := runtime.NumCPU() > 1 && opts.ParallelThreshold > 0

	// Calculate total work for progress reporting via common utility
	totalWork := CalcTotalWork(numBits)
	workDone := 0.0
	lastReportedProgress := -1.0

	for i := 0; i < numBits; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		// Harmonized reporting via common utility function
		// For Matrix Exponentiation, we iterate from LSB (small work) to MSB (large work).
		// However, ReportStepProgress assumes `i` counts down from MSB (large work) to LSB.
		// To correct this, we invert the index passed to ReportStepProgress so that
		// stepIndex becomes `i`, resulting in increasing work values.
		workDone = ReportStepProgress(reporter, &lastReportedProgress, totalWork, workDone, numBits-1-i, numBits)

		if (exponent>>uint(i))&1 == 1 {
			// Decide on parallelism based on the max size of the operands involved
			inParallel := useParallel && maxBitLenMatrix(state.p) > opts.ParallelThreshold
			multiplyMatrices(state.tempMatrix, state.res, state.p, state, inParallel, opts.FFTThreshold, opts.StrassenThreshold)
			state.res, state.tempMatrix = state.tempMatrix, state.res
		}

		if i < numBits-1 {
			inParallel := useParallel && maxBitLenMatrix(state.p) > opts.ParallelThreshold
			squareSymmetricMatrix(state.tempMatrix, state.p, state, inParallel, opts.FFTThreshold)
			state.p, state.tempMatrix = state.tempMatrix, state.p
		}
	}
	return new(big.Int).Set(state.res.a), nil
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
func multiplyMatrices(dest, m1, m2 *matrix, state *matrixState, inParallel bool, fftThreshold int, strassenThreshold int) {
	strassenThresholdBits := strassenThreshold
	if strassenThresholdBits == 0 {
		strassenThresholdBits = GetDefaultStrassenThreshold()
	}
	if maxBitLenTwoMatrices(m1, m2) <= strassenThresholdBits {
		multiplyMatricesClassic(dest, m1, m2, state, inParallel, fftThreshold)
		return
	}
	multiplyMatricesStrassen(dest, m1, m2, state, inParallel, fftThreshold)
}

// multiplyMatricesStrassen implements Strassen's algorithm for 2x2 matrices
// to reduce the number of multiplications from 8 to 7.
//
// Parameters:
//   - dest: The destination matrix.
//   - m1: The first matrix operand.
//   - m2: The second matrix operand.
//   - state: The matrix state providing temporary storage.
//   - inParallel: Whether to execute the operation in parallel.
//   - fftThreshold: The threshold for using FFT-based multiplication.
func multiplyMatricesStrassen(dest, m1, m2 *matrix, state *matrixState, inParallel bool, fftThreshold int) {
	// m1 = [[a, b], [c, d]] and m2 = [[e, f], [g, h]]
	// The temporary variables from the state object are used to store intermediate results.
	p1, p2, p3, p4, p5, p6, p7 := state.p1, state.p2, state.p3, state.p4, state.p5, state.p6, state.p7
	s1, s2, s3, s4, s5, s6, s7, s8, s9, s10 := state.s1, state.s2, state.s3, state.s4, state.s5, state.s6, state.s7, state.s8, state.s9, state.s10

	// Pre-calculate sums and differences
	s1.Sub(m2.b, m2.d)  // f - h
	s2.Add(m1.a, m1.b)  // a + b
	s3.Add(m1.c, m1.d)  // c + d
	s4.Sub(m2.c, m2.a)  // g - e
	s5.Add(m1.a, m1.d)  // a + d
	s6.Add(m2.a, m2.d)  // e + h
	s7.Sub(m1.b, m1.d)  // b - d
	s8.Add(m2.c, m2.d)  // g + h
	s9.Sub(m1.a, m1.c)  // a - c
	s10.Add(m2.a, m2.b) // e + f

	// Execute the 7 multiplications
	if inParallel {
		var wg sync.WaitGroup
		wg.Add(7)
		go func() { p1 = smartMultiply(p1, m1.a, s1, fftThreshold); wg.Done() }()
		go func() { p2 = smartMultiply(p2, s2, m2.d, fftThreshold); wg.Done() }()
		go func() { p3 = smartMultiply(p3, s3, m2.a, fftThreshold); wg.Done() }()
		go func() { p4 = smartMultiply(p4, m1.d, s4, fftThreshold); wg.Done() }()
		go func() { p5 = smartMultiply(p5, s5, s6, fftThreshold); wg.Done() }()
		go func() { p6 = smartMultiply(p6, s7, s8, fftThreshold); wg.Done() }()
		go func() { p7 = smartMultiply(p7, s9, s10, fftThreshold); wg.Done() }()
		wg.Wait()
	} else {
		p1 = smartMultiply(p1, m1.a, s1, fftThreshold)
		p2 = smartMultiply(p2, s2, m2.d, fftThreshold)
		p3 = smartMultiply(p3, s3, m2.a, fftThreshold)
		p4 = smartMultiply(p4, m1.d, s4, fftThreshold)
		p5 = smartMultiply(p5, s5, s6, fftThreshold)
		p6 = smartMultiply(p6, s7, s8, fftThreshold)
		p7 = smartMultiply(p7, s9, s10, fftThreshold)
	}

	// Calculate final matrix elements
	// Using temporary state variables to avoid modifying destination values prematurely.
	valA, valB, valC, valD := state.s1, state.s2, state.s3, state.s4
	valA.Add(p5, p4)
	valA.Sub(valA, p2)
	valA.Add(valA, p6)

	valB.Add(p1, p2)

	valC.Add(p3, p4)

	valD.Add(p5, p1)
	valD.Sub(valD, p3)
	valD.Sub(valD, p7)

	dest.a.Set(valA)
	dest.b.Set(valB)
	dest.c.Set(valC)
	dest.d.Set(valD)
}

// squareSymmetricMatrix computes the square of a symmetric matrix.
//
// This function is a performance optimization that reduces the number of integer
// multiplications required to square a matrix. For a symmetric matrix, where
// b equals c, some calculations become redundant. This method avoids those
// redundancies, resulting in a faster computation.
//
// Parameters:
//   - dest: The destination matrix.
//   - mat: The symmetric matrix to square.
//   - state: The matrix state providing temporary storage.
//   - inParallel: Whether to execute the operation in parallel.
//   - fftThreshold: The threshold for using FFT-based multiplication.
func squareSymmetricMatrix(dest, mat *matrix, state *matrixState, inParallel bool, fftThreshold int) {
	a2, b2, d2 := state.t1, state.t2, state.t3
	b_ad, ad := state.t4, state.t5
	ad.Add(mat.a, mat.d)

	if inParallel {
		var wg sync.WaitGroup
		wg.Add(4)
		go func() { a2 = smartMultiply(a2, mat.a, mat.a, fftThreshold); wg.Done() }()
		go func() { b2 = smartMultiply(b2, mat.b, mat.b, fftThreshold); wg.Done() }()
		go func() { d2 = smartMultiply(d2, mat.d, mat.d, fftThreshold); wg.Done() }()
		go func() { b_ad = smartMultiply(b_ad, mat.b, ad, fftThreshold); wg.Done() }()
		wg.Wait()
	} else {
		a2 = smartMultiply(a2, mat.a, mat.a, fftThreshold)
		b2 = smartMultiply(b2, mat.b, mat.b, fftThreshold)
		d2 = smartMultiply(d2, mat.d, mat.d, fftThreshold)
		b_ad = smartMultiply(b_ad, mat.b, ad, fftThreshold)
	}

	dest.a.Add(a2, b2)
	dest.b.Set(b_ad)
	dest.c.Set(b_ad)
	dest.d.Add(b2, d2)
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
func multiplyMatricesClassic(dest, m1, m2 *matrix, state *matrixState, inParallel bool, fftThreshold int) {
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

	if inParallel {
		var wg sync.WaitGroup
		wg.Add(8)
		go func() { ae = smartMultiply(ae, m1.a, m2.a, fftThreshold); wg.Done() }()
		go func() { bg = smartMultiply(bg, m1.b, m2.c, fftThreshold); wg.Done() }()
		go func() { af = smartMultiply(af, m1.a, m2.b, fftThreshold); wg.Done() }()
		go func() { bh = smartMultiply(bh, m1.b, m2.d, fftThreshold); wg.Done() }()
		go func() { ce = smartMultiply(ce, m1.c, m2.a, fftThreshold); wg.Done() }()
		go func() { dg = smartMultiply(dg, m1.d, m2.c, fftThreshold); wg.Done() }()
		go func() { cf = smartMultiply(cf, m1.c, m2.b, fftThreshold); wg.Done() }()
		go func() { dh = smartMultiply(dh, m1.d, m2.d, fftThreshold); wg.Done() }()
		wg.Wait()
	} else {
		ae = smartMultiply(ae, m1.a, m2.a, fftThreshold)
		bg = smartMultiply(bg, m1.b, m2.c, fftThreshold)
		af = smartMultiply(af, m1.a, m2.b, fftThreshold)
		bh = smartMultiply(bh, m1.b, m2.d, fftThreshold)
		ce = smartMultiply(ce, m1.c, m2.a, fftThreshold)
		dg = smartMultiply(dg, m1.d, m2.c, fftThreshold)
		cf = smartMultiply(cf, m1.c, m2.b, fftThreshold)
		dh = smartMultiply(dh, m1.d, m2.d, fftThreshold)
	}

	dest.a.Add(ae, bg)
	dest.b.Add(af, bh)
	dest.c.Add(ce, dg)
	dest.d.Add(cf, dh)
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
	return &matrix{new(big.Int), new(big.Int), new(big.Int), new(big.Int)}
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
	New: func() interface{} {
		s := &matrixState{
			res:        newMatrix(),
			p:          newMatrix(),
			tempMatrix: newMatrix(),
			p1:         new(big.Int), p2: new(big.Int), p3: new(big.Int), p4: new(big.Int),
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
//
// Returns:
//   - *matrixState: A fresh or reused matrixState.
func acquireMatrixState() *matrixState {
	s := matrixStatePool.Get().(*matrixState)
	s.Reset()
	return s
}

// releaseMatrixState puts a state back into the pool.
//
// Parameters:
//   - s: The matrixState to return to the pool.
func releaseMatrixState(s *matrixState) {
	matrixStatePool.Put(s)
}
