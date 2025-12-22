// Package calibration provides performance calibration for the Fibonacci calculator.
// This file implements fast micro-benchmarks for quick threshold estimation (~100ms).
package calibration

import (
	"context"
	"math/big"
	"runtime"
	"sync"
	"time"

	"github.com/agbru/fibcalc/internal/bigfft"
)

// ─────────────────────────────────────────────────────────────────────────────
// Micro-benchmark Configuration
// ─────────────────────────────────────────────────────────────────────────────

const (
	// MicroBenchIterations is the number of iterations per test for averaging.
	MicroBenchIterations = 3

	// MicroBenchTimeout is the maximum time for the entire micro-benchmark suite.
	MicroBenchTimeout = 150 * time.Millisecond

	// MicroBenchPerTestTimeout is the maximum time per individual test.
	MicroBenchPerTestTimeout = 30 * time.Millisecond
)

// MicroBenchTestSizes defines the word sizes to test for threshold estimation.
// These sizes are chosen to span the critical ranges where algorithm switches occur.
var MicroBenchTestSizes = []int{
	500,   // ~32K bits - small, definitely Karatsuba
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
		// For each size, test: Karatsuba seq, Karatsuba par, FFT seq, FFT par
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
func (mb *MicroBenchmark) runSingleTest(ctx context.Context, wordSize int, useFFT, parallel bool) (time.Duration, error) {
	// Create test numbers
	x := generateTestNumber(wordSize)
	y := generateTestNumber(wordSize)

	// Warm up
	_ = multiplyTest(x, y, useFFT)

	// Run timed iterations
	var totalDuration time.Duration
	for i := 0; i < mb.Iterations; i++ {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		start := time.Now()
		_ = multiplyTest(x, y, useFFT)
		totalDuration += time.Since(start)
	}

	return totalDuration / time.Duration(mb.Iterations), nil
}

// generateTestNumber creates a random-ish big.Int with the specified word count.
func generateTestNumber(words int) *big.Int {
	// Use a deterministic pattern for reproducibility
	bits := make([]big.Word, words)
	for i := range bits {
		// Pattern that exercises all bits without being uniform
		bits[i] = big.Word(0xAAAAAAAAAAAAAAAA ^ uint64(i*0x1234567))
	}
	z := new(big.Int)
	z.SetBits(bits)
	return z
}

// multiplyTest performs a multiplication using the specified method.
func multiplyTest(x, y *big.Int, useFFT bool) *big.Int {
	if useFFT {
		result, _ := bigfft.Mul(x, y)
		return result
	}
	return new(big.Int).Mul(x, y)
}

// analyzeResults examines test results to determine optimal thresholds.
func (mb *MicroBenchmark) analyzeResults(results []testResult) ThresholdResults {
	tr := ThresholdResults{
		// Start with conservative defaults
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

	// Analyze FFT crossover point
	fftCrossover := mb.findFFTCrossover(bySize)
	if fftCrossover > 0 {
		tr.FFTThreshold = fftCrossover
		tr.Confidence += 0.2
	}

	// Analyze parallel crossover point
	parallelCrossover := mb.findParallelCrossover(bySize)
	if parallelCrossover > 0 {
		tr.ParallelThreshold = parallelCrossover
		tr.Confidence += 0.2
	}

	// Cap confidence at 1.0
	if tr.Confidence > 1.0 {
		tr.Confidence = 1.0
	}

	return tr
}

// findFFTCrossover determines the bit size where FFT becomes faster than Karatsuba.
func (mb *MicroBenchmark) findFFTCrossover(bySize map[int][]testResult) int {
	var crossoverSize int

	for size, results := range bySize {
		var karatsubaDur, fftDur time.Duration
		var karatsubaCount, fftCount int

		for _, r := range results {
			if r.useFFT {
				fftDur += r.duration
				fftCount++
			} else {
				karatsubaDur += r.duration
				karatsubaCount++
			}
		}

		if karatsubaCount > 0 && fftCount > 0 {
			avgKaratsuba := karatsubaDur / time.Duration(karatsubaCount)
			avgFFT := fftDur / time.Duration(fftCount)

			// FFT is faster at this size
			if avgFFT < avgKaratsuba {
				bitSize := size * 64 // Words to bits (64-bit)
				if crossoverSize == 0 || bitSize < crossoverSize {
					crossoverSize = bitSize
				}
			}
		}
	}

	// If no crossover found, use a high default
	if crossoverSize == 0 {
		return 1000000
	}

	// Add some margin (FFT should be clearly better)
	return crossoverSize * 9 / 10
}

// findParallelCrossover determines the bit size where parallelism becomes beneficial.
func (mb *MicroBenchmark) findParallelCrossover(bySize map[int][]testResult) int {
	if runtime.NumCPU() <= 1 {
		return 0 // No parallelism on single-core
	}

	var crossoverSize int

	for size, results := range bySize {
		var seqDur, parDur time.Duration
		var seqCount, parCount int

		for _, r := range results {
			if !r.useFFT { // Only compare Karatsuba seq vs par
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

	// If no crossover found, use default
	if crossoverSize == 0 {
		return 4096
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

// QuickCalibrateWithDefaults performs quick calibration and returns values
// that can be directly used as configuration defaults.
//
// Parameters:
//   - ctx: The context for cancellation
//   - defaultFFT: The default FFT threshold to use if calibration fails
//   - defaultParallel: The default parallel threshold to use if calibration fails
//
// Returns:
//   - fftThreshold: The calibrated or default FFT threshold
//   - parallelThreshold: The calibrated or default parallel threshold
func QuickCalibrateWithDefaults(ctx context.Context, defaultFFT, defaultParallel int) (int, int) {
	results, err := QuickCalibrate(ctx)
	if err != nil || results.Confidence < 0.3 {
		return defaultFFT, defaultParallel
	}
	return results.FFTThreshold, results.ParallelThreshold
}
