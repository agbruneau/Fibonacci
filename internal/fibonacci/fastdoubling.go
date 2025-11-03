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
//
//	F(2k)   = F(k) * [2*F(k+1) - F(k)]
//	F(2k+1) = F(k)² + F(k+1)²
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

// Name returns the descriptive name of the algorithm. This name is displayed in
// the application's user interface, providing a clear and concise identification
// of the calculation method, including its key performance characteristics.
func (fd *OptimizedFastDoubling) Name() string {
	return "Fast Doubling (O(log n), Parallel, Zero-Alloc)"
}

// CalculateCore computes F(n) using the Fast Doubling algorithm.
//
// This function orchestrates the entire calculation process, which includes:
//   - Acquiring a `calculationState` from the object pool to avoid allocations.
//   - Iterating over the bits of `n` from most significant to least significant.
//   - Reporting progress to the caller.
//   - Returning the final result, F(n).
//
// Parameters:
//   - ctx: The context for managing cancellation.
//   - reporter: The function for reporting progress.
//   - n: The index of the Fibonacci number to calculate.
//   - threshold: The bit size threshold for parallelizing multiplications.
//   - fftThreshold: The bit size threshold for using FFT-based multiplication.
//
// Returns the calculated Fibonacci number and an error if one occurred.
func (fd *OptimizedFastDoubling) CalculateCore(ctx context.Context, reporter ProgressReporter, n uint64, threshold int, fftThreshold int) (*big.Int, error) {
	mul := func(dest, x, y *big.Int) {
		if fftThreshold > 0 {
			// Utilise FFT uniquement si les deux opérandes dépassent le seuil.
			// Raccourci: compare le min des tailles en bits.
			minBitLen := x.BitLen()
			if b := y.BitLen(); b < minBitLen {
				minBitLen = b
			}
			if minBitLen > fftThreshold {
				mulFFT(dest, x, y)
				return
			}
		}
		dest.Mul(x, y)
	}

	s := acquireState()
	defer releaseState(s)

	numBits := bits.Len64(n)
	// Cache la vérification de parallélisme pour éviter les appels répétés
	maxProcs := runtime.GOMAXPROCS(0)
	useParallel := maxProcs > 1 && threshold > 0

	// Calcul du travail total pour le reporting de progression via utilitaire commun
	totalWork := CalcTotalWork(numBits)
	var workDone, workOfStep big.Int
	lastReportedProgress := -1.0

	for i := numBits - 1; i >= 0; i-- {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// Doubling Step
		s.t2.Lsh(s.f_k1, 1).Sub(s.t2, s.f_k)

		// Parallélise lorsque au moins l'un des opérandes principaux est volumineux
		if useParallel && func() bool {
			bl := s.f_k1.BitLen()
			if b := s.f_k.BitLen(); b > bl {
				bl = b
			}
			return bl > threshold
		}() {
			parallelMultiply3Optimized(s, mul)
		} else {
			mul(s.t3, s.f_k, s.t2)
			mul(s.t1, s.f_k1, s.f_k1)
			mul(s.t4, s.f_k, s.f_k)
		}

		// F(2k+1) = F(k+1)² + F(k)². Store result in t2, which is free.
		s.t2.Add(s.t1, s.t4)
		// Swap the pointers for the next iteration.
		// f_k becomes F(2k) (from t3), f_k1 becomes F(2k+1) (from t2).
		// t2 and t3 become the old f_k and f_k1, now temporaries.
		s.f_k, s.f_k1, s.t2, s.t3 = s.t3, s.t2, s.f_k, s.f_k1

		// Addition Step: If the i-th bit of n is 1, update F(k) and F(k+1)
		// F(k) <- F(k+1)
		// F(k+1) <- F(k) + F(k+1)
		if (n>>uint(i))&1 == 1 {
			// s.t1 temporarily stores the new F(k+1)
			s.t1.Add(s.f_k, s.f_k1)
			// Swap pointers to avoid large allocations:
			// s.f_k becomes the old s.f_k1
			// s.f_k1 becomes the new sum (s.t1)
			// s.t1 becomes the old s.f_k, now a temporary
			s.f_k, s.f_k1, s.t1 = s.f_k1, s.t1, s.f_k
		}

		// Reporting harmonisé via fonction utilitaire commune
		ReportStepProgress(reporter, &lastReportedProgress, totalWork, &workDone, &workOfStep, i, numBits)
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
