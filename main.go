// main.go
//
// This program calculates the n-th Fibonacci number using distinct algorithms:
// 1. Binet's formula (using big.Float for high precision).
// 2. Fast Doubling algorithm.
// 3. Matrix Exponentiation algorithm (2x2 Matrix).
//
// It executes these algorithms concurrently, displays their real-time progress,
// and compares their execution times and results.
// A sync.Pool is used to reduce memory allocations for big.Int objects.
//
// Usage:
//   go run . -n <index> -timeout <duration> [-algorithms <comma_separated_list>]
// Example:
//   go run . -n 100000 -timeout 1m
//   go run . -n 100000 -timeout 1m -algorithms fast,matrix

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/big"
	"sort"
	"strings"
	"sync"
	"time"
)

// ------------------------------------------------------------
// Types and Structures
// ------------------------------------------------------------

// task represents a Fibonacci calculation task to be executed.
type task struct {
	name string  // Name of the algorithm
	fn   fibFunc // Algorithm function
}

// result stores the outcome of a calculation task.
type result struct {
	name     string        // Name of the algorithm
	value    *big.Int      // Calculated Fibonacci value
	duration time.Duration // Duration of the calculation
	err      error         // Potential error
}

// ------------------------------------------------------------
// Main Function: The Orchestrator
// ------------------------------------------------------------
//
// The `main` function orchestrates the entire process:
//  1. It reads command-line parameters (`-n`, `-timeout`, `-algorithms`).
//  2. It defines the list of tasks to execute based on the `-algorithms` flag.
//  3. It creates a `context` with a global timeout to ensure the program
//     doesn't run indefinitely. This context is passed to each calculation goroutine
//     to allow for cooperative cancellation.
//  4. It launches the `progressPrinter` goroutine for real-time display.
//  5. It launches a goroutine for each calculation task. Using goroutines
//     allows all selected algorithms to run concurrently.
//  6. It waits for all tasks to complete using a `sync.WaitGroup`.
//  7. It closes communication channels to signal recipient goroutines
//     (like `progressPrinter`) that there will be no more data.
//  8. Finally, it calls `collectAndDisplayResults` to analyze and present the results.
func main() {
	// 1. Read command-line parameters
	nFlag := flag.Int("n", 100000, "Index n of the Fibonacci term (non-negative integer)")
	timeoutFlag := flag.Duration("timeout", 1*time.Minute, "Global maximum execution time")
	algorithmsFlag := flag.String("algorithms", "all", "Comma-separated list of algorithms to run (e.g., fast,matrix,binet,iterative). 'all' runs all available.")
	flag.Parse()

	n := *nFlag
	timeout := *timeoutFlag

	if n < 0 {
		log.Fatalf("Index n must be greater than or equal to 0. Received: %d", n)
	}

	// 2. Define available tasks
	allAvailableTasks := map[string]fibFunc{
		"Fast Doubling": fibFastDoubling,
		"Matrix 2x2":    fibMatrix,
		"Binet":         fibBinet,
		"Iterative":     fibIterative,
	}

	selectedTaskNames := []string{}
	tasksToRun := []task{}

	if *algorithmsFlag == "all" {
		// Default order includes all known algorithms.
		// Iterative is often slower for large N, so it's placed after faster ones.
		defaultOrder := []string{"Fast Doubling", "Matrix 2x2", "Binet", "Iterative"}
		for _, name := range defaultOrder {
			if fn, ok := allAvailableTasks[name]; ok {
				// Check if already added (e.g. if allAvailableTasks has more than defaultOrder implies)
				isAlreadyAdded := false
				for _, existingTask := range tasksToRun {
					if existingTask.name == name {
						isAlreadyAdded = true
						break
					}
				}
				if !isAlreadyAdded {
					tasksToRun = append(tasksToRun, task{name, fn})
					selectedTaskNames = append(selectedTaskNames, name)
				}
			}
		}
		// Add any other algorithms from allAvailableTasks not in defaultOrder, preserving their map order (which is random)
		// This ensures any newly added algorithm in allAvailableTasks but not yet in defaultOrder gets included with "all"
		for nameInMap, fnInMap := range allAvailableTasks {
			isAlreadyAdded := false
			for _, addedTaskName := range selectedTaskNames {
				if nameInMap == addedTaskName {
					isAlreadyAdded = true
					break
				}
			}
			if !isAlreadyAdded {
				tasksToRun = append(tasksToRun, task{nameInMap, fnInMap})
				selectedTaskNames = append(selectedTaskNames, nameInMap)
			}
		}

	} else {
		algoNamesFromFlag := strings.Split(*algorithmsFlag, ",")
		for _, name := range algoNamesFromFlag {
			trimmedName := strings.TrimSpace(name)
			var foundAlgo fibFunc
			var actualName string
			// Case-insensitive matching for convenience
			for registeredName, fn := range allAvailableTasks {
				if strings.EqualFold(trimmedName, registeredName) {
					foundAlgo = fn
					actualName = registeredName
					break
				}
			}

			if foundAlgo != nil {
				// Avoid duplicates if user specifies an algo multiple times
				isAlreadyAdded := false
				for _, existingTask := range tasksToRun {
					if existingTask.name == actualName {
						isAlreadyAdded = true
						break
					}
				}
				if !isAlreadyAdded {
					tasksToRun = append(tasksToRun, task{actualName, foundAlgo})
					selectedTaskNames = append(selectedTaskNames, actualName)
				}
			} else {
				log.Printf("Warning: Algorithm '%s' not recognized. Skipping.", trimmedName)
			}
		}
	}

	if len(tasksToRun) == 0 {
		log.Fatalf("No algorithms selected or recognized to run. Check the -algorithms flag.")
	}

	log.Printf("Calculating F(%d) with a timeout of %v...", n, timeout)
	log.Printf("Algorithms to run: %s\n", strings.Join(selectedTaskNames, ", "))

	// 3. Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel() // Important to release resources associated with the context

	intPool := newIntPool()

	// Channels for communication between goroutines
	progressAggregatorCh := make(chan progressData, len(tasksToRun)*2) // Buffer size based on number of tasks
	resultsCh := make(chan result, len(tasksToRun))

	// 4. Launch progress display
	var wgDisplay sync.WaitGroup
	wgDisplay.Add(1)
	go func() {
		defer wgDisplay.Done()
		progressPrinter(ctx, progressAggregatorCh, selectedTaskNames)
	}()

	// 5. Launch concurrent calculations
	var wg sync.WaitGroup
	log.Println("Launching concurrent calculations...")

	for _, t := range tasksToRun {
		wg.Add(1)
		go func(currentTask task) {
			defer wg.Done()
			start := time.Now()
			v, err := currentTask.fn(ctx, progressAggregatorCh, n, intPool)
			duration := time.Since(start)
			resultsCh <- result{currentTask.name, v, duration, err}
		}(t)
	}

	// 6. Wait for all calculations to finish
	wg.Wait()
	log.Println("Calculations finished.")

	// 7. Close channels to signal end of transmissions
	close(progressAggregatorCh)
	close(resultsCh)

	// Wait for the display goroutine to finish
	wgDisplay.Wait()

	// 8. Collect and display results
	collectAndDisplayResults(ctx, resultsCh, n)

	log.Println("Program finished.")
}

