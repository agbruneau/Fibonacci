package fibonacci

import (
	"context"
	"math/big"
	"math/bits"
	"runtime"
	"sync"
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
//     constant factor but with higher overhead from additions/subtractions.
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
// It returns a string with the name of the algorithm.
func (c *MatrixExponentiation) Name() string {
	return "Matrix Exponentiation (O(log n), Parallel, Zero-Alloc)"
}

// CalculateCore computes F(n) using the matrix exponentiation method.
//
// This function implements the binary exponentiation algorithm to efficiently
// calculate the n-th power of the Fibonacci matrix. It also handles state
// management through pooling and reports progress to the caller.
//
// The context for managing cancellation is ctx. The function for reporting
// progress is reporter. The index of the Fibonacci number to calculate is n.
// The bit size threshold for parallelizing multiplications is threshold. The bit
// size threshold for using FFT-based multiplication is fftThreshold.
//
// It returns the calculated Fibonacci number and an error if one occurred.
func (c *MatrixExponentiation) CalculateCore(ctx context.Context, reporter ProgressReporter, n uint64, threshold int, fftThreshold int) (*big.Int, error) {
	if n == 0 {
		return big.NewInt(0), nil
	}

	state := acquireMatrixState()
	defer releaseMatrixState(state)

	mul := func(dest, x, y *big.Int) *big.Int {
		if fftThreshold > 0 {
			// Use FFT if the smaller of the two operands exceeds the threshold
			minBitLen := x.BitLen()
			if b := y.BitLen(); b < minBitLen {
				minBitLen = b
			}
			if minBitLen > fftThreshold {
				return mulFFT(x, y)
			}
		}
		return dest.Mul(x, y)
	}

	exponent := n - 1
	numBits := bits.Len64(exponent)
	useParallel := runtime.NumCPU() > 1 && threshold > 0

	// Calculate total work for progress reporting via common utility
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
			inParallel := useParallel && maxBitLenMatrix(state.p) > threshold
			multiplyMatrices(state.tempMatrix, state.res, state.p, state, inParallel, mul)
			state.res, state.tempMatrix = state.tempMatrix, state.res
		}

		if i < numBits-1 {
			inParallel := useParallel && maxBitLenMatrix(state.p) > threshold
			squareSymmetricMatrix(state.tempMatrix, state.p, state, inParallel, mul)
			state.p, state.tempMatrix = state.tempMatrix, state.p
		}
	}
	return new(big.Int).Set(state.res.a), nil
}

// DefaultStrassenThresholdBits controls the switch to Strassen's algorithm.
// It is the bit size threshold at which matrix multiplication switches from the
// classic algorithm to the more complex, but asymptotically faster, Strassen's
// algorithm. This value is modifiable at startup via configuration, allowing for
// performance tuning based on the specific hardware and workload.
var DefaultStrassenThresholdBits = 256

// multiplyMatrices dynamically decides between the classic and Strassen
// multiplication algorithms.
// The decision is based on a threshold on the bit size of the operands. For
// smaller sizes, the classic version is used to avoid the overhead of
// Strassen's additions.
//
// The destination matrix is dest. The matrices to be multiplied are m1 and m2.
// The matrixState provides temporary variables. If inParallel is true, the
// multiplications are parallelized. The multiplication function is mul.
func multiplyMatrices(dest, m1, m2 *matrix, state *matrixState, inParallel bool, mul func(dest, x, y *big.Int) *big.Int) {
	strassenThresholdBits := DefaultStrassenThresholdBits
	if maxBitLenTwoMatrices(m1, m2) <= strassenThresholdBits {
		multiplyMatricesClassic(dest, m1, m2, state, inParallel, mul)
		return
	}
	multiplyMatricesStrassen(dest, m1, m2, state, inParallel, mul)
}

