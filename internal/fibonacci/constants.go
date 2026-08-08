package fibonacci

// ─────────────────────────────────────────────────────────────────────────────
// Performance Tuning Constants
// ─────────────────────────────────────────────────────────────────────────────
//
// These constants are the LAST-RESORT defaults of the threshold resolution
// chain (see internal/config/thresholds.go): normalizeOptions substitutes them
// only for a threshold still left at 0. They are starting points, not tuned
// values — no benchmark in this repo pins any of them, and the calibration
// subsystem (internal/calibration) exists precisely to replace them with
// values measured on the host.

const (
	// DefaultParallelThreshold is the default bit size threshold at which
	// multiplications of large integers are parallelized across multiple cores.
	// Below this threshold, the overhead of goroutine creation is expected to
	// exceed the benefits of parallelism. 4096 is a conservative starting
	// point, not a measured optimum; config.EstimateOptimalParallelThreshold
	// refines it from core count and SIMD level, and calibration replaces it
	// with a timed value.
	DefaultParallelThreshold = 4096

	// DefaultFFTThreshold is the default bit size threshold at which the
	// algorithm switches from standard math/big multiplication to
	// FFT-based multiplication (Schönhage-Strassen).
	//
	// Below this threshold, math/big's O(n^1.585) complexity (Karatsuba
	// internally) wins on constant factors; above it, FFT's O(n log n) does.
	// 500,000 bits is a deliberately conservative placement of that crossover,
	// not a measured one — the actual crossover is host-dependent and is what
	// (*MicroBenchmark).findFFTCrossover, in internal/calibration, measures.
	DefaultFFTThreshold = 500_000

	// DefaultStrassenThreshold is the default bit size threshold at which
	// matrix multiplication switches to Strassen's algorithm.
	//
	// Strassen reduces multiplications from 8 to 7 at the cost of more
	// additions, so it only pays once the operands are large enough for a
	// saved multiplication to outweigh the extra adds. 3072 bits is the
	// default placement of that crossover; only CompleteStrategy calibration
	// measures it (the micro-benchmarks do not exercise matrix multiplication).
	DefaultStrassenThreshold = 3072

	// ParallelFFTThreshold is the bit size threshold above which parallel
	// execution of FFT multiplications becomes beneficial.
	//
	// FFT implementations (like bigfft) often saturate CPU cores internally.
	// Running multiple FFT operations in parallel causes contention and
	// reduces performance for numbers below this threshold. Consumed by
	// shouldParallelizeMultiplicationCached (fastdoubling.go), which is the
	// only reader.
	//
	// The value was lowered from 10M to 5M bits for high-core-count CPUs. No
	// benchmark backing that change survives in the repo — docs/audits/
	// bench-baseline.txt tops out at F(10M) (≈6.9M bits) and does not vary
	// this constant — so treat 5M as an unvalidated setting, not a measured
	// crossover.
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
