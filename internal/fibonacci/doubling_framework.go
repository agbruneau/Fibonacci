// Package fibonacci provides implementations for calculating Fibonacci numbers.
// This file contains the common Fast Doubling framework used by multiple
// calculator implementations to eliminate code duplication.
package fibonacci

import (
	"context"
	"math/big"
	"math/bits"
	"sync"
)

// DoublingFramework encapsulates the common Fast Doubling algorithm logic.
// It uses a MultiplicationStrategy to perform multiplications, allowing
// different strategies (adaptive, FFT-only, etc.) to be plugged in.
type DoublingFramework struct {
	strategy MultiplicationStrategy
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

// ExecuteDoublingLoop executes the Fast Doubling algorithm loop.
// This is the core computation logic shared by OptimizedFastDoubling and
// FFTBasedCalculator.
//
// The algorithm iterates over the bits of n from most significant to least
// significant, performing doubling steps and addition steps as needed.
//
// Parameters:
//   - ctx: The context for managing cancellation and deadlines.
//   - reporter: The function used for reporting progress.
//   - n: The index of the Fibonacci number to calculate.
//   - opts: Configuration options for the calculation.
//   - s: The calculation state (must be initialized with F_k=0, F_k1=1).
//
// Returns:
//   - *big.Int: The calculated Fibonacci number F(n).
//   - error: An error if one occurred (e.g., context cancellation).
func (f *DoublingFramework) ExecuteDoublingLoop(ctx context.Context, reporter ProgressReporter, n uint64, opts Options, s *CalculationState) (*big.Int, error) {
	numBits := bits.Len64(n)

	// Calculate total work for progress reporting via common utility
	totalWork := CalcTotalWork(numBits)
	workDone := 0.0
	lastReportedProgress := -1.0

	for i := numBits - 1; i >= 0; i-- {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// Doubling Step
		// T2 = 2*F_k1 - F_k
		s.T2.Lsh(s.F_k1, 1).Sub(s.T2, s.F_k)

		// Use strategy for multiplications
		// T3 = F_k * T2
		s.T3 = f.strategy.Multiply(s.T3, s.F_k, s.T2, opts)
		// T1 = F_k1^2 (using optimized squaring)
		s.T1 = f.strategy.Square(s.T1, s.F_k1, opts)
		// T4 = F_k^2 (using optimized squaring)
		s.T4 = f.strategy.Square(s.T4, s.F_k, opts)

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

		// Harmonized reporting via common utility function
		workDone = ReportStepProgress(reporter, &lastReportedProgress, totalWork, workDone, i, numBits)
	}
	return new(big.Int).Set(s.F_k), nil
}

// ExecuteDoublingLoopWithParallel executes the Fast Doubling algorithm loop
// with support for parallelization of multiplications when beneficial.
// This is used by OptimizedFastDoubling to leverage multi-core processors.
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
func (f *DoublingFramework) ExecuteDoublingLoopWithParallel(ctx context.Context, reporter ProgressReporter, n uint64, opts Options, s *CalculationState, useParallel bool) (*big.Int, error) {
	numBits := bits.Len64(n)

	// Calculate total work for progress reporting via common utility
	totalWork := CalcTotalWork(numBits)
	workDone := 0.0
	lastReportedProgress := -1.0

	for i := numBits - 1; i >= 0; i-- {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// Doubling Step
		s.T2.Lsh(s.F_k1, 1).Sub(s.T2, s.F_k)

		// Parallelize when at least one of the main operands is large
		if useParallel && ShouldParallelizeMultiplication(s, opts) {
			// Use parallel multiplication
			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer wg.Done()
				s.T3 = f.strategy.Multiply(s.T3, s.F_k, s.T2, opts)
			}()
			go func() {
				defer wg.Done()
				s.T1 = f.strategy.Square(s.T1, s.F_k1, opts)
			}()
			s.T4 = f.strategy.Square(s.T4, s.F_k, opts)
			wg.Wait()
		} else {
			// Sequential multiplication using strategy
			s.T3 = f.strategy.Multiply(s.T3, s.F_k, s.T2, opts)
			s.T1 = f.strategy.Square(s.T1, s.F_k1, opts)
			s.T4 = f.strategy.Square(s.T4, s.F_k, opts)
		}

		// F(2k+1) = F(k+1)² + F(k)². Store result in T2, which is free.
		s.T2.Add(s.T1, s.T4)
		// Swap the pointers for the next iteration.
		s.F_k, s.F_k1, s.T2, s.T3 = s.T3, s.T2, s.F_k, s.F_k1

		// Addition Step: If the i-th bit of n is 1, update F(k) and F(k+1)
		if (n>>uint(i))&1 == 1 {
			s.T1.Add(s.F_k, s.F_k1)
			s.F_k, s.F_k1, s.T1 = s.F_k1, s.T1, s.F_k
		}

		// Harmonized reporting via common utility function
		workDone = ReportStepProgress(reporter, &lastReportedProgress, totalWork, workDone, i, numBits)
	}
	return new(big.Int).Set(s.F_k), nil
}

