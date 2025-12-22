package server

import (
	"log"

	"time"

	"github.com/agbru/fibcalc/internal/logging"
	"github.com/agbru/fibcalc/internal/service"
)

// Option defines a functional option for configuring a Server.
type Option func(*Server)

// WithLogger sets a custom logger for the server using the unified logging interface.
// This is useful for testing or integrating with existing logging infrastructure.
//
// Parameters:
//   - logger: The logger to use. If nil, the default logger is used.
//
// Returns:
//   - Option: A functional option that configures the server's logger.
func WithLogger(logger logging.Logger) Option {
	return func(s *Server) {
		if logger != nil {
			s.logger = logger
		}
	}
}

// WithStdLogger sets a standard library log.Logger for the server.
// This provides backward compatibility with code using log.Logger.
//
// Parameters:
//   - logger: The standard log.Logger to use. If nil, the default logger is used.
//
// Returns:
//   - Option: A functional option that configures the server's logger.
func WithStdLogger(logger *log.Logger) Option {
	return func(s *Server) {
		if logger != nil {
			s.logger = logging.NewStdLoggerAdapter(logger)
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
//   - Option: A functional option that configures the server's service.
func WithService(svc service.Service) Option {
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
//   - Option: A functional option that configures the server's timeouts.
func WithTimeouts(timeouts Timeouts) Option {
	return func(s *Server) {
		s.timeouts = timeouts
	}
}

// Timeouts holds timeout configuration for the HTTP server.
// These can be customized via functional options for testing or deployment needs.
type Timeouts struct {
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

func DefaultServerTimeouts() Timeouts {
	return Timeouts{
		RequestTimeout:  5 * time.Minute,
		ShutdownTimeout: 30 * time.Second,
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    10 * time.Minute,
		IdleTimeout:     2 * time.Minute,
	}
}
