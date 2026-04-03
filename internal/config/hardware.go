package config

import (
	"fmt"
	"runtime"

	"golang.org/x/sys/cpu"
)

// SIMDKind classifies SIMD features relevant to FFT-heavy and wide-integer work.
// On x86, AVX2/AVX-512 accelerate vectorized paths used indirectly by large multiplications.
type SIMDKind int

const (
	// SIMDNone means no AVX2-class SIMD was detected, or the architecture is not x86.
	SIMDNone SIMDKind = iota
	// SIMDAVX2 is reported when golang.org/x/sys/cpu reports AVX2 on amd64/386.
	SIMDAVX2
	// SIMDAVX512 is reported when AVX-512 foundation (AVX512F) is available on amd64/386.
	SIMDAVX512
)

// HardwareHeuristic captures runtime-detected traits used for default threshold estimation
// when the user leaves thresholds at 0 (adaptive) and no calibration profile applies.
//
// Classification uses core count, GOARCH, and (on amd64/386) SIMD capability from
// golang.org/x/sys/cpu. It is intentionally conservative to avoid regressions on unknown hardware.
type HardwareHeuristic struct {
	NumCPU int
	GOARCH string
	SIMD   SIMDKind
}

// DetectHardwareHeuristic returns a snapshot of the current host for threshold heuristics.
func DetectHardwareHeuristic() HardwareHeuristic {
	h := HardwareHeuristic{
		NumCPU: runtime.NumCPU(),
		GOARCH: runtime.GOARCH,
		SIMD:   SIMDNone,
	}
	switch runtime.GOARCH {
	case "amd64", "386":
		switch {
		case cpu.X86.HasAVX512F:
			h.SIMD = SIMDAVX512
		case cpu.X86.HasAVX2:
			h.SIMD = SIMDAVX2
		}
	}
	return h
}

// HeuristicKey returns a stable string used to invalidate calibration profiles when the
// CPU SIMD class or architecture changes (NumCPU is validated separately).
func (h HardwareHeuristic) HeuristicKey() string {
	return fmt.Sprintf("%s-%s", h.GOARCH, h.simdTag())
}

func (h HardwareHeuristic) simdTag() string {
	switch h.SIMD {
	case SIMDAVX512:
		return "avx512"
	case SIMDAVX2:
		return "avx2"
	default:
		return "generic"
	}
}

// CurrentHardwareHeuristicKey is a shortcut for DetectHardwareHeuristic().HeuristicKey().
func CurrentHardwareHeuristicKey() string {
	return DetectHardwareHeuristic().HeuristicKey()
}
