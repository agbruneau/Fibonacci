package fibonacci

import (
	"context"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// Extended Benchmarks for Performance Optimization
// ─────────────────────────────────────────────────────────────────────────────

// BenchmarkFastDoubling100K benchmarks Fast Doubling for n=100,000.
func BenchmarkFastDoubling100K(b *testing.B) {
	runPerfBenchmark(b, NewCalculator(&OptimizedFastDoubling{}), 100_000)
}

// BenchmarkFastDoubling500K benchmarks Fast Doubling for n=500,000.
func BenchmarkFastDoubling500K(b *testing.B) {
	runPerfBenchmark(b, NewCalculator(&OptimizedFastDoubling{}), 500_000)
}

// BenchmarkFastDoubling2M benchmarks Fast Doubling for n=2,000,000.
func BenchmarkFastDoubling2M(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping 2M benchmark in short mode")
	}
	runPerfBenchmark(b, NewCalculator(&OptimizedFastDoubling{}), 2_000_000)
}

// BenchmarkFastDoubling5M benchmarks Fast Doubling for n=5,000,000.
func BenchmarkFastDoubling5M(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping 5M benchmark in short mode")
	}
	runPerfBenchmark(b, NewCalculator(&OptimizedFastDoubling{}), 5_000_000)
}

// BenchmarkFFTBased100K benchmarks FFT-based calculator for n=100,000.
func BenchmarkFFTBased100K(b *testing.B) {
	runPerfBenchmark(b, NewCalculator(&FFTBasedCalculator{}), 100_000)
}

// BenchmarkFFTBased500K benchmarks FFT-based calculator for n=500,000.
func BenchmarkFFTBased500K(b *testing.B) {
	runPerfBenchmark(b, NewCalculator(&FFTBasedCalculator{}), 500_000)
}

// BenchmarkFFTBased2M benchmarks FFT-based calculator for n=2,000,000.
func BenchmarkFFTBased2M(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping 2M benchmark in short mode")
	}
	runPerfBenchmark(b, NewCalculator(&FFTBasedCalculator{}), 2_000_000)
}

// BenchmarkFFTBased5M benchmarks FFT-based calculator for n=5,000,000.
func BenchmarkFFTBased5M(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping 5M benchmark in short mode")
	}
	runPerfBenchmark(b, NewCalculator(&FFTBasedCalculator{}), 5_000_000)
}

// BenchmarkMatrixExp100K benchmarks Matrix Exponentiation for n=100,000.
func BenchmarkMatrixExp100K(b *testing.B) {
	runPerfBenchmark(b, NewCalculator(&MatrixExponentiation{}), 100_000)
}

// BenchmarkMatrixExp500K benchmarks Matrix Exponentiation for n=500,000.
func BenchmarkMatrixExp500K(b *testing.B) {
	runPerfBenchmark(b, NewCalculator(&MatrixExponentiation{}), 500_000)
}

// BenchmarkMatrixExp2M benchmarks Matrix Exponentiation for n=2,000,000.
func BenchmarkMatrixExp2M(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping 2M benchmark in short mode")
	}
	runPerfBenchmark(b, NewCalculator(&MatrixExponentiation{}), 2_000_000)
}

// ─────────────────────────────────────────────────────────────────────────────
// Threshold Comparison Benchmarks
// ─────────────────────────────────────────────────────────────────────────────

// BenchmarkFastDoublingFFTThreshold compares performance with different FFT thresholds.
func BenchmarkFastDoublingFFTThreshold(b *testing.B) {
	n := uint64(1_000_000)
	calc := NewCalculator(&OptimizedFastDoubling{})
	ctx := context.Background()

	thresholds := []int{0, 100_000, 250_000, 500_000, 1_000_000}
	for _, threshold := range thresholds {
		b.Run(thresholdName(threshold), func(b *testing.B) {
			opts := Options{
				ParallelThreshold: DefaultParallelThreshold,
				FFTThreshold:      threshold,
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = calc.Calculate(ctx, nil, 0, n, opts)
			}
		})
	}
}

// BenchmarkFastDoublingParallelThreshold compares performance with different parallel thresholds.
func BenchmarkFastDoublingParallelThreshold(b *testing.B) {
	n := uint64(500_000)
	calc := NewCalculator(&OptimizedFastDoubling{})
	ctx := context.Background()

	thresholds := []int{0, 1024, 2048, 4096, 8192, 16384}
	for _, threshold := range thresholds {
		b.Run(thresholdName(threshold), func(b *testing.B) {
			opts := Options{
				ParallelThreshold: threshold,
				FFTThreshold:      DefaultFFTThreshold,
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = calc.Calculate(ctx, nil, 0, n, opts)
			}
		})
	}
}

func thresholdName(t int) string {
	if t == 0 {
		return "Disabled"
	}
	if t >= 1_000_000 {
		return string(rune('0'+t/1_000_000)) + "M"
	}
	if t >= 1_000 {
		return string(rune('0'+t/1_000)) + "K"
	}
	return string(rune('0' + t))
}

func runPerfBenchmark(b *testing.B, calc Calculator, n uint64) {
	ctx := context.Background()
	opts := Options{
		ParallelThreshold: DefaultParallelThreshold,
		FFTThreshold:      DefaultFFTThreshold,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = calc.Calculate(ctx, nil, 0, n, opts)
	}
}

