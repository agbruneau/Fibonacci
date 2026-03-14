package orchestration

import (
	"context"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	apperrors "github.com/agbru/fibcalc/internal/errors"
	"github.com/agbru/fibcalc/internal/fibonacci"
	"github.com/agbru/fibcalc/internal/progress"
)

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
// Parameters:
//   - ctx: The context for managing cancellation and deadlines.
//   - calculators: A slice of calculators to execute.
//   - n: The Fibonacci index to compute.
//   - opts: Calculation options (thresholds, etc.).
//   - progressReporter: The progress reporter for displaying updates (use NullProgressReporter for quiet mode).
//   - out: The io.Writer for displaying progress updates.
//
// Returns:
//   - []CalculationResult: A slice containing the results of each calculation.
func ExecuteCalculations(ctx context.Context, calculators []fibonacci.Calculator, n uint64, opts fibonacci.Options, progressReporter ProgressReporter, out io.Writer) []CalculationResult {
	results := make([]CalculationResult, len(calculators))
	progressChan := make(chan progress.ProgressUpdate, len(calculators)*ProgressBufferMultiplier)

	var displayWg sync.WaitGroup
	displayWg.Add(1)
	go progressReporter.DisplayProgress(&displayWg, progressChan, len(calculators), out)

	// Fast path: single calculator doesn't need errgroup overhead
	if len(calculators) == 1 {
		startTime := time.Now()
		res, err := calculators[0].Calculate(ctx, progressChan, 0, n, opts)
		results[0] = CalculationResult{
			Name: calculators[0].Name(), Result: res, Duration: time.Since(startTime), Err: err,
		}
	} else {
		g, ctx := errgroup.WithContext(ctx)
		for i, calc := range calculators {
			idx, calculator := i, calc
			g.Go(func() error {
				startTime := time.Now()
				res, err := calculator.Calculate(ctx, progressChan, idx, n, opts)
				results[idx] = CalculationResult{
					Name: calculator.Name(), Result: res, Duration: time.Since(startTime), Err: err,
				}
				return nil
			})
		}
		g.Wait()
	}

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
// Parameters:
//   - results: The slice of calculation results to analyze.
//   - presOpts: Presentation options (N, verbose, details, showValue).
//   - presenter: The result presenter for display formatting.
//   - out: The io.Writer for the summary report.
//
// Returns:
//   - int: An exit code indicating success (0) or the type of failure.
func AnalyzeComparisonResults(results []CalculationResult, presOpts PresentationOptions, presenter ResultPresenter, errHandler ErrorHandler, out io.Writer) int {
	sort.Slice(results, func(i, j int) bool {
		if (results[i].Err == nil) != (results[j].Err == nil) {
			return results[i].Err == nil
		}
		return results[i].Duration < results[j].Duration
	})

	var firstValidResult *CalculationResult
	var firstError error
	successCount := 0

	for i := range results {
		if results[i].Err != nil {
			if firstError == nil {
				firstError = results[i].Err
			}
		} else {
			successCount++
			if firstValidResult == nil {
				firstValidResult = &results[i]
			}
		}
	}

	// Present the comparison table
	presenter.PresentComparisonTable(results, out)

	if successCount == 0 {
		fmt.Fprintf(out, "\nGlobal Status: Failure. No algorithm could complete the calculation.\n")
		return errHandler.HandleError(firstError, 0, out)
	}

	mismatch := false
	for _, res := range results {
		if res.Err == nil && res.Result.Cmp(firstValidResult.Result) != 0 {
			mismatch = true
			break
		}
	}
	if mismatch {
		fmt.Fprintf(out, "\nGlobal Status: CRITICAL ERROR! An inconsistency was detected between the results of the algorithms.")
		return apperrors.ExitErrorMismatch
	}

	fmt.Fprintf(out, "\nGlobal Status: Success. All valid results are consistent.\n")
	presenter.PresentResult(*firstValidResult, presOpts.N, presOpts.Verbose, presOpts.Details, presOpts.ShowValue, out)
	return apperrors.ExitSuccess
}
