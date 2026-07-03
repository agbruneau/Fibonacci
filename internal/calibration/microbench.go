// This file implements fast micro-benchmarks for quick threshold estimation (~100ms).

package calibration

import (
	"context"
	"math/big"
	"runtime"
	"sync"
	"time"

	"github.com/agbruneau/FibGo/internal/bigfft"
	"github.com/agbruneau/FibGo/internal/config"
)

// ─────────────────────────────────────────────────────────────────────────────
// Micro-benchmark Configuration
// ─────────────────────────────────────────────────────────────────────────────

const (
	// MicroBenchIterations is the number of iterations per test for averaging.
	MicroBenchIterations = 3

	// MicroBenchPerTestTimeout is the maximum time per individual test.
	MicroBenchPerTestTimeout = 30 * time.Millisecond
)

// MicroBenchTimeout is the maximum time for the entire micro-benchmark suite.
//
// Sourced from config.DefaultThresholdTuning.MicroBenchTimeout since audit
// R4.2 — the canonical value lives there alongside the other dynamic-tuning
// knobs. Declared as a var (not const) so callers can re-point this package
// at an alternative profile in tests if needed without changing the
// public symbol name.
var MicroBenchTimeout = config.DefaultThresholdTuning.MicroBenchTimeout

// MicroBenchTestSizes defines the word sizes to test for threshold estimation.
// These sizes are chosen to span the critical ranges where algorithm switches occur.
var MicroBenchTestSizes = []int{
	500,   // ~32K bits - small, standard math/big territory
	2000,  // ~128K bits - medium, near parallel threshold
	8000,  // ~512K bits - large, near FFT threshold
	16000, // ~1M bits - very large, FFT territory
}

// ─────────────────────────────────────────────────────────────────────────────
// Micro-benchmark Types
// ─────────────────────────────────────────────────────────────────────────────

// MicroBenchmark performs fast tests to estimate optimal thresholds.
type MicroBenchmark struct {
	// TestSizes are the word sizes to test (default: MicroBenchTestSizes)
	TestSizes []int
	// Iterations is the number of iterations per test (default: MicroBenchIterations)
	Iterations int
	// Timeout is the maximum duration for the entire benchmark
	Timeout time.Duration
}

// ThresholdResults contains the estimated optimal thresholds from micro-benchmarks.
type ThresholdResults struct {
	// FFTThreshold is the estimated optimal FFT threshold in bits
	FFTThreshold int
	// ParallelThreshold is the estimated optimal parallel threshold in bits
	ParallelThreshold int
	// Confidence is a score from 0-1 indicating result reliability
	Confidence float64
	// Duration is how long the micro-benchmark took
	Duration time.Duration
}

// testResult holds timing data for a single configuration test.
type testResult struct {
	wordSize int
	useFFT   bool
	parallel bool
	duration time.Duration
	err      error
}

// ─────────────────────────────────────────────────────────────────────────────
// Micro-benchmark Implementation
// ─────────────────────────────────────────────────────────────────────────────

// NewMicroBenchmark creates a new MicroBenchmark with default settings.
func NewMicroBenchmark() *MicroBenchmark {
	return &MicroBenchmark{
		TestSizes:  MicroBenchTestSizes,
		Iterations: MicroBenchIterations,
		Timeout:    MicroBenchTimeout,
	}
}

// RunQuick performs rapid micro-benchmarks to estimate optimal thresholds.
// It tests multiplication performance with different configurations and
// uses the results to estimate where FFT and parallelism become beneficial.
//
// Returns:
//   - ThresholdResults: The estimated optimal thresholds
//   - error: An error if the benchmark failed critically
func (mb *MicroBenchmark) RunQuick(ctx context.Context) (ThresholdResults, error) {
	start := time.Now()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, mb.Timeout)
	defer cancel()

	// Run tests in parallel for speed
	results := mb.runParallelTests(ctx)

	// Analyze results to determine optimal thresholds
	thresholds := mb.analyzeResults(results)
	thresholds.Duration = time.Since(start)

	// FIB-03: RunQuick used to always report success even when nothing was
	// measured, letting a zero-confidence result look like a completed
	// benchmark to callers checking only the error. Surface the context
	// error when analyzeResults found no valid measurement to work with
	// (results may be non-empty but entirely errored, e.g. every worker
	// observed ctx.Done() before timing anything).
	if thresholds.Confidence == 0 {
		if err := ctx.Err(); err != nil {
			return thresholds, err
		}
	}

	return thresholds, nil
}

