package fibonacci

import (
	"context"
	"math/big"
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
