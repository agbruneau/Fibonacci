package fibonacci

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"
)

// TestFastDoubling_DynamicThresholds_Correctness guards the DTM-on integration
// path in CalculateCore (the EnableDynamicThresholds branch, fastdoubling.go),
// which the golden and benchmark suites never exercise with assertions
// (audit F-001; ADR-0001 keeps the dynamic-threshold manager). It runs
// FastDoubling with dynamic thresholds enabled and verifies the result against
// the immutable golden oracle and against the DTM-off result for the same N.
func TestFastDoubling_DynamicThresholds_Correctness(t *testing.T) {
	t.Parallel()

	goldenPath := filepath.Join("testdata", "fibonacci_golden.json")
	file, err := os.Open(goldenPath)
	if err != nil {
		t.Fatalf("Failed to open golden file: %v", err)
	}
	defer file.Close()

	var cases []GoldenData
	if err := json.NewDecoder(file).Decode(&cases); err != nil {
		t.Fatalf("Failed to decode golden file: %v", err)
	}

	calc := MustNewCalculator(&FastDoublingCalculator{})
	ctx := context.Background()

	for _, tc := range cases {
		t.Run(fmt.Sprintf("N=%d", tc.N), func(t *testing.T) {
			t.Parallel()

			expected := new(big.Int)
			expected.SetString(tc.Result, 10)

			// A small adjustment interval makes the dynamic-threshold manager's
			// ShouldAdjust logic actually fire across the doubling iterations,
			// rather than merely toggling the DTM branch on.
			dtmOpts := Options{
				ParallelThreshold:         DefaultParallelThreshold,
				EnableDynamicThresholds:   true,
				DynamicAdjustmentInterval: 2,
			}
			gotDTM, err := calc.Calculate(ctx, nil, 0, tc.N, dtmOpts)
			if err != nil {
				t.Fatalf("DTM-on calculation failed for N=%d: %v", tc.N, err)
			}
			if gotDTM.Cmp(expected) != 0 {
				t.Errorf("DTM-on result mismatch for N=%d:\n expected %s\n got      %s",
					tc.N, expected.String(), gotDTM.String())
			}

			// Enabling dynamic thresholds must not change the computed value.
			gotStatic, err := calc.Calculate(ctx, nil, 0, tc.N, Options{ParallelThreshold: DefaultParallelThreshold})
			if err != nil {
				t.Fatalf("DTM-off calculation failed for N=%d: %v", tc.N, err)
			}
			if gotDTM.Cmp(gotStatic) != 0 {
				t.Errorf("DTM-on vs DTM-off divergence for N=%d", tc.N)
			}
		})
	}
}
