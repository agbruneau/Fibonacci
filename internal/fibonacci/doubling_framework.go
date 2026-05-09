// This file contains the common Fast Doubling framework used by multiple
// calculator implementations to eliminate code duplication.

package fibonacci

import (
	"context"
	"fmt"
	"math/big"
	"math/bits"
	"time"

	"github.com/agbru/fibcalc/internal/fibonacci/threshold"
	"github.com/agbru/fibcalc/internal/progress"
)

// DoublingFramework encapsulates the common Fast Doubling algorithm logic.
// It uses a DoublingStepExecutor to perform multiplications, allowing
// different strategies (adaptive, FFT-only, etc.) to be plugged in.
//
// CacheStrategy is an optional pluggable hook called from inside the doubling
// loop to let an adapter tune an underlying transform cache (e.g. the bigfft
// global cache). When nil, no cache adjustment is performed. The DTM-driven
// constructors install the default bigfft-backed strategy automatically so
// the historical behaviour is preserved.
type DoublingFramework struct {
	strategy         DoublingStepExecutor
	dynamicThreshold *threshold.DynamicThresholdManager
	CacheStrategy    CacheStrategy
}

// NewDoublingFramework creates a new Fast Doubling framework with the given strategy.
//
// Parameters:
//   - strategy: The DoublingStepExecutor strategy to use.
//
// Returns:
//   - *DoublingFramework: A new framework instance.
func NewDoublingFramework(strategy DoublingStepExecutor) *DoublingFramework {
	return &DoublingFramework{strategy: strategy}
}

// NewDoublingFrameworkWithDynamicThresholds creates a framework with dynamic threshold adjustment.
//
// Parameters:
//   - strategy: The DoublingStepExecutor strategy to use.
//   - dtm: The dynamic threshold manager (can be nil to disable).
//
// Returns:
//   - *DoublingFramework: A new framework instance.
func NewDoublingFrameworkWithDynamicThresholds(strategy DoublingStepExecutor, dtm *threshold.DynamicThresholdManager) *DoublingFramework {
	return &DoublingFramework{
		strategy:         strategy,
		dynamicThreshold: dtm,
		CacheStrategy:    NewBigFFTCacheStrategy(),
	}
}

