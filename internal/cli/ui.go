// The cli package provides functions for building a command-line interface (CLI)
// for the Fibonacci calculation application. It handles the asynchronous
// display of calculation progress and formats the results for a clear and
// readable presentation.
package cli

//go:generate mockgen -source=ui.go -destination=mocks/mock_ui.go -package=mocks

import (
	"fmt"
	"io"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/agbru/fibcalc/internal/fibonacci"
	"github.com/agbru/fibcalc/internal/ui"
	"github.com/briandowns/spinner"
)

// FormatExecutionDuration formats a time.Duration for display.
// It shows microseconds for durations less than a millisecond, milliseconds for
// durations less than a second, and the default string representation otherwise.
// This approach provides a more human-readable output for short durations.
//
// Parameters:
//   - d: The duration to format.
//
// Returns:
//   - string: A formatted string representing the duration.
func FormatExecutionDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	} else if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return d.String()
}

const (
	// TruncationLimit is the digit threshold from which a result is truncated
	// in standard output to avoid cluttering the terminal.
	TruncationLimit = 100
	// DisplayEdges specifies the number of digits to display at the beginning
	// and end of a truncated number.
	DisplayEdges = 25
	// ProgressRefreshRate defines the refresh frequency of the progress bar.
	// Optimized to 200ms to reduce updates and improve performance.
	ProgressRefreshRate = 200 * time.Millisecond
	// ProgressBarWidth defines the width in characters of the progress bar.
	ProgressBarWidth = 40
)

// Color functions return ANSI escape codes from the current theme.
// These provide backward compatibility while allowing theme switching.
// They delegate to the ui package to reduce coupling.

// ColorReset returns the reset escape code from the current theme.
func ColorReset() string { return ui.GetCurrentTheme().Reset }

// ColorRed returns the error color from the current theme.
func ColorRed() string { return ui.GetCurrentTheme().Error }

// ColorGreen returns the success color from the current theme.
func ColorGreen() string { return ui.GetCurrentTheme().Success }

// ColorYellow returns the warning color from the current theme.
func ColorYellow() string { return ui.GetCurrentTheme().Warning }

// ColorBlue returns the primary color from the current theme.
func ColorBlue() string { return ui.GetCurrentTheme().Primary }

// ColorMagenta returns the info color from the current theme.
func ColorMagenta() string { return ui.GetCurrentTheme().Info }

// ColorCyan returns the secondary color from the current theme.
func ColorCyan() string { return ui.GetCurrentTheme().Secondary }

// ColorBold returns the bold escape code from the current theme.
func ColorBold() string { return ui.GetCurrentTheme().Bold }

// ColorUnderline returns the underline escape code from the current theme.
func ColorUnderline() string { return ui.GetCurrentTheme().Underline }

// Spinner is an interface that abstracts the behavior of a terminal spinner.
// This allows for the decoupling of the `DisplayProgress` function from a
// specific spinner implementation, facilitating easier testing and maintenance.
// It defines the essential controls for a spinner: starting, stopping, and
// updating its status message.
type Spinner interface {
	// Start begins the spinner animation.
	Start()
	// Stop halts the spinner animation.
	Stop()
	// UpdateSuffix sets the text that is displayed after the spinner.
	//
	// Parameters:
	//   - suffix: The text string to display.
	UpdateSuffix(suffix string)
}

// realSpinner is a wrapper for the `spinner.Spinner` that implements the
// `Spinner` interface. This adapter allows the `spinner` library to be used
// within the application's CLI framework.
type realSpinner struct {
	s *spinner.Spinner
}

// Start begins the spinner animation.
func (rs *realSpinner) Start() {
	rs.s.Start()
}

// Stop halts the spinner animation.
func (rs *realSpinner) Stop() {
	rs.s.Stop()
}

// UpdateSuffix sets the text that is displayed after the spinner.
//
// Parameters:
//   - suffix: The string to display.
func (rs *realSpinner) UpdateSuffix(suffix string) {
	rs.s.Suffix = suffix
}

