package config

// Threshold resolution chain as implemented, highest priority first:
//
//  1. CLI flags (--threshold, --fft-threshold, --strassen-threshold). Setting
//     one marks it explicit (markExplicitThresholds, env.go) and nothing
//     downstream overwrites it.
//  2. Environment variables (FIBCALC_THRESHOLD, FIBCALC_FFT_THRESHOLD,
//     FIBCALC_STRASSEN_THRESHOLD), applied by applyEnvOverrides to any flag
//     not set explicitly on the command line. These also mark the threshold
//     explicit: they are the user speaking, just through a different channel.
//  3. Cached calibration profile (~/.fibcalc_calibration.json), when it loads
//     and validates. app.New applies it after ParseConfig, and
//     calibration.LoadCachedCalibration now fills ONLY the thresholds left
//     implicit.
//
//     Until the 2026-09 audit (M-03) it overwrote all three unconditionally,
//     so a cached profile beat an explicit flag and the user got no hint that
//     their value had been dropped. The old behavior was documented here as a
//     "KNOWN SURPRISE" and pinned by a test through three audits without being
//     decided; it is decided now, in favor of the flag. A profile is the
//     tool's guess at what the user did not specify.
//  4. Adaptive hardware estimation (ApplyAdaptiveThresholds, this file). It
//     runs only on the no-cached-profile branch and fills only the values
//     still left at 0.
//  5. Static defaults in fibonacci/constants.go, applied by
//     fibonacci.normalizeOptions to anything still 0 at the calculator
//     boundary (e.g. the single-CPU case, where the estimator returns 0).
//
// A fresh --calibrate / --auto-calibrate sweep is outside this chain: there
// the user asked for a measurement, the result is printed, and it is the
// measured value that is persisted and applied.

// ApplyAdaptiveThresholds adjusts the configuration thresholds based on
// hardware characteristics (CPU cores, architecture) when default values
// are detected. This provides automatic performance optimization without
// requiring explicit calibration.
//
// The function only modifies thresholds that are set to their zero default,
// preserving any user-specified overrides via command-line flags.
func ApplyAdaptiveThresholds(cfg AppConfig) AppConfig {
	if cfg.Threshold == 0 {
		cfg.Threshold = EstimateOptimalParallelThreshold()
	}
	if cfg.FFTThreshold == 0 {
		cfg.FFTThreshold = EstimateOptimalFFTThreshold()
	}
	if cfg.StrassenThreshold == 0 {
		cfg.StrassenThreshold = EstimateOptimalStrassenThreshold()
	}
	return cfg
}

// EstimateOptimalParallelThreshold provides a heuristic estimate of the optimal
// parallel threshold without running benchmarks.
// This can be used as a fallback or starting point.
func EstimateOptimalParallelThreshold() int {
	return estimateParallelThresholdForHeuristic(DetectHardwareHeuristic())
}

// estimateParallelThresholdForHeuristic applies the same rules as EstimateOptimalParallelThreshold
// for a simulated host (tests, diagnostics).
func estimateParallelThresholdForHeuristic(h HardwareHeuristic) int {
	base := parallelThresholdFromCPUCount(h.NumCPU)
	// Slightly more aggressive parallelism when SIMD throughput is high and enough cores exist.
	switch h.SIMD {
	case SIMDAVX512:
		if h.NumCPU >= 8 && base > 512 {
			return max(512, base-512)
		}
	case SIMDAVX2:
		if h.NumCPU >= 8 && base > 512 {
			return max(512, base-256)
		}
	}
	return base
}

func parallelThresholdFromCPUCount(numCPU int) int {
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
	return estimateFFTThresholdForHeuristic(DetectHardwareHeuristic())
}

// estimateFFTThresholdForHeuristic applies the same rules as EstimateOptimalFFTThreshold
// for a simulated host (tests, diagnostics).
func estimateFFTThresholdForHeuristic(h HardwareHeuristic) int {
	wordSize := 32 << (^uint(0) >> 63)

	if wordSize != 64 {
		return 250000 // 32-bit: lower due to smaller word size
	}
	// On 64-bit x86 with strong SIMD, FFT paths become attractive slightly earlier (lower crossover).
	switch h.SIMD {
	case SIMDAVX512:
		return 460000
	case SIMDAVX2:
		return 480000
	default:
		return 500000
	}
}

// EstimateOptimalStrassenThreshold provides a heuristic estimate of the optimal
// Strassen threshold without running benchmarks.
func EstimateOptimalStrassenThreshold() int {
	return estimateStrassenThresholdForHeuristic(DetectHardwareHeuristic())
}

// estimateStrassenThresholdForHeuristic applies the same rules as EstimateOptimalStrassenThreshold
// for a simulated host (tests, diagnostics).
func estimateStrassenThresholdForHeuristic(h HardwareHeuristic) int {
	if h.NumCPU < 4 {
		return 3072 // Default from constants when parallelism is limited
	}
	switch h.SIMD {
	case SIMDAVX512:
		return 224
	case SIMDAVX2:
		return 240
	default:
		return 256
	}
}