// multiplyMatricesStrassen: 2x2 Strassen implementation (7 multiplications)
func multiplyMatricesStrassen(dest, m1, m2 *matrix, state *matrixState, inParallel bool, mul func(dest, x, y *big.Int) *big.Int) {
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
	// Execute the 7 multiplications
	if inParallel {
		var wg sync.WaitGroup
		wg.Add(7)
		go func() { p1 = mul(p1, m1.a, s1); wg.Done() }()
		go func() { p2 = mul(p2, s2, m2.d); wg.Done() }()
		go func() { p3 = mul(p3, s3, m2.a); wg.Done() }()
		go func() { p4 = mul(p4, m1.d, s4); wg.Done() }()
		go func() { p5 = mul(p5, s5, s6); wg.Done() }()
		go func() { p6 = mul(p6, s7, s8); wg.Done() }()
		go func() { p7 = mul(p7, s9, s10); wg.Done() }()
		wg.Wait()
	} else {
		p1 = mul(p1, m1.a, s1)
		p2 = mul(p2, s2, m2.d)
		p3 = mul(p3, s3, m2.a)
		p4 = mul(p4, m1.d, s4)
		p5 = mul(p5, s5, s6)
		p6 = mul(p6, s7, s8)
		p7 = mul(p7, s9, s10)
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
func squareSymmetricMatrix(dest, mat *matrix, state *matrixState, inParallel bool, mul func(dest, x, y *big.Int) *big.Int) {
	a2, b2, d2 := state.t1, state.t2, state.t3
	b_ad, ad := state.t4, state.t5
	ad.Add(mat.a, mat.d)

	if inParallel {
		var wg sync.WaitGroup
		wg.Add(4)
		go func() { a2 = mul(a2, mat.a, mat.a); wg.Done() }()
		go func() { b2 = mul(b2, mat.b, mat.b); wg.Done() }()
		go func() { d2 = mul(d2, mat.d, mat.d); wg.Done() }()
		go func() { b_ad = mul(b_ad, mat.b, ad); wg.Done() }()
		wg.Wait()
	} else {
		a2 = mul(a2, mat.a, mat.a)
		b2 = mul(b2, mat.b, mat.b)
		d2 = mul(d2, mat.d, mat.d)
		b_ad = mul(b_ad, mat.b, ad)
	}

	dest.a.Add(a2, b2)
	dest.b.Set(b_ad)
	dest.c.Set(b_ad)
	dest.d.Add(b2, d2)
}

// multiplyMatricesClassic: naive 2x2 multiplication (8 multiplications)
func multiplyMatricesClassic(dest, m1, m2 *matrix, state *matrixState, inParallel bool, mul func(dest, x, y *big.Int) *big.Int) {
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
		go func() { ae = mul(ae, m1.a, m2.a); wg.Done() }()
		go func() { bg = mul(bg, m1.b, m2.c); wg.Done() }()
		go func() { af = mul(af, m1.a, m2.b); wg.Done() }()
		go func() { bh = mul(bh, m1.b, m2.d); wg.Done() }()
		go func() { ce = mul(ce, m1.c, m2.a); wg.Done() }()
		go func() { dg = mul(dg, m1.d, m2.c); wg.Done() }()
		go func() { cf = mul(cf, m1.c, m2.b); wg.Done() }()
		go func() { dh = mul(dh, m1.d, m2.d); wg.Done() }()
		wg.Wait()
	} else {
		ae = mul(ae, m1.a, m2.a)
		bg = mul(bg, m1.b, m2.c)
		af = mul(af, m1.a, m2.b)
		bh = mul(bh, m1.b, m2.d)
		ce = mul(ce, m1.c, m2.a)
		dg = mul(dg, m1.d, m2.c)
		cf = mul(cf, m1.c, m2.b)
		dh = mul(dh, m1.d, m2.d)
	}

	dest.a.Add(ae, bg)
	dest.b.Add(af, bh)
	dest.c.Add(ce, dg)
	dest.d.Add(cf, dh)
}

// maxBitLenMatrix returns the maximum bit length among the 4 elements
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

// maxBitLenTwoMatrices returns the maximum bit length between two matrices
func maxBitLenTwoMatrices(m1, m2 *matrix) int {
	max := maxBitLenMatrix(m1)
	if v := maxBitLenMatrix(m2); v > max {
		max = v
	}
	return max
}

// executeTasks executes a set of tasks, in parallel if specified.
func executeTasks(inParallel bool, tasks []func()) {
	if !inParallel || len(tasks) < 2 {
		for _, task := range tasks {
			task()
		}
		return
	}
	var wg sync.WaitGroup
	wg.Add(len(tasks) - 1)
	for i := 0; i < len(tasks)-1; i++ {
		go func(i int) {
			defer wg.Done()
			tasks[i]()
		}(i)
	}
	tasks[len(tasks)-1]()
	wg.Wait()
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
// It returns a new *matrix.
func newMatrix() *matrix {
	return &matrix{new(big.Int), new(big.Int), new(big.Int), new(big.Int)}
}

// Set copies the values from another matrix into the receiver matrix.
// This is a deep copy, ensuring that the underlying *big.Int values are
// duplicated.
//
// The matrix to copy from is other.
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
func acquireMatrixState() *matrixState {
	s := matrixStatePool.Get().(*matrixState)
	s.Reset()
	return s
}

// releaseMatrixState puts a state back into the pool.
func releaseMatrixState(s *matrixState) {
	matrixStatePool.Put(s)
}
