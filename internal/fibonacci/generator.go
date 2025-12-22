// Package fibonacci provides implementations for calculating Fibonacci numbers.
// This file defines the SequenceGenerator interface for iterative/streaming
// generation of Fibonacci numbers.
package fibonacci

//go:generate mockgen -source=generator.go -destination=mocks/mock_generator.go -package=mocks

import (
	"context"
	"math/big"
)

// SequenceGenerator defines the interface for generating a sequence of Fibonacci values.
// Unlike Calculator which computes a single F(n), SequenceGenerator produces
// consecutive terms, enabling streaming use cases.
//
// This interface enables:
//   - Generating the first N Fibonacci numbers sequentially
//   - Streaming processing of Fibonacci sequences
//   - Memory-efficient iteration without computing intermediate values from scratch
//
// Example usage:
//
//	gen := fibonacci.NewIterativeGenerator()
//	for i := 0; i < 100; i++ {
//	    val, err := gen.Next(ctx)
//	    if err != nil {
//	        return err
//	    }
//	    // Use val
//	}
type SequenceGenerator interface {
	// Next advances the generator and returns the next Fibonacci number.
	// The first call returns F(0), the second F(1), etc.
	// Returns an error if the context is cancelled.
	//
	// Parameters:
	//   - ctx: The context for managing cancellation.
	//
	// Returns:
	//   - *big.Int: The next Fibonacci number in the sequence.
	//   - error: An error if generation fails or context is cancelled.
	Next(ctx context.Context) (*big.Int, error)

	// Current returns the current Fibonacci number without advancing.
	// If Next has never been called, returns nil.
	//
	// Returns:
	//   - *big.Int: The current Fibonacci number, or nil if not started.
	Current() *big.Int

	// Index returns the index of the current Fibonacci number.
	// If Next has never been called, returns 0.
	//
	// Returns:
	//   - uint64: The index of the current Fibonacci number.
	Index() uint64

	// Reset resets the generator to start from F(0).
	// After Reset(), the next call to Next() will return F(0).
	Reset()

	// Skip advances the generator to the n-th Fibonacci number without
	// returning intermediate values. This is more efficient than calling
	// Next() n times for large skips.
	//
	// Parameters:
	//   - ctx: The context for managing cancellation.
	//   - n: The index to skip to.
	//
	// Returns:
	//   - *big.Int: The Fibonacci number F(n).
	//   - error: An error if generation fails or context is cancelled.
	Skip(ctx context.Context, n uint64) (*big.Int, error)
}
