package config_test

import (
	"runtime"
	"strings"
	"testing"

	config "github.com/agbruneau/FibGo/internal/config"
	"golang.org/x/sys/cpu"
)

func TestDetectHardwareHeuristic(t *testing.T) {
	t.Parallel()
	h := config.DetectHardwareHeuristic()
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
	if config.CurrentHardwareHeuristicKey() != key {
		t.Error("CurrentHardwareHeuristicKey inconsistent with DetectHardwareHeuristic")
	}
}

func TestHardwareHeuristicHeuristicKey(t *testing.T) {
	t.Parallel()
	k := config.HardwareHeuristic{GOARCH: "amd64", SIMD: config.SIMDAVX2}.HeuristicKey()
	if k != "amd64-avx2" {
		t.Errorf("got %q, want amd64-avx2", k)
	}
}

// TestHeuristicKeySIMDVariants pins the calibration-profile invalidation key
// for every SIMD class; a silent rename would orphan stored profiles.
func TestHeuristicKeySIMDVariants(t *testing.T) {
	t.Parallel()
	cases := []struct {
		h    config.HardwareHeuristic
		want string
	}{
		{config.HardwareHeuristic{GOARCH: "amd64", SIMD: config.SIMDAVX512}, "amd64-avx512"},
		{config.HardwareHeuristic{GOARCH: "amd64", SIMD: config.SIMDAVX2}, "amd64-avx2"},
		{config.HardwareHeuristic{GOARCH: "arm64", SIMD: config.SIMDNone}, "arm64-generic"},
	}
	for _, tc := range cases {
		if got := tc.h.HeuristicKey(); got != tc.want {
			t.Errorf("HeuristicKey(%+v) = %q, want %q", tc.h, got, tc.want)
		}
	}
}

// TestDetectHardwareHeuristic_SIMDClassification forces the x86 feature flags
// reported by golang.org/x/sys/cpu to verify the AVX512 > AVX2 > none
// priority.
//
// Not t.Parallel(): it mutates the package-level golang.org/x/sys/cpu.X86
// feature flags that DetectHardwareHeuristic reads, and other tests in this
// binary (TestDetectHardwareHeuristic, TestApplyAdaptiveThresholds_*, and the
// white-box TestEstimateOptimalParallelThreshold_NonNegative family) call
// DetectHardwareHeuristic expecting the real host's flags. Go's test runner
// completes every non-parallel test — including this one's t.Cleanup
// restoration — before any t.Parallel() test body executes, so those callers
// never observe the forced values.
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
		want    config.SIMDKind
	}{
		{"avx512 wins over avx2", true, true, config.SIMDAVX512},
		{"avx2 only", false, true, config.SIMDAVX2},
		{"no simd", false, false, config.SIMDNone},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cpu.X86.HasAVX512F, cpu.X86.HasAVX2 = tc.avx512f, tc.avx2
			if got := config.DetectHardwareHeuristic().SIMD; got != tc.want {
				t.Errorf("SIMD = %v, want %v", got, tc.want)
			}
		})
	}
}
