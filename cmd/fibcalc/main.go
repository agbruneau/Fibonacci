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
	"example.com/fibcalc/internal/fibonacci"
)

// Application exit codes.
const (
	ExitSuccess       = 0
	ExitErrorGeneric  = 1
	ExitErrorTimeout  = 2
	ExitErrorMismatch = 3
	ExitErrorConfig   = 4
	ExitErrorCanceled = 130
)

const ProgressBufferMultiplier = 10

// AppConfig aggregates the application's configuration parameters.
type AppConfig struct {
	N            uint64
	Verbose      bool
	Details      bool
	Timeout      time.Duration
	Algo         string
	Threshold    int
	FFTThreshold int
	Calibrate    bool
}

// Validate checks the semantic consistency of the configuration.
func (c AppConfig) Validate(availableAlgos []string) error {
	if c.Timeout <= 0 {
		return errors.New("timeout value must be strictly positive")
	}
	if c.Threshold < 0 {
		return fmt.Errorf("parallelism threshold cannot be negative: %d", c.Threshold)
	}
	if c.FFTThreshold < 0 {
		return fmt.Errorf("FFT threshold cannot be negative: %d", c.FFTThreshold)
	}
	if c.Algo != "all" {
		if _, ok := calculatorRegistry[c.Algo]; !ok {
			return fmt.Errorf("unrecognized algorithm: '%s'. Valid algorithms: 'all' or one of [%s]", c.Algo, strings.Join(availableAlgos, ", "))
		}
	}
	return nil
}

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

func main() {
	config, err := parseConfig(os.Args[0], os.Args[1:], os.Stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(ExitSuccess)
		}
		os.Exit(ExitErrorConfig)
	}
	exitCode := run(context.Background(), config, os.Stdout)
	os.Exit(exitCode)
}

// parseConfig parses command-line arguments.
func parseConfig(programName string, args []string, errorWriter io.Writer) (AppConfig, error) {
	fs := flag.NewFlagSet(programName, flag.ContinueOnError)
	fs.SetOutput(errorWriter)
	availableAlgos := getSortedCalculatorKeys()
	algoHelp := fmt.Sprintf("Algorithm to use: 'all' (default) or one of [%s].", strings.Join(availableAlgos, ", "))

	config := AppConfig{}
	fs.Uint64Var(&config.N, "n", 250000000, "Index 'n' of the Fibonacci number to calculate.")
	fs.BoolVar(&config.Verbose, "v", false, "Display the full value of the result (can be very long).")
	fs.BoolVar(&config.Details, "d", false, "Display performance details and result metadata.")
	fs.BoolVar(&config.Details, "details", false, "Alias for -d.")
	fs.DurationVar(&config.Timeout, "timeout", 5*time.Minute, "Maximum execution time for the calculation.")
	fs.StringVar(&config.Algo, "algo", "all", algoHelp)
	fs.IntVar(&config.Threshold, "threshold", fibonacci.DefaultParallelThreshold, "Threshold (in bits) to enable parallelization of multiplications.")
	fs.IntVar(&config.FFTThreshold, "fft-threshold", 20000, "Threshold (in bits) to use FFT multiplication (0 to disable).")
	fs.BoolVar(&config.Calibrate, "calibrate", false, "Run calibration mode to determine the optimal parallelism threshold.")

	if err := fs.Parse(args); err != nil {
		return AppConfig{}, err
	}
	config.Algo = strings.ToLower(config.Algo)
	if err := config.Validate(availableAlgos); err != nil {
		fmt.Fprintln(errorWriter, "Configuration error:", err)
		fs.Usage()
		return AppConfig{}, errors.New("invalid configuration")
	}
	return config, nil
}

// CalculationResult encapsulates the result of a calculation.
type CalculationResult struct {
	Name     string
	Result   *big.Int
	Duration time.Duration
	Err      error
}

// runCalibration runs benchmarks to find the optimal parallelism threshold.
func runCalibration(ctx context.Context, config AppConfig, out io.Writer) int {
	fmt.Fprintln(out, "--- Calibration Mode: Finding the Optimal Parallelism Threshold ---")
	const calibrationN = 10_000_000
	calculator := calculatorRegistry["fast"]
	if calculator == nil {
		fmt.Fprintln(out, "Critical error: The 'fast' algorithm is required for calibration but was not found.")
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

	for _, threshold := range thresholdsToTest {
		if ctx.Err() != nil {
			fmt.Fprintln(out, "\nCalibration interrupted.")
			return ExitErrorCanceled
		}
		thresholdLabel := fmt.Sprintf("%d bits", threshold)
		if threshold == 0 {
			thresholdLabel = "Sequential"
		}
		fmt.Fprintf(out, "Testing threshold: %-12s...", thresholdLabel)
		startTime := time.Now()
		_, err := calculator.Calculate(ctx, nil, 0, calibrationN, threshold, 0)
		duration := time.Since(startTime)

		if err != nil {
			fmt.Fprintf(out, " ❌ Failure (%v)\n", err)
			results = append(results, calibrationResult{threshold, 0, err})
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return handleCalculationError(err, duration, config.Timeout, out)
			}
			continue
		}

		fmt.Fprintf(out, " ✅ Success (Duration: %s)\n", duration)
		results = append(results, calibrationResult{threshold, duration, nil})
		if duration < bestDuration {
			bestDuration, bestThreshold = duration, threshold
		}
	}

	fmt.Fprintln(out, "\n--- Calibration Summary ---")
	fmt.Fprintf(out, "  %-12s │ %s\n", "Threshold", "Execution Time")
	fmt.Fprintf(out, "  %s┼%s\n", strings.Repeat("─", 14), strings.Repeat("─", 25))
	for _, res := range results {
		thresholdLabel := fmt.Sprintf("%d bits", res.Threshold)
		if res.Threshold == 0 {
			thresholdLabel = "Sequential"
		}
		durationStr := "N/A"
		if res.Err == nil {
			durationStr = res.Duration.String()
		}
		highlight := ""
		if res.Threshold == bestThreshold && res.Err == nil {
			highlight = " (Optimal)"
		}
		fmt.Fprintf(out, "  %-12s │ %s%s\n", thresholdLabel, durationStr, highlight)
	}
	fmt.Fprintf(out, "\n✅ Recommendation for this machine: --threshold %d\n", bestThreshold)
	return ExitSuccess
}

