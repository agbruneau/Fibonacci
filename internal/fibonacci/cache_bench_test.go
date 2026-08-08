package fibonacci

import (
	"context"
	"testing"

	"github.com/agbruneau/FibGo/internal/bigfft"
)

// BenchmarkCacheImpact measures the performance impact of FFT cache configuration
// for large Fibonacci calculations where FFT is used.
func BenchmarkCacheImpact(b *testing.B) {
	// Test with a large N where FFT will be used
	n := uint64(10_000_000) // F(10M) uses FFT multiplication

	// Reset cache to defaults before each benchmark
	defaultConfig := bigfft.DefaultTransformCacheConfig()
	bigfft.SetTransformCacheConfig(defaultConfig)

	calc := MustNewCalculator(&FastDoublingCalculator{})
	ctx := context.Background()

	opts := Options{
		ParallelThreshold: DefaultParallelThreshold,
		FFTThreshold:      DefaultFFTThreshold,
	}

	b.Run("WithDefaultCache", func(b *testing.B) {
		// Use default cache configuration
		configureFFTCache(opts, n)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := calc.Calculate(ctx, nil, 0, n, opts)
			if err != nil {
				b.Fatalf("Calculation failed: %v", err)
			}
		}
	})

	b.Run("WithOptimizedCache", func(b *testing.B) {
		// P2-01: "Optimized" is a misnomer — this configuration
		// (MinBitLen=50000, MaxEntries=256) was found SLOWER than
		// WithDefaultCache, the diagnosis being that MinBitLen=50000
		// sits below the Fibonacci iteration's break-even point:
		// caching transforms for small-ish operands costs more in
		// hashing + deep-copy than the infrequent hit saves.
		//
		// The ms/op and hit-rate figures this comment used to quote
		// came from audit bench/TEAM_A_PERFORMANCE.md F-A6, which is
		// no longer in the repo, and this benchmark cannot reproduce
		// them: it drives a FastDoublingCalculator, whose FFT step
		// transforms with TransformWithBump and never consults the
		// cache, so the hit rate is 0 under both configurations and
		// the two arms differ only by the configureFFTCache call.
		// Measuring the cache would require a calculator that reaches
		// bigfft.Mul/Sqr — the matrix path.
		//
		// Retained as a benchmark reference so regressions in the
		// default config are visible against the misconfigured
		// baseline. See also options.go doc on FFTCacheMinBitLen.
		enabled := true
		optsOptimized := Options{
			ParallelThreshold:  DefaultParallelThreshold,
			FFTThreshold:       DefaultFFTThreshold,
			FFTCacheMinBitLen:  50000, // Deliberately low — see note above
			FFTCacheMaxEntries: 256,   // Larger cache
			FFTCacheEnabled:    &enabled,
		}
		configureFFTCache(optsOptimized, n)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := calc.Calculate(ctx, nil, 0, n, optsOptimized)
			if err != nil {
				b.Fatalf("Calculation failed: %v", err)
			}
		}
	})

	b.Run("CacheDisabled", func(b *testing.B) {
		// Disable cache to measure baseline
		disabled := false
		optsDisabled := Options{
			ParallelThreshold: DefaultParallelThreshold,
			FFTThreshold:      DefaultFFTThreshold,
			FFTCacheEnabled:   &disabled,
		}
		configureFFTCache(optsDisabled, n)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := calc.Calculate(ctx, nil, 0, n, optsDisabled)
			if err != nil {
				b.Fatalf("Calculation failed: %v", err)
			}
		}
	})

	// Reset to defaults
	bigfft.SetTransformCacheConfig(defaultConfig)
}

// BenchmarkCacheHitRate measures cache hit rate for iterative calculations
// where the same values are squared multiple times.
func BenchmarkCacheHitRate(b *testing.B) {
	// Use a moderate N that will trigger FFT but complete quickly
	n := uint64(1_000_000)

	defaultConfig := bigfft.DefaultTransformCacheConfig()
	bigfft.SetTransformCacheConfig(defaultConfig)

	calc := MustNewCalculator(&FastDoublingCalculator{})
	ctx := context.Background()

	enabled := true
	opts := Options{
		ParallelThreshold: DefaultParallelThreshold,
		FFTThreshold:      DefaultFFTThreshold,
		// Optimize cache for better hit rates
		FFTCacheMinBitLen:  50000,
		FFTCacheMaxEntries: 256,
		FFTCacheEnabled:    &enabled,
	}

	configureFFTCache(opts, n)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := calc.Calculate(ctx, nil, 0, n, opts)
		if err != nil {
			b.Fatalf("Calculation failed: %v", err)
		}
	}

	// Check cache statistics
	cache := bigfft.GetTransformCache()
	stats := cache.Stats()
	b.Logf("Cache stats - Hits: %d, Misses: %d, Hit Rate: %.2f%%, Size: %d",
		stats.Hits, stats.Misses, stats.HitRate*100, stats.Size)

	// Reset to defaults
	bigfft.SetTransformCacheConfig(defaultConfig)
}
