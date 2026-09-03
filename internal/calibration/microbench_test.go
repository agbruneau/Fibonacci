package calibration

import (
	"context"
	"math/bits"
	"testing"
	"time"

	"github.com/agbruneau/FibGo/internal/bigfft"
)

func TestNewMicroBenchmark(t *testing.T) {
	t.Parallel()
	mb := NewMicroBenchmark()
	if mb == nil {
		t.Fatal("Expected non-nil MicroBenchmark")
	}
	if len(mb.TestSizes) == 0 {
		t.Error("Expected default test sizes")
	}
	if mb.Iterations <= 0 {
		t.Error("Expected positive iterations")
	}
	if mb.Timeout <= 0 {
		t.Error("Expected positive timeout")
	}
}

func TestMicroBenchRunQuick(t *testing.T) {
	mb := NewMicroBenchmark()
	// Use very small sizes and iterations for fast test
	mb.TestSizes = []int{100, 200}
	mb.Iterations = 10
	mb.Timeout = 2 * time.Second

	// Ensure MinBitLen doesn't skip FFT tests
	bigfft.SetTransformCacheConfig(bigfft.TransformCacheConfig{
		MaxEntries: 10,
		MinBitLen:  0,
		Enabled:    true,
	})
	defer bigfft.SetTransformCacheConfig(bigfft.DefaultTransformCacheConfig())

	ctx := context.Background()
	results, err := mb.RunQuick(ctx)
	if err != nil {
		t.Fatalf("RunQuick failed: %v", err)
	}

	t.Logf("MicroBench Results: FFT=%d, Par=%d, Conf=%f, Dur=%v",
		results.FFTThreshold, results.ParallelThreshold, results.Confidence, results.Duration)

	if results.FFTThreshold <= 0 {
		t.Errorf("Expected positive FFT threshold, got %d", results.FFTThreshold)
	}
	if results.ParallelThreshold < 0 {
		t.Errorf("Expected non-negative parallel threshold, got %d", results.ParallelThreshold)
	}
	if results.Duration < 0 {
		t.Error("Expected non-negative duration")
	}
}

func TestQuickCalibrate(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	results, err := NewMicroBenchmark().RunQuick(ctx)
	if err != nil {
		t.Fatalf("QuickCalibrate failed: %v", err)
	}

	if results.Confidence < 0 || results.Confidence > 1.0 {
		t.Errorf("Invalid confidence score: %f", results.Confidence)
	}
}

func TestMicroBenchAnalyzeResultsEmpty(t *testing.T) {
	t.Parallel()
	mb := NewMicroBenchmark()
	results := mb.analyzeResults(nil)
	if results.Confidence != 0.0 {
		t.Errorf("Expected 0.0 confidence for empty results, got %f", results.Confidence)
	}
}

// TestMicroBenchAnalyzeResultsAllErrored is the FIB-03 red test: when every
// timed result errored, bySize ends up empty (no valid measurement at all),
// yet analyzeResults must not report a confident result. Today it starts at
// 0.5 and still adds the FFT/parallel "found a crossover" bonuses purely
// from the >0 defaults, landing at 0.9 despite zero real data.
func TestMicroBenchAnalyzeResultsAllErrored(t *testing.T) {
	t.Parallel()
	mb := NewMicroBenchmark()
	results := []testResult{
		{wordSize: 500, useFFT: false, parallel: false, err: context.DeadlineExceeded},
		{wordSize: 500, useFFT: true, parallel: false, err: context.DeadlineExceeded},
	}
	tr := mb.analyzeResults(results)
	if tr.Confidence >= 0.5 {
		t.Errorf("Confidence = %f with zero valid measurements, want < 0.5", tr.Confidence)
	}
}

func TestMicroBenchContextCancellation(t *testing.T) {
	t.Parallel()
	mb := NewMicroBenchmark()
	mb.Iterations = 100 // Many iterations to ensure it takes some time

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	results, err := mb.RunQuick(ctx)
	// FIB-03: RunQuick must propagate the cancellation instead of silently
	// returning a "successful" zero-confidence result as if nothing went
	// wrong.
	if err == nil {
		t.Error("RunQuick should propagate the context error when no measurement could be collected")
	}
	if results.Confidence != 0 {
		t.Errorf("Confidence = %f, want 0 when RunQuick errors out", results.Confidence)
	}
}

