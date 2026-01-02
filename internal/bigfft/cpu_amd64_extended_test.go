//go:build amd64

package bigfft

import (
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// CPU Feature Extended Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestHasBMI2(t *testing.T) {
	t.Parallel()
	hasBMI2 := HasBMI2()
	features := GetCPUFeatures()
	if hasBMI2 != features.BMI2 {
		t.Errorf("HasBMI2() = %v, but GetCPUFeatures().BMI2 = %v", hasBMI2, features.BMI2)
	}
	t.Logf("BMI2 available: %v", hasBMI2)
}

func TestHasADX(t *testing.T) {
	t.Parallel()
	hasADX := HasADX()
	features := GetCPUFeatures()
	if hasADX != features.ADX {
		t.Errorf("HasADX() = %v, but GetCPUFeatures().ADX = %v", hasADX, features.ADX)
	}
	t.Logf("ADX available: %v", hasADX)
}

func TestGetActiveImplementation(t *testing.T) {
	t.Parallel()
	impl := GetActiveImplementation()
	// Verify it's a valid SIMDLevel
	implStr := impl.String()
	if implStr == "" {
		t.Error("GetActiveImplementation() returned level with empty string representation")
	}
	t.Logf("Active implementation: %s", implStr)
}

func TestDisableAVX2(t *testing.T) {
	defer func() {
		// Restore default implementation
		selectImplementation()
	}()

	DisableAVX2()
	// After disabling, level should be SIMDNone or lower
	t.Logf("Implementation level after DisableAVX2: %s", implLevel.String())
}

func TestDisableAVX512(t *testing.T) {
	defer func() {
		// Restore default implementation
		selectImplementation()
	}()

	DisableAVX512()
	// AVX512 shouldn't be enabled after calling this
	if HasAVX512() && implLevel == SIMDAVX512 {
		t.Error("AVX512 should have been disabled")
	}
	t.Logf("Implementation level after DisableAVX512: %s", implLevel.String())
}

func TestEnableAllSIMD(t *testing.T) {
	defer func() {
		// Restore default implementation
		selectImplementation()
	}()

	// First disable everything
	UseDefault()

	// Then enable all
	EnableAllSIMD()

	// Should have selected highest available
	if HasAVX512() {
		if implLevel != SIMDAVX512 {
			t.Logf("Warning: AVX512 available but level is %s", implLevel.String())
		}
	} else if HasAVX2() {
		if implLevel != SIMDAVX2 {
			t.Errorf("AVX2 available but level is %s", implLevel.String())
		}
	}
	t.Logf("Implementation level after EnableAllSIMD: %s", implLevel.String())
}

func TestCPUFeaturesString(t *testing.T) {
	t.Parallel()
	features := GetCPUFeatures()
	str := features.String()
	if str == "" {
		t.Error("CPUFeatures.String() returned empty string")
	}
	t.Logf("CPU Features string: %s", str)
}

func TestSIMDLevelString(t *testing.T) {
	t.Parallel()
	levels := []SIMDLevel{SIMDNone, SIMDAVX2, SIMDAVX512, SIMDLevel(99)} // Including unknown
	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			t.Parallel()
			str := level.String()
			if str == "" {
				t.Errorf("SIMDLevel(%d).String() returned empty string", level)
			}
			t.Logf("SIMDLevel %d: %s", level, str)
		})
	}
}
