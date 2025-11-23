package fibonacci

import (
	"context"
	"math/big"
	"math/bits"
	"runtime"
	"sync"
)

// OptimizedFastDoubling provides a high-performance implementation of the "Fast
// Doubling" algorithm for calculating Fibonacci numbers.
// This method is highly efficient, making it one of the fastest known algorithms
// for this purpose.
//
// Formula Derivation:
// The algorithm's identities can be derived from the matrix exponentiation form:
//
//	[ F(n+1) F(n)   ] = [ 1 1 ]^n
//	[ F(n)   F(n-1) ]   [ 1 0 ]
//
// By squaring the matrix for F(k), we get the matrix for F(2k):
//
//	[ F(k+1) F(k) ]^2 = [ F(k+1)²+F(k)²     F(k+1)F(k)+F(k)F(k-1) ]
//	[ F(k)   F(k-1) ]  [ F(k)F(k+1)+F(k-1)F(k) F(k)²+F(k-1)²     ]
//
// This simplifies to the matrix for F(2k):
//
//	[ F(2k+1) F(2k)   ]
//	[ F(2k)   F(2k-1) ]
//
// From this, we extract the two core identities:
//
//	F(2k)   = F(k) * (F(k+1) + F(k-1)) = F(k) * (F(k+1) + (F(k+1) - F(k)))
//	        = F(k) * [2*F(k+1) - F(k)]
//	F(2k+1) = F(k+1)² + F(k)²
//
// Algorithmic Complexity:
// The time complexity is often cited as O(log n), which refers to the number of
// arithmetic operations. However, since we use arbitrary-precision integers
// (`math/big`), the cost of each multiplication dominates. The number of bits in
// F(n) is proportional to n. If M(k) is the time complexity of multiplying two
// k-bit numbers, the total complexity of this algorithm is O(log n * M(n)).
//   - For standard multiplication (Karatsuba), M(n) ≈ O(n^1.585).
//   - For FFT-based multiplication, M(n) ≈ O(n log n).
//
// Optimization Details:
// To achieve maximum performance, this implementation incorporates several
// advanced optimizations:
//   - Zero-Allocation Strategy: By using a sync.Pool, the calculator reuses
//     calculationState objects, which significantly reduces memory allocation
//     and garbage collector overhead during the main loop.
//   - Multi-core Parallelism: For very large numbers (exceeding a configurable
//     `threshold`), the algorithm parallelizes the three core multiplications
//     in the doubling step. This threshold defaults to 4096 bits, a value
//     determined empirically to balance the overhead of goroutine creation
//     against the gains of parallel computation.
//   - Adaptive Multiplication: To handle extremely large numbers efficiently,
//     the calculator dynamically switches to an FFT-based multiplication method
//     when the numbers exceed a specified `fftThreshold`. This threshold
//     defaults to 20000 bits, a conservative value where FFT's superior
//     asymptotic complexity reliably outperforms standard multiplication.
type OptimizedFastDoubling struct{}

// Name returns the descriptive name of the algorithm.
// This name is displayed in the application's user interface, providing a clear
// and concise identification of the calculation method, including its key
// performance characteristics.
//
// It returns a string with the name of the algorithm.
func (fd *OptimizedFastDoubling) Name() string {
	return "Fast Doubling (O(log n), Parallel, Zero-Alloc)"
}

