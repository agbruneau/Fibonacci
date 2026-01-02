package app

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/agbru/fibcalc/internal/calibration"
	"github.com/agbru/fibcalc/internal/cli"
	"github.com/agbru/fibcalc/internal/config"
	apperrors "github.com/agbru/fibcalc/internal/errors"
	"github.com/agbru/fibcalc/internal/fibonacci"
	"github.com/agbru/fibcalc/internal/orchestration"
	"github.com/agbru/fibcalc/internal/server"
	"github.com/agbru/fibcalc/internal/ui"
)

// Application represents the fibcalc application instance.
// It encapsulates the configuration and provides methods to run
// the application in various modes (CLI, server, REPL).
type Application struct {
	// Config holds the parsed application configuration.
	Config config.AppConfig
	// Factory provides access to the Fibonacci calculator implementations.
	// Uses the interface type for better testability and dependency injection.
	Factory fibonacci.CalculatorFactory
	// ErrWriter is the writer for error output (typically os.Stderr).
	ErrWriter io.Writer
}

// New creates a new Application instance by parsing command-line arguments.
// It validates the configuration and returns an error if parsing or validation fails.
//
// Parameters:
//   - args: The command-line arguments (typically os.Args).
//   - errWriter: The writer for error output.
//
// Returns:
//   - *Application: A new application instance.
//   - error: An error if configuration parsing or validation fails.
func New(args []string, errWriter io.Writer) (*Application, error) {
	factory := fibonacci.GlobalFactory()
	availableAlgos := factory.List()

	// args[0] is program name, args[1:] are the actual arguments
	programName := "fibcalc"
	var cmdArgs []string
	if len(args) > 0 {
		programName = args[0]
		cmdArgs = args[1:]
	}

	cfg, err := config.ParseConfig(programName, cmdArgs, errWriter, availableAlgos)
	if err != nil {
		return nil, err
	}

	// Try to load cached calibration profile first
	// This allows the application to use optimal thresholds found in previous runs
	if cfgWithProfile, loaded := calibration.LoadCachedCalibration(cfg, cfg.CalibrationProfile); loaded {
		cfg = cfgWithProfile
	} else {
		// Fallback to adaptive thresholds based on hardware characteristics
		// This provides automatic optimization without requiring --auto-calibrate
		cfg = applyAdaptiveThresholds(cfg)
	}

	return &Application{
		Config:    cfg,
		Factory:   factory,
		ErrWriter: errWriter,
	}, nil
}

// applyAdaptiveThresholds adjusts the configuration thresholds based on
// hardware characteristics (CPU cores, architecture) when default values
// are detected. This provides automatic performance optimization without
// requiring explicit calibration.
//
// The function only modifies thresholds that are set to their static default
// values, preserving any user-specified overrides via command-line flags.
//
// Parameters:
//   - cfg: The initial configuration with potentially default threshold values.
//
// Returns:
//   - config.AppConfig: The configuration with adaptive thresholds applied.
func applyAdaptiveThresholds(cfg config.AppConfig) config.AppConfig {
	// Only adjust thresholds if they're at the static default values.
	// This preserves explicit user overrides via --threshold, --fft-threshold, etc.

	// Parallel threshold: adapt based on CPU core count
	if cfg.Threshold == fibonacci.DefaultParallelThreshold {
		cfg.Threshold = calibration.EstimateOptimalParallelThreshold()
	}

	// FFT threshold: adapt based on architecture (32-bit vs 64-bit)
	if cfg.FFTThreshold == fibonacci.DefaultFFTThreshold {
		cfg.FFTThreshold = calibration.EstimateOptimalFFTThreshold()
	}

	// Strassen threshold: adapt based on CPU core count
	if cfg.StrassenThreshold == fibonacci.DefaultStrassenThreshold {
		cfg.StrassenThreshold = calibration.EstimateOptimalStrassenThreshold()
	}

	return cfg
}

