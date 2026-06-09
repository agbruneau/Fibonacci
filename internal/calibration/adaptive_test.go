package calibration

import (
	"runtime"
	"testing"
)

func TestGenerateParallelThresholds(t *testing.T) {
	t.Parallel()
	thresholds := GenerateParallelThresholds()

	// Should always include sequential (0)
	if len(thresholds) == 0 || thresholds[0] != 0 {
		t.Error("Expected thresholds to start with 0 (sequential)")
	}

	// Should have at least one threshold
	if len(thresholds) < 1 {
		t.Error("Expected at least one threshold")
	}

	// Thresholds should be non-negative
	for i, th := range thresholds {
		if th < 0 {
			t.Errorf("Threshold at index %d is negative: %d", i, th)
		}
	}

	// Verify thresholds are appropriate for CPU count
	numCPU := runtime.NumCPU()
	switch {
	case numCPU == 1:
		if len(thresholds) != 1 {
			t.Errorf("For 1 CPU, expected 1 threshold, got %d", len(thresholds))
		}
	case numCPU <= 4:
		if len(thresholds) < 5 {
			t.Errorf("For %d CPUs, expected at least 5 thresholds, got %d", numCPU, len(thresholds))
		}
		// Should include: 0, 512, 1024, 2048, 4096
		expected := []int{0, 512, 1024, 2048, 4096}
		for _, exp := range expected {
			found := false
			for _, th := range thresholds {
				if th == exp {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected threshold %d not found in %v", exp, thresholds)
			}
		}
	case numCPU <= 8:
		if len(thresholds) < 7 {
			t.Errorf("For %d CPUs, expected at least 7 thresholds, got %d", numCPU, len(thresholds))
		}
	case numCPU <= 16:
		if len(thresholds) < 8 {
			t.Errorf("For %d CPUs, expected at least 8 thresholds, got %d", numCPU, len(thresholds))
		}
	default:
		if len(thresholds) < 9 {
			t.Errorf("For %d CPUs, expected at least 9 thresholds, got %d", numCPU, len(thresholds))
		}
	}

	// Log the thresholds for visibility
	t.Logf("Generated %d parallel thresholds for %d CPUs: %v",
		len(thresholds), numCPU, thresholds)
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
		if len(thresholds) != 1 || thresholds[0] != 0 {
			t.Errorf("For 1 CPU, expected [0], got %v", thresholds)
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
	thresholds := GenerateQuickFFTThresholds()

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

// Benchmark threshold generation
func BenchmarkGenerateParallelThresholds(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = GenerateParallelThresholds()
	}
}
