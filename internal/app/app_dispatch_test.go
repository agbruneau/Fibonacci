package app

import (
	"bytes"
	"encoding/json"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/agbruneau/FibGo/internal/calibration"
	"github.com/agbruneau/FibGo/internal/config"
)

// Note: the TUI dispatch branch (runDispatch -> runTUI) is intentionally not
// tested. tui.Run hard-wires bubbletea to the process terminal with no
// injection point, and an empirical attempt showed that on a Windows console
// the program quits cleanly (alt-screen restored on context timeout) but
// tea.Program.Run never returns: the console input reader teardown
// (cancelreader on CONIN$) blocks until a real key event arrives. That makes
// any in-process test hang non-deterministically depending on TTY presence.

// TestNewLoadsCachedCalibrationProfile verifies that New applies the
// thresholds from a valid cached calibration profile instead of the adaptive
// defaults. The sentinel values are deliberately odd numbers the adaptive
// estimator never produces, so a regression of the LoadCachedCalibration
// branch in New fails loudly.
func TestNewLoadsCachedCalibrationProfile(t *testing.T) {
	t.Parallel()
	profilePath := t.TempDir() + "/profile.json"

	wordSize := 32 << (^uint(0) >> 63)
	profile := calibration.CalibrationProfile{
		ProfileVersion:           calibration.CurrentProfileVersion,
		NumCPU:                   runtime.NumCPU(),
		GOARCH:                   runtime.GOARCH,
		WordSize:                 wordSize,
		CPUHeuristicKey:          config.CurrentHardwareHeuristicKey(),
		OptimalParallelThreshold: 8191,
		OptimalFFTThreshold:      612345,
		OptimalStrassenThreshold: 4093,
		CalibratedAt:             time.Now(),
	}
	data, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("Failed to marshal profile: %v", err)
	}
	if writeErr := os.WriteFile(profilePath, data, 0o644); writeErr != nil {
		t.Fatalf("Failed to write profile: %v", writeErr)
	}

	var errBuf bytes.Buffer
	args := []string{"fibcalc", "-n", "100", "-calibration-profile", profilePath}

	app, err := New(args, &errBuf)

	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}
	if app.Config.Threshold != 8191 {
		t.Errorf("Threshold = %d, want 8191 (from cached profile)", app.Config.Threshold)
	}
	if app.Config.FFTThreshold != 612345 {
		t.Errorf("FFTThreshold = %d, want 612345 (from cached profile)", app.Config.FFTThreshold)
	}
	if app.Config.StrassenThreshold != 4093 {
		t.Errorf("StrassenThreshold = %d, want 4093 (from cached profile)", app.Config.StrassenThreshold)
	}
}

// TestNewExplicitFlagsBeatCachedProfile pins the precedence documented in
// internal/config/thresholds.go. Its predecessor,
// TestNewCachedProfileOverridesExplicitFlags, asserted the opposite and said
// so: "If the repo owner ever decides flags should win, this test is the thing
// that must change with the behavior." The 2026-09 audit (M-03) decided it.
//
// A cached profile is replayed unprompted on every start, so overwriting an
// explicit --threshold discarded the user's choice with nothing on screen to
// say so. The profile now fills only what the user left to the tool.
func TestNewExplicitFlagsBeatCachedProfile(t *testing.T) {
	t.Parallel()
	profilePath := t.TempDir() + "/profile.json"

	wordSize := 32 << (^uint(0) >> 63)
	profile := calibration.CalibrationProfile{
		ProfileVersion:           calibration.CurrentProfileVersion,
		NumCPU:                   runtime.NumCPU(),
		GOARCH:                   runtime.GOARCH,
		WordSize:                 wordSize,
		CPUHeuristicKey:          config.CurrentHardwareHeuristicKey(),
		OptimalParallelThreshold: 8191,
		OptimalFFTThreshold:      612345,
		OptimalStrassenThreshold: 4093,
		CalibratedAt:             time.Now(),
	}
	data, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("Failed to marshal profile: %v", err)
	}
	if writeErr := os.WriteFile(profilePath, data, 0o644); writeErr != nil {
		t.Fatalf("Failed to write profile: %v", writeErr)
	}

	var errBuf bytes.Buffer
	args := []string{
		"fibcalc", "-n", "100",
		"-calibration-profile", profilePath,
		"-threshold", "111",
		"-fft-threshold", "222222",
		"-strassen-threshold", "333",
	}

	app, err := New(args, &errBuf)
	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}
	if app.Config.Threshold != 111 {
		t.Errorf("Threshold = %d, want 111 (-threshold beats the cached 8191)", app.Config.Threshold)
	}
	if app.Config.FFTThreshold != 222222 {
		t.Errorf("FFTThreshold = %d, want 222222 (-fft-threshold beats the cached 612345)", app.Config.FFTThreshold)
	}
	if app.Config.StrassenThreshold != 333 {
		t.Errorf("StrassenThreshold = %d, want 333 (-strassen-threshold beats the cached 4093)", app.Config.StrassenThreshold)
	}
}

// TestNewFallsBackToAdaptiveThresholds verifies the other half of the New
// startup branch: when no cached calibration profile can be loaded, the
// auto (0) thresholds from flag parsing must be replaced by the adaptive
// hardware estimates. A nonexistent profile path is forced explicitly
// because the developer machine may have a valid profile at the default
// location, which would otherwise mask a regression of this fallback.
// ApplyAdaptiveThresholds is deterministic for a given machine, so exact
// equality is asserted.
func TestNewFallsBackToAdaptiveThresholds(t *testing.T) {
	t.Parallel()
	missingProfile := t.TempDir() + "/does-not-exist.json"

	var errBuf bytes.Buffer
	args := []string{"fibcalc", "-n", "100", "-calibration-profile", missingProfile}

	app, err := New(args, &errBuf)

	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}

	want := config.ApplyAdaptiveThresholds(config.AppConfig{})
	if app.Config.Threshold != want.Threshold {
		t.Errorf("Threshold = %d, want adaptive estimate %d", app.Config.Threshold, want.Threshold)
	}
	if app.Config.FFTThreshold != want.FFTThreshold {
		t.Errorf("FFTThreshold = %d, want adaptive estimate %d", app.Config.FFTThreshold, want.FFTThreshold)
	}
	if app.Config.StrassenThreshold != want.StrassenThreshold {
		t.Errorf("StrassenThreshold = %d, want adaptive estimate %d", app.Config.StrassenThreshold, want.StrassenThreshold)
	}
}
