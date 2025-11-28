package calibration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"example.com/fibcalc/internal/cli"
	"example.com/fibcalc/internal/config"
	apperrors "example.com/fibcalc/internal/errors"
	"example.com/fibcalc/internal/fibonacci"
	"example.com/fibcalc/internal/i18n"
)

// RunCalibration executes a comprehensive benchmark to determine the optimal
// parallelism threshold for the current hardware.
//
// It iterates through a predefined set of bit thresholds (from 0 to 16384),
// executing a standard Fibonacci calculation (N=10,000,000) for each. The
// execution times are recorded and compared to identify the threshold that yields
// the fastest performance.
//
// Parameters:
//   - ctx: The context for managing cancellation and deadlines.
//   - out: The io.Writer to which progress and results will be written.
//   - calculatorRegistry: A map of available calculators, which must include
//     the "fast" algorithm.
//
// Returns:
//   - int: The exit code (0 for success, non-zero for errors).
func RunCalibration(ctx context.Context, out io.Writer, calculatorRegistry map[string]fibonacci.Calculator) int {
	fmt.Fprintf(out, "%s\n", i18n.Messages["CalibrationTitle"])
	const calibrationN = 10_000_000
	calculator := calculatorRegistry["fast"]
	if calculator == nil {
		fmt.Fprintf(out, "%sCritical error: the 'fast' algorithm is required for calibration but was not found.%s\n", cli.ColorRed, cli.ColorReset)
		return apperrors.ExitErrorGeneric
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

	var wg sync.WaitGroup
	progressChan := make(chan fibonacci.ProgressUpdate, 5) // Buffer size 5
	wg.Add(1)
	go cli.DisplayProgress(&wg, progressChan, 1, out)

	for _, threshold := range thresholdsToTest {
		if ctx.Err() != nil {
			fmt.Fprintf(out, "\n%sCalibration interrupted.%s\n", cli.ColorYellow, cli.ColorReset)
			return apperrors.ExitErrorCanceled
		}

		startTime := time.Now()
		_, err := calculator.Calculate(ctx, progressChan, 0, calibrationN, fibonacci.Options{ParallelThreshold: threshold})
		duration := time.Since(startTime)

		if err != nil {
			fmt.Fprintf(out, "%s❌ Failure (%v)%s\n", cli.ColorRed, err, cli.ColorReset)
			results = append(results, calibrationResult{threshold, 0, err})
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				close(progressChan)
				wg.Wait()
				return handleCalculationError(err, duration, out)
			}
			continue
		}

		results = append(results, calibrationResult{threshold, duration, nil})
		if duration < bestDuration {
			bestDuration, bestThreshold = duration, threshold
		}
	}
	close(progressChan)
	wg.Wait()

	fmt.Fprintf(out, "\n%s\n", i18n.Messages["CalibrationSummary"])
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	fmt.Fprintf(tw, "  %sThreshold%s    │ %sExecution Time%s\n", cli.ColorUnderline, cli.ColorReset, cli.ColorUnderline, cli.ColorReset)
	fmt.Fprintf(tw, "  %s┼%s\n", strings.Repeat("─", 14), strings.Repeat("─", 25))
	for _, res := range results {
		thresholdLabel := fmt.Sprintf("%d bits", res.Threshold)
		if res.Threshold == 0 {
			thresholdLabel = "Sequential"
		}
		durationStr := fmt.Sprintf("%sN/A%s", cli.ColorRed, cli.ColorReset)
		if res.Err == nil {
			durationStr = cli.FormatExecutionDuration(res.Duration)
			if res.Duration == 0 {
				durationStr = "< 1µs"
			}
		}
		highlight := ""
		if res.Threshold == bestThreshold && res.Err == nil {
			highlight = fmt.Sprintf(" %s(Optimal)%s", cli.ColorGreen, cli.ColorReset)
		}
		fmt.Fprintf(tw, "  %s%-12s%s │ %s%s%s%s\n", cli.ColorCyan, thresholdLabel, cli.ColorReset, cli.ColorYellow, durationStr, cli.ColorReset, highlight)
	}
	tw.Flush()
	fmt.Fprintf(out, "\n%s✅ Recommendation for this machine: %s--threshold %d%s\n",
		cli.ColorGreen, cli.ColorYellow, bestThreshold, cli.ColorReset)
	return apperrors.ExitSuccess
}

