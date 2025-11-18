// The main package is the entry point of the fibcalc application. It handles
// command-line argument parsing, configuration, calculation orchestration,
// and result display.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"syscall"

	"example.com/fibcalc/internal/calibration"
	"example.com/fibcalc/internal/cli"
	"example.com/fibcalc/internal/config"
	"example.com/fibcalc/internal/fibonacci"
	"example.com/fibcalc/internal/i18n"
	"example.com/fibcalc/internal/orchestration"
	"example.com/fibcalc/internal/server"
)

// Application exit codes define the standard exit statuses for the application.
const (
	ExitSuccess       = 0
	ExitErrorGeneric  = 1
	ExitErrorTimeout  = 2
	ExitErrorMismatch = 3
	ExitErrorConfig   = 4
	ExitErrorCanceled = 130
)

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

func getSortedCalculatorKeys() []string {
	keys := make([]string, 0, len(calculatorRegistry))
	for k := range calculatorRegistry {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

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
			fmt.Fprintln(os.Stderr, "[i18n] failed to load translations:", err)
		}
	}
	// Setting the Strassen threshold for the matrix algorithm
	fibonacci.DefaultStrassenThresholdBits = cfg.StrassenThreshold

	if cfg.ServerMode {
		srv := &server.Server{Registry: calculatorRegistry}
		if err := srv.Start(cfg.Port); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(ExitErrorGeneric)
		}
		return
	}

	exitCode := run(context.Background(), cfg, os.Stdout)
	os.Exit(exitCode)
}

func run(ctx context.Context, cfg config.AppConfig, out io.Writer) int {
	if cfg.Calibrate {
		return calibration.RunCalibration(ctx, out, calculatorRegistry)
	}
	ctx, cancelTimeout := context.WithTimeout(ctx, cfg.Timeout)
	defer cancelTimeout()
	ctx, stopSignals := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	// Quick auto-calibration at startup (if enabled)
	if cfg.AutoCalibrate {
		if updated, ok := calibration.AutoCalibrate(ctx, cfg, out, calculatorRegistry); ok {
			cfg = updated
		}
	}

	if !cfg.JSONOutput {
		writeOut(out, "%s\n", i18n.Messages["ExecConfigTitle"])
		writeOut(out, "Calculating %sF(%d)%s with a timeout of %s%s%s.\n",
			cli.ColorMagenta, cfg.N, cli.ColorReset, cli.ColorYellow, cfg.Timeout, cli.ColorReset)
		writeOut(out, "Environment: %s%d%s logical processors, Go %s%s%s.\n",
			cli.ColorCyan, runtime.NumCPU(), cli.ColorReset, cli.ColorCyan, runtime.Version(), cli.ColorReset)
		writeOut(out, "Optimization thresholds: Parallelism=%s%d%s bits, FFT=%s%d%s bits.\n",
			cli.ColorCyan, cfg.Threshold, cli.ColorReset, cli.ColorCyan, cfg.FFTThreshold, cli.ColorReset)
	}

	calculatorsToRun := getCalculatorsToRun(cfg)
	if !cfg.JSONOutput {
		var modeDesc string
		if len(calculatorsToRun) > 1 {
			modeDesc = "Parallel comparison of all algorithms"
		} else {
			modeDesc = fmt.Sprintf("Single calculation with the %s%s%s algorithm",
				cli.ColorGreen, calculatorsToRun[0].Name(), cli.ColorReset)
		}
		writeOut(out, "Execution mode: %s.\n", modeDesc)
		writeOut(out, "\n%s\n", i18n.Messages["ExecStartTitle"])
	}

	results := orchestration.ExecuteCalculations(ctx, calculatorsToRun, cfg, out)

	if cfg.JSONOutput {
		return printJSONResults(results, out)
	}

	return orchestration.AnalyzeComparisonResults(results, cfg, out)
}

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

func writeOut(out io.Writer, format string, a ...interface{}) {
	if _, err := fmt.Fprintf(out, format, a...); err != nil {
		fmt.Fprintln(os.Stderr, "[Output Error]:", err)
	}
}

func printJSONResults(results []orchestration.CalculationResult, out io.Writer) int {
	type jsonResult struct {
		Algorithm string `json:"algorithm"`
		Duration  string `json:"duration"`
		Result    string `json:"result,omitempty"`
		Error     string `json:"error,omitempty"`
	}
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
		return ExitErrorGeneric
	}
	return ExitSuccess
}