// CalculateCore computes F(n) using the Fast Doubling algorithm.
//
// This function orchestrates the entire calculation process, which includes:
//   - Acquiring a calculationState from the object pool to avoid allocations.
//   - Iterating over the bits of `n` from most significant to least
//     significant.
//   - Reporting progress to the caller.
//   - Returning the final result, F(n).
//
// The context for managing cancellation is ctx. The function for reporting
// progress is reporter. The index of the Fibonacci number to calculate is n.
// The bit size threshold for parallelizing multiplications is threshold. The bit
// size threshold for using FFT-based multiplication is fftThreshold.
//
// It returns the calculated Fibonacci number and an error if one occurred.
func (fd *OptimizedFastDoubling) CalculateCore(ctx context.Context, reporter ProgressReporter, n uint64, threshold int, fftThreshold int) (*big.Int, error) {
	// mul is a closure that performs multiplication.
	// It returns a pointer to the result.
	// If FFT is used, it returns a new *big.Int (allocated).
	// If standard Mul is used, it uses 'dest' for storage and returns it.
	mul := func(dest, x, y *big.Int) *big.Int {
		if fftThreshold > 0 {
			// Use FFT only if both operands exceed the threshold.
			// Shortcut: compare the min of the bit lengths.
			minBitLen := x.BitLen()
			if b := y.BitLen(); b < minBitLen {
				minBitLen = b
			}
			if minBitLen > fftThreshold {
				return mulFFT(x, y)
			}
		}
		return dest.Mul(x, y)
	}

	s := acquireState()
	defer releaseState(s)

	numBits := bits.Len64(n)
	useParallel := runtime.GOMAXPROCS(0) > 1 && threshold > 0

	// Calculate total work for progress reporting via common utility
	totalWork := CalcTotalWork(numBits)
	workDone := 0.0
	lastReportedProgress := -1.0

	for i := numBits - 1; i >= 0; i-- {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// Doubling Step
		s.t2.Lsh(s.f_k1, 1).Sub(s.t2, s.f_k)

		// Parallelize when at least one of the main operands is large
		if useParallel && func() bool {
			bl := s.f_k1.BitLen()
			if b := s.f_k.BitLen(); b > bl {
				bl = b
			}
			// Disable parallel multiplication if FFT is likely to be used,
			// as FFT implementations (like bigfft) often saturate CPU cores,
			// and running them in parallel causes contention.
			// We check if the operands are large enough for FFT.
			// Note: We use the same threshold logic as in mul().
			// If we are using FFT (minBitLen > fftThreshold), we only parallelize
			// if the numbers are huge (e.g. > 10M bits) to overcome concurrency overhead.
			// Benchmarks show that at 7M bits (N=10M), sequential is faster (78ms vs 98ms).
			// At 173M bits (N=250M), parallel is essential.
			// We set a heuristic threshold of 10M bits for parallel FFT.
			if fftThreshold > 0 {
				minBitLen := s.f_k.BitLen()
				if minBitLen > fftThreshold {
					return minBitLen > 10_000_000
				}
			}
			return bl > threshold
		}() {
			// Inline parallel execution to avoid closure allocations
			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				s.t3 = mul(s.t3, s.f_k, s.t2)
				wg.Done()
			}()
			go func() {
				s.t1 = mul(s.t1, s.f_k1, s.f_k1)
				wg.Done()
			}()
			s.t4 = mul(s.t4, s.f_k, s.f_k)
			wg.Wait()
		} else {
			s.t3 = mul(s.t3, s.f_k, s.t2)
			s.t1 = mul(s.t1, s.f_k1, s.f_k1)
			s.t4 = mul(s.t4, s.f_k, s.f_k)
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

		// Harmonized reporting via common utility function
		workDone = ReportStepProgress(reporter, &lastReportedProgress, totalWork, workDone, i, numBits)
	}
	return new(big.Int).Set(s.f_k), nil
}

// parallelMultiply3Optimized leverages concurrency to accelerate the three key
// multiplications of the doubling step. By executing these multiplications in
// parallel, this function takes advantage of multi-core processors, leading to
// significant performance improvements for very large numbers.
func parallelMultiply3Optimized(s *calculationState, mul func(dest, x, y *big.Int) *big.Int) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		s.t3 = mul(s.t3, s.f_k, s.t2)
	}()
	go func() {
		defer wg.Done()
		s.t1 = mul(s.t1, s.f_k1, s.f_k1)
	}()
	s.t4 = mul(s.t4, s.f_k, s.f_k)
	wg.Wait()
}

// calculationState aggregates temporary variables for the "Fast Doubling"
// algorithm, allowing efficient management via an object pool.
type calculationState struct {
	f_k, f_k1, t1, t2, t3, t4 *big.Int
}

// Reset prepares the state for a new calculation.
// It initializes f_k to 0 and f_k1 to 1, which are the base values for the
// Fast Doubling algorithm.
func (s *calculationState) Reset() {
	s.f_k.SetInt64(0)
	s.f_k1.SetInt64(1)
}

var statePool = sync.Pool{
	New: func() interface{} {
		return &calculationState{
			f_k:  new(big.Int),
			f_k1: new(big.Int),
			t1:   new(big.Int),
			t2:   new(big.Int),
			t3:   new(big.Int),
			t4:   new(big.Int),
		}
	},
}

// acquireState gets a state from the pool and resets it.
func acquireState() *calculationState {
	s := statePool.Get().(*calculationState)
	s.Reset()
	return s
}

// releaseState puts a state back into the pool.
func releaseState(s *calculationState) {
	statePool.Put(s)
}