// Run executes the application based on the configured mode.
// It dispatches to the appropriate handler (completion, server, REPL, or CLI).
//
// Parameters:
//   - ctx: The context for managing cancellation and timeouts.
//   - out: The writer for standard output.
//
// Returns:
//   - int: An exit code (0 for success, non-zero for errors).
func (a *Application) Run(ctx context.Context, out io.Writer) int {
	// Handle completion script generation
	if a.Config.Completion != "" {
		return a.runCompletion(out)
	}

	// Initialize CLI theme (respects --no-color flag and NO_COLOR env var)
	ui.InitTheme(a.Config.NoColor)

	// Server mode
	if a.Config.ServerMode {
		return a.runServer()
	}

	// Interactive REPL mode
	if a.Config.Interactive {
		return a.runREPL()
	}

	// Calibration mode
	if a.Config.Calibrate {
		return a.runCalibration(ctx, out)
	}

	// Run auto-calibration if enabled
	a.Config = a.runAutoCalibrationIfEnabled(ctx, out)

	// Standard CLI calculation mode
	return a.runCalculate(ctx, out)
}

// runCompletion generates shell completion scripts.
func (a *Application) runCompletion(out io.Writer) int {
	availableAlgos := a.Factory.List()
	if err := cli.GenerateCompletion(out, a.Config.Completion, availableAlgos); err != nil {
		fmt.Fprintf(a.ErrWriter, "Error generating completion: %v\n", err)
		return apperrors.ExitErrorConfig
	}
	return apperrors.ExitSuccess
}

// runServer starts the HTTP server mode.
func (a *Application) runServer() int {
	srv := server.NewServer(a.Factory, a.Config)
	if err := srv.Start(); err != nil {
		fmt.Fprintf(a.ErrWriter, "Server error: %v\n", err)
		return apperrors.ExitErrorGeneric
	}
	return apperrors.ExitSuccess
}

// runREPL starts the interactive REPL mode.
func (a *Application) runREPL() int {
	repl := cli.NewREPL(a.Factory.GetAll(), cli.REPLConfig{
		DefaultAlgo:  a.Config.Algo,
		Timeout:      a.Config.Timeout,
		Threshold:    a.Config.Threshold,
		FFTThreshold: a.Config.FFTThreshold,
		HexOutput:    a.Config.HexOutput,
	})
	repl.Start()
	return apperrors.ExitSuccess
}

// runCalibration runs the full calibration mode.
func (a *Application) runCalibration(ctx context.Context, out io.Writer) int {
	return calibration.RunCalibration(ctx, out, a.Factory.GetAll())
}

// runAutoCalibrationIfEnabled runs auto-calibration if enabled in the configuration.
// Returns the potentially updated configuration with calibrated threshold values.
func (a *Application) runAutoCalibrationIfEnabled(ctx context.Context, out io.Writer) config.AppConfig {
	if a.Config.AutoCalibrate {
		if updated, ok := calibration.AutoCalibrate(ctx, a.Config, out, a.Factory.GetAll()); ok {
			return updated
		}
	}
	return a.Config
}

// runCalculate orchestrates the execution of the CLI calculation command.
func (a *Application) runCalculate(ctx context.Context, out io.Writer) int {
	// Setup lifecycle (timeout + signals)
	ctx, cancelTimeout := context.WithTimeout(ctx, a.Config.Timeout)
	defer cancelTimeout()
	ctx, stopSignals := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	// Get calculators to run
	calculatorsToRun := cli.GetCalculatorsToRun(a.Config, a.Factory)

	// Skip verbose output in quiet mode
	if !a.Config.JSONOutput && !a.Config.Quiet {
		cli.PrintExecutionConfig(a.Config, out)
		cli.PrintExecutionMode(calculatorsToRun, out)
	}

	// In quiet mode, use a discard writer for progress display
	progressOut := out
	if a.Config.Quiet {
		progressOut = io.Discard
	}

	// Execute calculations
	results := orchestration.ExecuteCalculations(ctx, calculatorsToRun, a.Config, progressOut)

	// Handle JSON output
	if a.Config.JSONOutput {
		return printJSONResults(results, out)
	}

	// Build output config for the CLI options
	outputCfg := cli.OutputConfig{
		OutputFile: a.Config.OutputFile,
		HexOutput:  a.Config.HexOutput,
		Quiet:      a.Config.Quiet,
		Verbose:    a.Config.Verbose,
		Concise:    a.Config.Concise,
	}

	return a.analyzeResultsWithOutput(results, outputCfg, out)
}

