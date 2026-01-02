package fibonacci

import (
	"context"
	"math/big"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestCassinisIdentity_PropertyBased verifies Cassini's Identity for the
// Fibonacci sequence using property-based testing.
// Cassini's Identity states that for any integer n > 0:
//
//	F(n-1) * F(n+1) - F(n)² = (-1)ⁿ
//
// This property provides a powerful correctness check for our Fibonacci
// implementations. The test generates a range of random `n` values and asserts
// that the identity holds true for each calculator.
func TestCassinisIdentity_PropertyBased(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	calculators := []coreCalculator{
		&OptimizedFastDoubling{},
		&MatrixExponentiation{},
		&FFTBasedCalculator{},
	}

	for _, calculator := range calculators {
		properties.Property(calculator.Name()+" satisfies Cassini's Identity", prop.ForAll(
			func(n uint64) bool {
				// We need n >= 1 for Cassini's Identity to hold.
				// The generator creates uint64, so we adjust to avoid n=0.
				if n == 0 {
					n = 1
				}
				if n > 25000 { // Keep n in a reasonable range to avoid excessive test duration
					n = 25000
				}

				ctx := context.Background()
				progressReporter := func(progress float64) {}

				// Calculate F(n-1), F(n), and F(n+1)
				fnMinus1, err := calculator.CalculateCore(ctx, progressReporter, n-1, Options{ParallelThreshold: 4096, FFTThreshold: 20000})
				if err != nil {
					t.Logf("Error calculating F(%d-1): %v", n, err)
					return false
				}
				fn, err := calculator.CalculateCore(ctx, progressReporter, n, Options{ParallelThreshold: 4096, FFTThreshold: 20000})
				if err != nil {
					t.Logf("Error calculating F(%d): %v", n, err)
					return false
				}
				fnPlus1, err := calculator.CalculateCore(ctx, progressReporter, n+1, Options{ParallelThreshold: 4096, FFTThreshold: 20000})
				if err != nil {
					t.Logf("Error calculating F(%d+1): %v", n, err)
					return false
				}

				// Left side of the identity: F(n-1) * F(n+1) - F(n)²
				leftSide := new(big.Int)
				fnSquared := new(big.Int).Mul(fn, fn)
				leftSide.Mul(fnMinus1, fnPlus1).Sub(leftSide, fnSquared)

				// Right side of the identity: (-1)ⁿ
				rightSide := big.NewInt(1)
				if n%2 != 0 {
					rightSide.Neg(rightSide)
				}

				// Check if the identity holds
				return leftSide.Cmp(rightSide) == 0
			},
			gen.UInt64Range(1, 25000), // Generate n in a range that is computationally feasible
		))
	}

	properties.TestingRun(t)
}
