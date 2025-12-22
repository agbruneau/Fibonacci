package cli

import (
	"fmt"
	"io"
	"runtime"

	"github.com/agbru/fibcalc/internal/config"
	"github.com/agbru/fibcalc/internal/fibonacci"
)

// GetCalculatorsToRun determines which calculators should be executed based on
// the configuration. Returns calculators in alphabetically sorted order for
// consistent, reproducible behavior.
//
// Parameters:
//   - cfg: The application configuration containing the algorithm selection.
//   - factory: The calculator factory to retrieve implementations from.
//
// Returns:
//   - []fibonacci.Calculator: A slice of calculators to execute.
func GetCalculatorsToRun(cfg config.AppConfig, factory fibonacci.CalculatorFactory) []fibonacci.Calculator {
	if cfg.Algo == "all" {
		keys := factory.List() // List() returns sorted keys
		calculators := make([]fibonacci.Calculator, 0, len(keys))
		for _, k := range keys {
			if calc, err := factory.Get(k); err == nil {
				calculators = append(calculators, calc)
			}
		}
		return calculators
	}
	if calc, err := factory.Get(cfg.Algo); err == nil {
		return []fibonacci.Calculator{calc}
	}
	return nil
}

// PrintExecutionConfig displays the current execution configuration to the user.
// It shows the target Fibonacci number, timeout, environment details, and
// optimization thresholds.
//
// Parameters:
//   - cfg: The application configuration.
//   - out: The writer for standard output.
func PrintExecutionConfig(cfg config.AppConfig, out io.Writer) {
	writeOut(out, "--- Execution Configuration ---\n")
	writeOut(out, "Calculating %sF(%d)%s with a timeout of %s%s%s.\n",
		ColorMagenta(), cfg.N, ColorReset(), ColorYellow(), cfg.Timeout, ColorReset())
	writeOut(out, "Environment: %s%d%s logical processors, Go %s%s%s.\n",
		ColorCyan(), runtime.NumCPU(), ColorReset(), ColorCyan(), runtime.Version(), ColorReset())
	writeOut(out, "Optimization thresholds: Parallelism=%s%d%s bits, FFT=%s%d%s bits.\n",
		ColorCyan(), cfg.Threshold, ColorReset(), ColorCyan(), cfg.FFTThreshold, ColorReset())
}

// PrintExecutionMode displays the execution mode (single algorithm vs comparison).
//
// Parameters:
//   - calculators: The slice of calculators that will be executed.
//   - out: The writer for standard output.
func PrintExecutionMode(calculators []fibonacci.Calculator, out io.Writer) {
	var modeDesc string
	if len(calculators) > 1 {
		modeDesc = "Parallel comparison of all algorithms"
	} else {
		modeDesc = fmt.Sprintf("Single calculation with the %s%s%s algorithm",
			ColorGreen(), calculators[0].Name(), ColorReset())
	}
	writeOut(out, "Execution mode: %s.\n", modeDesc)
	writeOut(out, "\n--- Starting Execution ---\n")
}

// writeOut writes a formatted string to the output writer.
// It uses zerolog for structured output if configured, or falls back to
// direct writing if the output is not a logger.
//
// Parameters:
//   - out: The destination writer.
//   - format: The format string (see fmt.Printf).
//   - a: Arguments for the format string.
func writeOut(out io.Writer, format string, a ...any) {
	fmt.Fprintf(out, format, a...)
}
