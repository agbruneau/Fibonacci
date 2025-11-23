package fibonacci

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"example.com/fibcalc/internal/config"
)

// knownFibResults is a test oracle containing reference values
// for the Fibonacci sequence, used to validate the accuracy of the calculations.
var knownFibResults = []struct {
	n      uint64
	result string
}{
	{0, "0"}, {1, "1"}, {2, "1"}, {10, "55"}, {20, "6765"},
	{50, "12586269025"}, {92, "7540113804746346429"},
	{93, "12200160415121876738"}, {100, "354224848179261915075"},
	{1000, "43466557686937456435688527675040625802564660517371780402481729089536555417949051890403879840079255169295922593080322634775209689623239873322471161642996440906533187938298969649928516003704476137795166849228875"},
}

// TestFibonacciCalculators systematically validates all implementations
// of `Calculator` against the `knownFibResults` test oracle.
func TestFibonacciCalculators(t *testing.T) {
	ctx := context.Background()
	calculators := map[string]Calculator{
		"FastDoubling": NewCalculator(&OptimizedFastDoubling{}),
		"MatrixExp":    NewCalculator(&MatrixExponentiation{}),
		"FFTBased":     NewCalculator(&FFTBasedCalculator{}),
	}

	for name, calc := range calculators {
		t.Run(name, func(t *testing.T) {
			for _, testCase := range knownFibResults {
				t.Run(fmt.Sprintf("N=%d", testCase.n), func(t *testing.T) {
					t.Parallel()
					expected := new(big.Int)
					expected.SetString(testCase.result, 10)
					result, err := calc.Calculate(ctx, nil, 0, testCase.n, config.DefaultParallelThreshold, 0)

					if err != nil {
						t.Fatalf("Unexpected error: %v", err)
					}
					if result == nil {
						t.Fatal("Nil result returned without an error.")
					}
					if result.Cmp(expected) != 0 {
						t.Errorf("Incorrect result.\nExpected: %s\nGot: %s", expected.String(), result.String())
					}
				})
			}
		})
	}
}

// TestProgressCalculationLogic validates that progress is not just monotonic,
// but also accurately reflects the work done.
func TestProgressCalculationLogic(t *testing.T) {
	const n = 100_000 // A sufficiently large number for the test
	calc := NewCalculator(&OptimizedFastDoubling{})
	progressChan := make(chan ProgressUpdate, 200)
	var progressUpdates []float64
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for update := range progressChan {
			progressUpdates = append(progressUpdates, update.Value)
		}
	}()

	_, err := calc.Calculate(context.Background(), progressChan, 0, n, config.DefaultParallelThreshold, 0)
	close(progressChan)
	wg.Wait()

	if err != nil {
		t.Fatalf("Calculation failed: %v", err)
	}

	if len(progressUpdates) < 2 {
		t.Fatal("Insufficient progress updates received to validate logic.")
	}

	// The work for the first half of the bits (MSB for Fast Doubling, or LSB for Matrix Exp)
	// corresponds to operations on small numbers. The work grows exponentially as we proceed.
	// Therefore, progress should be slow initially and accelerate towards the end.
	// We expect the first reported progress (after the initial 0.0) to be small.
	firstReportedProgress := progressUpdates[0]
	if firstReportedProgress > 0.25 {
		t.Errorf("Expected initial progress to be low, reflecting the small work of the early steps."+
			" Got: %f. This may indicate inverted progress logic.", firstReportedProgress)
	}
}

// TestLookupTableImmutability verifies that the lookup table (LUT)
// ensures the immutability of its data by returning copies.
func TestLookupTableImmutability(t *testing.T) {
	val1 := lookupSmall(10)
	expected := big.NewInt(55)
	if val1.Cmp(expected) != 0 {
		t.Fatalf("Incorrect F(10) value. Expected 55, got %s", val1.String())
	}
	val1.Add(val1, big.NewInt(1))
	val2 := lookupSmall(10)
	if val2.Cmp(expected) != 0 {
		t.Fatalf("Immutability violation: LUT modified. Expected F(10) to be 55, got %s", val2.String())
	}
}

