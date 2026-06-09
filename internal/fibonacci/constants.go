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
	// algorithm switches from standard math/big multiplication to
	// FFT-based multiplication (Schönhage-Strassen).
	//
	// Below this threshold, math/big's O(n^1.585) complexity (Karatsuba internally)
	// is faster due to lower constant factors. Above it, FFT's O(n log n) wins.
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
	// Benchmarks on multi-core CPUs (24+ cores) show:
	//   - At 5M bits: parallel becomes beneficial on high-core-count CPUs
	//   - At 173M bits (N=250M): parallel is essential
	//
	// Lowered from 10M to 5M bits based on profiling with modern high-core-count CPUs.
	ParallelFFTThreshold = 5_000_000
)

// ─────────────────────────────────────────────────────────────────────────────
// Progress Reporting Constants
// ─────────────────────────────────────────────────────────────────────────────

const (
	// FibonacciGrowthFactor is log2(phi), where phi ≈ 1.618 (golden ratio).
	// Used to estimate bit length of F(n).
	FibonacciGrowthFactor = 0.69424
)
