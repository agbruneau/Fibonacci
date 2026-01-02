package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/agbru/fibcalc/internal/config"
	"github.com/agbru/fibcalc/internal/fibonacci"
)

// MockCalculator is a mock implementation of fibonacci.Calculator for testing.
type MockCalculator struct {
	Result *big.Int
	Err    error
	// CapturedOpts stores the options passed to Calculate for verification.
	CapturedOpts fibonacci.Options
}

// Name returns the mock calculator's name.
func (m *MockCalculator) Name() string {
	return "Mock"
}

// Calculate implements the fibonacci.Calculator interface returning predefined results.
func (m *MockCalculator) Calculate(ctx context.Context, progressChan chan<- fibonacci.ProgressUpdate, calcIndex int, n uint64, opts fibonacci.Options) (*big.Int, error) {
	m.CapturedOpts = opts
	return m.Result, m.Err
}

// createTestServer initializes a server instance for testing with default configuration.
func createTestServer(registry map[string]fibonacci.Calculator) *Server {
	cfg := config.AppConfig{
		Port:              "8080",
		Threshold:         4096,
		FFTThreshold:      20000,
		StrassenThreshold: 256,
	}
	return NewServer(fibonacci.NewTestFactory(registry), cfg)
}

// TestHandleCalculate verifies the behavior of the calculation endpoint.
// It tests successful calculations, validation errors, and calculation failures.
func TestHandleCalculate(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		mockResult     *big.Int
		mockErr        error
		expectedStatus int
		expectedBody   string
		isJSON         bool
		checkError     bool
	}{
		{
			name:           "Success",
			queryParams:    "?n=10",
			mockResult:     big.NewInt(55),
			mockErr:        nil,
			expectedStatus: http.StatusOK,
			expectedBody:   `55`,
			isJSON:         true,
			checkError:     false,
		},
		{
			name:           "Missing n",
			queryParams:    "",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing 'n' parameter",
			isJSON:         true,
			checkError:     true,
		},
		{
			name:           "Invalid n",
			queryParams:    "?n=abc",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "must be a positive integer",
			isJSON:         true,
			checkError:     true,
		},
		{
			name:           "Unknown algorithm",
			queryParams:    "?n=10&algo=unknown",
			expectedStatus: http.StatusOK, // Server returns 200 with error in JSON body
			expectedBody:   "unknown calculator",
			isJSON:         true,
			checkError:     true,
		},
		{
			name:           "Calculation error",
			queryParams:    "?n=10",
			mockResult:     nil,
			mockErr:        errors.New("calc error"),
			expectedStatus: http.StatusOK,
			expectedBody:   "calc error",
			isJSON:         true,
			checkError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCalc := &MockCalculator{
				Result: tt.mockResult,
				Err:    tt.mockErr,
			}
			registry := map[string]fibonacci.Calculator{
				"fast": mockCalc,
			}
			server := createTestServer(registry)

			req := httptest.NewRequest("GET", "/calculate"+tt.queryParams, http.NoBody)
			w := httptest.NewRecorder()

			server.handleCalculate(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			bodyBytes, _ := io.ReadAll(resp.Body)

			if tt.isJSON {
				if tt.checkError {
					// For error responses
					if tt.expectedStatus != http.StatusOK {
						var errResp ErrorResponse
						if err := json.Unmarshal(bodyBytes, &errResp); err != nil {
							t.Errorf("Failed to unmarshal error response: %v", err)
						}
						if !strings.Contains(errResp.Message, tt.expectedBody) {
							t.Errorf("Expected error message to contain %q, got %q", tt.expectedBody, errResp.Message)
						}
					} else {
						// For calculation errors (200 OK but with error field)
						var jsonResp Response
						if err := json.Unmarshal(bodyBytes, &jsonResp); err != nil {
							t.Errorf("Failed to unmarshal JSON response: %v", err)
						}
						if !strings.Contains(jsonResp.Error, tt.expectedBody) {
							t.Errorf("Expected error message to contain %q, got %q", tt.expectedBody, jsonResp.Error)
						}
					}
				} else {
					// For success responses
					var jsonResp Response
					if err := json.Unmarshal(bodyBytes, &jsonResp); err != nil {
						t.Errorf("Failed to unmarshal JSON response: %v", err)
					}
					if jsonResp.Result.Cmp(big.NewInt(55)) != 0 {
						t.Errorf("Expected result 55, got %s", jsonResp.Result.String())
					}
					if jsonResp.N != 10 {
						t.Errorf("Expected n=10, got n=%d", jsonResp.N)
					}
					if jsonResp.Algorithm != "fast" {
						t.Errorf("Expected algorithm=fast, got algorithm=%s", jsonResp.Algorithm)
					}
				}
			}
		})
	}
}

