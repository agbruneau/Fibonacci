package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/signal"
	"syscall"
	"time"

	"github.com/agbruneau/FibGo/internal/cli"
	"github.com/agbruneau/FibGo/internal/config"
	apperrors "github.com/agbruneau/FibGo/internal/errors"
	"github.com/agbruneau/FibGo/internal/fibonacci"
	"github.com/agbruneau/FibGo/internal/fibonacci/memory"
	"github.com/agbruneau/FibGo/internal/orchestration"
	"github.com/agbruneau/FibGo/internal/ui"
)

// runCalculate is a thin orchestrator: it dispatches to the partial-digits
// path, validates the memory budget, sets up the lifecycle (timeout +
// signals), runs the calculators and presents the results.
func (a *Application) runCalculate(ctx context.Context, out io.Writer) int {
	// Partial computation mode: last K digits only
	if a.Config.LastDigits > 0 {
		return a.runLastDigits(ctx, out)
	}

	// Memory budget validation
	if code := a.validateMemoryBudget(out); code != apperrors.ExitSuccess {
		return code
	}

	// Setup lifecycle (timeout + signals)
	ctx, cancelTimeout := context.WithTimeout(ctx, a.Config.Timeout)
	defer cancelTimeout()
	ctx, stopSignals := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	// Execute calculations
	results := a.executeCalculations(ctx, out)

	// Build output config and present results
	outputCfg := cli.OutputConfig{
		OutputFile: a.Config.OutputFile,
		Quiet:      a.Config.Quiet,
		Verbose:    a.Config.Verbose,
		ShowValue:  a.Config.ShowValue,
	}
	return a.analyzeResultsWithOutput(results, outputCfg, out)
}

// executeCalculations runs the configured calculators and returns their
// results. It also handles the (verbose) banner printing and progress
// reporter selection based on quiet mode.
func (a *Application) executeCalculations(ctx context.Context, out io.Writer) []orchestration.CalculationResult {
	calculatorsToRun := orchestration.GetCalculatorsToRun(a.Config.Algo, a.Factory)

	if !a.Config.Quiet {
		cli.PrintExecutionConfig(a.Config, out)
		names := make([]string, len(calculatorsToRun))
		for i, c := range calculatorsToRun {
			names[i] = c.Name()
		}
		cli.PrintExecutionMode(names, out)
	}

	var progressReporter orchestration.ProgressReporter
	progressOut := out
	if a.Config.Quiet {
		progressOut = io.Discard
		progressReporter = orchestration.NullProgressReporter{}
	} else {
		progressReporter = orchestration.ProgressReporterFunc(cli.DisplayProgress)
	}

	opts := fibonacci.Options{
		ParallelThreshold: a.Config.Threshold,
		FFTThreshold:      a.Config.FFTThreshold,
		StrassenThreshold: a.Config.StrassenThreshold,
		GCMode:            a.Config.GCControl,
	}
	return orchestration.ExecuteCalculations(ctx, orchestration.ExecutionConfig{
		Calculators:      calculatorsToRun,
		N:                a.Config.N,
		Opts:             opts,
		ProgressReporter: progressReporter,
		Out:              progressOut,
	})
}

// validateMemoryBudget checks if the estimated memory usage fits within the
// configured limit. It delegates the pure validation logic to
// config.ValidateMemoryBudget and only handles presentation and exit-code
// mapping here.
func (a *Application) validateMemoryBudget(out io.Writer) int {
	report, err := config.ValidateMemoryBudget(a.Config)
	if err == nil {
		if a.Config.MemoryLimit != "" && !a.Config.Quiet {
			fmt.Fprintf(out, "Memory estimate: %s (limit: %s)\n",
				memory.FormatMemoryEstimate(report.Estimate), report.LimitRaw)
		}
		return apperrors.ExitSuccess
	}

	// Error paths report on ErrWriter: stdout stays reserved for results so
	// scripts capturing it never receive error text as data (ERR-02).
	var parseErr config.MemoryLimitParseError
	if errors.As(err, &parseErr) {
		fmt.Fprintf(a.ErrWriter, "Invalid --memory-limit: %v\n", parseErr.Cause)
		return apperrors.ExitErrorConfig
	}

	var memErr apperrors.MemoryError
	if errors.As(err, &memErr) {
		fmt.Fprintf(a.ErrWriter, "Estimated memory %s exceeds limit %s.\n",
			memory.FormatMemoryEstimate(report.Estimate),
			report.LimitRaw)
		if a.Config.LastDigits == 0 {
			fmt.Fprintf(a.ErrWriter, "Consider using --last-digits K for O(K) memory usage.\n")
		}
		return apperrors.ExitErrorConfig
	}

	// Unknown error type: surface it generically.
	fmt.Fprintf(a.ErrWriter, "Memory budget validation failed: %v\n", err)
	return apperrors.ExitErrorConfig
}

