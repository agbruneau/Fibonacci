package orchestration

import (
	"context"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	apperrors "github.com/agbruneau/FibGo/internal/errors"
	"github.com/agbruneau/FibGo/internal/fibonacci"
	"github.com/agbruneau/FibGo/internal/fibonacci/memory"
	"github.com/agbruneau/FibGo/internal/progress"
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
//   - cfg: Execution configuration (calculators, n, options, reporter, etc.).
//
// Returns:
//   - []CalculationResult: A slice containing the results of each calculation.
func ExecuteCalculations(ctx context.Context, cfg ExecutionConfig) []CalculationResult {
	results := make([]CalculationResult, len(cfg.Calculators))
	progressChan := make(chan progress.ProgressUpdate, len(cfg.Calculators)*ProgressBufferMultiplier)

	var displayWg sync.WaitGroup
	displayWg.Add(1)
	go cfg.ProgressReporter.DisplayProgress(&displayWg, progressChan, len(cfg.Calculators), cfg.Out)

	// Fast path: single calculator doesn't need errgroup overhead
	if len(cfg.Calculators) == 1 {
		startTime := time.Now()
		res, err := cfg.Calculators[0].Calculate(ctx, progressChan, 0, cfg.N, cfg.Opts)
		results[0] = CalculationResult{
			Name: cfg.Calculators[0].Name(), Result: res, Duration: time.Since(startTime),
			Err: wrapCalculationFailure(err, cfg.N, cfg.Opts),
		}
	} else {
		g, ctx := errgroup.WithContext(ctx)
		for i, calc := range cfg.Calculators {
			idx, calculator := i, calc
			g.Go(func() error {
				startTime := time.Now()
				res, err := calculator.Calculate(ctx, progressChan, idx, cfg.N, cfg.Opts)
				results[idx] = CalculationResult{
					Name: calculator.Name(), Result: res, Duration: time.Since(startTime),
					Err: wrapCalculationFailure(err, cfg.N, cfg.Opts),
				}
				// Propagate the raw calculation error to errgroup so that a
				// failure in one calculator cancels sibling goroutines via the
				// shared context. Each goroutine's error is already captured
				// in results[idx].Err above, so g.Wait()'s return value is
				// redundant for callers — but propagating here enables
				// cross-calculator cancellation (P0-08).
				return err
			})
		}
		// g.Wait() returns the first non-nil error from a goroutine. We
		// intentionally discard it: per-calculator errors are already stored
		// in results[i].Err (see goroutine body above), and callers read the
		// error from there. Wait() is still required to block until all
		// goroutines finish before closing progressChan below.
		_ = g.Wait()
	}

	close(progressChan)
	displayWg.Wait()

	return results
}

func wrapCalculationFailure(err error, n uint64, opts fibonacci.Options) error {
	if err == nil {
		return nil
	}
	est := memory.EstimateMemoryUsage(n)
	return apperrors.WrapCalculationError(err, apperrors.CalculationContext{
		N:                     n,
		ParallelThresholdBits: opts.ParallelThreshold,
		FFTThresholdBits:      opts.FFTThreshold,
		StrassenThresholdBits: opts.StrassenThreshold,
		MemoryEstimateBytes:   est.TotalBytes,
	})
}

// AnalyzeComparisonResults processes the results from multiple algorithms and
// generates a summary report.
//
// It sorts the results by execution time, validates consistency across
// successful calculations, and displays a comparative table. It handles the
// logic for determining global success or failure based on the individual
// outcomes.
//
// It mutates results in place (sort.Slice): callers must not hold pointers
// into the slice across this call, and must not rely on the input order
// afterward.
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
		if firstError == nil {
			// No calculator produced a result at all (e.g. none were constructed
			// or selected): there is no error to classify, but this is still a
			// failure — never report success.
			return apperrors.ExitErrorGeneric
		}
		return errHandler.HandleError(firstError, 0, out)
	}

	if HasResultMismatch(results) {
		fmt.Fprintf(out, "\nGlobal Status: CRITICAL ERROR! An inconsistency was detected between the results of the algorithms.")
		return apperrors.ExitErrorMismatch
	}

	fmt.Fprintf(out, "\nGlobal Status: Success. All valid results are consistent.\n")
	presenter.PresentResult(*firstValidResult, presOpts.N, presOpts.Verbose, presOpts.Details, presOpts.ShowValue, out)
	return apperrors.ExitSuccess
}

// HasResultMismatch reports whether two or more successful results disagree on
// the computed value. Failed results (Err != nil) are ignored; with fewer than
// two successes it returns false. It is the single source of truth for the
// cross-algorithm consistency check, shared by the comparison report and the
// quiet-mode fast path so the two cannot drift.
func HasResultMismatch(results []CalculationResult) bool {
	firstIdx := -1
	for i := range results {
		if results[i].Err != nil {
			continue
		}
		if firstIdx == -1 {
			firstIdx = i
			continue
		}
		if results[i].Result.Cmp(results[firstIdx].Result) != 0 {
			return true
		}
	}
	return false
}
