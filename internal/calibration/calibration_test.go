// Package calibration_test exercises internal/calibration's exported API
// only. It is the black-box counterpart to calibration_internal_test.go
// and runner_test.go, which stay package calibration because they call
// unexported helpers directly (see PLAN.md §3.3a).
package calibration_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/FibGo/internal/calibration"
	"github.com/agbruneau/FibGo/internal/config"
	"github.com/agbruneau/FibGo/internal/fibonacci"
)

func TestAutoCalibrate(t *testing.T) {
	t.Parallel()
	registry := map[string]fibonacci.Calculator{
		"fast":   calibration.NewMockCalculator("fast"),
		"matrix": calibration.NewMockCalculator("matrix"),
	}

	tmpProfile := t.TempDir() + "/profile.json"
	cfg := config.AppConfig{
		Timeout:            1 * time.Second, // Short timeout for test
		CalibrationProfile: tmpProfile,      // Avoid touching the real default path
	}

	updatedCfg, success := calibration.AutoCalibrate(context.Background(), cfg, io.Discard, registry)
	if !success {
		t.Error("AutoCalibrate should succeed with mock calculator")
	}
	if updatedCfg.Threshold == 0 {
		t.Error("Threshold should be set")
	}
}

func TestRunCalibration(t *testing.T) {
	t.Parallel()
	registry := map[string]fibonacci.Calculator{"fast": calibration.NewMockCalculator("fast")}

	exitCode := calibration.RunCalibration(context.Background(), io.Discard, registry, calibration.NoopProgressDisplay, calibration.NoopColorProvider{})
	if exitCode != 0 {
		t.Errorf("RunCalibration failed with code %d", exitCode)
	}
}

func TestRunCalibrationMissingFast(t *testing.T) {
	t.Parallel()
	registry := map[string]fibonacci.Calculator{} // Empty

	exitCode := calibration.RunCalibration(context.Background(), io.Discard, registry, calibration.NoopProgressDisplay, calibration.NoopColorProvider{})
	if exitCode == 0 {
		t.Error("RunCalibration should fail if 'fast' calculator is missing")
	}
}

func TestLoadCachedCalibration(t *testing.T) {
	t.Parallel()

	t.Run("Nonexistent profile", func(t *testing.T) {
		t.Parallel()
		_, success := calibration.LoadCachedCalibration(config.AppConfig{}, "nonexistent.json")
		if success {
			t.Error("Should return false for nonexistent profile")
		}
	})

	t.Run("Valid profile loaded", func(t *testing.T) {
		t.Parallel()
		profilePath := t.TempDir() + "/profile.json"

		profile := calibration.NewProfile()
		profile.OptimalParallelThreshold = 4096
		profile.OptimalFFTThreshold = 1000000
		profile.OptimalStrassenThreshold = 256
		if err := profile.SaveProfile(profilePath); err != nil {
			t.Fatalf("Failed to save profile: %v", err)
		}

		cfg := config.AppConfig{Threshold: 2048, FFTThreshold: 500000, StrassenThreshold: 512}

		updated, success := calibration.LoadCachedCalibration(cfg, profilePath)
		if !success {
			t.Error("Should return true for valid profile")
		}
		if updated.Threshold != 4096 {
			t.Errorf("Threshold = %d, want 4096", updated.Threshold)
		}
		if updated.FFTThreshold != 1000000 {
			t.Errorf("FFTThreshold = %d, want 1000000", updated.FFTThreshold)
		}
		if updated.StrassenThreshold != 256 {
			t.Errorf("StrassenThreshold = %d, want 256", updated.StrassenThreshold)
		}
	})

	t.Run("Invalid profile", func(t *testing.T) {
		t.Parallel()
		profilePath := t.TempDir() + "/invalid.json"
		if err := os.WriteFile(profilePath, []byte("{}"), 0o600); err != nil {
			t.Fatalf("Failed to write invalid profile: %v", err)
		}

		_, success := calibration.LoadCachedCalibration(config.AppConfig{}, profilePath)
		if success {
			t.Error("Should return false for invalid profile")
		}
	})
}

