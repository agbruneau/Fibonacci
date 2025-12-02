// Package fibonacci provides implementations for calculating Fibonacci numbers.
// This file contains the common Matrix Exponentiation framework used by the
// MatrixExponentiation calculator to eliminate code duplication potential.
package fibonacci

import (
	"context"
	"math/big"
	"math/bits"
	"runtime"
)

// MatrixFramework encapsulates the common Matrix Exponentiation algorithm logic.
// The framework manages the binary exponentiation loop and progress reporting.
type MatrixFramework struct{}

// NewMatrixFramework creates a new Matrix Exponentiation framework.
func NewMatrixFramework() *MatrixFramework {
	return &MatrixFramework{}
}

// ExecuteMatrixLoop executes the Matrix Exponentiation algorithm loop.
// This encapsulates the common logic for binary exponentiation of the Fibonacci matrix.
//
// Parameters:
//   - ctx: The context for managing cancellation and deadlines.
//   - reporter: The function used for reporting progress.
//   - n: The index of the Fibonacci number to calculate.
//   - opts: Configuration options for the calculation.
//   - state: The matrix state (must be initialized with res=identity, p=base Q).
//
// Returns:
//   - *big.Int: The calculated Fibonacci number F(n).
//   - error: An error if one occurred (e.g., context cancellation).
func (f *MatrixFramework) ExecuteMatrixLoop(ctx context.Context, reporter ProgressReporter, n uint64, opts Options, state *matrixState) (*big.Int, error) {
	if n == 0 {
		return big.NewInt(0), nil
	}

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

