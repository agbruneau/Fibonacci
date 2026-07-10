package orchestration

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/agbruneau/FibGo/internal/fibonacci"
)

// LastDigitsResult holds the outcome of a partial Fibonacci computation
// limited to the last K decimal digits.
type LastDigitsResult struct {
	// Value is F(n) mod 10^k. It is in [0, 10^k).
	Value *big.Int
	// Digits is Value formatted as a fixed-width string of length k,
	// zero-padded on the left.
	Digits string
	// Duration is the wall-clock time spent computing Value.
	Duration time.Duration
}

// ComputeLastDigits computes the last k decimal digits of F(n) using modular
// arithmetic. It uses O(k) memory regardless of n.
//
// The function is pure with respect to I/O: it performs no printing, no
// signal handling and no logging. The provided context is checked once before
// the computation begins so the caller can short-circuit on cancellation; the
// underlying modular calculator (FastDoublingMod) takes no context and runs to
// completion (it is O(k) memory and fast for realistic k).
//
// Returns:
//   - LastDigitsResult: The result, including a pre-formatted zero-padded string.
//   - error: An error if k <= 0, or if the modular computation fails (including
//     ctx cancellation).
func ComputeLastDigits(ctx context.Context, n uint64, k int) (LastDigitsResult, error) {
	if k <= 0 {
		return LastDigitsResult{}, fmt.Errorf("last-digits count must be > 0, got %d", k)
	}

	if err := ctx.Err(); err != nil {
		return LastDigitsResult{}, err
	}

	// modulus = 10^k. k is bounded by user input; treat as an int64 exponent.
	mod := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(k)), nil)

	start := time.Now()
	value, err := fibonacci.FastDoublingMod(n, mod)
	elapsed := time.Since(start)

	if err != nil {
		return LastDigitsResult{}, err
	}

	// Zero-pad on the left to exactly k digits. fmt's "*" width verb reads
	// the width from the argument list, so the format string never needs
	// to be built dynamically.
	digits := fmt.Sprintf("%0*s", k, value.String())

	return LastDigitsResult{
		Value:    value,
		Digits:   digits,
		Duration: elapsed,
	}, nil
}
