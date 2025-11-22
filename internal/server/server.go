package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"example.com/fibcalc/internal/config"
	"example.com/fibcalc/internal/fibonacci"
)

const (
	defaultRequestTimeout    = 30 * time.Second
	defaultReadHeaderTimeout = 5 * time.Second
	defaultWriteTimeout      = defaultRequestTimeout + 5*time.Second
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

func (s *Server) Start(port string) error {
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/calculate", s.handleCalculate)

	httpServer := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		WriteTimeout:      defaultWriteTimeout,
	}

	fmt.Printf("Starting server on port %s...\n", port)
	return httpServer.ListenAndServe()
}

func (s *Server) handleCalculate(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	nStr := query.Get("n")
	algo := query.Get("algo")

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

	timeout := defaultRequestTimeout
	if timeoutStr := query.Get("timeout"); timeoutStr != "" {
		parsed, err := time.ParseDuration(timeoutStr)
		if err != nil || parsed <= 0 {
			http.Error(w, "Invalid 'timeout' parameter", http.StatusBadRequest)
			return
		}
		timeout = parsed
	}

	threshold := config.DefaultParallelThreshold
	if thresholdStr := query.Get("threshold"); thresholdStr != "" {
		val, err := strconv.Atoi(thresholdStr)
		if err != nil || val < 0 {
			http.Error(w, "Invalid 'threshold' parameter", http.StatusBadRequest)
			return
		}
		threshold = val
	}

	fftThreshold := config.DefaultFFTThreshold
	if fftStr := query.Get("fft-threshold"); fftStr != "" {
		val, err := strconv.Atoi(fftStr)
		if err != nil || val < 0 {
			http.Error(w, "Invalid 'fft-threshold' parameter", http.StatusBadRequest)
			return
		}
		fftThreshold = val
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	start := time.Now()
	result, err := calc.Calculate(ctx, nil, 0, n, threshold, fftThreshold)
	duration := time.Since(start)

	resp := Response{
		N:        n,
		Result:   result,
		Duration: duration.String(),
	}

	statusCode := http.StatusOK
	if err != nil {
		resp.Error = err.Error()
		switch {
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			statusCode = http.StatusGatewayTimeout
		default:
			statusCode = http.StatusInternalServerError
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if encodeErr := json.NewEncoder(w).Encode(resp); encodeErr != nil {
		fmt.Printf("server: failed to encode response: %v\n", encodeErr)
	}
}
