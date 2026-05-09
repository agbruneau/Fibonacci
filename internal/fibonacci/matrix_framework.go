// This file contains the common Matrix Exponentiation framework used by the
// MatrixExponentiationCalculator calculator to eliminate code duplication potential.

package fibonacci

import (
	"context"
	"fmt"
	"math/big"
	"math/bits"
	"runtime"

	"github.com/agbru/fibcalc/internal/progress"
)

// SquareSymmetricMatrixFunc is the function signature for symmetric matrix squaring.
// It is injectable on MatrixFramework for testing purposes.
type SquareSymmetricMatrixFunc func(ctx context.Context, dest, mat *matrix, state *matrixState, inParallel bool, fftThreshold int) error

// MatrixFramework encapsulates the common Matrix Exponentiation algorithm logic.
// The framework manages the binary exponentiation loop and progress reporting.
//
// The SquareFunc field allows injection of a custom symmetric matrix squaring
// function for testing. By default, it uses the optimized squareSymmetricMatrix.
type MatrixFramework struct {
	SquareFunc SquareSymmetricMatrixFunc
}

// NewMatrixFramework creates a new Matrix Exponentiation framework
// with the default squareSymmetricMatrix implementation.
func NewMatrixFramework() *MatrixFramework {
	return &MatrixFramework{
		SquareFunc: squareSymmetricMatrix,
	}
}

// NewMatrixFrameworkWithSquareFunc creates a new Matrix Exponentiation framework
// with a custom symmetric matrix squaring function, useful for testing.
func NewMatrixFrameworkWithSquareFunc(fn SquareSymmetricMatrixFunc) *MatrixFramework {
	if fn == nil {
		fn = squareSymmetricMatrix
	}
	return &MatrixFramework{
		SquareFunc: fn,
	}
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
func (f *MatrixFramework) ExecuteMatrixLoop(ctx context.Context, reporter progress.ProgressCallback, n uint64, opts Options, state *matrixState) (*big.Int, error) {
	if n == 0 {
		return big.NewInt(0), nil
	}

	exponent := n - 1
	numBits := bits.Len64(exponent)
	// Normalize options to ensure consistent default threshold handling
	normalizedOpts := normalizeOptions(opts)
	useParallel := runtime.NumCPU() > 1 && normalizedOpts.ParallelThreshold > 0

	// Calculate total work for progress reporting via common utility
	totalWork := progress.CalcTotalWork(numBits)
	// Pre-compute powers of 4 for O(1) progress calculation
	powers := progress.PrecomputePowers4(numBits)
	workDone := 0.0
	lastReportedProgress := -1.0

	for i := 0; i < numBits; i++ {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("matrix exponentiation calculation canceled at bit %d/%d: %w", i, numBits-1, err)
		}

		// i is a bit index in [0, bits.Len64(exponent)-1] ⊂ [0, 63], uint cast safe. #nosec G115
		if (exponent>>uint(i))&1 == 1 {
			// Decide on parallelism based on the max size of the operands involved
			inParallel := useParallel && maxBitLenMatrix(state.p) > normalizedOpts.ParallelThreshold
			if err := multiplyMatrices(ctx, state.tempMatrix, state.res, state.p, state, inParallel, normalizedOpts.FFTThreshold, normalizedOpts.StrassenThreshold); err != nil {
				return nil, fmt.Errorf("matrix multiplication failed at bit %d/%d: %w", i, numBits-1, err)
			}
			state.res, state.tempMatrix = state.tempMatrix, state.res
		}

		if i < numBits-1 {
			inParallel := useParallel && maxBitLenMatrix(state.p) > normalizedOpts.ParallelThreshold
			if err := f.SquareFunc(ctx, state.tempMatrix, state.p, state, inParallel, normalizedOpts.FFTThreshold); err != nil {
				return nil, fmt.Errorf("matrix squaring failed at bit %d/%d: %w", i, numBits-1, err)
			}
			state.p, state.tempMatrix = state.tempMatrix, state.p
		}

		// Harmonized reporting via common utility function
		// For Matrix Exponentiation, we iterate from LSB (small work) to MSB (large work).
		// However, ReportStepProgress assumes `i` counts down from MSB (large work) to LSB.
		// To correct this, we invert the index passed to ReportStepProgress so that
		// stepIndex becomes `i`, resulting in increasing work values.
		workDone = progress.ReportStepProgress(reporter, &lastReportedProgress, totalWork, workDone, numBits-1-i, numBits, powers)
	}
	// Optimization: Avoid copying the entire result by "stealing" res.a from
	// the matrix state. We replace it with a fresh empty big.Int so the state
	// remains valid for pool return via releaseMatrixState. This eliminates an
	// O(n) copy of the result, trading it for a single 24-byte allocation.
	result := state.res.a
	state.res.a = new(big.Int)
	return result, nil
}