// TestNilCoreCalculatorPanic verifies that `NewCalculator` panics if called
// with a nil `coreCalculator`.
func TestNilCoreCalculatorPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("NewCalculator should have panicked with a nil core.")
		}
	}()
	_ = NewCalculator(nil)
}

// TestProgressReporter validates the monotonic notification of progress.
func TestProgressReporter(t *testing.T) {
	calculators := map[string]Calculator{
		"FastDoubling": NewCalculator(&OptimizedFastDoubling{}),
		"MatrixExp":    NewCalculator(&MatrixExponentiation{}),
	}

	for name, calc := range calculators {
		t.Run(name, func(t *testing.T) {
			progressChan := make(chan ProgressUpdate, 200)
			var lastProgress float64
			var wg sync.WaitGroup
			wg.Add(1)

			go func() {
				defer wg.Done()
				for update := range progressChan {
					if update.Value < lastProgress {
						t.Errorf("Non-monotonic progress. Previous: %f, Current: %f", lastProgress, update.Value)
					}
					lastProgress = update.Value
				}
			}()

			_, err := calc.Calculate(context.Background(), progressChan, 0, 10000, config.DefaultParallelThreshold, 0)
			close(progressChan)
			wg.Wait()

			if err != nil {
				t.Fatalf("Calculation failed: %v", err)
			}
			if lastProgress != 1.0 {
				t.Errorf("Final progress expected to be 1.0, got %f", lastProgress)
			}
		})
	}
}

// TestContextCancellation verifies the responsiveness of the algorithms to a
// context cancellation.
func TestContextCancellation(t *testing.T) {
	calculators := map[string]Calculator{
		"FastDoubling": NewCalculator(&OptimizedFastDoubling{}),
		"MatrixExp":    NewCalculator(&MatrixExponentiation{}),
	}

	for name, calc := range calculators {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			_, err := calc.Calculate(ctx, nil, 0, 100_000_000, config.DefaultParallelThreshold, 0)
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("Expected error: %v, Got: %v", context.DeadlineExceeded, err)
			}
		})
	}
}

func runBenchmark(b *testing.B, calc Calculator, n uint64) {
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = calc.Calculate(ctx, nil, 0, n, config.DefaultParallelThreshold, 0)
	}
}

func BenchmarkFastDoubling1M(b *testing.B) {
	runBenchmark(b, NewCalculator(&OptimizedFastDoubling{}), 1_000_000)
}

func BenchmarkMatrixExp1M(b *testing.B) {
	runBenchmark(b, NewCalculator(&MatrixExponentiation{}), 1_000_000)
}

func BenchmarkFastDoubling10M(b *testing.B) {
	runBenchmark(b, NewCalculator(&OptimizedFastDoubling{}), 10_000_000)
}

func BenchmarkMatrixExp10M(b *testing.B) {
	runBenchmark(b, NewCalculator(&MatrixExponentiation{}), 10_000_000)
}

func BenchmarkFFTBased1M(b *testing.B) {
	runBenchmark(b, NewCalculator(&FFTBasedCalculator{}), 1_000_000)
}

func BenchmarkFFTBased10M(b *testing.B) {
	runBenchmark(b, NewCalculator(&FFTBasedCalculator{}), 10_000_000)
}

// ExampleCalculator_Calculate illustrates the basic use of a Calculator
// to calculate a Fibonacci number.
func ExampleCalculator_Calculate() {
	// Create a new calculator with the Fast Doubling algorithm.
	calculator := NewCalculator(&OptimizedFastDoubling{})

	// Calculate the 20th Fibonacci number.
	result, err := calculator.Calculate(context.Background(), nil, 0, 20, config.DefaultParallelThreshold, 0)
	if err != nil {
		fmt.Printf("Calculation error: %v\n", err)
		return
	}

	fmt.Println(result)
	// Output: 6765
}
