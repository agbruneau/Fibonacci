package calibration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/agbruneau/FibGo/internal/config"
	apperrors "github.com/agbruneau/FibGo/internal/errors"
	"github.com/agbruneau/FibGo/internal/fibonacci"
	"github.com/agbruneau/FibGo/internal/progress"
	"github.com/agbruneau/FibGo/internal/ui"
)

// CalibrationN is the standard Fibonacci index used for performance
// calibration runs. This value provides a good balance between:
//   - Being large enough to measure meaningful performance differences
//   - Being small enough to complete calibration in reasonable time
//
// F(10,000,000) has approximately 2,089,877 decimal digits.
//
// Moved from internal/fibonacci/constants.go (audit 2026-06): the constant
// is consumed exclusively by this package.
const CalibrationN = 10_000_000

// DefaultProfileMaxAge is the default freshness window for a cached
// calibration profile. Beyond this age, AutoCalibrateWithProfile ignores
// the cached values and re-runs a full calibration so that the saved
// thresholds keep tracking the current hardware/runtime characteristics.
const DefaultProfileMaxAge = 7 * 24 * time.Hour

// ProfileMaxAgeEnv is the environment variable name read by
// profileMaxAgeFromEnv to override DefaultProfileMaxAge. The value must
// be parseable by time.ParseDuration (e.g. "168h", "30m"). Invalid or
// non-positive values fall back to DefaultProfileMaxAge.
const ProfileMaxAgeEnv = "FIBCALC_PROFILE_MAX_AGE"

// profileMaxAgeFromEnv returns the configured maximum age for a cached
// calibration profile. It honors the FIBCALC_PROFILE_MAX_AGE environment
// variable when set to a valid, positive time.Duration string; otherwise
// it returns DefaultProfileMaxAge.
func profileMaxAgeFromEnv() time.Duration {
	raw := os.Getenv(ProfileMaxAgeEnv)
	if raw == "" {
		return DefaultProfileMaxAge
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return DefaultProfileMaxAge
	}
	return d
}

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
		_, err := calculator.Calculate(ctx, progressChan, 0, CalibrationN, fibonacci.Options{ParallelThreshold: threshold})
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

