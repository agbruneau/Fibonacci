package calibration

import (
	"slices"
	"testing"

	"github.com/agbruneau/FibGo/internal/config"
)

// Every core-count branch of the threshold generators, exercised on one host.
//
// TestGenerateParallelThresholds calls the exported functions, which read
// runtime.NumCPU(), so it can only ever cover the branch matching the machine it
// runs on. That is how an expectation of a sequential baseline of 0 — a value
// FIB-02 had replaced with config.ThresholdDisabled — survived unnoticed inside
// the "4 cores or fewer" branch on a 24-core development host, until the first
// CI run on a 4-core runner (audit PRO-01).
//
// These tests call the pure cores, so the core count is an input.

func TestParallelThresholdsFor_EveryCoreBranch(t *testing.T) {
	t.Parallel()

	const seq = config.ThresholdDisabled

	tests := []struct {
		name   string
		numCPU int
		want   []int
	}{
		{"single core is sequential only", 1, []int{seq}},
		{"two cores", 2, []int{seq, 512, 1024, 2048, 4096}},
		{"four cores, upper bound of the branch", 4, []int{seq, 512, 1024, 2048, 4096}},
		{"five cores crosses into the medium branch", 5, []int{seq, 256, 512, 1024, 2048, 4096, 8192}},
		{"eight cores, upper bound", 8, []int{seq, 256, 512, 1024, 2048, 4096, 8192}},
		{"nine cores", 9, []int{seq, 256, 512, 1024, 2048, 4096, 8192, 16384}},
		{"sixteen cores, upper bound", 16, []int{seq, 256, 512, 1024, 2048, 4096, 8192, 16384}},
		{"seventeen cores takes the full range", 17, []int{seq, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768}},
		{"very high core count", 128, []int{seq, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parallelThresholdsFor(tt.numCPU)
			if !slices.Equal(got, tt.want) {
				t.Errorf("parallelThresholdsFor(%d) = %v, want %v", tt.numCPU, got, tt.want)
			}
		})
	}
}

func TestQuickParallelThresholdsFor_EveryCoreBranch(t *testing.T) {
	t.Parallel()

	const seq = config.ThresholdDisabled

	tests := []struct {
		name   string
		numCPU int
		want   []int
	}{
		{"single core is sequential only", 1, []int{seq}},
		{"two cores", 2, []int{seq, 2048, 4096}},
		{"four cores, upper bound", 4, []int{seq, 2048, 4096}},
		{"five cores", 5, []int{seq, 2048, 4096, 8192}},
		{"eight cores, upper bound", 8, []int{seq, 2048, 4096, 8192}},
		{"nine cores takes the full set", 9, []int{seq, 2048, 4096, 8192, 16384}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := quickParallelThresholdsFor(tt.numCPU)
			if !slices.Equal(got, tt.want) {
				t.Errorf("quickParallelThresholdsFor(%d) = %v, want %v", tt.numCPU, got, tt.want)
			}
		})
	}
}

// Whatever the core count, the sequential baseline must be present and must be
// ThresholdDisabled. A 0 there would be silently rewritten to the package
// default by normalizeOptions, so the no-parallelism run would never be
// measured — the defect FIB-02 fixed.
func TestParallelThresholds_AlwaysCarryTheSequentialBaseline(t *testing.T) {
	t.Parallel()

	for _, numCPU := range []int{1, 2, 3, 4, 6, 8, 12, 16, 24, 64, 256} {
		for _, got := range [][]int{
			parallelThresholdsFor(numCPU),
			quickParallelThresholdsFor(numCPU),
		} {
			if len(got) == 0 {
				t.Fatalf("numCPU=%d: empty candidate list", numCPU)
			}
			if got[0] != config.ThresholdDisabled {
				t.Errorf("numCPU=%d: first candidate = %d, want %d (sequential baseline)",
					numCPU, got[0], config.ThresholdDisabled)
			}
			if slices.Contains(got, 0) {
				t.Errorf("numCPU=%d: candidate list contains 0, which normalizeOptions "+
					"rewrites to the package default: %v", numCPU, got)
			}
		}
	}
}
