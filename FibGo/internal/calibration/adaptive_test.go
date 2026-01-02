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
		// Should include: 0, 2048, 4096
		expected := []int{0, 2048, 4096}
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
		if len(thresholds) != 4 {
			t.Errorf("For %d CPUs, expected 4 thresholds, got %d", numCPU, len(thresholds))
		}
		// Should include: 0, 2048, 4096, 8192
		expected := []int{0, 2048, 4096, 8192}
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
	default:
		if len(thresholds) != 5 {
			t.Errorf("For %d CPUs, expected 5 thresholds, got %d", numCPU, len(thresholds))
		}
		// Should include: 0, 2048, 4096, 8192, 16384
		expected := []int{0, 2048, 4096, 8192, 16384}
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
	}

	t.Logf("Generated %d quick parallel thresholds: %v", len(thresholds), thresholds)
}

func TestGenerateFFTThresholds(t *testing.T) {
	t.Parallel()
	thresholds := GenerateFFTThresholds()

	// Should include 0 (disabled)
	if len(thresholds) == 0 || thresholds[0] != 0 {
		t.Error("Expected FFT thresholds to start with 0 (disabled)")
	}

	// Should have multiple options
	if len(thresholds) < 2 {
		t.Error("Expected multiple FFT thresholds")
	}

	// Thresholds should be in ascending order (after 0)
	for i := 2; i < len(thresholds); i++ {
		if thresholds[i] < thresholds[i-1] {
			t.Errorf("FFT thresholds not in ascending order at index %d", i)
		}
	}

	t.Logf("Generated %d FFT thresholds: %v", len(thresholds), thresholds)
}

func TestGenerateQuickFFTThresholds(t *testing.T) {
	t.Parallel()
	thresholds := GenerateQuickFFTThresholds()

	if len(thresholds) < 2 {
		t.Error("Expected multiple quick FFT thresholds")
	}

	t.Logf("Generated %d quick FFT thresholds: %v", len(thresholds), thresholds)
}

func TestGenerateStrassenThresholds(t *testing.T) {
	t.Parallel()
	thresholds := GenerateStrassenThresholds()

	// Should include 0 (disabled)
	if len(thresholds) == 0 || thresholds[0] != 0 {
		t.Error("Expected Strassen thresholds to start with 0 (disabled)")
	}

	// Should have multiple options
	if len(thresholds) < 2 {
		t.Error("Expected multiple Strassen thresholds")
	}

	t.Logf("Generated %d Strassen thresholds: %v", len(thresholds), thresholds)
}

func TestGenerateQuickStrassenThresholds(t *testing.T) {
	t.Parallel()
	thresholds := GenerateQuickStrassenThresholds()

	if len(thresholds) < 2 {
		t.Error("Expected multiple quick Strassen thresholds")
	}

	t.Logf("Generated %d quick Strassen thresholds: %v", len(thresholds), thresholds)
}

func TestEstimateOptimalParallelThreshold(t *testing.T) {
	t.Parallel()
	threshold := EstimateOptimalParallelThreshold()

	// Should be non-negative
	if threshold < 0 {
		t.Errorf("Estimated parallel threshold is negative: %d", threshold)
	}

	// Should be in reasonable range
	if threshold > 65536 {
		t.Errorf("Estimated parallel threshold seems too high: %d", threshold)
	}

	// Test that it returns different values based on CPU count
	// The function uses runtime.NumCPU() internally, so we test the logic
	numCPU := runtime.NumCPU()
	threshold = EstimateOptimalParallelThreshold()

	// Verify threshold is appropriate for CPU count
	switch {
	case numCPU == 1:
		if threshold != 0 {
			t.Errorf("For 1 CPU, threshold should be 0, got %d", threshold)
		}
	case numCPU <= 2:
		if threshold != 8192 {
			t.Errorf("For %d CPUs, threshold should be 8192, got %d", numCPU, threshold)
		}
	case numCPU <= 4:
		if threshold != 4096 {
			t.Errorf("For %d CPUs, threshold should be 4096, got %d", numCPU, threshold)
		}
	case numCPU <= 8:
		if threshold != 2048 {
			t.Errorf("For %d CPUs, threshold should be 2048, got %d", numCPU, threshold)
		}
	case numCPU <= 16:
		if threshold != 1024 {
			t.Errorf("For %d CPUs, threshold should be 1024, got %d", numCPU, threshold)
		}
	default:
		if threshold != 512 {
			t.Errorf("For %d CPUs, threshold should be 512, got %d", numCPU, threshold)
		}
	}

	t.Logf("Estimated parallel threshold for %d CPUs: %d", numCPU, threshold)
}