func TestAutoCalibrateWithProfile(t *testing.T) {
	t.Parallel()

	t.Run("Load existing profile", func(t *testing.T) {
		t.Parallel()
		profilePath := t.TempDir() + "/profile.json"

		profile := calibration.NewProfile()
		profile.OptimalParallelThreshold = 4096
		profile.OptimalFFTThreshold = 1000000
		profile.OptimalStrassenThreshold = 256
		if err := profile.SaveProfile(profilePath); err != nil {
			t.Fatalf("Failed to save profile: %v", err)
		}

		registry := map[string]fibonacci.Calculator{"fast": calibration.NewMockCalculator("fast")}
		cfg := config.AppConfig{Timeout: 1 * time.Second}

		var outBuf bytes.Buffer
		updated, ok := calibration.AutoCalibrateWithProfile(context.Background(), cfg, &outBuf, registry, profilePath)

		if !ok {
			t.Error("AutoCalibrateWithProfile should succeed with existing profile")
		}
		if updated.Threshold != 4096 {
			t.Errorf("Threshold = %d, want 4096", updated.Threshold)
		}
		if output := outBuf.String(); !strings.Contains(output, "Using cached calibration") {
			t.Errorf("Output should mention cached calibration. Got: %s", output)
		}
	})

	// SEC-01: the runtime auto-calibrate path must not trust a hardware-valid
	// but forged on-disk profile whose thresholds are out of range. IsValid()
	// checks hardware compatibility only, so a negative threshold would leak
	// straight into the running config via applyCachedProfile. One subtest
	// per forgeable threshold: a partial re-validation (e.g. parallel-only)
	// must fail on the two thresholds it stopped checking.
	forgeries := []struct {
		name  string
		forge func(p *calibration.CalibrationProfile)
	}{
		{"negative parallel threshold", func(p *calibration.CalibrationProfile) { p.OptimalParallelThreshold = -1 }},
		{"negative FFT threshold", func(p *calibration.CalibrationProfile) { p.OptimalFFTThreshold = -1 }},
		{"negative Strassen threshold", func(p *calibration.CalibrationProfile) { p.OptimalStrassenThreshold = -1 }},
	}
	for _, tc := range forgeries {
		t.Run("Forged fresh profile is not applied: "+tc.name, func(t *testing.T) {
			t.Parallel()
			profilePath := t.TempDir() + "/forged.json"

			// NewProfile() stamps the current hardware fields, so IsValid()
			// passes; CalibratedAt is now, so it is fresh (not stale). Only
			// one threshold is forged per subtest.
			profile := calibration.NewProfile()
			profile.OptimalParallelThreshold = 4096
			profile.OptimalFFTThreshold = 600000
			profile.OptimalStrassenThreshold = 4096
			tc.forge(profile)
			if err := profile.SaveProfile(profilePath); err != nil {
				t.Fatalf("Failed to save forged profile: %v", err)
			}

			registry := map[string]fibonacci.Calculator{"fast": calibration.NewMockCalculator("fast")}
			cfg := config.AppConfig{Timeout: 5 * time.Second}

			var outBuf bytes.Buffer
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			updated, _ := calibration.AutoCalibrateWithProfile(ctx, cfg, &outBuf, registry, profilePath)

			if updated.Threshold < 0 || updated.FFTThreshold < 0 || updated.StrassenThreshold < 0 {
				t.Fatalf("forged negative threshold leaked via auto-calibrate: parallel=%d fft=%d strassen=%d",
					updated.Threshold, updated.FFTThreshold, updated.StrassenThreshold)
			}
			if strings.Contains(outBuf.String(), "Using cached calibration") {
				t.Errorf("forged profile must not be applied as a cached calibration; output=%q", outBuf.String())
			}
		})
	}

	t.Run("Quick calibration fallback", func(t *testing.T) {
		t.Parallel()
		profilePath := t.TempDir() + "/profile.json"
		registry := map[string]fibonacci.Calculator{"fast": calibration.NewMockCalculator("fast")}
		cfg := config.AppConfig{Timeout: 5 * time.Second}

		var outBuf bytes.Buffer
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		updated, ok := calibration.AutoCalibrateWithProfile(ctx, cfg, &outBuf, registry, profilePath)

		// Quick calibration may succeed or timeout.
		if ok {
			if updated.Threshold == 0 {
				t.Error("Threshold should be set after quick calibration")
			}
			if output := outBuf.String(); !strings.Contains(output, "calibration") {
				t.Errorf("Output should mention calibration. Got: %s", output)
			}
		}
	})

	t.Run("Full calibration fallback", func(t *testing.T) {
		t.Parallel()
		profilePath := t.TempDir() + "/profile.json"
		registry := map[string]fibonacci.Calculator{
			"fast":   calibration.NewMockCalculator("fast"),
			"matrix": calibration.NewMockCalculator("matrix"),
		}
		cfg := config.AppConfig{Timeout: 5 * time.Second}

		var outBuf bytes.Buffer
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		updated, ok := calibration.AutoCalibrateWithProfile(ctx, cfg, &outBuf, registry, profilePath)

		// Full calibration may succeed or timeout.
		if ok && updated.Threshold == 0 {
			t.Error("Threshold should be set after full calibration")
		}
	})

	t.Run("No fast calculator", func(t *testing.T) {
		t.Parallel()
		profilePath := t.TempDir() + "/profile.json"
		registry := map[string]fibonacci.Calculator{} // Empty
		cfg := config.AppConfig{Timeout: 1 * time.Second, Threshold: 0}

		var outBuf bytes.Buffer
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		updated, ok := calibration.AutoCalibrateWithProfile(ctx, cfg, &outBuf, registry, profilePath)

		// QuickCalibrate might succeed even without a "fast" calculator (it
		// uses bigfft directly), so check that the config is unchanged only
		// when calibration actually failed.
		if !ok {
			if updated.Threshold != cfg.Threshold {
				t.Errorf("Config should remain unchanged on failure. Threshold = %d, want %d",
					updated.Threshold, cfg.Threshold)
			}
		} else if updated.Threshold == 0 {
			t.Error("If calibration succeeded, threshold should be set")
		}
	})
}

