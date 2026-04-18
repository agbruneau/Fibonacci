// Package threshold implements the runtime-adaptive tuning of the FFT,
// parallelism, and Strassen switch-over thresholds used by the Fibonacci
// calculators.
//
// Rather than hard-coding a single threshold, a DynamicThresholdManager
// observes the per-iteration performance of the ongoing calculation (see
// DynamicAdjustmentInterval / MinMetricsForAdjustment / MaxMetricsHistory
// constants) and, when the measured speed-up crosses a minimum ratio
// (FFTSpeedupThreshold, ParallelSpeedupThreshold) outside a hysteresis margin
// (HysteresisMargin), raises or lowers the active threshold. This keeps the
// algorithm near the optimal operating point even when the initial
// calibration is imperfect or when hardware conditions shift mid-run.
//
// The package is concurrency-safe and expects to be called from the
// calculation goroutine; it performs no I/O.
package threshold
