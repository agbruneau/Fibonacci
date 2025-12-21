package fibonacci

import "time"

// IterationMetric records timing data for a single doubling iteration.
type IterationMetric struct {
	// BitLen is the bit length of FK at this iteration
	BitLen int
	// Duration is how long this iteration took
	Duration time.Duration
	// UsedFFT indicates if FFT multiplication was used
	UsedFFT bool
	// UsedParallel indicates if parallel multiplication was used
	UsedParallel bool
}

// ThresholdStats returns statistics about the dynamic threshold manager's activity.
type ThresholdStats struct {
	// CurrentFFT is the current FFT threshold
	CurrentFFT int
	// CurrentParallel is the current parallel threshold
	CurrentParallel int
	// OriginalFFT is the original FFT threshold
	OriginalFFT int
	// OriginalParallel is the original parallel threshold
	OriginalParallel int
	// MetricsCollected is the number of metrics collected
	MetricsCollected int
	// IterationsProcessed is the total number of iterations processed
	IterationsProcessed int
}

// DynamicThresholdConfig holds configuration for dynamic threshold adjustment.
type DynamicThresholdConfig struct {
	// InitialFFTThreshold is the starting FFT threshold
	InitialFFTThreshold int
	// InitialParallelThreshold is the starting parallel threshold
	InitialParallelThreshold int
	// AdjustmentInterval is how often to check for adjustments (in iterations)
	AdjustmentInterval int
	// Enabled controls whether dynamic adjustment is active
	Enabled bool
}