// runParallelTests executes multiplication tests in parallel.
func (mb *MicroBenchmark) runParallelTests(ctx context.Context) []testResult {
	var results []testResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Test configurations: (size, useFFT, parallel)
	type testConfig struct {
		wordSize int
		useFFT   bool
		parallel bool
	}

	configs := make([]testConfig, 0, len(mb.TestSizes)*4)
	for _, size := range mb.TestSizes {
		// For each size, test: math/big seq, math/big par, FFT seq, FFT par
		configs = append(configs,
			testConfig{size, false, false},
			testConfig{size, false, true},
			testConfig{size, true, false},
			testConfig{size, true, true},
		)
	}

	// Limit concurrency to avoid overwhelming the system
	semaphore := make(chan struct{}, runtime.NumCPU())

	for _, cfg := range configs {
		wg.Add(1)
		go func(c testConfig) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			}

			dur, err := mb.runSingleTest(ctx, c.wordSize, c.useFFT, c.parallel)

			mu.Lock()
			results = append(results, testResult{
				wordSize: c.wordSize,
				useFFT:   c.useFFT,
				parallel: c.parallel,
				duration: dur,
				err:      err,
			})
			mu.Unlock()
		}(cfg)
	}

	wg.Wait()
	return results
}

// runSingleTest performs a single multiplication test.
//
// The `parallel` flag is retained (even though the current implementation
// does not branch on it) because callers pass cfg.parallel to record the
// intent of the benchmark, and future work — e.g. actually running the
// multiplication through the parallel fibonacci harness — is expected to
// consume it. See P1-07.
func (mb *MicroBenchmark) runSingleTest(ctx context.Context, wordSize int, useFFT, parallel bool) (time.Duration, error) {
	_ = parallel // documented above; silence unparam without dropping the knob

	// Honor cancellation before the (potentially expensive) warm-up multiply,
	// not only inside the timed loop — a large wordSize warm-up can run long.
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	// Create test numbers
	x := generateTestNumber(wordSize)
	y := generateTestNumber(wordSize)

	// Warm up
	multiplyTest(x, y, useFFT)

	// Run timed iterations
	var totalDuration time.Duration
	for i := 0; i < mb.Iterations; i++ {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		start := time.Now()
		multiplyTest(x, y, useFFT)
		totalDuration += time.Since(start)
	}

	return totalDuration / time.Duration(mb.Iterations), nil
}

// generateTestNumber creates a random-ish big.Int with the specified word count.
func generateTestNumber(words int) *big.Int {
	// Use a deterministic pattern for reproducibility
	bits := make([]big.Word, words)
	for i := range bits {
		// Pattern that exercises all bits without being uniform.
		// i is a loop index over make([]big.Word, words) with words >= 0,
		// so uint64(i*0x1234567) never overflows meaningfully here. #nosec G115
		bits[i] = big.Word(0xAAAAAAAAAAAAAAAA ^ uint64(i*0x1234567))
	}
	z := new(big.Int)
	z.SetBits(bits)
	return z
}

// multiplyTest performs a multiplication using the specified method.
//
// The return value is intentionally discarded — every caller runs this
// for its side effect (measuring wall-clock time). Previously the
// function returned the product; P1-07 dropped the unused result to
// clean up the unparam finding while keeping the work the compiler
// actually performs (Mul still writes to a heap big.Int).
func multiplyTest(x, y *big.Int, useFFT bool) {
	if useFFT {
		_, _ = bigfft.Mul(x, y)
		return
	}
	_ = new(big.Int).Mul(x, y)
}