// TestHandleHealth verifies the health check endpoint.
func TestHandleHealth(t *testing.T) {
	server := createTestServer(nil)

	req := httptest.NewRequest("GET", "/health", http.NoBody)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var healthResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		t.Errorf("Failed to decode health response: %v", err)
	}

	if healthResp["status"] != "healthy" {
		t.Errorf("Expected status=healthy, got %v", healthResp["status"])
	}
}

// TestHandleAlgorithms verifies the algorithms listing endpoint.
func TestHandleAlgorithms(t *testing.T) {
	mockCalc := &MockCalculator{Result: big.NewInt(1)}
	registry := map[string]fibonacci.Calculator{
		"fast":   mockCalc,
		"matrix": mockCalc,
		"fft":    mockCalc,
	}
	server := createTestServer(registry)

	req := httptest.NewRequest("GET", "/algorithms", http.NoBody)
	w := httptest.NewRecorder()

	server.handleAlgorithms(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var algoResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&algoResp); err != nil {
		t.Errorf("Failed to decode algorithms response: %v", err)
	}

	algos, ok := algoResp["algorithms"].([]interface{})
	if !ok {
		t.Fatal("Expected algorithms to be an array")
	}

	if len(algos) != 3 {
		t.Errorf("Expected 3 algorithms, got %d", len(algos))
	}
}

// TestMethodNotAllowed verifies that non-GET methods are rejected.
func TestMethodNotAllowed(t *testing.T) {
	server := createTestServer(nil)

	tests := []struct {
		name     string
		endpoint string
		method   string
	}{
		{"Calculate POST", "/calculate", "POST"},
		{"Health POST", "/health", "POST"},
		{"Algorithms POST", "/algorithms", "POST"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.endpoint, http.NoBody)
			w := httptest.NewRecorder()

			switch tt.endpoint {
			case "/calculate":
				server.handleCalculate(w, req)
			case "/health":
				server.handleHealth(w, req)
			case "/algorithms":
				server.handleAlgorithms(w, req)
			}

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405, got %d", resp.StatusCode)
			}
		})
	}
}

// TestLoggingMiddleware verifies that the logging middleware executes the next handler.
func TestLoggingMiddleware(t *testing.T) {
	server := createTestServer(nil)

	handlerCalled := false
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}

	wrapped := server.loggingMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()

	// Give the logger a bit of time
	done := make(chan bool)
	go func() {
		wrapped(w, req)
		done <- true
	}()

	select {
	case <-done:
		if !handlerCalled {
			t.Error("Handler was not called")
		}
	case <-time.After(1 * time.Second):
		t.Error("Middleware timed out")
	}
}

