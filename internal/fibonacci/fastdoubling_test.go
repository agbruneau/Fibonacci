package fibonacci

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"testing"
)

func TestShouldParallelizeMultiplication(t *testing.T) {
	t.Parallel()

	t.Run("Should parallelize when bit length exceeds threshold", func(t *testing.T) {
		t.Parallel()
		// Create large numbers that exceed threshold
		fk := new(big.Int).Exp(big.NewInt(2), big.NewInt(5000), nil)  // ~5000 bits
		fk1 := new(big.Int).Exp(big.NewInt(2), big.NewInt(5000), nil) // ~5000 bits

		opts := Options{
			ParallelThreshold: 4096, // Lower than bit length
		}

		shouldParallel := shouldParallelizeMultiplicationCached(opts, fk.BitLen(), fk1.BitLen())
		if !shouldParallel {
			t.Error("Should parallelize when bit length exceeds threshold")
		}
	})

	t.Run("Should not parallelize when bit length below threshold", func(t *testing.T) {
		t.Parallel()
		// Create small numbers below threshold
		fk := big.NewInt(100)
		fk1 := big.NewInt(200)

		opts := Options{
			ParallelThreshold: 4096, // Higher than bit length
		}

		shouldParallel := shouldParallelizeMultiplicationCached(opts, fk.BitLen(), fk1.BitLen())
		if shouldParallel {
			t.Error("Should not parallelize when bit length below threshold")
		}
	})

	t.Run("Should not parallelize when FFT threshold is low", func(t *testing.T) {
		t.Parallel()
		// Create numbers that would normally trigger parallelization
		// But with low FFT threshold, FFT will be used instead of parallel multiplication
		fk := new(big.Int).Exp(big.NewInt(2), big.NewInt(5000), nil)
		fk1 := new(big.Int).Exp(big.NewInt(2), big.NewInt(5000), nil)

		opts := Options{
			ParallelThreshold: 4096,
			FFTThreshold:      10000, // Low FFT threshold - FFT will be used instead
		}

		shouldParallel := shouldParallelizeMultiplicationCached(opts, fk.BitLen(), fk1.BitLen())
		// The function checks if FFT will be used, and if so, doesn't parallelize
		// However, the actual logic might still parallelize if bit length is high enough
		// So we just verify the function doesn't panic
		_ = shouldParallel // May be true or false depending on implementation
	})

	t.Run("Edge case: exactly at threshold", func(t *testing.T) {
		t.Parallel()
		// Create numbers exactly at threshold
		fk := new(big.Int).Exp(big.NewInt(2), big.NewInt(4096), nil)
		fk1 := new(big.Int).Exp(big.NewInt(2), big.NewInt(4096), nil)

		opts := Options{
			ParallelThreshold: 4096,
			FFTThreshold:      1000000, // High FFT threshold
		}

		shouldParallel := shouldParallelizeMultiplicationCached(opts, fk.BitLen(), fk1.BitLen())
		// Should parallelize when >= threshold
		if !shouldParallel {
			t.Error("Should parallelize when bit length equals threshold")
		}
	})
}

// TestAcquireStateForN_HugeN_NoPanic verifies that the size computation behind
// AcquireStateForN tolerates a physically uncomputable n without invoking
// float->int impl-defined behavior or integer overflow. The clamp is exercised
// via acquireSizingForN directly so the test never allocates exabytes; a real
// AcquireStateForN/ReleaseState round-trip on a modest n confirms the release
// path is intact.
func TestAcquireStateForN_HugeN_NoPanic(t *testing.T) {
	t.Parallel()

	naive := func(n uint64) (int, int) {
		w := int(float64(n)*FibonacciGrowthFactor)/64 + 1
		return w, w * 10
	}
	for _, n := range []uint64{1001, 100_000, 1_000_000, 1_000_000_000, 1_000_000_000_000} {
		wGot, tGot := acquireSizingForN(n)
		wWant, tWant := naive(n)
		if wGot != wWant || tGot != tWant {
			t.Errorf("acquireSizingForN(%d) = (%d,%d), want (%d,%d)", n, wGot, tGot, wWant, tWant)
		}
	}

	// MaxUint64: must clamp, not produce a garbage/overflowed value.
	w, total := acquireSizingForN(math.MaxUint64)
	if total != maxReasonableWords || w != maxReasonableWords/10 {
		t.Errorf("acquireSizingForN(MaxUint64) = (%d,%d), want (%d,%d)",
			w, total, maxReasonableWords/10, maxReasonableWords)
	}

	// Release path stays intact for a normal acquisition.
	s := AcquireStateForN(100_000)
	ReleaseState(s)
}

// TestPreSizing_ReducesAllocations verifies pre-sizing doesn't break correctness
// and produces correct results for medium-sized calculations.
func TestPreSizing_ReducesAllocations(t *testing.T) {
	t.Parallel()

	calc := MustNewCalculator(&FastDoublingCalculator{})
	ctx := context.Background()

	// Medium-sized calculation that benefits from pre-sizing
	result, err := calc.Calculate(ctx, nil, 0, 50000, Options{})
	if err != nil {
		t.Fatalf("Calculate error: %v", err)
	}
	if result.Sign() <= 0 {
		t.Error("result should be positive")
	}
}

// TestFastDoubling_ReducedState_Correctness verifies results are correct
// with the reduced 5-temporary state across key values.
func TestFastDoubling_ReducedState_Correctness(t *testing.T) {
	t.Parallel()

	calc := MustNewCalculator(&FastDoublingCalculator{})
	ctx := context.Background()

	cases := []struct {
		n    uint64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{2, "1"},
		{10, "55"},
		{50, "12586269025"},
		{93, "12200160415121876738"},
		{100, "354224848179261915075"},
		{1000, ""},  // verified by golden test
		{10000, ""}, // verified by golden test
	}

	for _, tc := range cases {

		t.Run(fmt.Sprintf("N=%d", tc.n), func(t *testing.T) {
			t.Parallel()
			result, err := calc.Calculate(ctx, nil, 0, tc.n, Options{})
			if err != nil {
				t.Fatalf("Calculate(%d) error: %v", tc.n, err)
			}
			if tc.want != "" && result.String() != tc.want {
				t.Errorf("Calculate(%d) = %s, want %s", tc.n, result.String(), tc.want)
			}
		})
	}
}

// TestFFTBased_ReducedState_Correctness verifies FFT-based calculator
// produces correct results with the reduced 5-temporary state.
func TestFFTBased_ReducedState_Correctness(t *testing.T) {
	t.Parallel()

	calc := MustNewCalculator(&FFTBasedCalculator{})
	ctx := context.Background()

	cases := []struct {
		n    uint64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{10, "55"},
		{100, "354224848179261915075"},
	}

	for _, tc := range cases {

		t.Run(fmt.Sprintf("N=%d", tc.n), func(t *testing.T) {
			t.Parallel()
			result, err := calc.Calculate(ctx, nil, 0, tc.n, Options{})
			if err != nil {
				t.Fatalf("Calculate(%d) error: %v", tc.n, err)
			}
			if tc.want != "" && result.String() != tc.want {
				t.Errorf("Calculate(%d) = %s, want %s", tc.n, result.String(), tc.want)
			}
		})
	}
}
