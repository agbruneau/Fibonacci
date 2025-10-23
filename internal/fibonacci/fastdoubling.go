package fibonacci

import (
	"context"
	"math/big"
	"math/bits"
	"runtime"
	"sync"
)

// OptimizedFastDoubling provides a high-performance implementation of the "Fast Doubling"
// algorithm for calculating Fibonacci numbers. This method is highly efficient,
// boasting a time complexity of O(log n), making it one of the fastest known
// algorithms for this purpose.
//
// At its core, the algorithm relies on two mathematical identities:
//   F(2k)   = F(k) * [2*F(k+1) - F(k)]
//   F(2k+1) = F(k)² + F(k+1)²
//
// The calculation proceeds by examining the binary representation of the input `n`,
// from the most significant bit to the least. For each bit, a "doubling" step
// is performed, which computes F(2k) and F(2k+1) from the previously calculated
// F(k) and F(k+1). If the current bit is 1, an additional "addition" step is
// performed to advance the calculation.
//
// To achieve maximum performance, this implementation incorporates several advanced
// optimizations:
//   - Zero-Allocation Strategy: By using a `sync.Pool`, the calculator reuses
//     `calculationState` objects, which significantly reduces memory allocation
//     and garbage collector overhead.
//   - Multi-core Parallelism: For very large numbers (exceeding a configurable bit
//     threshold), the algorithm parallelizes the three core multiplications in the
//     doubling step, taking full advantage of modern multi-core processors.
//   - Adaptive Multiplication: To handle extremely large numbers efficiently, the
//     calculator dynamically switches to an FFT-based multiplication method when
//     the numbers exceed a specified `fftThreshold`.
type OptimizedFastDoubling struct{}

// Name returns the descriptive name of the algorithm.
func (fd *OptimizedFastDoubling) Name() string {
	return "Fast Doubling (O(log n), Parallel, Zero-Alloc)"
}

// CalculateCore computes F(n) using the Fast Doubling algorithm.
//
// This function orchestrates the entire calculation process, which includes:
// - Acquiring a `calculationState` from the object pool to avoid allocations.
// - Iterating over the bits of `n` from most significant to least significant.
// - Reporting progress to the caller.
// - Returning the final result, F(n).
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

// parallelMultiply3Optimized leverages concurrency to accelerate the three key
// multiplications of the doubling step. By executing these multiplications in
// parallel, this function takes advantage of multi-core processors, leading to
// significant performance improvements for very large numbers.
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