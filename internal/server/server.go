package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"example.com/fibcalc/internal/fibonacci"
)

type Server struct {
	Registry map[string]fibonacci.Calculator
}

type Response struct {
	N        uint64   `json:"n"`
	Result   *big.Int `json:"result"`
	Duration string   `json:"duration"`
	Error    string   `json:"error,omitempty"`
}

// Start initializes and starts the HTTP server with graceful shutdown support.
func (s *Server) Start(port string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/calculate", s.handleCalculate)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Minute, // Calculation can be long
		IdleTimeout:  15 * time.Second,
	}

	// Server run context
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	// Listen for syscall signals for process to interrupt/quit
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-sig

		// Shutdown signal with grace period of 30 seconds
		shutdownCtx, cancel := context.WithTimeout(serverCtx, 30*time.Second)
		defer cancel()

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				fmt.Println("graceful shutdown timed out.. forcing exit.")
			}
		}()

		// Trigger graceful shutdown
		err := srv.Shutdown(shutdownCtx)
		if err != nil {
			fmt.Printf("server shutdown error: %v\n", err)
		}
		serverStopCtx()
	}()

	fmt.Printf("Starting server on port %s...\n", port)
	err := srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// Wait for server context to be stopped
	<-serverCtx.Done()
	fmt.Println("Server exited properly")
	return nil
}

func (s *Server) handleCalculate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	nStr := r.URL.Query().Get("n")
	algo := r.URL.Query().Get("algo")

	if nStr == "" {
		http.Error(w, "Missing 'n' parameter", http.StatusBadRequest)
		return
	}

	n, err := strconv.ParseUint(nStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid 'n' parameter", http.StatusBadRequest)
		return
	}

	if algo == "" {
		algo = "fast"
	}

	calc, ok := s.Registry[algo]
	if !ok {
		http.Error(w, "Invalid 'algo' parameter", http.StatusBadRequest)
		return
	}

	// Create a context with timeout for the calculation
	// TODO: Make this timeout configurable via request or config
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	start := time.Now()
	// Use default thresholds for server mode for now
	result, err := calc.Calculate(ctx, nil, 0, n, 4096, 20000)
	duration := time.Since(start)

	resp := Response{
		N:        n,
		Result:   result,
		Duration: duration.String(),
	}

	if err != nil {
		resp.Error = err.Error()
		// If it's a client-side error (e.g. invalid input), 400.
		// If execution failed, 500.
		// Here we assume 500 mostly, unless context canceled.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			// Don't write specific status here as we might have already timed out writing?
			// Actually, we are building the response object, so we are fine.
			// But if the request context is cancelled, writing to w might fail.
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		// Log error if encoding fails
		fmt.Fprintf(os.Stderr, "Error encoding response: %v\n", err)
	}
}
