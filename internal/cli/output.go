// Package cli provides output utilities for exporting calculation results.
package cli

import (
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// OutputConfig holds configuration for result output.
type OutputConfig struct {
	// OutputFile is the path to save the result (empty for no file output).
	OutputFile string
	// HexOutput displays the result in hexadecimal format.
	HexOutput bool
	// Quiet mode suppresses verbose output.
	Quiet bool
	// Verbose shows the full result value.
	Verbose bool
	// Concise enables the calculated value display when true (disabled by default).
	Concise bool
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
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	file, err := os.Create(config.OutputFile)
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
	if config.HexOutput {
		fmt.Fprintf(file, "F(%d) [hex] =\n0x%s\n", n, result.Text(16))
	} else {
		fmt.Fprintf(file, "F(%d) =\n%s\n", n, result.String())
	}

	return nil
}

// FormatQuietResult formats a result for quiet mode output.
// Returns a single-line result suitable for scripting.
//
// Parameters:
//   - result: The calculated Fibonacci number.
//   - n: The index.
//   - duration: The calculation duration.
//   - hexOutput: Whether to format as hexadecimal.
//
// Returns:
//   - string: The formatted result string.
func FormatQuietResult(result *big.Int, n uint64, duration time.Duration, hexOutput bool) string {
	if hexOutput {
		return fmt.Sprintf("0x%s", result.Text(16))
	}
	return result.String()
}

// DisplayQuietResult outputs a result in quiet mode (minimal output).
//
// Parameters:
//   - out: The output writer.
//   - result: The calculated Fibonacci number.
//   - n: The index.
//   - duration: The calculation duration.
//   - hexOutput: Whether to format as hexadecimal.
func DisplayQuietResult(out io.Writer, result *big.Int, n uint64, duration time.Duration, hexOutput bool) {
	fmt.Fprintln(out, FormatQuietResult(result, n, duration, hexOutput))
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
		DisplayQuietResult(out, result, n, duration, config.HexOutput)
	} else {
		// Use standard display
		DisplayResult(result, n, duration, config.Verbose, true, config.Concise, out)

		// Show hex format if requested
		if config.HexOutput && !config.Quiet {
			fmt.Fprintf(out, "\n%sHexadecimal format:%s\n", ColorBold(), ColorReset())
			hexStr := result.Text(16)
			if len(hexStr) > 100 && !config.Verbose {
				fmt.Fprintf(out, "F(%d) [hex] = %s0x%s...%s%s\n",
					n, ColorGreen(), hexStr[:40], hexStr[len(hexStr)-40:], ColorReset())
			} else {
				fmt.Fprintf(out, "F(%d) [hex] = %s0x%s%s\n",
					n, ColorGreen(), hexStr, ColorReset())
			}
		}
	}

	// Save to file if requested
	if config.OutputFile != "" {
		if err := WriteResultToFile(result, n, duration, algo, config); err != nil {
			return err
		}
		if !config.Quiet {
			fmt.Fprintf(out, "\n%s✓ Result saved to: %s%s%s\n",
				ColorGreen(), ColorCyan(), config.OutputFile, ColorReset())
		}
	}

	return nil
}