// analyzeResults examines test results to determine optimal thresholds.
//
// FIB-03: find*Crossover report 0 when they have no measured crossover to
// offer (see their doc comments); the conservative defaults live here, and
// applying a default earns no confidence bonus — confidence must reflect
// only what was actually measured. The historical +0.2 "parallel crossover
// found" bonus is dropped entirely: runSingleTest does not branch on the
// parallel flag, so any detected "crossover" was measurement noise between
// two identical configurations, not a real signal.
func (mb *MicroBenchmark) analyzeResults(results []testResult) ThresholdResults {
	tr := ThresholdResults{
		// Conservative defaults, used only when no measurement says otherwise.
		FFTThreshold:      500000,
		ParallelThreshold: 4096,
		Confidence:        0.5,
	}

	if len(results) == 0 {
		// If no results obtained (e.g. timeout), set confidence to zero
		tr.Confidence = 0.0
		return tr
	}

	// Group results by word size
	bySize := make(map[int][]testResult)
	for _, r := range results {
		if r.err == nil {
			bySize[r.wordSize] = append(bySize[r.wordSize], r)
		}
	}

	if len(bySize) == 0 {
		// Every result errored: no valid measurement at all.
		tr.Confidence = 0.0
		return tr
	}

	// Analyze FFT crossover point; only a real measurement earns confidence.
	if fftCrossover := mb.findFFTCrossover(bySize); fftCrossover > 0 {
		tr.FFTThreshold = fftCrossover
		tr.Confidence += 0.2
	}

	// Parallel crossover is not currently measured (runSingleTest ignores
	// the parallel flag); keep the conservative default without a
	// confidence bonus.
	_ = mb.findParallelCrossover(bySize)

	// Cap confidence at 1.0
	if tr.Confidence > 1.0 {
		tr.Confidence = 1.0
	}

	return tr
}

// findFFTCrossover determines the bit size where FFT becomes faster than
// standard math/big. Returns 0 when no crossover was observed in the
// measured data — analyzeResults owns the conservative default for that
// case (FIB-03: a fallback value is not a measurement and must not be
// treated as one for confidence purposes).
func (mb *MicroBenchmark) findFFTCrossover(bySize map[int][]testResult) int {
	var crossoverSize int

	for size, results := range bySize {
		var stdDur, fftDur time.Duration
		var stdCount, fftCount int

		for _, r := range results {
			if r.useFFT {
				fftDur += r.duration
				fftCount++
			} else {
				stdDur += r.duration
				stdCount++
			}
		}

		if stdCount > 0 && fftCount > 0 {
			avgStd := stdDur / time.Duration(stdCount)
			avgFFT := fftDur / time.Duration(fftCount)

			// FFT is faster at this size
			if avgFFT < avgStd {
				bitSize := size * 64 // Words to bits (64-bit)
				if crossoverSize == 0 || bitSize < crossoverSize {
					crossoverSize = bitSize
				}
			}
		}
	}

	// Add some margin (FFT should be clearly better). crossoverSize == 0
	// (no crossover observed) passes through unchanged: 0 * 9 / 10 == 0.
	return crossoverSize * 9 / 10
}

// findParallelCrossover determines the bit size where parallelism becomes
// beneficial. Returns 0 when no crossover was observed — analyzeResults
// owns the conservative default (see findFFTCrossover; FIB-03 applies
// identically here, and today the crossover it computes is unused for a
// confidence bonus since runSingleTest does not branch on parallel).
func (mb *MicroBenchmark) findParallelCrossover(bySize map[int][]testResult) int {
	if runtime.NumCPU() <= 1 {
		return 0 // No parallelism on single-core
	}

	var crossoverSize int

	for size, results := range bySize {
		var seqDur, parDur time.Duration
		var seqCount, parCount int

		for _, r := range results {
			if !r.useFFT { // Only compare math/big seq vs par
				if r.parallel {
					parDur += r.duration
					parCount++
				} else {
					seqDur += r.duration
					seqCount++
				}
			}
		}

		if seqCount > 0 && parCount > 0 {
			avgSeq := seqDur / time.Duration(seqCount)
			avgPar := parDur / time.Duration(parCount)

			// Parallel is faster at this size (require at least 10% improvement)
			if avgPar < avgSeq*9/10 {
				bitSize := size * 64
				if crossoverSize == 0 || bitSize < crossoverSize {
					crossoverSize = bitSize
				}
			}
		}
	}

	return crossoverSize
}

// ─────────────────────────────────────────────────────────────────────────────
// Quick Calibration Function
// ─────────────────────────────────────────────────────────────────────────────

// QuickCalibrate performs a fast calibration using micro-benchmarks.
// This is designed to run in ~100ms and provide reasonable threshold estimates.
//
// Parameters:
//   - ctx: The context for cancellation
//
// Returns:
//   - ThresholdResults: The estimated optimal thresholds
//   - error: An error if calibration failed
func QuickCalibrate(ctx context.Context) (ThresholdResults, error) {
	mb := NewMicroBenchmark()
	return mb.RunQuick(ctx)
}
