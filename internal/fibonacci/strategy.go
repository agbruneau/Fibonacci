// This file defines the multiplication strategy abstraction to eliminate code
// duplication between different calculator implementations.

package fibonacci

import (
	"context"
	"fmt"
	"math/big"

	"github.com/agbruneau/FibGo/internal/bigfft"
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

// Multiplier defines pure multiplication and squaring operations used in
// Fibonacci calculations. Different implementations can choose between
// standard math/big, FFT, or other multiplication algorithms.
//
// This is the narrow interface: consumers that only need Multiply/Square
// should depend on Multiplier rather than the wider DoublingStepExecutor.
type Multiplier interface {
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
}

// DoublingStepExecutor extends Multiplier with a doubling-step-aware execution
// method. Consumers that need the full doubling step (which combines multiple
// multiplications with algorithm-specific optimizations like FFT transform
// reuse) should depend on this interface.
type DoublingStepExecutor interface {
	Multiplier

	// ExecuteStep performs a complete doubling step calculation:
	// F(2k) = F(k) * (2*F(k+1) - F(k))
	// F(2k+1) = F(k+1)^2 + F(k)^2
	//
	// This specialized method allows strategies to optimize the doubling step
	// by reusing temporary results or transformations (e.g., FFT transforms).
	//
	// Parameters:
	//   - ctx: The context for cancellation checking between multiplications.
	//   - s: The calculation state containing operands and temporaries.
	//   - opts: Configuration options.
	//   - inParallel: Whether to execute multiplications in parallel.
	//
	// Returns:
	//   - error: An error if the calculation failed.
	ExecuteStep(ctx context.Context, s *CalculationState, opts Options, inParallel bool) error
}

// AdaptiveStrategy uses smartMultiply and smartSquare to adaptively choose
// between math/big and FFT-based multiplication based on operand sizes
// and thresholds.
type AdaptiveStrategy struct{}

// Name returns the name of the adaptive strategy.
func (s *AdaptiveStrategy) Name() string {
	return "Adaptive (math/big + FFT)"
}

// Multiply performs adaptive multiplication using smartMultiply.
func (s *AdaptiveStrategy) Multiply(z, x, y *big.Int, opts Options) (*big.Int, error) {
	return smartMultiply(z, x, y, opts.FFTThreshold)
}

// Square performs adaptive squaring using smartSquare.
func (s *AdaptiveStrategy) Square(z, x *big.Int, opts Options) (*big.Int, error) {
	return smartSquare(z, x, opts.FFTThreshold)
}

// ExecuteStep performs a doubling step, choosing between standard logic
// and optimized FFT transform reuse based on operand size.
func (s *AdaptiveStrategy) ExecuteStep(ctx context.Context, state *CalculationState, opts Options, inParallel bool) error {
	// If operands are large enough for FFT, use specialized reuse logic
	if opts.FFTThreshold > 0 && state.FK1.BitLen() > opts.FFTThreshold {
		return executeDoublingStepFFT(ctx, state, opts, inParallel)
	}
	// Fallback to standard doubling step multiplication
	return executeDoublingStepMultiplications(ctx, s, state, opts, inParallel)
}

// FFTOnlyStrategy forces FFT-based multiplication for all operations,
// regardless of operand size. This is useful for benchmarking FFT performance
// or for very large numbers where FFT is always optimal.
type FFTOnlyStrategy struct{}

// Name returns the name of the FFT-only strategy.
func (s *FFTOnlyStrategy) Name() string {
	return "FFT-Only"
}

// Multiply performs FFT-based multiplication. When z != nil it writes directly
// into z via bigfft.MulTo (reusing z's buffer), avoiding the fresh allocation +
// O(n) copy that mulFFT()+setOrReturn would incur. bigfft.MulTo applies the same
// getFFTThreshold gate (and math/big fallback + sign handling) as mulFFT, so the
// result is identical. A3-03.
func (s *FFTOnlyStrategy) Multiply(z, x, y *big.Int, opts Options) (*big.Int, error) {
	if z == nil {
		res, err := mulFFT(x, y)
		if err != nil {
			return nil, fmt.Errorf("FFT multiplication failed: %w", err)
		}
		return res, nil
	}
	res, err := bigfft.MulTo(z, x, y)
	if err != nil {
		return nil, fmt.Errorf("FFT multiplication failed: %w", err)
	}
	return res, nil
}

// Square performs FFT-based squaring. When z != nil it writes directly into z
// via bigfft.SqrTo, avoiding the fresh allocation + O(n) copy. A3-03.
func (s *FFTOnlyStrategy) Square(z, x *big.Int, opts Options) (*big.Int, error) {
	if z == nil {
		res, err := sqrFFT(x)
		if err != nil {
			return nil, fmt.Errorf("FFT squaring failed: %w", err)
		}
		return res, nil
	}
	res, err := bigfft.SqrTo(z, x)
	if err != nil {
		return nil, fmt.Errorf("FFT squaring failed: %w", err)
	}
	return res, nil
}

// ExecuteStep performs a doubling step using FFT transform reuse.
func (s *FFTOnlyStrategy) ExecuteStep(ctx context.Context, state *CalculationState, opts Options, inParallel bool) error {
	return executeDoublingStepFFT(ctx, state, opts, inParallel)
}
