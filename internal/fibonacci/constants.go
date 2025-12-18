// Package fibonacci provides implementations for calculating Fibonacci numbers.
package fibonacci

// ─────────────────────────────────────────────────────────────────────────────
// Performance Tuning Constants
// ─────────────────────────────────────────────────────────────────────────────
//
// These constants control the behavior of adaptive algorithms and are based on
// empirical benchmarks across various hardware configurations.

const (
	// DefaultParallelThreshold is the default bit size threshold at which
	// multiplications of large integers are parallelized across multiple cores.
	// Below this threshold, the overhead of goroutine creation exceeds the
	// benefits of parallelism.
	//
	// Empirically determined: 4096 bits provides optimal performance on most
	// modern multi-core CPUs for Fibonacci calculations.
	DefaultParallelThreshold = 4096

	// DefaultFFTThreshold is the default bit size threshold at which the
	// algorithm switches from Karatsuba multiplication (used by math/big) to
	// FFT-based multiplication (Schönhage-Strassen).
	//
	// Below this threshold, Karatsuba's O(n^1.585) complexity is faster due to
	// lower constant factors. Above it, FFT's O(n log n) complexity wins.
	// Value of 500,000 bits is optimal on modern CPUs with large L3 caches,
	// providing a good balance between FFT overhead and multiplication gains.
	DefaultFFTThreshold = 500_000

	// DefaultStrassenThreshold is the default bit size threshold at which
	// matrix multiplication switches to Strassen's algorithm.
	//
	// Strassen reduces multiplications from 8 to 7 at the cost of more
	// additions. For small matrices (number sizes), standard multiplication
	// is faster. 3072 bits is the crossover point on typical hardware.
	DefaultStrassenThreshold = 3072

	// ParallelFFTThreshold is the bit size threshold above which parallel
	// execution of FFT multiplications becomes beneficial.
	//
	// FFT implementations (like bigfft) often saturate CPU cores internally.
	// Running multiple FFT operations in parallel causes contention and
	// reduces performance for numbers below this threshold.
	//
	// Benchmarks show:
	//   - At 7M bits (N=10M): sequential is faster (78ms vs 98ms)
	//   - At 173M bits (N=250M): parallel is essential
	//
	// 10,000,000 bits (~3M decimal digits) is the empirical crossover point.
	ParallelFFTThreshold = 10_000_000

	// DefaultKaratsubaThreshold is the bit size threshold at which we switch
	// from math/big's default multiplication to our custom optimized Karatsuba.
	//
	// Values between 2048 and 500,000 bits benefit from the custom memory
	// pooling and parallel recursion available in bigfft.KaratsubaMultiply.
	DefaultKaratsubaThreshold = 2048

	// CalibrationN is the standard Fibonacci index used for performance
	// calibration runs. This value provides a good balance between:
	//   - Being large enough to measure meaningful performance differences
	//   - Being small enough to complete calibration in reasonable time
	//
	// F(10,000,000) has approximately 2,089,877 decimal digits.
	CalibrationN = 10_000_000
)

// ─────────────────────────────────────────────────────────────────────────────
// Progress Reporting Constants
// ─────────────────────────────────────────────────────────────────────────────

const (
	// ProgressReportThreshold is the minimum progress change (0.0 to 1.0) required
	// before a new progress update is sent. This prevents excessive UI updates
	// that could slow down calculations.
	//
	// A value of 0.01 (1%) provides smooth progress updates without overhead.
	ProgressReportThreshold = 0.01
)