var newSpinner = func(options ...spinner.Option) Spinner {
	// Using the same interval as ProgressRefreshRate to synchronize
	s := spinner.New(spinner.CharSets[11], ProgressRefreshRate, options...)
	return &realSpinner{s}
}

// ProgressState encapsulates the aggregated progress of concurrent calculations.
// It maintains the individual progress of each calculator and computes the
// average, which is essential for providing a consolidated progress view when
// multiple algorithms are running in parallel.
type ProgressState struct {
	progresses     []float64
	numCalculators int
}

// NewProgressState creates and initializes a new ProgressState.
// It sets up the internal storage for tracking the progress of a specified
// number of calculators.
//
// Parameters:
//   - numCalculators: The number of calculators to track.
//
// Returns:
//   - *ProgressState: A pointer to the new progress state object.
func NewProgressState(numCalculators int) *ProgressState {
	return &ProgressState{
		progresses:     make([]float64, numCalculators),
		numCalculators: numCalculators,
	}
}

// Update records a new progress value for a specific calculator.
// It is designed to be safe for concurrent use, although in the current
// implementation it is called sequentially. The method ensures that updates are
// only applied for valid calculator indices.
//
// Parameters:
//   - index: The index of the calculator (0 to numCalculators-1).
//   - value: The progress value (0.0 to 1.0).
func (ps *ProgressState) Update(index int, value float64) {
	if index >= 0 && index < len(ps.progresses) {
		ps.progresses[index] = value
	}
}

// CalculateAverage computes the average progress across all tracked calculators.
// This is used to display a single, consolidated progress bar to the user,
// representing the overall progress of the application.
//
// Returns:
//   - float64: The average progress (0.0 to 1.0).
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

// progressBar generates a string representing a textual progress bar.
//
// Parameters:
//   - progress: The normalized progress value (0.0 to 1.0).
//   - length: The total character width of the progress bar.
//
// Returns:
//   - string: A string representation of the progress bar.
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

// DisplayProgress manages the asynchronous display of a spinner and progress bar.
// It is designed to run in a dedicated goroutine and orchestrates the UI updates
// for the duration of the calculations.
//
// The function's responsibilities include:
//   - Receiving progress updates from a channel.
//   - Aggregating these updates to calculate the average progress.
//   - Calculating and displaying the estimated time remaining (ETA).
//   - Periodically refreshing the spinner and progress bar.
//   - Gracefully shutting down when the progress channel is closed.
//
// Parameters:
//   - wg: A WaitGroup to signal when the display routine is complete.
//   - progressChan: The channel receiving progress updates.
//   - numCalculators: The number of calculators contributing to the progress.
//   - out: The io.Writer to which the progress bar is rendered.
func DisplayProgress(wg *sync.WaitGroup, progressChan <-chan fibonacci.ProgressUpdate, numCalculators int, out io.Writer) {
	defer wg.Done()
	if numCalculators <= 0 {
		for range progressChan { // Drain the channel
		}
		return
	}

	state := NewProgressWithETA(numCalculators)
	s := newSpinner(spinner.WithWriter(out))
	s.Start()
	spinnerStopped := false
	defer func() {
		if !spinnerStopped {
			s.Stop()
		}
	}()

	ticker := time.NewTicker(ProgressRefreshRate)
	defer ticker.Stop()

	for {
		select {
		case update, ok := <-progressChan:
			if !ok {
				// Stop the spinner first to free the line
				if !spinnerStopped {
					s.Stop()
					spinnerStopped = true
				}

				// Display final 100% progress permanently by printing directly to output
				bar := progressBar(1.0, ProgressBarWidth)
				label := "Progress"
				if numCalculators > 1 {
					label = "Avg progress"
				}
				// Print the final progress line with a newline so it persists
				fmt.Fprintf(out, "%s: %6.2f%% [%s] ETA: %s\n", label, 100.0, bar, "< 1s")
				return
			}
			state.UpdateWithETA(update.CalculatorIndex, update.Value)
		case <-ticker.C:
			avgProgress := state.CalculateAverage()
			eta := state.GetETA()
			bar := progressBar(avgProgress, ProgressBarWidth)
			label := "Progress"
			if numCalculators > 1 {
				label = "Avg progress"
			}
			etaStr := FormatETA(eta)
			s.UpdateSuffix(fmt.Sprintf(" %s: %6.2f%% [%s] ETA: %s", label, avgProgress*100, bar, etaStr))
		}
	}
}

