package server

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/agbru/fibcalc/internal/config"
	apperrors "github.com/agbru/fibcalc/internal/errors"
	"github.com/agbru/fibcalc/internal/fibonacci"
	"github.com/agbru/fibcalc/internal/logging"
	"github.com/agbru/fibcalc/internal/service"
)

// Server represents the HTTP server for the Fibonacci calculator API.
// It wraps the standard http.Server and adds application-specific configuration
// and graceful shutdown capabilities.
type Server struct {
	factory        fibonacci.CalculatorFactory
	service        service.Service
	cfg            config.AppConfig
	httpServer     *http.Server
	logger         logging.Logger
	shutdownSignal chan os.Signal
	rateLimiter    *RateLimiter
	securityConfig SecurityConfig
	metrics        *Metrics
	timeouts       Timeouts
}

// NewServer creates a new Server instance with the given calculator registry and configuration.
// It initializes the HTTP server with timeouts and a request multiplexer.
//
// Parameters:
//   - factory: The calculator factory to retrieve implementations from.
//   - cfg: The application configuration (port, thresholds, etc.).
//   - opts: Optional functional options for customizing the server (e.g., WithLogger).
//
// Returns:
//   - *Server: A pointer to the initialized Server.
func NewServer(factory fibonacci.CalculatorFactory, cfg config.AppConfig, opts ...Option) *Server {
	s := &Server{
		factory:        factory,
		cfg:            cfg,
		logger:         logging.NewLogger(os.Stdout, "server"), // Default unified logger
		shutdownSignal: make(chan os.Signal, 1),
		securityConfig: DefaultSecurityConfig(),
		metrics:        NewMetrics(),
		timeouts:       DefaultServerTimeouts(),
	}

	// Apply any provided options
	for _, opt := range opts {
		opt(s)
	}

	// Initialize service if not provided
	if s.service == nil {
		s.service = service.NewCalculatorService(s.factory, s.cfg, s.securityConfig.MaxNValue)
	}

	// Create default rate limiter if not provided
	if s.rateLimiter == nil {
		s.rateLimiter = NewRateLimiter(DefaultRateLimiterConfig())
	}

	mux := http.NewServeMux()

	// Apply middleware chain: Security -> RateLimit -> Logging -> Metrics -> Handler
	mux.HandleFunc("/calculate", s.wrapWithMiddleware(s.handleCalculate))
	mux.HandleFunc("/health", s.wrapWithMiddleware(s.handleHealth))
	mux.HandleFunc("/algorithms", s.wrapWithMiddleware(s.handleAlgorithms))
	mux.HandleFunc("/metrics", s.wrapWithMiddleware(s.handleMetrics))

	s.httpServer = &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  s.timeouts.ReadTimeout,
		WriteTimeout: s.timeouts.WriteTimeout,
		IdleTimeout:  s.timeouts.IdleTimeout,
	}

	return s
}

// wrapWithMiddleware applies the full middleware chain to a handler.
func (s *Server) wrapWithMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	// Apply in reverse order: Security -> RateLimit -> Logging -> Metrics -> Handler
	wrapped := s.metricsMiddleware(handler)
	wrapped = s.loggingMiddleware(wrapped)
	wrapped = RateLimitMiddleware(s.rateLimiter, wrapped)
	wrapped = SecurityMiddleware(s.securityConfig, wrapped)
	return wrapped
}

// Start initializes and starts the HTTP server.
// It listens for incoming requests on the configured port and handles system
// signals (SIGINT, SIGTERM) to ensure a graceful shutdown.
//
// Returns:
//   - error: An error if the server fails to start or shuts down unexpectedly.
func (s *Server) Start() error {
	// Setup signal handling for graceful shutdown
	signal.Notify(s.shutdownSignal, os.Interrupt, syscall.SIGTERM)

	// Channel for server startup errors
	errCh := make(chan error, 1)

	// Start the server in a goroutine
	go func() {
		s.logger.Printf("Starting server on %s\n", s.httpServer.Addr)
		s.logger.Printf("Configuration: threshold=%d, fft_threshold=%d, strassen_threshold=%d\n",
			s.cfg.Threshold, s.cfg.FFTThreshold, s.cfg.StrassenThreshold)
		s.logger.Println("Available endpoints:")
		s.logger.Println("  GET /calculate?n=<number>&algo=<algorithm>")
		s.logger.Println("  GET /health")
		s.logger.Println("  GET /algorithms")

		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case <-s.shutdownSignal:
		s.logger.Println("Shutdown signal received, initiating graceful shutdown...")
	case err := <-errCh:
		return apperrors.NewServerError("server failed to start", err)
	}

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), s.timeouts.ShutdownTimeout)
	defer cancel()

	// Attempt graceful shutdown
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return apperrors.NewServerError("failed to gracefully shutdown server", err)
	}

	s.logger.Println("Server stopped gracefully")
	return nil
}
