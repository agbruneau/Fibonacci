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
func (c *FFTBasedCalculator) CalculateCore(ctx context.Context, reporter ProgressReporter, n uint64, threshold int, fftThreshold int) (*big.Int, error) {
	s := acquireState()
	defer releaseState(s)

	numBits := bits.Len64(n)

	var totalWork, workDone, workOfStep, four big.Int
	four.SetInt64(4)
	if numBits > 0 {
		totalWork.Exp(&four, big.NewInt(int64(numBits)), nil).Sub(&totalWork, big.NewInt(1)).Div(&totalWork, big.NewInt(3))
	}
	lastReportedProgress := -1.0
	const reportThreshold = 0.01

	for i := numBits - 1; i >= 0; i-- {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// Doubling Step
		s.t2.Lsh(s.f_k1, 1).Sub(s.t2, s.f_k)
		mulFFT(s.t3, s.f_k, s.t2)
		mulFFT(s.t1, s.f_k1, s.f_k1)
		mulFFT(s.t4, s.f_k, s.f_k)
		s.f_k.Set(s.t3)
		s.f_k1.Add(s.t1, s.t4)

		// Addition Step: If the i-th bit of n is 1, update F(k) and F(k+1)
		// F(k) <- F(k+1)
		// F(k+1) <- F(k) + F(k+1)
		if (n>>uint(i))&1 == 1 {
			// s.t1 temporarily stores the new F(k+1)
			s.t1.Add(s.f_k, s.f_k1)
			// s.f_k becomes the old s.f_k1
			s.f_k.Set(s.f_k1)
			// s.f_k1 takes the new value s.t1
			s.f_k1.Set(s.t1)
		}

		if totalWork.Sign() > 0 {
			j := int64(numBits - 1 - i)
			workOfStep.Exp(&four, big.NewInt(j), nil)
			workDone.Add(&workDone, &workOfStep)
			workDoneFloat, _ := new(big.Float).SetInt(&workDone).Float64()
			totalWorkFloat, _ := new(big.Float).SetInt(&totalWork).Float64()
			currentProgress := workDoneFloat / totalWorkFloat
			if currentProgress-lastReportedProgress >= reportThreshold || i == 0 {
				reporter(currentProgress)
				lastReportedProgress = currentProgress
			}
		}
	}
	return new(big.Int).Set(s.f_k), nil
}