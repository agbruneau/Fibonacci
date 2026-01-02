// Package fibonacci provides implementations for calculating Fibonacci numbers.
// This file defines the multiplication strategy abstraction to eliminate code
// duplication between different calculator implementations.
package fibonacci

//go:generate mockgen -source=strategy.go -destination=mocks/mock_strategy.go -package=mocks

import (
	"fmt"
	"math/big"
)

// setOrReturn sets z to result if z is non-nil, otherwise returns result directly.
// This is a common pattern for methods that optionally reuse a destination buffer,
// eliminating code duplication in strategy implementations.
func setOrReturn(z, result *big.Int) *big.Int {
	if z != nil {
		z.Set(result)
		return z
	}
	return result
}

// MultiplicationStrategy defines the interface for multiplication and squaring
// operations used in Fibonacci calculations. Different strategies can choose
// between Karatsuba, FFT, or other multiplication algorithms.
type MultiplicationStrategy interface {
	// Multiply computes x * y and stores the result in z (which may be reused).
	// The result is returned, which may be z or a new *big.Int.
	//
	// Parameters:
	//   - z: The destination big.Int (may be nil or reused).
	//   - x: The first operand.
	//   - y: The second operand.
	//   - opts: Configuration options.
	//
	// Returns:
	//   - *big.Int: The product of x and y.
	//   - error: An error if the calculation failed.
	Multiply(z, x, y *big.Int, opts Options) (*big.Int, error)

	// Square computes x * x and stores the result in z (which may be reused).
	// Squaring is optimized compared to general multiplication.
	//
	// Parameters:
	//   - z: The destination big.Int (may be nil or reused).
	//   - x: The operand to square.
	//   - opts: Configuration options.
	//
	// Returns:
	//   - *big.Int: The square of x.
	//   - error: An error if the calculation failed.
	Square(z, x *big.Int, opts Options) (*big.Int, error)

	// Name returns a descriptive name for the strategy.
	Name() string

	// ExecuteStep performs a complete doubling step calculation:
	// F(2k) = F(k) * (2*F(k+1) - F(k))
	// F(2k+1) = F(k+1)^2 + F(k)^2
	//
	// This specialized method allows strategies to optimize the doubling step
	// by reusing temporary results or transformations (e.g., FFT transforms).
	//
	// Parameters:
	//   - s: The calculation state containing operands and temporaries.
	//   - opts: Configuration options.
	//   - inParallel: Whether to execute multiplications in parallel.
	//
	// Returns:
	//   - error: An error if the calculation failed.
	ExecuteStep(s *CalculationState, opts Options, inParallel bool) error
}

// AdaptiveStrategy uses smartMultiply and smartSquare to adaptively choose
// between Karatsuba (via math/big) and FFT-based multiplication based on
// operand sizes and thresholds.
type AdaptiveStrategy struct{}

// Name returns the name of the adaptive strategy.
func (s *AdaptiveStrategy) Name() string {
	return "Adaptive (Karatsuba/FFT)"
}

// Multiply performs adaptive multiplication using smartMultiply.
func (s *AdaptiveStrategy) Multiply(z, x, y *big.Int, opts Options) (*big.Int, error) {
	return smartMultiply(z, x, y, opts.FFTThreshold, opts.KaratsubaThreshold)
}

// Square performs adaptive squaring using smartSquare.
func (s *AdaptiveStrategy) Square(z, x *big.Int, opts Options) (*big.Int, error) {
	return smartSquare(z, x, opts.FFTThreshold, opts.KaratsubaThreshold)
}

// ExecuteStep performs a doubling step, choosing between standard logic
// and optimized FFT transform reuse based on operand size.
func (s *AdaptiveStrategy) ExecuteStep(state *CalculationState, opts Options, inParallel bool) error {
	// If operands are large enough for FFT, use specialized reuse logic
	if opts.FFTThreshold > 0 && state.FK1.BitLen() > opts.FFTThreshold {
		return executeDoublingStepFFT(state, opts, inParallel)
	}
	// Fallback to standard doubling step multiplication
	return executeDoublingStepMultiplications(s, state, opts, inParallel)
}

// FFTOnlyStrategy forces FFT-based multiplication for all operations,
// regardless of operand size. This is useful for benchmarking FFT performance
// or for very large numbers where FFT is always optimal.
type FFTOnlyStrategy struct{}

// Name returns the name of the FFT-only strategy.
func (s *FFTOnlyStrategy) Name() string {
	return "FFT-Only"
}

// Multiply performs FFT-based multiplication using mulFFT.
func (s *FFTOnlyStrategy) Multiply(z, x, y *big.Int, opts Options) (*big.Int, error) {
	res, err := mulFFT(x, y)
	if err != nil {
		return nil, fmt.Errorf("FFT multiplication failed: %w", err)
	}
	return setOrReturn(z, res), nil
}

// Square performs FFT-based squaring using sqrFFT.
func (s *FFTOnlyStrategy) Square(z, x *big.Int, opts Options) (*big.Int, error) {
	res, err := sqrFFT(x)
	if err != nil {
		return nil, fmt.Errorf("FFT squaring failed: %w", err)
	}
	return setOrReturn(z, res), nil
}

// ExecuteStep performs a doubling step using FFT transform reuse.
func (s *FFTOnlyStrategy) ExecuteStep(state *CalculationState, opts Options, inParallel bool) error {
	return executeDoublingStepFFT(state, opts, inParallel)
}

// KaratsubaStrategy forces Karatsuba multiplication (via math/big) for all
// operations, regardless of operand size. This is primarily useful for
// testing and comparison purposes.
type KaratsubaStrategy struct{}

// Name returns the name of the Karatsuba-only strategy.
func (s *KaratsubaStrategy) Name() string {
	return "Karatsuba-Only"
}

// Multiply performs Karatsuba multiplication using math/big.Mul.
func (s *KaratsubaStrategy) Multiply(z, x, y *big.Int, opts Options) (*big.Int, error) {
	if z == nil {
		z = new(big.Int)
	}
	return z.Mul(x, y), nil
}

// Square performs Karatsuba squaring using math/big.Mul.
func (s *KaratsubaStrategy) Square(z, x *big.Int, opts Options) (*big.Int, error) {
	if z == nil {
		z = new(big.Int)
	}
	return z.Mul(x, x), nil
}

// ExecuteStep performs a standard doubling step using Karatsuba multiplication.
func (s *KaratsubaStrategy) ExecuteStep(state *CalculationState, opts Options, inParallel bool) error {
	return executeDoublingStepMultiplications(s, state, opts, inParallel)
}
