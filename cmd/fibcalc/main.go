// The main package is the entry point of the fibcalc application. It handles
// command-line argument parsing, configuration, calculation orchestration,
// and result display.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"text/tabwriter"
	"time"

	"golang.org/x/sync/errgroup"

	"example.com/fibcalc/internal/cli"
	"example.com/fibcalc/internal/config"
	"example.com/fibcalc/internal/fibonacci"
)

// Application exit codes define the standard exit statuses for the application.
const (
	// ExitSuccess indicates a successful execution without errors.
	ExitSuccess = 0
	// ExitErrorGeneric indicates a general, unspecified error.
	ExitErrorGeneric = 1
	// ExitErrorTimeout signals that the calculation exceeded the configured timeout.
	ExitErrorTimeout = 2
	// ExitErrorMismatch indicates an inconsistency detected between the results of different algorithms.
	ExitErrorMismatch = 3
	// ExitErrorConfig denotes an error related to configuration or command-line arguments.
	ExitErrorConfig = 4
	// ExitErrorCanceled is used when the execution is canceled by the user (e.g., via SIGINT).
	ExitErrorCanceled = 130
)

const (
	// ANSI escape codes for text styling.
	ColorReset     = "\033[0m"
	ColorRed       = "\033[31m"
	ColorGreen     = "\033[32m"
	ColorYellow    = "\033[33m"
	ColorBlue      = "\033[34m"
	ColorMagenta   = "\033[35m"
	ColorCyan      = "\033[36m"
	ColorBold      = "\033[1m"
	ColorUnderline = "\033[4m"
)

// ProgressBufferMultiplier defines the buffer size of the progress channel,
// calculated as a multiple of the number of active calculators. A larger
// buffer reduces the risk of blocking progress updates.
const ProgressBufferMultiplier = 10

var calculatorRegistry = map[string]fibonacci.Calculator{
	"fast":   fibonacci.NewCalculator(&fibonacci.OptimizedFastDoubling{}),
	"matrix": fibonacci.NewCalculator(&fibonacci.MatrixExponentiation{}),
	"fft":    fibonacci.NewCalculator(&fibonacci.FFTBasedCalculator{}),
}

func init() {
	for name, calc := range calculatorRegistry {
		if calc == nil {
			panic(fmt.Sprintf("Critical initialization error: the calculator registered under the name '%s' is nil.", name))
		}
	}
}

// getSortedCalculatorKeys returns the sorted keys of the calculator registry.
func getSortedCalculatorKeys() []string {
	keys := make([]string, 0, len(calculatorRegistry))
	for k := range calculatorRegistry {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// main is the entry point of the application. It parses the command-line
// arguments, validates the configuration, and orchestrates the execution of the
// Fibonacci calculation. The application's exit code is determined by the
// outcome of the `run` function.
func main() {
	availableAlgos := getSortedCalculatorKeys()
	cfg, err := config.ParseConfig(os.Args[0], os.Args[1:], os.Stderr, availableAlgos)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(ExitSuccess)
		}
		os.Exit(ExitErrorConfig)
	}
	exitCode := run(context.Background(), cfg, os.Stdout)
	os.Exit(exitCode)
}

// CalculationResult encapsulates the outcome of a single Fibonacci calculation.
// It holds the result, execution duration, and any error that occurred, facilitating
// the aggregation and comparison of results from multiple algorithms.
//
// Fields:
//   - Name: The identifier of the algorithm used for the calculation.
//   - Result: The calculated Fibonacci number. It is nil if an error occurred.
//   - Duration: The total time taken for the calculation.
//   - Err: Any error encountered during the calculation.
type CalculationResult struct {
	// Name is the identifier of the algorithm used for the calculation.
	Name string
	// Result is the calculated Fibonacci number. It is nil if an error occurred.
	Result *big.Int
	// Duration is the total time taken for the calculation.
	Duration time.Duration
	// Err holds any error encountered during the calculation.
	Err error
}

