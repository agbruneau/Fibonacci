package fibonacci

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// Edge Case Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestFibonacciZero verifies F(0) = 0 for all algorithms.
func TestFibonacciZero(t *testing.T) {
	calculators := map[string]coreCalculator{
		"FastDoubling": &OptimizedFastDoubling{},
		"MatrixExp":    &MatrixExponentiation{},
		"FFTBased":     &FFTBasedCalculator{},
	}

	ctx := context.Background()
	opts := Options{ParallelThreshold: DefaultParallelThreshold}

	for name, calc := range calculators {
		t.Run(name, func(t *testing.T) {
			result, err := calc.CalculateCore(ctx, func(float64) {}, 0, opts)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result.Cmp(big.NewInt(0)) != 0 {
				t.Errorf("F(0) should be 0, got %s", result.String())
			}
		})
	}
}

// TestFibonacciOne verifies F(1) = 1 for all algorithms.
func TestFibonacciOne(t *testing.T) {
	calculators := map[string]coreCalculator{
		"FastDoubling": &OptimizedFastDoubling{},
		"MatrixExp":    &MatrixExponentiation{},
		"FFTBased":     &FFTBasedCalculator{},
	}

	ctx := context.Background()
	opts := Options{ParallelThreshold: DefaultParallelThreshold}

	for name, calc := range calculators {
		t.Run(name, func(t *testing.T) {
			result, err := calc.CalculateCore(ctx, func(float64) {}, 1, opts)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result.Cmp(big.NewInt(1)) != 0 {
				t.Errorf("F(1) should be 1, got %s", result.String())
			}
		})
	}
}

// TestFibonacciTwo verifies F(2) = 1 for all algorithms.
func TestFibonacciTwo(t *testing.T) {
	calculators := map[string]coreCalculator{
		"FastDoubling": &OptimizedFastDoubling{},
		"MatrixExp":    &MatrixExponentiation{},
		"FFTBased":     &FFTBasedCalculator{},
	}

	ctx := context.Background()
	opts := Options{ParallelThreshold: DefaultParallelThreshold}

	for name, calc := range calculators {
		t.Run(name, func(t *testing.T) {
			result, err := calc.CalculateCore(ctx, func(float64) {}, 2, opts)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result.Cmp(big.NewInt(1)) != 0 {
				t.Errorf("F(2) should be 1, got %s", result.String())
			}
		})
	}
}

// TestFibonacciMaxUint64 verifies F(93), the largest Fibonacci that fits in uint64.
func TestFibonacciMaxUint64(t *testing.T) {
	// F(93) = 12200160415121876738, which is the largest that fits in uint64
	expected := new(big.Int)
	expected.SetString("12200160415121876738", 10)

	calculators := map[string]coreCalculator{
		"FastDoubling": &OptimizedFastDoubling{},
		"MatrixExp":    &MatrixExponentiation{},
		"FFTBased":     &FFTBasedCalculator{},
	}

	ctx := context.Background()
	opts := Options{ParallelThreshold: DefaultParallelThreshold}

	for name, calc := range calculators {
		t.Run(name, func(t *testing.T) {
			result, err := calc.CalculateCore(ctx, func(float64) {}, 93, opts)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result.Cmp(expected) != 0 {
				t.Errorf("F(93) incorrect.\nExpected: %s\nGot: %s", expected.String(), result.String())
			}
		})
	}
}

// TestFibonacciOverflowUint64 verifies F(94), which overflows uint64.
func TestFibonacciOverflowUint64(t *testing.T) {
	// F(94) = 19740274219868223167, which overflows uint64
	expected := new(big.Int)
	expected.SetString("19740274219868223167", 10)

	calculators := map[string]coreCalculator{
		"FastDoubling": &OptimizedFastDoubling{},
		"MatrixExp":    &MatrixExponentiation{},
		"FFTBased":     &FFTBasedCalculator{},
	}

	ctx := context.Background()
	opts := Options{ParallelThreshold: DefaultParallelThreshold}

	for name, calc := range calculators {
		t.Run(name, func(t *testing.T) {
			result, err := calc.CalculateCore(ctx, func(float64) {}, 94, opts)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result.Cmp(expected) != 0 {
				t.Errorf("F(94) incorrect.\nExpected: %s\nGot: %s", expected.String(), result.String())
			}
		})
	}
}

