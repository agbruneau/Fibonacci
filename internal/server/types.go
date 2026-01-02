package server

import (
	"math/big"
)

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

// CalculateParseError represents a parameter parsing error with HTTP status.
type CalculateParseError struct {
	Message    string
	StatusCode int
}

// Error implements the error interface.
func (e CalculateParseError) Error() string {
	return e.Message
}
