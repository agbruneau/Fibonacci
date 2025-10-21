package fibonacci

import (
	"context"
	"math/big"
	"math/bits"
	"runtime"
	"sync"
)

// OptimizedFastDoubling implements the `coreCalculator` interface using the
// "Fast Doubling" algorithm.
//
// The "Fast Doubling" algorithm is based on the following identities:
// F(2k) = F(k) * [2*F(k+1) - F(k)]
// F(2k+1) = F(k)² + F(k+1)²
// It iterates over the bits of `n` from most significant to least significant,
// performing a "doubling" step and, if the bit is 1, an "addition" step,
// achieving a complexity of O(log n).
type OptimizedFastDoubling struct{}

// Name returns the name of the algorithm.
func (fd *OptimizedFastDoubling) Name() string {
	return "Fast Doubling (O(log n), Parallel, Zero-Alloc)"
}

// CalculateCore executes the calculation of F(n) using the "Fast Doubling"
// algorithm. It optimizes memory management by using an object pool for
// calculation states and parallelizes multiplications of large numbers.
func (fd *OptimizedFastDoubling) CalculateCore(ctx context.Context, reporter ProgressReporter, n uint64, threshold int, fftThreshold int) (*big.Int, error) {
	mul := func(dest, x, y *big.Int) {
		if fftThreshold > 0 && x.BitLen() > fftThreshold && y.BitLen() > fftThreshold {
			mulFFT(dest, x, y)
		} else {
			dest.Mul(x, y)
		}
	}

	s := acquireState()
	defer releaseState(s)

	numBits := bits.Len64(n)
	useParallel := runtime.GOMAXPROCS(0) > 1 && threshold > 0

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

		if useParallel && s.f_k1.BitLen() > threshold {
			parallelMultiply3Optimized(s, mul)
		} else {
			mul(s.t3, s.f_k, s.t2)
			mul(s.t1, s.f_k1, s.f_k1)
			mul(s.t4, s.f_k, s.f_k)
		}

		s.f_k.Add(s.t1, s.t4)
		s.f_k, s.f_k1, s.t3 = s.t3, s.f_k, s.f_k1

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

// parallelMultiply3Optimized executes the three multiplications of the
// doubling step in parallel to optimize performance on multi-core machines.
func parallelMultiply3Optimized(s *calculationState, mul func(dest, x, y *big.Int)) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		mul(s.t3, s.f_k, s.t2)
	}()
	go func() {
		defer wg.Done()
		mul(s.t1, s.f_k1, s.f_k1)
	}()
	mul(s.t4, s.f_k, s.f_k)
	wg.Wait()
}