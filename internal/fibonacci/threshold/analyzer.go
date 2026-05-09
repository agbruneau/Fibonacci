package threshold

import "time"

// ThresholdAnalyzer implements the pure analytical layer of the dynamic
// threshold subsystem: filtering, speedup ratio computation, hysteresis,
// and multiplicative adjustment of a threshold.
//
// The analyzer is stateless — every method takes its inputs explicitly —
// so it can be unit-tested in isolation and reused for FFT, parallel, or
// any future threshold without coupling to the Manager's lifecycle or
// concurrency model.
type ThresholdAnalyzer struct{}

// AnalysisParams captures the per-threshold configuration consumed by
// ThresholdAnalyzer.Analyze. It is exported only because it crosses file
// boundaries inside the package; callers outside the package have no need
// to construct one.
type AnalysisParams struct {
	// Predicate selects metrics belonging to the "optimized" mode
	// (e.g. metric.UsedFFT or metric.UsedParallel).
	Predicate func(IterationMetric) bool
	// SpeedupThreshold is the minimum ratio at which the optimized mode
	// is considered beneficial. Its reciprocal is the cut-off below
	// which the optimized mode is considered detrimental.
	SpeedupThreshold float64
	// LowerNumerator and RaiseNumerator (interpreted over 10) are the
	// multiplicative factors applied when lowering or raising the
	// threshold respectively (e.g. 9 → ×0.9, 11 → ×1.1).
	LowerNumerator int
	RaiseNumerator int
	// MinThreshold is the floor for the adjusted value.
	MinThreshold int
	// MaxCapMultiplier multiplied by OriginalThreshold gives the ceiling.
	MaxCapMultiplier int
	// CurrentThreshold is the value being evaluated.
	CurrentThreshold int
	// OriginalThreshold is the value the manager started with; used to
	// derive the ceiling.
	OriginalThreshold int
}

// CalculateSpeedupRatio returns avgBaseline / avgOptimized, or 0 if either
// average is non-positive (insufficient data to compute a ratio).
func (ThresholdAnalyzer) CalculateSpeedupRatio(avgOptimized, avgBaseline float64) float64 {
	if avgOptimized <= 0 || avgBaseline <= 0 {
		return 0
	}
	return avgBaseline / avgOptimized
}

// AvgTimePerBit returns the average duration (in nanoseconds) per bit
// across the supplied metrics, or 0 when the input is empty or the total
// bit count is zero.
func (ThresholdAnalyzer) AvgTimePerBit(metrics []IterationMetric) float64 {
	if len(metrics) == 0 {
		return 0
	}
	var totalTime time.Duration
	var totalBits int64
	for _, metric := range metrics {
		totalTime += metric.Duration
		totalBits += int64(metric.BitLen)
	}
	if totalBits == 0 {
		return 0
	}
	return float64(totalTime.Nanoseconds()) / float64(totalBits)
}

// SignificantChange reports whether |new − old| / |old| exceeds
// HysteresisMargin. A change from zero to a non-zero value is always
// considered significant.
func (ThresholdAnalyzer) SignificantChange(oldVal, newVal int) bool {
	if oldVal == 0 {
		return newVal != 0
	}
	change := float64(newVal-oldVal) / float64(oldVal)
	if change < 0 {
		change = -change
	}
	return change > HysteresisMargin
}

// Analyze evaluates a candidate adjustment for the threshold described by
// params. It partitions the metrics by predicate, computes the per-bit
// speedup ratio, and applies the multiplicative adjustment. When data is
// insufficient to draw a conclusion, params.CurrentThreshold is returned
// unchanged.
func (a ThresholdAnalyzer) Analyze(metrics []IterationMetric, params AnalysisParams) int {
	if len(metrics) == 0 {
		return params.CurrentThreshold
	}

	optimized, baseline := filterMetricsByMode(metrics, params.Predicate)
	if len(optimized) == 0 || len(baseline) == 0 {
		return params.CurrentThreshold
	}

	ratio := a.CalculateSpeedupRatio(a.AvgTimePerBit(optimized), a.AvgTimePerBit(baseline))
	if ratio == 0 {
		return params.CurrentThreshold
	}

	return a.ApplyThresholdAdjustment(ratio, params)
}

// ApplyThresholdAdjustment lowers the threshold when the optimized mode
// is meaningfully faster (ratio > SpeedupThreshold) and raises it when
// the optimized mode is meaningfully slower (ratio < 1/SpeedupThreshold).
// Bounds are enforced via MinThreshold and OriginalThreshold ×
// MaxCapMultiplier.
func (ThresholdAnalyzer) ApplyThresholdAdjustment(ratio float64, params AnalysisParams) int {
	if ratio > params.SpeedupThreshold {
		// Optimized mode is faster — lower the threshold.
		newThreshold := params.CurrentThreshold * params.LowerNumerator / 10
		if newThreshold < params.MinThreshold {
			newThreshold = params.MinThreshold
		}
		return newThreshold
	}
	if ratio < 1.0/params.SpeedupThreshold {
		// Optimized mode is slower — raise the threshold.
		newThreshold := params.CurrentThreshold * params.RaiseNumerator / 10
		maxCap := params.OriginalThreshold * params.MaxCapMultiplier
		if newThreshold > maxCap {
			newThreshold = maxCap
		}
		return newThreshold
	}
	return params.CurrentThreshold
}

// filterMetricsByMode partitions metrics into (matching, nonMatching)
// according to predicate. Lives in this file because it is the only
// caller's natural collaborator.
func filterMetricsByMode(metrics []IterationMetric, predicate func(IterationMetric) bool) (matching, nonMatching []IterationMetric) {
	matching = make([]IterationMetric, 0, len(metrics))
	nonMatching = make([]IterationMetric, 0, len(metrics))
	for _, metric := range metrics {
		if predicate(metric) {
			matching = append(matching, metric)
		} else {
			nonMatching = append(nonMatching, metric)
		}
	}
	return matching, nonMatching
}
