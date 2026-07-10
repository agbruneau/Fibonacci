// Package calibration_test exercises internal/calibration's exported API
// only. It is the black-box counterpart to adaptive_internal_test.go,
// which stays package calibration because it calls the unexported
// generateParallelThresholds/generateQuickFFTThresholds seams directly.
package calibration_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/agbruneau/FibGo/internal/calibration"
	"github.com/agbruneau/FibGo/internal/fibonacci"
)

func TestGenerateQuickParallelThresholds(t *testing.T) {
	t.Parallel()
	thresholds := calibration.GenerateQuickParallelThresholds()
	fullThresholds := calibration.GenerateParallelThresholds()

	if len(thresholds) > len(fullThresholds) {
		t.Error("Quick thresholds should not be longer than full thresholds")
	}
	if len(thresholds) < 1 {
		t.Error("Expected at least one threshold")
	}
	// The baseline candidate must always be the genuine sequential sentinel
	// (FIB-02): 0 would silently duplicate the package default via
	// fibonacci.normalizeOptions.
	if thresholds[0] >= 0 {
		t.Errorf("GenerateQuickParallelThresholds()[0] = %d, want negative baseline", thresholds[0])
	}

	// GenerateQuickParallelThresholds calls runtime.NumCPU() directly (no
	// injected seam, unlike GenerateParallelThresholds/P-06), so only the
	// branch matching this machine's core count is exercised here -- exact
	// per-tier length matches the production switch in adaptive.go.
	numCPU := runtime.NumCPU()
	switch {
	case numCPU == 1:
		if len(thresholds) != 1 || thresholds[0] != -1 {
			t.Errorf("For 1 CPU, expected [-1], got %v", thresholds)
		}
	case numCPU <= 4:
		if len(thresholds) != 3 {
			t.Errorf("For %d CPUs, expected 3 thresholds, got %d", numCPU, len(thresholds))
		}
	case numCPU <= 8:
		if len(thresholds) != 4 {
			t.Errorf("For %d CPUs, expected 4 thresholds, got %d", numCPU, len(thresholds))
		}
	default:
		if len(thresholds) != 5 {
			t.Errorf("For %d CPUs, expected 5 thresholds, got %d", numCPU, len(thresholds))
		}
	}
}

func TestGenerateQuickStrassenThresholds(t *testing.T) {
	t.Parallel()
	thresholds := calibration.GenerateQuickStrassenThresholds()
	if len(thresholds) < 2 {
		t.Error("Expected multiple quick Strassen thresholds")
	}
}

// TestBaselineCandidateIsGenuinelySequential is the FIB-02 red test: the
// "Sequential (no parallelism)" candidate emitted by the generators must
// survive fibonacci.normalizeOptions as a real baseline. normalizeOptions
// treats ==0 as "unset" and substitutes the package defaults, so a 0
// candidate silently becomes a duplicate of the default run -- only a
// negative threshold reliably disables both parallelism (useParallel :=
// ... && ParallelThreshold > 0) and FFT (FFTThreshold > 0 && ...). We spy
// on the effective behavior via the real "fast" calculator: F(64) is tiny
// enough that a -1 threshold and the (previous, buggy) 0 threshold would
// both compute correctly, so this asserts the generator output directly,
// which is what actually reaches Calculate.
func TestBaselineCandidateIsGenuinelySequential(t *testing.T) {
	t.Parallel()

	baseline := calibration.GenerateParallelThresholds()[0]
	if baseline >= 0 {
		t.Fatalf("baseline parallel candidate = %d, want negative (0 duplicates the default via normalizeOptions)", baseline)
	}

	fftBaseline := calibration.GenerateFFTThresholds()[0]
	if fftBaseline >= 0 {
		t.Fatalf("baseline FFT candidate = %d, want negative (0 duplicates the default via normalizeOptions)", fftBaseline)
	}

	// Confirm the negative candidate is actually honored end-to-end by the
	// real calculator (not just by the generator): a tiny calculation must
	// still succeed with the baseline options.
	calc := fibonacci.MustNewCalculator(&fibonacci.FastDoublingCalculator{})
	if _, err := calc.Calculate(context.Background(), nil, 0, 64, fibonacci.Options{ParallelThreshold: baseline, FFTThreshold: fftBaseline}); err != nil {
		t.Fatalf("Calculate with baseline options failed: %v", err)
	}
}

func BenchmarkGenerateParallelThresholds(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = calibration.GenerateParallelThresholds()
	}
}
