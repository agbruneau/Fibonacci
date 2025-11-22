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

	"example.com/fibcalc/internal/fibonacci"
)

// MockCalculator is a mock implementation of fibonacci.Calculator for testing.
type MockCalculator struct {
	Result *big.Int
	Err    error
	Delay  time.Duration
}

func (m *MockCalculator) Name() string {
	return "Mock"
}

// Update signature to match fibonacci.Calculator interface
func (m *MockCalculator) Calculate(ctx context.Context, progressChan chan<- fibonacci.ProgressUpdate, calcIndex int, n uint64, threshold int, fftThreshold int) (*big.Int, error) {
	if m.Delay > 0 {
		timer := time.NewTimer(m.Delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return m.Result, m.Err
}

func TestHandleCalculate(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		mockResult     *big.Int
		mockErr        error
		mockDelay      time.Duration
		expectedStatus int
		expectedBody   string
		isJSON         bool
	}{
		{
			name:           "Success",
			queryParams:    "?n=10",
			mockResult:     big.NewInt(55),
			expectedStatus: http.StatusOK,
			expectedBody:   `55`,
			isJSON:         true,
		},
		{
			name:           "Missing n",
			queryParams:    "",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing 'n' parameter",
			isJSON:         false,
		},
		{
			name:           "Invalid n",
			queryParams:    "?n=abc",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid 'n' parameter",
			isJSON:         false,
		},
		{
			name:           "Unknown algorithm",
			queryParams:    "?n=10&algo=unknown",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid 'algo' parameter",
			isJSON:         false,
		},
		{
			name:           "Calculation error",
			queryParams:    "?n=10",
			mockErr:        errors.New("calc error"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "calc error",
			isJSON:         true,
		},
		{
			name:           "Request timeout",
			queryParams:    "?n=10&timeout=1ms",
			mockDelay:      20 * time.Millisecond,
			expectedStatus: http.StatusGatewayTimeout,
			expectedBody:   context.DeadlineExceeded.Error(),
			isJSON:         true,
		},
		{
			name:           "Invalid timeout parameter",
			queryParams:    "?n=10&timeout=foobar",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid 'timeout' parameter",
			isJSON:         false,
		},
		{
			name:           "Invalid threshold parameter",
			queryParams:    "?n=10&threshold=-1",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid 'threshold' parameter",
			isJSON:         false,
		},
		{
			name:           "Invalid fft-threshold parameter",
			queryParams:    "?n=10&fft-threshold=foo",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid 'fft-threshold' parameter",
			isJSON:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCalc := &MockCalculator{
				Result: tt.mockResult,
				Err:    tt.mockErr,
				Delay:  tt.mockDelay,
			}
			registry := map[string]fibonacci.Calculator{
				"fast": mockCalc,
			}
			server := &Server{Registry: registry}

			req := httptest.NewRequest("GET", "/calculate"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			server.handleCalculate(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			bodyBytes, _ := io.ReadAll(resp.Body)
			bodyString := strings.TrimSpace(string(bodyBytes))

			if tt.isJSON {
				var jsonResp Response
				if err := json.Unmarshal(bodyBytes, &jsonResp); err != nil {
					t.Errorf("Failed to unmarshal JSON response: %v", err)
				}
				if tt.expectedStatus >= http.StatusBadRequest {
					if jsonResp.Error != tt.expectedBody {
						t.Errorf("Expected error message %q, got %q", tt.expectedBody, jsonResp.Error)
					}
				} else {
					if jsonResp.Result == nil {
						t.Fatalf("Expected a result, got nil")
					}
					if jsonResp.Result.Cmp(big.NewInt(55)) != 0 { // assuming 55 for success test
						t.Errorf("Expected result 55, got %s", jsonResp.Result.String())
					}
				}
			} else {
				if !strings.Contains(bodyString, tt.expectedBody) {
					t.Errorf("Expected body to contain %q, got %q", tt.expectedBody, bodyString)
				}
			}
		})
	}
}
