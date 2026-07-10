package calibration

import (
	"context"
	"runtime"
	"slices"
	"testing"

	"github.com/agbruneau/FibGo/internal/fibonacci"
)

func TestGenerateParallelThresholds(t *testing.T) {
	t.Parallel()

	// Table-driven over the injected CPU count so every CPU-gated branch is
	// exercised on any development machine (runtime.NumCPU ignores
	// GOMAXPROCS, so the branches below 8 CPUs were dead on real hardware).
	// FIB-02: the baseline candidate must be -1 (genuinely sequential —
	// normalizeOptions only replaces ==0 with the default, so 0 was a
	// duplicate of the default candidate, never a real baseline).
	cases := []struct {
		numCPU int
		want   []int
	}{
		{1, []int{-1}},
		{2, []int{-1, 512, 1024, 2048, 4096}},
		{4, []int{-1, 512, 1024, 2048, 4096}},
		{8, []int{-1, 256, 512, 1024, 2048, 4096, 8192}},
		{16, []int{-1, 256, 512, 1024, 2048, 4096, 8192, 16384}},
		{24, []int{-1, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768}},
	}
	for _, tc := range cases {
		if got := generateParallelThresholds(tc.numCPU); !slices.Equal(got, tc.want) {
			t.Errorf("generateParallelThresholds(%d) = %v, want %v", tc.numCPU, got, tc.want)
		}
	}

	// The exported wrapper must delegate to the current machine's CPU count.
	if got, want := GenerateParallelThresholds(), generateParallelThresholds(runtime.NumCPU()); !slices.Equal(got, want) {
		t.Errorf("GenerateParallelThresholds() = %v, want %v", got, want)
	}
}

func TestGenerateQuickParallelThresholds(t *testing.T) {
	t.Parallel()
	thresholds := GenerateQuickParallelThresholds()

	// Should be shorter than full list
	fullThresholds := GenerateParallelThresholds()
	if len(thresholds) > len(fullThresholds) {
		t.Error("Quick thresholds should not be longer than full thresholds")
	}

	// Should have at least one threshold
	if len(thresholds) < 1 {
		t.Error("Expected at least one threshold")
	}

	// Verify thresholds are appropriate for CPU count
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

	t.Logf("Generated %d quick parallel thresholds: %v", len(thresholds), thresholds)
}

func TestGenerateQuickFFTThresholds(t *testing.T) {
	t.Parallel()
	thresholds := generateQuickFFTThresholds()

	if len(thresholds) < 2 {
		t.Error("Expected multiple quick FFT thresholds")
	}

	t.Logf("Generated %d quick FFT thresholds: %v", len(thresholds), thresholds)
}

func TestGenerateQuickStrassenThresholds(t *testing.T) {
	t.Parallel()
	thresholds := GenerateQuickStrassenThresholds()

	if len(thresholds) < 2 {
		t.Error("Expected multiple quick Strassen thresholds")
	}

	t.Logf("Generated %d quick Strassen thresholds: %v", len(thresholds), thresholds)
}

// TestBaselineCandidateIsGenuinelySequential is the FIB-02 red test: the
// "Sequential (no parallelism)" candidate emitted by the generators must
// survive fibonacci.normalizeOptions as a real baseline. normalizeOptions
// treats ==0 as "unset" and substitutes the package defaults, so a 0
// candidate silently becomes a duplicate of the default run — only a
// negative threshold reliably disables both parallelism (useParallel :=
// ... && ParallelThreshold > 0) and FFT (FFTThreshold > 0 && ...). We spy
// on the effective behavior via the real "fast" calculator: F(64) is tiny
// enough that a -1 threshold and the (previous, buggy) 0 threshold would
// both compute correctly, so this asserts the generator output directly,
// which is what actually reaches Calculate.
func TestBaselineCandidateIsGenuinelySequential(t *testing.T) {
	t.Parallel()

	baseline := GenerateParallelThresholds()[0]
	if baseline >= 0 {
		t.Fatalf("baseline parallel candidate = %d, want negative (0 duplicates the default via normalizeOptions)", baseline)
	}

	fftBaseline := GenerateFFTThresholds()[0]
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

// Benchmark threshold generation
func BenchmarkGenerateParallelThresholds(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = GenerateParallelThresholds()
	}
}
