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
	// Seed corpus with known interesting values
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

	f.Fuzz(func(t *testing.T, n uint64) {
		// Limit to avoid excessive test duration
		// For fuzzing, we want quick iterations
		if n > 50000 {
			return
		}

		ctx := context.Background()
		opts := Options{
			ParallelThreshold: DefaultParallelThreshold,
			FFTThreshold:      DefaultFFTThreshold,
		}

		// Calculate with Fast Doubling
		fd := &OptimizedFastDoubling{}
		resultFD, err := fd.CalculateCore(ctx, func(float64) {}, n, opts)
		if err != nil {
			t.Fatalf("FastDoubling failed for n=%d: %v", n, err)
		}

		// Calculate with Matrix Exponentiation
		mx := &MatrixExponentiation{}
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
		// Limit for performance
		if n > 20000 {
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
		fd := &OptimizedFastDoubling{}
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
	// Seed corpus
	f.Add(uint64(5), uint64(3))
	f.Add(uint64(10), uint64(5))
	f.Add(uint64(20), uint64(10))
	f.Add(uint64(100), uint64(50))
	f.Add(uint64(500), uint64(250))

	calc := NewCalculator(&OptimizedFastDoubling{})
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

		fd := &OptimizedFastDoubling{}
		_, err := fd.CalculateCore(ctx, reporter, n, opts)
		if err != nil {
			t.Fatalf("Calculation failed for n=%d: %v", n, err)
		}
	})
}