// runLastDigits computes only the last K decimal digits of F(N) using modular
// arithmetic, requiring O(K) memory regardless of N. It owns the lifecycle
// (timeout + signals) and presentation; the math itself lives in
// orchestration.ComputeLastDigits.
func (a *Application) runLastDigits(ctx context.Context, out io.Writer) int {
	ctx, cancelTimeout := context.WithTimeout(ctx, a.Config.Timeout)
	defer cancelTimeout()
	ctx, stopSignals := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	k := a.Config.LastDigits
	n := a.Config.N

	// Defense in depth: config.Validate already rejects this at parse time
	// (audit M-02). The guard stays for callers that build an Application
	// programmatically and never run Validate, which is exactly how the
	// SEC-03 regression test reaches this path.
	if k > config.MaxLastDigits {
		fmt.Fprintf(a.ErrWriter, "Error: --last-digits %d exceeds the maximum of %d digits.\n", k, config.MaxLastDigits)
		return apperrors.ExitErrorConfig
	}

	if !a.Config.Quiet {
		fmt.Fprintf(out, "Computing last %d digits of F(%d)...\n", k, n)
	}

	start := time.Now()
	res, err := orchestration.ComputeLastDigits(ctx, n, k)
	if err != nil {
		// Centralized error handling: maps timeout/cancel/generic to the
		// correct exit code and writes a uniform "Status: …" message to
		// the user-facing stream (matches the comparison-mode behavior).
		colors := apperrors.ColorProvider(cli.CLIColorProvider{})
		if a.Config.MachineOutput {
			colors = apperrors.DefaultColorProvider{}
		}
		return apperrors.HandleCalculationError(err, time.Since(start), a.ErrWriter, colors)
	}

	if a.Config.Quiet {
		fmt.Fprintln(out, res.Digits)
	} else {
		fmt.Fprintf(out, "Last %d digits of F(%d): %s\n", k, n, res.Digits)
		fmt.Fprintf(out, "Computed in %s\n", res.Duration.Round(time.Millisecond))
	}

	return apperrors.ExitSuccess
}

// analyzeResultsWithOutput chains the three result-handling responsibilities:
// selecting the best result, presenting it (quiet vs verbose), and saving it
// to a file when requested.
func (a *Application) analyzeResultsWithOutput(results []orchestration.CalculationResult, outputCfg cli.OutputConfig, out io.Writer) int {
	bestPtr := a.selectBest(results)
	// Copy the value out before present(): AnalyzeComparisonResults sorts
	// results in place, which would invalidate a pointer held into the slice.
	var best *orchestration.CalculationResult
	if bestPtr != nil {
		v := *bestPtr
		best = &v
	}
	exitCode := a.present(results, bestPtr, outputCfg, out)
	if best != nil && exitCode == apperrors.ExitSuccess {
		if err := a.save(best, outputCfg, out); err != nil {
			return apperrors.ExitErrorGeneric
		}
	}
	return exitCode
}

// selectBest picks the fastest successful result, or nil if there is none.
func (a *Application) selectBest(results []orchestration.CalculationResult) *orchestration.CalculationResult {
	return findBestResult(results)
}

// present renders the results to the user and returns the resulting exit
// code. In quiet mode with at least one success, it prints the value alone;
// otherwise it delegates to the comparison analyzer.
func (a *Application) present(
	results []orchestration.CalculationResult,
	best *orchestration.CalculationResult,
	outputCfg cli.OutputConfig,
	out io.Writer,
) int {
	if outputCfg.Quiet && best != nil {
		// Quiet mode must still honor comparison mode's purpose: never emit a
		// value when the algorithms disagree. Without this, `--algo all --quiet`
		// would print the fastest result and exit 0 on a real divergence.
		//
		// Quiet silences stdout, not diagnostics: reporting nothing at all left
		// the caller with an empty stdout and exit 3 to interpret on its own
		// (audit M-06). The message goes to ErrWriter, so a script capturing
		// stdout still gets exactly nothing.
		if orchestration.HasResultMismatch(results) {
			fmt.Fprintf(a.ErrWriter, "%s\n", orchestration.MismatchMessage)
			return apperrors.ExitErrorMismatch
		}
		cli.DisplayQuietResult(out, best.Result)
		return apperrors.ExitSuccess
	}
	presOpts := orchestration.PresentationOptions{
		N:         a.Config.N,
		Verbose:   a.Config.Verbose,
		Details:   a.Config.Details,
		ShowValue: a.Config.ShowValue,
	}
	presenter := cli.CLIResultPresenter{MachineOutput: a.Config.MachineOutput}
	return orchestration.AnalyzeComparisonResults(results, presOpts, presenter, presenter, out, a.ErrWriter)
}

// save writes the result to disk if requested and prints a success notice
// (only in non-quiet mode).
func (a *Application) save(best *orchestration.CalculationResult, outputCfg cli.OutputConfig, out io.Writer) error {
	if err := a.saveResultIfNeeded(best, outputCfg); err != nil {
		return err
	}
	if outputCfg.OutputFile != "" && !outputCfg.Quiet {
		fmt.Fprintf(out, "\n%s✓ Result saved to: %s%s%s\n",
			ui.ColorGreen(), ui.ColorCyan(), outputCfg.OutputFile, ui.ColorReset())
	}
	return nil
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
		fmt.Fprintf(a.ErrWriter, "Error saving result: %v\n", err)
		return err
	}
	return nil
}
