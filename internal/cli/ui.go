// The cli package provides functions for building a command-line interface (CLI)
// for the Fibonacci calculation application. It handles the asynchronous
// display of calculation progress and formats the results for a clear and
// readable presentation.
package cli

import (
	"fmt"
	"io"
	"math/big"
	"strings"
	"sync"
	"time"

	"example.com/fibcalc/internal/fibonacci"
)

const (
	// ProgressRefreshRate defines the refresh frequency of the progress bar.
	ProgressRefreshRate = 100 * time.Millisecond
	// ProgressBarWidth defines the width in characters of the progress bar.
	ProgressBarWidth = 40
	// TruncationLimit is the digit threshold from which a result is truncated.
	TruncationLimit = 100
	// DisplayEdges specifies the number of digits to display at the beginning
	// and end of a truncated number.
	DisplayEdges = 25
)

// ProgressState encapsulates the aggregated progress of one or more concurrent
// calculations. It maintains the individual progress of each calculator and
// provides methods to compute the average and display a progress bar.
type ProgressState struct {
	progresses     []float64
	numCalculators int
	out            io.Writer
}

// NewProgressState is a factory function that initializes and returns a new
// `ProgressState`. It requires the total number of calculators to track and the
// output writer for displaying the progress bar.
func NewProgressState(numCalculators int, out io.Writer) *ProgressState {
	return &ProgressState{
		progresses:     make([]float64, numCalculators),
		numCalculators: numCalculators,
		out:            out,
	}
}

// Update records a new progress value for a specific calculator, identified by
// its index. This method is safe for concurrent use.
func (ps *ProgressState) Update(index int, value float64) {
	if index >= 0 && index < len(ps.progresses) {
		ps.progresses[index] = value
	}
}

// CalculateAverage computes the average progress across all tracked calculators.
// This is used to display a single, aggregated progress bar when multiple
// algorithms are run in parallel.
func (ps *ProgressState) CalculateAverage() float64 {
	var totalProgress float64
	for _, p := range ps.progresses {
		totalProgress += p
	}
	if ps.numCalculators == 0 {
		return 0.0
	}
	return totalProgress / float64(ps.numCalculators)
}

// PrintBar renders and displays the current state of the progress bar to the
// configured output writer. It can be a final print (with a newline) or an
// in-place update.
func (ps *ProgressState) PrintBar(final bool) {
	avgProgress := ps.CalculateAverage()
	label := "Progress"
	if ps.numCalculators > 1 {
		label = "Average Progress"
	}
	bar := progressBar(avgProgress, ProgressBarWidth)
	fmt.Fprintf(ps.out, "\r\033[K%s : %6.2f%% [%s]", label, avgProgress*100, bar)
	if final {
		fmt.Fprintln(ps.out)
	}
}

// DisplayAggregateProgress manages the asynchronous display of a progress bar. It
// is designed to run in a dedicated goroutine. It listens for `ProgressUpdate`
// messages on a channel, aggregates them in a `ProgressState`, and periodically
// refreshes the progress bar on the screen. The function ensures the final state
// of the bar is printed before exiting.
func DisplayAggregateProgress(wg *sync.WaitGroup, progressChan <-chan fibonacci.ProgressUpdate, numCalculators int, out io.Writer) {
	defer wg.Done()
	if numCalculators <= 0 {
		// Drain the channel to prevent sender goroutines from blocking.
		for range progressChan {
		}
		return
	}

	state := NewProgressState(numCalculators, out)
	ticker := time.NewTicker(ProgressRefreshRate)
	defer ticker.Stop()

	for {
		select {
		case update, ok := <-progressChan:
			if !ok {
				// Channel closed, print the final bar and exit.
				state.PrintBar(true)
				return
			}
			state.Update(update.CalculatorIndex, update.Value)
		case <-ticker.C:
			// Refresh the bar periodically.
			state.PrintBar(false)
		}
	}
}

// progressBar generates a string representing a textual progress bar.
func progressBar(progress float64, length int) string {
	if progress > 1.0 {
		progress = 1.0
	}
	if progress < 0.0 {
		progress = 0.0
	}
	count := int(progress * float64(length))
	var builder strings.Builder
	builder.Grow(length)
	for i := 0; i < length; i++ {
		if i < count {
			builder.WriteRune('█')
		} else {
			builder.WriteRune('░')
		}
	}
	return builder.String()
}

// DisplayResult formats and prints the final calculation result to the specified
// output writer. It provides different levels of detail based on the `verbose`
// and `details` flags, including metadata like binary size, number of digits,
// and scientific notation. For very large numbers, it truncates the output
// unless `verbose` is true.
func DisplayResult(result *big.Int, n uint64, duration time.Duration, verbose, details bool, out io.Writer) {
	bitLen := result.BitLen()
	fmt.Fprintf(out, "Binary Size of the Result: %s bits.\n", formatNumberString(fmt.Sprintf("%d", bitLen)))

	if !details {
		fmt.Fprintln(out, "(Use the -d or --details option for a full report)")
		return
	}

	fmt.Fprintln(out, "\n--- Detailed Result Analysis ---")
	if duration > 0 {
		fmt.Fprintf(out, "Calculation time      : %s\n", duration)
	}

	resultStr := result.String()
	numDigits := len(resultStr)
	fmt.Fprintf(out, "Number of digits    : %s\n", formatNumberString(fmt.Sprintf("%d", numDigits)))

	if numDigits > 6 {
		f := new(big.Float).SetInt(result)
		fmt.Fprintf(out, "Scientific notation : %.6e\n", f)
	}

	fmt.Fprintln(out, "\n--- Calculated Value ---")
	if verbose {
		fmt.Fprintf(out, "F(%d) =\n%s\n", n, formatNumberString(resultStr))
	} else if numDigits > TruncationLimit {
		fmt.Fprintf(out, "F(%d) (truncated) = %s...%s\n", n, resultStr[:DisplayEdges], resultStr[numDigits-DisplayEdges:])
		fmt.Fprintln(out, "(Use the -v or --verbose option to display the full value)")
	} else {
		fmt.Fprintf(out, "F(%d) = %s\n", n, formatNumberString(resultStr))
	}
}

// formatNumberString inserts thousand separators into a numeric string.
func formatNumberString(s string) string {
	if len(s) == 0 {
		return ""
	}
	prefix := ""
	if s[0] == '-' {
		prefix = "-"
		s = s[1:]
	}
	n := len(s)
	if n <= 3 {
		return prefix + s
	}

	var builder strings.Builder
	builder.Grow(len(prefix) + n + (n-1)/3)
	builder.WriteString(prefix)

	firstGroupLen := n % 3
	if firstGroupLen == 0 {
		firstGroupLen = 3
	}
	builder.WriteString(s[:firstGroupLen])

	for i := firstGroupLen; i < n; i += 3 {
		builder.WriteByte(',')
		builder.WriteString(s[i : i+3])
	}
	return builder.String()
}