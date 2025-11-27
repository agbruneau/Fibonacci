package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"example.com/fibcalc/internal/config"
	"example.com/fibcalc/internal/fibonacci"
)

// MockCalculator is a mock implementation of fibonacci.Calculator for testing.
type MockCalculator struct {
	Result *big.Int
	Err    error
}

// Name returns the mock calculator's name.
func (m *MockCalculator) Name() string {
	return "Mock"
}

// Calculate implements the fibonacci.Calculator interface returning predefined results.
func (m *MockCalculator) Calculate(ctx context.Context, progressChan chan<- fibonacci.ProgressUpdate, calcIndex int, n uint64, opts fibonacci.Options) (*big.Int, error) {
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
	return NewServer(registry, cfg)
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
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "not a valid algorithm",
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

			req := httptest.NewRequest("GET", "/calculate"+tt.queryParams, nil)
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

	req := httptest.NewRequest("GET", "/health", nil)
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

	req := httptest.NewRequest("GET", "/algorithms", nil)
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
			req := httptest.NewRequest(tt.method, tt.endpoint, nil)
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

	req := httptest.NewRequest("GET", "/test", nil)
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
