// Package calibration provides performance calibration for the Fibonacci calculator.
// This file implements adaptive threshold generation based on hardware characteristics.
package calibration

import (
	"runtime"
	"sort"
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

// ─────────────────────────────────────────────────────────────────────────────
// Adaptive FFT Threshold Generation
// ─────────────────────────────────────────────────────────────────────────────

// GenerateFFTThresholds generates FFT thresholds to test.
// FFT becomes beneficial for very large numbers where its O(n log n) complexity
// beats Karatsuba's O(n^1.585).
//
// The crossover point depends on:
// - CPU cache sizes (larger cache = higher threshold)
// - Memory bandwidth
// - Implementation efficiency
func GenerateFFTThresholds() []int {
	// FFT thresholds are less CPU-count dependent
	// They're more about memory hierarchy
	wordSize := 32 << (^uint(0) >> 63) // 32 or 64

	// Base thresholds
	thresholds := []int{0} // Disabled (always use Karatsuba)

	if wordSize == 64 {
		// 64-bit systems typically have higher crossover points
		thresholds = append(thresholds,
			500000,  // ~500K bits
			750000,  // ~750K bits
			1000000, // ~1M bits (default)
			1500000, // ~1.5M bits
			2000000, // ~2M bits
		)
	} else {
		// 32-bit systems have lower crossover points
		thresholds = append(thresholds,
			250000,  // ~250K bits
			500000,  // ~500K bits
			750000,  // ~750K bits
			1000000, // ~1M bits
		)
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

// GenerateStrassenThresholds generates Strassen algorithm thresholds to test.
// Strassen reduces multiplications from 8 to 7 at the cost of more additions.
// The crossover depends on the relative cost of multiplication vs addition.
func GenerateStrassenThresholds() []int {
	numCPU := runtime.NumCPU()

	// Strassen benefits depend on:
	// - Size of operands (larger = more benefit from fewer multiplications)
	// - Cache efficiency (Strassen has worse cache behavior)
	// - Parallelism availability (more cores = can parallelize the 7 multiplications)

	thresholds := []int{0} // Disabled (always use standard)

	if numCPU >= 4 {
		// With parallelism, Strassen can be beneficial at lower thresholds
		thresholds = append(thresholds, 192, 256, 384, 512, 768, 1024)
	} else {
		// Without much parallelism, need larger operands to benefit
		thresholds = append(thresholds, 256, 512, 1024, 2048, 3072)
	}

	return thresholds
}

// GenerateQuickStrassenThresholds generates a smaller set for quick calibration.
func GenerateQuickStrassenThresholds() []int {
	return []int{192, 256, 384, 512}
}

// ─────────────────────────────────────────────────────────────────────────────
// Threshold Estimation (without benchmarking)
// ─────────────────────────────────────────────────────────────────────────────

// EstimateOptimalParallelThreshold provides a heuristic estimate of the optimal
// parallel threshold without running benchmarks.
// This can be used as a fallback or starting point.
func EstimateOptimalParallelThreshold() int {
	numCPU := runtime.NumCPU()

	switch {
	case numCPU == 1:
		return 0 // No parallelism
	case numCPU <= 2:
		return 8192 // High threshold - parallelism overhead is significant
	case numCPU <= 4:
		return 4096 // Default
	case numCPU <= 8:
		return 2048 // Can use more parallelism
	case numCPU <= 16:
		return 1024 // Many cores available
	default:
		return 512 // High core count - aggressive parallelism
	}
}

// EstimateOptimalFFTThreshold provides a heuristic estimate of the optimal
// FFT threshold without running benchmarks.
func EstimateOptimalFFTThreshold() int {
	wordSize := 32 << (^uint(0) >> 63)

	if wordSize == 64 {
		return 500000 // 500K bits on 64-bit (optimal for modern CPUs with large L3 caches)
	}
	return 250000 // 250K bits on 32-bit (lower due to smaller word size)
}

// EstimateOptimalStrassenThreshold provides a heuristic estimate of the optimal
// Strassen threshold without running benchmarks.
func EstimateOptimalStrassenThreshold() int {
	numCPU := runtime.NumCPU()

	if numCPU >= 4 {
		return 256 // With parallelism, lower threshold
	}
	return 3072 // Default from constants
}

// ─────────────────────────────────────────────────────────────────────────────
// Threshold Validation
// ─────────────────────────────────────────────────────────────────────────────

// ValidateThresholds ensures thresholds are within reasonable bounds.
func ValidateThresholds(parallel, fft, strassen int) (int, int, int) {
	// Parallel threshold: 0 to 65536
	if parallel < 0 {
		parallel = 0
	}
	if parallel > 65536 {
		parallel = 65536
	}

	// FFT threshold: 0 to 10M
	if fft < 0 {
		fft = 0
	}
	if fft > 10000000 {
		fft = 10000000
	}

	// Strassen threshold: 0 to 10000
	if strassen < 0 {
		strassen = 0
	}
	if strassen > 10000 {
		strassen = 10000
	}

	return parallel, fft, strassen
}

// ─────────────────────────────────────────────────────────────────────────────
// Combined Threshold Generation
// ─────────────────────────────────────────────────────────────────────────────

// ThresholdSet represents a complete set of thresholds to test.
type ThresholdSet struct {
	Parallel []int
	FFT      []int
	Strassen []int
}

// GenerateFullThresholdSet generates all thresholds for comprehensive calibration.
func GenerateFullThresholdSet() ThresholdSet {
	return ThresholdSet{
		Parallel: GenerateParallelThresholds(),
		FFT:      GenerateFFTThresholds(),
		Strassen: GenerateStrassenThresholds(),
	}
}

// GenerateQuickThresholdSet generates thresholds for quick auto-calibration.
func GenerateQuickThresholdSet() ThresholdSet {
	return ThresholdSet{
		Parallel: GenerateQuickParallelThresholds(),
		FFT:      GenerateQuickFFTThresholds(),
		Strassen: GenerateQuickStrassenThresholds(),
	}
}

// EstimatedThresholds returns heuristic estimates without benchmarking.
func EstimatedThresholds() (parallel, fft, strassen int) {
	return EstimateOptimalParallelThreshold(),
		EstimateOptimalFFTThreshold(),
		EstimateOptimalStrassenThreshold()
}

// SortThresholds sorts each threshold slice in ascending order.
func (t *ThresholdSet) SortThresholds() {
	sort.Ints(t.Parallel)
	sort.Ints(t.FFT)
	sort.Ints(t.Strassen)
}
