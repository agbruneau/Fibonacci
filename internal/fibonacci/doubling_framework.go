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
)

// DoublingFramework encapsulates the common Fast Doubling algorithm logic.
// It uses a DoublingStepExecutor to perform multiplications, allowing
// different strategies (adaptive, FFT-only, etc.) to be plugged in.
type DoublingFramework struct {
	strategy         DoublingStepExecutor
	dynamicThreshold *threshold.DynamicThresholdManager
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
// This is the core computation logic shared by OptimizedFastDoubling and
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
func (f *DoublingFramework) ExecuteDoublingLoop(ctx context.Context, reporter ProgressCallback, n uint64, opts Options, s *CalculationState, useParallel bool) (*big.Int, error) {
	numBits := bits.Len64(n)

	// Calculate total work for progress reporting via common utility
	totalWork := CalcTotalWork(numBits)
	// Pre-compute powers of 4 for O(1) progress calculation
	powers := PrecomputePowers4(numBits)
	workDone := 0.0
	lastReportedProgress := -1.0

	// Normalize options to ensure consistent default threshold handling
	currentOpts := normalizeOptions(opts)
	dtm := f.dynamicThreshold

	for i := numBits - 1; i >= 0; i-- {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("fast doubling calculation canceled at bit %d/%d: %w", i, numBits-1, err)
		}

		// Track iteration timing for dynamic threshold adjustment
		var iterStart time.Time
		if dtm != nil {
			iterStart = time.Now()
		}

		// Doubling Step
		// Cache bit lengths to avoid repeated calls (BitLen() traverses internal representation)
		fkBitLen := s.FK.BitLen()
		fk1BitLen := s.FK1.BitLen()

		// Get current bit length for metrics (use cached value)
		bitLen := fkBitLen

		// Check if we should use FFT based on current thresholds
		// (thresholds are already normalized, so no need to check for 0)
		usedFFT := bitLen > currentOpts.FFTThreshold
		usedParallel := false

		// Execute the three multiplications for the doubling step:
		// T3 = FK × FK1, T2 = FK², T1 = FK1²
		// All three have independent destinations and read-only sources.
		shouldParallel := useParallel && shouldParallelizeMultiplicationCached(currentOpts, fkBitLen, fk1BitLen)
		if shouldParallel {
			usedParallel = true
		}
		if err := f.strategy.ExecuteStep(ctx, s, currentOpts, shouldParallel); err != nil {
			return nil, fmt.Errorf("doubling step failed at bit %d/%d: %w", i, numBits-1, err)
		}

		// Post-multiply: compute F(2k) and F(2k+1) from the three products.
		// F(2k)   = 2·FK·FK1 - FK² = 2·T3 - T2
		// F(2k+1) = FK1² + FK²     = T1 + T2
		s.T3.Lsh(s.T3, 1)
		s.T3.Sub(s.T3, s.T2)
		s.T1.Add(s.T1, s.T2)

		// Swap the pointers for the next iteration.
		// FK becomes F(2k) (from T3), FK1 becomes F(2k+1) (from T1).
		// T2 and T3 become the old FK and FK1, now temporaries.
		// T1 becomes the old T2 (free).
		s.FK, s.FK1, s.T2, s.T3, s.T1 = s.T3, s.T1, s.FK, s.FK1, s.T2

		// Addition Step: If the i-th bit of n is 1, update F(k) and F(k+1)
		// F(k) <- F(k+1)
		// F(k+1) <- F(k) + F(k+1)
		if (n>>uint(i))&1 == 1 {
			// s.T1 temporarily stores the new F(k+1).
			// T1 is free after the rotation (holds old T2).
			s.T1.Add(s.FK, s.FK1)
			// Swap pointers to avoid large allocations:
			// s.FK becomes the old s.FK1
			// s.FK1 becomes the new sum (s.T1)
			// s.T1 becomes the old s.FK, now a temporary
			s.FK, s.FK1, s.T1 = s.FK1, s.T1, s.FK
		}

		// Record metrics and check for threshold adjustments
		if dtm != nil {
			iterDuration := time.Since(iterStart)
			dtm.RecordIteration(bitLen, iterDuration, usedFFT, usedParallel)

			// Check if thresholds should be adjusted
			newFFT, newParallel, adjusted := dtm.ShouldAdjust()
			if adjusted {
				currentOpts.FFTThreshold = newFFT
				currentOpts.ParallelThreshold = newParallel
			}
		}

		// Harmonized reporting via common utility function
		workDone = ReportStepProgress(reporter, &lastReportedProgress, totalWork, workDone, i, numBits, powers)
	}
	// Optimization: Avoid copying the entire result by "stealing" FK from the
	// calculation state. We replace FK with a fresh empty big.Int so the state
	// remains valid for pool return via ReleaseState. This eliminates an O(n)
	// copy where n is the word count of the result (e.g., ~109K words / ~850 KB
	// for F(10M)), trading it for a single 24-byte big.Int header allocation.
	result := s.FK
	s.FK = new(big.Int)
	return result, nil
}
