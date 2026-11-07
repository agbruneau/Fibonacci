// This file implements adaptive threshold generation based on hardware characteristics.

package calibration

import (
	"runtime"

	"github.com/agbruneau/FibGo/internal/config"
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
	return parallelThresholdsFor(runtime.NumCPU())
}

// parallelThresholdsFor is the pure core of GenerateParallelThresholds, split
// out so every core-count branch is reachable from a test (audit PRO-01).
//
// Reading the core count inside the function meant exactly one of the five
// branches ran on any given machine, and the other four were dead as far as the
// suite was concerned. That is how a stale expectation survived: the test
// asserted a sequential baseline of 0 after FIB-02 had changed it to
// ThresholdDisabled, and the assertion lived in the <=4-core branch, which the
// maintainer's 24-core host never took. The first CI run, on a 4-core runner,
// failed on it within three minutes.
func parallelThresholdsFor(numCPU int) []int {
	// Base thresholds always tested. ThresholdDisabled (-1, not 0) is the
	// genuine sequential baseline: normalizeOptions only replaces ==0 with the
	// package default, so 0 silently duplicated the default candidate and the
	// true no-parallelism run was never measured (FIB-02). Since audit H-02 the
	// value is also accepted by config.Validate, so a profile in which it wins
	// survives the next start instead of being discarded.
	thresholds := []int{config.ThresholdDisabled} // Sequential (no parallelism)

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
	return quickParallelThresholdsFor(runtime.NumCPU())
}

// quickParallelThresholdsFor is the pure core of
// GenerateQuickParallelThresholds; see parallelThresholdsFor for why the core
// count is a parameter rather than a runtime read.
func quickParallelThresholdsFor(numCPU int) []int {
	if numCPU == 1 {
		return []int{config.ThresholdDisabled}
	}

	// Reduced set for quick calibration
	switch {
	case numCPU <= 4:
		return []int{config.ThresholdDisabled, 2048, 4096}
	case numCPU <= 8:
		return []int{config.ThresholdDisabled, 2048, 4096, 8192}
	default:
		return []int{config.ThresholdDisabled, 2048, 4096, 8192, 16384}
	}
}

// GenerateFFTThresholds generates a comprehensive list of FFT thresholds to test,
// sweeping the range 200K-1M bits by steps of 50K bits.
func GenerateFFTThresholds() []int {
	// 1 no-FFT baseline + the 17 sweep steps (200K..1M by 50K).
	thresholds := make([]int, 0, 18)
	thresholds = append(thresholds, config.ThresholdDisabled) // Always include the no-FFT baseline

	for t := 200000; t <= 1000000; t += 50000 {
		thresholds = append(thresholds, t)
	}

	return thresholds
}

// ─────────────────────────────────────────────────────────────────────────────
// Adaptive Strassen Threshold Generation
// ─────────────────────────────────────────────────────────────────────────────

// GenerateQuickStrassenThresholds generates a smaller set for quick calibration.
func GenerateQuickStrassenThresholds() []int {
	return []int{192, 256, 384, 512}
}

// Threshold estimation without benchmarking lives in internal/config
// (config.EstimateOptimal*); callers use it directly rather than through
// pass-through delegates here (audit 2026-06).
