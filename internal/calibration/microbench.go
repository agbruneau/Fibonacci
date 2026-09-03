// This file implements fast micro-benchmarks for quick threshold estimation (~100ms).

package calibration

import (
	"context"
	"math/big"
	"math/bits"
	"runtime"
	"sort"
	"time"

	"github.com/agbruneau/FibGo/internal/bigfft"
	"github.com/agbruneau/FibGo/internal/config"
)

// ─────────────────────────────────────────────────────────────────────────────
// Micro-benchmark Configuration
// ─────────────────────────────────────────────────────────────────────────────

// MicroBenchIterations is the number of iterations per test for averaging.
//
// Raised from 3 to 7 in audit M-01, paid for by the switch to sequential
// execution: with the 16 configurations no longer contending with each other
// (and with bigfft's own recursion) a full pass costs about 60 ms of the
// 150 ms MicroBenchTimeout, so there is room for more samples. More samples
// tighten the fastest/slowest bracket that timingStability scores, which is
// what decides whether the fast pass is trusted or escalated.
const MicroBenchIterations = 7

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
	// fastest and slowest bracket the individual timed iterations. Their ratio
	// is the dispersion analyzeResults turns into a confidence score (audit
	// M-01): the previous code awarded a flat +0.2 for "a crossover was found"
	// without ever asking whether the timings behind it were stable.
	//
	// The full min/max bracket is deliberate, and deliberately harsh. A
	// median would forgive one descheduled iteration, but the asymmetry of the
	// consequences argues the other way: a wrongly ACCEPTED fast result is
	// persisted to disk and replayed on every subsequent start, while a
	// wrongly REJECTED one only costs a single slower calibration through
	// CompleteStrategy, which is what --auto-calibrate asked for anyway.
	fastest time.Duration
	slowest time.Duration
	err     error
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
	results := mb.runTests(ctx)

	// Analyze results to determine optimal thresholds
	thresholds := mb.analyzeResults(results)
	thresholds.Duration = time.Since(start)

	// FIB-03: RunQuick used to always report success even when nothing was
	// measured, letting a result look like a completed benchmark to callers
	// checking only the error. Surface the context error when not a single
	// timing was collected (results may be non-empty but entirely errored,
	// e.g. every test observed ctx.Done() before timing anything).
	//
	// The test is on the measurements, not on the confidence score. It used to
	// read `Confidence == 0`, which was equivalent while the score started at
	// 0.5 and only a total failure could bring it to zero. Since M-01 rebased
	// it at zero, a zero score also means "measured cleanly, but no decisive
	// crossover" — a legitimate outcome that must NOT be reported as an error.
	if !hasUsableResult(results) {
		if err := ctx.Err(); err != nil {
			return thresholds, err
		}
	}

	return thresholds, nil
}

// hasUsableResult reports whether at least one test produced a timing.
func hasUsableResult(results []testResult) bool {
	for _, r := range results {
		if r.err == nil {
			return true
		}
	}
	return false
}

// runTests executes the multiplication tests that can actually inform a
// crossover, one at a time.
//
// Two changes from the original (audit M-01):
//
// Sequential, not concurrent. The 16 configurations used to run together under
// a NumCPU semaphore while bigfft.Mul parallelises its own recursion, so the
// timings measured contention between the benchmark's own goroutines rather
// than the cost of a multiplication.
//
// Filtered. bigfft.Mul only takes the FFT path above FFTThresholdWords; below
// it, useFFT=true and useFFT=false run the identical math/big code. Timing
// them against each other and calling the difference a crossover reported
// noise as a measurement — with the default 1800-word threshold, the 500-word
// row could declare a crossover at 28800 bits on nothing but jitter.
func (mb *MicroBenchmark) runTests(ctx context.Context) []testResult {
	fftThresholdWords := bigfft.FFTThresholdWords()

	type testConfig struct {
		wordSize int
		useFFT   bool
	}

	configs := make([]testConfig, 0, len(mb.TestSizes)*2)
	for _, size := range mb.TestSizes {
		// The baseline is always worth timing; the FFT arm only when the size
		// is large enough for bigfft to actually take that path.
		configs = append(configs, testConfig{size, false})
		if size > fftThresholdWords {
			configs = append(configs, testConfig{size, true})
		}
	}

	results := make([]testResult, 0, len(configs))
	for _, c := range configs {
		if ctx.Err() != nil {
			break
		}
		dur, fastest, slowest, err := mb.runSingleTest(ctx, c.wordSize, c.useFFT, false)
		results = append(results, testResult{
			wordSize: c.wordSize,
			useFFT:   c.useFFT,
			duration: dur,
			fastest:  fastest,
			slowest:  slowest,
			err:      err,
		})
	}
	return results
}