// TestStrassenThresholdPassedToCalculator verifies that the StrassenThreshold
// configuration is correctly passed to the calculator in API requests.
// This test was added to verify the fix for a bug where StrassenThreshold
// was configured but not used in server calculations.
func TestStrassenThresholdPassedToCalculator(t *testing.T) {
	mockCalc := &MockCalculator{
		Result: big.NewInt(55),
		Err:    nil,
	}
	registry := map[string]fibonacci.Calculator{
		"fast": mockCalc,
	}

	// Create server with specific threshold values
	cfg := config.AppConfig{
		Port:              "8080",
		Threshold:         1234,
		FFTThreshold:      5678,
		StrassenThreshold: 9999, // Specific value to verify
	}
	server := NewServer(fibonacci.NewTestFactory(registry), cfg)

	req := httptest.NewRequest("GET", "/calculate?n=10", http.NoBody)
	w := httptest.NewRecorder()

	server.handleCalculate(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify that the StrassenThreshold was passed correctly
	if mockCalc.CapturedOpts.StrassenThreshold != 9999 {
		t.Errorf("Expected StrassenThreshold=9999, got %d", mockCalc.CapturedOpts.StrassenThreshold)
	}

	// Also verify the other thresholds are passed correctly
	if mockCalc.CapturedOpts.ParallelThreshold != 1234 {
		t.Errorf("Expected ParallelThreshold=1234, got %d", mockCalc.CapturedOpts.ParallelThreshold)
	}

	if mockCalc.CapturedOpts.FFTThreshold != 5678 {
		t.Errorf("Expected FFTThreshold=5678, got %d", mockCalc.CapturedOpts.FFTThreshold)
	}
}

// TestParseCalculateParams verifies the parameter parsing helper function.
func TestParseCalculateParams(t *testing.T) {
	tests := []struct {
		name          string
		queryParams   string
		expectedN     uint64
		expectedAlgo  string
		expectedError bool
		errorMessage  string
	}{
		{
			name:          "Valid n with default algo",
			queryParams:   "?n=42",
			expectedN:     42,
			expectedAlgo:  "fast",
			expectedError: false,
		},
		{
			name:          "Valid n with specified algo",
			queryParams:   "?n=100&algo=matrix",
			expectedN:     100,
			expectedAlgo:  "matrix",
			expectedError: false,
		},
		{
			name:          "Missing n parameter",
			queryParams:   "",
			expectedError: true,
			errorMessage:  "Missing 'n' parameter",
		},
		{
			name:          "Missing n with algo only",
			queryParams:   "?algo=fast",
			expectedError: true,
			errorMessage:  "Missing 'n' parameter",
		},
		{
			name:          "Invalid n - non-numeric",
			queryParams:   "?n=abc",
			expectedError: true,
			errorMessage:  "must be a positive integer",
		},
		{
			name:          "Invalid n - negative",
			queryParams:   "?n=-5",
			expectedError: true,
			errorMessage:  "must be a positive integer",
		},
		{
			name:          "Large valid n",
			queryParams:   "?n=18446744073709551615", // Max uint64
			expectedN:     18446744073709551615,
			expectedAlgo:  "fast",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/calculate"+tt.queryParams, http.NoBody)
			n, algo, err := parseCalculateParams(req)

			if tt.expectedError {
				if err == nil {
					t.Error("Expected error, got nil")
					return
				}
				parseErr, ok := err.(CalculateParseError)
				if !ok {
					t.Errorf("Expected CalculateParseError, got %T", err)
					return
				}
				if !strings.Contains(parseErr.Message, tt.errorMessage) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errorMessage, parseErr.Message)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if n != tt.expectedN {
					t.Errorf("Expected n=%d, got n=%d", tt.expectedN, n)
				}
				if algo != tt.expectedAlgo {
					t.Errorf("Expected algo=%s, got algo=%s", tt.expectedAlgo, algo)
				}
			}
		})
	}
}

// TestWithLogger verifies the WithLogger option.
func TestWithLogger(t *testing.T) {
	registry := map[string]fibonacci.Calculator{}
	cfg := config.AppConfig{Port: "8080"}

	// Test with nil logger (should not change default)
	server := NewServer(fibonacci.NewTestFactory(registry), cfg, WithLogger(nil))
	if server.logger == nil {
		t.Error("expected default logger to be set")
	}

	// Test with custom standard logger using WithStdLogger
	customLogger := log.New(io.Discard, "[CUSTOM] ", 0)
	server = NewServer(fibonacci.NewTestFactory(registry), cfg, WithStdLogger(customLogger))
	if server.logger == nil {
		t.Error("expected custom logger to be set")
	}
}

// TestWithService verifies the WithService option.
func TestWithService(t *testing.T) {
	registry := map[string]fibonacci.Calculator{}
	cfg := config.AppConfig{Port: "8080"}

	// Test with nil service (should use default)
	server := NewServer(fibonacci.NewTestFactory(registry), cfg, WithService(nil))
	if server.service == nil {
		t.Error("expected default service to be initialized")
	}

	// Test with custom service
	customService := &mockService{result: big.NewInt(123)}
	server = NewServer(fibonacci.NewTestFactory(registry), cfg, WithService(customService))
	if server.service != customService {
		t.Error("expected custom service to be set")
	}
}

