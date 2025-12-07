package fibonacci

import (
	"context"
	"math/big"
)

// FFTBasedCalculator is a specialized Fibonacci calculator that uses the Fast
// Doubling algorithm, but with a significant modification: it exclusively relies
// on FFT-based multiplication for all big.Int operations.
//
// Unlike the OptimizedFastDoubling calculator, which adaptively switches
// between standard and FFT-based multiplication, this implementation uses
// mulFFT for every multiplication, regardless of the numbers' size. This makes
// it an excellent tool for benchmarking the performance of FFT-based
// multiplication in Fibonacci calculations. It is also particularly effective
// for computing exceptionally large Fibonacci numbers, where FFT-based methods
// are consistently faster.
type FFTBasedCalculator struct{}

// Name returns the name of the algorithm, indicating its reliance on FFT.
//
// Returns:
//   - string: The name of the algorithm.
func (c *FFTBasedCalculator) Name() string {
	return "FFT-Based Doubling"
}

// CalculateCore computes F(n) using the Fast Doubling algorithm, with all
// multiplications performed via FFT.
//
// This implementation uses the DoublingFramework with FFTOnlyStrategy to
// consistently use FFT-based multiplication for all operations, regardless
// of operand size. This design makes it ideal for scenarios where FFT is
// expected to be the most performant option, such as with extremely large
// numbers.
//
// Parameters:
//   - ctx: The context for managing cancellation and deadlines.
//   - reporter: The function used for reporting progress.
//   - n: The index of the Fibonacci number to calculate.
//   - opts: Configuration options for the calculation (thresholds ignored).
//
// Returns:
//   - *big.Int: The calculated Fibonacci number.
//   - error: An error if one occurred (e.g., context cancellation).
func (c *FFTBasedCalculator) CalculateCore(ctx context.Context, reporter ProgressReporter, n uint64, opts Options) (*big.Int, error) {
	s := acquireState()
	defer releaseState(s)

	// Use framework with FFT-only strategy
	strategy := &FFTOnlyStrategy{}
	framework := NewDoublingFramework(strategy)

	// Execute the doubling loop (no parallelization for FFT-based)
	return framework.ExecuteDoublingLoop(ctx, reporter, n, opts, s, false)
}
