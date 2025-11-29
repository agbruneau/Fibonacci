package calibration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
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

// cliColorProvider implements apperrors.ColorProvider using cli theme functions.
type cliColorProvider struct{}

func (c cliColorProvider) Yellow() string { return cli.ColorYellow() }
func (c cliColorProvider) Reset() string  { return cli.ColorReset() }

// i18nMessageProvider implements apperrors.ErrorMessageProvider using i18n.Messages.
type i18nMessageProvider struct{}

func (m i18nMessageProvider) GetMessage(key string) string {
	if msg, ok := i18n.Messages[key]; ok {
		return msg
	}
	return key
}

// CalibrationOptions configures the calibration process.
type CalibrationOptions struct {
	// ProfilePath is the path to save/load the calibration profile.
	// If empty, uses the default path.
	ProfilePath string
	// SaveProfile indicates whether to save the calibration results.
	SaveProfile bool
	// LoadProfile indicates whether to try loading an existing profile.
	LoadProfile bool
}

// calibrationResult holds the result of a single threshold test.
type calibrationResult struct {
	Threshold int
	Duration  time.Duration
	Err       error
}

// RunCalibration executes a comprehensive benchmark to determine the optimal
// parallelism threshold for the current hardware.
//
// It uses adaptive threshold generation based on CPU characteristics and
// iterates through the generated thresholds, executing a standard Fibonacci
// calculation (N=10,000,000) for each. The execution times are recorded and
// compared to identify the threshold that yields the fastest performance.
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
	return RunCalibrationWithOptions(ctx, out, calculatorRegistry, CalibrationOptions{
		SaveProfile: true,
		LoadProfile: false, // Full calibration should run fresh
	})
}