// DisplayResult formats and prints the final calculation result.
// It provides different levels of detail based on the verbose and details flags,
// including metadata like binary size, number of digits, and scientific
// notation. For very large numbers, it truncates the output unless verbose is
// true.
//
// Parameters:
//   - result: The calculation result.
//   - n: The index of the Fibonacci number calculated.
//   - duration: The time taken for the calculation.
//   - verbose: If true, prints the full number regardless of size.
//   - details: If true, prints detailed execution metrics.
//   - concise: If true, displays the calculated value section (disabled by default).
//   - out: The io.Writer for the output.
func DisplayResult(result *big.Int, n uint64, duration time.Duration, verbose, details, concise bool, out io.Writer) {
	bitLen := result.BitLen()
	fmt.Fprintf(out, "Result binary size: %s%s%s bits.\n", ColorCyan(), formatNumberString(fmt.Sprintf("%d", bitLen)), ColorReset())

	if details {
		fmt.Fprintf(out, "\n%s--- Detailed result analysis ---%s\n", ColorBold(), ColorReset())
		durationStr := FormatExecutionDuration(duration)
		if duration == 0 {
			durationStr = "< 1µs"
		}
		fmt.Fprintf(out, "Calculation time        : %s%s%s\n", ColorGreen(), durationStr, ColorReset())

		resultStr := result.String()
		numDigits := len(resultStr)
		fmt.Fprintf(out, "Number of digits      : %s%s%s\n", ColorCyan(), formatNumberString(fmt.Sprintf("%d", numDigits)), ColorReset())

		if numDigits > 6 {
			f := new(big.Float).SetInt(result)
			fmt.Fprintf(out, "Scientific notation    : %s%.6e%s\n", ColorCyan(), f, ColorReset())
		}
	}

	// Skip calculated value display unless -c/--calculate flag is set
	if !concise {
		return
	}

	resultStr := result.String()
	numDigits := len(resultStr)

	fmt.Fprintf(out, "\n%s--- Calculated value ---%s\n", ColorBold(), ColorReset())
	if verbose {
		fmt.Fprintf(out, "F(%s%d%s) =\n%s%s%s\n", ColorMagenta(), n, ColorReset(), ColorGreen(), formatNumberString(resultStr), ColorReset())
	} else if numDigits > TruncationLimit {
		fmt.Fprintf(out, "F(%s%d%s) (truncated) = %s%s...%s%s\n",
			ColorMagenta(), n, ColorReset(),
			ColorGreen(), resultStr[:DisplayEdges], resultStr[numDigits-DisplayEdges:], ColorReset())
		fmt.Fprintf(out, "(Tip: use the %s-v%s or %s--verbose%s option to display the full value)\n", ColorYellow(), ColorReset(), ColorYellow(), ColorReset())
	} else {
		fmt.Fprintf(out, "F(%s%d%s) = %s%s%s\n", ColorMagenta(), n, ColorReset(), ColorGreen(), formatNumberString(resultStr), ColorReset())
	}
}

// formatNumberString inserts thousand separators into a numeric string.
// Optimized to reduce memory allocations
//
// Parameters:
//   - s: The numeric string to format.
//
// Returns:
//   - string: The formatted string with comma separators.
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

	// Precise calculation of the required capacity to avoid reallocations
	numSeparators := (n - 1) / 3
	capacity := len(prefix) + n + numSeparators
	var builder strings.Builder
	builder.Grow(capacity)
	builder.WriteString(prefix)

	firstGroupLen := n % 3
	if firstGroupLen == 0 {
		firstGroupLen = 3
	}
	builder.WriteString(s[:firstGroupLen])

	// Optimized loop with fewer function calls
	for i := firstGroupLen; i < n; i += 3 {
		builder.WriteByte(',')
		builder.WriteString(s[i : i+3])
	}
	return builder.String()
}
