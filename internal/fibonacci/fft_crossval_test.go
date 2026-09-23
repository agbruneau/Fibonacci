package fibonacci

import (
	"context"
	"math/big"
	"testing"
)

// TestCalculators_AboveDefaultFFTThreshold cross-validates the three
// calculators in the default FFT regime, then recoups the result against an
// independent modular oracle.
//
// The FFT branch of AdaptiveStrategy.ExecuteStep is taken when the operand
// FK1 = F(k+1) exceeds DefaultFFTThreshold (500_000 bits), and the largest k a
// doubling loop reaches is about n/2. F(k) has about 0.694·k bits, so n must
// exceed ≈ 1.44M; n = 1_000_000, used here until 2026-09-23, never reached the
// branch (EVAL-13). n = 1_500_000 gives an FK1 of ≈ 520k bits on the last step.
func TestCalculators_AboveDefaultFFTThreshold(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping FFT-regime cross-validation in -short mode")
	}
	t.Parallel()

	const n = uint64(1_500_000)

	ctx := context.Background()
	opts := Options{
		ParallelThreshold: DefaultParallelThreshold,
		FFTThreshold:      DefaultFFTThreshold,
	}

	fd := MustNewCalculator(&FastDoublingCalculator{})
	gotFD, err := fd.Calculate(ctx, nil, 0, n, opts)
	if err != nil {
		t.Fatalf("FastDoubling F(%d) failed: %v", n, err)
	}

	for _, core := range []CoreCalculator{&MatrixExponentiationCalculator{}, &FFTBasedCalculator{}} {
		got, err := MustNewCalculator(core).Calculate(ctx, nil, 0, n, opts)
		if err != nil {
			t.Fatalf("%s F(%d) failed: %v", core.Name(), n, err)
		}
		if gotFD.Cmp(got) != 0 {
			t.Errorf("FastDoubling and %s disagree at F(%d)", core.Name(), n)
		}
	}

	// Independent modular oracle: the last 50 decimal digits must match.
	mod := new(big.Int).Exp(big.NewInt(10), big.NewInt(50), nil)
	want := new(big.Int).Mod(gotFD, mod)
	gotMod, err := FastDoublingMod(context.Background(), n, mod)
	if err != nil {
		t.Fatalf("FastDoublingMod(%d) failed: %v", n, err)
	}
	if gotMod.Cmp(want) != 0 {
		t.Errorf("modular oracle mismatch at F(%d) mod 10^50:\n  FastDoublingMod = %s\n  F(n) mod        = %s",
			n, gotMod, want)
	}
}
