// Package calibration_test exercises internal/calibration's exported API
// only. It is the black-box counterpart to microbench_internal_test.go,
// which stays package calibration because it calls analyzeResults,
// runSingleTest and generateTestNumber directly.
package calibration_test

import (
	"context"
	"testing"
	"time"

	"github.com/agbruneau/FibGo/internal/bigfft"
	"github.com/agbruneau/FibGo/internal/calibration"
)

func TestNewMicroBenchmark(t *testing.T) {
	t.Parallel()
	mb := calibration.NewMicroBenchmark()
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

// TestMicroBenchRunQuick is NOT parallel: it calls
// bigfft.SetTransformCacheConfig, whose own doc comment says it "should be
// called before any FFT operations for consistent behavior" -- this
// package's many other tests perform FFT work (via QuickCalibrate,
// CompleteStrategy, etc.) and would observe the temporarily-mutated global
// config if this test ran concurrently with them. SetTransformCacheConfig
// itself is data-race-free (guarded by an internal mutex), but a
// same-package mutex here would only serialize participants that opt in --
// no other test in this package touches the same global, so the correct
// fix is keeping this one test in the sequential phase rather than adding
// protection with nothing else to coordinate with.
func TestMicroBenchRunQuick(t *testing.T) {
	mb := calibration.NewMicroBenchmark()
	// Use very small sizes and iterations for a fast test.
	mb.TestSizes = []int{100, 200}
	mb.Iterations = 10
	mb.Timeout = 2 * time.Second

	// Ensure MinBitLen doesn't skip FFT tests.
	bigfft.SetTransformCacheConfig(bigfft.TransformCacheConfig{
		MaxEntries: 10,
		MinBitLen:  0,
		Enabled:    true,
	})
	defer bigfft.SetTransformCacheConfig(bigfft.DefaultTransformCacheConfig())

	results, err := mb.RunQuick(context.Background())
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

	results, err := calibration.QuickCalibrate(ctx)
	if err != nil {
		t.Fatalf("QuickCalibrate failed: %v", err)
	}
	if results.Confidence < 0 || results.Confidence > 1.0 {
		t.Errorf("Invalid confidence score: %f", results.Confidence)
	}
}

func TestMicroBenchContextCancellation(t *testing.T) {
	t.Parallel()
	mb := calibration.NewMicroBenchmark()
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
