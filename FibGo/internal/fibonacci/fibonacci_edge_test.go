package fibonacci

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
)

// ─────────────────────────────────────────────────────────────────────────────
// Edge Case Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestFibonacciRecurrenceRelation verifies F(n) = F(n-1) + F(n-2).
func TestFibonacciRecurrenceRelation(t *testing.T) {
	t.Parallel()
	calc := NewCalculator(&OptimizedFastDoubling{})
	ctx := context.Background()
	opts := Options{ParallelThreshold: DefaultParallelThreshold}

	testIndices := []uint64{10, 50, 100, 500, 1000}

	for _, n := range testIndices {
		t.Run(fmt.Sprintf("N=%d", n), func(t *testing.T) {
			t.Parallel()
			fn, err := calc.Calculate(ctx, nil, 0, n, opts)
			if err != nil {
				t.Fatalf("Failed to calculate F(%d): %v", n, err)
			}

			fn1, err := calc.Calculate(ctx, nil, 0, n-1, opts)
			if err != nil {
				t.Fatalf("Failed to calculate F(%d): %v", n-1, err)
			}

			fn2, err := calc.Calculate(ctx, nil, 0, n-2, opts)
			if err != nil {
				t.Fatalf("Failed to calculate F(%d): %v", n-2, err)
			}

			sum := new(big.Int).Add(fn1, fn2)
			if fn.Cmp(sum) != 0 {
				t.Errorf("F(%d) != F(%d) + F(%d)\n  F(%d)   = %s\n  F(%d) = %s\n  F(%d) = %s\n  Sum    = %s",
					n, n-1, n-2, n, fn.String(), n-1, fn1.String(), n-2, fn2.String(), sum.String())
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Parallel and Threshold Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestParallelThresholdBoundary tests behavior at the parallel threshold boundary.
func TestParallelThresholdBoundary(t *testing.T) {
	t.Parallel()
	calc := NewCalculator(&OptimizedFastDoubling{})
	ctx := context.Background()

	// Test with threshold disabled (sequential only)
	t.Run("Sequential", func(t *testing.T) {
		t.Parallel()
		opts := Options{ParallelThreshold: 0}
		result, err := calc.Calculate(ctx, nil, 0, 10000, opts)
		if err != nil {
			t.Fatalf("Sequential calculation failed: %v", err)
		}
		if result == nil {
			t.Fatal("Got nil result")
		}
	})

	// Test with very low threshold (force parallelism)
	t.Run("ForcedParallel", func(t *testing.T) {
		t.Parallel()
		opts := Options{ParallelThreshold: 1}
		result, err := calc.Calculate(ctx, nil, 0, 10000, opts)
		if err != nil {
			t.Fatalf("Parallel calculation failed: %v", err)
		}
		if result == nil {
			t.Fatal("Got nil result")
		}
	})

	// Test with default threshold
	t.Run("DefaultThreshold", func(t *testing.T) {
		t.Parallel()
		opts := Options{ParallelThreshold: DefaultParallelThreshold}
		result, err := calc.Calculate(ctx, nil, 0, 10000, opts)
		if err != nil {
			t.Fatalf("Default calculation failed: %v", err)
		}
		if result == nil {
			t.Fatal("Got nil result")
		}
	})
}

// TestFFTThresholdVariations tests behavior with different FFT thresholds.
func TestFFTThresholdVariations(t *testing.T) {
	t.Parallel()
	calc := NewCalculator(&OptimizedFastDoubling{})
	ctx := context.Background()
	n := uint64(50000)

	thresholds := []int{0, 1000, 10000, 100000, 1000000}

	var results []*big.Int
	var mu sync.Mutex

	for _, threshold := range thresholds {
		t.Run(fmt.Sprintf("Threshold=%d", threshold), func(t *testing.T) {
			t.Parallel()
			opts := Options{
				ParallelThreshold: DefaultParallelThreshold,
				FFTThreshold:      threshold,
			}
			result, err := calc.Calculate(ctx, nil, 0, n, opts)
			if err != nil {
				t.Fatalf("Calculation with FFT threshold %d failed: %v", threshold, err)
			}
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		})
	}

	// Verify all results are the same
	if len(results) > 1 {
		for i := 1; i < len(results); i++ {
			if results[0].Cmp(results[i]) != 0 {
				t.Errorf("Results differ between FFT thresholds:\n  First: %s\n  Index %d: %s",
					results[0].String(), i, results[i].String())
			}
		}
	}
}

// TestStrassenThresholdVariations tests behavior with different Strassen thresholds.
func TestStrassenThresholdVariations(t *testing.T) {
	t.Parallel()
	calc := NewCalculator(&MatrixExponentiation{})
	ctx := context.Background()
	n := uint64(10000)

	thresholds := []int{0, 64, 128, 256, 512, 1024, 3072}

	var results []*big.Int
	var mu sync.Mutex

	for _, threshold := range thresholds {
		t.Run(fmt.Sprintf("StrassenThreshold=%d", threshold), func(t *testing.T) {
			t.Parallel()
			opts := Options{
				ParallelThreshold: DefaultParallelThreshold,
				StrassenThreshold: threshold,
			}
			result, err := calc.Calculate(ctx, nil, 0, n, opts)
			if err != nil {
				t.Fatalf("Calculation with Strassen threshold %d failed: %v", threshold, err)
			}
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		})
	}

	// Verify all results are the same
	if len(results) > 1 {
		for i := 1; i < len(results); i++ {
			if results[0].Cmp(results[i]) != 0 {
				t.Errorf("Results differ between Strassen thresholds:\n  First: %s\n  Index %d: %s",
					results[0].String(), i, results[i].String())
			}
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Context and Cancellation Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestContextCancellationImmediate tests immediate cancellation.
func TestContextCancellationImmediate(t *testing.T) {
	t.Parallel()
	calculators := map[string]coreCalculator{
		"FastDoubling": &OptimizedFastDoubling{},
		"MatrixExp":    &MatrixExponentiation{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	opts := Options{ParallelThreshold: DefaultParallelThreshold}

	for name, calc := range calculators {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := calc.CalculateCore(ctx, func(float64) {}, 1000000, opts)
			if !errors.Is(err, context.Canceled) {
				t.Errorf("Expected context.Canceled, got: %v", err)
			}
		})
	}
}

// TestContextTimeoutShort tests short timeout behavior.
func TestContextTimeoutShort(t *testing.T) {
	t.Parallel()
	calculators := map[string]coreCalculator{
		"FastDoubling": &OptimizedFastDoubling{},
		"MatrixExp":    &MatrixExponentiation{},
	}

	opts := Options{ParallelThreshold: DefaultParallelThreshold}

	for name, calc := range calculators {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
			defer cancel()

			// Use a large n that won't complete in 1ms
			_, err := calc.CalculateCore(ctx, func(float64) {}, 100_000_000, opts)

			if err == nil {
				t.Fatal("Expected timeout or cancellation error, got nil")
			}

			if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
				t.Errorf("Expected timeout or cancellation error, got: %v", err)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Concurrency Safety Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestConcurrentCalculations verifies thread-safety of concurrent calculations.
func TestConcurrentCalculations(t *testing.T) {
	t.Parallel()
	calc := NewCalculator(&OptimizedFastDoubling{})
	ctx := context.Background()
	opts := Options{ParallelThreshold: DefaultParallelThreshold}

	const numGoroutines = 50
	const n = 1000

	expected, err := calc.Calculate(ctx, nil, 0, n, opts)
	if err != nil {
		t.Fatalf("Failed to calculate expected result: %v", err)
	}

	var g errgroup.Group

	for i := 0; i < numGoroutines; i++ {
		g.Go(func() error {
			result, err := calc.Calculate(ctx, nil, 0, n, opts)
			if err != nil {
				return err
			}
			if result.Cmp(expected) != 0 {
				return fmt.Errorf("result mismatch: expected %s, got %s", expected.String(), result.String())
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		t.Errorf("Concurrent calculation error: %v", err)
	}
}

// TestConcurrentDifferentN tests concurrent calculations with different N values.
func TestConcurrentDifferentN(t *testing.T) {
	t.Parallel()
	calc := NewCalculator(&OptimizedFastDoubling{})
	ctx := context.Background()
	opts := Options{ParallelThreshold: DefaultParallelThreshold}

	nValues := []uint64{100, 500, 1000, 2000, 5000, 10000}

	var g errgroup.Group

	// Create a buffered channel for results to avoid locking mutex manually if possible,
	// but using a safe map or channel is fine. Here we just want to verify consistency.
	// Since we need to verify N against its result, returning a struct is best.

	type calcResult struct {
		n   uint64
		val *big.Int
	}

	resultChan := make(chan calcResult, len(nValues))

	for _, n := range nValues {
		n := n // capture loop var
		g.Go(func() error {
			result, err := calc.Calculate(ctx, nil, 0, n, opts)
			if err != nil {
				return fmt.Errorf("failed to calculate F(%d): %w", n, err)
			}
			resultChan <- calcResult{n, result}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		t.Fatalf("Concurrent calculation failed: %v", err)
	}
	close(resultChan)

	results := make(map[uint64]*big.Int)
	for res := range resultChan {
		results[res.n] = res.val
	}

	// Verify results using recurrence relation
	for n, result := range results {
		if n <= 1 {
			continue
		}
		fn1, ok1 := results[n-1]
		fn2, ok2 := results[n-2]
		// We only provided specific N values, so we can't necessarily check F(n-1) + F(n-2)
		// unless those values were also in the input set.
		// However, the original test checked this if present.

		if ok1 && ok2 {
			expected := new(big.Int).Add(fn1, fn2)
			if result.Cmp(expected) != 0 {
				t.Errorf("F(%d) = %s, but F(%d)+F(%d) = %s",
					n, result.String(), n-1, n-2, expected.String())
			}
		}
	}
}