func TestAutoCalibrateWithProfile_StaleProfileTriggersRecalibration(t *testing.T) {
	// Not parallel: t.Setenv pins this test to a single goroutine (Go's
	// testing package forbids combining t.Setenv with t.Parallel).
	profilePath := t.TempDir() + "/profile.json"

	// Build a profile that is hardware-valid (NewProfile populates current
	// hardware fields) but predates the freshness window by two weeks.
	profile := calibration.NewProfile()
	profile.OptimalParallelThreshold = 4096
	profile.OptimalFFTThreshold = 1000000
	profile.OptimalStrassenThreshold = 256
	profile.Confidence = 1.0
	profile.CalibratedAt = time.Now().Add(-14 * 24 * time.Hour)

	// Sanity: IsValid stays true (hardware match), IsStale must report true
	// against any reasonable max-age (default = 7 days).
	if !profile.IsValid() {
		t.Fatalf("precondition failed: profile should be valid for current hardware")
	}
	if !profile.IsStale(calibration.DefaultProfileMaxAge) {
		t.Fatalf("precondition failed: 14-day-old profile should be stale vs default max age")
	}

	if err := profile.SaveProfile(profilePath); err != nil {
		t.Fatalf("Failed to save profile: %v", err)
	}

	// Force the default max-age (7d) regardless of caller environment.
	t.Setenv(calibration.ProfileMaxAgeEnv, "")

	registry := map[string]fibonacci.Calculator{
		"fast":   calibration.NewMockCalculator("fast"),
		"matrix": calibration.NewMockCalculator("matrix"),
	}
	cfg := config.AppConfig{Timeout: 5 * time.Second}

	var outBuf bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	updated, ok := calibration.AutoCalibrateWithProfile(ctx, cfg, &outBuf, registry, profilePath)
	if !ok {
		t.Fatalf("AutoCalibrateWithProfile should succeed; output=%q", outBuf.String())
	}

	output := outBuf.String()
	if !strings.Contains(output, "Profile stale") {
		t.Errorf("Output should mention stale profile re-calibration. Got: %s", output)
	}
	if !strings.Contains(output, "re-calibrating") {
		t.Errorf("Output should mention re-calibrating. Got: %s", output)
	}
	if strings.Contains(output, "Using cached calibration") {
		t.Errorf("Output should NOT advertise cached calibration when profile is stale. Got: %s", output)
	}
	if strings.Contains(output, "Quick calibration") {
		t.Errorf("Stale profile must skip the quick path and run full calibration. Got: %s", output)
	}

	// The full calibration path must overwrite the on-disk profile with a
	// fresh CalibratedAt timestamp; verify the persisted profile is no
	// longer stale.
	reloaded, reloadedOK := calibration.LoadOrCreateProfile(profilePath)
	if !reloadedOK {
		t.Fatalf("Reloaded profile should be valid after re-calibration")
	}
	if reloaded.IsStale(calibration.DefaultProfileMaxAge) {
		t.Errorf("Profile persisted after re-calibration should not be stale; CalibratedAt=%s", reloaded.CalibratedAt)
	}
	if updated.Threshold == 0 {
		t.Errorf("Re-calibrated config should have a non-zero parallel threshold")
	}
}

