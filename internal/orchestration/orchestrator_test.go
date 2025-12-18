package orchestration

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/agbru/fibcalc/internal/config"
	apperrors "github.com/agbru/fibcalc/internal/errors"
	"github.com/agbru/fibcalc/internal/fibonacci"
)

// MockCalculator is a mock implementation of fibonacci.Calculator
// used for testing the orchestration logic without invoking real algorithms.
type MockCalculator struct {
	NameFunc      func() string
	CalculateFunc func(ctx context.Context, reporter fibonacci.ProgressReporter, index int, n uint64, opts fibonacci.Options) (*big.Int, error)
}

// Name returns the mocked name of the calculator.
func (m *MockCalculator) Name() string {
	if m.NameFunc != nil {
		return m.NameFunc()
	}
	return "Mock"
}

// Calculate invokes the mocked CalculateFunc.
func (m *MockCalculator) Calculate(ctx context.Context, progressChan chan<- fibonacci.ProgressUpdate, index int, n uint64, opts fibonacci.Options) (*big.Int, error) {
	if m.CalculateFunc != nil {
		// Create a dummy reporter that sends to the channel
		reporter := func(progress float64) {
			if progressChan != nil {
				progressChan <- fibonacci.ProgressUpdate{CalculatorIndex: index, Value: progress}
			}
		}
		return m.CalculateFunc(ctx, reporter, index, n, opts)
	}
	return big.NewInt(0), nil
}

// TestExecuteCalculations verifies that the orchestrator correctly runs calculators
// and aggregates their results.
func TestExecuteCalculations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		calculators []fibonacci.Calculator
		expectedLen int
		expectError bool
	}{
		{
			name: "Single success",
			calculators: []fibonacci.Calculator{
				&MockCalculator{
					CalculateFunc: func(ctx context.Context, reporter fibonacci.ProgressReporter, index int, n uint64, opts fibonacci.Options) (*big.Int, error) {
						return big.NewInt(1), nil
					},
				},
			},
			expectedLen: 1,
			expectError: false,
		},
		{
			name: "Single failure",
			calculators: []fibonacci.Calculator{
				&MockCalculator{
					CalculateFunc: func(ctx context.Context, reporter fibonacci.ProgressReporter, index int, n uint64, opts fibonacci.Options) (*big.Int, error) {
						return nil, errors.New("mock error")
					},
				},
			},
			expectedLen: 1,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			results := ExecuteCalculations(context.Background(), tt.calculators, config.AppConfig{}, &DiscardWriter{})
			if len(results) != tt.expectedLen {
				t.Errorf("expected %d results, got %d", tt.expectedLen, len(results))
			}
			if tt.expectError {
				if results[0].Err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if results[0].Err != nil {
					t.Errorf("unexpected error: %v", results[0].Err)
				}
			}
		})
	}
}

// TestAnalyzeComparisonResults verifies the logic for comparing results from
// multiple algorithms. It checks for consistent results, handling of failures,
// and detection of mismatches.
func TestAnalyzeComparisonResults(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		results        []CalculationResult
		expectedStatus int
	}{
		{
			name: "All success",
			results: []CalculationResult{
				{Name: "A", Result: big.NewInt(5), Duration: time.Millisecond, Err: nil},
				{Name: "B", Result: big.NewInt(5), Duration: time.Millisecond, Err: nil},
			},
			expectedStatus: apperrors.ExitSuccess,
		},
		{
			name: "Mismatch",
			results: []CalculationResult{
				{Name: "A", Result: big.NewInt(5), Duration: time.Millisecond, Err: nil},
				{Name: "B", Result: big.NewInt(6), Duration: time.Millisecond, Err: nil},
			},
			expectedStatus: apperrors.ExitErrorMismatch,
		},
		{
			name: "All failure",
			results: []CalculationResult{
				{Name: "A", Result: nil, Duration: time.Millisecond, Err: errors.New("fail")},
				{Name: "B", Result: nil, Duration: time.Millisecond, Err: errors.New("fail")},
			},
			expectedStatus: apperrors.ExitErrorGeneric,
		},
		{
			name: "Mixed success/failure",
			results: []CalculationResult{
				{Name: "A", Result: big.NewInt(5), Duration: time.Millisecond, Err: nil},
				{Name: "B", Result: nil, Duration: time.Millisecond, Err: errors.New("fail")},
			},
			expectedStatus: apperrors.ExitSuccess,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			status := AnalyzeComparisonResults(tt.results, config.AppConfig{}, &DiscardWriter{})
			if status != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, status)
			}
		})
	}
}

// DiscardWriter is a helper that implements io.Writer and discards all data.
type DiscardWriter struct{}

func (d *DiscardWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}
