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

	"github.com/agbru/fibcalc/internal/service"
)

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

	response := map[string]any{
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

	algorithms := s.factory.List()

	response := map[string]any{
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

	// Parse and validate parameters using helper
	n, algo, err := parseCalculateParams(r)
	if err != nil {
		if parseErr, ok := err.(CalculateParseError); ok {
			s.writeErrorResponse(w, parseErr.StatusCode, parseErr.Message)
		} else {
			s.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	// Create a context with timeout for the calculation
	ctx, cancel := context.WithTimeout(r.Context(), s.timeouts.RequestTimeout)
	defer cancel()

	// Perform the calculation
	start := time.Now()
	result, err := s.service.Calculate(ctx, algo, n)
	duration := time.Since(start)

	// Handle max value exceeded error
	if errors.Is(err, service.ErrMaxValueExceeded) {
		s.writeErrorResponse(w, http.StatusBadRequest,
			fmt.Sprintf("Value of 'n' exceeds maximum allowed (%d). This limit prevents resource exhaustion.", s.securityConfig.MaxNValue))
		return
	}

	// Build and send response using helper
	resp := buildCalculateResponse(n, algo, result, duration, err)
	s.writeJSONResponse(w, http.StatusOK, resp)
}

// parseCalculateParams extracts and validates the calculation parameters from the request.
//
// Parameters:
//   - r: The HTTP request containing query parameters.
//
// Returns:
//   - n: The parsed Fibonacci index.
//   - algo: The algorithm name (defaults to "fast" if not specified).
//   - err: A CalculateParseError if validation fails, nil otherwise.
func parseCalculateParams(r *http.Request) (n uint64, algo string, err error) {
	nStr := r.URL.Query().Get("n")
	if nStr == "" {
		return 0, "", CalculateParseError{
			Message:    "Missing 'n' parameter",
			StatusCode: http.StatusBadRequest,
		}
	}

	n, parseErr := strconv.ParseUint(nStr, 10, 64)
	if parseErr != nil {
		// strconv.ParseUint will return an error if the input has a negative sign,
		// effectively enforcing non-negative inputs as required for security.
		return 0, "", CalculateParseError{
			Message:    "Invalid 'n' parameter: must be a positive integer",
			StatusCode: http.StatusBadRequest,
		}
	}

	algo = r.URL.Query().Get("algo")
	if algo == "" {
		algo = "fast" // Default algorithm
	}

	return n, algo, nil
}

// buildCalculateResponse constructs the response struct for a calculation.
//
// Parameters:
//   - n: The Fibonacci index that was calculated.
//   - algo: The algorithm name used.
//   - result: The calculation result (may be nil if error occurred).
//   - duration: The time taken for the calculation.
//   - err: Any error that occurred during calculation.
//
// Returns:
//   - Response: The constructed response struct.
func buildCalculateResponse(n uint64, algo string, result *big.Int, duration time.Duration, err error) Response {
	resp := Response{
		N:         n,
		Duration:  duration.String(),
		Algorithm: algo,
	}

	if err != nil {
		resp.Error = err.Error()
	} else {
		resp.Result = result
	}

	return resp
}

// writeJSONResponse helper function to write a JSON response with the correct content type.
//
// Parameters:
//   - w: The HTTP response writer.
//   - statusCode: The HTTP status code to write.
//   - data: The data to be encoded as JSON.
func (s *Server) writeJSONResponse(w http.ResponseWriter, statusCode int, data any) {
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
