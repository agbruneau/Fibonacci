//go:build gmp

package fibonacci

import (
	"context"
	"fmt"
	"testing"
)

func TestGMPCalculator_CalculateCore(t *testing.T) {
	t.Parallel()

	calc := &GMPCalculator{}
	ctx := context.Background()
	noopReporter := func(float64) {}
	opts := Options{}

	tests := []struct {
		n    uint64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{2, "1"},
		{3, "2"},
		{4, "3"},
		{5, "5"},
		{10, "55"},
		{20, "6765"},
		{50, "12586269025"},
		{100, "354224848179261915075"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("n=%d", tt.n), func(t *testing.T) {
			t.Parallel()
			got, err := calc.CalculateCore(ctx, noopReporter, tt.n, opts)
			if err != nil {
				t.Errorf("CalculateCore(%d) error = %v", tt.n, err)
				return
			}
			if got.String() != tt.want {
				t.Errorf("CalculateCore(%d) = %v, want %v", tt.n, got, tt.want)
			}
		})
	}
}

func TestGMPCalculator_CalculateCore_Cancel(t *testing.T) {
	t.Parallel()

	calc := &GMPCalculator{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	noopReporter := func(float64) {}
	opts := Options{}

	_, err := calc.CalculateCore(ctx, noopReporter, 1000, opts)
	if err == nil {
		t.Error("CalculateCore(canceled context) expected error, got nil")
	}
}

func TestGMPCalculator_Name(t *testing.T) {
	t.Parallel()

	calc := &GMPCalculator{}
	if calc.Name() != "GMP (Fast Doubling)" {
		t.Errorf("Name() = %v, want %v", calc.Name(), "GMP (Fast Doubling)")
	}
}

// TestGMPCalculator_CrossValidateFastDoubling cross-validates the GMP backend
// against the native Fast Doubling calculator across the small, large, and
// FFT-regime size classes. n=1_000_000 (~694k bits) exceeds DefaultFFTThreshold
// (500_000 bits), so the native path exercises FFT multiplication.
func TestGMPCalculator_CrossValidateFastDoubling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping GMP cross-validation in -short mode")
	}
	t.Parallel()

	ctx := context.Background()
	noop := func(float64) {}
	opts := Options{
		ParallelThreshold: DefaultParallelThreshold,
		FFTThreshold:      DefaultFFTThreshold,
	}

	gmp := &GMPCalculator{}
	fd := &FastDoublingCalculator{}

	for _, n := range []uint64{1000, 100_000, 1_000_000} {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			t.Parallel()

			gotGMP, err := gmp.CalculateCore(ctx, noop, n, opts)
			if err != nil {
				t.Fatalf("GMP F(%d) failed: %v", n, err)
			}

			gotFD, err := fd.CalculateCore(ctx, noop, n, opts)
			if err != nil {
				t.Fatalf("FastDoubling F(%d) failed: %v", n, err)
			}

			if gotGMP.Cmp(gotFD) != 0 {
				t.Errorf("GMP and FastDoubling disagree at F(%d)", n)
			}
		})
	}
}

// BenchmarkGMPCalculator benchmarks the GMP calculator for various input sizes.
// This allows comparison with other calculator implementations.
func BenchmarkGMPCalculator(b *testing.B) {
	calc := &GMPCalculator{}
	ctx := context.Background()
	noopReporter := func(float64) {}
	opts := Options{}

	benchmarks := []uint64{100, 1000, 10000}

	for _, n := range benchmarks {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = calc.CalculateCore(ctx, noopReporter, n, opts)
			}
		})
	}
}
