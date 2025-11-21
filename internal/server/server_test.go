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

	"example.com/fibcalc/internal/fibonacci"
)

// MockCalculator is a mock implementation of fibonacci.Calculator for testing.
type MockCalculator struct {
	Result *big.Int
	Err    error
}

func (m *MockCalculator) Name() string {
	return "Mock"
}

// Update signature to match fibonacci.Calculator interface
func (m *MockCalculator) Calculate(ctx context.Context, progressChan chan<- fibonacci.ProgressUpdate, calcIndex int, n uint64, threshold int, fftThreshold int) (*big.Int, error) {
	return m.Result, m.Err
}

func TestHandleCalculate(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		mockResult     *big.Int
		mockErr        error
		expectedStatus int
		expectedBody   string
		isJSON         bool
	}{
		{
			name:           "Success",
			queryParams:    "?n=10",
			mockResult:     big.NewInt(55),
			mockErr:        nil,
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
			mockResult:     nil,
			mockErr:        errors.New("calc error"),
			expectedStatus: http.StatusOK,
			expectedBody:   "calc error",
			isJSON:         true,
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
				if tt.mockErr != nil {
					if jsonResp.Error != tt.expectedBody {
						t.Errorf("Expected error message %q, got %q", tt.expectedBody, jsonResp.Error)
					}
				} else {
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
