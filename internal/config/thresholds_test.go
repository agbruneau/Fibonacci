// This file covers thresholds.go's private surface: the *ForHeuristic
// functions take a simulated HardwareHeuristic to exercise CPU/SIMD branches
// the real host may not have, and the two "wrapper agrees with the private
// impl" cross-checks below need direct access to that private impl. Tests
// that only exercise the public ApplyAdaptiveThresholds/EstimateOptimal*
// surface without needing a simulated host live in the black-box
// hardware_test.go instead.

package config

import "testing"

func TestEstimateParallelThresholdForHeuristic(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		h    HardwareHeuristic
		want int
	}{
		{"1cpu", HardwareHeuristic{NumCPU: 1, GOARCH: "amd64", SIMD: SIMDNone}, 0},
		{"2cpu", HardwareHeuristic{NumCPU: 2, GOARCH: "amd64", SIMD: SIMDNone}, 8192},
		{"4cpu", HardwareHeuristic{NumCPU: 4, GOARCH: "amd64", SIMD: SIMDNone}, 4096},
		{"8cpu_generic", HardwareHeuristic{NumCPU: 8, GOARCH: "amd64", SIMD: SIMDNone}, 2048},
		{"8cpu_avx2", HardwareHeuristic{NumCPU: 8, GOARCH: "amd64", SIMD: SIMDAVX2}, 1792},
		{"8cpu_avx512", HardwareHeuristic{NumCPU: 8, GOARCH: "amd64", SIMD: SIMDAVX512}, 1536},
		{"32cpu_arm64", HardwareHeuristic{NumCPU: 32, GOARCH: "arm64", SIMD: SIMDNone}, 512},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := estimateParallelThresholdForHeuristic(tc.h)
			if got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestEstimateFFTThresholdForHeuristic(t *testing.T) {
	t.Parallel()
	if 32<<(^uint(0)>>63) != 64 {
		t.Skip("SIMD-tuned FFT defaults are defined for 64-bit words")
	}
	h64 := HardwareHeuristic{NumCPU: 8, GOARCH: "amd64", SIMD: SIMDNone}
	if got := estimateFFTThresholdForHeuristic(h64); got != 500000 {
		t.Errorf("generic 64-bit: got %d, want 500000", got)
	}
	if got := estimateFFTThresholdForHeuristic(HardwareHeuristic{NumCPU: 8, GOARCH: "amd64", SIMD: SIMDAVX2}); got != 480000 {
		t.Errorf("avx2: got %d, want 480000", got)
	}
	if got := estimateFFTThresholdForHeuristic(HardwareHeuristic{NumCPU: 8, GOARCH: "amd64", SIMD: SIMDAVX512}); got != 460000 {
		t.Errorf("avx512: got %d, want 460000", got)
	}
}

func TestEstimateStrassenThresholdForHeuristic(t *testing.T) {
	t.Parallel()
	if got := estimateStrassenThresholdForHeuristic(HardwareHeuristic{NumCPU: 2, GOARCH: "amd64", SIMD: SIMDAVX512}); got != 3072 {
		t.Errorf("numCPU<4: got %d, want 3072", got)
	}
	if got := estimateStrassenThresholdForHeuristic(HardwareHeuristic{NumCPU: 8, GOARCH: "amd64", SIMD: SIMDNone}); got != 256 {
		t.Errorf("8cpu generic: got %d, want 256", got)
	}
	if got := estimateStrassenThresholdForHeuristic(HardwareHeuristic{NumCPU: 8, GOARCH: "amd64", SIMD: SIMDAVX2}); got != 240 {
		t.Errorf("avx2: got %d, want 240", got)
	}
	if got := estimateStrassenThresholdForHeuristic(HardwareHeuristic{NumCPU: 8, GOARCH: "amd64", SIMD: SIMDAVX512}); got != 224 {
		t.Errorf("avx512: got %d, want 224", got)
	}
}

// TestEstimateOptimalParallelThreshold_NonNegative verifies the estimator
// returns a non-negative value that matches the heuristic function.
func TestEstimateOptimalParallelThreshold_NonNegative(t *testing.T) {
	t.Parallel()

	got := EstimateOptimalParallelThreshold()
	if got < 0 {
		t.Errorf("EstimateOptimalParallelThreshold returned negative value: %d", got)
	}
	// The wrapper must agree with the inner implementation on the detected host.
	want := estimateParallelThresholdForHeuristic(DetectHardwareHeuristic())
	if got != want {
		t.Errorf("EstimateOptimalParallelThreshold = %d; estimateParallelThresholdForHeuristic(detected) = %d", got, want)
	}
}

// TestEstimateOptimalFFTThreshold_Positive verifies the estimator returns a
// positive threshold consistent with the heuristic function.
func TestEstimateOptimalFFTThreshold_Positive(t *testing.T) {
	t.Parallel()

	got := EstimateOptimalFFTThreshold()
	if got <= 0 {
		t.Errorf("EstimateOptimalFFTThreshold returned non-positive value: %d", got)
	}
	want := estimateFFTThresholdForHeuristic(DetectHardwareHeuristic())
	if got != want {
		t.Errorf("EstimateOptimalFFTThreshold = %d; estimateFFTThresholdForHeuristic(detected) = %d", got, want)
	}
}

// TestEstimateOptimalStrassenThreshold_Positive verifies the estimator returns
// a positive threshold consistent with the heuristic function.
func TestEstimateOptimalStrassenThreshold_Positive(t *testing.T) {
	t.Parallel()

	got := EstimateOptimalStrassenThreshold()
	if got <= 0 {
		t.Errorf("EstimateOptimalStrassenThreshold returned non-positive value: %d", got)
	}
	want := estimateStrassenThresholdForHeuristic(DetectHardwareHeuristic())
	if got != want {
		t.Errorf("EstimateOptimalStrassenThreshold = %d; estimateStrassenThresholdForHeuristic(detected) = %d", got, want)
	}
}

// TestParallelThresholdFromCPUCount pins the full CPU-count ladder, including
// tiers (e.g. 9-16 cores) that host-detected heuristics never reach on a
// given machine. A reshuffled tier would silently change parallelism defaults.
func TestParallelThresholdFromCPUCount(t *testing.T) {
	t.Parallel()
	cases := []struct {
		numCPU int
		want   int
	}{
		{1, 0},
		{2, 8192},
		{3, 4096},
		{4, 4096},
		{5, 2048},
		{8, 2048},
		{9, 1024},
		{16, 1024},
		{17, 512},
		{64, 512},
	}
	for _, tc := range cases {
		if got := parallelThresholdFromCPUCount(tc.numCPU); got != tc.want {
			t.Errorf("parallelThresholdFromCPUCount(%d) = %d, want %d", tc.numCPU, got, tc.want)
		}
	}
}
