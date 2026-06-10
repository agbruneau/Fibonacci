package config

import (
	"runtime"
	"strings"
	"testing"

	"golang.org/x/sys/cpu"
)

func TestDetectHardwareHeuristic(t *testing.T) {
	t.Parallel()
	h := DetectHardwareHeuristic()
	if h.NumCPU < 1 {
		t.Errorf("NumCPU = %d, want >= 1", h.NumCPU)
	}
	if h.GOARCH == "" {
		t.Error("GOARCH empty")
	}
	key := h.HeuristicKey()
	if !strings.Contains(key, h.GOARCH) {
		t.Errorf("HeuristicKey %q should contain GOARCH %q", key, h.GOARCH)
	}
	if CurrentHardwareHeuristicKey() != key {
		t.Error("CurrentHardwareHeuristicKey inconsistent with DetectHardwareHeuristic")
	}
}

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

func TestHardwareHeuristicHeuristicKey(t *testing.T) {
	t.Parallel()
	k := HardwareHeuristic{GOARCH: "amd64", SIMD: SIMDAVX2}.HeuristicKey()
	if k != "amd64-avx2" {
		t.Errorf("got %q, want amd64-avx2", k)
	}
}

// TestHeuristicKeySIMDVariants pins the calibration-profile invalidation key
// for every SIMD class; a silent rename would orphan stored profiles.
func TestHeuristicKeySIMDVariants(t *testing.T) {
	t.Parallel()
	cases := []struct {
		h    HardwareHeuristic
		want string
	}{
		{HardwareHeuristic{GOARCH: "amd64", SIMD: SIMDAVX512}, "amd64-avx512"},
		{HardwareHeuristic{GOARCH: "amd64", SIMD: SIMDAVX2}, "amd64-avx2"},
		{HardwareHeuristic{GOARCH: "arm64", SIMD: SIMDNone}, "arm64-generic"},
	}
	for _, tc := range cases {
		if got := tc.h.HeuristicKey(); got != tc.want {
			t.Errorf("HeuristicKey(%+v) = %q, want %q", tc.h, got, tc.want)
		}
	}
}

// TestDetectHardwareHeuristic_SIMDClassification forces the x86 feature flags
// reported by golang.org/x/sys/cpu to verify the AVX512 > AVX2 > none
// priority. It mutates package-level state in x/sys/cpu, so it must not run
// in parallel; the original flags are restored via t.Cleanup.
func TestDetectHardwareHeuristic_SIMDClassification(t *testing.T) {
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "386" {
		t.Skipf("SIMD classification reads cpu.X86 only on amd64/386 (GOARCH=%s)", runtime.GOARCH)
	}
	origAVX512, origAVX2 := cpu.X86.HasAVX512F, cpu.X86.HasAVX2
	t.Cleanup(func() {
		cpu.X86.HasAVX512F, cpu.X86.HasAVX2 = origAVX512, origAVX2
	})

	cases := []struct {
		name    string
		avx512f bool
		avx2    bool
		want    SIMDKind
	}{
		{"avx512 wins over avx2", true, true, SIMDAVX512},
		{"avx2 only", false, true, SIMDAVX2},
		{"no simd", false, false, SIMDNone},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cpu.X86.HasAVX512F, cpu.X86.HasAVX2 = tc.avx512f, tc.avx2
			if got := DetectHardwareHeuristic().SIMD; got != tc.want {
				t.Errorf("SIMD = %v, want %v", got, tc.want)
			}
		})
	}
}
