// Package calibration provides hardware-adaptive performance calibration
// for the Fibonacci calculator.
//
// # Role
//
// The package measures the host machine (CPU, cache, scheduler) through
// micro-benchmarks and derives the crossover thresholds used by the
// calculator selectors — parallelism threshold, FFT threshold, Strassen
// threshold. Results can be persisted to a profile (JSON) and reloaded on
// subsequent runs to avoid re-measuring.
//
// # Invariants
//
//   - Calibration runs are bounded by a context.Context deadline; callers
//     are expected to pass a cancellable context so the user can abort.
//   - Profiles are versioned (profile.go: ProfileVersion vs
//     CurrentProfileVersion); a mismatch causes a silent re-calibration
//     rather than loading stale thresholds.
//   - Micro-benchmarks (microbench.go) average N timed iterations per
//     configuration to reduce noise; the configurations themselves run
//     concurrently, bounded by a runtime.NumCPU semaphore.
//
// # Entry points
//
//   - RunCalibration / RunCalibrationWithOptions — full interactive flow.
//   - AutoCalibrate / AutoCalibrateWithProfile — fast startup path, loads
//     cached profile when valid or re-calibrates briefly otherwise.
//   - LoadCachedCalibration — read-only fast path, no measurement.
//
// # Example
//
//	cfg, ok := calibration.AutoCalibrate(ctx, cfg, os.Stderr, registry)
//	if !ok {
//	    // fall back to static defaults
//	}
//
// See docs/CALIBRATION.md for the full calibration protocol.
package calibration
