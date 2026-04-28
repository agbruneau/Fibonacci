package app

import (
	"context"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/agbru/fibcalc/internal/cli"
	apperrors "github.com/agbru/fibcalc/internal/errors"
	"github.com/agbru/fibcalc/internal/fibonacci"
	"github.com/agbru/fibcalc/internal/fibonacci/memory"
	"github.com/agbru/fibcalc/internal/orchestration"
	"github.com/agbru/fibcalc/internal/ui"
)

// runCalculate orchestrates the execution of the CLI calculation command.
func (a *Application) runCalculate(ctx context.Context, out io.Writer) int {
	// Partial computation mode: last K digits only
	if a.Config.LastDigits > 0 {
		return a.runLastDigits(ctx, out)
	}

	// Memory budget validation
	if a.Config.MemoryLimit != "" {
		if code := a.validateMemoryBudget(out); code != apperrors.ExitSuccess {
			return code
		}
	}

	// Setup lifecycle (timeout + signals)
	ctx, cancelTimeout := context.WithTimeout(ctx, a.Config.Timeout)
	defer cancelTimeout()
	ctx, stopSignals := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	// Get calculators to run
	calculatorsToRun := orchestration.GetCalculatorsToRun(a.Config.Algo, a.Factory)

	// Skip verbose output in quiet mode
	if !a.Config.Quiet {
		cli.PrintExecutionConfig(a.Config, out)
		names := make([]string, len(calculatorsToRun))
		for i, c := range calculatorsToRun {
			names[i] = c.Name()
		}
		cli.PrintExecutionMode(names, out)
	}

	// Choose progress reporter based on quiet mode
	var progressReporter orchestration.ProgressReporter
	progressOut := out
	if a.Config.Quiet {
		progressOut = io.Discard
		progressReporter = orchestration.NullProgressReporter{}
	} else {
		progressReporter = orchestration.ProgressReporterFunc(cli.DisplayProgress)
	}

	// Execute calculations
	opts := fibonacci.Options{
		ParallelThreshold: a.Config.Threshold,
		FFTThreshold:      a.Config.FFTThreshold,
		StrassenThreshold: a.Config.StrassenThreshold,
	}
	results := orchestration.ExecuteCalculations(ctx, orchestration.ExecutionConfig{
		Calculators:      calculatorsToRun,
		N:                a.Config.N,
		Opts:             opts,
		ProgressReporter: progressReporter,
		Out:              progressOut,
	})

	// Build output config for the CLI options
	outputCfg := cli.OutputConfig{
		OutputFile: a.Config.OutputFile,
		Quiet:      a.Config.Quiet,
		Verbose:    a.Config.Verbose,
		ShowValue:  a.Config.ShowValue,
	}

	return a.analyzeResultsWithOutput(results, outputCfg, out)
}

// validateMemoryBudget checks if the estimated memory usage fits within the configured limit.
func (a *Application) validateMemoryBudget(out io.Writer) int {
	limit, err := memory.ParseMemoryLimit(a.Config.MemoryLimit)
	if err != nil {
		fmt.Fprintf(out, "Invalid --memory-limit: %v\n", err)
		return apperrors.ExitErrorConfig
	}
	est := memory.EstimateMemoryUsage(a.Config.N)
	if est.TotalBytes > limit {
		fmt.Fprintf(out, "Estimated memory %s exceeds limit %s.\n",
			memory.FormatMemoryEstimate(est),
			a.Config.MemoryLimit)
		if a.Config.LastDigits == 0 {
			fmt.Fprintf(out, "Consider using --last-digits K for O(K) memory usage.\n")
		}
		return apperrors.ExitErrorConfig
	}
	if !a.Config.Quiet {
		fmt.Fprintf(out, "Memory estimate: %s (limit: %s)\n",
			memory.FormatMemoryEstimate(est), a.Config.MemoryLimit)
	}
	return apperrors.ExitSuccess
}

// runLastDigits computes only the last K decimal digits of F(N) using modular
// arithmetic, requiring O(K) memory regardless of N.
func (a *Application) runLastDigits(ctx context.Context, out io.Writer) int {
	ctx, cancelTimeout := context.WithTimeout(ctx, a.Config.Timeout)
	defer cancelTimeout()
	ctx, stopSignals := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	k := a.Config.LastDigits
	n := a.Config.N

	// Compute modulus = 10^k
	mod := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(k)), nil)

	if !a.Config.Quiet {
		fmt.Fprintf(out, "Computing last %d digits of F(%d)...\n", k, n)
	}

	start := time.Now()
	result, err := fibonacci.FastDoublingMod(n, mod)
	elapsed := time.Since(start)

	if err != nil {
		fmt.Fprintf(a.ErrWriter, "Error: %v\n", err)
		return apperrors.ExitErrorGeneric
	}

	// Format with leading zeros to exactly k digits
	format := fmt.Sprintf("%%0%ds", k)
	digits := fmt.Sprintf(format, result.String())

	if a.Config.Quiet {
		fmt.Fprintln(out, digits)
	} else {
		fmt.Fprintf(out, "Last %d digits of F(%d): %s\n", k, n, digits)
		fmt.Fprintf(out, "Computed in %s\n", elapsed.Round(time.Millisecond))
	}

	return apperrors.ExitSuccess
}

func (a *Application) analyzeResultsWithOutput(results []orchestration.CalculationResult, outputCfg cli.OutputConfig, out io.Writer) int {
	bestResult := findBestResult(results)

	// Handle quiet mode for single result
	if outputCfg.Quiet && bestResult != nil {
		cli.DisplayQuietResult(out, bestResult.Result, a.Config.N, bestResult.Duration)

		// Save to file if requested
		if err := a.saveResultIfNeeded(bestResult, outputCfg); err != nil {
			return apperrors.ExitErrorGeneric
		}

		return apperrors.ExitSuccess
	}

	// Use standard analysis for non-quiet mode
	presOpts := orchestration.PresentationOptions{
		N:         a.Config.N,
		Verbose:   a.Config.Verbose,
		Details:   a.Config.Details,
		ShowValue: a.Config.ShowValue,
	}
	presenter := cli.CLIResultPresenter{MachineOutput: a.Config.MachineOutput}
	exitCode := orchestration.AnalyzeComparisonResults(results, presOpts, presenter, presenter, out)

	// Handle file output for non-quiet mode
	if bestResult != nil && exitCode == apperrors.ExitSuccess {
		// Save to file if requested
		if err := a.saveResultIfNeeded(bestResult, outputCfg); err != nil {
			return apperrors.ExitErrorGeneric
		}
		if outputCfg.OutputFile != "" {
			fmt.Fprintf(out, "\n%s✓ Result saved to: %s%s%s\n",
				ui.ColorGreen(), ui.ColorCyan(), outputCfg.OutputFile, ui.ColorReset())
		}
	}

	return exitCode
}

func findBestResult(results []orchestration.CalculationResult) *orchestration.CalculationResult {
	var bestResult *orchestration.CalculationResult
	for i := range results {
		if results[i].Err == nil {
			if bestResult == nil || results[i].Duration < bestResult.Duration {
				bestResult = &results[i]
			}
		}
	}
	return bestResult
}

func (a *Application) saveResultIfNeeded(res *orchestration.CalculationResult, cfg cli.OutputConfig) error {
	if cfg.OutputFile == "" {
		return nil
	}
	if err := cli.WriteResultToFile(res.Result, a.Config.N, res.Duration, res.Name, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving result: %v\n", err)
		return err
	}
	return nil
}
