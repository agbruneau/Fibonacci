// The main package is the entry point of the fibcalc application. It handles
// command-line argument parsing, configuration, calculation orchestration,
// and result display.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"syscall"

	"example.com/fibcalc/internal/calibration"
	"example.com/fibcalc/internal/cli"
	"example.com/fibcalc/internal/config"
	apperrors "example.com/fibcalc/internal/errors"
	"example.com/fibcalc/internal/fibonacci"
	"example.com/fibcalc/internal/i18n"
	"example.com/fibcalc/internal/orchestration"
	"example.com/fibcalc/internal/server"
)

var calculatorRegistry = map[string]fibonacci.Calculator{
	"fast":   fibonacci.NewCalculator(&fibonacci.OptimizedFastDoubling{}),
	"matrix": fibonacci.NewCalculator(&fibonacci.MatrixExponentiation{}),
	"fft":    fibonacci.NewCalculator(&fibonacci.FFTBasedCalculator{}),
}

// init initializes the application and verifies the integrity of the calculator
// registry. It ensures that all registered calculators are properly instantiated
// before the application starts.
func init() {
	for name, calc := range calculatorRegistry {
		if calc == nil {
			panic(fmt.Sprintf("Critical initialization error: the calculator registered under the name '%s' is nil.", name))
		}
	}
}

// getSortedCalculatorKeys returns a sorted list of the names of available
// algorithms. This ensures a consistent order when displaying options or running
// comparisons.
//
// Returns:
//   - []string: A slice containing the sorted names of the available algorithms.
func getSortedCalculatorKeys() []string {
	keys := make([]string, 0, len(calculatorRegistry))
	for k := range calculatorRegistry {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// main is the entry point of the application.
// It performs the following steps:
// 1. Parses the configuration from command-line arguments.
// 2. Loads internationalization resources (optional).
// 3. Configures global settings like the Strassen threshold.
// 4. Starts the application in either server mode or CLI mode based on the
//    configuration.
func main() {
	availableAlgos := getSortedCalculatorKeys()
	cfg, err := config.ParseConfig(os.Args[0], os.Args[1:], os.Stderr, availableAlgos)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(apperrors.ExitSuccess)
		}
		os.Exit(apperrors.ExitErrorConfig)
	}
	// Optional i18n loading
	if cfg.I18nDir != "" {
		if err := i18n.LoadFromDir(cfg.I18nDir, cfg.Lang); err != nil {
			fmt.Fprintln(os.Stderr, "[i18n] failed to load translations:", err)
		}
	}
	// Setting the Strassen threshold for the matrix algorithm
	fibonacci.DefaultStrassenThresholdBits = cfg.StrassenThreshold

	if cfg.ServerMode {
		srv := server.NewServer(calculatorRegistry, cfg)
		if err := srv.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(apperrors.ExitErrorGeneric)
		}
		return
	}

	exitCode := run(context.Background(), cfg, os.Stdout)
	os.Exit(exitCode)
}

