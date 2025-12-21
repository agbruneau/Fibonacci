//go:build gmp

// This file provides a GMP-based Fibonacci calculator, conditionally compiled
// with the "gmp" build tag. The build tag architecture ensures that:
//   - Projects can build without GMP (the default, using math/big)
//   - GMP support is opt-in, requiring: go build -tags=gmp
//   - The codebase remains portable across systems without libgmp installed
//
// System Requirements for GMP:
//   - Linux: sudo apt-get install libgmp-dev (Debian/Ubuntu)
//   - macOS: brew install gmp
//   - Windows: Requires MinGW or WSL with libgmp
//
// Architectural Decision:
// The direct use of github.com/ncw/gmp in this file is intentional. While an
// abstract BigInt interface could provide more flexibility, the performance
// overhead of interface indirection would negate GMP's speed benefits.
// The build tag approach provides clean separation without runtime cost.

package fibonacci

import (
	"context"
	"math/big"

	"github.com/ncw/gmp"
)

func init() {
	RegisterCalculator("gmp", func() coreCalculator { return &GMPCalculator{} })
}

// GMPCalculator implements the Fibonacci calculation using the GMP library.
// It requires the 'gmp' build tag and the libgmp library installed on the system.
// This implementation uses the Fast Doubling algorithm but leverages GMP's
// highly optimized C assembly routines for arithmetic operations.
//
// Performance Characteristics:
//   - Excels for extremely large N (> 100,000,000) where GMP's assembly-optimized
//     multiplication routines outperform Go's math/big
//   - For smaller N, the CGO call overhead may make math/big faster
//   - Memory is managed by reusing gmp.Int instances to minimize allocations
type GMPCalculator struct{}

// Name returns the name of the algorithm.
func (c *GMPCalculator) Name() string {
	return "GMP (Fast Doubling)"
}

// findHighestBit returns the number of bits needed to represent n.
// For n=0, returns 0. For n>0, returns floor(log2(n)) + 1.
func findHighestBit(n uint64) int {
	for i := 63; i >= 0; i-- {
		if (n>>i)&1 == 1 {
			return i + 1
		}
	}
	return 0
}

// gmpDoublingStep performs the Fast Doubling step on GMP integers.
// Given F(k) in a and F(k+1) in b, computes:
//   - F(2k) = F(k) * (2*F(k+1) - F(k))
//   - F(2k+1) = F(k+1)² + F(k)²
//
// After this call, a contains F(2k) and b contains F(2k+1).
// t1 and t2 are temporary variables to avoid allocations.
func gmpDoublingStep(a, b, t1, t2 *gmp.Int) {
	// t1 = 2b - a
	t1.MulUint32(b, 2)
	t1.Sub(t1, a)
	// t1 = a * (2b - a) = F(2k)
	t1.Mul(a, t1)

	// t2 = a²
	t2.Mul(a, a)
	// Temporarily store b² in a (we have F(2k) safe in t1)
	a.Mul(b, b)
	// t2 = a² + b² = F(2k+1)
	t2.Add(t2, a)

	// Update results
	a.Set(t1)
	b.Set(t2)
}

// gmpAdditionStep performs the addition step when the current bit is 1.
// Transforms (a, b) from (F(k), F(k+1)) to (F(k+1), F(k)+F(k+1)).
// t is a temporary variable to avoid allocations.
func gmpAdditionStep(a, b, t *gmp.Int) {
	t.Add(a, b)
	a.Set(b)
	b.Set(t)
}

// gmpToStdBigInt converts a gmp.Int to a standard library big.Int.
func gmpToStdBigInt(g *gmp.Int) *big.Int {
	return new(big.Int).SetBytes(g.Bytes())
}

// CalculateCore executes the calculation using GMP's optimized arithmetic.
func (c *GMPCalculator) CalculateCore(ctx context.Context, reporter ProgressReporter, n uint64, opts Options) (*big.Int, error) {
	// Handle base cases
	if n == 0 {
		return big.NewInt(0), nil
	}
	if n == 1 {
		return big.NewInt(1), nil
	}

	// Initialize GMP integers: a = F(0), b = F(1)
	a := gmp.NewInt(0)
	b := gmp.NewInt(1)
	t1 := gmp.NewInt(0)
	t2 := gmp.NewInt(0)

	// Calculate bit length for iteration
	numBits := findHighestBit(n)

	// Progress reporting setup
	var lastReported float64
	totalWork := CalcTotalWork(numBits)
	currentWork := 0.0
	powers := PrecomputePowers4(numBits)

	// Iterate from MSB-1 down to 0
	for i := numBits - 1; i >= 0; i-- {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Doubling step: (F(k), F(k+1)) -> (F(2k), F(2k+1))
		gmpDoublingStep(a, b, t1, t2)

		// Addition step: if bit is 1, advance to k = 2k + 1
		if (n>>i)&1 == 1 {
			gmpAdditionStep(a, b, t1)
		}

		// Report progress
		currentWork = ReportStepProgress(reporter, &lastReported, totalWork, currentWork, i, numBits, powers)
	}

	return gmpToStdBigInt(a), nil
}
