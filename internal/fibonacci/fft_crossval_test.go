package fibonacci

import (
	"context"
	"math/big"
	"testing"
)

// TestFFTRegimeCrossValidation cross-validates the Fast Doubling and Matrix
// Exponentiation calculators in the default FFT regime, then recoups the
// result against an independent modular oracle.
//
// n = 1_000_000 yields ~694k bits, comfortably past DefaultFFTThreshold
// (500_000 bits), so both calculators exercise the FFT multiplication path
// under their default thresholds rather than schoolbook/Karatsuba.
func TestFFTRegimeCrossValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping FFT-regime cross-validation in -short mode")
	}
	t.Parallel()

	const n = uint64(1_000_000)

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

	mx := MustNewCalculator(&MatrixExponentiationCalculator{})
	gotMX, err := mx.Calculate(ctx, nil, 0, n, opts)
	if err != nil {
		t.Fatalf("Matrix F(%d) failed: %v", n, err)
	}

	if gotFD.Cmp(gotMX) != 0 {
		t.Errorf("FastDoubling and Matrix disagree at F(%d)", n)
	}

	// Independent modular oracle: the last 50 decimal digits must match.
	mod := new(big.Int).Exp(big.NewInt(10), big.NewInt(50), nil)
	want := new(big.Int).Mod(gotFD, mod)
	gotMod, err := FastDoublingMod(n, mod)
	if err != nil {
		t.Fatalf("FastDoublingMod(%d) failed: %v", n, err)
	}
	if gotMod.Cmp(want) != 0 {
		t.Errorf("modular oracle mismatch at F(%d) mod 10^50:\n  FastDoublingMod = %s\n  F(n) mod        = %s",
			n, gotMod, want)
	}
}