// TestWithTimeouts verifies the WithTimeouts option.
func TestWithTimeouts(t *testing.T) {
	registry := map[string]fibonacci.Calculator{}
	cfg := config.AppConfig{Port: "8080"}

	customTimeouts := Timeouts{
		RequestTimeout:  10 * time.Minute,
		ShutdownTimeout: 60 * time.Second,
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    15 * time.Minute,
		IdleTimeout:     5 * time.Minute,
	}

	server := NewServer(fibonacci.NewTestFactory(registry), cfg, WithTimeouts(customTimeouts))
	if server.timeouts.RequestTimeout != customTimeouts.RequestTimeout {
		t.Errorf("expected RequestTimeout=%v, got %v", customTimeouts.RequestTimeout, server.timeouts.RequestTimeout)
	}
	if server.timeouts.ShutdownTimeout != customTimeouts.ShutdownTimeout {
		t.Errorf("expected ShutdownTimeout=%v, got %v", customTimeouts.ShutdownTimeout, server.timeouts.ShutdownTimeout)
	}
	if server.httpServer.ReadTimeout != customTimeouts.ReadTimeout {
		t.Errorf("expected ReadTimeout=%v, got %v", customTimeouts.ReadTimeout, server.httpServer.ReadTimeout)
	}
}

// TestWithMaxN verifies the WithMaxN option.
func TestWithMaxN(t *testing.T) {
	registry := map[string]fibonacci.Calculator{
		"fast": &MockCalculator{Result: big.NewInt(55)},
	}
	cfg := config.AppConfig{Port: "8080"}

	server := NewServer(fibonacci.NewTestFactory(registry), cfg, WithMaxN(1000))
	if server.securityConfig.MaxNValue != 1000 {
		t.Errorf("expected MaxN=1000, got %d", server.securityConfig.MaxNValue)
	}
}

// TestCalculateParseErrorMessage verifies the CalculateParseError.Error() method.
func TestCalculateParseErrorMessage(t *testing.T) {
	err := CalculateParseError{
		Message:    "test error message",
		StatusCode: http.StatusBadRequest,
	}

	if err.Error() != "test error message" {
		t.Errorf("expected 'test error message', got '%s'", err.Error())
	}
}

// mockService implements service.Service for testing.
type mockService struct {
	result *big.Int
	err    error
}

func (m *mockService) Calculate(ctx context.Context, algoName string, n uint64) (*big.Int, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

// TestBuildCalculateResponse verifies the response building helper function.
func TestBuildCalculateResponse(t *testing.T) {
	tests := []struct {
		name           string
		n              uint64
		algo           string
		result         *big.Int
		duration       time.Duration
		err            error
		hasResult      bool
		hasError       bool
		expectedResult int64
		expectedError  string
	}{
		{
			name:           "Successful calculation",
			n:              10,
			algo:           "fast",
			result:         big.NewInt(55),
			duration:       100 * time.Millisecond,
			err:            nil,
			hasResult:      true,
			hasError:       false,
			expectedResult: 55,
		},
		{
			name:          "Calculation with error",
			n:             999,
			algo:          "matrix",
			result:        nil,
			duration:      50 * time.Millisecond,
			err:           errors.New("calculation failed"),
			hasResult:     false,
			hasError:      true,
			expectedError: "calculation failed",
		},
		{
			name:           "Zero result",
			n:              0,
			algo:           "fast",
			result:         big.NewInt(0),
			duration:       1 * time.Nanosecond,
			err:            nil,
			hasResult:      true,
			hasError:       false,
			expectedResult: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := buildCalculateResponse(tt.n, tt.algo, tt.result, tt.duration, tt.err)

			if resp.N != tt.n {
				t.Errorf("Expected N=%d, got N=%d", tt.n, resp.N)
			}
			if resp.Algorithm != tt.algo {
				t.Errorf("Expected Algorithm=%s, got Algorithm=%s", tt.algo, resp.Algorithm)
			}
			if resp.Duration != tt.duration.String() {
				t.Errorf("Expected Duration=%s, got Duration=%s", tt.duration.String(), resp.Duration)
			}

			if tt.hasResult {
				if resp.Result == nil {
					t.Error("Expected Result to be set, got nil")
				} else if resp.Result.Cmp(big.NewInt(tt.expectedResult)) != 0 {
					t.Errorf("Expected Result=%d, got Result=%s", tt.expectedResult, resp.Result.String())
				}
			} else {
				if resp.Result != nil {
					t.Errorf("Expected Result to be nil, got %s", resp.Result.String())
				}
			}

			if tt.hasError {
				if resp.Error != tt.expectedError {
					t.Errorf("Expected Error=%q, got Error=%q", tt.expectedError, resp.Error)
				}
			} else {
				if resp.Error != "" {
					t.Errorf("Expected no Error, got Error=%q", resp.Error)
				}
			}
		})
	}
}
