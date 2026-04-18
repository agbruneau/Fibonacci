package calibration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/agbru/fibcalc/internal/config"
	apperrors "github.com/agbru/fibcalc/internal/errors"
	"github.com/agbru/fibcalc/internal/fibonacci"
	"github.com/agbru/fibcalc/internal/progress"
	"github.com/agbru/fibcalc/internal/ui"
)

// ProgressDisplayFunc is a function that displays progress from a channel.
// It decouples calibration from CLI display concerns.
type ProgressDisplayFunc func(wg *sync.WaitGroup, progressChan <-chan progress.ProgressUpdate, numCalculators int, out io.Writer)

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
func RunCalibration(ctx context.Context, out io.Writer, calculatorRegistry map[string]fibonacci.Calculator, progressDisplay ProgressDisplayFunc, colorProvider apperrors.ColorProvider) int {
	return RunCalibrationWithOptions(ctx, out, calculatorRegistry, CalibrationOptions{
		SaveProfile: true,
		LoadProfile: false, // Full calibration should run fresh
	}, progressDisplay, colorProvider)
}

// RunCalibrationWithOptions executes calibration with the specified options.
//
// P2-05: the body was previously a single 80-line function mixing profile
// short-circuit, hardware detection, pass execution, result aggregation and
// profile save. It now delegates to three focused helpers
// (configureHardwareDetection, runPassSequence, persistCalibrationProfile)
// so each concern stays under the package's funlen / cyclo thresholds and
// can be exercised independently by tests.
func RunCalibrationWithOptions(ctx context.Context, out io.Writer, calculatorRegistry map[string]fibonacci.Calculator, opts CalibrationOptions, progressDisplay ProgressDisplayFunc, colorProvider apperrors.ColorProvider) int {
	fmt.Fprintf(out, "--- Calibration Mode: Finding the Optimal Parallelism Threshold ---\n")

	// Try to load existing profile if requested
	if opts.LoadProfile {
		if code, handled := tryUseCachedCalibrationProfile(opts.ProfilePath, out); handled {
			return code
		}
	}

	calculator, thresholdsToTest, code := configureHardwareDetection(out, calculatorRegistry)
	if calculator == nil {
		return code
	}

	calibrationStart := time.Now()
	bestThreshold, results, code := runPassSequence(ctx, out, calculator, thresholdsToTest, progressDisplay, colorProvider)
	if code != apperrors.ExitSuccess {
		return code
	}
	calibrationDuration := time.Since(calibrationStart)

	// Print results table
	printCalibrationResults(out, results, bestThreshold)

	fmt.Fprintf(out, "\n%s✅ Recommendation for this machine: %s--threshold %d%s\n",
		ui.ColorGreen(), ui.ColorYellow(), bestThreshold, ui.ColorReset())

	if opts.SaveProfile {
		persistCalibrationProfile(out, opts.ProfilePath, bestThreshold, calibrationDuration)
	}

	return apperrors.ExitSuccess
}

// tryUseCachedCalibrationProfile attempts to short-circuit calibration by
// loading an existing valid profile. Returns (exitCode, true) if the caller
// should return early with that code; (0, false) if no valid profile exists
// and the caller must run a fresh calibration.
func tryUseCachedCalibrationProfile(profilePath string, out io.Writer) (int, bool) {
	profile, loaded := LoadOrCreateProfile(profilePath)
	if !loaded || !profile.IsValid() {
		return 0, false
	}
	fmt.Fprintf(out, "%sLoaded existing calibration profile from %s%s\n",
		ui.ColorGreen(), GetDefaultProfilePath(), ui.ColorReset())
	fmt.Fprintf(out, "Profile: %s\n", profile.String())
	fmt.Fprintf(out, "\n%s✅ Using cached calibration: %s--threshold %d%s\n",
		ui.ColorGreen(), ui.ColorYellow(), profile.OptimalParallelThreshold, ui.ColorReset())
	return apperrors.ExitSuccess, true
}

// configureHardwareDetection resolves the calculator used for calibration
// and the hardware-adaptive threshold list. On failure (missing "fast"
// calculator) it returns (nil, nil, ExitErrorGeneric); the caller must
// return that code. On success it returns the calculator, the ordered
// threshold list to test, and ExitSuccess.
func configureHardwareDetection(out io.Writer, calculatorRegistry map[string]fibonacci.Calculator) (fibonacci.Calculator, []int, int) {
	calculator := calculatorRegistry["fast"]
	if calculator == nil {
		fmt.Fprintf(out, "%sCritical error: the 'fast' algorithm is required for calibration but was not found.%s\n", ui.ColorRed(), ui.ColorReset())
		return nil, nil, apperrors.ExitErrorGeneric
	}
	thresholdsToTest := GenerateParallelThresholds()
	fmt.Fprintf(out, "%sUsing adaptive thresholds for %d CPU cores%s\n",
		ui.ColorCyan(), runtime.NumCPU(), ui.ColorReset())
	return calculator, thresholdsToTest, apperrors.ExitSuccess
}