// AutoCalibrate runs a quick startup calibration to fine-tune performance
// parameters.
//
// Unlike the full `RunCalibration`, this function performs a heuristic search
// for optimal values for parallelism, FFT, and Strassen thresholds using a
// subset of candidates. It is designed to be fast enough to run at application
// startup without significant delay.
//
// Parameters:
//   - parentCtx: The context used to manage the calibration timeout.
//   - cfg: The initial application configuration, providing starting values.
//   - out: The io.Writer for logging calibration results.
//   - calculatorRegistry: The map of available calculators.
//
// Returns:
//   - config.AppConfig: The updated configuration with optimized thresholds.
//   - bool: True if calibration was successful, false otherwise.
func AutoCalibrate(parentCtx context.Context, cfg config.AppConfig, out io.Writer, calculatorRegistry map[string]fibonacci.Calculator) (config.AppConfig, bool) {
	calc := calculatorRegistry["fast"]
	if calc == nil {
		return cfg, false
	}

	perTrial := cfg.Timeout / 6
	if perTrial < 2*time.Second {
		perTrial = 2 * time.Second
	}

	const nForCalibration = 10_000_000

	tryRun := func(threshold, fftThreshold int) (time.Duration, error) {
		ctx, cancel := context.WithTimeout(parentCtx, perTrial)
		defer cancel()
		start := time.Now()
		_, err := calc.Calculate(ctx, nil, 0, nForCalibration, fibonacci.Options{ParallelThreshold: threshold, FFTThreshold: fftThreshold})
		return time.Since(start), err
	}

	parallelCandidates := []int{0, 2048, 4096, 8192, 16384}
	bestPar := cfg.Threshold
	bestParDur := time.Duration(1<<63 - 1)
	for _, cand := range parallelCandidates {
		dur, err := tryRun(cand, 0)
		if err != nil {
			continue
		}
		if dur < bestParDur {
			bestParDur, bestPar = dur, cand
		}
	}

	fftCandidates := []int{0, 16000, 20000, 28000}
	bestFFT := cfg.FFTThreshold
	bestFFTDur := time.Duration(1<<63 - 1)
	for _, cand := range fftCandidates {
		dur, err := tryRun(bestPar, cand)
		if err != nil {
			continue
		}
		if dur < bestFFTDur {
			bestFFTDur, bestFFT = dur, cand
		}
	}

	matCalc := calculatorRegistry["matrix"]
	bestStrassen := cfg.StrassenThreshold
	bestStrassenDur := time.Duration(1<<63 - 1)
	if matCalc != nil {
		strassenCandidates := []int{192, 256, 384, 512}
		for _, cand := range strassenCandidates {
			ctx, cancel := context.WithTimeout(parentCtx, perTrial)
			start := time.Now()
			_, err := matCalc.Calculate(ctx, nil, 0, nForCalibration, fibonacci.Options{ParallelThreshold: bestPar, StrassenThreshold: cand})
			cancel()
			dur := time.Since(start)
			if err != nil {
				continue
			}
			if dur < bestStrassenDur {
				bestStrassenDur = dur
				bestStrassen = cand
			}
		}
	}

	if bestParDur == time.Duration(1<<63-1) && bestFFTDur == time.Duration(1<<63-1) {
		return cfg, false
	}

	updated := cfg
	if bestParDur != time.Duration(1<<63-1) {
		updated.Threshold = bestPar
	}
	if bestFFTDur != time.Duration(1<<63-1) {
		updated.FFTThreshold = bestFFT
	}
	if bestStrassenDur != time.Duration(1<<63-1) {
		updated.StrassenThreshold = bestStrassen
	}

	fmt.Fprintf(out, "%sAuto-calibration%s: parallelism=%s%d%s bits, FFT=%s%d%s bits, Strassen=%s%d%s bits\n",
		cli.ColorGreen, cli.ColorReset,
		cli.ColorYellow, updated.Threshold, cli.ColorReset,
		cli.ColorYellow, updated.FFTThreshold, cli.ColorReset,
		cli.ColorYellow, updated.StrassenThreshold, cli.ColorReset)
	return updated, true
}

// handleCalculationError formats and prints error messages related to failed calculations.
// It distinguishes between different error types (timeout, cancellation, generic)
// to provide the user with specific feedback.
//
// Parameters:
//   - err: The error that occurred.
//   - duration: The duration of the calculation before it failed.
//   - out: The io.Writer to which the error message will be written.
//
// Returns:
//   - int: The appropriate exit code for the error type.
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
