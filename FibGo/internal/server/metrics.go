// Package server provides the HTTP server implementation for the Fibonacci calculator API.
package server

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics collects and exposes server metrics in Prometheus format.
// It tracks:
//   - Active requests (gauge)
//   - Total requests (counter)
//   - Server uptime (implicitly via process metrics)
//
// Calculation metrics (total, duration) are now tracked directly
// in the fibonacci package.
type Metrics struct {
	handler http.Handler
}

// Prometheus metrics for server-level observability
var (
	activeRequests = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "fibcalc_active_requests",
		Help: "Current number of active requests",
	})
	totalRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "fibcalc_requests_total",
		Help: "Total number of requests received",
	})
)

// NewMetrics creates a new Metrics instance.
func NewMetrics() *Metrics {
	return &Metrics{
		handler: promhttp.Handler(),
	}
}

// IncrementActiveRequests increments the active requests gauge
// and the total requests counter.
func (m *Metrics) IncrementActiveRequests() {
	activeRequests.Inc()
	totalRequests.Inc()
}

// DecrementActiveRequests decrements the active requests gauge.
func (m *Metrics) DecrementActiveRequests() {
	activeRequests.Dec()
}

// WritePrometheus writes metrics in Prometheus text format to the HTTP response.
//
// Parameters:
//   - w: The writer to output metrics to.
//   - r: The original HTTP request.
func (m *Metrics) WritePrometheus(w http.ResponseWriter, r *http.Request) {
	m.handler.ServeHTTP(w, r)
}

// handleMetrics is the HTTP handler for the /metrics endpoint.
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	s.metrics.WritePrometheus(w, r)
}

// metricsMiddleware tracks active requests.
func (s *Server) metricsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.metrics.IncrementActiveRequests()
		defer s.metrics.DecrementActiveRequests()
		next(w, r)
	}
}