func (a *Application) analyzeResultsWithOutput(results []orchestration.CalculationResult, outputCfg cli.OutputConfig, out io.Writer) int {
	bestResult := findBestResult(results)

	// Handle quiet mode for single result
	if outputCfg.Quiet && bestResult != nil {
		cli.DisplayQuietResult(out, bestResult.Result, a.Config.N, bestResult.Duration, outputCfg.HexOutput)

		// Save to file if requested
		if err := a.saveResultIfNeeded(bestResult, outputCfg); err != nil {
			return apperrors.ExitErrorGeneric
		}

		return apperrors.ExitSuccess
	}

	// Use standard analysis for non-quiet mode
	exitCode := orchestration.AnalyzeComparisonResults(results, a.Config, out)

	// Handle file output and hex display for non-quiet mode
	if bestResult != nil && exitCode == apperrors.ExitSuccess {
		// Display hex format if requested
		a.displayHexIfNeeded(bestResult, outputCfg, out)

		// Save to file if requested
		if err := a.saveResultIfNeeded(bestResult, outputCfg); err != nil {
			return apperrors.ExitErrorGeneric
		}
		if outputCfg.OutputFile != "" {
			fmt.Fprintf(out, "\n%s✓ Result saved to: %s%s%s\n",
				cli.ColorGreen(), cli.ColorCyan(), outputCfg.OutputFile, cli.ColorReset())
		}
	}

	return exitCode
}

// IsHelpError checks if the error is a help flag error (--help was used).
// This is useful for determining if the application should exit with success
// after displaying help text.
//
// Parameters:
//   - err: The error to check.
//
// Returns:
//   - bool: True if the error indicates help was requested.
func IsHelpError(err error) bool {
	return errors.Is(err, flag.ErrHelp)
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

func (a *Application) displayHexIfNeeded(res *orchestration.CalculationResult, cfg cli.OutputConfig, out io.Writer) {
	if !cfg.HexOutput {
		return
	}
	fmt.Fprintf(out, "\n%s--- Hexadecimal Format ---%s\n", cli.ColorBold(), cli.ColorReset())
	hexStr := res.Result.Text(16)
	if len(hexStr) > 100 && !a.Config.Verbose {
		fmt.Fprintf(out, "F(%s%d%s) [hex] = %s0x%s...%s%s\n",
			cli.ColorMagenta(), a.Config.N, cli.ColorReset(),
			cli.ColorGreen(), hexStr[:40], hexStr[len(hexStr)-40:], cli.ColorReset())
	} else {
		fmt.Fprintf(out, "F(%s%d%s) [hex] = %s0x%s%s\n",
			cli.ColorMagenta(), a.Config.N, cli.ColorReset(),
			cli.ColorGreen(), hexStr, cli.ColorReset())
	}
}

// jsonResult represents a single calculation result in JSON format.
type jsonResult struct {
	Algorithm string `json:"algorithm"`
	Duration  string `json:"duration"`
	Result    string `json:"result,omitempty"`
	Error     string `json:"error,omitempty"`
}

// printJSONResults formats the calculation results as a JSON array and writes
// them to the output. This is useful for programmatic consumption of the results.
func printJSONResults(results []orchestration.CalculationResult, out io.Writer) int {
	output := make([]jsonResult, len(results))
	for i, res := range results {
		jr := jsonResult{
			Algorithm: res.Name,
			Duration:  res.Duration.String(),
		}
		if res.Err != nil {
			jr.Error = res.Err.Error()
		} else {
			jr.Result = res.Result.String()
		}
		output[i] = jr
	}

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		return apperrors.ExitErrorGeneric
	}
	return apperrors.ExitSuccess
}