// TestRunCalibrationWithOptions_LoadProfile_ShowsEffectivePath is the
// FIB-09 red test: when a custom (non-default) ProfilePath is used, the
// path printed to the user must be that effective path, not
// GetDefaultProfilePath() -- otherwise a custom profile is loaded from one
// location but the user is told it came from another.
func TestRunCalibrationWithOptions_LoadProfile_ShowsEffectivePath(t *testing.T) {
	t.Parallel()
	profilePath := filepath.Join(t.TempDir(), "custom_profile.json")
	if profilePath == calibration.GetDefaultProfilePath() {
		t.Fatal("test precondition violated: custom path collides with default path")
	}

	profile := calibration.NewProfile()
	profile.OptimalParallelThreshold = 1234
	if err := profile.SaveProfile(profilePath); err != nil {
		t.Fatalf("Failed to save profile: %v", err)
	}

	opts := calibration.CalibrationOptions{ProfilePath: profilePath, LoadProfile: true}

	var outBuf strings.Builder
	exitCode := calibration.RunCalibrationWithOptions(context.Background(), &outBuf, map[string]fibonacci.Calculator{}, opts, calibration.NoopProgressDisplay, calibration.NoopColorProvider{})

	if exitCode != 0 {
		t.Fatalf("Expected exit code 0, got %d", exitCode)
	}
	if output := outBuf.String(); !strings.Contains(output, profilePath) {
		t.Errorf("Output should show the effective profile path %q, got: %s", profilePath, output)
	}
}

func TestRunCalibrationWithOptions_LoadProfile(t *testing.T) {
	t.Parallel()
	profilePath := filepath.Join(t.TempDir(), "profile.json")

	profile := calibration.NewProfile()
	profile.OptimalParallelThreshold = 1234
	if err := profile.SaveProfile(profilePath); err != nil {
		t.Fatalf("Failed to save profile: %v", err)
	}

	opts := calibration.CalibrationOptions{ProfilePath: profilePath, LoadProfile: true}

	// Registry not needed if loading profile succeeds early.
	exitCode := calibration.RunCalibrationWithOptions(context.Background(), io.Discard, map[string]fibonacci.Calculator{}, opts, calibration.NoopProgressDisplay, calibration.NoopColorProvider{})
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}
}

