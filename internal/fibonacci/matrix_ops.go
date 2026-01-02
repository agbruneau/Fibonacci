package fibonacci

import (
	"sync/atomic"
)

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
		return multiplyMatrix2x2(dest, m1, m2, state, inParallel, fftThreshold)
	}
	return multiplyMatrixStrassen(dest, m1, m2, state, inParallel, fftThreshold)
}

// multiplyMatrixStrassen implements the Strassen-Winograd algorithm for 2x2 matrices.
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
func multiplyMatrixStrassen(dest, m1, m2 *matrix, state *matrixState, inParallel bool, fftThreshold int) error {
	// Winograd's variant uses 7 multiplications and 15 additions/subtractions.
	//
	// Pre-computations (8 additions/subtractions) are handled by computeStrassenIntermediates.
	// Multiplications (7 multiplications) are handled here.
	// Post-computations (7 additions/subtractions) are handled by assembleStrassenResult.

	// 1. Pre-computations (S1-S8)
	computeStrassenIntermediates(state, m1, m2)

	// Map temporaries to state variables for task creation
	s1, s2, s3, s4 := state.s1, state.s2, state.s3, state.s4
	s5, s6, s7, s8 := state.s5, state.s6, state.s7, state.s8
	p1, p2, p3, p4 := state.p1, state.p2, state.p3, state.p4
	p5, p6, p7 := state.p5, state.p6, state.p7

	// 2. Execute the 7 multiplications using the generic task executor
	tasks := []multiplicationTask{
		{&p1, s2, s6, fftThreshold, 0},
		{&p2, m1.a, m2.a, fftThreshold, 0},
		{&p3, m1.b, m2.c, fftThreshold, 0},
		{&p4, s3, s7, fftThreshold, 0},
		{&p5, s1, s5, fftThreshold, 0},
		{&p6, s4, m2.d, fftThreshold, 0},
		{&p7, m1.d, s8, fftThreshold, 0},
	}
	if err := executeTasks[multiplicationTask, *multiplicationTask](tasks, inParallel); err != nil {
		return err
	}

	// 3. Post-computations and Assembly
	assembleStrassenResult(dest, state)

	return nil
}

// computeStrassenIntermediates performs the 8 additions/subtractions required
// for the pre-computation phase of Strassen's algorithm (Winograd variant).
//
// Parameters:
//   - state: The matrix state providing temporary storage (S1-S8).
//   - m1: The first matrix operand.
//   - m2: The second matrix operand.
func computeStrassenIntermediates(state *matrixState, m1, m2 *matrix) {
	// Map temporaries to state variables
	s1, s2, s3, s4 := state.s1, state.s2, state.s3, state.s4
	s5, s6, s7, s8 := state.s5, state.s6, state.s7, state.s8

	// Pre-computations
	s1.Add(m1.c, m1.d) // S1 = A21 + A22
	s2.Sub(s1, m1.a)   // S2 = S1 - A11
	s3.Sub(m1.a, m1.c) // S3 = A11 - A21
	s4.Sub(m1.b, s2)   // S4 = A12 - S2
	s5.Sub(m2.b, m2.a) // S5 = B12 - B11
	s6.Sub(m2.d, s5)   // S6 = B22 - S5
	s7.Sub(m2.d, m2.b) // S7 = B22 - B12
	s8.Sub(s6, m2.c)   // S8 = S6 - B21
}

// assembleStrassenResult performs the post-computations (T1, T2) and
// assembles the final result matrix.
//
// Parameters:
//   - dest: The destination matrix.
//   - state: The matrix state with computed products (P1-P7) and temporaries.
func assembleStrassenResult(dest *matrix, state *matrixState) {
	// Map temporaries
	p1, p2, p3, p4 := state.p1, state.p2, state.p3, state.p4
	p5, p6, p7 := state.p5, state.p6, state.p7
	t1, t2 := state.t1, state.t2

	// Post-computations
	t1.Add(p1, p2) // T1 = P1 + P2
	t2.Add(t1, p4) // T2 = T1 + P4

	// Calculate final matrix elements
	// C11 = P2 + P3
	dest.a.Add(p2, p3)

	// C12 = T1 + P5 + P6
	dest.b.Add(t1, p5)
	dest.b.Add(dest.b, p6)

	// C21 = T2 - P7
	dest.c.Sub(t2, p7)

	// C22 = T2 + P5
	dest.d.Add(t2, p5)
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
	bAd, ad := state.t4, state.t5
	ad.Add(mat.a, mat.d)

	// Execute the 3 squaring operations using optimized squaring
	sqrTasks := []squaringTask{
		{&a2, mat.a, fftThreshold, 0},
		{&b2, mat.b, fftThreshold, 0},
		{&d2, mat.d, fftThreshold, 0},
	}

	// Execute the 1 general multiplication (b * (a+d))
	mulTasks := []multiplicationTask{
		{&bAd, mat.b, ad, fftThreshold, 0},
	}

	// Use unified execution function for both parallel and sequential cases
	if err := executeMixedTasks(sqrTasks, mulTasks, inParallel); err != nil {
		return err
	}

	dest.a.Add(a2, b2)
	dest.b.Set(bAd)
	dest.c.Set(bAd)
	dest.d.Add(b2, d2)
	return nil
}

// multiplyMatrix2x2 performs a naive 2x2 matrix multiplication.
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
func multiplyMatrix2x2(dest, m1, m2 *matrix, state *matrixState, inParallel bool, fftThreshold int) error {
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
		{&ae, m1.a, m2.a, fftThreshold, 0},
		{&bg, m1.b, m2.c, fftThreshold, 0},
		{&af, m1.a, m2.b, fftThreshold, 0},
		{&bh, m1.b, m2.d, fftThreshold, 0},
		{&ce, m1.c, m2.a, fftThreshold, 0},
		{&dg, m1.d, m2.c, fftThreshold, 0},
		{&cf, m1.c, m2.b, fftThreshold, 0},
		{&dh, m1.d, m2.d, fftThreshold, 0},
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
// of the matrix. This function caches BitLen() calls to avoid redundant
// traversals of the internal big.Int representation.
//
// Parameters:
//   - m: The matrix to check.
//
// Returns:
//   - int: The maximum bit length found.
func maxBitLenMatrix(m *matrix) int {
	// Cache all BitLen() calls first to avoid redundant traversals
	aLen := m.a.BitLen()
	bLen := m.b.BitLen()
	cLen := m.c.BitLen()
	dLen := m.d.BitLen()

	maxLen := aLen
	if bLen > maxLen {
		maxLen = bLen
	}
	if cLen > maxLen {
		maxLen = cLen
	}
	if dLen > maxLen {
		maxLen = dLen
	}
	return maxLen
}

// maxBitLenTwoMatrices returns the maximum bit length between all elements
// of two matrices. This function is optimized to cache BitLen() calls.
//
// Parameters:
//   - m1: The first matrix.
//   - m2: The second matrix.
//
// Returns:
//   - int: The overall maximum bit length.
func maxBitLenTwoMatrices(m1, m2 *matrix) int {
	maxLen := maxBitLenMatrix(m1)
	if v := maxBitLenMatrix(m2); v > maxLen {
		maxLen = v
	}
	return maxLen
}
