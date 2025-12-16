package fibonacci

import (
	"context"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// Cache Integration Benchmarks
// ─────────────────────────────────────────────────────────────────────────────

// BenchmarkFibonacciListWithoutCache benchmarks calculating a list of Fibonacci
// numbers without caching - each calculation is independent.
func BenchmarkFibonacciListWithoutCache(b *testing.B) {
	calc := NewCalculator(&OptimizedFastDoubling{})
	ctx := context.Background()
	opts := Options{
		ParallelThreshold: DefaultParallelThreshold,
		FFTThreshold:      DefaultFFTThreshold,
		Cache:             nil, // No cache
	}

	// Calculate F(1000) to F(1010) - 10 values
	indices := []uint64{1000, 1001, 1002, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, n := range indices {
			_, _ = calc.Calculate(ctx, nil, 0, n, opts)
		}
	}
}

// BenchmarkFibonacciListWithCache benchmarks calculating a list of Fibonacci
// numbers with caching - repeated lookups hit the cache.
func BenchmarkFibonacciListWithCache(b *testing.B) {
	calc := NewCalculator(&OptimizedFastDoubling{})
	ctx := context.Background()
	cache := NewFibonacciCache(100)
	opts := Options{
		ParallelThreshold: DefaultParallelThreshold,
		FFTThreshold:      DefaultFFTThreshold,
		Cache:             cache,
	}

	// Calculate F(1000) to F(1010) - 10 values
	indices := []uint64{1000, 1001, 1002, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, n := range indices {
			_, _ = calc.Calculate(ctx, nil, 0, n, opts)
		}
	}
}

// BenchmarkFibonacciRepeatedWithoutCache benchmarks the same Fibonacci number
// calculated multiple times without caching.
func BenchmarkFibonacciRepeatedWithoutCache(b *testing.B) {
	calc := NewCalculator(&OptimizedFastDoubling{})
	ctx := context.Background()
	opts := Options{
		ParallelThreshold: DefaultParallelThreshold,
		FFTThreshold:      DefaultFFTThreshold,
		Cache:             nil, // No cache
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = calc.Calculate(ctx, nil, 0, 10000, opts)
	}
}

// BenchmarkFibonacciRepeatedWithCache benchmarks the same Fibonacci number
// calculated multiple times with caching - subsequent calls hit cache.
func BenchmarkFibonacciRepeatedWithCache(b *testing.B) {
	calc := NewCalculator(&OptimizedFastDoubling{})
	ctx := context.Background()
	cache := NewFibonacciCache(100)
	opts := Options{
		ParallelThreshold: DefaultParallelThreshold,
		FFTThreshold:      DefaultFFTThreshold,
		Cache:             cache,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = calc.Calculate(ctx, nil, 0, 10000, opts)
	}
}

// BenchmarkFibonacciLarge100KRepeatedWithCache benchmarks larger calculations
// with cache to show significant speedup.
func BenchmarkFibonacciLarge100KRepeatedWithCache(b *testing.B) {
	calc := NewCalculator(&OptimizedFastDoubling{})
	ctx := context.Background()
	cache := NewFibonacciCache(100)
	opts := Options{
		ParallelThreshold: DefaultParallelThreshold,
		FFTThreshold:      DefaultFFTThreshold,
		Cache:             cache,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = calc.Calculate(ctx, nil, 0, 100_000, opts)
	}
}

// BenchmarkFibonacciLarge100KRepeatedWithoutCache benchmarks larger calculations
// without cache for comparison.
func BenchmarkFibonacciLarge100KRepeatedWithoutCache(b *testing.B) {
	calc := NewCalculator(&OptimizedFastDoubling{})
	ctx := context.Background()
	opts := Options{
		ParallelThreshold: DefaultParallelThreshold,
		FFTThreshold:      DefaultFFTThreshold,
		Cache:             nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = calc.Calculate(ctx, nil, 0, 100_000, opts)
	}
}
