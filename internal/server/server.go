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

	"example.com/fibcalc/internal/config"
	apperrors "example.com/fibcalc/internal/errors"
	"example.com/fibcalc/internal/fibonacci"
)

const (
	// DefaultRequestTimeout is the maximum duration for a single request
	DefaultRequestTimeout = 5 * time.Minute
	// DefaultShutdownTimeout is the maximum duration for graceful shutdown
	DefaultShutdownTimeout = 30 * time.Second
	// DefaultReadTimeout is the maximum duration for reading the request
	DefaultReadTimeout = 10 * time.Second
	// DefaultWriteTimeout is the maximum duration for writing the response
	DefaultWriteTimeout = 10 * time.Minute
	// DefaultIdleTimeout is the maximum duration for idle connections
	DefaultIdleTimeout = 2 * time.Minute
)

// Server represents the HTTP server for the Fibonacci calculator API.
type Server struct {
	registry       map[string]fibonacci.Calculator
	cfg            config.AppConfig
	httpServer     *http.Server
	logger         *log.Logger
	shutdownSignal chan os.Signal
}

// Response represents the JSON response for a calculation request.
type Response struct {
	N        uint64   `json:"n"`
	Result   *big.Int `json:"result,omitempty"`
	Duration string   `json:"duration"`
	Error    string   `json:"error,omitempty"`
	Algorithm string  `json:"algorithm"`
}

// ErrorResponse represents the JSON response for an error.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// NewServer creates a new server instance with the given calculator registry and configuration.
func NewServer(registry map[string]fibonacci.Calculator, cfg config.AppConfig) *Server {
	logger := log.New(os.Stdout, "[SERVER] ", log.LstdFlags)
	
	s := &Server{
		registry:       registry,
		cfg:            cfg,
		logger:         logger,
		shutdownSignal: make(chan os.Signal, 1),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/calculate", s.loggingMiddleware(s.handleCalculate))
	mux.HandleFunc("/health", s.loggingMiddleware(s.handleHealth))
	mux.HandleFunc("/algorithms", s.loggingMiddleware(s.handleAlgorithms))

	s.httpServer = &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  DefaultReadTimeout,
		WriteTimeout: DefaultWriteTimeout,
		IdleTimeout:  DefaultIdleTimeout,
	}

	return s
}

// Start starts the HTTP server with graceful shutdown support.
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

// loggingMiddleware logs incoming requests and their duration.
func (s *Server) loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		s.logger.Printf("%s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		
		next(w, r)
		
		duration := time.Since(start)
		s.logger.Printf("%s %s completed in %v", r.Method, r.URL.Path, duration)
	}
}

// handleHealth responds with the server health status.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	response := map[string]interface{}{
		"status": "healthy",
		"timestamp": time.Now().Unix(),
	}
	
	s.writeJSONResponse(w, http.StatusOK, response)
}

// handleAlgorithms returns the list of available algorithms.
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

// handleCalculate handles Fibonacci calculation requests.
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
	result, err := calc.Calculate(ctx, nil, 0, n, s.cfg.Threshold, s.cfg.FFTThreshold)
	duration := time.Since(start)

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

// writeJSONResponse writes a JSON response with the given status code.
func (s *Server) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Printf("Error encoding JSON response: %v", err)
	}
}

// writeErrorResponse writes an error response as JSON.
func (s *Server) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	errResp := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	}
	s.writeJSONResponse(w, statusCode, errResp)
}