// runSingleTest performs a single multiplication test and reports the mean
// alongside the fastest and slowest individual iteration.
//
// The bracket is what lets analyzeResults score its own reliability (audit
// M-01): a mean is not evidence on its own, and the previous code awarded
// confidence for finding a crossover without ever looking at how noisy the
// timings behind it were.
//
// The `parallel` flag is retained (even though the current implementation
// does not branch on it) because callers pass it to record the intent of the
// benchmark, and future work — e.g. actually running the multiplication
// through the parallel fibonacci harness — is expected to consume it. See
// P1-07.
func (mb *MicroBenchmark) runSingleTest(ctx context.Context, wordSize int, useFFT, parallel bool) (mean, fastest, slowest time.Duration, err error) {
	_ = parallel // documented above; silence unparam without dropping the knob

	// Honor cancellation before the (potentially expensive) warm-up multiply,
	// not only inside the timed loop — a large wordSize warm-up can run long.
	if err := ctx.Err(); err != nil {
		return 0, 0, 0, err
	}

	// Create test numbers
	x := generateTestNumber(wordSize)
	y := generateTestNumber(wordSize)

	// Warm up
	multiplyTest(x, y, useFFT)

	// Run timed iterations
	samples := make([]time.Duration, 0, mb.Iterations)
	var totalDuration time.Duration
	for i := 0; i < mb.Iterations; i++ {
		select {
		case <-ctx.Done():
			return 0, 0, 0, ctx.Err()
		default:
		}

		start := time.Now()
		multiplyTest(x, y, useFFT)
		elapsed := time.Since(start)

		totalDuration += elapsed
		samples = append(samples, elapsed)
	}

	if len(samples) == 0 {
		return 0, 0, 0, nil
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })

	return totalDuration / time.Duration(len(samples)), samples[0], samples[len(samples)-1], nil
}