// persistCalibrationProfile materializes a full-confidence profile for
// the winning parallel threshold and writes it to profilePath. Warnings
// are printed to out but save errors are non-fatal (the caller's exit
// code reflects calibration success, not persistence success).
func persistCalibrationProfile(out io.Writer, profilePath string, bestThreshold int, calibrationDuration time.Duration) {
	profile := NewProfile()
	profile.OptimalParallelThreshold = bestThreshold
	profile.OptimalFFTThreshold = config.EstimateOptimalFFTThreshold()
	profile.OptimalStrassenThreshold = config.EstimateOptimalStrassenThreshold()
	profile.CalibrationN = CalibrationN
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
//
// Lookup order (each step short-circuits when it succeeds):
//  1. Load the on-disk profile; if hardware-valid AND fresh, apply it.
//  2. If the profile is hardware-valid but stale (R1.3), skip the fast
//     tier and run CompleteStrategy so the persisted thresholds track
//     current runtime conditions.
//  3. Otherwise escalate FastStrategy → CompleteStrategy: keep the
//     fast result iff its Confidence is ≥ EscalationConfidenceThreshold.
//
// The exported signature is preserved for compatibility with cmd/ and
// existing tests; the body delegates to the CalibrationStrategy
// abstraction (see strategy.go).
func AutoCalibrateWithProfile(parentCtx context.Context, cfg config.AppConfig, out io.Writer, calculatorRegistry map[string]fibonacci.Calculator, profilePath string) (updated config.AppConfig, ok bool) {
	// Check if calculators are available before attempting calibration.
	// CompleteStrategy needs "fast"; without it we cannot escalate even
	// if FastStrategy returns low confidence, so refuse early.
	if calculatorRegistry["fast"] == nil {
		return cfg, false
	}

	// Try to load existing profile first.
	profile, loaded := LoadOrCreateProfile(profilePath)
	profileFresh := loaded && profile.IsValid()
	maxAge := profileMaxAgeFromEnv()
	profileStale := profileFresh && profile.IsStale(maxAge)

	if profileFresh && !profileStale {
		return applyCachedProfile(cfg, profile, out), true
	}

	stratOpts := StrategyOptions{
		BaseConfig:         cfg,
		CalculatorRegistry: calculatorRegistry,
		Out:                out,
	}

	if profileStale {
		// R1.3: profile hardware-compatible but expired. Skip the fast
		// tier and re-measure with the authoritative complete sweep so
		// the on-disk values reflect today's runtime characteristics.
		age := time.Since(profile.CalibratedAt).Round(time.Second)
		fmt.Fprintf(out, "%sProfile stale (age=%s), re-calibrating%s\n",
			ui.ColorYellow(), age, ui.ColorReset())
		return runStrategy(parentCtx, NewCompleteStrategy(), stratOpts, profilePath, true)
	}

	// Standard escalation: fast first; if it does not clear the
	// confidence bar, fall through to complete.
	if updated, ok := tryFastThenEscalate(parentCtx, stratOpts, profilePath); ok {
		return updated, true
	}
	return runStrategy(parentCtx, NewCompleteStrategy(), stratOpts, profilePath, true)
}

// applyCachedProfile copies the cached threshold values onto cfg and
// emits the legacy "Using cached calibration" log line. It does not
// touch disk: the caller has already loaded and validated the profile.
func applyCachedProfile(cfg config.AppConfig, profile *CalibrationProfile, out io.Writer) config.AppConfig {
	updated := cfg
	updated.Threshold = profile.OptimalParallelThreshold
	updated.FFTThreshold = profile.OptimalFFTThreshold
	updated.StrassenThreshold = profile.OptimalStrassenThreshold

	fmt.Fprintf(out, "%sUsing cached calibration%s: parallelism=%s%d%s bits, FFT=%s%d%s bits, Strassen=%s%d%s bits\n",
		ui.ColorGreen(), ui.ColorReset(),
		ui.ColorYellow(), updated.Threshold, ui.ColorReset(),
		ui.ColorYellow(), updated.FFTThreshold, ui.ColorReset(),
		ui.ColorYellow(), updated.StrassenThreshold, ui.ColorReset())
	return updated
}

// tryFastThenEscalate runs FastStrategy and returns (updated, true)
// only if the strategy produced a profile with confidence high enough
// to skip the complete sweep. Any error or low-confidence result
// returns (_, false) so the caller can escalate.
func tryFastThenEscalate(parentCtx context.Context, stratOpts StrategyOptions, profilePath string) (config.AppConfig, bool) {
	fast := NewFastStrategy()
	profile, conf, err := fast.Calibrate(parentCtx, stratOpts)
	if err != nil || conf < EscalationConfidenceThreshold {
		return stratOpts.BaseConfig, false
	}
	return finalizeStrategyResult(stratOpts.BaseConfig, profile, profilePath, stratOpts.Out, false), true
}

// runStrategy invokes a CalibrationStrategy and converts its
// (profile, confidence, error) tuple into the (config, ok) shape the
// public AutoCalibrateWithProfile contract requires. When announce is
// true it also emits the legacy "Auto-calibration: ..." summary line
// — historically only the complete path printed it.
func runStrategy(parentCtx context.Context, strategy CalibrationStrategy, stratOpts StrategyOptions, profilePath string, announce bool) (config.AppConfig, bool) {
	profile, _, err := strategy.Calibrate(parentCtx, stratOpts)
	if err != nil || profile == nil {
		return stratOpts.BaseConfig, false
	}
	return finalizeStrategyResult(stratOpts.BaseConfig, profile, profilePath, stratOpts.Out, announce), true
}

// finalizeStrategyResult merges a strategy's *CalibrationProfile back
// into the running config, persists it for future startups, and
// optionally prints the human-facing summary. Persistence failures are
// non-fatal (saveCalibrationProfile already logs a warning).
func finalizeStrategyResult(cfg config.AppConfig, profile *CalibrationProfile, profilePath string, out io.Writer, announce bool) config.AppConfig {
	updated := cfg
	updated.Threshold = profile.OptimalParallelThreshold
	updated.FFTThreshold = profile.OptimalFFTThreshold
	updated.StrassenThreshold = profile.OptimalStrassenThreshold

	saveCalibrationProfile(updated, profilePath, out, profile.Confidence)
	if announce {
		printCalibrationOutput(updated, out)
	}
	return updated
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
	profile.CalibrationN = CalibrationN
	profile.Confidence = confidence

	if err := profile.SaveProfile(profilePath); err != nil {
		fmt.Fprintf(out, "%sWarning: could not save calibration profile: %v%s\n",
			ui.ColorYellow(), err, ui.ColorReset())
	}
}
