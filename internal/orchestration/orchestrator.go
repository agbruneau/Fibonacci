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
	"example.com/fibcalc/internal/fibonacci"
	"example.com/fibcalc/internal/i18n"
)

// CalculationResult encapsulates the outcome of a single Fibonacci calculation.
type CalculationResult struct {
	Name     string
	Result   *big.Int
	Duration time.Duration
	Err      error
}

const ProgressBufferMultiplier = 5

// ExecuteCalculations orchestrates the concurrent execution of one or more Fibonacci calculations.
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

// ComparisonSummary holds the result of the comparison analysis.
type ComparisonSummary struct {
	SortedResults []CalculationResult
	FirstValid    *CalculationResult
	FirstError    error
	SuccessCount  int
	Mismatch      bool
}

// CompareResults processes the results and returns a summary.
func CompareResults(results []CalculationResult) ComparisonSummary {
	// Clone to avoid modifying the input slice in place if needed,
	// but sort does in-place. We'll copy first.
	sorted := make([]CalculationResult, len(results))
	copy(sorted, results)

	sort.Slice(sorted, func(i, j int) bool {
		if (sorted[i].Err == nil) != (sorted[j].Err == nil) {
			return sorted[i].Err == nil
		}
		return sorted[i].Duration < sorted[j].Duration
	})

	summary := ComparisonSummary{SortedResults: sorted}

	for i := range sorted {
		res := &sorted[i] // Use pointer to avoid copying loop var issues (though here it's a slice of structs)
		if res.Err != nil {
			if summary.FirstError == nil {
				summary.FirstError = res.Err
			}
		} else {
			summary.SuccessCount++
			if summary.FirstValid == nil {
				summary.FirstValid = res
			} else {
				// Check for mismatch against the first valid result
				if res.Result.Cmp(summary.FirstValid.Result) != 0 {
					summary.Mismatch = true
				}
			}
		}
	}
	return summary
}

// AnalyzeComparisonResults processes and displays the final results.
func AnalyzeComparisonResults(results []CalculationResult, cfg config.AppConfig, out io.Writer) int {
	summary := CompareResults(results)

	fmt.Fprintf(out, "\n%s\n", i18n.Messages["ComparisonSummary"])
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	fmt.Fprintf(tw, "%sAlgorithm%s\t%sDuration%s\t%sStatus%s\n",
		cli.ColorUnderline, cli.ColorReset, cli.ColorUnderline, cli.ColorReset, cli.ColorUnderline, cli.ColorReset)

	for _, res := range summary.SortedResults {
		var status string
		if res.Err != nil {
			status = fmt.Sprintf("%s❌ Failure (%v)%s", cli.ColorRed, res.Err, cli.ColorReset)
		} else {
			status = fmt.Sprintf("%s✅ Success%s", cli.ColorGreen, cli.ColorReset)
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

	if summary.SuccessCount == 0 {
		fmt.Fprintf(out, "\n%s\n", i18n.Messages["GlobalStatusFailure"])
		return handleCalculationError(summary.FirstError, 0, out)
	}

	if summary.Mismatch {
		fmt.Fprintf(out, "\n%s", i18n.Messages["StatusCriticalMismatch"])
		return 3 // ExitErrorMismatch
	}

	fmt.Fprintf(out, "\n%s", i18n.Messages["GlobalStatusSuccess"])
	cli.DisplayResult(summary.FirstValid.Result, cfg.N, summary.FirstValid.Duration, cfg.Verbose, cfg.Details, out)
	return 0 // ExitSuccess
}

func handleCalculationError(err error, duration time.Duration, out io.Writer) int {
	if err == nil {
		return 0
	}
	msgSuffix := ""
	if duration > 0 {
		msgSuffix = fmt.Sprintf(" after %s%s%s", cli.ColorYellow, duration, cli.ColorReset)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		fmt.Fprintf(out, "%s\n", i18n.Messages["StatusTimeout"])
		return 2 // ExitErrorTimeout
	}
	if errors.Is(err, context.Canceled) {
		fmt.Fprintf(out, "%s%s%s.%s\n", cli.ColorYellow, i18n.Messages["StatusCanceled"], msgSuffix, cli.ColorReset)
		return 130 // ExitErrorCanceled
	}
	fmt.Fprintf(out, "%s\n", i18n.Messages["StatusFailure"])
	return 1 // ExitErrorGeneric
}