// runPassSequence executes one calibration pass per threshold, tracking
// the fastest. It manages the progress-display goroutine lifecycle and
// translates context cancellation / calculation errors into exit codes.
//
// Returns:
//   - bestThreshold: the threshold that yielded the shortest duration.
//   - results: every pass's outcome (used for the summary table).
//   - code: ExitSuccess when at least one pass produced a valid timing;
//     ExitErrorCanceled on context cancellation mid-loop;
//     ExitErrorGeneric when every pass failed to produce a valid result;
//     HandleCalculationError's return code on unrecoverable calc error.
func runPassSequence(ctx context.Context, out io.Writer, calculator fibonacci.Calculator, thresholdsToTest []int, progressDisplay ProgressDisplayFunc, colorProvider apperrors.ColorProvider) (int, []calibrationResult, int) {
	results := make([]calibrationResult, 0, len(thresholdsToTest))
	bestDuration := time.Duration(1<<63 - 1)
	bestThreshold := 0

	var wg sync.WaitGroup
	progressChan := make(chan progress.ProgressUpdate, 5)
	wg.Add(1)
	go progressDisplay(&wg, progressChan, 1, out)

	for _, threshold := range thresholdsToTest {
		if ctx.Err() != nil {
			fmt.Fprintf(out, "\n%sCalibration interrupted.%s\n", ui.ColorYellow(), ui.ColorReset())
			close(progressChan)
			wg.Wait()
			return 0, results, apperrors.ExitErrorCanceled
		}

		startTime := time.Now()
		_, err := calculator.Calculate(ctx, progressChan, 0, fibonacci.CalibrationN, fibonacci.Options{ParallelThreshold: threshold})
		duration := time.Since(startTime)

		if err != nil {
			fmt.Fprintf(out, "%s❌ Failure (%v)%s\n", ui.ColorRed(), err, ui.ColorReset())
			results = append(results, calibrationResult{threshold, 0, err})
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				close(progressChan)
				wg.Wait()
				return 0, results, apperrors.HandleCalculationError(err, duration, out, colorProvider)
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

	if bestDuration == time.Duration(1<<63-1) {
		fmt.Fprintf(out, "\n%sCalibration failed: no valid results obtained.%s\n", ui.ColorRed(), ui.ColorReset())
		return 0, results, apperrors.ExitErrorGeneric
	}
	return bestThreshold, results, apperrors.ExitSuccess
}

// persistCalibrationProfile materialises a full-confidence profile for
// the winning parallel threshold and writes it to profilePath. Warnings
// are printed to out but save errors are non-fatal (the caller's exit
// code reflects calibration success, not persistence success).
func persistCalibrationProfile(out io.Writer, profilePath string, bestThreshold int, calibrationDuration time.Duration) {
	profile := NewProfile()
	profile.OptimalParallelThreshold = bestThreshold
	profile.OptimalFFTThreshold = config.EstimateOptimalFFTThreshold()
	profile.OptimalStrassenThreshold = config.EstimateOptimalStrassenThreshold()
	profile.CalibrationN = fibonacci.CalibrationN
	profile.CalibrationTime = calibrationDuration.String()
	profile.Confidence = 1.0

	if err := profile.SaveProfile(profilePath); err != nil {
		fmt.Fprintf(out, "%sWarning: failed to save profile: %v%s\n",
			ui.ColorYellow(), err, ui.ColorReset())
		return
	}
	fmt.Fprintf(out, "%sCalibration profile saved to %s%s\n",
		ui.ColorGreen(), GetDefaultProfilePath(), ui.ColorReset())
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
			ui.ColorGreen(), ui.ColorReset(),
			ui.ColorYellow(), updated.Threshold, ui.ColorReset(),
			ui.ColorYellow(), updated.FFTThreshold, ui.ColorReset(),
			ui.ColorYellow(), updated.StrassenThreshold, ui.ColorReset())
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
			ui.ColorGreen(), ui.ColorReset(),
			microResults.Duration.Round(time.Millisecond),
			ui.ColorYellow(), updated.Threshold, ui.ColorReset(),
			ui.ColorYellow(), updated.FFTThreshold, ui.ColorReset(),
			microResults.Confidence*100)

		// Save profile for future use
		saveCalibrationProfile(updated, profilePath, out, microResults.Confidence)
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
	saveCalibrationProfile(updated, profilePath, out, 1.0)
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
//   - confidence: The confidence score of the calibration.
func saveCalibrationProfile(cfg config.AppConfig, profilePath string, out io.Writer, confidence float64) {
	profile := NewProfile()
	profile.OptimalParallelThreshold = cfg.Threshold
	profile.OptimalFFTThreshold = cfg.FFTThreshold
	profile.OptimalStrassenThreshold = cfg.StrassenThreshold
	profile.CalibrationN = fibonacci.CalibrationN
	profile.Confidence = confidence

	if err := profile.SaveProfile(profilePath); err != nil {
		fmt.Fprintf(out, "%sWarning: could not save calibration profile: %v%s\n",
			ui.ColorYellow(), err, ui.ColorReset())
	}
}
