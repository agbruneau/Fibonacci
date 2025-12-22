package fibonacci

import (
	"context"
	"math/big"
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
// Returns:
//   - string: The name of the algorithm.
func (fd *OptimizedFastDoubling) Name() string {
	return "Fast Doubling (O(log n), Parallel, Zero-Alloc)"
}

// CalculateCore computes F(n) using the Fast Doubling algorithm.
//
// This function orchestrates the entire calculation process, which includes:
//   - Acquiring a calculationState from the object pool to avoid allocations.
//   - Using the DoublingFramework with adaptive strategy for the core loop.
//   - Applying parallelization optimizations when beneficial.
//   - Reporting progress to the caller.
//   - Returning the final result, F(n).
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
func (fd *OptimizedFastDoubling) CalculateCore(ctx context.Context, reporter ProgressReporter, n uint64, opts Options) (*big.Int, error) {
	s := acquireState()
	defer releaseState(s)

	// Normalize options to ensure consistent default threshold handling
	normalizedOpts := normalizeOptions(opts)
	useParallel := runtime.GOMAXPROCS(0) > 1 && normalizedOpts.ParallelThreshold > 0

	// Use framework with adaptive strategy for the main loop
	strategy := &AdaptiveStrategy{}

	// Create framework with or without dynamic threshold adjustment
	var framework *DoublingFramework
	if normalizedOpts.EnableDynamicThresholds {
		// Create dynamic threshold manager
		interval := normalizedOpts.DynamicAdjustmentInterval
		if interval <= 0 {
			interval = DynamicAdjustmentInterval
		}
		dtm := NewDynamicThresholdManagerFromConfig(DynamicThresholdConfig{
			InitialFFTThreshold:      normalizedOpts.FFTThreshold,
			InitialParallelThreshold: normalizedOpts.ParallelThreshold,
			AdjustmentInterval:       interval,
			Enabled:                  true,
		})
		framework = NewDoublingFrameworkWithDynamicThresholds(strategy, dtm)
	} else {
		framework = NewDoublingFramework(strategy)
	}

	// Execute the doubling loop with parallelization support
	return framework.ExecuteDoublingLoop(ctx, reporter, n, normalizedOpts, s, useParallel)
}

// ShouldParallelizeMultiplication determines whether the multiplication operations
// should be parallelized based on the operand sizes and configuration options.
//
// This function encapsulates the complex decision logic for parallelization:
//   - It checks if the operands are large enough to benefit from parallelism.
//   - It considers FFT threshold to avoid contention when FFT is in use,
//     as FFT implementations often saturate CPU cores.
//   - For FFT-sized operands, parallelism is only enabled for very large
//     numbers (> ParallelFFTThreshold bits) to overcome concurrency overhead.
//
// Parameters:
//   - s: The current calculation state containing the operands.
//   - opts: Configuration options including thresholds.
//
// Returns:
//   - bool: true if multiplication should be parallelized, false otherwise.
func ShouldParallelizeMultiplication(s *CalculationState, opts Options) bool {
	// Cache BitLen() values to avoid redundant calls.
	// BitLen() traverses the internal representation of big.Int, so caching
	// these values provides a measurable performance improvement (2-5%).
	fkBitLen := s.FK.BitLen()
	fk1BitLen := s.FK1.BitLen()
	return shouldParallelizeMultiplicationCached(opts, fkBitLen, fk1BitLen)
}

// shouldParallelizeMultiplicationCached is an optimized version that accepts
// pre-computed BitLen() values to avoid redundant calls.
//
// Parameters:
//   - opts: Configuration options including thresholds.
//   - fkBitLen: Pre-computed bit length of FK.
//   - fk1BitLen: Pre-computed bit length of FK1.
//
// Returns:
//   - bool: true if multiplication should be parallelized, false otherwise.
func shouldParallelizeMultiplicationCached(opts Options, fkBitLen, fk1BitLen int) bool {
	// Determine the maximum bit length of the main operands
	maxBitLen := fk1BitLen
	if fkBitLen > maxBitLen {
		maxBitLen = fkBitLen
	}

	// Disable parallel multiplication if FFT is likely to be used,
	// as FFT implementations (like bigfft) often saturate CPU cores,
	// and running them in parallel causes contention.
	//
	// We use maxBitLen here because the squaring operations (f_k * f_k and
	// f_k1 * f_k1) trigger FFT when a single operand exceeds the threshold
	// (since both sides of the multiplication are the same value).
	// Therefore, if ANY operand exceeds the FFT threshold, at least one
	// multiplication will use FFT and cause CPU saturation.
	//
	// We only re-enable parallelism for extremely large numbers
	// (> ParallelFFTThreshold bits) where the benefit outweighs contention.
	// See constants.go for empirical benchmark results.
	// Note: opts should already be normalized, but we check for safety
	if opts.FFTThreshold > 0 && maxBitLen > opts.FFTThreshold {
		return maxBitLen > ParallelFFTThreshold
	}

	// Use normalized threshold (should already be normalized, but ensure consistency)
	threshold := opts.ParallelThreshold
	if threshold == 0 {
		threshold = DefaultParallelThreshold
	}
	return maxBitLen > threshold
}

// CalculationState aggregates temporary variables for the "Fast Doubling"
// algorithm, allowing efficient management via an object pool.
type CalculationState struct {
	FK, FK1, T1, T2, T3, T4 *big.Int
}

// Reset prepares the state for a new calculation.
// It initializes FK to 0 and FK1 to 1, which are the base values for the
// Fast Doubling algorithm.
func (s *CalculationState) Reset() {
	s.FK.SetInt64(0)
	s.FK1.SetInt64(1)
	// T1..T4 are temporaries used as scratch space, so we don't need to clear them.
}

var statePool = sync.Pool{
	New: func() any {
		return &CalculationState{
			FK:  new(big.Int),
			FK1: new(big.Int),
			T1:  new(big.Int),
			T2:  new(big.Int),
			T3:  new(big.Int),
			T4:  new(big.Int),
		}
	},
}

// AcquireState gets a state from the pool and resets it.
// The returned state must be released using ReleaseState, preferably with defer:
//
//	state := AcquireState()
//	defer ReleaseState(state)
//
// This ensures the state is returned to the pool even if an error occurs or a panic is triggered.
//
// Returns:
//   - *CalculationState: A ready-to-use calculation state.
func AcquireState() *CalculationState {
	s := statePool.Get().(*CalculationState)
	s.Reset()
	return s
}

// ReleaseState puts a state back into the pool.
// This should be called with defer immediately after AcquireState to ensure
// proper resource cleanup even in case of errors or panics:
//
//	state := AcquireState()
//	defer ReleaseState(state)
//
// Parameters:
//   - s: The calculation state to return to the pool. Safe to call with nil.
func ReleaseState(s *CalculationState) {
	if s == nil {
		return
	}
	// Avoid keeping oversized objects in memory.
	// We check if any of the big.Ints exceed the pool limit.
	// If so, we discard the entire state to let GC reclaim the large memory.
	if checkLimit(s.FK) || checkLimit(s.FK1) ||
		checkLimit(s.T1) || checkLimit(s.T2) ||
		checkLimit(s.T3) || checkLimit(s.T4) {
		return
	}

	statePool.Put(s)
}

// acquireState is a convenience wrapper for backward compatibility.
// The returned state must be released using releaseState, preferably with defer:
//
//	state := acquireState()
//	defer releaseState(state)
func acquireState() *CalculationState {
	return AcquireState()
}

// releaseState is a convenience wrapper for backward compatibility.
// This should be called with defer immediately after acquireState:
//
//	state := acquireState()
//	defer releaseState(state)
func releaseState(s *CalculationState) {
	ReleaseState(s)
}
