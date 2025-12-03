// Package fibonacci provides implementations for calculating Fibonacci numbers.
// This file defines the multiplication strategy abstraction to eliminate code
// duplication between different calculator implementations.
package fibonacci

import (
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
	Multiply(z, x, y *big.Int, opts Options) *big.Int

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
	Square(z, x *big.Int, opts Options) *big.Int

	// Name returns a descriptive name for the strategy.
	Name() string
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
func (s *AdaptiveStrategy) Multiply(z, x, y *big.Int, opts Options) *big.Int {
	return smartMultiply(z, x, y, opts.FFTThreshold)
}

// Square performs adaptive squaring using smartSquare.
func (s *AdaptiveStrategy) Square(z, x *big.Int, opts Options) *big.Int {
	return smartSquare(z, x, opts.FFTThreshold)
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
func (s *FFTOnlyStrategy) Multiply(z, x, y *big.Int, opts Options) *big.Int {
	return setOrReturn(z, mulFFT(x, y))
}

// Square performs FFT-based squaring using sqrFFT.
func (s *FFTOnlyStrategy) Square(z, x *big.Int, opts Options) *big.Int {
	return setOrReturn(z, sqrFFT(x))
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
func (s *KaratsubaStrategy) Multiply(z, x, y *big.Int, opts Options) *big.Int {
	if z == nil {
		z = new(big.Int)
	}
	return z.Mul(x, y)
}

// Square performs Karatsuba squaring using math/big.Mul.
func (s *KaratsubaStrategy) Square(z, x *big.Int, opts Options) *big.Int {
	if z == nil {
		z = new(big.Int)
	}
	return z.Mul(x, x)
}
