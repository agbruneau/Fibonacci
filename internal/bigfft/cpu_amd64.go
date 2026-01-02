//go:build amd64

// Package bigfft implements multiplication of big.Int using FFT.
// This file provides CPU feature detection for SIMD optimizations on amd64.
package bigfft

import (
	"sync"

	"golang.org/x/sys/cpu"
)

// ─────────────────────────────────────────────────────────────────────────────
// CPU Feature Detection
// ─────────────────────────────────────────────────────────────────────────────

// CPU feature flags detected at init time
var (
	// hasAVX2 indicates AVX2 support (256-bit SIMD)
	hasAVX2 bool

	// hasAVX512 indicates full AVX-512 support (512-bit SIMD)
	// Requires AVX512F (foundation) and AVX512DQ (double/quad word)
	hasAVX512 bool

	// hasBMI2 indicates BMI2 support (MULX, SHRX, etc.)
	hasBMI2 bool

	// hasADX indicates ADX support (ADCX, ADOX for extended precision)
	hasADX bool

	// cpuDetectionOnce ensures CPU detection runs exactly once
	cpuDetectionOnce sync.Once
)

// SIMDLevel represents the SIMD capability level of the CPU
type SIMDLevel int

const (
	// SIMDNone indicates no SIMD acceleration available
	SIMDNone SIMDLevel = iota
	// SIMDAVX2 indicates AVX2 (256-bit) acceleration available
	SIMDAVX2
	// SIMDAVX512 indicates AVX-512 (512-bit) acceleration available
	SIMDAVX512
)

// String returns a human-readable name for the SIMD level
func (l SIMDLevel) String() string {
	switch l {
	case SIMDAVX512:
		return "AVX-512"
	case SIMDAVX2:
		return "AVX2"
	default:
		return "None"
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Initialization
// ─────────────────────────────────────────────────────────────────────────────

func init() {
	detectCPUFeatures()
}

// detectCPUFeatures detects CPU SIMD capabilities
func detectCPUFeatures() {
	cpuDetectionOnce.Do(func() {
		// Detect AVX2 support
		hasAVX2 = cpu.X86.HasAVX2

		// Detect AVX-512 support (requires multiple feature flags)
		// We require AVX512F (foundation) and AVX512DQ (double/quad operations)
		// for our arithmetic operations
		hasAVX512 = cpu.X86.HasAVX512F && cpu.X86.HasAVX512DQ

		// Detect BMI2 for MULX instruction (faster multiplication)
		hasBMI2 = cpu.X86.HasBMI2

		// Detect ADX for ADCX/ADOX (parallel carry chains)
		hasADX = cpu.X86.HasADX
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Public Query Functions
// ─────────────────────────────────────────────────────────────────────────────

// HasAVX2 returns true if the CPU supports AVX2 instructions.
func HasAVX2() bool {
	return hasAVX2
}

// HasAVX512 returns true if the CPU supports AVX-512F and AVX-512DQ instructions.
func HasAVX512() bool {
	return hasAVX512
}

// HasBMI2 returns true if the CPU supports BMI2 instructions (MULX, etc.).
func HasBMI2() bool {
	return hasBMI2
}

// HasADX returns true if the CPU supports ADX instructions (ADCX, ADOX).
func HasADX() bool {
	return hasADX
}

// GetSIMDLevel returns the highest SIMD level supported by the CPU.
func GetSIMDLevel() SIMDLevel {
	if hasAVX512 {
		return SIMDAVX512
	}
	if hasAVX2 {
		return SIMDAVX2
	}
	return SIMDNone
}

// GetCPUFeatures returns a summary of detected CPU features.
func GetCPUFeatures() CPUFeatures {
	return CPUFeatures{
		AVX2:      hasAVX2,
		AVX512:    hasAVX512,
		BMI2:      hasBMI2,
		ADX:       hasADX,
		SIMDLevel: GetSIMDLevel(),
	}
}

// CPUFeatures holds detected CPU feature flags.
type CPUFeatures struct {
	AVX2      bool
	AVX512    bool
	BMI2      bool
	ADX       bool
	SIMDLevel SIMDLevel
}

// String returns a human-readable summary of CPU features.
func (f CPUFeatures) String() string {
	features := []string{}
	if f.AVX512 {
		features = append(features, "AVX-512")
	}
	if f.AVX2 {
		features = append(features, "AVX2")
	}
	if f.BMI2 {
		features = append(features, "BMI2")
	}
	if f.ADX {
		features = append(features, "ADX")
	}
	if len(features) == 0 {
		return "No SIMD features detected"
	}
	result := "CPU Features: "
	for i, f := range features {
		if i > 0 {
			result += ", "
		}
		result += f
	}
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// Implementation Selection
// ─────────────────────────────────────────────────────────────────────────────

// implLevel tracks which implementation level is currently active
var implLevel = SIMDNone

// GetActiveImplementation returns the currently active SIMD implementation level.
func GetActiveImplementation() SIMDLevel {
	return implLevel
}

// selectImplementation chooses the best available implementation.
// This is called at init time to set up function pointers.
func selectImplementation() {
	if hasAVX512 && useAVX512Impl {
		implLevel = SIMDAVX512
		selectAVX512Impl()
	} else if hasAVX2 && useAVX2Impl {
		implLevel = SIMDAVX2
		selectAVX2Impl()
	} else {
		implLevel = SIMDNone
		// Use default go:linkname implementations from arith_decl.go
	}
}

// Configuration flags to enable/disable specific implementations
// These can be modified before init for testing purposes
var (
	useAVX2Impl   = true
	useAVX512Impl = true
)

// DisableAVX2 disables AVX2 implementation (for testing/benchmarking).
// Must be called before any arithmetic operations.
func DisableAVX2() {
	useAVX2Impl = false
	selectImplementation()
}

// DisableAVX512 disables AVX-512 implementation (for testing/benchmarking).
// Must be called before any arithmetic operations.
func DisableAVX512() {
	useAVX512Impl = false
	selectImplementation()
}

// EnableAllSIMD re-enables all SIMD implementations.
func EnableAllSIMD() {
	useAVX2Impl = true
	useAVX512Impl = true
	selectImplementation()
}

// selectAVX2Impl configures AVX2 function pointers.
// This is a placeholder that will be implemented in arith_amd64.go
func selectAVX2Impl() {
	// Will be implemented to set function pointers to AVX2 versions
}

// selectAVX512Impl configures AVX-512 function pointers.
// This is a placeholder for future AVX-512 implementation
func selectAVX512Impl() {
	// Will be implemented to set function pointers to AVX-512 versions
	// For now, fall back to AVX2 if available
	if hasAVX2 {
		selectAVX2Impl()
	}
}
