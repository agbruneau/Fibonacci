package fibonacci

import (
	"context"
	"math/big"
	"testing"
)

// FuzzFastDoublingConsistency verifies that the Fast Doubling algorithm
// produces results consistent with the Matrix Exponentiation algorithm.
// This fuzz test helps catch edge cases and numerical errors that might
// not be covered by unit tests.
func FuzzFastDoublingConsistency(f *testing.F) {
	// Seed corpus with known interesting values, including indices that
	// straddle the FFT activation threshold (Audit-PRD E8-R3 — exercise
	// the FFT regime past F(50000)).
	f.Add(uint64(0))
	f.Add(uint64(1))
	f.Add(uint64(2))
	f.Add(uint64(10))
	f.Add(uint64(50))
	f.Add(uint64(92)) // Near max uint64 Fibonacci
	f.Add(uint64(93)) // Max Fibonacci that fits in uint64
	f.Add(uint64(100))
	f.Add(uint64(500))
	f.Add(uint64(1000))
	f.Add(uint64(5000))
	f.Add(uint64(50_000))  // boundary of the historical cap
	f.Add(uint64(100_000)) // FFT regime entry
	f.Add(uint64(150_000)) // mid FFT regime

	f.Fuzz(func(t *testing.T, n uint64) {
		// Cap raised from 50_000 to 200_000 to exercise the FFT regime.
		// A single calc at F(200_000) takes ~tens of ms on commodity hw —
		// acceptable for the fuzz harness's seconds-long budget per iter.
		if n > 200_000 {
			return
		}

		ctx := context.Background()
		opts := Options{
			ParallelThreshold: DefaultParallelThreshold,
			FFTThreshold:      DefaultFFTThreshold,
		}

		// Calculate with Fast Doubling
		fd := &FastDoublingCalculator{}
		resultFD, err := fd.CalculateCore(ctx, func(float64) {}, n, opts)
		if err != nil {
			t.Fatalf("FastDoubling failed for n=%d: %v", n, err)
		}

		// Calculate with Matrix Exponentiation
		mx := &MatrixExponentiationCalculator{}
		resultMX, err := mx.CalculateCore(ctx, func(float64) {}, n, opts)
		if err != nil {
			t.Fatalf("Matrix failed for n=%d: %v", n, err)
		}

		// Verify consistency between algorithms
		if resultFD.Cmp(resultMX) != 0 {
			t.Errorf("Inconsistent results for n=%d:\n  FastDoubling: %s\n  Matrix:       %s",
				n, resultFD.String(), resultMX.String())
		}

		// Additional sanity checks
		if resultFD.Sign() < 0 {
			t.Errorf("Negative result for n=%d: %s", n, resultFD.String())
		}
	})
}

// FuzzFFTBasedConsistency verifies that the FFT-based calculator produces
// results consistent with the Fast Doubling algorithm.
func FuzzFFTBasedConsistency(f *testing.F) {
	// Seed corpus
	f.Add(uint64(0))
	f.Add(uint64(1))
	f.Add(uint64(100))
	f.Add(uint64(1000))
	f.Add(uint64(5000))
	f.Add(uint64(10000))

	f.Fuzz(func(t *testing.T, n uint64) {
		// Cap raised from 20_000 to 200_000 to exercise the FFT regime
		// at non-trivial sizes (Audit-PRD E8-R3).
		if n > 200_000 {
			return
		}

		ctx := context.Background()
		opts := Options{
			ParallelThreshold: DefaultParallelThreshold,
			FFTThreshold:      0, // Force FFT usage for testing
		}

		// Calculate with FFT-based
		fft := &FFTBasedCalculator{}
		resultFFT, err := fft.CalculateCore(ctx, func(float64) {}, n, opts)
		if err != nil {
			t.Fatalf("FFT failed for n=%d: %v", n, err)
		}

		// Calculate with Fast Doubling (reference)
		fd := &FastDoublingCalculator{}
		resultFD, err := fd.CalculateCore(ctx, func(float64) {}, n, Options{
			ParallelThreshold: DefaultParallelThreshold,
			FFTThreshold:      DefaultFFTThreshold,
		})
		if err != nil {
			t.Fatalf("FastDoubling failed for n=%d: %v", n, err)
		}

		// Verify consistency
		if resultFFT.Cmp(resultFD) != 0 {
			t.Errorf("Inconsistent results for n=%d:\n  FFT:          %s\n  FastDoubling: %s",
				n, resultFFT.String(), resultFD.String())
		}
	})
}