// runCalibration runs a series of benchmarks to determine the optimal parallelism
// threshold for the current machine. It tests a predefined set of threshold
// values and measures the execution time for each, ultimately recommending the
// value that yields the best performance. This function is invoked when the
// `--calibrate` flag is provided.
//
// The calibration process involves:
// - Iterating through a list of threshold values.
// - For each threshold, calculating a large Fibonacci number.
// - Recording the duration of each calculation.
// - Displaying a summary table comparing the performance of each threshold.
// - Recommending the threshold that resulted in the shortest execution time.
//
// Parameters:
//   - ctx: The context for managing cancellation.
//   - cfg: The application's configuration, used for settings like timeout.
//   - out: The output writer for displaying progress and results.
//
// Returns an exit code indicating the outcome of the calibration process.
func runCalibration(ctx context.Context, cfg config.AppConfig, out io.Writer) int {
	fmt.Fprintf(out, "%s--- Calibration Mode: Finding the Optimal Parallelism Threshold ---%s\n", ColorBold, ColorReset)
	const calibrationN = 10_000_000
	calculator := calculatorRegistry["fast"]
	if calculator == nil {
		fmt.Fprintf(out, "%sCritical error: The 'fast' algorithm is required for calibration but was not found.%s\n", ColorRed, ColorReset)
		return ExitErrorGeneric
	}

	thresholdsToTest := []int{0, 256, 512, 1024, 2048, 4096, 8192, 16384}
	type calibrationResult struct {
		Threshold int
		Duration  time.Duration
		Err       error
	}
	results := make([]calibrationResult, 0, len(thresholdsToTest))
	bestDuration := time.Duration(1<<63 - 1)
	bestThreshold := 0

	var wg sync.WaitGroup
	progressChan := make(chan fibonacci.ProgressUpdate, 1*ProgressBufferMultiplier)
	wg.Add(1)
	go cli.DisplayProgress(&wg, progressChan, 1, out)

	for _, threshold := range thresholdsToTest {
		if ctx.Err() != nil {
			fmt.Fprintf(out, "\n%sCalibration interrupted.%s\n", ColorYellow, ColorReset)
			return ExitErrorCanceled
		}

		startTime := time.Now()
		_, err := calculator.Calculate(ctx, progressChan, 0, calibrationN, threshold, 0)
		duration := time.Since(startTime)

		if err != nil {
			fmt.Fprintf(out, "%s❌ Failure (%v)%s\n", ColorRed, err, ColorReset)
			results = append(results, calibrationResult{threshold, 0, err})
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				close(progressChan)
				wg.Wait()
				return handleCalculationError(err, duration, cfg.Timeout, out)
			}
			continue
		}

		results = append(results, calibrationResult{threshold, duration, nil})
		if duration < bestDuration {
			bestDuration, bestThreshold = duration, threshold
		}
	}
	close(progressChan)
	wg.Wait()

	fmt.Fprintf(out, "\n%s--- Calibration Summary ---%s\n", ColorBold, ColorReset)
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	fmt.Fprintf(tw, "  %sThreshold%s    │ %sExecution Time%s\n", ColorUnderline, ColorReset, ColorUnderline, ColorReset)
	fmt.Fprintf(tw, "  %s┼%s\n", strings.Repeat("─", 14), strings.Repeat("─", 25))
	for _, res := range results {
		thresholdLabel := fmt.Sprintf("%d bits", res.Threshold)
		if res.Threshold == 0 {
			thresholdLabel = "Sequential"
		}
		durationStr := fmt.Sprintf("%sN/A%s", ColorRed, ColorReset)
		if res.Err == nil {
			durationStr = res.Duration.String()
		}
		highlight := ""
		if res.Threshold == bestThreshold && res.Err == nil {
			highlight = fmt.Sprintf(" %s(Optimal)%s", ColorGreen, ColorReset)
		}
		fmt.Fprintf(tw, "  %s%-12s%s │ %s%s%s%s\n", ColorCyan, thresholdLabel, ColorReset, ColorYellow, durationStr, ColorReset, highlight)
	}
	tw.Flush()
	fmt.Fprintf(out, "\n%s✅ Recommendation for this machine: %s--threshold %d%s\n",
		ColorGreen, ColorYellow, bestThreshold, ColorReset)
	return ExitSuccess
}

