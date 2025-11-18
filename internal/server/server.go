package server

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
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

func (s *Server) Start(port string) error {
	http.HandleFunc("/calculate", s.handleCalculate)
	addr := ":" + port
	fmt.Printf("Starting server on port %s...\n", port)
	return http.ListenAndServe(addr, nil)
}

func (s *Server) handleCalculate(w http.ResponseWriter, r *http.Request) {
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

	ctx := context.Background()
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
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
