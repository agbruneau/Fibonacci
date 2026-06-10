//go:build amd64

package bigfft

import "testing"

// TestGetSIMDLevelBranches forces every detection combination so the level
// mapping is verified on any host CPU.
//
// Global state: the package-level feature flags are mutated (no t.Parallel)
// and restored via Cleanup. Only pure flag reads happen while the flags are
// altered; no SIMD code path executes (policy from incident 9cad06e).
func TestGetSIMDLevelBranches(t *testing.T) {
	origAVX2, origAVX512 := hasAVX2, hasAVX512
	t.Cleanup(func() { hasAVX2, hasAVX512 = origAVX2, origAVX512 })

	tests := []struct {
		avx512, avx2 bool
		want         SIMDLevel
	}{
		{true, true, SIMDAVX512},
		{true, false, SIMDAVX512},
		{false, true, SIMDAVX2},
		{false, false, SIMDNone},
	}
	for _, tc := range tests {
		hasAVX512, hasAVX2 = tc.avx512, tc.avx2
		if got := GetSIMDLevel(); got != tc.want {
			t.Errorf("GetSIMDLevel(avx512=%v, avx2=%v) = %v, want %v",
				tc.avx512, tc.avx2, got, tc.want)
		}
	}
}

// TestCPUFeaturesStringCombos pins the formatting of every branch of
// CPUFeatures.String on explicit values (independent of the host CPU).
func TestCPUFeaturesStringCombos(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		f    CPUFeatures
		want string
	}{
		{"none", CPUFeatures{}, "No SIMD features detected"},
		{"all", CPUFeatures{AVX2: true, AVX512: true, BMI2: true, ADX: true},
			"CPU Features: AVX-512, AVX2, BMI2, ADX"},
		{"adx only", CPUFeatures{ADX: true}, "CPU Features: ADX"},
		{"avx2 and bmi2", CPUFeatures{AVX2: true, BMI2: true}, "CPU Features: AVX2, BMI2"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.f.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}
