// This file contains configuration options for Fibonacci calculations.

package fibonacci

import (
	"math"
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
	// pessimisation for typical Fibonacci workloads — operand values change
	// every iteration, so most transforms are cached and never read back
	// while hashing and the deep copy are paid on every put. Raise this,
	// don't lower it, unless profiling on your exact workload confirms a
	// win. The hit-rate and slowdown percentages this comment used to quote
	// came from an audit document (bench/TEAM_A_PERFORMANCE.md) that is no
	// longer in the repo, and nothing replaced them: BenchmarkCacheImpact
	// and BenchmarkCacheHitRate configure the cache but drive a
	// FastDoublingCalculator, whose FFT step never consults it, so they
	// report a 0% hit rate whatever this value is set to.
	FFTCacheMinBitLen int
	// FFTCacheMaxEntries is the maximum number of cached FFT transforms.
	// If 0, configureFFTCache sizes it from n as clamp(2*bits.Len64(n), 64,
	// 4096), which is 64..128 in practice since bits.Len64 caps at 64; the
	// package default of 256 only applies when n is 0. Larger values improve
	// hit rates but consume more memory.
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
	// This is a defense-in-depth complement to config.ValidateMemoryBudget
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

// configureFFTCache installs the process-global bigfft transform cache
// configuration derived from opts, sizing MaxEntries from n when the caller
// did not pin it.
//
// Scope of what this actually affects: the cache is only consulted by
// bigfft.Mul/MulTo/Sqr/SqrTo (fft_core.go fftmulTo/fftsqrTo). The default Fast
// Doubling FFT step goes through executeDoublingStepFFT, which transforms with
// TransformWithBump and never reads the cache — so tuning these knobs does not
// touch that path. The production callers it does reach are the
// matrix-exponentiation calculator (matrix_ops.go, which calls smartMultiply
// and smartSquare from fft.go) and internal/calibration/microbench.go.
//
// No figure for the cache's effect is asserted here: the repo carries no
// artifact measuring it, and no benchmark that would produce one —
// BenchmarkCacheImpact drives a FastDoublingCalculator, which by the paragraph
// above never reaches the cache.
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

	// Bound the cache in BYTES as well as entries (audit M-08). An entry holds
	// K*(n+1) words, roughly twice its operand, so a fixed entry budget is not
	// a memory bound: at F(10M) the matrix path retained 20 entries of ~1.7 MB,
	// and the same 64-128 entry budget at F(100M) would allow gigabytes.
	// Nothing frees the cache between calculations in a long-lived process.
	//
	// FFTCacheMaxBytesFactor is sized to hold ONE calculation's transforms and
	// no more. A tighter bound was measured and rejected: 4x the size of F(n)
	// (2 entries at F(10M)) cost MatrixExp/10M +22% sec/op, +76% B/op and
	// +137% allocs/op on benchstat, well past the 5% gate of the ADR-0009 R4
	// protocol. The zero hit rate measured for a single run is real, but it
	// does not generalise: a repeated calculation of the same n -- the TUI's
	// restart key, a calibration sweep, the benchmarks themselves -- replays
	// identical operands and does hit. Bounding is the goal here; starving is
	// not.
	if n > 0 {
		fibBytes := uint64(float64(n)*FibonacciGrowthFactor) / 8
		if budget := FFTCacheMaxBytesFactor * fibBytes; budget > 0 && budget <= uint64(math.MaxInt) {
			config.MaxBytes = int(budget)
		}
	}

	// Apply configuration to global cache
	bigfft.SetTransformCacheConfig(config)
}