// FuzzFibonacciIdentities verifies mathematical identities of Fibonacci numbers.
// These identities provide an independent verification of the implementation.
func FuzzFibonacciIdentities(f *testing.F) {
	// Seed corpus — includes powers of 2 and known Fibonacci indices
	f.Add(uint64(5), uint64(3))
	f.Add(uint64(10), uint64(5))
	f.Add(uint64(20), uint64(10))
	f.Add(uint64(100), uint64(50))
	f.Add(uint64(500), uint64(250))
	f.Add(uint64(1024), uint64(512))  // 2^10
	f.Add(uint64(144), uint64(72))    // F(12)=144
	f.Add(uint64(233), uint64(117))   // F(13)=233
	f.Add(uint64(2), uint64(1))       // Small edge case
	f.Add(uint64(9999), uint64(5000)) // Near upper bound

	calc := MustNewCalculator(&FastDoublingCalculator{})
	ctx := context.Background()
	opts := Options{ParallelThreshold: DefaultParallelThreshold}

	f.Fuzz(func(t *testing.T, n, m uint64) {
		// Limit for performance and ensure m <= n
		if n > 10000 || m > n {
			return
		}
		if m == 0 {
			return
		}

		// Calculate F(n), F(m), F(n-m), F(n+m), F(n-m+1)
		fn, err := calc.Calculate(ctx, nil, 0, n, opts)
		if err != nil {
			t.Fatalf("Failed to calculate F(%d): %v", n, err)
		}

		fm, err := calc.Calculate(ctx, nil, 0, m, opts)
		if err != nil {
			t.Fatalf("Failed to calculate F(%d): %v", m, err)
		}

		fnm, err := calc.Calculate(ctx, nil, 0, n-m, opts)
		if err != nil {
			t.Fatalf("Failed to calculate F(%d): %v", n-m, err)
		}

		// Identity: F(n+m) = F(n)*F(m+1) + F(n-1)*F(m)
		// We can also verify: F(2n) = F(n) * (2*F(n+1) - F(n))
		if n >= 2 && m == n {
			// Verify doubling identity: F(2n) = F(n) * (2*F(n+1) - F(n))
			f2n, err := calc.Calculate(ctx, nil, 0, 2*n, opts)
			if err != nil {
				t.Fatalf("Failed to calculate F(%d): %v", 2*n, err)
			}

			fn1, err := calc.Calculate(ctx, nil, 0, n+1, opts)
			if err != nil {
				t.Fatalf("Failed to calculate F(%d): %v", n+1, err)
			}

			// 2*F(n+1) - F(n)
			twoFn1 := new(big.Int).Lsh(fn1, 1)
			diff := new(big.Int).Sub(twoFn1, fn)

			// F(n) * (2*F(n+1) - F(n))
			expected := new(big.Int).Mul(fn, diff)

			if f2n.Cmp(expected) != 0 {
				t.Errorf("Doubling identity violated for n=%d:\n  F(2n)=%s\n  F(n)*(2*F(n+1)-F(n))=%s",
					n, f2n.String(), expected.String())
			}
		}

		// Verify d'Ocagne's identity: F(m)*F(n+1) - F(m+1)*F(n) = (-1)^n * F(n-m)
		// This is complex to verify with signs, so we use absolute value
		if n > m {
			fn1, err := calc.Calculate(ctx, nil, 0, n+1, opts)
			if err != nil {
				t.Fatalf("Failed to calculate F(%d): %v", n+1, err)
			}

			fm1, err := calc.Calculate(ctx, nil, 0, m+1, opts)
			if err != nil {
				t.Fatalf("Failed to calculate F(%d): %v", m+1, err)
			}

			// F(m)*F(n+1)
			left := new(big.Int).Mul(fm, fn1)
			// F(m+1)*F(n)
			right := new(big.Int).Mul(fm1, fn)
			// |F(m)*F(n+1) - F(m+1)*F(n)|
			diff := new(big.Int).Sub(left, right)
			diff.Abs(diff)

			// Should equal F(n-m)
			if diff.Cmp(fnm) != 0 {
				t.Errorf("d'Ocagne identity violated for n=%d, m=%d:\n  |F(m)*F(n+1) - F(m+1)*F(n)|=%s\n  F(n-m)=%s",
					n, m, diff.String(), fnm.String())
			}
		}

		// Verify Cassini identity: F(n-1)*F(n+1) - F(n)^2 = (-1)^n
		// Requires n >= 1 so that F(n-1) is defined.
		if n >= 1 {
			fnMinus1, err := calc.Calculate(ctx, nil, 0, n-1, opts)
			if err != nil {
				t.Fatalf("Failed to calculate F(%d): %v", n-1, err)
			}

			fnPlus1, err := calc.Calculate(ctx, nil, 0, n+1, opts)
			if err != nil {
				t.Fatalf("Failed to calculate F(%d): %v", n+1, err)
			}

			// F(n-1)*F(n+1)
			prod := new(big.Int).Mul(fnMinus1, fnPlus1)
			// F(n)^2
			fnSq := new(big.Int).Mul(fn, fn)
			// F(n-1)*F(n+1) - F(n)^2
			cassini := new(big.Int).Sub(prod, fnSq)

			// Expected: (-1)^n — +1 when n is even, -1 when n is odd
			expected := big.NewInt(1)
			if n%2 != 0 {
				expected.SetInt64(-1)
			}

			if cassini.Cmp(expected) != 0 {
				t.Errorf("Cassini identity violated for n=%d:\n  F(n-1)*F(n+1) - F(n)^2 = %s, want %s",
					n, cassini.String(), expected.String())
			}
		}

		// Verify addition identity: F(m+n) = F(m)*F(n+1) + F(m-1)*F(n)
		// Requires m >= 1 so that F(m-1) is defined, and m+n <= 10000.
		if m >= 1 && m+n <= 10000 {
			fmn, err := calc.Calculate(ctx, nil, 0, m+n, opts)
			if err != nil {
				t.Fatalf("Failed to calculate F(%d): %v", m+n, err)
			}

			fn1, err := calc.Calculate(ctx, nil, 0, n+1, opts)
			if err != nil {
				t.Fatalf("Failed to calculate F(%d): %v", n+1, err)
			}

			fmMinus1, err := calc.Calculate(ctx, nil, 0, m-1, opts)
			if err != nil {
				t.Fatalf("Failed to calculate F(%d): %v", m-1, err)
			}

			// F(m)*F(n+1) + F(m-1)*F(n)
			term1 := new(big.Int).Mul(fm, fn1)
			term2 := new(big.Int).Mul(fmMinus1, fn)
			sum := new(big.Int).Add(term1, term2)

			if fmn.Cmp(sum) != 0 {
				t.Errorf("Addition identity violated for m=%d, n=%d:\n  F(m+n)=%s\n  F(m)*F(n+1)+F(m-1)*F(n)=%s",
					m, n, fmn.String(), sum.String())
			}
		}
	})
}

