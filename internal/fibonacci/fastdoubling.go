package fibonacci

import (
	"context"
	"math"
	"math/big"
	"runtime"
	"sync"

	"github.com/agbruneau/FibGo/internal/fibonacci/memory"
	"github.com/agbruneau/FibGo/internal/fibonacci/threshold"
	"github.com/agbruneau/FibGo/internal/progress"
)

// maxArenaPoolWords caps the size of a CalculationArena retained in the pool
// via a CalculationState. Above this threshold the arena is dropped to GC
// rather than being kept indefinitely, so a single huge call (e.g. F(50M+))
// does not pin ~400 MB of backing memory in the pool forever.
const maxArenaPoolWords = 50_000_000

// FastDoublingCalculator provides a high-performance implementation of the "Fast
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
//   - For standard multiplication (math/big, which uses Karatsuba internally), M(n) ≈ O(n^1.585).
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
//     defaults to 500,000 bits, a conservative value where FFT's superior
//     asymptotic complexity reliably outperforms standard multiplication.
type FastDoublingCalculator struct{}

// Name returns the descriptive name of the algorithm.
// This name is displayed in the application's user interface, providing a clear
// and concise identification of the calculation method, including its key
// performance characteristics.
//
// Returns:
//   - string: The name of the algorithm.
func (fd *FastDoublingCalculator) Name() string {
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
func (fd *FastDoublingCalculator) CalculateCore(ctx context.Context, reporter progress.ProgressCallback, n uint64, opts Options) (*big.Int, error) {
	// AcquireStateForN reuses (or grows) the state's bound arena and pre-sizes
	// all big.Int slots from it. The arena is owned by the state, so the
	// invariant "no big.Int alias outlives its backing arena" is enforced
	// at Release time — see ReleaseStateWithResult.
	s := AcquireStateForN(n)

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
			interval = threshold.DynamicAdjustmentInterval
		}
		dtm := threshold.NewDynamicThresholdManagerFromConfig(threshold.DynamicThresholdConfig{
			InitialFFTThreshold:      normalizedOpts.FFTThreshold,
			InitialParallelThreshold: normalizedOpts.ParallelThreshold,
			AdjustmentInterval:       interval,
			Enabled:                  true,
		})
		framework = NewDoublingFrameworkWithDynamicThresholds(strategy, dtm)
	} else {
		framework = NewDoublingFramework(strategy)
	}

	// Execute the doubling loop with parallelization support.
	// On error we must release without preserving a result; on success we
	// hand off to ReleaseStateWithResult which deep-copies the result out
	// of the arena before returning the state (and arena) to the pool.
	raw, err := framework.ExecuteDoublingLoop(ctx, reporter, n, normalizedOpts, s, useParallel)
	if err != nil {
		ReleaseState(s)
		return nil, err
	}
	return ReleaseStateWithResult(s, raw), nil
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
// Uses 5 big.Int: FK (F(k)), FK1 (F(k+1)), and T1-T3 for intermediate
// multiplication results.
//
// P1-04: the state owns its CalculationArena. By tying the two pools together
// we make the invariant "no pooled big.Int header may alias an arena buffer
// reused by a different state" trivially enforceable: the arena is acquired
// and released atomically with the state, and Release detaches every aliased
// slot before resetting the arena.
type CalculationState struct {
	FK, FK1, T1, T2, T3 *big.Int
	arena               *memory.CalculationArena
	arenaCapWords       int // cached len(arena.buf) to avoid a method call on hot paths
}

// Reset prepares the state for a new calculation.
// It initializes FK to 0 and FK1 to 1, which are the base values for the
// Fast Doubling algorithm.
//
// Note: Reset deliberately does NOT touch arena/arenaCapWords. The arena is
// part of the state's pooled identity (lifecycle managed by AcquireStateForN
// and ReleaseStateWithResult/ReleaseState), not per-call scratch.
func (s *CalculationState) Reset() {
	s.FK.SetInt64(0)
	s.FK1.SetInt64(1)
	// T1..T3 are temporaries used as scratch space, so we don't need to clear them.
}

