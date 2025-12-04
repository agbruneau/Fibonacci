package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/agbru/fibcalc/internal/config"
	apperrors "github.com/agbru/fibcalc/internal/errors"
	"github.com/agbru/fibcalc/internal/fibonacci"
)

const (
	// DefaultRequestTimeout is the maximum duration for a single request.
	DefaultRequestTimeout = 5 * time.Minute
	// DefaultShutdownTimeout is the maximum duration allowed for graceful shutdown.
	DefaultShutdownTimeout = 30 * time.Second
	// DefaultReadTimeout is the maximum duration for reading the entire request, including the body.
	DefaultReadTimeout = 10 * time.Second
	// DefaultWriteTimeout is the maximum duration before timing out writes of the response.
	DefaultWriteTimeout = 10 * time.Minute
	// DefaultIdleTimeout is the maximum amount of time to wait for the next request when keep-alives are enabled.
	DefaultIdleTimeout = 2 * time.Minute
)

// Server represents the HTTP server for the Fibonacci calculator API.
// It wraps the standard http.Server and adds application-specific configuration
// and graceful shutdown capabilities.
type Server struct {
	registry       map[string]fibonacci.Calculator
	cfg            config.AppConfig
	httpServer     *http.Server
	logger         *log.Logger
	shutdownSignal chan os.Signal
	rateLimiter    *RateLimiter
	securityConfig SecurityConfig
	metrics        *Metrics
}

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

// Response represents the standardized JSON response for a calculation request.
type Response struct {
	// N is the index of the Fibonacci number requested.
	N uint64 `json:"n"`
	// Result is the calculated Fibonacci number. It is omitted if an error occurred.
	Result *big.Int `json:"result,omitempty"`
	// Duration is the formatted execution time string.
	Duration string `json:"duration"`
	// Error contains the error message if the calculation failed.
	Error string `json:"error,omitempty"`
	// Algorithm is the name of the algorithm used for the calculation.
	Algorithm string `json:"algorithm"`
}

// ErrorResponse represents the standardized JSON response for an API error.
type ErrorResponse struct {
	// Error is the short error code or status text.
	Error string `json:"error"`
	// Message is a descriptive error message.
	Message string `json:"message,omitempty"`
}

// NewServer creates a new Server instance with the given calculator registry and configuration.
// It initializes the HTTP server with timeouts and a request multiplexer.
//
// Parameters:
//   - registry: A map of algorithm names to their calculator implementations.
//   - cfg: The application configuration (port, thresholds, etc.).
//   - opts: Optional functional options for customizing the server (e.g., WithLogger).
//
// Returns:
//   - *Server: A pointer to the initialized Server.
func NewServer(registry map[string]fibonacci.Calculator, cfg config.AppConfig, opts ...ServerOption) *Server {
	s := &Server{
		registry:       registry,
		cfg:            cfg,
		logger:         log.New(os.Stdout, "[SERVER] ", log.LstdFlags), // Default logger
		shutdownSignal: make(chan os.Signal, 1),
		securityConfig: DefaultSecurityConfig(),
		metrics:        NewMetrics(),
	}

	// Apply any provided options
	for _, opt := range opts {
		opt(s)
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
		ReadTimeout:  DefaultReadTimeout,
		WriteTimeout: DefaultWriteTimeout,
		IdleTimeout:  DefaultIdleTimeout,
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

	// Start the server in a goroutine
	go func() {
		s.logger.Printf("Starting server on %s\n", s.httpServer.Addr)
		s.logger.Printf("Configuration: threshold=%d, fft_threshold=%d, strassen_threshold=%d\n",
			s.cfg.Threshold, s.cfg.FFTThreshold, s.cfg.StrassenThreshold)
		s.logger.Println("Available endpoints:")
		s.logger.Println("  GET /calculate?n=<number>&algo=<algorithm>")
		s.logger.Println("  GET /health")
		s.logger.Println("  GET /algorithms")

		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatalf("Server error: %v\n", err)
		}
	}()

	// Wait for interrupt signal
	<-s.shutdownSignal
	s.logger.Println("Shutdown signal received, initiating graceful shutdown...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), DefaultShutdownTimeout)
	defer cancel()

	// Attempt graceful shutdown
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return apperrors.NewServerError("failed to gracefully shutdown server", err)
	}

	s.logger.Println("Server stopped gracefully")
	return nil
}

