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

// GoldenData represents the structure of our golden file entries
type GoldenData struct {
	N      uint64 `json:"n"`
	Result string `json:"result"`
}

func TestCalculatorsAgainstGoldenFile(t *testing.T) {
	// Load golden data
	goldenPath := filepath.Join("testdata", "fibonacci_golden.json")
	file, err := os.Open(goldenPath)
	if err != nil {
		t.Fatalf("Failed to open golden file: %v. Did you run 'go run cmd/generate-golden/main.go'?", err)
	}
	defer file.Close()

	var cases []GoldenData
	if err := json.NewDecoder(file).Decode(&cases); err != nil {
		t.Fatalf("Failed to decode golden file: %v", err)
	}

	calculators := map[string]Calculator{
		"FastDoubling": NewCalculator(&OptimizedFastDoubling{}),
		"MatrixExp":    NewCalculator(&MatrixExponentiation{}),
		"FFTBased":     NewCalculator(&FFTBasedCalculator{}),
	}

	ctx := context.Background()

	for name, calc := range calculators {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			for _, tc := range cases {
				// Capture range variable
				tc := tc
				t.Run(fmt.Sprintf("N=%d", tc.N), func(t *testing.T) {
					t.Parallel()

					expected := new(big.Int)
					expected.SetString(tc.Result, 10)

					got, err := calc.Calculate(ctx, nil, 0, tc.N, Options{ParallelThreshold: DefaultParallelThreshold})
					if err != nil {
						t.Fatalf("Calculation failed for N=%d: %v", tc.N, err)
					}

					if got.Cmp(expected) != 0 {
						t.Errorf("Mismatch for N=%d.\nExpected: %s\nGot:      %s", tc.N, expected.String(), got.String())
					}
				})
			}
		})
	}
}
