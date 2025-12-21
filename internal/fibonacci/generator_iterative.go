// Package fibonacci provides implementations for calculating Fibonacci numbers.
// This file provides an iterative implementation of SequenceGenerator.
package fibonacci

import (
	"context"
	"math/big"
	"sync"
)

// IterativeGenerator generates Fibonacci numbers sequentially using the
// classic iterative algorithm. It maintains O(1) state (two big.Int values)
// and produces each subsequent term in O(k) time where k is the number of
// digits.
//
// This generator is ideal for:
//   - Generating consecutive Fibonacci numbers
//   - Streaming scenarios where memory efficiency matters
//   - Cases where you need to iterate through the sequence
//
// For computing a single large F(n), use Calculator instead, which employs
// O(log n) algorithms like Fast Doubling or Matrix Exponentiation.
//
// Thread Safety:
// IterativeGenerator is NOT safe for concurrent use. Each goroutine should
// have its own generator instance, or use external synchronization.
//
// Example:
//
//	gen := fibonacci.NewIterativeGenerator()
//	for i := 0; i < 1000; i++ {
//	    val, _ := gen.Next(context.Background())
//	    // process val
//	}
type IterativeGenerator struct {
	// current holds F(index) after Next() is called
	current *big.Int
	// next holds F(index+1) for efficient advancement
	next *big.Int
	// index is the current position in the sequence
	index uint64
	// started indicates whether Next() has been called at least once
	started bool
	// calculator is used for Skip() optimization
	calculator Calculator
	// mu protects against concurrent access (optional safety)
	mu sync.Mutex
}

// NewIterativeGenerator creates a new IterativeGenerator starting from F(0).
// The generator is ready to use immediately; the first call to Next() will
// return F(0).
//
// Returns:
//   - *IterativeGenerator: A new generator instance.
func NewIterativeGenerator() *IterativeGenerator {
	return &IterativeGenerator{
		current:    big.NewInt(0),
		next:       big.NewInt(1),
		index:      0,
		started:    false,
		calculator: nil, // Lazily initialized on Skip()
	}
}

// NewIterativeGeneratorWithCalculator creates a generator with a custom
// Calculator for optimized Skip() operations.
//
// Parameters:
//   - calc: The Calculator to use for Skip() operations.
//
// Returns:
//   - *IterativeGenerator: A new generator instance.
func NewIterativeGeneratorWithCalculator(calc Calculator) *IterativeGenerator {
	gen := NewIterativeGenerator()
	gen.calculator = calc
	return gen
}

// Next advances the generator and returns the next Fibonacci number.
// The first call returns F(0), the second F(1), and so on.
//
// Parameters:
//   - ctx: The context for managing cancellation.
//
// Returns:
//   - *big.Int: The next Fibonacci number. The returned value is a copy
//     and is safe to modify.
//   - error: An error if the context is cancelled.
func (g *IterativeGenerator) Next(ctx context.Context) (*big.Int, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.started {
		g.started = true
		// First call: return F(0) = 0
		return new(big.Int).Set(g.current), nil
	}

	// Advance the sequence: F(n+1), F(n+2) = F(n+2), F(n+1) + F(n+2)
	// But we store (current, next) = (F(n), F(n+1))
	// After advance: (current, next) = (F(n+1), F(n+2)) = (next, current+next)
	g.index++

	// Swap and add: new_current = old_next, new_next = old_current + old_next
	g.current, g.next = g.next, new(big.Int).Add(g.current, g.next)

	return new(big.Int).Set(g.current), nil
}

// Current returns the current Fibonacci number without advancing.
// If Next has never been called, returns nil.
//
// Returns:
//   - *big.Int: A copy of the current Fibonacci number, or nil if not started.
func (g *IterativeGenerator) Current() *big.Int {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.started {
		return nil
	}
	return new(big.Int).Set(g.current)
}

// Index returns the index of the current Fibonacci number.
// If Next has never been called, returns 0.
//
// Returns:
//   - uint64: The index of the current Fibonacci number.
func (g *IterativeGenerator) Index() uint64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.index
}

// Reset resets the generator to start from F(0).
// After Reset(), the next call to Next() will return F(0).
func (g *IterativeGenerator) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.current = big.NewInt(0)
	g.next = big.NewInt(1)
	g.index = 0
	g.started = false
}

// Skip advances the generator to the n-th Fibonacci number without
// returning intermediate values.
//
// For small skips (n < current index + 1000), this iterates forward.
// For large skips, it uses the Calculator's O(log n) algorithm for efficiency.
//
// Parameters:
//   - ctx: The context for managing cancellation.
//   - n: The index to skip to.
//
// Returns:
//   - *big.Int: The Fibonacci number F(n).
//   - error: An error if generation fails or context is cancelled.
func (g *IterativeGenerator) Skip(ctx context.Context, n uint64) (*big.Int, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// If we haven't started yet or n is 0, handle specially
	if n == 0 {
		g.current = big.NewInt(0)
		g.next = big.NewInt(1)
		g.index = 0
		g.started = true
		return new(big.Int).Set(g.current), nil
	}

	// Determine current effective index
	currentIdx := g.index
	if !g.started {
		currentIdx = 0
	}

	// If n is close to current position, iterate forward
	const iterativeThreshold = 1000
	if n >= currentIdx && n-currentIdx < iterativeThreshold {
		// Unlock for iteration (we'll handle locking per-iteration in the unlocked version)
		g.mu.Unlock()
		defer g.mu.Lock()

		for g.index < n || !g.started {
			if _, err := g.Next(ctx); err != nil {
				return nil, err
			}
		}
		return g.Current(), nil
	}

	// For large jumps, use Calculator
	if g.calculator == nil {
		// Lazily initialize with default fast calculator
		factory := GlobalFactory()
		calc, err := factory.Get("fast")
		if err != nil {
			return nil, err
		}
		g.calculator = calc
	}

	// Calculate F(n) and F(n+1) directly
	result, err := g.calculator.Calculate(ctx, nil, 0, n, Options{})
	if err != nil {
		return nil, err
	}

	// Also calculate F(n+1) for proper state
	nextResult, err := g.calculator.Calculate(ctx, nil, 0, n+1, Options{})
	if err != nil {
		return nil, err
	}

	// Update state
	g.current = new(big.Int).Set(result)
	g.next = new(big.Int).Set(nextResult)
	g.index = n
	g.started = true

	return new(big.Int).Set(g.current), nil
}

// compile-time interface check
var _ SequenceGenerator = (*IterativeGenerator)(nil)