// loggingMiddleware wraps an http.HandlerFunc to log the details of each request.
// It records the HTTP method, URL path, remote address, and the duration required
// to process the request.
//
// Parameters:
//   - next: The next handler in the chain.
//
// Returns:
//   - http.HandlerFunc: A new handler with logging capability.
func (s *Server) loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		s.logger.Printf("%s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		next(w, r)

		duration := time.Since(start)
		s.logger.Printf("%s %s completed in %v", r.Method, r.URL.Path, duration)
	}
}

// handleHealth responds to health check requests.
// It returns a 200 OK status with a JSON payload indicating the service is healthy.
//
// Parameters:
//   - w: The HTTP response writer.
//   - r: The HTTP request.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	}

	s.writeJSONResponse(w, http.StatusOK, response)
}

// handleAlgorithms returns the list of available Fibonacci calculation algorithms.
// It queries the internal registry and returns the keys as a JSON array.
//
// Parameters:
//   - w: The HTTP response writer.
//   - r: The HTTP request.
func (s *Server) handleAlgorithms(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	algorithms := make([]string, 0, len(s.registry))
	for name := range s.registry {
		algorithms = append(algorithms, name)
	}

	response := map[string]interface{}{
		"algorithms": algorithms,
	}

	s.writeJSONResponse(w, http.StatusOK, response)
}

// handleCalculate processes requests to calculate Fibonacci numbers.
// It parses the query parameters 'n' (the index) and 'algo' (the algorithm),
// executes the calculation, and returns the result in JSON format.
//
// Parameters:
//   - w: The HTTP response writer.
//   - r: The HTTP request.
func (s *Server) handleCalculate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse and validate parameters
	nStr := r.URL.Query().Get("n")
	if nStr == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "Missing 'n' parameter")
		return
	}

	n, err := strconv.ParseUint(nStr, 10, 64)
	if err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "Invalid 'n' parameter: must be a positive integer")
		return
	}

	// Validate n against maximum allowed value (DoS protection)
	if s.securityConfig.MaxNValue > 0 && n > s.securityConfig.MaxNValue {
		s.writeErrorResponse(w, http.StatusBadRequest,
			fmt.Sprintf("Value of 'n' exceeds maximum allowed (%d). This limit prevents resource exhaustion.", s.securityConfig.MaxNValue))
		return
	}

	algo := r.URL.Query().Get("algo")
	if algo == "" {
		algo = "fast" // Default algorithm
	}

	calc, ok := s.registry[algo]
	if !ok {
		s.writeErrorResponse(w, http.StatusBadRequest,
			fmt.Sprintf("Invalid 'algo' parameter: '%s' is not a valid algorithm", algo))
		return
	}

	// Create a context with timeout for the calculation
	ctx, cancel := context.WithTimeout(r.Context(), DefaultRequestTimeout)
	defer cancel()

	// Perform the calculation
	start := time.Now()
	result, err := calc.Calculate(ctx, nil, 0, n, fibonacci.Options{ParallelThreshold: s.cfg.Threshold, FFTThreshold: s.cfg.FFTThreshold, StrassenThreshold: s.cfg.StrassenThreshold})
	duration := time.Since(start)

	// Record metrics
	status := "success"
	if err != nil {
		status = "error"
	}
	s.metrics.RecordCalculation(algo, status, duration)

	// Build the response
	resp := Response{
		N:         n,
		Duration:  duration.String(),
		Algorithm: algo,
	}

	if err != nil {
		resp.Error = err.Error()
		// Still return 200 OK with error in the JSON body for consistency
	} else {
		resp.Result = result
	}

	s.writeJSONResponse(w, http.StatusOK, resp)
}

// writeJSONResponse helper function to write a JSON response with the correct content type.
//
// Parameters:
//   - w: The HTTP response writer.
//   - statusCode: The HTTP status code to write.
//   - data: The data to be encoded as JSON.
func (s *Server) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Printf("Error encoding JSON response: %v", err)
	}
}

// writeErrorResponse helper function to write a standardized error response.
//
// Parameters:
//   - w: The HTTP response writer.
//   - statusCode: The HTTP status code to write.
//   - message: The error message to be included in the response body.
func (s *Server) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	errResp := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	}
	s.writeJSONResponse(w, statusCode, errResp)
}