// TestRunCalibrationWithOptions_SaveProfile_ShowsEffectivePath is the
// FIB-09 red test for the persistCalibrationProfile save-path message: a
// full calibration run with a custom, non-default ProfilePath must report
// that same path in the "profile saved to" message.
func TestRunCalibrationWithOptions_SaveProfile_ShowsEffectivePath(t *testing.T) {
	t.Parallel()
	profilePath := filepath.Join(t.TempDir(), "custom_save_profile.json")
	if profilePath == calibration.GetDefaultProfilePath() {
		t.Fatal("test precondition violated: custom path collides with default path")
	}

	registry := map[string]fibonacci.Calculator{"fast": calibration.NewMockCalculator("fast")}
	opts := calibration.CalibrationOptions{ProfilePath: profilePath, LoadProfile: false, SaveProfile: true}

	var outBuf strings.Builder
	exitCode := calibration.RunCalibrationWithOptions(context.Background(), &outBuf, registry, opts, calibration.NoopProgressDisplay, calibration.NoopColorProvider{})

	if exitCode != 0 {
		t.Fatalf("Expected exit code 0, got %d", exitCode)
	}
	if output := outBuf.String(); !strings.Contains(output, profilePath) {
		t.Errorf("Output should show the effective save path %q, got: %s", profilePath, output)
	}
}

func TestRunCalibrationWithOptions_CalculationError(t *testing.T) {
	t.Parallel()
	registry := map[string]fibonacci.Calculator{"fast": &calibration.MockFailingCalculator{}}
	opts := calibration.CalibrationOptions{LoadProfile: false, SaveProfile: false}

	exitCode := calibration.RunCalibrationWithOptions(context.Background(), io.Discard, registry, opts, calibration.NoopProgressDisplay, calibration.NoopColorProvider{})
	if exitCode == 0 {
		t.Error("Expected non-zero exit code due to calculation error")
	}
}

func TestRunCalibrationWithOptions_ContextCanceled(t *testing.T) {
	t.Parallel()
	blockChan := make(chan struct{})
	enteredChan := make(chan struct{})
	registry := map[string]fibonacci.Calculator{
		"fast": &calibration.MockBlockingCalculator{BlockChan: blockChan, EnteredChan: enteredChan},
	}
	opts := calibration.CalibrationOptions{LoadProfile: false, SaveProfile: false}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel only once the mock has actually entered Calculate, instead of
	// guessing with a fixed sleep (TEST-03).
	go func() {
		<-enteredChan
		cancel()
		close(blockChan) // Unblock to allow clean exit if needed.
	}()

	exitCode := calibration.RunCalibrationWithOptions(ctx, io.Discard, registry, opts, calibration.NoopProgressDisplay, calibration.NoopColorProvider{})
	if exitCode == 0 {
		t.Error("Expected non-zero exit code due to cancellation")
	}
}

func TestAutoCalibrateWithProfile_FallbackAndMissingMatrix(t *testing.T) {
	t.Parallel()
	// Setup: missing profile (force fallback), missing "matrix" calculator.
	profilePath := filepath.Join(t.TempDir(), "profile_missing.json")
	registry := map[string]fibonacci.Calculator{"fast": calibration.NewMockCalculator("fast")}
	cfg := config.AppConfig{Timeout: 1 * time.Second}

	// Should fall back to full calibration (mocked via the "fast" calculator).
	updatedCfg, ok := calibration.AutoCalibrateWithProfile(context.Background(), cfg, io.Discard, registry, profilePath)
	if !ok {
		t.Error("AutoCalibrateWithProfile should succeed even with missing matrix calc")
	}
	if updatedCfg.Threshold == 0 {
		t.Error("Threshold should have been updated")
	}
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		t.Error("Profile should have been saved")
	}
}
