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
//   - Pure formatting helpers (duration, numbers, ETA) live in the format
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
	"bufio"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/agbruneau/FibGo/internal/format"
	"github.com/agbruneau/FibGo/internal/metrics"
	"github.com/agbruneau/FibGo/internal/orchestration"
	"github.com/agbruneau/FibGo/internal/progress"
	"github.com/agbruneau/FibGo/internal/ui"
	"github.com/briandowns/spinner"
)

// === UI rendering ===

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
// resultStr is the caller's single decimal rendering of the result (audit
// L-11): big.Int.String() on a multi-million-digit value costs seconds, and
// this function used to produce its own copy alongside displayCalculatedValue's
// and the two writeResult made. Every figure here derives from that string.
//
// Parameters:
//   - out: The io.Writer for the output.
//   - resultStr: result rendered in base 10, converted once by the caller.
//   - duration: The time taken for the calculation.
func displayDetailedAnalysis(out io.Writer, resultStr string, duration time.Duration) {
	fmt.Fprintf(out, "\n%s--- Detailed result analysis ---%s\n", ui.ColorBold(), ui.ColorReset())

	durationStr := format.FormatExecutionDuration(duration)
	if duration == 0 {
		durationStr = "< 1µs"
	}
	fmt.Fprintf(out, "Calculation time        : %s%s%s\n", ui.ColorGreen(), durationStr, ui.ColorReset())

	numDigits := len(resultStr)
	fmt.Fprintf(out, "Number of digits      : %s%s%s\n",
		ui.ColorCyan(), format.FormatNumberString(fmt.Sprintf("%d", numDigits)), ui.ColorReset())

	if numDigits > 6 {
		fmt.Fprintf(out, "Scientific notation    : %s%s%s\n", ui.ColorCyan(), scientificNotation(resultStr), ui.ColorReset())
	}
}

// scientificNotation renders a non-empty decimal digit string (no sign, no
// leading zeros) as d.dddddde+XX, i.e. exactly what fmt's %.6e prints for the
// same integer: seven significant digits, rounded half-to-even on an exact tie.
//
// It works on the digits the caller already has instead of going through
// big.Float: Float.Text materializes the whole integer to convert it, so the
// previous %.6e cost a second full base-10 conversion of the result — the
// same seconds-long pass the L-11 audit had just deduplicated — to print a
// 15-character line.
func scientificNotation(digits string) string {
	const sig = 7
	exp := len(digits) - 1
	mant := []byte(digits)
	if len(mant) < sig {
		mant = append(mant, strings.Repeat("0", sig-len(mant))...)
	}
	rest := mant[sig:]
	mant = mant[:sig]
	if len(rest) > 0 && (rest[0] > '5' ||
		(rest[0] == '5' && (strings.TrimRight(string(rest[1:]), "0") != "" || (mant[sig-1]-'0')%2 == 1))) {
		i := sig - 1
		for ; i >= 0 && mant[i] == '9'; i-- {
			mant[i] = '0'
		}
		if i < 0 {
			mant[0] = '1' // 9999999 rounds to 1000000 one decade up
			exp++
		} else {
			mant[i]++
		}
	}
	return fmt.Sprintf("%c.%se+%02d", mant[0], mant[1:], exp)
}

// displayCalculatedValue prints the Fibonacci value, truncating if necessary.
//
// resultStr is the caller's single decimal rendering (audit L-11).
//
// Parameters:
//   - out: The io.Writer for the output.
//   - resultStr: result rendered in base 10, converted once by the caller.
//   - n: The index of the Fibonacci number calculated.
//   - verbose: If true, prints the full number regardless of size.
func displayCalculatedValue(out io.Writer, resultStr string, n uint64, verbose bool) {
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

	// One base-10 conversion for the whole function (audit L-11). It is the
	// dominant cost here at large n — seconds for a 21-million-digit F(100M) —
	// and `-d -c` used to pay for it twice, once per section.
	var resultStr string
	if details || showValue {
		resultStr = result.String()
	}

	if details {
		displayDetailedAnalysis(out, resultStr, duration)
		if duration > 0 {
			displayIndicators(out, metrics.Compute(result, n, duration))
		}
	}

	if showValue {
		displayCalculatedValue(out, resultStr, n, verbose)
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

// === Persistence fichier ===

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
func WriteResultToFile(result *big.Int, n uint64, duration time.Duration, algo string, config OutputConfig) (err error) {
	if config.OutputFile == "" {
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(config.OutputFile)
	if dir != "" && dir != "." {
		if mkErr := os.MkdirAll(dir, 0o700); mkErr != nil {
			return fmt.Errorf("failed to create directory: %w", mkErr)
		}
	}

	file, err := os.OpenFile(config.OutputFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() {
		// Close flushes OS buffers; on a full disk this is where the
		// failure often surfaces. Losing it would report a saved result
		// that was never durably written.
		if cerr := file.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close output file: %w", cerr)
		}
	}()

	return writeResult(file, result, n, duration, algo)
}

// writeResult writes the formatted result to w. Fprintf errors are sticky in
// the bufio.Writer, so the single Flush check surfaces the first failure.
func writeResult(w io.Writer, result *big.Int, n uint64, duration time.Duration, algo string) error {
	bw := bufio.NewWriter(w)

	// One base-10 conversion, shared by the Digits header and the body
	// (audit L-11): this function used to call String() twice.
	resultStr := result.String()

	// Write header
	fmt.Fprintf(bw, "# Fibonacci Calculation Result\n")
	fmt.Fprintf(bw, "# Generated: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(bw, "# Algorithm: %s\n", algo)
	fmt.Fprintf(bw, "# Duration: %s\n", duration)
	fmt.Fprintf(bw, "# N: %d\n", n)
	fmt.Fprintf(bw, "# Bits: %d\n", result.BitLen())
	fmt.Fprintf(bw, "# Digits: %d\n", len(resultStr))
	fmt.Fprintf(bw, "\n")

	// Write result
	fmt.Fprintf(bw, "F(%d) =\n%s\n", n, resultStr)

	if err := bw.Flush(); err != nil {
		return fmt.Errorf("failed to write result: %w", err)
	}
	return nil
}

// === Special modes ===

// DisplayQuietResult outputs a result in quiet mode: the bare decimal value on
// one line, suitable for scripting.
func DisplayQuietResult(out io.Writer, result *big.Int) {
	fmt.Fprintln(out, result.String())
}