// collectAndDisplayResults retrieves, sorts, and displays calculation results.
//
// This function is responsible for the final presentation:
//  1. It collects all results from the `resultsCh` channel until it's closed.
//  2. It sorts the results: successes first (by increasing duration), then failures.
//  3. It displays a clear summary table.
//  4. It performs cross-validation: if multiple algorithms succeeded,
//     it checks that they all produced the same result.
//  5. It highlights the winning algorithm and displays details about the calculated number.
func collectAndDisplayResults(ctx context.Context, resultsCh <-chan result, n int) {
	var results []result
	// This for-range loop reads from the channel until it's closed and empty.
	for r := range resultsCh {
		if r.err != nil {
			// Distinguish a timeout from other errors for a clearer message.
			if err := ctx.Err(); err == context.DeadlineExceeded && r.err == context.DeadlineExceeded {
				log.Printf("⚠️ Task '%s' was interrupted by the global timeout after %v", r.name, r.duration.Round(time.Microsecond))
				// r.err is already context.DeadlineExceeded
			} else if r.err == context.DeadlineExceeded {
				// Task itself might have returned ctx.Err() before global timeout if it checks ctx.Done()
				log.Printf("⚠️ Task '%s' self-terminated due to context cancellation (possibly timeout) after %v", r.name, r.duration.Round(time.Microsecond))
			} else {
				log.Printf("❌ Error for task '%s': %v (duration: %v)", r.name, r.err, r.duration.Round(time.Microsecond))
			}
		}
		results = append(results, r)
	}

	// Sort results: successes by duration, then failures.
	sort.Slice(results, func(i, j int) bool {
		if results[i].err == nil && results[j].err != nil {
			return true // i is a success, j is a failure -> i comes first
		}
		if results[i].err != nil && results[j].err == nil {
			return false // i is a failure, j is a success -> j comes first
		}
		// Both are successes or both are failures -> sort by duration
		return results[i].duration < results[j].duration
	})

	fmt.Println("\n--------------------------- ORDERED RESULTS ---------------------------")
	var firstSuccessfulResult *result
	allValidResultsIdentical := true
	successfulResultsCount := 0

	for i, r := range results {
		status := "OK"
		valStr := "N/A"
		if r.err != nil {
			if r.err == context.DeadlineExceeded {
				status = "Timeout"
			} else {
				status = fmt.Sprintf("Error: %v", r.err)
			}
		} else if r.value != nil {
			successfulResultsCount++
			// Display an abbreviated version for very large numbers
			if len(r.value.String()) > 15 {
				valStr = r.value.String()[:5] + "..." + r.value.String()[len(r.value.String())-5:]
			} else {
				valStr = r.value.String()
			}

			// Cross-validation of results
			if firstSuccessfulResult == nil {
				firstSuccessfulResult = &results[i] // Store pointer to the element in the slice
			} else if r.value.Cmp(firstSuccessfulResult.value) != 0 {
				allValidResultsIdentical = false
			}
		}
		fmt.Printf("%-16s : %-12v [%-14s] Result: %s\n", r.name, r.duration.Round(time.Microsecond), status, valStr)
	}

	fmt.Println("------------------------------------------------------------------------")

	if firstSuccessfulResult != nil {
		fmt.Printf("\n🏆 Fastest algorithm (that succeeded): %s (%v)\n", firstSuccessfulResult.name, firstSuccessfulResult.duration.Round(time.Microsecond))
		printFibResultDetails(firstSuccessfulResult.value, n)
		if successfulResultsCount > 1 {
			if allValidResultsIdentical {
				fmt.Println("✅ All valid results produced are identical.")
			} else {
				fmt.Println("❌ DISCREPANCY! Results from successful algorithms differ.")
			}
		} else {
			fmt.Println("ℹ️ Only one algorithm succeeded, no cross-validation possible.")
		}
	} else {
		fmt.Println("\nNo algorithm could complete the calculation successfully.")
	}
}

// printFibResultDetails displays detailed information about the calculated Fibonacci number.
func printFibResultDetails(value *big.Int, n int) {
	if value == nil {
		return
	}

	digits := len(value.Text(10))
	fmt.Printf("Number of digits in F(%d): %d\n", n, digits)

	// Use scientific notation for numbers too large to display.
	if digits > 20 {
		floatVal := new(big.Float).SetPrec(uint(digits + 10)).SetInt(value)
		sci := floatVal.Text('e', 8) // 8 digits of precision for scientific notation
		fmt.Printf("Value (scientific notation) ≈ %s\n", sci)
	} else {
		fmt.Printf("Value = %s\n", value.Text(10))
	}
}
