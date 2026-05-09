package threshold

import (
	"testing"
	"time"
)

func TestMetricsBuffer_EmptyState(t *testing.T) {
	t.Parallel()
	var b MetricsBuffer
	if got := b.Count(); got != 0 {
		t.Errorf("Count() empty: expected 0, got %d", got)
	}
	if got := b.Written(); got != 0 {
		t.Errorf("Written() empty: expected 0, got %d", got)
	}
	if got := b.RecentMetrics(); got != nil {
		t.Errorf("RecentMetrics() empty: expected nil, got %v", got)
	}
}

func TestMetricsBuffer_RecordBelowCapacity(t *testing.T) {
	t.Parallel()
	var b MetricsBuffer

	const n = 5
	for i := 0; i < n; i++ {
		b.Record(1000+i, time.Duration(i+1)*time.Millisecond, i%2 == 0, i%3 == 0)
	}

	if got := b.Count(); got != n {
		t.Errorf("Count() after %d records: expected %d, got %d", n, n, got)
	}
	if got := b.Written(); got != n {
		t.Errorf("Written() after %d records: expected %d, got %d", n, n, got)
	}

	metrics := b.RecentMetrics()
	if len(metrics) != n {
		t.Fatalf("RecentMetrics(): expected %d, got %d", n, len(metrics))
	}

	// Check that all recorded values are present (order-independent).
	seen := make(map[int]bool)
	for _, m := range metrics {
		seen[m.BitLen] = true
	}
	for i := 0; i < n; i++ {
		if !seen[1000+i] {
			t.Errorf("missing BitLen %d in RecentMetrics()", 1000+i)
		}
	}
}

func TestMetricsBuffer_RingBufferWrap(t *testing.T) {
	t.Parallel()
	var b MetricsBuffer

	// Record more than capacity.
	total := MaxMetricsHistory + 10
	for i := 0; i < total; i++ {
		b.Record(i, time.Millisecond, true, false)
	}

	if got := b.Count(); got != MaxMetricsHistory {
		t.Errorf("Count() after wrap: expected %d, got %d", MaxMetricsHistory, got)
	}
	if got := b.Written(); got != total {
		t.Errorf("Written() after wrap: expected %d, got %d", total, got)
	}
	if got := len(b.RecentMetrics()); got != MaxMetricsHistory {
		t.Errorf("len(RecentMetrics()): expected %d, got %d", MaxMetricsHistory, got)
	}
}

func TestMetricsBuffer_RecentMetricsCopiesData(t *testing.T) {
	t.Parallel()
	var b MetricsBuffer
	b.Record(42, time.Millisecond, true, false)

	metrics := b.RecentMetrics()
	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}

	// Mutating the returned slice must not affect the buffer.
	metrics[0].BitLen = 999
	again := b.RecentMetrics()
	if again[0].BitLen != 42 {
		t.Errorf("RecentMetrics() leaks internal state: got %d after external mutation", again[0].BitLen)
	}
}

func TestMetricsBuffer_RecordPreservesValues(t *testing.T) {
	t.Parallel()
	var b MetricsBuffer
	b.Record(7, 250*time.Microsecond, true, true)

	metrics := b.RecentMetrics()
	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}
	got := metrics[0]
	if got.BitLen != 7 {
		t.Errorf("BitLen: expected 7, got %d", got.BitLen)
	}
	if got.Duration != 250*time.Microsecond {
		t.Errorf("Duration: expected 250µs, got %v", got.Duration)
	}
	if !got.UsedFFT {
		t.Error("UsedFFT: expected true")
	}
	if !got.UsedParallel {
		t.Error("UsedParallel: expected true")
	}
}

func TestMetricsBuffer_Reset(t *testing.T) {
	t.Parallel()
	var b MetricsBuffer

	for i := 0; i < MaxMetricsHistory+5; i++ {
		b.Record(i, time.Millisecond, true, false)
	}
	b.Reset()

	if got := b.Count(); got != 0 {
		t.Errorf("Count() after Reset: expected 0, got %d", got)
	}
	if got := b.Written(); got != 0 {
		t.Errorf("Written() after Reset: expected 0, got %d", got)
	}
	if got := b.RecentMetrics(); got != nil {
		t.Errorf("RecentMetrics() after Reset: expected nil, got %v", got)
	}

	// Reset must not corrupt subsequent recording.
	b.Record(1, time.Millisecond, false, false)
	if got := b.Count(); got != 1 {
		t.Errorf("Count() after post-Reset record: expected 1, got %d", got)
	}
}
