package threshold

import "time"

// MetricsBuffer is a fixed-capacity ring buffer storing the most recent
// IterationMetric samples. Operations are O(1) and allocation-free; the
// buffer is intentionally not goroutine-safe — synchronization is the
// caller's responsibility (the Manager owns this concern).
//
// The buffer keeps a "lifetime" iteration counter independent from the
// number of currently retained samples, which Count() exposes capped to
// the buffer's capacity.
type MetricsBuffer struct {
	data    [MaxMetricsHistory]IterationMetric
	head    int // Index of the next slot to write
	written int // Total samples ever written (monotonically increasing)
}

// Record appends a metric to the buffer, overwriting the oldest sample
// once capacity is reached.
func (b *MetricsBuffer) Record(bitLen int, duration time.Duration, usedFFT, usedParallel bool) {
	b.data[b.head] = IterationMetric{
		BitLen:       bitLen,
		Duration:     duration,
		UsedFFT:      usedFFT,
		UsedParallel: usedParallel,
	}
	b.head = (b.head + 1) % MaxMetricsHistory
	b.written++
}

// Count returns the number of valid samples currently held in the buffer,
// capped at MaxMetricsHistory.
func (b *MetricsBuffer) Count() int {
	if b.written > MaxMetricsHistory {
		return MaxMetricsHistory
	}
	return b.written
}

// writtenCount returns the total number of samples ever written, even after
// the buffer wraps. Useful to drive periodic adjustment cadences.
func (b *MetricsBuffer) writtenCount() int {
	return b.written
}

// RecentMetrics returns a copy of the currently held samples.
//
// The order is unspecified: temporal order is irrelevant for the
// aggregate computations performed by ThresholdAnalyzer (averages over
// the window). A fresh slice is returned so callers may mutate it freely
// without affecting the buffer.
func (b *MetricsBuffer) RecentMetrics() []IterationMetric {
	count := b.Count()
	if count == 0 {
		return nil
	}
	out := make([]IterationMetric, count)
	if b.written <= MaxMetricsHistory {
		copy(out, b.data[:count])
	} else {
		// Buffer wrapped; samples occupy the entire backing array.
		copy(out, b.data[:])
	}
	return out
}

// Reset clears the buffer state. Backing storage is retained.
func (b *MetricsBuffer) Reset() {
	b.head = 0
	b.written = 0
}
