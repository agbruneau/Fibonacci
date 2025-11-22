package server

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"example.com/fibcalc/internal/fibonacci"
)

// MockCalculator implements fibonacci.Calculator for testing
type MockCalculator struct {
	Result *big.Int
	Err    error
}

func (m *MockCalculator) Calculate(ctx context.Context, progressChan chan<- fibonacci.ProgressUpdate, calcIndex int, n uint64, threshold int, fftThreshold int) (*big.Int, error) {
	return m.Result, m.Err
}

func (m *MockCalculator) Name() string {
	return "Mock"
}

func TestHandleCalculate(t *testing.T) {
	// Setup
	mockResult := big.NewInt(55) // F(10)
	registry := map[string]fibonacci.Calculator{
		"mock": &MockCalculator{Result: mockResult, Err: nil},
	}
	srv := &Server{Registry: registry}

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedN      uint64
		expectedResult string
		expectError    bool
	}{
		{
			name:           "Valid Request",
			queryParams:    "?n=10&algo=mock",
			expectedStatus: http.StatusOK,
			expectedN:      10,
			expectedResult: "55",
			expectError:    false,
		},
		{
			name:           "Missing N",
			queryParams:    "?algo=mock",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "Invalid N",
			queryParams:    "?n=abc&algo=mock",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "Invalid Algo",
			queryParams:    "?n=10&algo=unknown",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "Default Algo (fallback to fast, but not in mock registry so error)",
			queryParams:    "?n=10",
			expectedStatus: http.StatusBadRequest, // Fails because "fast" is not in our test registry
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/calculate"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			srv.handleCalculate(w, req)

			res := w.Result()
			defer res.Body.Close()

			if res.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, res.StatusCode)
			}

			if !tt.expectError && res.StatusCode == http.StatusOK {
				var response Response
				if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if response.N != tt.expectedN {
					t.Errorf("Expected N=%d, got %d", tt.expectedN, response.N)
				}

				if response.Result.Cmp(mockResult) != 0 {
					t.Errorf("Expected result %s, got %s", mockResult.String(), response.Result.String())
				}

				if response.Error != "" {
					t.Errorf("Unexpected error in response: %s", response.Error)
				}
			}
		})
	}
}
