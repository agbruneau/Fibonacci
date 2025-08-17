// Package metrics gère l'instrumentation Prometheus de l'application.
package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics encapsule les compteurs et jauges.
type Metrics struct {
	CalculationsTotal   *prometheus.CounterVec
	CalculationDuration *prometheus.HistogramVec
	ErrorsTotal         *prometheus.CounterVec
}

var m *Metrics

// init initialise le singleton des métriques (idiomatique pour Prometheus).
func init() {
	m = &Metrics{
		CalculationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "fibbench_calculations_total",
				Help: "The total number of Fibonacci calculations performed.",
			},
			[]string{"algorithm"},
		),
		CalculationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "fibbench_calculation_duration_seconds",
				Help: "The duration of successful Fibonacci calculations.",
				// Buckets de 1ms à ~65s.
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 16),
			},
			[]string{"algorithm"},
		),
		ErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "fibbench_errors_total",
				Help: "The total number of errors encountered during calculations.",
			},
			[]string{"algorithm", "error_type"},
		),
	}
}

// RecordStart initialise le compteur et retourne une fonction de clôture (closure)
// qui enregistre la durée et les erreurs lorsque le calcul se termine.
func RecordStart(algorithmName string) func(error) {
	m.CalculationsTotal.WithLabelValues(algorithmName).Inc()
	start := time.Now()

	return func(err error) {
		duration := time.Since(start).Seconds()
		if err != nil {
			errorType := "calculation_error"
			if err == context.Canceled {
				errorType = "canceled"
			} else if err == context.DeadlineExceeded {
				errorType = "timeout"
			}
			m.ErrorsTotal.WithLabelValues(algorithmName, errorType).Inc()
		} else {
			// Enregistrement de la durée uniquement pour les succès.
			m.CalculationDuration.WithLabelValues(algorithmName).Observe(duration)
		}
	}
}

// StartServer démarre un serveur HTTP pour exposer les métriques.
// Retourne le serveur HTTP pour permettre un arrêt gracieux (Shutdown).
func StartServer(port int) *http.Server {
	if port <= 0 {
		slog.Info("Metrics server disabled (port <= 0)")
		return nil
	}

	addr := fmt.Sprintf(":%d", port)
	slog.Info("Starting metrics server", "address", "http://localhost"+addr+"/metrics")

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
		// Configuration des timeouts pour la robustesse (SRE best practice).
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Metrics server failed", "error", err)
		}
	}()

	return server
}
