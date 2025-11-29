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

// Build-time variables set via -ldflags.
// These are populated during CI/CD builds to provide version information.
//
// Example build command:
//
//	go build -ldflags="-X main.Version=v1.2.3 -X main.Commit=abc123 -X main.BuildDate=2025-01-01T00:00:00Z"
var (
	// Version is the semantic version of the application (e.g., "v1.0.0").
	Version = "dev"
	// Commit is the short Git commit hash (e.g., "abc123").
	Commit = "unknown"
	// BuildDate is the ISO 8601 timestamp of the build (e.g., "2025-01-01T00:00:00Z").
	BuildDate = "unknown"
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
//  1. Parses the configuration from command-line arguments.
//  2. Loads internationalization resources (optional).
//  3. Configures global settings like the Strassen threshold.
//  4. Starts the application in either server mode or CLI mode based on the
//     configuration.
func main() {
	// Check for version flag in any position
	if hasVersionFlag(os.Args[1:]) {
		printVersion(os.Stdout)
		os.Exit(apperrors.ExitSuccess)
	}

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

	// Initialize CLI theme (respects --no-color flag and NO_COLOR env var)
	cli.InitTheme(cfg.NoColor)

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

// hasVersionFlag checks if any argument is a version flag.
// This allows --version to work in any position (e.g., "fibcalc --server --version").
func hasVersionFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--version" || arg == "-version" || arg == "-V" {
			return true
		}
	}
	return false
}

// printVersion outputs version information to the given writer.
func printVersion(out io.Writer) {
	fmt.Fprintf(out, "fibcalc %s\n", Version)
	fmt.Fprintf(out, "  Commit:     %s\n", Commit)
	fmt.Fprintf(out, "  Built:      %s\n", BuildDate)
	fmt.Fprintf(out, "  Go version: %s\n", runtime.Version())
	fmt.Fprintf(out, "  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
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

	cfg = runAutoCalibrationIfEnabled(ctx, cfg, out)
	calculatorsToRun := getCalculatorsToRun(cfg)

	if !cfg.JSONOutput {
		printExecutionConfig(cfg, out)
		printExecutionMode(calculatorsToRun, out)
	}

	results := orchestration.ExecuteCalculations(ctx, calculatorsToRun, cfg, out)

	if cfg.JSONOutput {
		return printJSONResults(results, out)
	}

	return orchestration.AnalyzeComparisonResults(results, cfg, out)
}

// runAutoCalibrationIfEnabled runs auto-calibration if it's enabled in the configuration.
// Returns the potentially updated configuration.
//
// Parameters:
//   - ctx: The context for managing cancellation.
//   - cfg: The current application configuration.
//   - out: The writer for standard output.
//
// Returns:
//   - config.AppConfig: The configuration, possibly updated with calibrated values.
func runAutoCalibrationIfEnabled(ctx context.Context, cfg config.AppConfig, out io.Writer) config.AppConfig {
	if cfg.AutoCalibrate {
		if updated, ok := calibration.AutoCalibrate(ctx, cfg, out, calculatorRegistry); ok {
			return updated
		}
	}
	return cfg
}

// printExecutionConfig displays the current execution configuration to the user.
// It shows the target Fibonacci number, timeout, environment details, and
// optimization thresholds.
//
// Parameters:
//   - cfg: The application configuration.
//   - out: The writer for standard output.
func printExecutionConfig(cfg config.AppConfig, out io.Writer) {
	writeOut(out, "%s\n", i18n.Messages["ExecConfigTitle"])
	writeOut(out, "Calculating %sF(%d)%s with a timeout of %s%s%s.\n",
		cli.ColorMagenta(), cfg.N, cli.ColorReset(), cli.ColorYellow(), cfg.Timeout, cli.ColorReset())
	writeOut(out, "Environment: %s%d%s logical processors, Go %s%s%s.\n",
		cli.ColorCyan(), runtime.NumCPU(), cli.ColorReset(), cli.ColorCyan(), runtime.Version(), cli.ColorReset())
	writeOut(out, "Optimization thresholds: Parallelism=%s%d%s bits, FFT=%s%d%s bits.\n",
		cli.ColorCyan(), cfg.Threshold, cli.ColorReset(), cli.ColorCyan(), cfg.FFTThreshold, cli.ColorReset())
}

// printExecutionMode displays the execution mode (single algorithm vs comparison).
//
// Parameters:
//   - calculators: The slice of calculators that will be executed.
//   - out: The writer for standard output.
func printExecutionMode(calculators []fibonacci.Calculator, out io.Writer) {
	var modeDesc string
	if len(calculators) > 1 {
		modeDesc = "Parallel comparison of all algorithms"
	} else {
		modeDesc = fmt.Sprintf("Single calculation with the %s%s%s algorithm",
			cli.ColorGreen(), calculators[0].Name(), cli.ColorReset())
	}
	writeOut(out, "Execution mode: %s.\n", modeDesc)
	writeOut(out, "\n%s\n", i18n.Messages["ExecStartTitle"])
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
