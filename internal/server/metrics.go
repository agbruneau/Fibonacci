// Package server provides the HTTP server implementation for the Fibonacci calculator API.
package server

import (
	"fmt"
	"io"
	"net/http"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics collects and exposes server metrics in Prometheus format.
// It tracks:
//   - Total calculation requests (by algorithm and status)
//   - Calculation duration histograms
//   - Active requests gauge
//   - Server uptime
type Metrics struct {
	startTime time.Time

	// Counters
	calculationsTotal   map[string]*uint64 // key: "algorithm:status"
	calculationsMu      sync.RWMutex

	// Histograms (duration buckets in seconds)
	durationHistogram   map[string]*histogram // key: algorithm
	durationMu          sync.RWMutex

	// Gauges
	activeRequests      int64
	totalRequests       uint64
}

// histogram represents a simple histogram for duration tracking.
type histogram struct {
	buckets []float64         // bucket boundaries in seconds
	counts  []uint64          // count per bucket
	sum     float64           // sum of all observed values
	count   uint64            // total count
	mu      sync.Mutex
}

// defaultBuckets defines the default histogram buckets (in seconds).
var defaultBuckets = []float64{
	0.001,  // 1ms
	0.005,  // 5ms
	0.01,   // 10ms
	0.025,  // 25ms
	0.05,   // 50ms
	0.1,    // 100ms
	0.25,   // 250ms
	0.5,    // 500ms
	1.0,    // 1s
	2.5,    // 2.5s
	5.0,    // 5s
	10.0,   // 10s
	30.0,   // 30s
	60.0,   // 1min
	300.0,  // 5min
}

// NewMetrics creates a new Metrics instance.
func NewMetrics() *Metrics {
	return &Metrics{
		startTime:         time.Now(),
		calculationsTotal: make(map[string]*uint64),
		durationHistogram: make(map[string]*histogram),
	}
}

// newHistogram creates a new histogram with the default buckets.
func newHistogram() *histogram {
	return &histogram{
		buckets: defaultBuckets,
		counts:  make([]uint64, len(defaultBuckets)+1), // +1 for +Inf bucket
	}
}

// observe records a value in the histogram.
func (h *histogram) observe(value float64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.sum += value
	h.count++

	// Find the appropriate bucket
	for i, boundary := range h.buckets {
		if value <= boundary {
			h.counts[i]++
			return
		}
	}
	// Value exceeds all buckets, goes to +Inf
	h.counts[len(h.buckets)]++
}

// RecordCalculation records a calculation request.
//
// Parameters:
//   - algorithm: The algorithm used (e.g., "fast", "matrix", "fft").
//   - status: The result status ("success" or "error").
//   - duration: The time taken for the calculation.
func (m *Metrics) RecordCalculation(algorithm, status string, duration time.Duration) {
	key := algorithm + ":" + status

	// Increment counter
	m.calculationsMu.Lock()
	if m.calculationsTotal[key] == nil {
		var zero uint64
		m.calculationsTotal[key] = &zero
	}
	atomic.AddUint64(m.calculationsTotal[key], 1)
	m.calculationsMu.Unlock()

	// Record duration histogram
	m.durationMu.Lock()
	if m.durationHistogram[algorithm] == nil {
		m.durationHistogram[algorithm] = newHistogram()
	}
	hist := m.durationHistogram[algorithm]
	m.durationMu.Unlock()

	hist.observe(duration.Seconds())
}

// IncrementActiveRequests increments the active requests gauge.
func (m *Metrics) IncrementActiveRequests() {
	atomic.AddInt64(&m.activeRequests, 1)
	atomic.AddUint64(&m.totalRequests, 1)
}

// DecrementActiveRequests decrements the active requests gauge.
func (m *Metrics) DecrementActiveRequests() {
	atomic.AddInt64(&m.activeRequests, -1)
}

// WritePrometheus writes metrics in Prometheus text format.
//
// Parameters:
//   - w: The writer to output metrics to.
func (m *Metrics) WritePrometheus(w io.Writer) {
	// Server uptime
	uptime := time.Since(m.startTime).Seconds()
	fmt.Fprintf(w, "# HELP fibcalc_uptime_seconds Server uptime in seconds\n")
	fmt.Fprintf(w, "# TYPE fibcalc_uptime_seconds gauge\n")
	fmt.Fprintf(w, "fibcalc_uptime_seconds %.3f\n\n", uptime)

	// Active requests
	fmt.Fprintf(w, "# HELP fibcalc_active_requests Current number of active requests\n")
	fmt.Fprintf(w, "# TYPE fibcalc_active_requests gauge\n")
	fmt.Fprintf(w, "fibcalc_active_requests %d\n\n", atomic.LoadInt64(&m.activeRequests))

	// Total requests
	fmt.Fprintf(w, "# HELP fibcalc_requests_total Total number of requests received\n")
	fmt.Fprintf(w, "# TYPE fibcalc_requests_total counter\n")
	fmt.Fprintf(w, "fibcalc_requests_total %d\n\n", atomic.LoadUint64(&m.totalRequests))

	// Calculation totals by algorithm and status
	fmt.Fprintf(w, "# HELP fibcalc_calculations_total Total Fibonacci calculations by algorithm and status\n")
	fmt.Fprintf(w, "# TYPE fibcalc_calculations_total counter\n")

	m.calculationsMu.RLock()
	keys := make([]string, 0, len(m.calculationsTotal))
	for k := range m.calculationsTotal {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		// Parse algorithm:status
		algo, status := parseKey(key)
		count := atomic.LoadUint64(m.calculationsTotal[key])
		fmt.Fprintf(w, "fibcalc_calculations_total{algorithm=%q,status=%q} %d\n", algo, status, count)
	}
	m.calculationsMu.RUnlock()
	fmt.Fprintln(w)

	// Duration histograms
	fmt.Fprintf(w, "# HELP fibcalc_calculation_duration_seconds Histogram of calculation durations\n")
	fmt.Fprintf(w, "# TYPE fibcalc_calculation_duration_seconds histogram\n")

	m.durationMu.RLock()
	algoKeys := make([]string, 0, len(m.durationHistogram))
	for k := range m.durationHistogram {
		algoKeys = append(algoKeys, k)
	}
	sort.Strings(algoKeys)
	for _, algo := range algoKeys {
		hist := m.durationHistogram[algo]
		hist.mu.Lock()

		// Cumulative counts for histogram buckets
		var cumulative uint64
		for i, boundary := range hist.buckets {
			cumulative += hist.counts[i]
			fmt.Fprintf(w, "fibcalc_calculation_duration_seconds_bucket{algorithm=%q,le=\"%.3f\"} %d\n",
				algo, boundary, cumulative)
		}
		cumulative += hist.counts[len(hist.buckets)]
		fmt.Fprintf(w, "fibcalc_calculation_duration_seconds_bucket{algorithm=%q,le=\"+Inf\"} %d\n",
			algo, cumulative)
		fmt.Fprintf(w, "fibcalc_calculation_duration_seconds_sum{algorithm=%q} %.6f\n", algo, hist.sum)
		fmt.Fprintf(w, "fibcalc_calculation_duration_seconds_count{algorithm=%q} %d\n", algo, hist.count)

		hist.mu.Unlock()
	}
	m.durationMu.RUnlock()
	fmt.Fprintln(w)

	// Go runtime metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	fmt.Fprintf(w, "# HELP fibcalc_go_goroutines Number of goroutines\n")
	fmt.Fprintf(w, "# TYPE fibcalc_go_goroutines gauge\n")
	fmt.Fprintf(w, "fibcalc_go_goroutines %d\n\n", runtime.NumGoroutine())

	fmt.Fprintf(w, "# HELP fibcalc_go_memstats_alloc_bytes Bytes allocated and still in use\n")
	fmt.Fprintf(w, "# TYPE fibcalc_go_memstats_alloc_bytes gauge\n")
	fmt.Fprintf(w, "fibcalc_go_memstats_alloc_bytes %d\n\n", memStats.Alloc)

	fmt.Fprintf(w, "# HELP fibcalc_go_memstats_heap_objects Number of allocated heap objects\n")
	fmt.Fprintf(w, "# TYPE fibcalc_go_memstats_heap_objects gauge\n")
	fmt.Fprintf(w, "fibcalc_go_memstats_heap_objects %d\n\n", memStats.HeapObjects)

	fmt.Fprintf(w, "# HELP fibcalc_go_gc_duration_seconds Duration of GC pauses\n")
	fmt.Fprintf(w, "# TYPE fibcalc_go_gc_duration_seconds gauge\n")
	fmt.Fprintf(w, "fibcalc_go_gc_duration_seconds %.6f\n", float64(memStats.PauseTotalNs)/1e9)
}

// parseKey splits "algorithm:status" into separate parts.
func parseKey(key string) (algorithm, status string) {
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == ':' {
			return key[:i], key[i+1:]
		}
	}
	return key, ""
}

// handleMetrics is the HTTP handler for the /metrics endpoint.
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	s.metrics.WritePrometheus(w)
}

// metricsMiddleware tracks active requests.
func (s *Server) metricsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.metrics.IncrementActiveRequests()
		defer s.metrics.DecrementActiveRequests()
		next(w, r)
	}
}

