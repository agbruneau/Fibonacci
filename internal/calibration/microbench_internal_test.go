// This file stays package calibration (white-box) because it exercises
// analyzeResults, testResult, runSingleTest and generateTestNumber
// directly -- none of these are exported.
package calibration

import (
	"context"
	"testing"
)

func TestMicroBenchAnalyzeResultsEmpty(t *testing.T) {
	t.Parallel()
	mb := NewMicroBenchmark()
	results := mb.analyzeResults(nil)
	if results.Confidence != 0.0 {
		t.Errorf("Expected 0.0 confidence for empty results, got %f", results.Confidence)
	}
}

// TestMicroBenchAnalyzeResultsAllErrored is the FIB-03 red test: when every
// timed result errored, bySize ends up empty (no valid measurement at
// all), yet analyzeResults must not report a confident result. Before the
// fix it started at 0.5 and still added the FFT/parallel "found a
// crossover" bonuses purely from the >0 defaults, landing at 0.9 despite
// zero real data.
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

// TestRunSingleTestRespectsCancellationBeforeWarmup verifies the early ctx
// guard added before the warm-up multiply: a pre-canceled context returns
// the cancellation error rather than running a (potentially expensive)
// warm-up.
func TestRunSingleTestRespectsCancellationBeforeWarmup(t *testing.T) {
	t.Parallel()
	mb := NewMicroBenchmark()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	d, err := mb.runSingleTest(ctx, 1024, false, false)
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled before warm-up, got d=%v err=%v", d, err)
	}
}

// TestGenerateTestNumber checks that generateTestNumber builds a *big.Int
// whose word count never exceeds the requested size (SetBits trims
// trailing zero words, so it may be smaller but never larger) and is
// non-empty for a positive word count.
func TestGenerateTestNumber(t *testing.T) {
	t.Parallel()
	const words = 10
	num := generateTestNumber(words)
	if num == nil {
		t.Fatal("Expected non-nil big.Int")
	}
	if got := len(num.Bits()); got == 0 || got > words {
		t.Errorf("len(num.Bits()) = %d, want in (0, %d]", got, words)
	}
}
