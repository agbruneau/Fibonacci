// Display, progress reporting, and persistence functions for the CLI.
//
// # Naming Conventions
//
// Functions in this file follow consistent naming patterns based on their behavior:
//
//   - Display* functions write formatted output to an [io.Writer].
//     They handle presentation logic and colorization.
//     Examples: [DisplayResult], [DisplayQuietResult], [DisplayProgress].
//
//   - Format* functions return a formatted string without performing I/O.
//     They are pure functions suitable for composition.
//     Example: [FormatQuietResult].
//     Pure formatting helpers (duration, numbers, ETA) live in the format
//     package and should be imported from there directly.
//
//   - Write* functions write data to files on the filesystem.
//     They handle file creation, directory setup, and error handling.
//     Examples: [WriteResultToFile].
//
//   - Print* functions write to stdout as convenience wrappers.
//     Examples: [PrintExecutionConfig], [PrintExecutionMode].

package cli

import (
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/agbru/fibcalc/internal/format"
	"github.com/agbru/fibcalc/internal/metrics"
	"github.com/agbru/fibcalc/internal/orchestration"
	"github.com/agbru/fibcalc/internal/progress"
	"github.com/agbru/fibcalc/internal/ui"
	"github.com/briandowns/spinner"
)

// === Rendu UI ===

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
func DisplayProgress(wg *sync.WaitGroup, progressChan <-chan progress.ProgressUpdate, numCalculators int, out io.Writer) {
	defer wg.Done()

	agg := orchestration.NewProgressAggregator(numCalculators)
	if agg == nil {
		orchestration.DrainChannel(progressChan)
		return
	}

	s := newSpinner(spinner.WithWriter(out))
	s.Start()
	spinnerStopped := false
	defer func() {
		if !spinnerStopped {
			s.Stop()
		}
	}()

	label := "Progress"
	if agg.IsMultiCalculator() {
		label = "Avg progress"
	}

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

				// Display actual final progress (not hardcoded 100%).
				// Progress may be less than 100% if calculation was canceled or timed out.
				finalProgress := agg.CalculateAverage()
				bar := format.ProgressBar(finalProgress, ProgressBarWidth)
				etaStr := "< 1s"
				if finalProgress < 1.0 {
					etaStr = "N/A (interrupted)"
				}
				fmt.Fprintf(out, "%s: %6.2f%% [%s] ETA: %s\n", label, finalProgress*100, bar, etaStr)
				return
			}
			agg.Update(update)
		case <-ticker.C:
			avgProgress := agg.CalculateAverage()
			eta := agg.GetETA()
			bar := format.ProgressBar(avgProgress, ProgressBarWidth)
			etaStr := format.FormatETA(eta)
			s.UpdateSuffix(fmt.Sprintf(" %s: %6.2f%% [%s] ETA: %s", label, avgProgress*100, bar, etaStr))
		}
	}
}

// displayResultHeader prints the binary size of the result.
//
// Parameters:
//   - out: The io.Writer for the output.
//   - bitLen: The number of bits in the result.
func displayResultHeader(out io.Writer, bitLen int) {
	fmt.Fprintf(out, "Result binary size: %s%s%s bits.\n",
		ui.ColorCyan(), format.FormatNumberString(fmt.Sprintf("%d", bitLen)), ui.ColorReset())
}

// displayDetailedAnalysis prints detailed execution metrics including
// calculation time, number of digits, and scientific notation for large numbers.
//
// Parameters:
//   - out: The io.Writer for the output.
//   - result: The calculation result.
//   - duration: The time taken for the calculation.
func displayDetailedAnalysis(out io.Writer, result *big.Int, duration time.Duration) {
	fmt.Fprintf(out, "\n%s--- Detailed result analysis ---%s\n", ui.ColorBold(), ui.ColorReset())

	durationStr := format.FormatExecutionDuration(duration)
	if duration == 0 {
		durationStr = "< 1µs"
	}
	fmt.Fprintf(out, "Calculation time        : %s%s%s\n", ui.ColorGreen(), durationStr, ui.ColorReset())

	resultStr := result.String()
	numDigits := len(resultStr)
	fmt.Fprintf(out, "Number of digits      : %s%s%s\n",
		ui.ColorCyan(), format.FormatNumberString(fmt.Sprintf("%d", numDigits)), ui.ColorReset())

	if numDigits > 6 {
		f := new(big.Float).SetInt(result)
		fmt.Fprintf(out, "Scientific notation    : %s%.6e%s\n", ui.ColorCyan(), f, ui.ColorReset())
	}
}

