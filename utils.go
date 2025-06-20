package main

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
)

// ------------------------------------------------------------
// Progress Display Management
// ------------------------------------------------------------

const progressRefreshInterval = 100 * time.Millisecond

// progressData encapsulates progress information for a task.
// This is the canonical definition.
type progressData struct {
	name string  // Name of the task
	pct  float64 // Percentage of progress
}

// progressPrinter manages consolidated progress display for all tasks.
// It refreshes the display at regular intervals or upon receiving new data.
//
// Concept:
// A dedicated goroutine continuously listens on a shared channel (progress).
// It collects percentages from each task and refreshes a single line
// on the terminal to display the overall status. The `\r` (carriage return) trick
// allows rewriting on the same line, creating a smooth progress animation.
func progressPrinter(ctx context.Context, progress <-chan progressData, taskNames []string) {
	status := make(map[string]float64)
	for _, name := range taskNames {
		status[name] = 0.0 // Initialize progress of each task to 0%
	}

	ticker := time.NewTicker(progressRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case p, ok := <-progress:
			if !ok { // Channel is closed, signifies end of progress updates.
				printStatus(status, taskNames) // Print one last time
				fmt.Println()                  // Move to a new line after all progress is done
				return
			}
			status[p.name] = p.pct
			printStatus(status, taskNames) // Print current status

		case <-ticker.C:
			// Periodically refresh display to show the program is still active,
			// even if no new progress updates have been received.
			printStatus(status, taskNames)

		case <-ctx.Done():
			// Main context is done (e.g., timeout or cancellation), stop displaying.
			// Print one last status before exiting, then a newline.
			printStatus(status, taskNames)
			fmt.Println()
			return
		}
	}
}

// printStatus displays the current progress status for each task on a single line.
func printStatus(status map[string]float64, keys []string) {
	var b strings.Builder
	b.WriteString("\r") // Carriage return to overwrite the previous line

	for i, k := range keys {
		if i > 0 {
			b.WriteString("   ") // Separator between tasks
		}
		// Format string for aligned display: Task Name: XX.YY%
		fmt.Fprintf(&b, "%-15s %6.2f%%", k+":", status[k])
	}
	// Add trailing spaces to clear any remnants of a longer previous line.
	// Adjust the number of spaces if task names or formatting changes significantly.
	b.WriteString("                    ") // Increased padding
	fmt.Print(b.String())
}

// ------------------------------------------------------------
// *big.Int Object Pool for Memory Reuse
// ------------------------------------------------------------
//
// Memory Optimization Concept (sync.Pool):
// Calculations for large Fibonacci numbers require handling integers
// that exceed the capacity of standard types (e.g., int64). Go's `math/big.Int` is used.
// The problem: Creating numerous `big.Int` objects, especially in loops for complex
// algorithms, puts significant pressure on the Garbage Collector (GC). Frequent GC cycles
// can pause the program and degrade performance.
// The solution: A `sync.Pool` provides a way to reuse objects that are otherwise
// short-lived. Instead of allocating a new `big.Int` each time one is needed,
// the program requests one from the pool. After the object is used, it's returned
// to the pool. This drastically reduces the number of allocations and, consequently,
// the GC overhead, leading to improved performance for memory-intensive operations.

// newIntPool creates a new sync.Pool specifically for *big.Int objects.
// The New function in the pool is called when Get is invoked on an empty pool.
func newIntPool() *sync.Pool {
	return &sync.Pool{
		New: func() interface{} {
			// Allocate a new *big.Int instance when the pool is empty.
			return new(big.Int)
		},
	}
}
