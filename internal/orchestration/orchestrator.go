package orchestration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sort"
	"sync"
	"text/tabwriter"
	"time"

	"golang.org/x/sync/errgroup"

	"example.com/fibcalc/internal/cli"
	"example.com/fibcalc/internal/config"
	apperrors "example.com/fibcalc/internal/errors"
	"example.com/fibcalc/internal/fibonacci"
	"example.com/fibcalc/internal/i18n"
)

// CalculationResult encapsulates the outcome of a single Fibonacci calculation.
// It serves as a standardized container for results from different algorithms,
// facilitating comparison and reporting.
type CalculationResult struct {
	// Name is the identifier of the algorithm used (e.g., "Fast Doubling").
	Name string
	// Result is the computed Fibonacci number. It is nil if an error occurred.
	Result *big.Int
	// Duration is the time taken to complete the calculation.
	Duration time.Duration
	// Err contains any error that occurred during the calculation.
	Err error
}

// ProgressBufferMultiplier defines the buffer size multiplier for the progress
// channel. A larger buffer reduces the likelihood of blocking calculation
// goroutines when the UI is slow to consume updates.
const ProgressBufferMultiplier = 5

// ExecuteCalculations orchestrates the concurrent execution of one or more
// Fibonacci calculations.
//
// It manages the lifecycle of calculation goroutines, collects their results,
// and coordinates the display of progress updates. This function is the core of
// the application's concurrency model.
//
// The context ctx is used for cancellation. The list of calculators to run is
// calculators. The application configuration is cfg. The writer out is used
// for progress display.
//
// It returns a slice of CalculationResult containing the outcomes of all
// executed calculations.
func ExecuteCalculations(ctx context.Context, calculators []fibonacci.Calculator, cfg config.AppConfig, out io.Writer) []CalculationResult {
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
			return nil
		})
	}

	g.Wait()
	close(progressChan)
	displayWg.Wait()

	return results
}

// AnalyzeComparisonResults processes the results from multiple algorithms and
// generates a summary report.
//
// It sorts the results by execution time, validates consistency across
// successful calculations, and displays a comparative table. It handles the
// logic for determining global success or failure based on the individual
// outcomes.
//
// The results to analyze are results. The application configuration is cfg. The
// writer out is used for the report output.
//
// It returns an exit code (0 for success, non-zero for errors).
func AnalyzeComparisonResults(results []CalculationResult, cfg config.AppConfig, out io.Writer) int {
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

	fmt.Fprintf(out, "\n%s\n", i18n.Messages["ComparisonSummary"])
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	fmt.Fprintf(tw, "%sAlgorithm%s\t%sDuration%s\t%sStatus%s\n",
		cli.ColorUnderline, cli.ColorReset, cli.ColorUnderline, cli.ColorReset, cli.ColorUnderline, cli.ColorReset)

	for _, res := range results {
		var status string
		if res.Err != nil {
			status = fmt.Sprintf("%s❌ Failure (%v)%s", cli.ColorRed, res.Err, cli.ColorReset)
			if firstError == nil {
				firstError = res.Err
			}
		} else {
			status = fmt.Sprintf("%s✅ Success%s", cli.ColorGreen, cli.ColorReset)
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
		fmt.Fprintf(tw, "%s%s%s\t%s%s%s\t%s\n",
			cli.ColorBlue, res.Name, cli.ColorReset,
			cli.ColorYellow, duration, cli.ColorReset,
			status)
	}
	tw.Flush()

	if successCount == 0 {
		fmt.Fprintf(out, "\n%s\n", i18n.Messages["GlobalStatusFailure"])
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
		fmt.Fprintf(out, "\n%s", i18n.Messages["StatusCriticalMismatch"])
		return apperrors.ExitErrorMismatch
	}

	fmt.Fprintf(out, "\n%s", i18n.Messages["GlobalStatusSuccess"])
	cli.DisplayResult(firstValidResult, cfg.N, firstValidResultDuration, cfg.Verbose, cfg.Details, out)
	return apperrors.ExitSuccess
}

// handleCalculationError formats and displays an error message for a failed calculation.
// It provides specific messages for timeout and cancellation scenarios.
//
// The error is err. The execution duration is duration. The writer out is used for output.
//
// It returns an appropriate exit code.
func handleCalculationError(err error, duration time.Duration, out io.Writer) int {
	if err == nil {
		return apperrors.ExitSuccess
	}
	msgSuffix := ""
	if duration > 0 {
		msgSuffix = fmt.Sprintf(" after %s%s%s", cli.ColorYellow, duration, cli.ColorReset)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		fmt.Fprintf(out, "%s\n", i18n.Messages["StatusTimeout"])
		return apperrors.ExitErrorTimeout
	}
	if errors.Is(err, context.Canceled) {
		fmt.Fprintf(out, "%s%s%s.%s\n", cli.ColorYellow, i18n.Messages["StatusCanceled"], msgSuffix, cli.ColorReset)
		return apperrors.ExitErrorCanceled
	}
	fmt.Fprintf(out, "%s\n", i18n.Messages["StatusFailure"])
	return apperrors.ExitErrorGeneric
}