// displayCalculatedValue prints the Fibonacci value, truncating if necessary.
//
// Parameters:
//   - out: The io.Writer for the output.
//   - result: The calculation result.
//   - n: The index of the Fibonacci number calculated.
//   - verbose: If true, prints the full number regardless of size.
func displayCalculatedValue(out io.Writer, result *big.Int, n uint64, verbose bool) {
	resultStr := result.String()
	numDigits := len(resultStr)

	fmt.Fprintf(out, "\n%s--- Calculated value ---%s\n", ui.ColorBold(), ui.ColorReset())

	if verbose {
		fmt.Fprintf(out, "F(%s%d%s) =\n%s%s%s\n",
			ui.ColorMagenta(), n, ui.ColorReset(),
			ui.ColorGreen(), format.FormatNumberString(resultStr), ui.ColorReset())
		return
	}

	if numDigits > TruncationLimit {
		fmt.Fprintf(out, "F(%s%d%s) (truncated) = %s%s...%s%s\n",
			ui.ColorMagenta(), n, ui.ColorReset(),
			ui.ColorGreen(), resultStr[:DisplayEdges], resultStr[numDigits-DisplayEdges:], ui.ColorReset())
		fmt.Fprintf(out, "(Tip: use the %s-v%s or %s--verbose%s option to display the full value)\n",
			ui.ColorYellow(), ui.ColorReset(), ui.ColorYellow(), ui.ColorReset())
		return
	}

	fmt.Fprintf(out, "F(%s%d%s) = %s%s%s\n",
		ui.ColorMagenta(), n, ui.ColorReset(),
		ui.ColorGreen(), format.FormatNumberString(resultStr), ui.ColorReset())
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
//   - showValue: If true, displays the calculated value section (disabled by default).
//   - out: The io.Writer for the output.
func DisplayResult(result *big.Int, n uint64, duration time.Duration, verbose, details, showValue bool, out io.Writer) {
	displayResultHeader(out, result.BitLen())

	if details {
		displayDetailedAnalysis(out, result, duration)
		if duration > 0 {
			displayIndicators(out, metrics.Compute(result, n, duration))
		}
	}

	if showValue {
		displayCalculatedValue(out, result, n, verbose)
	}
}

// displayIndicators prints post-calculation indicators of interest.
// These are computed after the calculation completes, so they have zero
// impact on the measured execution time.
func displayIndicators(out io.Writer, ind *metrics.Indicators) {
	fmt.Fprintf(out, "\n%s--- Indicators of interest ---%s\n", ui.ColorBold(), ui.ColorReset())

	// Performance
	fmt.Fprintf(out, "Throughput (bits)       : %s%s%s\n",
		ui.ColorGreen(), metrics.FormatBitsPerSecond(ind.BitsPerSecond), ui.ColorReset())
	fmt.Fprintf(out, "Throughput (digits)     : %s%s%s\n",
		ui.ColorGreen(), metrics.FormatDigitsPerSecond(ind.DigitsPerSecond), ui.ColorReset())
	fmt.Fprintf(out, "Doubling steps          : %s%d%s  (%s%.2f steps/s%s)\n",
		ui.ColorCyan(), ind.DoublingSteps, ui.ColorReset(),
		ui.ColorCyan(), ind.StepsPerSecond, ui.ColorReset())

	// Mathematical
	fmt.Fprintf(out, "Golden ratio deviation  : %s%.4f%%%s\n",
		ui.ColorMagenta(), ind.GoldenRatioDeviation, ui.ColorReset())
	fmt.Fprintf(out, "Digital root            : %s%d%s\n",
		ui.ColorMagenta(), ind.DigitalRoot, ui.ColorReset())
	fmt.Fprintf(out, "Last 20 digits          : %s%s%s\n",
		ui.ColorMagenta(), ind.LastDigits, ui.ColorReset())

	parity := "odd"
	if ind.IsEven {
		parity = "even"
	}
	fmt.Fprintf(out, "Parity                  : %s%s%s\n",
		ui.ColorMagenta(), parity, ui.ColorReset())
}

// === Persistance fichier ===

// OutputConfig holds configuration for result output.
type OutputConfig struct {
	// OutputFile is the path to save the result (empty for no file output).
	OutputFile string
	// Quiet mode suppresses verbose output.
	Quiet bool
	// Verbose shows the full result value.
	Verbose bool
	// ShowValue enables the calculated value display when true (disabled by default).
	ShowValue bool
}

// WriteResultToFile writes a calculation result to a file.
//
// Parameters:
//   - result: The calculated Fibonacci number.
//   - n: The index of the Fibonacci number.
//   - duration: The calculation duration.
//   - algo: The algorithm name used.
//   - config: Output configuration.
//
// Returns:
//   - error: An error if the file cannot be written.
func WriteResultToFile(result *big.Int, n uint64, duration time.Duration, algo string, config OutputConfig) error {
	if config.OutputFile == "" {
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(config.OutputFile)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	file, err := os.OpenFile(config.OutputFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Write header
	fmt.Fprintf(file, "# Fibonacci Calculation Result\n")
	fmt.Fprintf(file, "# Generated: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(file, "# Algorithm: %s\n", algo)
	fmt.Fprintf(file, "# Duration: %s\n", duration)
	fmt.Fprintf(file, "# N: %d\n", n)
	fmt.Fprintf(file, "# Bits: %d\n", result.BitLen())
	fmt.Fprintf(file, "# Digits: %d\n", len(result.String()))
	fmt.Fprintf(file, "\n")

	// Write result
	fmt.Fprintf(file, "F(%d) =\n%s\n", n, result.String())

	return nil
}

// === Modes spéciaux ===

// FormatQuietResult formats a result for quiet mode output.
// Returns a single-line result suitable for scripting.
//
// Parameters:
//   - result: The calculated Fibonacci number.
//   - n: The index.
//   - duration: The calculation duration.
//
// Returns:
//   - string: The formatted result string.
func FormatQuietResult(result *big.Int, n uint64, duration time.Duration) string {
	return result.String()
}

// DisplayQuietResult outputs a result in quiet mode (minimal output).
//
// Parameters:
//   - out: The output writer.
//   - result: The calculated Fibonacci number.
//   - n: The index.
//   - duration: The calculation duration.
func DisplayQuietResult(out io.Writer, result *big.Int, n uint64, duration time.Duration) {
	fmt.Fprintln(out, FormatQuietResult(result, n, duration))
}

// DisplayResultWithConfig displays a result with the given output configuration.
// This is a unified function that handles all output modes.
//
// Parameters:
//   - out: The output writer.
//   - result: The calculated Fibonacci number.
//   - n: The index.
//   - duration: The calculation duration.
//   - algo: The algorithm name.
//   - config: Output configuration.
//
// Returns:
//   - error: An error if file output fails.
func DisplayResultWithConfig(out io.Writer, result *big.Int, n uint64, duration time.Duration, algo string, config OutputConfig) error {
	// Handle quiet mode
	if config.Quiet {
		DisplayQuietResult(out, result, n, duration)
	} else {
		// Use standard display
		DisplayResult(result, n, duration, config.Verbose, true, config.ShowValue, out)
	}

	// Save to file if requested
	if config.OutputFile != "" {
		if err := WriteResultToFile(result, n, duration, algo, config); err != nil {
			return err
		}
		if !config.Quiet {
			fmt.Fprintf(out, "\n%s✓ Result saved to: %s%s%s\n",
				ui.ColorGreen(), ui.ColorCyan(), config.OutputFile, ui.ColorReset())
		}
	}

	return nil
}
