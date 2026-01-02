package orchestration

import (
	"context"
	"fmt"
	"io"
	"math/big"
	"sort"
	"sync"
	"text/tabwriter"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/agbru/fibcalc/internal/cli"
	"github.com/agbru/fibcalc/internal/config"
	apperrors "github.com/agbru/fibcalc/internal/errors"
	"github.com/agbru/fibcalc/internal/fibonacci"
	"github.com/agbru/fibcalc/internal/ui"
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
// Parameters:
//   - ctx: The context for managing cancellation and deadlines.
//   - calculators: A slice of calculators to execute.
//   - cfg: The application configuration (N, thresholds, etc.).
//   - out: The io.Writer for displaying progress updates.
//
// Returns:
//   - []CalculationResult: A slice containing the results of each calculation.
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
			res, err := calculator.Calculate(ctx, progressChan, idx, cfg.N, cfg.ToCalculationOptions())
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
// Parameters:
//   - results: The slice of calculation results to analyze.
//   - cfg: The application configuration.
//   - out: The io.Writer for the summary report.
//
// Returns:
//   - int: An exit code indicating success (0) or the type of failure.
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

	fmt.Fprintf(out, "\n--- Comparison Summary ---\n")
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	fmt.Fprintf(tw, "%sAlgorithm%s\t%sDuration%s\t%sStatus%s\n",
		ui.ColorUnderline(), ui.ColorReset(), ui.ColorUnderline(), ui.ColorReset(), ui.ColorUnderline(), ui.ColorReset())

	for _, res := range results {
		var status string
		if res.Err != nil {
			status = fmt.Sprintf("%s❌ Failure (%v)%s", ui.ColorRed(), res.Err, ui.ColorReset())
			if firstError == nil {
				firstError = res.Err
			}
		} else {
			status = fmt.Sprintf("%s✅ Success%s", ui.ColorGreen(), ui.ColorReset())
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
			ui.ColorBlue(), res.Name, ui.ColorReset(),
			ui.ColorYellow(), duration, ui.ColorReset(),
			status)
	}
	if err := tw.Flush(); err != nil {
		fmt.Fprintf(out, "Warning: failed to flush tabwriter: %v\n", err)
	}

	if successCount == 0 {
		fmt.Fprintf(out, "\nGlobal Status: Failure. No algorithm could complete the calculation.\n")
		return apperrors.HandleCalculationError(firstError, 0, out, cli.CLIColorProvider{})
	}

	mismatch := false
	for _, res := range results {
		if res.Err == nil && res.Result.Cmp(firstValidResult) != 0 {
			mismatch = true
			break
		}
	}
	if mismatch {
		fmt.Fprintf(out, "\nGlobal Status: CRITICAL ERROR! An inconsistency was detected between the results of the algorithms.")
		return apperrors.ExitErrorMismatch
	}

	fmt.Fprintf(out, "\nGlobal Status: Success. All valid results are consistent.")
	cli.DisplayResult(firstValidResult, cfg.N, firstValidResultDuration, cfg.Verbose, cfg.Details, cfg.Concise, out)
	return apperrors.ExitSuccess
}
