package fibonacci

import "github.com/agbruneau/FibGo/internal/fibonacci/fibmath"

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
//
// Each comment below separates the ARGUMENT that places the value (a cost
// model, not a measurement) from what a measurement says. The numbers, their
// sources and the one-host measurement are in docs/CALIBRATION.md, § "Where
// the Defaults Come From". Git history does not explain any of the three
// values: they arrived with the initial import (5bfc50e).
//
// The CLI rarely sees these values. Without a cached profile,
// config.ApplyAdaptiveThresholds (called from internal/app/app.go) first
// fills every zero threshold with a hardware estimate. Only the one-CPU
// parallel estimate, which is 0, falls through to here. The values mainly
// reach library callers that leave Options at 0.

const (
	// DefaultParallelThreshold is the operand size, in bits, above which the
	// three multiplications of a doubling step (F(k)·F(k+1), F(k)², F(k+1)²)
	// run on three goroutines via executeParallel3. The matrix path uses the
	// same gate for its 7 or 8 products.
	//
	// Argument (a cost model, not a measurement of the threshold): parallelism
	// saves at most two of the three operations. It pays once two of them cost
	// more than the fixed dispatch cost (three goroutines, semaphore,
	// wake-ups, WaitGroup). At 4096 bits (64 words), the three operations are
	// one product with a single Karatsuba level (math/big
	// karatsubaThreshold = 40 words) and two schoolbook squares
	// (karatsubaSqrThreshold = 80 words). On the host in CALIBRATION.md each
	// took about 1 µs and the dispatch about 1.6 µs. That puts 4096 in the
	// right decade; it does not pin it.
	//
	// Measurement: on one 24-thread host the dispatch cost was not fixed; it
	// grew with operand size, and parallel was still 1.7x slower at 4096 bits.
	// The two were tied at 16,384 bits and parallel won 2.1x at 65,536 bits.
	DefaultParallelThreshold = 4096

	// DefaultFFTThreshold is the operand size, in bits, above which
	// smartMultiply (both operands) and smartSquare hand the work to
	// internal/bigfft (a one-level Schönhage-Strassen FFT: Θ(n^1.585) like
	// Karatsuba, with a smaller constant; see docs/algorithms/FFT.md) instead of
	// math/big (schoolbook below 40 words, Karatsuba O(n^1.585) above). bigfft has its
	// own threshold, 1800 words = 115,200 bits (defaultFFTThresholdWords),
	// under which it falls back to math/big; at 500,000 bits that one is not
	// binding.
	//
	// Argument (not a measurement): counted as in the GMP manual, an FFT of
	// 2^k pieces for a full product does 2^k pointwise products, each about
	// 1/2^(k-2) of the operand size. That is an O(N^(k/(k-2))) method, first
	// below Karatsuba's exponent log2(3) ≈ 1.585 at k = 6 (1.5). The argument
	// places the crossover no lower than k = 6. It says nothing that singles
	// out 500,000 bits.
	//
	// Observed: bigfft's 1800 words rests on an upstream TestCalibrate run
	// (the comment on defaultFFTThresholdWords; the test is not in this
	// repo). There, fftSize picks k = 8, two steps past the argument, the same
	// kind of gap GMP reports against Toom-3. 500,000 bits is 4.3x that
	// threshold (k = 9). No rationale for the 4.3x survives. On one host,
	// bigfft beat math/big from 1800 words on and was 2.3-3x faster at 8000
	// words (512,000 bits). This default is conservative, not an estimate of
	// the crossover. (*MicroBenchmark).findFFTCrossover, in
	// internal/calibration, is what measures it.
	DefaultFFTThreshold = 500_000

	// DefaultStrassenThreshold is the matrix-entry size, in bits, above which
	// multiplyMatrices switches from the classic 2x2 product (8 products, 4
	// additions) to Strassen-Winograd (7 products, 15 additions/subtractions:
	// 8 on entries, 7 on products). Only the res×p step reaches it; squaring
	// goes through squareSymmetricMatrix.
	//
	// Argument (not a measurement): Winograd wins once one entry-sized product
	// M(s) costs more than the extra linear work, about 8·A(s) + 3·A(2s)
	// (A = one big.Int addition). With only one level on a 2x2 matrix, the
	// log2(7) exponent does not apply. The gain is capped at one product in
	// eight (12.5%). With math/big timings from one host, the argument puts
	// the crossover between 512 and 1024 bits, not at 3072.
	//
	// Measurement on the same host agrees with the argument, not with this
	// value. Classic was 1.12x faster at 512 bits. Winograd was 9-13% faster
	// from 1024 bits on, 3072 included. So 3072 has no derivation, and it
	// sits 3-6x above the measured crossover. Only CompleteStrategy
	// calibration measures this threshold (the micro-benchmarks do not
	// exercise matrix multiplication).
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
	//
	// Defined in internal/fibonacci/fibmath, which internal/fibonacci/memory
	// can also import — the two used to carry separate copies of the literal
	// because neither could import the other (audit TYP-04). Kept as an alias
	// here so existing references and the exported name are unchanged.
	FibonacciGrowthFactor = fibmath.GrowthFactor
)

// FFTCacheMaxBytesFactor is the byte budget configureFFTCache gives the global
// transform cache, as a multiple of the byte size of F(n) (audit M-08).
//
// Sized to hold one calculation's transforms: the matrix path — the only
// production path that reaches the cache — retained 20 entries of about twice
// the operand each at F(10M), i.e. roughly 40x. 48x leaves headroom without
// letting the cache grow to the ~256x its entry cap alone would permit.
//
// Do not tighten it without re-running the ADR-0009 R4 protocol: 4x was
// measured at MatrixExp/10M +22% sec/op and +137% allocs/op and rejected.
const FFTCacheMaxBytesFactor = 48