// run is the main function that orchestrates the application's execution flow.
// It is responsible for setting up the execution context, including timeouts and
// signal handling, and then initiating the Fibonacci calculations.
//
// The process includes:
// - Configuring a context for timeout and graceful shutdown.
// - Displaying the execution configuration to the user.
// - Selecting the appropriate calculator(s) based on the configuration.
// - Executing the calculation(s).
// - Analyzing and displaying the results.
//
// Parameters:
//   - ctx: The parent context for the execution.
//   - cfg: The application's configuration.
//   - out: The output writer for displaying information and results.
//
// Returns an exit code that reflects the outcome of the execution.
func run(ctx context.Context, cfg config.AppConfig, out io.Writer) int {
	if cfg.Calibrate {
		return runCalibration(ctx, cfg, out)
	}
	ctx, cancelTimeout := context.WithTimeout(ctx, cfg.Timeout)
	defer cancelTimeout()
	ctx, stopSignals := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	fmt.Fprintf(out, "%s--- Execution Configuration ---%s\n", ColorBold, ColorReset)
	fmt.Fprintf(out, "Calculating %sF(%d)%s with a timeout of %s%s%s.\n",
		ColorMagenta, cfg.N, ColorReset, ColorYellow, cfg.Timeout, ColorReset)
	fmt.Fprintf(out, "Environment: %s%d%s logical CPUs, Go %s%s%s.\n",
		ColorCyan, runtime.NumCPU(), ColorReset, ColorCyan, runtime.Version(), ColorReset)
	fmt.Fprintf(out, "Optimization thresholds: Parallelism=%s%d%s bits, FFT=%s%d%s bits.\n",
		ColorCyan, cfg.Threshold, ColorReset, ColorCyan, cfg.FFTThreshold, ColorReset)

	calculatorsToRun := getCalculatorsToRun(cfg)
	var modeDesc string
	if len(calculatorsToRun) > 1 {
		modeDesc = "Parallel comparison of all algorithms"
	} else {
		modeDesc = fmt.Sprintf("Simple calculation with the %s%s%s algorithm",
			ColorGreen, calculatorsToRun[0].Name(), ColorReset)
	}
	fmt.Fprintf(out, "Execution mode: %s.\n", modeDesc)
	fmt.Fprintf(out, "\n%s--- Start of Execution ---%s\n", ColorBold, ColorReset)

	results := executeCalculations(ctx, calculatorsToRun, cfg, out)
	return analyzeComparisonResults(results, cfg, out)
}

// getCalculatorsToRun selects the calculators to be run based on the application's
// configuration. If the "all" algorithm is specified, it returns a list of all
// registered calculators. Otherwise, it returns the specific calculator that was
// requested.
//
// Parameters:
//   - cfg: The application's configuration, which specifies the desired algorithm.
//
// Returns a slice of `fibonacci.Calculator` instances to be executed.
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


// executeCalculations orchestrates the concurrent execution of one or more
// Fibonacci calculations. It uses an `errgroup` to manage the lifecycle of the
// calculation goroutines and ensures that they can be gracefully canceled.
//
// This function is responsible for:
// - Setting up a progress channel for real-time updates.
// - Launching a separate goroutine for the progress display.
// - Starting a goroutine for each calculation.
// - Aggregating the results of each calculation.
// - Waiting for all calculations and the progress display to complete.
//
// Parameters:
//   - ctx: The context for managing cancellation.
//   - calculators: A slice of `fibonacci.Calculator` instances to be executed.
//   - cfg: The application's configuration.
//   - out: The output writer for the progress display.
//
// Returns a slice of `CalculationResult`, with each element corresponding to the
// outcome of a single calculation.
func executeCalculations(ctx context.Context, calculators []fibonacci.Calculator, cfg config.AppConfig, out io.Writer) []CalculationResult {
	g, ctx := errgroup.WithContext(ctx)
	results := make([]CalculationResult, len(calculators))
	progressChan := make(chan fibonacci.ProgressUpdate, len(calculators)*ProgressBufferMultiplier)

	var displayWg sync.WaitGroup
	displayWg.Add(1)
	go cli.DisplayProgress(&displayWg, progressChan, len(calculators), out)

	for i, calc := range calculators {
		idx, calculator := i, calc
		g.Go(func() error {
			startTime := time.Now()
			res, err := calculator.Calculate(ctx, progressChan, idx, cfg.N, cfg.Threshold, cfg.FFTThreshold)
			results[idx] = CalculationResult{
				Name: calculator.Name(), Result: res, Duration: time.Since(startTime), Err: err,
			}
			// We return nil because we want all calculators to complete, even if one fails.
			// The error is captured in the results slice and handled later.
			return nil
		})
	}

	g.Wait()
	close(progressChan)
	displayWg.Wait()

	return results
}

