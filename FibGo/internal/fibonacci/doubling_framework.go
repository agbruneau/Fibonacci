// Package fibonacci provides implementations for calculating Fibonacci numbers.
// This file contains the common Fast Doubling framework used by multiple
// calculator implementations to eliminate code duplication.
package fibonacci

import (
	"context"
	"fmt"
	"math/big"
	"math/bits"
	"sync"
	"time"

	"github.com/agbru/fibcalc/internal/parallel"
)

// DoublingFramework encapsulates the common Fast Doubling algorithm logic.
// It uses a MultiplicationStrategy to perform multiplications, allowing
// different strategies (adaptive, FFT-only, etc.) to be plugged in.
type DoublingFramework struct {
	strategy         MultiplicationStrategy
	dynamicThreshold *DynamicThresholdManager
}

// NewDoublingFramework creates a new Fast Doubling framework with the given strategy.
//
// Parameters:
//   - strategy: The multiplication strategy to use.
//
// Returns:
//   - *DoublingFramework: A new framework instance.
func NewDoublingFramework(strategy MultiplicationStrategy) *DoublingFramework {
	return &DoublingFramework{strategy: strategy}
}

// NewDoublingFrameworkWithDynamicThresholds creates a framework with dynamic threshold adjustment.
//
// Parameters:
//   - strategy: The multiplication strategy to use.
//   - dtm: The dynamic threshold manager (can be nil to disable).
//
// Returns:
//   - *DoublingFramework: A new framework instance.
func NewDoublingFrameworkWithDynamicThresholds(strategy MultiplicationStrategy, dtm *DynamicThresholdManager) *DoublingFramework {
	return &DoublingFramework{
		strategy:         strategy,
		dynamicThreshold: dtm,
	}
}

// executeDoublingStepMultiplications performs the three multiplications required
// for a doubling step, either sequentially or in parallel based on the inParallel flag.
// This function encapsulates the parallelization logic to keep ExecuteDoublingLoop clean.
//
// Parameters:
//   - strategy: The multiplication strategy to use.
//   - s: The calculation state containing operands and temporaries.
//   - opts: Configuration options for the calculation.
//   - inParallel: Whether to execute multiplications in parallel.
//
// Returns:
//   - error: An error if any multiplication failed, with context about which operation failed.
func executeDoublingStepMultiplications(strategy MultiplicationStrategy, s *CalculationState, opts Options, inParallel bool) error {
	if inParallel {
		var wg sync.WaitGroup
		var ec parallel.ErrorCollector
		wg.Add(3)

		// 1. T3 = FK * T4
		go func() {
			defer wg.Done()
			var err error
			// Note: We access s.T3, s.FK, s.T4 safely because each goroutine
			// operates on disjoint destination sets or reads shared sources
			// (FK, T4, FK1 are read-only here).
			// T3 is destination for this goroutine.
			s.T3, err = strategy.Multiply(s.T3, s.FK, s.T4, opts)
			if err != nil {
				ec.SetError(fmt.Errorf("parallel multiply FK * T4 failed: %w", err))
			}
		}()

		// 2. T1 = FK1^2
		go func() {
			defer wg.Done()
			var err error
			// T1 is destination for this goroutine.
			s.T1, err = strategy.Square(s.T1, s.FK1, opts)
			if err != nil {
				ec.SetError(fmt.Errorf("parallel square FK1 failed: %w", err))
			}
		}()

		// 3. T2 = FK^2
		go func() {
			defer wg.Done()
			var err error
			// T2 is destination for this goroutine.
			s.T2, err = strategy.Square(s.T2, s.FK, opts)
			if err != nil {
				ec.SetError(fmt.Errorf("parallel square FK failed: %w", err))
			}
		}()

		wg.Wait()
		return ec.Err()
	}

	// Sequential execution
	var err error
	s.T3, err = strategy.Multiply(s.T3, s.FK, s.T4, opts)
	if err != nil {
		return fmt.Errorf("multiply FK * T4 failed: %w", err)
	}
	s.T1, err = strategy.Square(s.T1, s.FK1, opts)
	if err != nil {
		return fmt.Errorf("square FK1 failed: %w", err)
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
func (f *DoublingFramework) ExecuteDoublingLoop(ctx context.Context, reporter ProgressReporter, n uint64, opts Options, s *CalculationState, useParallel bool) (*big.Int, error) {
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
		// T4 = 2*FK1 - FK

		// Optimization: Check if T1 has larger capacity than T4.
		// If so, swap them to reuse the larger buffer for the T4 calculation (2*FK1 - FK),
		// which typically requires a buffer of size k (matching T1's typical capacity from previous step's T2),
		// whereas T4 often holds a smaller capacity (k/2).
		// Note: We compare capacities of the underlying word slices.
		if cap(s.T1.Bits()) > cap(s.T4.Bits()) {
			s.T1, s.T4 = s.T4, s.T1
		}

		// Optimization: Use T4 because it holds F(k)^2 (large) or F(k) (medium) from previous step,
		// avoiding reallocation. T2 (old FK) is typically smaller.
		s.T4.Lsh(s.FK1, 1).Sub(s.T4, s.FK)

		// Cache bit lengths to avoid repeated calls (BitLen() traverses internal representation)
		fkBitLen := s.FK.BitLen()
		fk1BitLen := s.FK1.BitLen()

		// Get current bit length for metrics (use cached value)
		bitLen := fkBitLen

		// Check if we should use FFT based on current thresholds
		// (thresholds are already normalized, so no need to check for 0)
		usedFFT := bitLen > currentOpts.FFTThreshold
		usedParallel := false

		// Execute the three multiplications for the doubling step
		// Parallelize when at least one of the main operands is large
		// Pass cached bit lengths to avoid redundant BitLen() calls
		shouldParallel := useParallel && shouldParallelizeMultiplicationCached(currentOpts, fkBitLen, fk1BitLen)
		if shouldParallel {
			usedParallel = true
		}
		if err := f.strategy.ExecuteStep(s, currentOpts, shouldParallel); err != nil {
			return nil, fmt.Errorf("doubling step failed at bit %d/%d: %w", i, numBits-1, err)
		}

		// F(2k+1) = F(k+1)² + F(k)².
		// Optimization: Use T1 as destination because it already holds F(k+1)²
		// which has the same bit length order as the result, avoiding reallocation.
		// T4 (holding 2*FK1 - FK) is significantly smaller.
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
			// s.T4 temporarily stores the new F(k+1).
			// Optimization: Use T4 instead of T1 because T4 holds F(k)² (large)
			// whereas T1 holds "old T2" (small). This reduces reallocation probability.
			s.T4.Add(s.FK, s.FK1)
			// Swap pointers to avoid large allocations:
			// s.FK becomes the old s.FK1
			// s.FK1 becomes the new sum (s.T4)
			// s.T4 becomes the old s.FK, now a temporary
			s.FK, s.FK1, s.T4 = s.FK1, s.T4, s.FK
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
	return new(big.Int).Set(s.FK), nil
}
