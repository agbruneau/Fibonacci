package main

import (
	"errors"
	"fmt"
	"math/big"
	"math/bits"
	"sync"
	"time"
)

const MAX_FIB_VALUE = 100000001 // Maximum value of n that can be calculated
var two = big.NewInt(2)        // Constant value 2 as a big.Int for calculations

// Optimized memoization map with better concurrency control
var memo = &sync.Map{}
var memoMutex = &sync.Mutex{} // Mutex to provide additional concurrency control

// fibDoubling calculates the nth Fibonacci number using the doubling method
func fibDoubling(n int) (*big.Int, error) {
	// Return the value directly if n is 0 or 1
	if n < 2 {
		return big.NewInt(int64(n)), nil
	} else if n > MAX_FIB_VALUE {
		// Error if the value is too large to calculate within reasonable time
		return nil, errors.New("n is too large for this implementation")
	}
	// Calculate the Fibonacci value using an iterative helper
	result := fibDoublingHelperIterative(n)
	return result, nil
}

// fibDoublingHelperIterative is an iterative function that uses the doubling method to compute Fibonacci numbers
func fibDoublingHelperIterative(n int) *big.Int {
	// Use a mutex to prevent race conditions when accessing the memoization map
	memoMutex.Lock()
	if val, exists := memo.Load(n); exists {
		// If the value is already cached, return it to save computation time
		memoMutex.Unlock()
		return val.(*big.Int)
	}
	memoMutex.Unlock()

	// Initialize base Fibonacci values F(0) = 0 and F(1) = 1
	a, b := big.NewInt(0), big.NewInt(1)
	c, d := new(big.Int), new(big.Int) // Preallocate big.Int variables to reuse for calculations

	// Determine the number of bits needed to represent n
	bitLength := bits.Len(uint(n))

	// Iterate over each bit from the most significant to the least significant
	for i := bitLength - 1; i >= 0; i-- {
		// Use the doubling formulas:
		// F(2k) = F(k) * [2 * F(k+1) - F(k)]
		c.Mul(b, two)     // c = 2 * F(k+1)
		c.Sub(c, a)       // c = 2 * F(k+1) - F(k)
		c.Mul(a, c)       // c = F(k) * (2 * F(k+1) - F(k))
		// F(2k + 1) = F(k)^2 + F(k+1)^2
		d.Mul(a, a)       // d = F(k)^2
		d.Add(d, new(big.Int).Mul(b, b)) // d = F(k)^2 + F(k+1)^2

		// Update a and b based on the current bit of n
		if (n>>i)&1 == 0 {
			a.Set(c) // If the bit is 0, set F(2k) to a
			b.Set(d) // Set F(2k+1) to b
		} else {
			a.Set(d) // If the bit is 1, set F(2k+1) to a
			b.Add(c, d) // Set F(2k + 2) to b
		}
	}

	// Cache the result locally before storing it in the memoization map
	result := new(big.Int).Set(a)

	// Store the computed value in the memoization map for future use
	memoMutex.Lock()
	memo.Store(n, result)
	memoMutex.Unlock()
	return result
}

// printError prints an error message in a consistent format
func printError(n int, err error) {
	fmt.Printf("fibDoubling(%d): %s\n", n, err)
}

// benchmarkFib benchmarks the Fibonacci calculations for a list of values
func benchmarkFib(nValues []int, repetitions int) {
	// Clear the memoization map before benchmarking to ensure consistent results
	memo.Range(func(key, value interface{}) bool {
		memo.Delete(key)
		return true
	})

	var wg sync.WaitGroup // WaitGroup to manage concurrency

	for _, n := range nValues {
		// Ensure that wg.Add(1) is called outside the goroutine
		wg.Add(1)
		// Launch a goroutine to calculate Fibonacci concurrently
		go func(n int) {
			defer wg.Done() // Mark this goroutine as done when it completes

			totalExecTime := big.NewInt(0)
			// Repeat the calculation for more accurate benchmarking
			for i := 0; i < repetitions; i++ {
				start := time.Now()
				_, err := fibDoubling(n)
				if err != nil {
					// Print an error message if n is too large
					printError(n, err)
					continue
				}
				// Accumulate the execution time in nanoseconds
				totalExecTime.Add(totalExecTime, big.NewInt(time.Since(start).Nanoseconds()))
			}
			// Calculate the average execution time
			execTime := new(big.Int).Div(totalExecTime, big.NewInt(int64(repetitions)))
			// Print the average execution time for the given value of n
			fmt.Printf("fibDoubling(%d) averaged over %d runs: %s nanoseconds\n", n, repetitions, execTime.String())
		}(n)
	}

	// Wait for all goroutines to complete
	wg.Wait()
}

// main function to execute the benchmarking
func main() {
	// Define the list of values for which to benchmark the Fibonacci calculation
	nValues := []int{1000000, 10000000, 100000000} // List of values to benchmark
	// Define the number of repetitions for better accuracy
	repetitions := 3                                // Number of repetitions for better accuracy
	// Run the benchmark
	benchmarkFib(nValues, repetitions)
}