// RunCalibrationWithOptions executes calibration with the specified options.
func RunCalibrationWithOptions(ctx context.Context, out io.Writer, calculatorRegistry map[string]fibonacci.Calculator, opts CalibrationOptions) int {
	fmt.Fprintf(out, "%s\n", i18n.Messages["CalibrationTitle"])

	// Try to load existing profile if requested
	if opts.LoadProfile {
		profile, loaded := LoadOrCreateProfile(opts.ProfilePath)
		if loaded && profile.IsValid() {
			fmt.Fprintf(out, "%sLoaded existing calibration profile from %s%s\n",
				cli.ColorGreen(), GetDefaultProfilePath(), cli.ColorReset())
			fmt.Fprintf(out, "Profile: %s\n", profile.String())
			fmt.Fprintf(out, "\n%s✅ Using cached calibration: %s--threshold %d%s\n",
				cli.ColorGreen(), cli.ColorYellow(), profile.OptimalParallelThreshold, cli.ColorReset())
			return apperrors.ExitSuccess
		}
	}

	calculator := calculatorRegistry["fast"]
	if calculator == nil {
		fmt.Fprintf(out, "%sCritical error: the 'fast' algorithm is required for calibration but was not found.%s\n", cli.ColorRed(), cli.ColorReset())
		return apperrors.ExitErrorGeneric
	}

	// Use adaptive thresholds based on CPU characteristics
	thresholdsToTest := GenerateParallelThresholds()
	fmt.Fprintf(out, "%sUsing adaptive thresholds for %d CPU cores%s\n",
		cli.ColorCyan(), runtime.NumCPU(), cli.ColorReset())

	results := make([]calibrationResult, 0, len(thresholdsToTest))
	bestDuration := time.Duration(1<<63 - 1)
	bestThreshold := 0
	calibrationStart := time.Now()

	var wg sync.WaitGroup
	progressChan := make(chan fibonacci.ProgressUpdate, 5)
	wg.Add(1)
	go cli.DisplayProgress(&wg, progressChan, 1, out)

	for _, threshold := range thresholdsToTest {
		if ctx.Err() != nil {
			fmt.Fprintf(out, "\n%sCalibration interrupted.%s\n", cli.ColorYellow(), cli.ColorReset())
			close(progressChan)
			wg.Wait()
			return apperrors.ExitErrorCanceled
		}

		startTime := time.Now()
		_, err := calculator.Calculate(ctx, progressChan, 0, fibonacci.CalibrationN, fibonacci.Options{ParallelThreshold: threshold})
		duration := time.Since(startTime)

		if err != nil {
			fmt.Fprintf(out, "%s❌ Failure (%v)%s\n", cli.ColorRed(), err, cli.ColorReset())
			results = append(results, calibrationResult{threshold, 0, err})
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				close(progressChan)
				wg.Wait()
				return apperrors.HandleCalculationError(err, duration, out, cliColorProvider{}, i18nMessageProvider{})
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

	calibrationDuration := time.Since(calibrationStart)

	// Print results table
	printCalibrationResults(out, results, bestThreshold)

	fmt.Fprintf(out, "\n%s✅ Recommendation for this machine: %s--threshold %d%s\n",
		cli.ColorGreen(), cli.ColorYellow(), bestThreshold, cli.ColorReset())

	// Save profile if requested
	if opts.SaveProfile {
		profile := NewProfile()
		profile.OptimalParallelThreshold = bestThreshold
		profile.OptimalFFTThreshold = EstimateOptimalFFTThreshold()
		profile.OptimalStrassenThreshold = EstimateOptimalStrassenThreshold()
		profile.CalibrationN = fibonacci.CalibrationN
		profile.CalibrationTime = calibrationDuration.String()

		if err := profile.SaveProfile(opts.ProfilePath); err != nil {
			fmt.Fprintf(out, "%sWarning: failed to save profile: %v%s\n",
				cli.ColorYellow(), err, cli.ColorReset())
		} else {
			fmt.Fprintf(out, "%sCalibration profile saved to %s%s\n",
				cli.ColorGreen(), GetDefaultProfilePath(), cli.ColorReset())
		}
	}

	return apperrors.ExitSuccess
}

// printCalibrationResults formats and prints the calibration results table.
func printCalibrationResults(out io.Writer, results []calibrationResult, bestThreshold int) {
	fmt.Fprintf(out, "\n%s\n", i18n.Messages["CalibrationSummary"])
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	fmt.Fprintf(tw, "  %sThreshold%s    │ %sExecution Time%s\n", cli.ColorUnderline(), cli.ColorReset(), cli.ColorUnderline(), cli.ColorReset())
	fmt.Fprintf(tw, "  %s┼%s\n", strings.Repeat("─", 14), strings.Repeat("─", 25))
	for _, res := range results {
		thresholdLabel := fmt.Sprintf("%d bits", res.Threshold)
		if res.Threshold == 0 {
			thresholdLabel = "Sequential"
		}
		durationStr := fmt.Sprintf("%sN/A%s", cli.ColorRed(), cli.ColorReset())
		if res.Err == nil {
			durationStr = cli.FormatExecutionDuration(res.Duration)
			if res.Duration == 0 {
				durationStr = "< 1µs"
			}
		}
		highlight := ""
		if res.Threshold == bestThreshold && res.Err == nil {
			highlight = fmt.Sprintf(" %s(Optimal)%s", cli.ColorGreen(), cli.ColorReset())
		}
		fmt.Fprintf(tw, "  %s%-12s%s │ %s%s%s%s\n", cli.ColorCyan(), thresholdLabel, cli.ColorReset(), cli.ColorYellow(), durationStr, cli.ColorReset(), highlight)
	}
	tw.Flush()
}

// AutoCalibrate runs a quick startup calibration to fine-tune performance
// parameters.
//
// Unlike the full `RunCalibration`, this function performs a heuristic search
// for optimal values for parallelism, FFT, and Strassen thresholds using a
// subset of candidates generated adaptively based on CPU characteristics.
// It is designed to be fast enough to run at application startup without
// significant delay.
//
// The function first checks for an existing valid calibration profile. If found
// and valid for the current hardware, it uses the cached values instead of
// running benchmarks.
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
	return AutoCalibrateWithProfile(parentCtx, cfg, out, calculatorRegistry, cfg.CalibrationProfile)
}

// AutoCalibrateWithProfile runs auto-calibration with a specific profile path.
func AutoCalibrateWithProfile(parentCtx context.Context, cfg config.AppConfig, out io.Writer, calculatorRegistry map[string]fibonacci.Calculator, profilePath string) (config.AppConfig, bool) {
	// Try to load existing profile first
	if profile, loaded := LoadOrCreateProfile(profilePath); loaded && profile.IsValid() {
		// Use cached calibration
		updated := cfg
		updated.Threshold = profile.OptimalParallelThreshold
		updated.FFTThreshold = profile.OptimalFFTThreshold
		updated.StrassenThreshold = profile.OptimalStrassenThreshold

		fmt.Fprintf(out, "%sUsing cached calibration%s: parallelism=%s%d%s bits, FFT=%s%d%s bits, Strassen=%s%d%s bits\n",
			cli.ColorGreen(), cli.ColorReset(),
			cli.ColorYellow(), updated.Threshold, cli.ColorReset(),
			cli.ColorYellow(), updated.FFTThreshold, cli.ColorReset(),
			cli.ColorYellow(), updated.StrassenThreshold, cli.ColorReset())
		return updated, true
	}

	calc := calculatorRegistry["fast"]
	if calc == nil {
		return cfg, false
	}

	perTrial := cfg.Timeout / 6
	if perTrial < 2*time.Second {
		perTrial = 2 * time.Second
	}

	tryRun := func(threshold, fftThreshold int) (time.Duration, error) {
		ctx, cancel := context.WithTimeout(parentCtx, perTrial)
		defer cancel()
		start := time.Now()
		_, err := calc.Calculate(ctx, nil, 0, fibonacci.CalibrationN, fibonacci.Options{ParallelThreshold: threshold, FFTThreshold: fftThreshold})
		return time.Since(start), err
	}

	// Use adaptive thresholds
	parallelCandidates := GenerateQuickParallelThresholds()
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

	fftCandidates := GenerateQuickFFTThresholds()
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
		strassenCandidates := GenerateQuickStrassenThresholds()
		for _, cand := range strassenCandidates {
			ctx, cancel := context.WithTimeout(parentCtx, perTrial)
			start := time.Now()
			_, err := matCalc.Calculate(ctx, nil, 0, fibonacci.CalibrationN, fibonacci.Options{ParallelThreshold: bestPar, StrassenThreshold: cand})
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

	// Save the calibration profile for future use
	profile := NewProfile()
	profile.OptimalParallelThreshold = updated.Threshold
	profile.OptimalFFTThreshold = updated.FFTThreshold
	profile.OptimalStrassenThreshold = updated.StrassenThreshold
	profile.CalibrationN = fibonacci.CalibrationN
	if err := profile.SaveProfile(profilePath); err != nil {
		// Non-fatal: just log and continue
		fmt.Fprintf(out, "%sWarning: could not save calibration profile: %v%s\n",
			cli.ColorYellow(), err, cli.ColorReset())
	}

	fmt.Fprintf(out, "%sAuto-calibration%s: parallelism=%s%d%s bits, FFT=%s%d%s bits, Strassen=%s%d%s bits\n",
		cli.ColorGreen(), cli.ColorReset(),
		cli.ColorYellow(), updated.Threshold, cli.ColorReset(),
		cli.ColorYellow(), updated.FFTThreshold, cli.ColorReset(),
		cli.ColorYellow(), updated.StrassenThreshold, cli.ColorReset())
	return updated, true
}

// LoadCachedCalibration attempts to load a cached calibration profile and
// apply it to the configuration. Returns the updated config and true if
// a valid cached profile was found.
func LoadCachedCalibration(cfg config.AppConfig, profilePath string) (config.AppConfig, bool) {
	profile, loaded := LoadOrCreateProfile(profilePath)
	if !loaded || !profile.IsValid() {
		return cfg, false
	}

	updated := cfg
	updated.Threshold = profile.OptimalParallelThreshold
	updated.FFTThreshold = profile.OptimalFFTThreshold
	updated.StrassenThreshold = profile.OptimalStrassenThreshold
	return updated, true
}