// generateTestNumber creates a random-ish big.Int with the specified word count.
func generateTestNumber(words int) *big.Int {
	// Deterministic pattern that exercises all bits without being uniform:
	// word i is 0xAAAA… ^ (i * 0x1234567). The step is accumulated in uint64
	// rather than derived from the loop index, so no signed→unsigned
	// conversion appears; both forms wrap mod 2^64 and yield the same words.
	// Named words, not bits: the latter shadows the math/bits import this file
	// now uses for UintSize (gocritic importShadow).
	buf := make([]big.Word, words)
	var delta uint64
	for i := range buf {
		buf[i] = big.Word(0xAAAAAAAAAAAAAAAA ^ delta)
		delta += 0x1234567
	}
	z := new(big.Int)
	z.SetBits(buf)
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
//
// M-01: confidence is now earned, on two axes, from zero.
//
// The old score was a flat 0.5 baseline plus a flat +0.2 for having found a
// crossover at all. Two consecutive runs could therefore persist thresholds a
// factor of four apart (115200 and 460800 bits were both observed) while both
// reported 0.70 — and, worse, the 0.5 baseline equalled the escalation bar
// exactly, so no result was ever weak enough to fall through to the full
// sweep.
//
// The score now starts at zero and is the product of:
//   - stability: how repeatable the individual iterations were;
//   - decisiveness: how far past the acceptance margin the winning size was.
//
// Both matter. A crossover found on stable timings that only just clears the
// margin is exactly the flapping case: the smallest qualifying size decides
// the answer, so a boundary size flipping sides moves the result by 4x. Such a
// run now scores below the bar and escalates to CompleteStrategy, which is
// what --auto-calibrate asked for.
func (mb *MicroBenchmark) analyzeResults(results []testResult) ThresholdResults {
	tr := ThresholdResults{
		// Conservative defaults, used only when no measurement says otherwise.
		// Confidence starts at ZERO, not 0.5 (audit M-01): the old baseline was
		// numerically equal to EscalationConfidenceThreshold, so
		// tryFastThenEscalate's `conf < threshold` test could never fire on a
		// run that produced any result at all. The fast pass was therefore
		// always accepted and persisted, however marginal, and CompleteStrategy
		// was unreachable except on total failure. Defaults are not a
		// measurement and must not carry the confidence of one.
		FFTThreshold:      500000,
		ParallelThreshold: 4096,
		Confidence:        0.0,
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

	// Analyze FFT crossover point; only a real measurement earns confidence,
	// and only as much as its stability AND its decisiveness justify.
	if fftCrossover, decisiveness := mb.findFFTCrossover(bySize); fftCrossover > 0 {
		tr.FFTThreshold = fftCrossover
		tr.Confidence = timingStability(results) * decisiveness
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

// timingStability scores how repeatable the timings were, in [0, 1].
//
// Each successful test brackets its iterations with a fastest and a slowest
// sample; their ratio is that test's spread. 1.0 means every iteration took
// the same time, 0.0 means at least one test's slowest iteration took twice
// its fastest or worse. The worst test in the batch decides, because a
// crossover is a comparison and one unstable arm is enough to invalidate it.
func timingStability(results []testResult) float64 {
	worst := 1.0
	seen := false
	for _, r := range results {
		if r.err != nil || r.fastest <= 0 {
			continue
		}
		seen = true
		if ratio := float64(r.slowest) / float64(r.fastest); ratio > worst {
			worst = ratio
		}
	}
	if !seen {
		return 0
	}
	// worst == 1 -> 1.0 (perfectly stable); worst >= 2 -> 0.0.
	stability := 2 - worst
	if stability < 0 {
		return 0
	}
	if stability > 1 {
		return 1
	}
	return stability
}

const (
	// fftCrossoverMargin is how much faster FFT must be before a size counts
	// as a crossover. Without it (audit M-01) a one-percent difference — well
	// inside the spread these short runs show — was enough to move the
	// persisted threshold by a factor of four between consecutive startups.
	// findParallelCrossover has always required the same 10%.
	fftCrossoverMargin = 0.9

	// decisiveSpeedup is the speedup at which a crossover is considered
	// unambiguous. Between the margin (1/0.9 ≈ 1.11x) and this value, the
	// decisiveness score ramps from 0 to 1.
	//
	// It exists because the margin alone does not settle the flapping (M-01):
	// findFFTCrossover reports the SMALLEST size where FFT wins, so a size
	// sitting right at the boundary — 2000 words is one, just past bigfft's
	// own 1800-word threshold — flips the answer by a factor of four
	// depending on which side of the margin a given run lands. A win that
	// barely clears the bar should not be persisted with high confidence; it
	// should send AutoCalibrateWithProfile on to the full sweep.
	decisiveSpeedup = 2.0
)

// findFFTCrossover determines the bit size where FFT becomes clearly faster
// than standard math/big. Returns 0 when no crossover was observed in the
// measured data — analyzeResults owns the conservative default for that
// case (FIB-03: a fallback value is not a measurement and must not be
// treated as one for confidence purposes).
//
// Sizes at or below bigfft.FFTThresholdWords never reach this function with
// both arms present: runTests does not emit an FFT configuration for them,
// because bigfft would run the same math/big code for both (M-01).
func (mb *MicroBenchmark) findFFTCrossover(bySize map[int][]testResult) (bitSize int, decisiveness float64) {
	// Speedup (standard / FFT) at each size that has both arms.
	ratios := make(map[int]float64, len(bySize))
	sizes := make([]int, 0, len(bySize))
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

		if stdCount == 0 || fftCount == 0 {
			continue
		}
		avgFFT := fftDur / time.Duration(fftCount)
		if avgFFT <= 0 {
			continue
		}
		ratios[size] = float64(stdDur/time.Duration(stdCount)) / float64(avgFFT)
		sizes = append(sizes, size)
	}
	if len(sizes) == 0 {
		return 0, 0
	}
	sort.Ints(sizes)

	// A crossover is a MONOTONE transition: past it, FFT should stay ahead.
	// Requiring every larger measured size to win too is what stops the
	// flapping (audit M-01). Taking merely the smallest winning size let a
	// boundary size — 2000 words sits just past bigfft's own 1800-word
	// threshold — decide the answer whenever it happened to clear the margin,
	// moving the persisted threshold by a factor of four between consecutive
	// startups even though nothing about the machine had changed.
	const minRatio = 1 / fftCrossoverMargin
	crossoverIdx := -1
	for i := len(sizes) - 1; i >= 0; i-- {
		if ratios[sizes[i]] <= minRatio {
			break
		}
		crossoverIdx = i
	}
	if crossoverIdx < 0 {
		return 0, 0
	}

	// The weakest link in the winning suffix sets the decisiveness: the
	// transition is only as convincing as its least convincing size.
	weakest := ratios[sizes[crossoverIdx]]
	for _, size := range sizes[crossoverIdx:] {
		if ratios[size] < weakest {
			weakest = ratios[size]
		}
	}

	// Add some margin (FFT should be clearly better).
	return sizes[crossoverIdx] * bits.UintSize * 9 / 10, decisivenessOf(weakest)
}

// decisivenessOf maps the speedup at the deciding size onto [0, 1]: 0 at the
// acceptance margin, 1 once FFT is decisiveSpeedup times faster.
func decisivenessOf(ratio float64) float64 {
	const minRatio = 1 / fftCrossoverMargin
	if ratio <= minRatio {
		return 0
	}
	d := (ratio - minRatio) / (decisiveSpeedup - minRatio)
	if d > 1 {
		return 1
	}
	return d
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
				bitSize := size * bits.UintSize
				if crossoverSize == 0 || bitSize < crossoverSize {
					crossoverSize = bitSize
				}
			}
		}
	}

	return crossoverSize
}
