package fibonacci

import (
	"context"
	"math/big"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// defaultTestOpts returns Options used across all property tests.
func defaultTestOpts() Options {
	return Options{ParallelThreshold: 4096, FFTThreshold: 20000}
}

// calcF is a shorthand that computes F(n) with the given calculator.
func calcF(calc CoreCalculator, n uint64) (*big.Int, error) {
	return calc.CalculateCore(context.Background(), func(float64) {}, n, defaultTestOpts())
}

// allCalculators returns the three core calculator implementations.
func allCalculators() []CoreCalculator {
	return []CoreCalculator{
		&OptimizedFastDoubling{},
		&MatrixExponentiation{},
		&FFTBasedCalculator{},
	}
}

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

	for _, calculator := range allCalculators() {
		properties.Property(calculator.Name()+" satisfies Cassini's Identity", prop.ForAll(
			func(n uint64) bool {
				if n == 0 {
					n = 1
				}
				if n > 25000 {
					n = 25000
				}

				fnMinus1, err := calcF(calculator, n-1)
				if err != nil {
					t.Logf("Error calculating F(%d-1): %v", n, err)
					return false
				}
				fn, err := calcF(calculator, n)
				if err != nil {
					t.Logf("Error calculating F(%d): %v", n, err)
					return false
				}
				fnPlus1, err := calcF(calculator, n+1)
				if err != nil {
					t.Logf("Error calculating F(%d+1): %v", n, err)
					return false
				}

				// Left side: F(n-1) * F(n+1) - F(n)²
				leftSide := new(big.Int)
				fnSquared := new(big.Int).Mul(fn, fn)
				leftSide.Mul(fnMinus1, fnPlus1).Sub(leftSide, fnSquared)

				// Right side: (-1)ⁿ
				rightSide := big.NewInt(1)
				if n%2 != 0 {
					rightSide.Neg(rightSide)
				}

				return leftSide.Cmp(rightSide) == 0
			},
			gen.UInt64Range(1, 25000),
		))
	}

	properties.TestingRun(t)
}

// TestRecurrenceRelation_PropertyBased verifies the fundamental recurrence:
//
//	F(n) = F(n-1) + F(n-2)  for n >= 2
//
// This is the defining property of the Fibonacci sequence.
func TestRecurrenceRelation_PropertyBased(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	for _, calculator := range allCalculators() {
		properties.Property(calculator.Name()+" satisfies recurrence F(n) = F(n-1) + F(n-2)", prop.ForAll(
			func(n uint64) bool {
				if n < 2 {
					n = 2
				}
				if n > 25000 {
					n = 25000
				}

				fn, err := calcF(calculator, n)
				if err != nil {
					return false
				}
				fn1, err := calcF(calculator, n-1)
				if err != nil {
					return false
				}
				fn2, err := calcF(calculator, n-2)
				if err != nil {
					return false
				}

				sum := new(big.Int).Add(fn1, fn2)
				return fn.Cmp(sum) == 0
			},
			gen.UInt64Range(2, 25000),
		))
	}

	properties.TestingRun(t)
}

// TestDoublingIdentity_PropertyBased verifies the doubling identity:
//
//	F(2n) = F(n) * (2*F(n+1) - F(n))
//
// This is the identity at the heart of the Fast Doubling algorithm.
func TestDoublingIdentity_PropertyBased(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	for _, calculator := range allCalculators() {
		properties.Property(calculator.Name()+" satisfies doubling identity F(2n) = F(n)*(2*F(n+1)-F(n))", prop.ForAll(
			func(n uint64) bool {
				if n == 0 {
					n = 1
				}
				if n > 12500 {
					n = 12500 // 2n stays within 25000
				}

				fn, err := calcF(calculator, n)
				if err != nil {
					return false
				}
				fn1, err := calcF(calculator, n+1)
				if err != nil {
					return false
				}
				f2n, err := calcF(calculator, 2*n)
				if err != nil {
					return false
				}

				// F(2n) = F(n) * (2*F(n+1) - F(n))
				twoFn1 := new(big.Int).Lsh(fn1, 1)       // 2*F(n+1)
				twoFn1.Sub(twoFn1, fn)                   // 2*F(n+1) - F(n)
				expected := new(big.Int).Mul(fn, twoFn1) // F(n) * (...)

				return f2n.Cmp(expected) == 0
			},
			gen.UInt64Range(1, 12500),
		))
	}

	properties.TestingRun(t)
}

// TestGCDIdentity_PropertyBased verifies the GCD identity:
//
//	GCD(F(m), F(n)) = F(GCD(m, n))
//
// This is a deep number-theoretic property of the Fibonacci sequence.
func TestGCDIdentity_PropertyBased(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Use a single fast calculator since we're testing a mathematical property,
	// not cross-validating implementations.
	calculator := &OptimizedFastDoubling{}

	properties.Property("GCD(F(m), F(n)) = F(GCD(m, n))", prop.ForAll(
		func(m, n uint64) bool {
			if m == 0 {
				m = 1
			}
			if n == 0 {
				n = 1
			}
			if m > 5000 {
				m = 5000
			}
			if n > 5000 {
				n = 5000
			}

			fm, err := calcF(calculator, m)
			if err != nil {
				return false
			}
			fn, err := calcF(calculator, n)
			if err != nil {
				return false
			}

			// GCD of the two indices
			gcdMN := gcdUint64(m, n)
			fGCD, err := calcF(calculator, gcdMN)
			if err != nil {
				return false
			}

			// GCD(F(m), F(n))
			gcdResult := new(big.Int).GCD(nil, nil, fm, fn)

			return gcdResult.Cmp(fGCD) == 0
		},
		gen.UInt64Range(1, 5000),
		gen.UInt64Range(1, 5000),
	))

	properties.TestingRun(t)
}

// gcdUint64 computes the greatest common divisor of a and b.
func gcdUint64(a, b uint64) uint64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
