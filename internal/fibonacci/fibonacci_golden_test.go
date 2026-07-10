package fibonacci_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/agbruneau/FibGo/internal/fibonacci"
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

	calculators := map[string]fibonacci.Calculator{
		"FastDoubling": fibonacci.MustNewCalculator(&fibonacci.FastDoublingCalculator{}),
		"MatrixExp":    fibonacci.MustNewCalculator(&fibonacci.MatrixExponentiationCalculator{}),
		"FFTBased":     fibonacci.MustNewCalculator(&fibonacci.FFTBasedCalculator{}),
	}

	ctx := context.Background()

	for name, calc := range calculators {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			for _, tc := range cases {
				// Capture range variable

				t.Run(fmt.Sprintf("N=%d", tc.N), func(t *testing.T) {
					t.Parallel()

					expected := new(big.Int)
					if _, ok := expected.SetString(tc.Result, 10); !ok {
						t.Fatalf("golden file entry for N=%d is malformed: %q is not a valid base-10 integer", tc.N, tc.Result)
					}

					got, err := calc.Calculate(ctx, nil, 0, tc.N, fibonacci.Options{ParallelThreshold: fibonacci.DefaultParallelThreshold})
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
