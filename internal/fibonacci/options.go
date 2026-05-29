// This file contains configuration options for Fibonacci calculations.

package fibonacci

import (
	"math/bits"

	"github.com/agbruneau/FibGo/internal/bigfft"
)

// Options configures the Fibonacci calculation.
type Options struct {
	// ParallelThreshold is the bit size threshold for parallelizing multiplications.
	// If 0, a default value may be used by the implementation.
	ParallelThreshold int
	// FFTThreshold is the bit size threshold for using FFT-based multiplication.
	// If 0, a default value may be used by the implementation.
	FFTThreshold int
	// StrassenThreshold is the bit size threshold for switching to Strassen's algorithm.
	// If 0, a default value may be used by the implementation.
	StrassenThreshold int
	// FFTCacheMinBitLen is the minimum operand bit length to cache FFT transforms.
	// Smaller values don't benefit from caching. If 0, uses the default (100,000 bits).
	//
	// P2-01 warning: lowering this below the default (e.g. 50,000) is a
	// pessimisation for typical Fibonacci workloads — operand values
	// change every iteration so the hit rate collapses (~4.55% measured
	// at MinBitLen=50000) and hashing + deep-copy cost dominates. Raise
	// this, don't lower it, unless profiling on your exact workload
	// confirms a win.
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
	// GCMode controls the garbage collector during calculation.
	// Valid values: "auto" (default), "aggressive", "disabled".
	GCMode string
	// MemoryLimitBytes, when > 0, enforces an upper bound on the estimated
	// memory required to compute F(n). If the estimate (see
	// memory.EstimateMemoryUsage) exceeds this limit, Calculate returns an
	// apperrors.MemoryError before any heavy work is done. A zero value
	// disables the check.
	//
	// This is a defence-in-depth complement to config.ValidateMemoryBudget
	// (R3.6): the validator runs at config-parsing time, while this check
	// runs at the calculator boundary so the protection survives any path
	// that bypasses the validator (programmatic embedding, tests, …).
	MemoryLimitBytes uint64
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
	if normalized.StrassenThreshold == 0 {
		normalized.StrassenThreshold = DefaultStrassenThreshold
	}
	return normalized
}

// configureFFTCache configures the FFT transform cache based on the provided options.
// This optimization allows reusing expensive FFT transforms across iterations,
// providing 15-30% speedup for large calculations where FFT is used.
// It sizes the cache dynamically based on the Fibonacci number n to be computed.
func configureFFTCache(opts Options, n uint64) {
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
	} else if n > 0 {
		// Log2(N) iterations roughly, producing at most 2 new transforms each
		dynamicSize := 2 * bits.Len64(n)
		if dynamicSize < 64 {
			dynamicSize = 64
		} else if dynamicSize > 4096 {
			dynamicSize = 4096
		}
		config.MaxEntries = dynamicSize
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
