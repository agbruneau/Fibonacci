// main_test.go

package main

import (
	"context"
	"math/big"
	"sync"
	"testing"
)

// TestFibonacciAlgorithms verifies the correctness of each Fibonacci algorithm
// using a table-driven approach.
func TestFibonacciAlgorithms(t *testing.T) {
	// Test cases with well-known Fibonacci values.
	testCases := []struct {
		name    string
		n       int
		want    *big.Int
		wantErr bool // If an error is expected (e.g., for n < 0)
	}{
		{"n=0", 0, big.NewInt(0), false},
		{"n=1", 1, big.NewInt(1), false},
		{"n=2", 2, big.NewInt(1), false},
		{"n=7", 7, big.NewInt(13), false},
		{"n=10", 10, big.NewInt(55), false},
		{"n=20", 20, big.NewInt(6765), false},
		{"negative n", -1, nil, true}, // Test case for negative input
	}

	// Map of algorithms to test.
	algos := map[string]fibFunc{
		"Fast Doubling": fibFastDoubling,
		"Matrix 2x2":    fibMatrix,
		"Binet":         fibBinet,
		"Iterative":     fibIterative,
	}

	pool := newIntPool()
	ctx := context.Background() // Use a background context for tests

	// Iterate over each algorithm.
	for algoName, algoFunc := range algos {
		// Iterate over each test case.
		for _, tc := range testCases {
			// t.Run creates sub-tests, making debugging easier.
			// The test name will be, for example, "Fast Doubling/n=10".
			t.Run(algoName+"/"+tc.name, func(t *testing.T) {
				// Execute the algorithm function.
				// The progress channel is not needed for correctness testing.
				got, err := algoFunc(ctx, nil, tc.n, pool)

				// Check if an error was expected.
				if tc.wantErr {
					if err == nil {
						t.Errorf("expected an error for n=%d, but got none", tc.n)
					}
					return // Test is done if an error was expected and occurred.
				}

				// Check if an unexpected error occurred.
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				// Compare the obtained result with the expected result.
				if got == nil && tc.want == nil {
					// This case should ideally be covered by wantErr if nil result means error
				} else if got == nil && tc.want != nil {
					t.Errorf("for F(%d), expected %s, but got nil", tc.n, tc.want.String())
				} else if got != nil && tc.want == nil {
					t.Errorf("for F(%d), expected nil, but got %s", tc.n, got.String())
				} else if got.Cmp(tc.want) != 0 {
					t.Errorf("for F(%d), expected %s, but got %s", tc.n, tc.want.String(), got.String())
				}
			})
		}
	}
}

// TestFibonacciConsistencyForLargeN verifies that the exact algorithms
// produce the same result for a larger n.
func TestFibonacciConsistencyForLargeN(t *testing.T) {
	n := 1000 // A reasonably large n, but not too long to compute for tests.
	// For very large n, Binet might show precision issues.

	pool := newIntPool()
	ctx := context.Background()

	var results = make(map[string]*big.Int)
	var errors = make(map[string]error)

	// Algorithms to test for consistency (excluding Binet for very large N due to precision)
	consistentAlgos := map[string]fibFunc{
		"Fast Doubling": fibFastDoubling,
		"Matrix 2x2":    fibMatrix,
		"Iterative":     fibIterative,
	}

	for name, fn := range consistentAlgos {
		t.Run(name, func(t *testing.T) {
			res, err := fn(ctx, nil, n, pool)
			results[name] = res
			errors[name] = err
			if err != nil {
				t.Fatalf("%s failed for n=%d: %v", name, n, err)
			}
			if res == nil {
				t.Fatalf("%s returned nil for n=%d without error", name, n)
			}
		})
	}

	// Compare results
	var referenceResult *big.Int
	var referenceAlgoName string

	for name, res := range results {
		if referenceResult == nil {
			referenceResult = res
			referenceAlgoName = name
		} else {
			if res.Cmp(referenceResult) != 0 {
				t.Errorf("Discrepancy for F(%d): %s (%s...) vs %s (%s...)",
					n,
					referenceAlgoName, referenceResult.String()[:min(20, len(referenceResult.String()))],
					name, res.String()[:min(20, len(res.String()))])
			}
		}
	}

	// Note: Binet's formula is not compared for very large n here due to
	// potential floating-point precision errors that can lead to slight rounding differences.
	// It's tested for smaller values in TestFibonacciAlgorithms.
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ------------------------------------------------------------
// Benchmarks
// ------------------------------------------------------------

// Common n for all benchmarks for fair comparison.
const benchmarkN = 100000 // Adjusted for potentially slower iterative, but still substantial

// BenchmarkFibFastDoubling measures the performance of the Fast Doubling algorithm.
func BenchmarkFibFastDoubling(b *testing.B) {
	pool := newIntPool()
	ctx := context.Background()
	b.ReportAllocs() // Display memory allocations.
	b.ResetTimer()   // Reset timer to exclude setup time.

	for i := 0; i < b.N; i++ {
		// The result is not verified here; focus is on performance.
		_, _ = fibFastDoubling(ctx, nil, benchmarkN, pool)
	}
}

// BenchmarkFibMatrix measures the performance of the matrix exponentiation algorithm.
func BenchmarkFibMatrix(b *testing.B) {
	pool := newIntPool()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = fibMatrix(ctx, nil, benchmarkN, pool)
	}
}

// BenchmarkFibBinet measures the performance of Binet's formula.
func BenchmarkFibBinet(b *testing.B) {
	// The pool is not actively used by Binet for big.Ints, but passed for API consistency.
	var pool *sync.Pool // Binet does not use the pool in its current form.
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = fibBinet(ctx, nil, benchmarkN, pool)
	}
}

// BenchmarkFibIterative measures the performance of the iterative algorithm.
func BenchmarkFibIterative(b *testing.B) {
	pool := newIntPool()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	// For iterative, benchmarkN might be too large for reasonable benchmark times.
	// Consider a smaller N for iterative if it's too slow, or adjust b.N.
	// For this example, we'll use benchmarkN, but be mindful of execution time.
	// If benchmarkN is very large (e.g. 10,000,000), this will be slow.
	// Let's use a smaller N for iterative benchmark for practical reasons.
	iterativeBenchmarkN := 100000                            // Can be adjusted if too slow/fast.
	if benchmarkN < iterativeBenchmarkN && benchmarkN > 20 { // if global N is smaller, use that
		iterativeBenchmarkN = benchmarkN
	}

	for i := 0; i < b.N; i++ {
		_, _ = fibIterative(ctx, nil, iterativeBenchmarkN, pool)
	}
}
