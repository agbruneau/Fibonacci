package calibration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/agbru/fibcalc/internal/cli"
	"github.com/agbru/fibcalc/internal/config"
	apperrors "github.com/agbru/fibcalc/internal/errors"
	"github.com/agbru/fibcalc/internal/fibonacci"
)

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
	fmt.Fprintf(out, "--- Calibration Mode: Finding the Optimal Parallelism Threshold ---\n")

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
				return apperrors.HandleCalculationError(err, duration, out, cli.CLIColorProvider{})
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

	// Check if we found any valid result
	if bestDuration == time.Duration(1<<63-1) {
		fmt.Fprintf(out, "\n%sCalibration failed: no valid results obtained.%s\n", cli.ColorRed(), cli.ColorReset())
		return apperrors.ExitErrorGeneric
	}

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
func AutoCalibrate(parentCtx context.Context, cfg config.AppConfig, out io.Writer, calculatorRegistry map[string]fibonacci.Calculator) (updated config.AppConfig, ok bool) {
	return AutoCalibrateWithProfile(parentCtx, cfg, out, calculatorRegistry, cfg.CalibrationProfile)
}

// AutoCalibrateWithProfile runs auto-calibration with a specific profile path.
// It first tries to load a cached profile, then falls back to quick micro-benchmarks,
// and finally uses full calibration if needed.
func AutoCalibrateWithProfile(parentCtx context.Context, cfg config.AppConfig, out io.Writer, calculatorRegistry map[string]fibonacci.Calculator, profilePath string) (updated config.AppConfig, ok bool) {
	// Check if calculators are available before attempting calibration
	fastCalc := calculatorRegistry["fast"]
	if fastCalc == nil {
		// No calculators available - cannot calibrate
		return cfg, false
	}

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

	// Try quick micro-benchmarks first (~100ms)
	microResults, err := QuickCalibrate(parentCtx)
	if err == nil && microResults.Confidence >= 0.5 {
		updated := cfg
		updated.Threshold = microResults.ParallelThreshold
		updated.FFTThreshold = microResults.FFTThreshold
		// Keep default Strassen threshold (micro-benchmarks don't test it)

		fmt.Fprintf(out, "%sQuick calibration%s (%v): parallelism=%s%d%s bits, FFT=%s%d%s bits (confidence: %.0f%%)\n",
			cli.ColorGreen(), cli.ColorReset(),
			microResults.Duration.Round(time.Millisecond),
			cli.ColorYellow(), updated.Threshold, cli.ColorReset(),
			cli.ColorYellow(), updated.FFTThreshold, cli.ColorReset(),
			microResults.Confidence*100)

		// Save profile for future use
		saveCalibrationProfile(updated, profilePath, out)
		return updated, true
	}

	// Fall back to full calibration if quick calibration failed or has low confidence

	runner := newCalibrationRunner(parentCtx, cfg.Timeout)

	// Find optimal thresholds
	bestPar, bestParDur := runner.findBestParallelThreshold(fastCalc, cfg.Threshold)
	bestFFT, bestFFTDur := runner.findBestFFTThreshold(fastCalc, bestPar, cfg.FFTThreshold)

	// Find optimal Strassen threshold using matrix calculator
	bestStrassen := cfg.StrassenThreshold
	bestStrassenDur := time.Duration(1<<63 - 1)
	if matCalc := calculatorRegistry["matrix"]; matCalc != nil {
		bestStrassen, bestStrassenDur = runner.findBestStrassenThreshold(matCalc, bestPar, cfg.StrassenThreshold)
	}

	// Apply results and check if calibration was successful
	updated, ok = applyCalibrationResults(cfg, bestPar, bestParDur, bestFFT, bestFFTDur, bestStrassen, bestStrassenDur)
	if !ok {
		return cfg, false
	}

	// Save profile and print output
	saveCalibrationProfile(updated, profilePath, out)
	printCalibrationOutput(updated, out)

	return updated, true
}

// LoadCachedCalibration attempts to load a cached calibration profile and
// apply it to the configuration. Returns the updated config and true if
// a valid cached profile was found.
func LoadCachedCalibration(cfg config.AppConfig, profilePath string) (updated config.AppConfig, ok bool) {
	profile, loaded := LoadOrCreateProfile(profilePath)
	if !loaded || !profile.IsValid() {
		return cfg, false
	}

	updated = cfg
	updated.Threshold = profile.OptimalParallelThreshold
	updated.FFTThreshold = profile.OptimalFFTThreshold
	updated.StrassenThreshold = profile.OptimalStrassenThreshold
	return updated, true
}

// applyCalibrationResults updates the configuration with the calibration results.
//
// Parameters:
//   - cfg: The original configuration.
//   - bestPar: The best parallel threshold found.
//   - bestParDur: The duration achieved with the best parallel threshold.
//   - bestFFT: The best FFT threshold found.
//   - bestFFTDur: The duration achieved with the best FFT threshold.
//   - bestStrassen: The best Strassen threshold found.
//   - bestStrassenDur: The duration achieved with the best Strassen threshold.
//
// Returns:
//   - config.AppConfig: The updated configuration.
//   - bool: true if any valid results were found, false otherwise.
func applyCalibrationResults(cfg config.AppConfig, bestPar int, bestParDur time.Duration, bestFFT int, bestFFTDur time.Duration, bestStrassen int, bestStrassenDur time.Duration) (updated config.AppConfig, ok bool) {
	maxDuration := time.Duration(1<<63 - 1)
	if bestParDur == maxDuration && bestFFTDur == maxDuration {
		return cfg, false
	}

	updated = cfg
	if bestParDur != maxDuration {
		updated.Threshold = bestPar
	}
	if bestFFTDur != maxDuration {
		updated.FFTThreshold = bestFFT
	}
	if bestStrassenDur != maxDuration {
		updated.StrassenThreshold = bestStrassen
	}
	return updated, true
}

// saveCalibrationProfile saves the calibration results to a profile.
//
// Parameters:
//   - cfg: The updated configuration with calibration results.
//   - profilePath: The path to save the profile.
//   - out: The writer for warning messages.
func saveCalibrationProfile(cfg config.AppConfig, profilePath string, out io.Writer) {
	profile := NewProfile()
	profile.OptimalParallelThreshold = cfg.Threshold
	profile.OptimalFFTThreshold = cfg.FFTThreshold
	profile.OptimalStrassenThreshold = cfg.StrassenThreshold
	profile.CalibrationN = fibonacci.CalibrationN

	if err := profile.SaveProfile(profilePath); err != nil {
		fmt.Fprintf(out, "%sWarning: could not save calibration profile: %v%s\n",
			cli.ColorYellow(), err, cli.ColorReset())
	}
}
