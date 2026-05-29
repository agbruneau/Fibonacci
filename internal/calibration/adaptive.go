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
	numCPU := runtime.NumCPU()

	// Base thresholds always tested
	thresholds := []int{0} // Sequential (no parallelism)

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
		return []int{0}
	}

	// Reduced set for quick calibration
	switch {
	case numCPU <= 4:
		return []int{0, 2048, 4096}
	case numCPU <= 8:
		return []int{0, 2048, 4096, 8192}
	default:
		return []int{0, 2048, 4096, 8192, 16384}
	}
}

// GenerateFFTThresholds generates a comprehensive list of FFT thresholds to test,
// sweeping the range 200K-1M bits by steps of 50K bits.
func GenerateFFTThresholds() []int {
	thresholds := []int{0} // Always include sequential baseline

	for t := 200000; t <= 1000000; t += 50000 {
		thresholds = append(thresholds, t)
	}

	return thresholds
}

// GenerateQuickFFTThresholds generates a smaller set for quick calibration.
func GenerateQuickFFTThresholds() []int {
	return []int{0, 750000, 1000000, 1500000}
}

// ─────────────────────────────────────────────────────────────────────────────
// Adaptive Strassen Threshold Generation
// ─────────────────────────────────────────────────────────────────────────────

// GenerateQuickStrassenThresholds generates a smaller set for quick calibration.
func GenerateQuickStrassenThresholds() []int {
	return []int{192, 256, 384, 512}
}

// ─────────────────────────────────────────────────────────────────────────────
// Threshold Estimation (without benchmarking)
// Delegates to config.EstimateOptimal* — canonical implementations live there.
// ─────────────────────────────────────────────────────────────────────────────

// EstimateOptimalParallelThreshold delegates to config.EstimateOptimalParallelThreshold.
func EstimateOptimalParallelThreshold() int { return config.EstimateOptimalParallelThreshold() }

// EstimateOptimalFFTThreshold delegates to config.EstimateOptimalFFTThreshold.
func EstimateOptimalFFTThreshold() int { return config.EstimateOptimalFFTThreshold() }

// EstimateOptimalStrassenThreshold delegates to config.EstimateOptimalStrassenThreshold.
func EstimateOptimalStrassenThreshold() int { return config.EstimateOptimalStrassenThreshold() }
