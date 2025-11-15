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
	"example.com/fibcalc/internal/i18n"
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
// Reduced to 5 to optimize memory usage while avoiding blocking.
const ProgressBufferMultiplier = 5

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
	// Optional i18n loading
	if cfg.I18nDir != "" {
		if err := i18n.LoadFromDir(cfg.I18nDir, cfg.Lang); err != nil {
			// Non-blocking: continue with built-in messages
			fmt.Fprintln(os.Stderr, "[i18n] failed to load translations:", err)
		}
	}
	// Setting the Strassen threshold for the matrix algorithm
	fibonacci.DefaultStrassenThresholdBits = cfg.StrassenThreshold
	exitCode := run(context.Background(), cfg, os.Stdout)
	os.Exit(exitCode)
}

// CalculationResult encapsulates the outcome of a single Fibonacci calculation.
// It holds the result, execution duration, and any error that occurred,
// facilitating the aggregation and comparison of results from multiple
// algorithms.
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

// runCalibration runs benchmarks to determine the optimal parallelism threshold.
// It's invoked with the --calibrate flag and tests a range of threshold values
// to find the one that offers the best performance on the current machine.
//
// The process is as follows:
//  1. Iterates through a predefined set of threshold values.
//  2. For each threshold, it calculates a large Fibonacci number.
//  3. It records the duration of each calculation.
//  4. A summary table is displayed, comparing the performance of each threshold.
//  5. The function recommends the threshold that resulted in the shortest
//     execution time.
//
// The context is used for managing cancellation. The out writer is for
// displaying progress and results.
//
// It returns an exit code indicating the outcome of the calibration process.
func runCalibration(ctx context.Context, out io.Writer) int {
	writeOut(out, "%s\n", i18n.Messages["CalibrationTitle"])
	const calibrationN = 10_000_000
	calculator := calculatorRegistry["fast"]
	if calculator == nil {
		writeOut(out, "%sCritical error: the 'fast' algorithm is required for calibration but was not found.%s\n", ColorRed, ColorReset)
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
			writeOut(out, "\n%sCalibration interrupted.%s\n", ColorYellow, ColorReset)
			return ExitErrorCanceled
		}

		startTime := time.Now()
		_, err := calculator.Calculate(ctx, progressChan, 0, calibrationN, threshold, 0)
		duration := time.Since(startTime)

		if err != nil {
			writeOut(out, "%s❌ Failure (%v)%s\n", ColorRed, err, ColorReset)
			results = append(results, calibrationResult{threshold, 0, err})
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				close(progressChan)
				wg.Wait()
				return handleCalculationError(err, duration, out)
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

	// Note: The discrete ternary search has been removed to improve loading times.
	// The initial tests already provide a good estimate.
	// For a more precise calibration, use --calibrate with more time.

	writeOut(out, "\n%s\n", i18n.Messages["CalibrationSummary"])
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	writeOut(tw, "  %sThreshold%s    │ %sExecution Time%s\n", ColorUnderline, ColorReset, ColorUnderline, ColorReset)
	writeOut(tw, "  %s┼%s\n", strings.Repeat("─", 14), strings.Repeat("─", 25))
	for _, res := range results {
		thresholdLabel := fmt.Sprintf("%d bits", res.Threshold)
		if res.Threshold == 0 {
			thresholdLabel = "Sequential"
		}
		durationStr := fmt.Sprintf("%sN/A%s", ColorRed, ColorReset)
		if res.Err == nil {
			durationStr = cli.FormatExecutionDuration(res.Duration)
			if res.Duration == 0 {
				durationStr = "< 1µs"
			}
		}
		highlight := ""
		if res.Threshold == bestThreshold && res.Err == nil {
			highlight = fmt.Sprintf(" %s(Optimal)%s", ColorGreen, ColorReset)
		}
		writeOut(tw, "  %s%-12s%s │ %s%s%s%s\n", ColorCyan, thresholdLabel, ColorReset, ColorYellow, durationStr, ColorReset, highlight)
	}
	tw.Flush()
	writeOut(out, "\n%s✅ Recommendation for this machine: %s--threshold %d%s\n",
		ColorGreen, ColorYellow, bestThreshold, ColorReset)
	return ExitSuccess
}

// run is the main function that orchestrates the application's execution flow.
// It sets up the execution context, including timeouts and signal handling, and
// then initiates the Fibonacci calculations.
//
// The process includes:
//   - Configuring a context for timeout and graceful shutdown.
//   - Displaying the execution configuration to the user.
//   - Selecting the appropriate calculator(s) based on the configuration.
//   - Executing the calculation(s).
//   - Analyzing and displaying the results.
//
// The parent context for the execution is ctx. The application's configuration
// is cfg, and out is the output writer for displaying information and results.
//
// It returns an exit code that reflects the outcome of the execution.
func run(ctx context.Context, cfg config.AppConfig, out io.Writer) int {
	if cfg.Calibrate {
		return runCalibration(ctx, out)
	}
	ctx, cancelTimeout := context.WithTimeout(ctx, cfg.Timeout)
	defer cancelTimeout()
	ctx, stopSignals := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	// Quick auto-calibration at startup (if enabled)
	if cfg.AutoCalibrate {
		if updated, ok := autoCalibrate(ctx, cfg, out); ok {
			cfg = updated
		}
	}

	writeOut(out, "%s\n", i18n.Messages["ExecConfigTitle"])
	writeOut(out, "Calculating %sF(%d)%s with a timeout of %s%s%s.\n",
		ColorMagenta, cfg.N, ColorReset, ColorYellow, cfg.Timeout, ColorReset)
	writeOut(out, "Environment: %s%d%s logical processors, Go %s%s%s.\n",
		ColorCyan, runtime.NumCPU(), ColorReset, ColorCyan, runtime.Version(), ColorReset)
	writeOut(out, "Optimization thresholds: Parallelism=%s%d%s bits, FFT=%s%d%s bits.\n",
		ColorCyan, cfg.Threshold, ColorReset, ColorCyan, cfg.FFTThreshold, ColorReset)

	calculatorsToRun := getCalculatorsToRun(cfg)
	var modeDesc string
	if len(calculatorsToRun) > 1 {
		modeDesc = "Parallel comparison of all algorithms"
	} else {
		modeDesc = fmt.Sprintf("Single calculation with the %s%s%s algorithm",
			ColorGreen, calculatorsToRun[0].Name(), ColorReset)
	}
	writeOut(out, "Execution mode: %s.\n", modeDesc)
	writeOut(out, "\n%s\n", i18n.Messages["ExecStartTitle"])

	results := executeCalculations(ctx, calculatorsToRun, cfg, out)
	return analyzeComparisonResults(results, cfg, out)
}

// autoCalibrate performs a quick calibration of parallelism and FFT thresholds
// for the current machine. It is short and opportunistic: if the context is
// canceled or if a trial exceeds a small fraction of the timeout, the current
// values are kept.
// Returns (updatedCfg, true) if updated, otherwise (originalCfg, false).
func autoCalibrate(parentCtx context.Context, cfg config.AppConfig, out io.Writer) (config.AppConfig, bool) {
	// Do not run auto-calibration in all-algorithm comparison mode:
	// we target the fast (doubling) implementation for speed and consistency.
	calc := calculatorRegistry["fast"]
	if calc == nil {
		return cfg, false
	}

	// Short window: each trial has at most 1/6 of the global timeout,
	// with a useful lower bound to avoid being too short (e.g., 2s).
	perTrial := cfg.Timeout / 6
	if perTrial < 2*time.Second {
		perTrial = 2 * time.Second
	}

	// Input size for calibration: large enough to trigger the paths of
	// interest without being too long.
	const nForCalibration = 10_000_000

	tryRun := func(threshold, fftThreshold int) (time.Duration, error) {
		ctx, cancel := context.WithTimeout(parentCtx, perTrial)
		defer cancel()
		start := time.Now()
		_, err := calc.Calculate(ctx, nil, 0, nForCalibration, threshold, fftThreshold)
		return time.Since(start), err
	}

	// 1) Calibrate parallelism threshold (FFT disabled for stability)
	// Reduced number of candidates to improve loading time
	parallelCandidates := []int{0, 2048, 4096, 8192, 16384}
	bestPar := cfg.Threshold
	bestParDur := time.Duration(1<<63 - 1)
	for _, cand := range parallelCandidates {
		dur, err := tryRun(cand, 0)
		if err != nil {
			continue
		}
		if dur < bestParDur {
			bestParDur, bestPar = dur, cand
		}
	}

	// 2) Calibrate FFT threshold (using the best parallelism found)
	// Reduced number of candidates to improve loading time
	fftCandidates := []int{0, 16000, 20000, 28000}
	bestFFT := cfg.FFTThreshold
	bestFFTDur := time.Duration(1<<63 - 1)
	for _, cand := range fftCandidates {
		dur, err := tryRun(bestPar, cand)
		if err != nil {
			continue
		}
		if dur < bestFFTDur {
			bestFFTDur, bestFFT = dur, cand
		}
	}

	// 3) Calibrate Strassen threshold (with the matrix algorithm)
	//    We evaluate several candidates and keep the best one.
	// Reduced number of candidates to improve loading time
	matCalc := calculatorRegistry["matrix"]
	bestStrassen := cfg.StrassenThreshold
	bestStrassenDur := time.Duration(1<<63 - 1)
	if matCalc != nil {
		// Disable FFT to isolate the Strassen effect
		strassenCandidates := []int{192, 256, 384, 512}
		for _, cand := range strassenCandidates {
			ctx, cancel := context.WithTimeout(parentCtx, perTrial)
			start := time.Now()
			_, err := matCalc.Calculate(ctx, nil, 0, nForCalibration, bestPar, 0)
			cancel()
			dur := time.Since(start)
			if err != nil {
				continue
			}
			// Simulate the impact of the threshold by playing with a global variable
			fibonacci.DefaultStrassenThresholdBits = cand
			if dur < bestStrassenDur {
				bestStrassenDur = dur
				bestStrassen = cand
			}
		}
	}

	// If no valid measurement was made, do not change anything
	if bestParDur == time.Duration(1<<63-1) && bestFFTDur == time.Duration(1<<63-1) {
		return cfg, false
	}

	// Apply the best values found
	updated := cfg
	if bestParDur != time.Duration(1<<63-1) {
		updated.Threshold = bestPar
	}
	if bestFFTDur != time.Duration(1<<63-1) {
		updated.FFTThreshold = bestFFT
	}
	if bestStrassenDur != time.Duration(1<<63-1) {
		updated.StrassenThreshold = bestStrassen
		fibonacci.DefaultStrassenThresholdBits = bestStrassen
	}

	// Succinct display
	writeOut(out, "%sAuto-calibration%s: parallelism=%s%d%s bits, FFT=%s%d%s bits, Strassen=%s%d%s bits\n",
		ColorGreen, ColorReset,
		ColorYellow, updated.Threshold, ColorReset,
		ColorYellow, updated.FFTThreshold, ColorReset,
		ColorYellow, updated.StrassenThreshold, ColorReset)
	return updated, true
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

// analyzeComparisonResults processes and displays the final results.
// It sorts the results by performance, checks for inconsistencies, and presents
// a summary to the user.
//
// The analysis includes the following steps:
//   - Sorting the results by duration, prioritizing successful calculations.
//   - Displaying a detailed comparison summary in a tabular format.
//   - Checking for mismatches between the results of different algorithms.
//   - Reporting the final status (success, failure, or mismatch).
//   - Displaying the final calculated value and performance details.
//
// The slice of CalculationResult from the calculations is results. The
// application's configuration is cfg, and out is the output writer for
// displaying the analysis.
//
// It returns an exit code that reflects the outcome of the analysis.
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

	writeOut(out, "\n%s\n", i18n.Messages["ComparisonSummary"])
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	writeOut(tw, "%sAlgorithm%s\t%sDuration%s\t%sStatus%s\n",
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
		duration := cli.FormatExecutionDuration(res.Duration)
		if res.Duration == 0 {
			duration = "< 1µs"
		}
		writeOut(tw, "%s%s%s\t%s%s%s\t%s\n",
			ColorBlue, res.Name, ColorReset,
			ColorYellow, duration, ColorReset,
			status)
	}
	tw.Flush()

	if successCount == 0 {
		writeOut(out, "\n%s\n", i18n.Messages["GlobalStatusFailure"])
		return handleCalculationError(firstError, 0, out)
	}

	mismatch := false
	for _, res := range results {
		if res.Err == nil && res.Result.Cmp(firstValidResult) != 0 {
			mismatch = true
			break
		}
	}
	if mismatch {
		writeOut(out, "\n%s", i18n.Messages["StatusCriticalMismatch"])
		return ExitErrorMismatch
	}

	writeOut(out, "\n%s", i18n.Messages["GlobalStatusSuccess"])
	cli.DisplayResult(firstValidResult, cfg.N, firstValidResultDuration, cfg.Verbose, cfg.Details, out)
	return ExitSuccess
}

// handleCalculationError interprets a calculation error and translates it into a
// human-readable message and a corresponding exit code. It handles specific
// error types, such as context cancellation and deadline exceeded, to provide
// more informative feedback to the user.
//
// The error to be handled is err. If it's nil, the function returns
// ExitSuccess. The execution duration at the time of the error is duration, and
// out is the output writer for displaying the error message.
//
// It returns an exit code that corresponds to the nature of the error.
func handleCalculationError(err error, duration time.Duration, out io.Writer) int {
	if err == nil {
		return ExitSuccess
	}
	msgSuffix := ""
	if duration > 0 {
		msgSuffix = fmt.Sprintf(" after %s%s%s", ColorYellow, duration, ColorReset)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		writeOut(out, "%s\n", i18n.Messages["StatusTimeout"])
		return ExitErrorTimeout
	}
	if errors.Is(err, context.Canceled) {
		writeOut(out, "%s%s%s.%s\n", ColorYellow, i18n.Messages["StatusCanceled"], msgSuffix, ColorReset)
		return ExitErrorCanceled
	}
	writeOut(out, "%s\n", i18n.Messages["StatusFailure"])
	return ExitErrorGeneric
}

// writeOut centralizes writing to out and handles (or logs) the error.
func writeOut(out io.Writer, format string, a ...interface{}) {
	if _, err := fmt.Fprintf(out, format, a...); err != nil {
		// I/O error on user output, usually critical!
		// Here we log to stderr via fmt.Fprintln but we could exit immediately.
		fmt.Fprintln(os.Stderr, "[Output Error]:", err)
		// os.Exit(1) // In production, we might consider an exit.
	}
}

// Centralized messages: see internal/i18n/messages.go