// TestRunSingleTestRespectsCancellationBeforeWarmup verifies the early ctx
// guard added before the warm-up multiply: a pre-canceled context returns the
// cancellation error rather than running a (potentially expensive) warm-up.
func TestRunSingleTestRespectsCancellationBeforeWarmup(t *testing.T) {
	t.Parallel()
	mb := NewMicroBenchmark()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	d, _, _, err := mb.runSingleTest(ctx, 1024, false, false)
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled before warm-up, got d=%v err=%v", d, err)
	}
}

func TestGenerateTestNumber(t *testing.T) {
	t.Parallel()
	words := 10
	num := generateTestNumber(words)
	if num == nil {
		t.Fatal("Expected non-nil big.Int")
	}
	// For words = 10 the most significant word (0xAAAA… ^ 9*0x1234567) is
	// non-zero on both 32- and 64-bit big.Word, so SetBits cannot trim it.
	// The pattern is NOT zero-free in general: 0x1234567 is odd, hence
	// invertible mod 2^W, so exactly one index per 2^W zeroes its word
	// (i = 3549559750 for W = 32) — far above any count used here.
	if len(num.Bits()) != words {
		t.Errorf("len(num.Bits()) = %d, want %d", len(num.Bits()), words)
	}
}

// mkResults builds a synthetic bySize map: for each size, one standard-arm and
// one FFT-arm result whose mean durations are in the requested ratio
// (std / fft). A ratio above 1 means FFT is faster at that size.
func mkResults(sizes map[int]float64) map[int][]testResult {
	const base = 1000 * time.Microsecond
	out := make(map[int][]testResult, len(sizes))
	for size, ratio := range sizes {
		fft := time.Duration(float64(base) / ratio)
		out[size] = []testResult{
			{wordSize: size, useFFT: false, duration: base, fastest: base, slowest: base},
			{wordSize: size, useFFT: true, duration: fft, fastest: fft, slowest: fft},
		}
	}
	return out
}

// TestFindFFTCrossoverRequiresMargin pins the first half of audit M-01: a
// difference too small to distinguish from noise is not a crossover. Without
// the margin a 1% edge at a boundary size moved the persisted threshold by a
// factor of four between consecutive startups.
func TestFindFFTCrossoverRequiresMargin(t *testing.T) {
	t.Parallel()
	mb := NewMicroBenchmark()

	t.Run("a 1 percent edge is not a crossover", func(t *testing.T) {
		t.Parallel()
		got, dec := mb.findFFTCrossover(mkResults(map[int]float64{2000: 1.01, 8000: 1.01}))
		if got != 0 {
			t.Errorf("crossover = %d, want 0: 1%% is inside the noise", got)
		}
		if dec != 0 {
			t.Errorf("decisiveness = %v, want 0", dec)
		}
	})

	t.Run("a clear win is a crossover", func(t *testing.T) {
		t.Parallel()
		got, dec := mb.findFFTCrossover(mkResults(map[int]float64{2000: 3.0, 8000: 3.0}))
		if got == 0 {
			t.Fatal("a 3x win must register as a crossover")
		}
		if dec != 1 {
			t.Errorf("decisiveness = %v, want 1 for a 3x win", dec)
		}
	})
}

// TestFindFFTCrossoverIsMonotone pins the second half of audit M-01: a
// crossover is a transition, so every size above it must win too. Taking
// merely the smallest winning size let a boundary size decide the answer
// whenever noise pushed it over the margin.
func TestFindFFTCrossoverIsMonotone(t *testing.T) {
	t.Parallel()
	mb := NewMicroBenchmark()

	// 2000 wins, but 8000 does not: that is not a transition, it is noise at
	// 2000. The crossover must be reported above it, not at it.
	got, _ := mb.findFFTCrossover(mkResults(map[int]float64{2000: 2.0, 8000: 1.0, 16000: 2.0}))
	want := 16000 * bits.UintSize * 9 / 10
	if got != want {
		t.Errorf("crossover = %d, want %d (the smallest size whose whole suffix wins)", got, want)
	}
}