var statePool = sync.Pool{
	New: func() any {
		return &CalculationState{
			FK:  new(big.Int),
			FK1: new(big.Int),
			T1:  new(big.Int),
			T2:  new(big.Int),
			T3:  new(big.Int),
		}
	},
}

// AcquireState gets a state from the pool and resets it. It does NOT pre-size
// the state's big.Ints from an arena — use AcquireStateForN for that. This
// entry point exists for call sites (notably tests) that do not know n at
// acquisition time.
//
// The returned state must be released using ReleaseState, preferably with defer:
//
//	state := AcquireState()
//	defer ReleaseState(state)
//
// Returns:
//   - *CalculationState: A ready-to-use calculation state.
func AcquireState() *CalculationState {
	s := statePool.Get().(*CalculationState)
	s.Reset()
	return s
}

// maxReasonableWords caps arena sizing well above any computable F(n)
// (≈1.15e18 words ≈ 9 EB). It exists only to keep the float->int conversion
// and the ×15 multiplication in acquireSizingForN from invoking impl-defined
// behavior / integer overflow on a physically impossible n — defense-in-depth.
const maxReasonableWords = 1 << 60

// acquireSizingForN computes (wordsNeeded, totalWords) for F(n): one big.Int of
// ~n*FibonacciGrowthFactor bits, with totalWords sized for 15 temporaries to
// match NewCalculationArena. The clamp only changes the result for n far beyond
// the computable range; for realistic n it is bit-identical to the naive
// estimatedBits/64+1 then ×15 computation.
func acquireSizingForN(n uint64) (wordsNeeded, totalWords int) {
	estimatedFloat := float64(n) * FibonacciGrowthFactor
	// A float64 outside the int range yields an impl-defined value on
	// conversion; clamp before converting.
	if estimatedFloat >= float64(math.MaxInt64) {
		return maxReasonableWords / 15, maxReasonableWords
	}
	wordsNeeded = int(estimatedFloat)/64 + 1
	if wordsNeeded > maxReasonableWords/15 {
		return wordsNeeded, maxReasonableWords
	}
	return wordsNeeded, wordsNeeded * 15
}

// AcquireStateForN gets a state from the pool, resets it, and pre-sizes its
// big.Int slots from the state's bound CalculationArena. For n > 1000 the
// arena is reused if it is large enough; otherwise a fresh arena is allocated.
// For n <= 1000 no pre-sizing is performed and any existing arena is reset.
//
// The returned state must be released via ReleaseStateWithResult on success
// (which deep-copies the result out of the arena) or ReleaseState on the
// error path.
func AcquireStateForN(n uint64) *CalculationState {
	s := statePool.Get().(*CalculationState)
	s.Reset()

	if n > 1000 {
		wordsNeeded, totalWords := acquireSizingForN(n)

		if s.arena == nil || s.arenaCapWords < totalWords {
			s.arena = memory.NewCalculationArena(n)
			s.arenaCapWords = s.arena.CapacityWords()
		} else {
			s.arena.Reset()
		}

		s.arena.PreSizeFromArena(s.FK, wordsNeeded)
		s.arena.PreSizeFromArena(s.FK1, wordsNeeded)
		// SetInt64 must come AFTER PreSizeFromArena: PreSizeFromArena calls
		// big.Int.SetBits with a length-0 slice (zeroing the value), so we
		// re-establish FK=0, FK1=1 after the arena hand-off.
		s.FK.SetInt64(0)
		s.FK1.SetInt64(1)
		s.arena.PreSizeFromArena(s.T1, wordsNeeded)
		s.arena.PreSizeFromArena(s.T2, wordsNeeded)
		s.arena.PreSizeFromArena(s.T3, wordsNeeded)
	} else if s.arena != nil {
		// Even on the small-N path we reset the arena so any previous
		// large-N tenancy is cleared before potential reuse.
		s.arena.Reset()
	}

	return s
}

