package fibonacci

import (
	"context"
	"fmt"

	"github.com/agbruneau/FibGo/internal/progress"
)

// ExampleNewCalculator demonstrates creating a Calculator with
// different algorithm implementations.
func ExampleMustNewCalculator() {
	// Create calculators for each algorithm.
	fast := MustNewCalculator(&FastDoublingCalculator{})
	matrix := MustNewCalculator(&MatrixExponentiationCalculator{})
	fft := MustNewCalculator(&FFTBasedCalculator{})

	fmt.Println(fast.Name())
	fmt.Println(matrix.Name())
	fmt.Println(fft.Name())
	// Output:
	// Fast Doubling (O(log n), Parallel, Zero-Alloc)
	// Matrix Exponentiation (O(log n), Parallel, Zero-Alloc)
	// FFT-Based Doubling
}

// ExampleDefaultFactory demonstrates using the factory to obtain
// pre-registered calculators by name.
func ExampleDefaultFactory() {
	factory := NewDefaultFactory()

	// List available algorithms.
	fmt.Println(factory.List())

	// Get a calculator by name.
	calc, err := factory.Get("fast")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	result, err := calc.Calculate(context.Background(), nil, 0, 10, Options{})
	if err != nil {
		fmt.Printf("Calculation error: %v\n", err)
		return
	}

	fmt.Println(result)
	// Output:
	// [fast fft matrix]
	// 55
}

// ExampleFibCalculator_CalculateWithObservers demonstrates observer-based
// progress tracking during a calculation.
func ExampleFibCalculator_CalculateWithObservers() {
	calc := MustNewCalculator(&FastDoublingCalculator{}).(*FibCalculator)

	// Create a subject with a channel observer.
	subject := progress.NewProgressSubject()
	progressChan := make(chan progress.ProgressUpdate, 100)
	subject.Register(progress.NewChannelObserver(progressChan))

	result, err := calc.CalculateWithObservers(
		context.Background(), subject, 0, 50, Options{},
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Drain the progress channel.
	close(progressChan)
	var lastProgress float64
	for update := range progressChan {
		lastProgress = update.Value
	}

	fmt.Println(result)
	fmt.Println(lastProgress)
	// Output:
	// 12586269025
	// 1
}

// Example_smallValues shows that small Fibonacci values (n <= 93)
// are computed via the optimized iterative path.
func Example_smallValues() {
	calc := MustNewCalculator(&FastDoublingCalculator{})

	// F(0) through F(93) use the fast iterative path.
	for _, n := range []uint64{0, 1, 2, 10, 93} {
		result, _ := calc.Calculate(context.Background(), nil, 0, n, Options{})
		fmt.Printf("F(%d) = %s\n", n, result)
	}
	// Output:
	// F(0) = 0
	// F(1) = 1
	// F(2) = 1
	// F(10) = 55
	// F(93) = 12200160415121876738
}
