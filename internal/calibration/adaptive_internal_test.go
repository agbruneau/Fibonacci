// This file stays package calibration (white-box) because it exercises
// the unexported generateParallelThresholds(numCPU) seam and
// generateQuickFFTThresholds directly. See PLAN.md §3.3a and §4.3 (P-06):
// runtime.NumCPU ignores GOMAXPROCS, so generateParallelThresholds injects
// the CPU count as a parameter to make every branch reachable on any
// development machine.
package calibration

import (
	"runtime"
	"slices"
	"testing"
)

func TestGenerateParallelThresholds(t *testing.T) {
	t.Parallel()

	// Table-driven over the injected CPU count so every CPU-gated branch is
	// exercised on any development machine. FIB-02: the baseline candidate
	// must be -1 (genuinely sequential -- normalizeOptions only replaces
	// ==0 with the default, so 0 was a duplicate of the default candidate,
	// never a real baseline).
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

func TestGenerateQuickFFTThresholds(t *testing.T) {
	t.Parallel()
	thresholds := generateQuickFFTThresholds()
	if len(thresholds) < 2 {
		t.Error("Expected multiple quick FFT thresholds")
	}
}
