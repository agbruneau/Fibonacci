package fibonacci_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/agbru/fibcalc/internal/bigfft"
	"github.com/agbru/fibcalc/internal/fibonacci"
)

// TestArenaIntegration verifies that the integration between the Calculator and
// the arena allocator works correctly. It checks that calculations (especially
// large ones triggering FFT) complete successfully when the arena is active
// (which it is by default in Calculate).
func TestArenaIntegration(t *testing.T) {
	// Use a large number that triggers FFT multiplication.
	// We need enough bits to exceed the default FFT threshold (usually 1800 words * 64 = 115200 bits).
	// F(200,000) has roughly 138,000 bits.
	n := uint64(200_000)

	calc := fibonacci.NewCalculator(&fibonacci.OptimizedFastDoubling{})

	// Run calculation
	// This will call internal/fibonacci/calculator.go:Calculate which:
	// 1. Creates a NewCalculationArena
	// 2. Calls bigfft.PreWarmPools(n)
	// 3. Passes the arena in Options (though currently mostly for future proofing)
	// 4. Executes the calculation using strategies that use bigfft.MulTo -> global pools
	result, err := calc.Calculate(context.Background(), nil, 0, n, fibonacci.Options{})
	if err != nil {
		t.Fatalf("Calculation failed with arena integration: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	if result.BitLen() < 100_000 {
		t.Errorf("Result seems too small: %d bits", result.BitLen())
	}

	// Verify the result using property: F(n) is close to phi^n / sqrt(5)
	// Or simply verify it's positive and non-zero.
	if result.Sign() <= 0 {
		t.Error("Result should be positive")
	}
}

// TestPreWarmPoolsIntegration explicitly verifies that PreWarmPools does not panic
// and effectively prepares the global pools. While we can't inspect internal pool state,
// we can verify the side effects (successful allocation) and performance stability.
func TestPreWarmPoolsIntegration(t *testing.T) {
	n := uint64(100_000)

	// This function is called by Calculator.Calculate.
	// We call it directly here to ensure it works in isolation.
	bigfft.PreWarmPools(n)

	// If PreWarmPools worked, we should be able to run a calculation
	// immediately after without issues.
	// We simulate a bigfft usage.
	x := new(big.Int).Exp(big.NewInt(2), big.NewInt(50000), nil)
	y := new(big.Int).Exp(big.NewInt(2), big.NewInt(50000), nil)

	// This should use the warmed pools
	z, err := bigfft.Mul(x, y)
	if err != nil {
		t.Fatalf("bigfft.Mul failed after PreWarmPools: %v", err)
	}

	if z.BitLen() < 100000 {
		t.Error("Result too small")
	}
}
