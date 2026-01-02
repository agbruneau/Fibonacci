// Package fibonacci provides implementations for calculating Fibonacci numbers.
// This file contains configuration options for Fibonacci calculations.
package fibonacci

import "github.com/agbru/fibcalc/internal/bigfft"

// Options configures the Fibonacci calculation.
type Options struct {
	// ParallelThreshold is the bit size threshold for parallelizing multiplications.
	// If 0, a default value may be used by the implementation.
	ParallelThreshold int
	// FFTThreshold is the bit size threshold for using FFT-based multiplication.
	// If 0, a default value may be used by the implementation.
	FFTThreshold int
	// KaratsubaThreshold is the bit size threshold for using optimized Karatsuba multiplication.
	// If 0, a default value may be used by the implementation.
	KaratsubaThreshold int
	// StrassenThreshold is the bit size threshold for switching to Strassen's algorithm.
	// If 0, a default value may be used by the implementation.
	StrassenThreshold int
	// FFTCacheMinBitLen is the minimum operand bit length to cache FFT transforms.
	// Smaller values don't benefit from caching. If 0, uses the default (100,000 bits).
	FFTCacheMinBitLen int
	// FFTCacheMaxEntries is the maximum number of cached FFT transforms.
	// If 0, uses the default (128 entries). Larger values improve hit rates
	// but consume more memory.
	FFTCacheMaxEntries int
	// FFTCacheEnabled controls whether FFT transform caching is active.
	// Default is true. Set to false to disable caching (useful for memory-constrained scenarios).
	FFTCacheEnabled *bool
	// EnableDynamicThresholds enables real-time threshold adjustment during calculation.
	// When enabled, the algorithm monitors iteration performance and adjusts FFT and
	// parallel thresholds dynamically based on observed timing.
	// Default is false (use static thresholds).
	EnableDynamicThresholds bool
	// DynamicAdjustmentInterval is the number of iterations between threshold checks.
	// If 0, uses the default (5 iterations). Only used when EnableDynamicThresholds is true.
	DynamicAdjustmentInterval int
}

// normalizeOptions returns a copy of opts with default values filled in for zero values.
// This ensures consistent threshold handling across all calculator implementations.
//
// Parameters:
//   - opts: The options to normalize.
//
// Returns:
//   - Options: A normalized copy of opts with defaults applied.
func normalizeOptions(opts Options) Options {
	normalized := opts
	if normalized.ParallelThreshold == 0 {
		normalized.ParallelThreshold = DefaultParallelThreshold
	}
	if normalized.FFTThreshold == 0 {
		normalized.FFTThreshold = DefaultFFTThreshold
	}
	if normalized.KaratsubaThreshold == 0 {
		normalized.KaratsubaThreshold = DefaultKaratsubaThreshold
	}
	if normalized.StrassenThreshold == 0 {
		normalized.StrassenThreshold = DefaultStrassenThreshold
	}
	return normalized
}

// configureFFTCache configures the FFT transform cache based on the provided options.
// This optimization allows reusing expensive FFT transforms across iterations,
// providing 15-30% speedup for large calculations where FFT is used.
func configureFFTCache(opts Options) {
	// Get default config to use as base
	defaultConfig := bigfft.DefaultTransformCacheConfig()
	config := bigfft.TransformCacheConfig{
		MaxEntries: defaultConfig.MaxEntries,
		MinBitLen:  defaultConfig.MinBitLen,
		Enabled:    defaultConfig.Enabled,
	}

	// Override with user-provided options if specified
	if opts.FFTCacheMaxEntries > 0 {
		config.MaxEntries = opts.FFTCacheMaxEntries
	}
	if opts.FFTCacheMinBitLen > 0 {
		config.MinBitLen = opts.FFTCacheMinBitLen
	}
	if opts.FFTCacheEnabled != nil {
		config.Enabled = *opts.FFTCacheEnabled
	}

	// Apply configuration to global cache
	bigfft.SetTransformCacheConfig(config)
}
