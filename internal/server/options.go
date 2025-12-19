package server

import (
	"log"
	"time"

	"github.com/agbru/fibcalc/internal/service"
)

// ServerOption defines a functional option for configuring a Server.
type ServerOption func(*Server)

// WithLogger sets a custom logger for the server.
// This is useful for testing or integrating with existing logging infrastructure.
//
// Parameters:
//   - logger: The logger to use. If nil, the default logger is used.
//
// Returns:
//   - ServerOption: A functional option that configures the server's logger.
func WithLogger(logger *log.Logger) ServerOption {
	return func(s *Server) {
		if logger != nil {
			s.logger = logger
		}
	}
}

// WithService sets a custom service for the server.
// This enables dependency injection for testing with mock services.
//
// Parameters:
//   - svc: The service implementation to use.
//
// Returns:
//   - ServerOption: A functional option that configures the server's service.
func WithService(svc service.Service) ServerOption {
	return func(s *Server) {
		if svc != nil {
			s.service = svc
		}
	}
}

// WithTimeouts sets custom timeout configuration for the server.
// This allows fine-tuning server behavior for different deployment scenarios.
//
// Parameters:
//   - timeouts: The timeout configuration.
//
// Returns:
//   - ServerOption: A functional option that configures the server's timeouts.
func WithTimeouts(timeouts ServerTimeouts) ServerOption {
	return func(s *Server) {
		s.timeouts = timeouts
	}
}

// ServerTimeouts holds timeout configuration for the HTTP server.
// These can be customized via functional options for testing or deployment needs.
type ServerTimeouts struct {
	// RequestTimeout is the maximum duration for a single request.
	RequestTimeout time.Duration
	// ShutdownTimeout is the maximum duration allowed for graceful shutdown.
	ShutdownTimeout time.Duration
	// ReadTimeout is the maximum duration for reading the entire request, including the body.
	ReadTimeout time.Duration
	// WriteTimeout is the maximum duration before timing out writes of the response.
	WriteTimeout time.Duration
	// IdleTimeout is the maximum amount of time to wait for the next request when keep-alives are enabled.
	IdleTimeout time.Duration
}

// DefaultServerTimeouts returns the default timeout configuration.
func DefaultServerTimeouts() ServerTimeouts {
	return ServerTimeouts{
		RequestTimeout:  5 * time.Minute,
		ShutdownTimeout: 30 * time.Second,
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    10 * time.Minute,
		IdleTimeout:     2 * time.Minute,
	}
}
