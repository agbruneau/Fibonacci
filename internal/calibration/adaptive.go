// This file implements adaptive threshold generation based on hardware characteristics.

package calibration

import (
	"runtime"
)

// ─────────────────────────────────────────────────────────────────────────────
// Adaptive Parallel Threshold Generation
// ─────────────────────────────────────────────────────────────────────────────

// GenerateParallelThresholds generates a list of parallel thresholds to test
// based on the number of available CPU cores.
//
// The rationale:
// - Single-core: Only test sequential (0) as parallelism has no benefit
// - 2-4 cores: Test lower thresholds as parallelism overhead is relatively high
// - 8+ cores: Include higher thresholds as more parallelism can be beneficial
// - 16+ cores: Add even higher thresholds for very fine-grained parallelism
func GenerateParallelThresholds() []int {
	return generateParallelThresholds(runtime.NumCPU())
}

// generateParallelThresholds is the CPU-count-injected core of
// GenerateParallelThresholds: runtime.NumCPU ignores GOMAXPROCS, so the
// low-CPU branches are untestable without this seam.
func generateParallelThresholds(numCPU int) []int {
	// Base thresholds always tested. -1 (not 0) is the genuine sequential
	// baseline: normalizeOptions only replaces ==0 with the package
	// default, so 0 silently duplicated the default candidate and the
	// true no-parallelism run was never measured (FIB-02).
	thresholds := []int{-1} // Sequential (no parallelism)

	switch {
	case numCPU == 1:
		// Single core: only sequential makes sense
		return thresholds

	case numCPU <= 4:
		// Few cores: test moderate thresholds
		thresholds = append(thresholds, 512, 1024, 2048, 4096)

	case numCPU <= 8:
		// Medium core count: broader range
		thresholds = append(thresholds, 256, 512, 1024, 2048, 4096, 8192)

	case numCPU <= 16:
		// Many cores: include higher thresholds
		thresholds = append(thresholds, 256, 512, 1024, 2048, 4096, 8192, 16384)

	default:
		// High core count (16+): full range including very high thresholds
		thresholds = append(thresholds, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768)
	}

	return thresholds
}

// GenerateQuickParallelThresholds generates a smaller set of thresholds for
// quick auto-calibration at startup.
func GenerateQuickParallelThresholds() []int {
	numCPU := runtime.NumCPU()

	if numCPU == 1 {
		return []int{-1}
	}

	// Reduced set for quick calibration
	switch {
	case numCPU <= 4:
		return []int{-1, 2048, 4096}
	case numCPU <= 8:
		return []int{-1, 2048, 4096, 8192}
	default:
		return []int{-1, 2048, 4096, 8192, 16384}
	}
}

// GenerateFFTThresholds generates a comprehensive list of FFT thresholds to test,
// sweeping the range 200K-1M bits by steps of 50K bits.
func GenerateFFTThresholds() []int {
	thresholds := []int{-1} // Always include sequential (no-FFT) baseline

	for t := 200000; t <= 1000000; t += 50000 {
		thresholds = append(thresholds, t)
	}

	return thresholds
}

// generateQuickFFTThresholds generates a smaller set for quick calibration.
func generateQuickFFTThresholds() []int {
	return []int{-1, 750000, 1000000, 1500000}
}

// ─────────────────────────────────────────────────────────────────────────────
// Adaptive Strassen Threshold Generation
// ─────────────────────────────────────────────────────────────────────────────

// GenerateQuickStrassenThresholds generates a smaller set for quick calibration.
func GenerateQuickStrassenThresholds() []int {
	return []int{192, 256, 384, 512}
}

// Threshold estimation without benchmarking lives in internal/config
// (config.EstimateOptimal*); callers use it directly. The pass-through
// delegates that used to live here were removed (audit 2026-06).
