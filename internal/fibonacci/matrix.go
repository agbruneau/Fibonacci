package fibonacci

import (
	"context"
	"math/big"
	"math/bits"
	"runtime"
	"sync"
)

// MatrixExponentiation offers a classic and efficient approach to calculating Fibonacci
// numbers, with a time complexity of O(log n). This method is based on a
// fundamental property of the Fibonacci sequence, which can be expressed in
// matrix form:
//   [ F(n+1) F(n)   ] = [ 1 1 ]^n
//   [ F(n)   F(n-1) ]   [ 1 0 ]
//
// To compute F(n), the algorithm calculates the n-th power of the matrix Q = [[1, 1], [1, 0]]
// using a technique known as binary exponentiation (or exponentiation by squaring).
// This dramatically reduces the number of required matrix multiplications compared
// to a naive iterative approach.
//
// This implementation is further enhanced with several key optimizations:
//   - Zero-Allocation: A `sync.Pool` is used to recycle `matrixState` objects,
//     which hold the matrices and temporary variables. This practice minimizes
//     memory allocations and reduces pressure on the garbage collector.
//   - Parallel Processing: When dealing with matrices containing very large numbers
//     (as determined by a configurable threshold), the matrix multiplication
//     process is parallelized to leverage the power of multi-core processors.
//   - Symmetric Squaring: The algorithm uses a specialized function, `squareSymmetricMatrix`,
//     for squaring symmetric matrices. This optimization reduces the total number
//     of `big.Int` multiplications required, leading to a noticeable performance gain.
type MatrixExponentiation struct{}

// Name returns the descriptive name of the algorithm.
func (c *MatrixExponentiation) Name() string {
	return "Matrix Exponentiation (O(log n), Parallel, Zero-Alloc)"
}

// CalculateCore computes F(n) using the matrix exponentiation method.
//
// This function implements the binary exponentiation algorithm to efficiently
// calculate the n-th power of the Fibonacci matrix. It also handles state
// management through pooling and reports progress to the caller.
func (c *MatrixExponentiation) CalculateCore(ctx context.Context, reporter ProgressReporter, n uint64, threshold int, fftThreshold int) (*big.Int, error) {
	if n == 0 {
		return big.NewInt(0), nil
	}

	state := acquireMatrixState()
	defer releaseMatrixState(state)

	mul := func(dest, x, y *big.Int) {
		useFFT := fftThreshold > 0 && x.BitLen() > fftThreshold && y.BitLen() > fftThreshold
		if useFFT {
			mulFFT(dest, x, y)
		} else {
			dest.Mul(x, y)
		}
	}

	exponent := n - 1
	numBits := bits.Len64(exponent)
	useParallel := runtime.NumCPU() > 1 && threshold > 0

	var invNumBits float64
	if numBits > 0 {
		invNumBits = 1.0 / float64(numBits)
	}

	for i := 0; i < numBits; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		reporter(float64(i) * invNumBits)

		if (exponent>>uint(i))&1 == 1 {
			inParallel := useParallel && state.p.a.BitLen() > threshold
			multiplyMatrices(state.tempMatrix, state.res, state.p, state, inParallel, mul)
			state.res, state.tempMatrix = state.tempMatrix, state.res
		}

		if i < numBits-1 {
			inParallel := useParallel && state.p.a.BitLen() > threshold
			squareSymmetricMatrix(state.tempMatrix, state.p, state, inParallel, mul)
			state.p, state.tempMatrix = state.tempMatrix, state.p
		}
	}
	return new(big.Int).Set(state.res.a), nil
}

// multiplyMatrices computes the product of two 2x2 matrices, C = A * B.
// It can perform the underlying integer multiplications in parallel for improved
// performance.
func multiplyMatrices(dest, m1, m2 *matrix, state *matrixState, inParallel bool, mul func(dest, x, y *big.Int)) {
	tasks := []func(){
		func() { mul(state.t1, m1.a, m2.a) }, func() { mul(state.t2, m1.b, m2.c) },
		func() { mul(state.t3, m1.a, m2.b) }, func() { mul(state.t4, m1.b, m2.d) },
		func() { mul(state.t5, m1.c, m2.a) }, func() { mul(state.t6, m1.d, m2.c) },
		func() { mul(state.t7, m1.c, m2.b) }, func() { mul(state.t8, m1.d, m2.d) },
	}
	executeTasks(inParallel, tasks)

	dest.a.Add(state.t1, state.t2)
	dest.b.Add(state.t3, state.t4)
	dest.c.Add(state.t5, state.t6)
	dest.d.Add(state.t7, state.t8)
}

// squareSymmetricMatrix computes the square of a symmetric matrix.
//
// This function is a performance optimization that reduces the number of integer
// multiplications required to square a matrix. For a symmetric matrix, where
// b equals c, some calculations become redundant. This method avoids those
// redundancies, resulting in a faster computation.
func squareSymmetricMatrix(dest, mat *matrix, state *matrixState, inParallel bool, mul func(dest, x, y *big.Int)) {
	a2, b2, d2 := state.t1, state.t2, state.t3
	b_ad, ad := state.t4, state.t5
	ad.Add(mat.a, mat.d)

	tasks := []func(){
		func() { mul(a2, mat.a, mat.a) },
		func() { mul(b2, mat.b, mat.b) },
		func() { mul(d2, mat.d, mat.d) },
		func() { mul(b_ad, mat.b, ad) },
	}
	executeTasks(inParallel, tasks)

	dest.a.Add(a2, b2)
	dest.b.Set(b_ad)
	dest.c.Set(b_ad)
	dest.d.Add(b2, d2)
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