// FuzzFastDoublingMod verifies modular Fibonacci computation for random inputs.
func FuzzFastDoublingMod(f *testing.F) {
	f.Add(uint64(0), int64(1000))
	f.Add(uint64(1), int64(1000))
	f.Add(uint64(100), int64(10000))
	f.Add(uint64(93), int64(1000000))

	f.Fuzz(func(t *testing.T, n uint64, modVal int64) {
		if modVal <= 0 || modVal > 1_000_000_000 {
			t.Skip()
		}
		if n > 100_000 {
			t.Skip()
		}
		m := big.NewInt(modVal)
		result, err := FastDoublingMod(n, m)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if result.Sign() < 0 || result.Cmp(m) >= 0 {
			t.Errorf("result %s out of range [0, %s)", result, m)
		}

		// Independent oracle cross-check for tractable n: compute the full
		// F(n) and reduce it mod m, then compare against the modular result.
		if n <= 10000 {
			calc := MustNewCalculator(&FastDoublingCalculator{})
			full, err := calc.Calculate(context.Background(), nil, 0, n, Options{ParallelThreshold: DefaultParallelThreshold})
			if err != nil {
				t.Fatalf("full F(%d) failed: %v", n, err)
			}
			want := new(big.Int).Mod(full, m)
			if result.Cmp(want) != 0 {
				t.Errorf("FastDoublingMod(%d, %s) = %s, want F(n) mod m = %s", n, m, result, want)
			}
		}
	})
}

// FuzzProgressMonotonicity verifies that progress updates are always monotonically increasing.
func FuzzProgressMonotonicity(f *testing.F) {
	f.Add(uint64(100))
	f.Add(uint64(1000))
	f.Add(uint64(5000))
	f.Add(uint64(10000))

	f.Fuzz(func(t *testing.T, n uint64) {
		if n < 10 || n > 20000 {
			return
		}

		ctx := context.Background()
		opts := Options{ParallelThreshold: DefaultParallelThreshold}

		var lastProgress float64
		reporter := func(progress float64) {
			if progress < lastProgress {
				t.Errorf("Non-monotonic progress for n=%d: %f -> %f", n, lastProgress, progress)
			}
			if progress < 0 || progress > 1 {
				t.Errorf("Invalid progress value for n=%d: %f", n, progress)
			}
			lastProgress = progress
		}

		fd := &FastDoublingCalculator{}
		_, err := fd.CalculateCore(ctx, reporter, n, opts)
		if err != nil {
			t.Fatalf("Calculation failed for n=%d: %v", n, err)
		}
	})
}
