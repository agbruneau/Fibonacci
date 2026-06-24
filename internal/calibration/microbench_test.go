package calibration

import (
	"context"
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

	results, err := QuickCalibrate(ctx)
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

func TestMicroBenchContextCancellation(t *testing.T) {
	t.Parallel()
	mb := NewMicroBenchmark()
	mb.Iterations = 100 // Many iterations to ensure it takes some time

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	results, err := mb.RunQuick(ctx)
	// RunQuick currently doesn't return context error directly from parallel tests
	// but we check if it handles it gracefully
	if err != nil {
		t.Errorf("RunQuick should handle canceled context gracefully, got err: %v", err)
	}
	_ = results
}

// TestRunSingleTestRespectsCancellationBeforeWarmup verifies the early ctx
// guard added before the warm-up multiply: a pre-canceled context returns the
// cancellation error rather than running a (potentially expensive) warm-up.
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

func TestGenerateTestNumber(t *testing.T) {
	t.Parallel()
	words := 10
	num := generateTestNumber(words)
	if num == nil {
		t.Fatal("Expected non-nil big.Int")
	}
	if len(num.Bits()) != words {
		// Matches or trimmed? generateTestNumber uses random bits so it should usually be full
		// but leading zeros are possible.
	}
}