// analyzeComparisonResults processes and displays the final results of the
// calculations. It sorts the results by performance, checks for inconsistencies,
// and presents a summary to the user.
//
// The analysis includes the following steps:
// - Sorting the results by duration, with successful calculations prioritized.
// - Displaying a detailed comparison summary in a tabular format.
// - Checking for mismatches between the results of different algorithms.
// - Reporting the final status (success, failure, or mismatch).
// - Displaying the final calculated value and performance details.
//
// Parameters:
//   - results: The slice of `CalculationResult` from the calculations.
//   - cfg: The application's configuration.
//   - out: The output writer for displaying the analysis.
//
// Returns an exit code that reflects the outcome of the analysis.
func analyzeComparisonResults(results []CalculationResult, cfg config.AppConfig, out io.Writer) int {
	sort.Slice(results, func(i, j int) bool {
		if (results[i].Err == nil) != (results[j].Err == nil) {
			return results[i].Err == nil
		}
		return results[i].Duration < results[j].Duration
	})

	var firstValidResult *big.Int
	var firstValidResultDuration time.Duration
	var firstError error
	successCount := 0

	fmt.Fprintf(out, "\n%s--- Comparison Summary ---%s\n", ColorBold, ColorReset)
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	fmt.Fprintf(tw, "%sAlgorithm%s\t%sDuration%s\t%sStatus%s\n",
		ColorUnderline, ColorReset, ColorUnderline, ColorReset, ColorUnderline, ColorReset)

	for _, res := range results {
		var status string
		if res.Err != nil {
			status = fmt.Sprintf("%s❌ Failure (%v)%s", ColorRed, res.Err, ColorReset)
			if firstError == nil {
				firstError = res.Err
			}
		} else {
			status = fmt.Sprintf("%s✅ Success%s", ColorGreen, ColorReset)
			successCount++
			if firstValidResult == nil {
				firstValidResult = res.Result
				firstValidResultDuration = res.Duration
			}
		}
		fmt.Fprintf(tw, "%s%s%s\t%s%s%s\t%s\n",
			ColorBlue, res.Name, ColorReset,
			ColorYellow, res.Duration.String(), ColorReset,
			status)
	}
	tw.Flush()

	if successCount == 0 {
		fmt.Fprintf(out, "\n%sGlobal Status: Failure. None of the algorithms could complete the calculation.%s\n", ColorRed, ColorReset)
		return handleCalculationError(firstError, 0, cfg.Timeout, out)
	}

	mismatch := false
	for _, res := range results {
		if res.Err == nil && res.Result.Cmp(firstValidResult) != 0 {
			mismatch = true
			break
		}
	}
	if mismatch {
		fmt.Fprintln(out, "\nGlobal Status: CRITICAL FAILURE! An inconsistency was detected between the results of the algorithms.")
		return ExitErrorMismatch
	}

	fmt.Fprintln(out, "\nGlobal Status: Success. All valid results are consistent.")
	cli.DisplayResult(firstValidResult, cfg.N, firstValidResultDuration, cfg.Verbose, cfg.Details, out)
	return ExitSuccess
}

// handleCalculationError interprets a calculation error and translates it into a
// human-readable message and a corresponding exit code. It handles specific
// error types, such as context cancellation and deadline exceeded, to provide
// more informative feedback to the user.
//
// Parameters:
//   - err: The error to be handled. If nil, the function returns ExitSuccess.
//   - duration: The execution duration at the time of the error.
//   - timeout: The configured timeout for the operation.
//   - out: The output writer for displaying the error message.
//
// Returns an exit code that corresponds to the nature of the error.
func handleCalculationError(err error, duration time.Duration, timeout time.Duration, out io.Writer) int {
	if err == nil {
		return ExitSuccess
	}
	msgSuffix := ""
	if duration > 0 {
		msgSuffix = fmt.Sprintf(" after %s%s%s", ColorYellow, duration, ColorReset)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		fmt.Fprintf(out, "%sStatus: Failure (Timeout). The execution time limit of %s%s%s was exceeded%s.%s\n",
			ColorRed, ColorYellow, timeout, ColorReset, msgSuffix, ColorReset)
		return ExitErrorTimeout
	}
	if errors.Is(err, context.Canceled) {
		fmt.Fprintf(out, "%sStatus: Canceled by user%s.%s\n",
			ColorYellow, msgSuffix, ColorReset)
		return ExitErrorCanceled
	}
	fmt.Fprintf(out, "%sStatus: Failure. An unexpected error occurred: %v%s\n", ColorRed, err, ColorReset)
	return ExitErrorGeneric
}
