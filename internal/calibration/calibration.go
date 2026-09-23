package calibration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/agbruneau/FibGo/internal/apperrors"
	"github.com/agbruneau/FibGo/internal/config"
	"github.com/agbruneau/FibGo/internal/fibonacci"
	"github.com/agbruneau/FibGo/internal/progress"
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

// profileMaxAge resolves the freshness window for a cached profile: the
// caller's setting when positive, DefaultProfileMaxAge otherwise.
//
// It takes the value from AppConfig rather than reading FIBCALC_PROFILE_MAX_AGE
// itself (audit CFG-02). The variable is still honored — internal/config's
// override table maps it, alongside --profile-max-age — but it is now parsed
// where every other setting is parsed, reported by --help, and covered by the
// integrity test that keeps the flag set and the table in sync.
func profileMaxAge(cfg config.AppConfig) time.Duration {
	if cfg.ProfileMaxAge > 0 {
		return cfg.ProfileMaxAge
	}
	return DefaultProfileMaxAge
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
//   - profilePath: destination of the saved profile; empty selects the
//     default path (~/.fibcalc_calibration.json). Honors the CLI's
//     --calibration-profile flag (audit Fable5 CAL-01).
//
// Returns:
//   - int: The exit code (0 for success, non-zero for errors).
func RunCalibration(ctx context.Context, out io.Writer, rep Reporter, calculatorRegistry map[string]fibonacci.Calculator, profilePath string, progressDisplay ProgressDisplayFunc) int {
	return RunCalibrationWithOptions(ctx, out, rep, calculatorRegistry, CalibrationOptions{
		SaveProfile: true,
		LoadProfile: false, // Full calibration should run fresh
		ProfilePath: profilePath,
	}, progressDisplay)
}

// RunCalibrationWithOptions executes calibration with the specified options.
//
// P2-05: the body delegates to three focused helpers
// (configureHardwareDetection, runPassSequence, persistCalibrationProfile)
// so each concern stays under the package's funlen / cyclo thresholds and
// can be exercised independently by tests.
func RunCalibrationWithOptions(ctx context.Context, out io.Writer, rep Reporter, calculatorRegistry map[string]fibonacci.Calculator, opts CalibrationOptions, progressDisplay ProgressDisplayFunc) int {
	rep.Notice("--- Calibration Mode: Finding the Optimal Parallelism Threshold ---")

	// Try to load existing profile if requested
	if opts.LoadProfile && tryUseCachedCalibrationProfile(opts.ProfilePath, rep) {
		return apperrors.ExitSuccess
	}

	calculator, thresholdsToTest, code := configureHardwareDetection(rep, calculatorRegistry)
	if calculator == nil {
		return code
	}

	calibrationStart := time.Now()
	bestThreshold, results, code := runPassSequence(ctx, out, rep, calculator, thresholdsToTest, progressDisplay)
	if code != apperrors.ExitSuccess {
		return code
	}
	calibrationDuration := time.Since(calibrationStart)

	rep.Summary(results, bestThreshold)

	recommendation := fmt.Sprintf("--threshold %d", bestThreshold)
	if bestThreshold == config.ThresholdDisabled {
		// -1 is the genuine sequential baseline (FIB-02). It is also a valid
		// CLI value (audit H-02), so recommend the flag AND say what it
		// means: the user gets both the outcome and a way to reproduce it.
		recommendation = fmt.Sprintf("--threshold %d (sequential, no parallelism)", config.ThresholdDisabled)
	}
	rep.Notice("Recommendation for this machine: %s", recommendation)

	if opts.SaveProfile {
		persistCalibrationProfile(rep, opts.ProfilePath, bestThreshold, calibrationDuration)
	}

	return apperrors.ExitSuccess
}

// tryUseCachedCalibrationProfile attempts to short-circuit calibration by
// loading an existing valid profile. Returns true if a valid profile was found
// and reported, in which case the caller returns ExitSuccess without running a
// fresh calibration; false when the caller must calibrate.
func tryUseCachedCalibrationProfile(profilePath string, rep Reporter) bool {
	profile, loaded := LoadOrCreateProfile(profilePath)
	if !loaded || !profile.IsValid() {
		return false
	}
	rep.Notice("Loaded existing calibration profile from %s", effectiveProfilePath(profilePath))
	rep.Notice("Profile: %s", profile.String())
	rep.Notice("Using cached calibration: --threshold %d", profile.OptimalParallelThreshold)
	return true
}

// configureHardwareDetection resolves the calculator used for calibration
// and the hardware-adaptive threshold list. On failure (missing "fast"
// calculator) it returns (nil, nil, ExitErrorGeneric); the caller must
// return that code. On success it returns the calculator, the ordered
// threshold list to test, and ExitSuccess.
func configureHardwareDetection(rep Reporter, calculatorRegistry map[string]fibonacci.Calculator) (calculator fibonacci.Calculator, thresholdsToTest []int, code int) {
	calculator = calculatorRegistry["fast"]
	if calculator == nil {
		rep.Error("the 'fast' algorithm is required for calibration but was not found")
		return nil, nil, apperrors.ExitErrorGeneric
	}
	thresholdsToTest = GenerateParallelThresholds()
	rep.Notice("Using adaptive thresholds for %d CPU cores", runtime.NumCPU())
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
//     apperrors.ExitCodeFor's code on an unrecoverable calculation error.
func runPassSequence(ctx context.Context, out io.Writer, rep Reporter, calculator fibonacci.Calculator, thresholdsToTest []int, progressDisplay ProgressDisplayFunc) (bestThreshold int, results []PassResult, code int) {
	results = make([]PassResult, 0, len(thresholdsToTest))
	bestDuration := time.Duration(1<<63 - 1)

	// One progress consumer PER PASS (audit L-07). A single consumer shared by
	// every threshold saw progress restart from 0 at each new calculation, so
	// the smoothed rate it maintains — and the ETA derived from it — described
	// a curve no individual run followed, and the closing "100%" line printed
	// once for the whole sweep instead of once per pass.
	runPass := func(threshold int) (time.Duration, error) {
		var wg sync.WaitGroup
		progressChan := make(chan progress.ProgressUpdate, 5)
		wg.Add(1)
		go progressDisplay(&wg, progressChan, 1, out)

		startTime := time.Now()
		_, err := calculator.Calculate(ctx, progressChan, 0, CalibrationN, fibonacci.Options{ParallelThreshold: threshold})
		duration := time.Since(startTime)

		close(progressChan)
		wg.Wait()
		return duration, err
	}

	for _, threshold := range thresholdsToTest {
		// Reachable from the binary since audit CON-01: --calibrate now runs
		// under the signal root and the --timeout budget. Distinguish the two
		// so the exit code matches the cause, as HandleCalculationError does
		// for a failure raised inside a pass.
		if err := ctx.Err(); err != nil {
			rep.Warning("Calibration interrupted.")
			if errors.Is(err, context.DeadlineExceeded) {
				return 0, results, apperrors.ExitErrorTimeout
			}
			return 0, results, apperrors.ExitErrorCanceled
		}

		duration, err := runPass(threshold)

		if err != nil {
			rep.Error("Failure (%v)", err)
			results = append(results, PassResult{threshold, 0, err})
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				// The pass failure has already been printed above; all that is
				// needed from the error is the exit code (audit API-04).
				return 0, results, apperrors.ExitCodeFor(err)
			}
			continue
		}

		results = append(results, PassResult{threshold, duration, nil})
		if duration < bestDuration {
			bestDuration, bestThreshold = duration, threshold
		}
	}

	if bestDuration == time.Duration(1<<63-1) {
		rep.Error("Calibration failed: no valid results obtained.")
		return 0, results, apperrors.ExitErrorGeneric
	}
	return bestThreshold, results, apperrors.ExitSuccess
}

// persistCalibrationProfile materializes a full-confidence profile for
// the winning parallel threshold and writes it to profilePath. Warnings
// are printed to out but save errors are non-fatal (the caller's exit
// code reflects calibration success, not persistence success).
func persistCalibrationProfile(rep Reporter, profilePath string, bestThreshold int, calibrationDuration time.Duration) {
	profile := NewProfile()
	profile.OptimalParallelThreshold = bestThreshold
	profile.OptimalFFTThreshold = config.EstimateOptimalFFTThreshold()
	profile.OptimalStrassenThreshold = config.EstimateOptimalStrassenThreshold()
	profile.CalibrationN = CalibrationN
	profile.CalibrationTime = calibrationDuration.String()
	profile.Confidence = 1.0

	if err := profile.SaveProfile(profilePath); err != nil {
		rep.Warning("failed to save profile: %v", err)
		return
	}
	rep.Notice("Calibration profile saved to %s", effectiveProfilePath(profilePath))
}

// effectiveProfilePath resolves the path a profile load/save call actually
// used: SaveProfile/LoadOrCreateProfile substitute GetDefaultProfilePath()
// when profilePath is empty, so display code must apply the same
// resolution to avoid claiming the default path was used when a custom one
// was in effect (FIB-09).
func effectiveProfilePath(profilePath string) string {
	if profilePath == "" {
		return GetDefaultProfilePath()
	}
	return profilePath
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
func AutoCalibrate(parentCtx context.Context, cfg config.AppConfig, rep Reporter, calculatorRegistry map[string]fibonacci.Calculator) (updated config.AppConfig, ok bool) {
	return AutoCalibrateWithProfile(parentCtx, cfg, rep, calculatorRegistry, cfg.CalibrationProfile)
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
func AutoCalibrateWithProfile(parentCtx context.Context, cfg config.AppConfig, rep Reporter, calculatorRegistry map[string]fibonacci.Calculator, profilePath string) (updated config.AppConfig, ok bool) {
	// Check if calculators are available before attempting calibration.
	// CompleteStrategy needs "fast"; without it we cannot escalate even
	// if FastStrategy returns low confidence, so refuse early.
	if calculatorRegistry["fast"] == nil {
		return cfg, false
	}

	// Try to load existing profile first.
	profile, loaded := LoadOrCreateProfile(profilePath)
	profileFresh := loaded && profile.IsValid()
	maxAge := profileMaxAge(cfg)
	profileStale := profileFresh && profile.IsStale(maxAge)

	if profileFresh && !profileStale {
		// SEC-01: IsValid() checks only hardware compatibility, never
		// threshold ranges, so a forged on-disk profile can carry an
		// out-of-range value. Re-validate the three thresholds the profile
		// controls before trusting them; on failure fall through to a fresh
		// calibration instead of leaking the forged value into the running
		// config. The bounds mirror config.Validate exactly, ThresholdDisabled
		// included: rejecting -1 here (audit H-02) threw away the calibration's
		// own sequential/no-FFT result and silently replaced it with a default.
		if profile.OptimalParallelThreshold >= config.ThresholdDisabled &&
			profile.OptimalFFTThreshold >= config.ThresholdDisabled &&
			profile.OptimalStrassenThreshold >= 0 {
			return applyCachedProfile(cfg, profile, rep), true
		}
		rep.Warning("Cached calibration profile has invalid thresholds, re-calibrating")
	}

	stratOpts := StrategyOptions{
		BaseConfig:         cfg,
		CalculatorRegistry: calculatorRegistry,
		Reporter:           rep,
	}

	if profileStale {
		// R1.3: profile hardware-compatible but expired. Skip the fast
		// tier and re-measure with the authoritative complete sweep so
		// the on-disk values reflect today's runtime characteristics.
		age := time.Since(profile.CalibratedAt).Round(time.Second)
		rep.Warning("Profile stale (age=%s), re-calibrating", age)
		return runStrategy(parentCtx, NewCompleteStrategy(), stratOpts, profilePath, true)
	}

	// Standard escalation: fast first; if it does not clear the
	// confidence bar, fall through to complete.
	if updated, ok := tryFastThenEscalate(parentCtx, stratOpts, profilePath); ok {
		return updated, true
	}
	return runStrategy(parentCtx, NewCompleteStrategy(), stratOpts, profilePath, true)
}

// applyProfileThresholds copies a profile's thresholds onto cfg, leaving alone
// any threshold the user pinned on the command line or through the
// environment (audit M-03).
//
// A cached profile that overwrote all three unconditionally would silently
// discard an explicit --threshold / --fft-threshold / --strassen-threshold on
// any machine that had ever run --calibrate. A calibration profile is the
// tool's own guess at what the user did not specify; it does not outrank what
// the user did specify.
//
// It applies to the two paths that REPLAY a stored profile
// (LoadCachedCalibration and applyCachedProfile). It deliberately does not
// apply to a fresh --calibrate / --auto-calibrate sweep, where the user asked
// for a measurement, the measured value is announced on screen, and it is that
// measured value — not the pin — that gets persisted.
func applyProfileThresholds(cfg config.AppConfig, profile *CalibrationProfile) config.AppConfig {
	updated := cfg
	if !cfg.ThresholdExplicit {
		updated.Threshold = profile.OptimalParallelThreshold
	}
	if !cfg.FFTThresholdExplicit {
		updated.FFTThreshold = profile.OptimalFFTThreshold
	}
	if !cfg.StrassenThresholdExplicit {
		updated.StrassenThreshold = profile.OptimalStrassenThreshold
	}
	return updated
}

// applyCachedProfile copies the cached threshold values onto cfg and
// emits the legacy "Using cached calibration" log line. It does not
// touch disk: the caller has already loaded and validated the profile.
func applyCachedProfile(cfg config.AppConfig, profile *CalibrationProfile, rep Reporter) config.AppConfig {
	updated := applyProfileThresholds(cfg, profile)

	// Report the EFFECTIVE values, which are what the calculation will use:
	// a threshold the user pinned shows their value, not the profile's.
	rep.Notice("Using cached calibration: parallelism=%d bits, FFT=%d bits, Strassen=%d bits",
		updated.Threshold, updated.FFTThreshold, updated.StrassenThreshold)
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
	return finalizeStrategyResult(stratOpts.BaseConfig, profile, profilePath, stratOpts.Reporter, false), true
}

// runStrategy invokes a CalibrationStrategy and converts its
// (profile, confidence, error) tuple into the (config, ok) shape the
// public AutoCalibrateWithProfile contract requires. When announce is
// true it also emits the legacy "Auto-calibration: ..." summary line
// — historically only the complete path printed it.
func runStrategy(parentCtx context.Context, strategy CalibrationStrategy, stratOpts StrategyOptions, profilePath string, announce bool) (config.AppConfig, bool) {
	profile, _, err := strategy.Calibrate(parentCtx, stratOpts)
	if err != nil || profile == nil {
		// ERR-05: runStrategy is the terminal fallback of the escalation
		// chain — the user explicitly asked for auto-calibration, so a
		// silent return would leave them believing it happened.
		if err != nil {
			stratOpts.Reporter.Warning("auto-calibration failed (%v), using default thresholds", err)
		} else {
			stratOpts.Reporter.Warning("auto-calibration failed (no usable profile), using default thresholds")
		}
		return stratOpts.BaseConfig, false
	}
	return finalizeStrategyResult(stratOpts.BaseConfig, profile, profilePath, stratOpts.Reporter, announce), true
}

// finalizeStrategyResult merges a strategy's *CalibrationProfile back
// into the running config, persists it for future startups, and
// optionally prints the human-facing summary. Persistence failures are
// non-fatal (saveCalibrationProfile already logs a warning).
func finalizeStrategyResult(cfg config.AppConfig, profile *CalibrationProfile, profilePath string, rep Reporter, announce bool) config.AppConfig {
	updated := cfg
	updated.Threshold = profile.OptimalParallelThreshold
	updated.FFTThreshold = profile.OptimalFFTThreshold
	updated.StrassenThreshold = profile.OptimalStrassenThreshold

	saveCalibrationProfile(updated, profilePath, rep, profile.Confidence)
	if announce {
		printCalibrationOutput(updated, rep)
	}
	return updated
}

// LoadCachedCalibration attempts to load a cached calibration profile and
// apply it to the configuration. Returns the updated config and true if
// a valid cached profile was found.
//
// Thresholds the user pinned explicitly are preserved (audit M-03): this runs
// unprompted on every start, so overwriting them would discard the user's
// choice with nothing on screen to say so.
func LoadCachedCalibration(cfg config.AppConfig, profilePath string) (updated config.AppConfig, ok bool) {
	profile, loaded := LoadOrCreateProfile(profilePath)
	if !loaded || !profile.IsValid() {
		return cfg, false
	}

	return applyProfileThresholds(cfg, profile), true
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
	if bestParDur == noTiming && bestFFTDur == noTiming {
		return cfg, false
	}

	updated = cfg
	if bestParDur != noTiming {
		updated.Threshold = bestPar
	}
	if bestFFTDur != noTiming {
		updated.FFTThreshold = bestFFT
	}
	if bestStrassenDur != noTiming {
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
func saveCalibrationProfile(cfg config.AppConfig, profilePath string, rep Reporter, confidence float64) {
	profile := NewProfile()
	profile.OptimalParallelThreshold = cfg.Threshold
	profile.OptimalFFTThreshold = cfg.FFTThreshold
	profile.OptimalStrassenThreshold = cfg.StrassenThreshold
	profile.CalibrationN = CalibrationN
	profile.Confidence = confidence

	if err := profile.SaveProfile(profilePath); err != nil {
		rep.Warning("could not save calibration profile: %v", err)
	}
}