// run is the main function that orchestrates the application's execution.
func run(ctx context.Context, config AppConfig, out io.Writer) int {
	if config.Calibrate {
		return runCalibration(ctx, config, out)
	}
	ctx, cancelTimeout := context.WithTimeout(ctx, config.Timeout)
	defer cancelTimeout()
	ctx, stopSignals := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	fmt.Fprintln(out, "--- Execution Configuration ---")
	fmt.Fprintf(out, "Calculating F(%d) with a timeout of %s.\n", config.N, config.Timeout)
	fmt.Fprintf(out, "Environment: %d logical CPUs, Go %s.\n", runtime.NumCPU(), runtime.Version())
	fmt.Fprintf(out, "Optimization thresholds: Parallelism=%d bits, FFT=%d bits.\n", config.Threshold, config.FFTThreshold)

	calculatorsToRun := getCalculatorsToRun(config)
	if len(calculatorsToRun) > 1 {
		fmt.Fprintln(out, "Execution mode: Parallel comparison of all algorithms.")
	} else {
		fmt.Fprintf(out, "Execution mode: Simple calculation with the %s algorithm.\n", calculatorsToRun[0].Name())
	}
	fmt.Fprintln(out, "\n--- Start of Execution ---")

	results := executeCalculations(ctx, calculatorsToRun, config, out)
	return analyzeComparisonResults(results, config, out)
}

// getCalculatorsToRun selects the calculators to run.
func getCalculatorsToRun(config AppConfig) []fibonacci.Calculator {
	if config.Algo == "all" {
		keys := getSortedCalculatorKeys()
		calculators := make([]fibonacci.Calculator, len(keys))
		for i, k := range keys {
			calculators[i] = calculatorRegistry[k]
		}
		return calculators
	}
	return []fibonacci.Calculator{calculatorRegistry[config.Algo]}
}

// executeCalculations orchestrates the concurrent execution of calculations.
func executeCalculations(ctx context.Context, calculators []fibonacci.Calculator, config AppConfig, out io.Writer) []CalculationResult {
	g, ctx := errgroup.WithContext(ctx)
	results := make([]CalculationResult, len(calculators))
	progressChan := make(chan fibonacci.ProgressUpdate, len(calculators)*ProgressBufferMultiplier)

	for i, calc := range calculators {
		idx, calculator := i, calc
		g.Go(func() error {
			startTime := time.Now()
			res, err := calculator.Calculate(ctx, progressChan, idx, config.N, config.Threshold, config.FFTThreshold)
			results[idx] = CalculationResult{
				Name: calculator.Name(), Result: res, Duration: time.Since(startTime), Err: err,
			}
			return nil
		})
	}

	var displayWg sync.WaitGroup
	displayWg.Add(1)
	go cli.DisplayAggregateProgress(&displayWg, progressChan, len(calculators), out)

	_ = g.Wait()
	close(progressChan)
	displayWg.Wait()

	return results
}

// analyzeComparisonResults analyzes and displays the results.
func analyzeComparisonResults(results []CalculationResult, config AppConfig, out io.Writer) int {
	sort.Slice(results, func(i, j int) bool {
		if (results[i].Err == nil) != (results[j].Err == nil) {
			return results[i].Err == nil
		}
		return results[i].Duration < results[j].Duration
	})

	var firstValidResult *big.Int
	var firstError error
	successCount := 0

	fmt.Fprintln(out, "\n--- Comparison Summary ---")
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "Algorithm\tDuration\tStatus")
	fmt.Fprintln(tw, "----------\t-----\t------")
	for _, res := range results {
		var status string
		if res.Err != nil {
			status = fmt.Sprintf("❌ Failure (%v)", res.Err)
			if firstError == nil {
				firstError = res.Err
			}
		} else {
			status = "✅ Success"
			successCount++
			if firstValidResult == nil {
				firstValidResult = res.Result
			}
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", res.Name, res.Duration.String(), status)
	}
	tw.Flush()

	if successCount == 0 {
		fmt.Fprintln(out, "\nGlobal Status: Failure. None of the algorithms could complete the calculation.")
		return handleCalculationError(firstError, 0, config.Timeout, out)
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
	cli.DisplayResult(firstValidResult, config.N, 0, config.Verbose, config.Details, out)
	return ExitSuccess
}

// handleCalculationError interprets an error and returns the appropriate exit code.
func handleCalculationError(err error, duration time.Duration, timeout time.Duration, out io.Writer) int {
	if err == nil {
		return ExitSuccess
	}
	msgSuffix := ""
	if duration > 0 {
		msgSuffix = fmt.Sprintf(" after %s", duration)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		fmt.Fprintf(out, "Status: Failure (Timeout). The execution time limit of %s was exceeded%s.\n", timeout, msgSuffix)
		return ExitErrorTimeout
	}
	if errors.Is(err, context.Canceled) {
		fmt.Fprintf(out, "Status: Canceled by user%s.\n", msgSuffix)
		return ExitErrorCanceled
	}
	fmt.Fprintf(out, "Status: Failure. An unexpected error occurred: %v\n", err)
	return ExitErrorGeneric
}