// run orchestrates the execution of the CLI application.
// It manages the lifecycle of the application, including handling timeouts and
// termination signals. It delegates the actual work to the calibration,
// orchestration, or calculation modules based on the user's configuration.
//
// Parameters:
//   - ctx: The context for managing cancellation.
//   - cfg: The application configuration.
//   - out: The writer for standard output.
//
// Returns:
//   - int: An exit code (0 for success, non-zero for errors).
func run(ctx context.Context, cfg config.AppConfig, out io.Writer) int {
	if cfg.Calibrate {
		return calibration.RunCalibration(ctx, out, calculatorRegistry)
	}
	ctx, cancelTimeout := context.WithTimeout(ctx, cfg.Timeout)
	defer cancelTimeout()
	ctx, stopSignals := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	// Quick auto-calibration at startup (if enabled)
	if cfg.AutoCalibrate {
		if updated, ok := calibration.AutoCalibrate(ctx, cfg, out, calculatorRegistry); ok {
			cfg = updated
		}
	}

	if !cfg.JSONOutput {
		writeOut(out, "%s\n", i18n.Messages["ExecConfigTitle"])
		writeOut(out, "Calculating %sF(%d)%s with a timeout of %s%s%s.\n",
			cli.ColorMagenta, cfg.N, cli.ColorReset, cli.ColorYellow, cfg.Timeout, cli.ColorReset)
		writeOut(out, "Environment: %s%d%s logical processors, Go %s%s%s.\n",
			cli.ColorCyan, runtime.NumCPU(), cli.ColorReset, cli.ColorCyan, runtime.Version(), cli.ColorReset)
		writeOut(out, "Optimization thresholds: Parallelism=%s%d%s bits, FFT=%s%d%s bits.\n",
			cli.ColorCyan, cfg.Threshold, cli.ColorReset, cli.ColorCyan, cfg.FFTThreshold, cli.ColorReset)
	}

	calculatorsToRun := getCalculatorsToRun(cfg)
	if !cfg.JSONOutput {
		var modeDesc string
		if len(calculatorsToRun) > 1 {
			modeDesc = "Parallel comparison of all algorithms"
		} else {
			modeDesc = fmt.Sprintf("Single calculation with the %s%s%s algorithm",
				cli.ColorGreen, calculatorsToRun[0].Name(), cli.ColorReset)
		}
		writeOut(out, "Execution mode: %s.\n", modeDesc)
		writeOut(out, "\n%s\n", i18n.Messages["ExecStartTitle"])
	}

	results := orchestration.ExecuteCalculations(ctx, calculatorsToRun, cfg, out)

	if cfg.JSONOutput {
		return printJSONResults(results, out)
	}

	return orchestration.AnalyzeComparisonResults(results, cfg, out)
}

// getCalculatorsToRun determines which calculators should be executed based on
// the configuration.
//
// Parameters:
//   - cfg: The application configuration specifying the selected algorithm.
//
// Returns:
//   - []fibonacci.Calculator: A slice of calculators to be executed.
func getCalculatorsToRun(cfg config.AppConfig) []fibonacci.Calculator {
	if cfg.Algo == "all" {
		keys := getSortedCalculatorKeys()
		calculators := make([]fibonacci.Calculator, len(keys))
		for i, k := range keys {
			calculators[i] = calculatorRegistry[k]
		}
		return calculators
	}
	return []fibonacci.Calculator{calculatorRegistry[cfg.Algo]}
}

// writeOut writes a formatted string to the output writer, handling any write
// errors by printing to standard error. This ensures that output issues do not
// crash the application but are reported.
//
// Parameters:
//   - out: The destination writer.
//   - format: The format string (see fmt.Printf).
//   - a: Arguments for the format string.
func writeOut(out io.Writer, format string, a ...interface{}) {
	if _, err := fmt.Fprintf(out, format, a...); err != nil {
		fmt.Fprintln(os.Stderr, "[Output Error]:", err)
	}
}

// printJSONResults formats the calculation results as a JSON array and writes
// them to the output. This is useful for programmatic consumption of the results.
//
// Parameters:
//   - results: The calculation results to verify and print.
//   - out: The destination writer.
//
// Returns:
//   - int: An exit code indicating success or failure.
func printJSONResults(results []orchestration.CalculationResult, out io.Writer) int {
	type jsonResult struct {
		Algorithm string `json:"algorithm"`
		Duration  string `json:"duration"`
		Result    string `json:"result,omitempty"`
		Error     string `json:"error,omitempty"`
	}
	output := make([]jsonResult, len(results))
	for i, res := range results {
		jr := jsonResult{
			Algorithm: res.Name,
			Duration:  res.Duration.String(),
		}
		if res.Err != nil {
			jr.Error = res.Err.Error()
		} else {
			jr.Result = res.Result.String()
		}
		output[i] = jr
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		return apperrors.ExitErrorGeneric
	}
	return apperrors.ExitSuccess
}
