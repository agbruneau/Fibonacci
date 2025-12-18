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
		tasks := []func() error{
			func() error {
				var err error
				s.T3, err = strategy.Multiply(s.T3, s.F_k, s.T2, opts)
				if err != nil {
					return fmt.Errorf("parallel multiply F_k * T2 failed: %w", err)
				}
				return nil
			},
			func() error {
				var err error
				s.T1, err = strategy.Square(s.T1, s.F_k1, opts)
				if err != nil {
					return fmt.Errorf("parallel square F_k1 failed: %w", err)
				}
				return nil
			},
			func() error {
				var err error
				s.T4, err = strategy.Square(s.T4, s.F_k, opts)
				if err != nil {
					return fmt.Errorf("parallel square F_k failed: %w", err)
				}
				return nil
			},
		}

		var wg sync.WaitGroup
		var ec parallel.ErrorCollector
		wg.Add(len(tasks))
		for _, t := range tasks {
			go func(f func() error) {
				defer wg.Done()
				ec.SetError(f())
			}(t)
		}
		wg.Wait()
		return ec.Err()
	}

	// Sequential execution
	var err error
	s.T3, err = strategy.Multiply(s.T3, s.F_k, s.T2, opts)
	if err != nil {
		return fmt.Errorf("multiply F_k * T2 failed: %w", err)
	}
	s.T1, err = strategy.Square(s.T1, s.F_k1, opts)
	if err != nil {
		return fmt.Errorf("square F_k1 failed: %w", err)
	}
	s.T4, err = strategy.Square(s.T4, s.F_k, opts)
	if err != nil {
		return fmt.Errorf("square F_k failed: %w", err)
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
//   - s: The calculation state (must be initialized with F_k=0, F_k1=1).
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
			return nil, fmt.Errorf("fast doubling calculation cancelled at bit %d/%d: %w", i, numBits-1, err)
		}

		// Track iteration timing for dynamic threshold adjustment
		var iterStart time.Time
		if dtm != nil {
			iterStart = time.Now()
		}

		// Doubling Step
		// T2 = 2*F_k1 - F_k
		s.T2.Lsh(s.F_k1, 1).Sub(s.T2, s.F_k)

		// Cache bit lengths to avoid repeated calls (BitLen() traverses internal representation)
		fkBitLen := s.F_k.BitLen()
		fk1BitLen := s.F_k1.BitLen()

		// Get current bit length for metrics (use cached value)
		bitLen := fkBitLen

		// Check if we should use FFT based on current thresholds
		// (thresholds are already normalized, so no need to check for 0)
		usedFFT := bitLen > currentOpts.FFTThreshold
		usedParallel := false

		// Execute the three multiplications for the doubling step
		// Parallelize when at least one of the main operands is large
		// Pass cached bit lengths to avoid redundant BitLen() calls
		shouldParallel := useParallel && shouldParallelizeMultiplicationCached(s, currentOpts, fkBitLen, fk1BitLen)
		if shouldParallel {
			usedParallel = true
		}
		if err := executeDoublingStepMultiplications(f.strategy, s, currentOpts, shouldParallel); err != nil {
			return nil, fmt.Errorf("doubling step multiplication failed at bit %d/%d: %w", i, numBits-1, err)
		}

		// F(2k+1) = F(k+1)² + F(k)². Store result in T2, which is free.
		s.T2.Add(s.T1, s.T4)
		// Swap the pointers for the next iteration.
		// F_k becomes F(2k) (from T3), F_k1 becomes F(2k+1) (from T2).
		// T2 and T3 become the old F_k and F_k1, now temporaries.
		s.F_k, s.F_k1, s.T2, s.T3 = s.T3, s.T2, s.F_k, s.F_k1

		// Addition Step: If the i-th bit of n is 1, update F(k) and F(k+1)
		// F(k) <- F(k+1)
		// F(k+1) <- F(k) + F(k+1)
		if (n>>uint(i))&1 == 1 {
			// s.T1 temporarily stores the new F(k+1)
			s.T1.Add(s.F_k, s.F_k1)
			// Swap pointers to avoid large allocations:
			// s.F_k becomes the old s.F_k1
			// s.F_k1 becomes the new sum (s.T1)
			// s.T1 becomes the old s.F_k, now a temporary
			s.F_k, s.F_k1, s.T1 = s.F_k1, s.T1, s.F_k
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
	return new(big.Int).Set(s.F_k), nil
}