// TestFindFFTCrossoverDecisivenessIsWeakestLink checks that a transition is
// only as convincing as its least convincing size.
func TestFindFFTCrossoverDecisivenessIsWeakestLink(t *testing.T) {
	t.Parallel()
	mb := NewMicroBenchmark()

	_, dec := mb.findFFTCrossover(mkResults(map[int]float64{2000: 5.0, 8000: 1.15}))
	if dec >= 0.2 {
		t.Errorf("decisiveness = %v, want a low score: 8000 barely clears the margin", dec)
	}
}

// TestAnalyzeResultsConfidenceStartsAtZero pins the escalation fix of audit
// M-01. The confidence baseline used to be 0.5, numerically equal to
// EscalationConfidenceThreshold, so tryFastThenEscalate's `conf < threshold`
// test could never fire on a run that produced any result at all: the fast
// pass was always accepted and persisted, and CompleteStrategy was
// unreachable except on total failure.
func TestAnalyzeResultsConfidenceStartsAtZero(t *testing.T) {
	t.Parallel()
	mb := NewMicroBenchmark()

	// Valid measurements, but no size clears the margin: nothing was learned,
	// so nothing is claimed.
	var flat []testResult
	for size, rs := range mkResults(map[int]float64{2000: 1.0, 8000: 1.0}) {
		_ = size
		flat = append(flat, rs...)
	}
	tr := mb.analyzeResults(flat)

	if tr.Confidence != 0 {
		t.Errorf("Confidence = %v, want 0 when no crossover was measured", tr.Confidence)
	}
	if Confidence(tr.Confidence) >= EscalationConfidenceThreshold {
		t.Errorf("Confidence %v must stay below the escalation bar %v so the full sweep is reachable",
			tr.Confidence, EscalationConfidenceThreshold)
	}
}

// TestTimingStability scores the dispersion the confidence rests on.
func TestTimingStability(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		results []testResult
		want    float64
	}{
		{"perfectly stable", []testResult{{fastest: 100, slowest: 100}}, 1},
		{"slowest is 1.5x the fastest", []testResult{{fastest: 100, slowest: 150}}, 0.5},
		{"slowest is 2x the fastest", []testResult{{fastest: 100, slowest: 200}}, 0},
		{"beyond 2x floors at zero", []testResult{{fastest: 100, slowest: 900}}, 0},
		{"the worst arm decides", []testResult{{fastest: 100, slowest: 100}, {fastest: 100, slowest: 200}}, 0},
		{"no usable sample", []testResult{{err: context.Canceled}}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := timingStability(tt.results); got != tt.want {
				t.Errorf("timingStability = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRunTestsSkipsIdenticalWorkloads pins the third part of audit M-01: below
// bigfft's own threshold, useFFT true and false run the same math/big code, so
// emitting both arms produced a comparison of a workload against itself.
func TestRunTestsSkipsIdenticalWorkloads(t *testing.T) {
	t.Parallel()

	threshold := bigfft.FFTThresholdWords()
	mb := NewMicroBenchmark()
	mb.TestSizes = []int{threshold / 2, threshold * 4}
	mb.Iterations = 1

	results := mb.runTests(context.Background())

	for _, r := range results {
		if r.useFFT && r.wordSize <= threshold {
			t.Errorf("size %d is at or below the %d-word FFT threshold; timing an 'FFT' arm there compares math/big with itself",
				r.wordSize, threshold)
		}
	}
	// The large size must still get both arms, otherwise nothing is comparable.
	var large int
	for _, r := range results {
		if r.wordSize == threshold*4 {
			large++
		}
	}
	if large != 2 {
		t.Errorf("the above-threshold size got %d arms, want 2", large)
	}
}