// clearStateAliases re-points every big.Int slot in s to a fresh, empty
// big.Int. This is the critical step before returning the state to the pool:
// it severs every alias from a slot's Bits() into the arena's backing buffer,
// so a subsequent acquisition can safely Reset() and reuse the arena.
func clearStateAliases(s *CalculationState) {
	s.FK = new(big.Int)
	s.FK1 = new(big.Int)
	s.T1 = new(big.Int)
	s.T2 = new(big.Int)
	s.T3 = new(big.Int)
}

// ReleaseState puts a state back into the pool. This is the error-path
// release: it does not preserve any result. Aliases into the arena are
// detached before pool return, the arena is reset, and oversized arenas are
// dropped to GC.
//
// Parameters:
//   - s: The calculation state to return to the pool. Safe to call with nil.
func ReleaseState(s *CalculationState) {
	if s == nil {
		return
	}
	finalizeStateRelease(s)
}

// ReleaseStateWithResult is the success-path release. It deep-copies src out
// of the arena (so the returned big.Int does not alias arena memory), then
// detaches every state slot, resets the arena, applies anti-bloat, and
// returns the state to the pool.
//
// The deep copy is mandatory — see the package note on P1-04. Without it,
// the next AcquireStateForN that reuses the arena would call arena.Reset()
// and the next allocation would silently overwrite the previous result's
// backing memory.
//
// Parameters:
//   - s: The calculation state to release. Safe to call with nil (src is
//     returned as-is in that case).
//   - src: The big.Int result aliasing the arena.
//
// Returns:
//   - *big.Int: An independent big.Int whose backing memory is not owned by
//     any pooled arena. Equal in value to src.
func ReleaseStateWithResult(s *CalculationState, src *big.Int) *big.Int {
	if s == nil {
		return src
	}

	// Deep-copy the result OUT of the arena before we touch anything else.
	// If src happens to be nil (defensive), preserve the historical
	// permissive contract: callers should not depend on this branch.
	var result *big.Int
	if src != nil {
		result = new(big.Int).Set(src)
	}

	finalizeStateRelease(s)
	return result
}

// finalizeStateRelease is the single shared teardown path used by both
// ReleaseState (error path) and ReleaseStateWithResult (success path). It
// guarantees by construction that:
//
//  1. clearStateAliases is ALWAYS invoked before any return — including the
//     overLimit branch where the state itself is dropped from the pool. This
//     prevents an arena buffer that has been detached from the pooled state
//     from continuing to be aliased by stale big.Int headers held elsewhere
//     (e.g., a future statePool.Get caller that inherits the dropped state's
//     replacement, or an arena that is reused on a different state after the
//     dropped state is GC'd).
//  2. The arena reset and over-limit drop are applied uniformly.
//
// The ordering is critical: checkLimit must be evaluated BEFORE
// clearStateAliases (otherwise every BitLen would be 0 and overLimit would
// never trigger), and clearStateAliases must run BEFORE statePool.Put so a
// concurrent acquirer never observes a state whose slots still alias the
// arena. Any future edit must preserve this ordering — see the unit test
// TestReleaseState_OverLimit_AliasesCleared for the regression guard.
//
// Precondition: s != nil. Callers handle the nil case before invocation.
func finalizeStateRelease(s *CalculationState) {
	// Evaluate overLimit FIRST while the original big.Int headers still
	// reflect the calculation's true bit-length.
	overLimit := checkLimit(s.FK) || checkLimit(s.FK1) ||
		checkLimit(s.T1) || checkLimit(s.T2) ||
		checkLimit(s.T3)

	// Detach every alias from the arena. This must happen on EVERY release
	// path (nominal or overLimit) so that no big.Int header survives pointing
	// into arena memory after this function returns.
	clearStateAliases(s)

	if s.arena != nil {
		s.arena.Reset()
		if s.arenaCapWords > maxArenaPoolWords {
			s.arena = nil
			s.arenaCapWords = 0
		}
	}

	if overLimit {
		// Drop the state from the pool, but only AFTER clearStateAliases has
		// run, so the dropped state cannot leak aliased slots through any
		// remaining reference.
		return
	}
	statePool.Put(s)
}