// executeDoublingStepMultiplications performs the three multiplications required
// for a doubling step, either sequentially or in parallel based on the inParallel flag.
// This function encapsulates the parallelization logic to keep ExecuteDoublingLoop clean.
//
// It depends on the narrow Multiplier interface since it only calls Multiply and Square.
//
// Parameters:
//   - ctx: The context for cancellation checking between sequential multiplications.
//   - strategy: The Multiplier to use for Multiply/Square operations.
//   - s: The calculation state containing operands and temporaries.
//   - opts: Configuration options for the calculation.
//   - inParallel: Whether to execute multiplications in parallel.
//
// Returns:
//   - error: An error if any multiplication failed, with context about which operation failed.
func executeDoublingStepMultiplications(ctx context.Context, strategy Multiplier, s *CalculationState, opts Options, inParallel bool) error {
	if inParallel {
		// Each goroutine writes to a disjoint destination (T3, T1, T2)
		// and reads shared sources (FK, FK1) which are read-only here.
		return executeParallel3(ctx,
			func() error {
				var err error
				s.T3, err = strategy.Multiply(s.T3, s.FK, s.FK1, opts)
				if err != nil {
					return fmt.Errorf("parallel multiply FK * FK1 failed: %w", err)
				}
				return nil
			},
			func() error {
				var err error
				s.T1, err = strategy.Square(s.T1, s.FK1, opts)
				if err != nil {
					return fmt.Errorf("parallel square FK1 failed: %w", err)
				}
				return nil
			},
			func() error {
				var err error
				s.T2, err = strategy.Square(s.T2, s.FK, opts)
				if err != nil {
					return fmt.Errorf("parallel square FK failed: %w", err)
				}
				return nil
			},
		)
	}

	// Sequential execution with context checks between multiplications
	var err error
	s.T3, err = strategy.Multiply(s.T3, s.FK, s.FK1, opts)
	if err != nil {
		return fmt.Errorf("multiply FK * FK1 failed: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("canceled after multiply: %w", err)
	}
	s.T1, err = strategy.Square(s.T1, s.FK1, opts)
	if err != nil {
		return fmt.Errorf("square FK1 failed: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("canceled after square FK1: %w", err)
	}
	s.T2, err = strategy.Square(s.T2, s.FK, opts)
	if err != nil {
		return fmt.Errorf("square FK failed: %w", err)
	}
	return nil
}

// ExecuteDoublingLoop executes the Fast Doubling algorithm loop.
// This is the core computation logic shared by FastDoublingCalculator and
// FFTBasedCalculator.
//
// The algorithm iterates over the bits of n from most significant to least
// significant, performing doubling steps and addition steps as needed.
// When useParallel is true and operands are large enough, multiplications
// are executed in parallel to leverage multi-core processors.
//
// Parameters:
//   - ctx: The context for managing cancellation and deadlines.
//   - reporter: The function used for reporting progress.
//   - n: The index of the Fibonacci number to calculate.
//   - opts: Configuration options for the calculation.
//   - s: The calculation state (must be initialized with FK=0, FK1=1).
//   - useParallel: Whether to use parallelization when beneficial.
//
// Returns:
//   - *big.Int: The calculated Fibonacci number F(n).
//   - error: An error if one occurred (e.g., context cancellation).
func (f *DoublingFramework) ExecuteDoublingLoop(ctx context.Context, reporter progress.ProgressCallback, n uint64, opts Options, s *CalculationState, useParallel bool) (*big.Int, error) {
	numBits := bits.Len64(n)

	// Calculate total work for progress reporting via common utility
	totalWork := progress.CalcTotalWork(numBits)
	// Pre-compute powers of 4 for O(1) progress calculation
	powers := progress.PrecomputePowers4(numBits)
	workDone := 0.0
	lastReportedProgress := -1.0

	// Normalize options to ensure consistent default threshold handling
	currentOpts := normalizeOptions(opts)
	dtm := f.dynamicThreshold
	// P1-02 / R3.2: cache adjustment is delegated to a CacheStrategy. The
	// strategy implements its own throttling so the loop stays algorithm-only.
	cacheStrategy := f.CacheStrategy
	iterCount := 0

	for i := numBits - 1; i >= 0; i-- {
		iterCount++
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("fast doubling calculation canceled at bit %d/%d: %w", i, numBits-1, err)
		}

		// Track iteration timing for dynamic threshold adjustment
		var iterStart time.Time
		if dtm != nil {
			iterStart = time.Now()
		}

		// Doubling Step — cache bit lengths once (BitLen() walks the rep).
		fkBitLen := s.FK.BitLen()
		fk1BitLen := s.FK1.BitLen()
		bitLen := fkBitLen
		usedFFT := bitLen > currentOpts.FFTThreshold

		// Execute the three multiplications: T3 = FK·FK1, T2 = FK², T1 = FK1².
		shouldParallel := useParallel && shouldParallelizeMultiplicationCached(currentOpts, fkBitLen, fk1BitLen)
		if err := f.strategy.ExecuteStep(ctx, s, currentOpts, shouldParallel); err != nil {
			return nil, fmt.Errorf("doubling step failed at bit %d/%d: %w", i, numBits-1, err)
		}

		// Post-multiply: F(2k) = 2·T3 - T2, F(2k+1) = T1 + T2.
		s.T3.Lsh(s.T3, 1)
		s.T3.Sub(s.T3, s.T2)
		s.T1.Add(s.T1, s.T2)

		// Rotate pointers so FK/FK1 hold F(2k)/F(2k+1) and the rest become temps.
		s.FK, s.FK1, s.T2, s.T3, s.T1 = s.T3, s.T1, s.FK, s.FK1, s.T2

		// Addition Step: if bit i of n is set, advance one extra index.
		// i is a bit index in [0, bits.Len64(n)-1] ⊂ [0, 63], uint conversion safe. #nosec G115
		if (n>>uint(i))&1 == 1 {
			s.T1.Add(s.FK, s.FK1)
			s.FK, s.FK1, s.T1 = s.FK1, s.T1, s.FK
		}

		// Record metrics and check for threshold adjustments
		if dtm != nil {
			iterDuration := time.Since(iterStart)
			dtm.RecordIteration(bitLen, iterDuration, usedFFT, shouldParallel)

			if newFFT, newParallel, adjusted := dtm.ShouldAdjust(); adjusted {
				currentOpts.FFTThreshold = newFFT
				currentOpts.ParallelThreshold = newParallel
			}

			// Cache tuning is delegated to the pluggable strategy. Preserves
			// the historical "only when DTM is enabled" gate so loops without
			// dynamic thresholds remain side-effect-free on the global cache.
			if cacheStrategy != nil {
				if err := cacheStrategy.Sample(iterCount, numBits); err != nil {
					return nil, fmt.Errorf("cache strategy sample failed at bit %d/%d: %w", i, numBits-1, err)
				}
			}
		}

		// Harmonized reporting via common utility function
		workDone = progress.ReportStepProgress(reporter, &lastReportedProgress, totalWork, workDone, i, numBits, powers)
	}
	// P1-04: do NOT "steal" s.FK here. The state's big.Ints alias the
	// state-bound arena's backing buffer. If we returned s.FK directly and
	// nulled the slot, the result would still alias the arena that the
	// pooled state retains — the next AcquireStateForN that reuses the
	// arena would Reset() its offset and the next allocation would silently
	// overwrite this result. ReleaseStateWithResult at the call site
	// performs the necessary deep-copy out of the arena (a single ~850 KB
	// memcpy for F(10M), <0.01% of the total runtime, far cheaper than the
	// race conditions documented in P1-04-SKIPPED.md).
	return s.FK, nil
}