func TestEstimateOptimalFFTThreshold(t *testing.T) {
	t.Parallel()
	threshold := EstimateOptimalFFTThreshold()

	// Should be positive
	if threshold <= 0 {
		t.Errorf("Estimated FFT threshold should be positive: %d", threshold)
	}

	// Should be in reasonable range
	if threshold > 10000000 {
		t.Errorf("Estimated FFT threshold seems too high: %d", threshold)
	}

	t.Logf("Estimated FFT threshold: %d", threshold)
}

func TestEstimateOptimalStrassenThreshold(t *testing.T) {
	t.Parallel()
	threshold := EstimateOptimalStrassenThreshold()

	// Should be positive
	if threshold <= 0 {
		t.Errorf("Estimated Strassen threshold should be positive: %d", threshold)
	}

	t.Logf("Estimated Strassen threshold: %d", threshold)
}

func TestValidateThresholds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		parallel     int
		fft          int
		strassen     int
		wantParallel int
		wantFFT      int
		wantStrassen int
	}{
		{"normal values", 4096, 1000000, 256, 4096, 1000000, 256},
		{"negative parallel", -100, 1000000, 256, 0, 1000000, 256},
		{"negative fft", 4096, -100, 256, 4096, 0, 256},
		{"negative strassen", 4096, 1000000, -100, 4096, 1000000, 0},
		{"too high parallel", 100000, 1000000, 256, 65536, 1000000, 256},
		{"too high fft", 4096, 50000000, 256, 4096, 10000000, 256},
		{"too high strassen", 4096, 1000000, 50000, 4096, 1000000, 10000},
		{"all zeros", 0, 0, 0, 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p, f, s := ValidateThresholds(tt.parallel, tt.fft, tt.strassen)
			if p != tt.wantParallel {
				t.Errorf("parallel = %d, want %d", p, tt.wantParallel)
			}
			if f != tt.wantFFT {
				t.Errorf("fft = %d, want %d", f, tt.wantFFT)
			}
			if s != tt.wantStrassen {
				t.Errorf("strassen = %d, want %d", s, tt.wantStrassen)
			}
		})
	}
}

func TestGenerateFullThresholdSet(t *testing.T) {
	t.Parallel()
	set := GenerateFullThresholdSet()

	if len(set.Parallel) == 0 {
		t.Error("Expected non-empty parallel thresholds")
	}
	if len(set.FFT) == 0 {
		t.Error("Expected non-empty FFT thresholds")
	}
	if len(set.Strassen) == 0 {
		t.Error("Expected non-empty Strassen thresholds")
	}

	t.Logf("Full threshold set: Parallel=%d, FFT=%d, Strassen=%d",
		len(set.Parallel), len(set.FFT), len(set.Strassen))
}

func TestGenerateQuickThresholdSet(t *testing.T) {
	t.Parallel()
	quick := GenerateQuickThresholdSet()
	full := GenerateFullThresholdSet()

	// Quick should generally be smaller or equal
	if len(quick.Parallel) > len(full.Parallel) {
		t.Error("Quick parallel thresholds should not exceed full")
	}

	t.Logf("Quick threshold set: Parallel=%d, FFT=%d, Strassen=%d",
		len(quick.Parallel), len(quick.FFT), len(quick.Strassen))
}

func TestEstimatedThresholds(t *testing.T) {
	t.Parallel()
	p, f, s := EstimatedThresholds()

	if p < 0 || f < 0 || s < 0 {
		t.Errorf("Estimated thresholds contain negative values: p=%d, f=%d, s=%d", p, f, s)
	}

	t.Logf("Estimated thresholds: parallel=%d, FFT=%d, Strassen=%d", p, f, s)
}

func TestThresholdSetSort(t *testing.T) {
	t.Parallel()
	set := ThresholdSet{
		Parallel: []int{4096, 256, 1024, 512},
		FFT:      []int{1000000, 500000, 2000000},
		Strassen: []int{512, 128, 256},
	}

	set.SortThresholds()

	// Check parallel is sorted
	for i := 1; i < len(set.Parallel); i++ {
		if set.Parallel[i] < set.Parallel[i-1] {
			t.Error("Parallel thresholds not sorted")
		}
	}

	// Check FFT is sorted
	for i := 1; i < len(set.FFT); i++ {
		if set.FFT[i] < set.FFT[i-1] {
			t.Error("FFT thresholds not sorted")
		}
	}

	// Check Strassen is sorted
	for i := 1; i < len(set.Strassen); i++ {
		if set.Strassen[i] < set.Strassen[i-1] {
			t.Error("Strassen thresholds not sorted")
		}
	}
}

// Benchmark threshold generation
func BenchmarkGenerateParallelThresholds(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = GenerateParallelThresholds()
	}
}

func BenchmarkGenerateFullThresholdSet(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = GenerateFullThresholdSet()
	}
}

func BenchmarkEstimatedThresholds(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _ = EstimatedThresholds()
	}
}
