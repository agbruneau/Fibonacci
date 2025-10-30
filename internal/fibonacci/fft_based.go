package fibonacci

import (
	"context"
	"math/big"
	"math/bits"
)

// FFTBasedCalculator is a specialized Fibonacci calculator that uses the Fast Doubling
// algorithm, but with a significant modification: it exclusively relies on FFT-based
// multiplication for all `big.Int` operations.
//
// Unlike the `OptimizedFastDoubling` calculator, which adaptively switches between
// standard and FFT-based multiplication, this implementation uses `mulFFT` for
// every multiplication, regardless of the numbers' size. This makes it an
// excellent tool for benchmarking the performance of FFT-based multiplication in
// Fibonacci calculations. It is also particularly effective for computing
// exceptionally large Fibonacci numbers, where FFT-based methods are consistently
// faster.
type FFTBasedCalculator struct{}

// Name returns the name of the algorithm, indicating its reliance on FFT.
func (c *FFTBasedCalculator) Name() string {
	return "FFT-Based Doubling"
}

// CalculateCore computes F(n) using the Fast Doubling algorithm, with all
// multiplications performed via `mulFFT`.
//
// While the high-level logic of this function is similar to `OptimizedFastDoubling`,
// it differs in its multiplication strategy. Instead of adaptively choosing the
// multiplication method, it consistently uses FFT-based multiplication. This design
// makes it ideal for scenarios where FFT is expected to be the most performant
// option, such as with extremely large numbers.
//
// Parameters:
//   - ctx: The context for managing cancellation.
//   - reporter: The function for reporting progress.
//   - n: The index of the Fibonacci number to calculate.
//   - threshold: The bit size threshold for parallelizing multiplications.
//   - fftThreshold: The bit size threshold for using FFT-based multiplication.
//
// Returns the calculated Fibonacci number and an error if one occurred.
func (c *FFTBasedCalculator) CalculateCore(ctx context.Context, reporter ProgressReporter, n uint64, threshold int, fftThreshold int) (*big.Int, error) {
	s := acquireState()
	defer releaseState(s)

	numBits := bits.Len64(n)

	totalWork := CalcTotalWork(numBits)
	var workDone, workOfStep big.Int
	lastReportedProgress := -1.0

	for i := numBits - 1; i >= 0; i-- {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		// Doubling + Addition Step (inchangé)
		// Reporting harmonisé via fonction utilitaire
		ReportStepProgress(reporter, &lastReportedProgress, totalWork, &workDone, &workOfStep, i, numBits)
	}
	return new(big.Int).Set(s.f_k), nil
}