// TestFibonacciLargePowerOfTwo tests F(n) where n is a power of 2.
// These are interesting edge cases for the binary algorithms.
// Values verified against known Fibonacci number databases.
func TestFibonacciLargePowerOfTwo(t *testing.T) {
	testCases := []struct {
		n        uint64
		expected string
	}{
		{64, "10610209857723"},
		{128, "251728825683549488150424261"},
		{256, "141693817714056513234709965875411919657707794958199867"},
	}

	calculators := map[string]coreCalculator{
		"FastDoubling": &OptimizedFastDoubling{},
		"MatrixExp":    &MatrixExponentiation{},
	}

	ctx := context.Background()
	opts := Options{ParallelThreshold: DefaultParallelThreshold}

	for _, tc := range testCases {
		expected := new(big.Int)
		expected.SetString(tc.expected, 10)

		for name, calc := range calculators {
			t.Run(fmt.Sprintf("%s/N=%d", name, tc.n), func(t *testing.T) {
				result, err := calc.CalculateCore(ctx, func(float64) {}, tc.n, opts)
				if err != nil {
					t.Fatalf("Unexpected error for F(%d): %v", tc.n, err)
				}
				if result.Cmp(expected) != 0 {
					t.Errorf("F(%d) incorrect.\nExpected: %s\nGot: %s", tc.n, expected.String(), result.String())
				}
			})
		}
	}
}

