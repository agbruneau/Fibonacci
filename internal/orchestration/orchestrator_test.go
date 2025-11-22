package orchestration

import (
	"errors"
	"math/big"
	"testing"
	"time"
)

func TestCompareResults(t *testing.T) {
	res1 := big.NewInt(100)
	res2 := big.NewInt(100)
	resDifferent := big.NewInt(200)

	tests := []struct {
		name             string
		input            []CalculationResult
		expectedSuccess  int
		expectedMismatch bool
		expectedFirstErr error
	}{
		{
			name: "All success matching",
			input: []CalculationResult{
				{Name: "A", Result: res1, Duration: 10 * time.Millisecond},
				{Name: "B", Result: res2, Duration: 5 * time.Millisecond},
			},
			expectedSuccess:  2,
			expectedMismatch: false,
		},
		{
			name: "Mismatch detected",
			input: []CalculationResult{
				{Name: "A", Result: res1, Duration: 10 * time.Millisecond},
				{Name: "B", Result: resDifferent, Duration: 5 * time.Millisecond},
			},
			expectedSuccess:  2,
			expectedMismatch: true,
		},
		{
			name: "Partial failure",
			input: []CalculationResult{
				{Name: "A", Result: res1, Duration: 10 * time.Millisecond},
				{Name: "B", Err: errors.New("fail"), Duration: 5 * time.Millisecond},
			},
			expectedSuccess:  1,
			expectedMismatch: false,
		},
		{
			name: "All fail",
			input: []CalculationResult{
				{Name: "A", Err: errors.New("fail1"), Duration: 10 * time.Millisecond},
				{Name: "B", Err: errors.New("fail2"), Duration: 5 * time.Millisecond},
			},
			expectedSuccess:  0,
			expectedMismatch: false,
			expectedFirstErr: errors.New("fail1"), // Check only existence/message ideally
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := CompareResults(tt.input)

			if summary.SuccessCount != tt.expectedSuccess {
				t.Errorf("Expected success count %d, got %d", tt.expectedSuccess, summary.SuccessCount)
			}

			if summary.Mismatch != tt.expectedMismatch {
				t.Errorf("Expected mismatch %v, got %v", tt.expectedMismatch, summary.Mismatch)
			}

			if tt.expectedFirstErr != nil && summary.FirstError == nil {
				t.Error("Expected error, got nil")
			}
			
			// Check sorting: First result in sorted list should be the fastest successful one (or failed if all failed)
			if len(summary.SortedResults) > 1 {
				first := summary.SortedResults[0]
				second := summary.SortedResults[1]
				// Success comes before failure
				if first.Err == nil && second.Err != nil {
					// Correct order
				} else if first.Err == nil && second.Err == nil {
					if first.Duration > second.Duration {
						t.Error("Results not sorted by duration")
					}
				}
			}
		})
	}
}