// TestFibonacciRecurrenceRelation verifies F(n) = F(n-1) + F(n-2).
func TestFibonacciRecurrenceRelation(t *testing.T) {
	calc := NewCalculator(&OptimizedFastDoubling{})
	ctx := context.Background()
	opts := Options{ParallelThreshold: DefaultParallelThreshold}

	testIndices := []uint64{10, 50, 100, 500, 1000}

	for _, n := range testIndices {
		t.Run("N="+string(rune(n)), func(t *testing.T) {
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
	calc := NewCalculator(&OptimizedFastDoubling{})
	ctx := context.Background()

	// Test with threshold disabled (sequential only)
	t.Run("Sequential", func(t *testing.T) {
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
	calc := NewCalculator(&OptimizedFastDoubling{})
	ctx := context.Background()
	n := uint64(50000)

	thresholds := []int{0, 1000, 10000, 100000, 1000000}

	var results []*big.Int

	for _, threshold := range thresholds {
		t.Run("Threshold="+string(rune(threshold)), func(t *testing.T) {
			opts := Options{
				ParallelThreshold: DefaultParallelThreshold,
				FFTThreshold:      threshold,
			}
			result, err := calc.Calculate(ctx, nil, 0, n, opts)
			if err != nil {
				t.Fatalf("Calculation with FFT threshold %d failed: %v", threshold, err)
			}
			results = append(results, result)
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
	calc := NewCalculator(&MatrixExponentiation{})
	ctx := context.Background()
	n := uint64(10000)

	thresholds := []int{0, 64, 128, 256, 512, 1024, 3072}

	var results []*big.Int

	for _, threshold := range thresholds {
		t.Run("StrassenThreshold="+string(rune(threshold)), func(t *testing.T) {
			opts := Options{
				ParallelThreshold: DefaultParallelThreshold,
				StrassenThreshold: threshold,
			}
			result, err := calc.Calculate(ctx, nil, 0, n, opts)
			if err != nil {
				t.Fatalf("Calculation with Strassen threshold %d failed: %v", threshold, err)
			}
			results = append(results, result)
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
	calculators := map[string]coreCalculator{
		"FastDoubling": &OptimizedFastDoubling{},
		"MatrixExp":    &MatrixExponentiation{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	opts := Options{ParallelThreshold: DefaultParallelThreshold}

	for name, calc := range calculators {
		t.Run(name, func(t *testing.T) {
			_, err := calc.CalculateCore(ctx, func(float64) {}, 1000000, opts)
			if err != context.Canceled {
				t.Errorf("Expected context.Canceled, got: %v", err)
			}
		})
	}
}

// TestContextTimeoutShort tests short timeout behavior.
func TestContextTimeoutShort(t *testing.T) {
	calculators := map[string]coreCalculator{
		"FastDoubling": &OptimizedFastDoubling{},
		"MatrixExp":    &MatrixExponentiation{},
	}

	opts := Options{ParallelThreshold: DefaultParallelThreshold}

	for name, calc := range calculators {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
			defer cancel()

			// Use a large n that won't complete in 1ms
			_, err := calc.CalculateCore(ctx, func(float64) {}, 100_000_000, opts)
			if err != context.DeadlineExceeded {
				// It might complete or get cancelled, both are acceptable
				if err != nil && err != context.Canceled {
					t.Errorf("Expected timeout or cancellation error, got: %v", err)
				}
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Concurrency Safety Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestConcurrentCalculations verifies thread-safety of concurrent calculations.
func TestConcurrentCalculations(t *testing.T) {
	calc := NewCalculator(&OptimizedFastDoubling{})
	ctx := context.Background()
	opts := Options{ParallelThreshold: DefaultParallelThreshold}

	const numGoroutines = 50
	const n = 1000

	expected, err := calc.Calculate(ctx, nil, 0, n, opts)
	if err != nil {
		t.Fatalf("Failed to calculate expected result: %v", err)
	}

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := calc.Calculate(ctx, nil, 0, n, opts)
			if err != nil {
				errors <- err
				return
			}
			if result.Cmp(expected) != 0 {
				errors <- fmt.Errorf("result mismatch: expected %s, got %s", expected.String(), result.String())
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent calculation error: %v", err)
	}
}

// TestConcurrentDifferentN tests concurrent calculations with different N values.
func TestConcurrentDifferentN(t *testing.T) {
	calc := NewCalculator(&OptimizedFastDoubling{})
	ctx := context.Background()
	opts := Options{ParallelThreshold: DefaultParallelThreshold}

	nValues := []uint64{100, 500, 1000, 2000, 5000, 10000}

	var wg sync.WaitGroup
	results := make(map[uint64]*big.Int)
	var mu sync.Mutex

	for _, n := range nValues {
		wg.Add(1)
		go func(n uint64) {
			defer wg.Done()
			result, err := calc.Calculate(ctx, nil, 0, n, opts)
			if err != nil {
				t.Errorf("Failed to calculate F(%d): %v", n, err)
				return
			}
			mu.Lock()
			results[n] = result
			mu.Unlock()
		}(n)
	}

	wg.Wait()

	// Verify results using recurrence relation
	for n, result := range results {
		if n <= 1 {
			continue
		}
		fn1, ok1 := results[n-1]
		fn2, ok2 := results[n-2]
		if ok1 && ok2 {
			expected := new(big.Int).Add(fn1, fn2)
			if result.Cmp(expected) != 0 {
				t.Errorf("F(%d) = %s, but F(%d)+F(%d) = %s",
					n, result.String(), n-1, n-2, expected.String())
			}
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Registry Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestRegistryCreate tests the factory Create method.
func TestRegistryCreate(t *testing.T) {
	factory := NewDefaultFactory()

	t.Run("ValidCalculator", func(t *testing.T) {
		calc, err := factory.Create("fast")
		if err != nil {
			t.Fatalf("Failed to create 'fast' calculator: %v", err)
		}
		if calc == nil {
			t.Fatal("Got nil calculator")
		}
	})

	t.Run("InvalidCalculator", func(t *testing.T) {
		_, err := factory.Create("nonexistent")
		if err == nil {
			t.Fatal("Expected error for nonexistent calculator")
		}
	})
}

// TestRegistryGet tests the factory Get method with caching.
func TestRegistryGet(t *testing.T) {
	factory := NewDefaultFactory()

	calc1, err := factory.Get("fast")
	if err != nil {
		t.Fatalf("First Get failed: %v", err)
	}

	calc2, err := factory.Get("fast")
	if err != nil {
		t.Fatalf("Second Get failed: %v", err)
	}

	// Should return the same cached instance
	if calc1 != calc2 {
		t.Error("Expected cached instance to be returned")
	}
}

// TestRegistryList tests the factory List method.
func TestRegistryList(t *testing.T) {
	factory := NewDefaultFactory()

	list := factory.List()

	expectedNames := []string{"fast", "fft", "matrix"}
	if len(list) != len(expectedNames) {
		t.Errorf("Expected %d calculators, got %d", len(expectedNames), len(list))
	}

	for i, name := range expectedNames {
		if list[i] != name {
			t.Errorf("Expected calculator %d to be %s, got %s", i, name, list[i])
		}
	}
}

// TestRegistryMustGet tests the factory MustGet method.
func TestRegistryMustGet(t *testing.T) {
	factory := NewDefaultFactory()

	t.Run("ValidCalculator", func(t *testing.T) {
		calc := factory.MustGet("fast")
		if calc == nil {
			t.Fatal("Got nil calculator")
		}
	})

	t.Run("InvalidCalculatorPanics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("Expected panic for invalid calculator")
			}
		}()
		factory.MustGet("nonexistent")
	})
}

// TestRegistryCustomCalculator tests registering custom calculators.
func TestRegistryCustomCalculator(t *testing.T) {
	factory := NewDefaultFactory()

	// Register a custom calculator
	factory.Register("custom", func() coreCalculator {
		return &OptimizedFastDoubling{}
	})

	calc, err := factory.Get("custom")
	if err != nil {
		t.Fatalf("Failed to get custom calculator: %v", err)
	}

	// Verify it works
	ctx := context.Background()
	result, err := calc.Calculate(ctx, nil, 0, 10, Options{})
	if err != nil {
		t.Fatalf("Custom calculator failed: %v", err)
	}
	if result.Cmp(big.NewInt(55)) != 0 {
		t.Errorf("Expected 55, got %s", result.String())
	}
}
