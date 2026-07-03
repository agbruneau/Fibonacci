package calibration

import (
	"context"
	"errors"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/FibGo/internal/config"
	"github.com/agbruneau/FibGo/internal/fibonacci"
	"github.com/agbruneau/FibGo/internal/progress"
)

// MockFailingCalculator simulates calculation errors
type MockFailingCalculator struct{}

func (m *MockFailingCalculator) Name() string { return "fail" }
func (m *MockFailingCalculator) Calculate(ctx context.Context, progressChan chan<- progress.ProgressUpdate, calcIndex int, n uint64, opts fibonacci.Options) (*big.Int, error) {
	return nil, errors.New("simulated error")
}

// MockBlockingCalculator simulates a long running calculation. Entered
// closes EnteredChan (if set) so callers can synchronize on the calculation
// having actually started, instead of guessing with a fixed sleep.
type MockBlockingCalculator struct {
	BlockChan   chan struct{}
	EnteredChan chan struct{}
}

func (m *MockBlockingCalculator) Name() string { return "block" }
func (m *MockBlockingCalculator) Calculate(ctx context.Context, progressChan chan<- progress.ProgressUpdate, calcIndex int, n uint64, opts fibonacci.Options) (*big.Int, error) {
	if m.EnteredChan != nil {
		close(m.EnteredChan)
	}
	if m.BlockChan != nil {
		<-m.BlockChan
	}
	return big.NewInt(1), nil
}

// TestRunCalibrationWithOptions_LoadProfile_ShowsEffectivePath is the
// FIB-09 red test: when a custom (non-default) ProfilePath is used, the
// path printed to the user must be that effective path, not
// GetDefaultProfilePath() — otherwise a custom profile is loaded from one
// location but the user is told it came from another.
func TestRunCalibrationWithOptions_LoadProfile_ShowsEffectivePath(t *testing.T) {
	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "custom_profile.json")
	if profilePath == GetDefaultProfilePath() {
		t.Fatal("test precondition violated: custom path collides with default path")
	}

	profile := NewProfile()
	profile.OptimalParallelThreshold = 1234
	if err := profile.SaveProfile(profilePath); err != nil {
		t.Fatalf("Failed to save profile: %v", err)
	}

	opts := CalibrationOptions{
		ProfilePath: profilePath,
		LoadProfile: true,
	}

	var outBuf strings.Builder
	registry := map[string]fibonacci.Calculator{}
	ctx := context.Background()
	exitCode := RunCalibrationWithOptions(ctx, &outBuf, registry, opts, noopProgressDisplay, noopColorProvider{})

	if exitCode != 0 {
		t.Fatalf("Expected exit code 0, got %d", exitCode)
	}
	output := outBuf.String()
	if !strings.Contains(output, profilePath) {
		t.Errorf("Output should show the effective profile path %q, got: %s", profilePath, output)
	}
}

func TestRunCalibrationWithOptions_LoadProfile(t *testing.T) {
	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "profile.json")

	// Create a dummy profile
	profile := NewProfile()
	profile.OptimalParallelThreshold = 1234
	if err := profile.SaveProfile(profilePath); err != nil {
		t.Fatalf("Failed to save profile: %v", err)
	}

	opts := CalibrationOptions{
		ProfilePath: profilePath,
		LoadProfile: true,
	}

	// Registry not needed if loading profile succeeds early
	registry := map[string]fibonacci.Calculator{}
	ctx := context.Background()
	exitCode := RunCalibrationWithOptions(ctx, io.Discard, registry, opts, noopProgressDisplay, noopColorProvider{})

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}
}

// TestRunCalibrationWithOptions_SaveProfile_ShowsEffectivePath is the
// FIB-09 red test for the persistCalibrationProfile save-path message: a
// full calibration run with a custom, non-default ProfilePath must report
// that same path in the "profile saved to" message.
func TestRunCalibrationWithOptions_SaveProfile_ShowsEffectivePath(t *testing.T) {
	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "custom_save_profile.json")
	if profilePath == GetDefaultProfilePath() {
		t.Fatal("test precondition violated: custom path collides with default path")
	}

	registry := map[string]fibonacci.Calculator{
		"fast": &MockCalculator{name: "fast"},
	}

	opts := CalibrationOptions{
		ProfilePath: profilePath,
		LoadProfile: false,
		SaveProfile: true,
	}

	var outBuf strings.Builder
	ctx := context.Background()
	exitCode := RunCalibrationWithOptions(ctx, &outBuf, registry, opts, noopProgressDisplay, noopColorProvider{})

	if exitCode != 0 {
		t.Fatalf("Expected exit code 0, got %d", exitCode)
	}
	output := outBuf.String()
	if !strings.Contains(output, profilePath) {
		t.Errorf("Output should show the effective save path %q, got: %s", profilePath, output)
	}
}

func TestRunCalibrationWithOptions_CalculationError(t *testing.T) {
	registry := map[string]fibonacci.Calculator{
		"fast": &MockFailingCalculator{},
	}

	opts := CalibrationOptions{
		LoadProfile: false,
		SaveProfile: false,
	}

	ctx := context.Background()
	exitCode := RunCalibrationWithOptions(ctx, io.Discard, registry, opts, noopProgressDisplay, noopColorProvider{})

	// Should fail because calculation failed
	if exitCode == 0 {
		t.Error("Expected non-zero exit code due to calculation error")
	}
}

func TestRunCalibrationWithOptions_ContextCanceled(t *testing.T) {
	blockChan := make(chan struct{})
	enteredChan := make(chan struct{})
	registry := map[string]fibonacci.Calculator{
		"fast": &MockBlockingCalculator{BlockChan: blockChan, EnteredChan: enteredChan},
	}

	opts := CalibrationOptions{
		LoadProfile: false,
		SaveProfile: false,
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel only once the mock has actually entered Calculate, instead of
	// guessing with a fixed sleep (TEST-03).
	go func() {
		<-enteredChan
		cancel()
		close(blockChan) // Unblock to allow clean exit if needed
	}()

	exitCode := RunCalibrationWithOptions(ctx, io.Discard, registry, opts, noopProgressDisplay, noopColorProvider{})

	// Should fail due to cancellation
	if exitCode == 0 {
		t.Error("Expected non-zero exit code due to cancellation")
	}
}

func TestAutoCalibrateWithProfile_FallbackAndMissingMatrix(t *testing.T) {
	// 1. Setup: Missing profile (force fallback), Missing Matrix calculator
	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "profile_missing.json")

	registry := map[string]fibonacci.Calculator{
		"fast": &MockCalculator{name: "fast"}, // From calibration_test.go
		// "matrix" is missing
	}

	cfg := config.AppConfig{
		Timeout: 1 * time.Second,
	}

	ctx := context.Background()
	// Should fallback to full calibration (mocked via fast calc)
	updatedCfg, ok := AutoCalibrateWithProfile(ctx, cfg, io.Discard, registry, profilePath)

	if !ok {
		t.Error("AutoCalibrateWithProfile should succeed even with missing matrix calc")
	}
	if updatedCfg.Threshold == 0 {
		t.Error("Threshold should have been updated")
	}

	// Verify profile was saved
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		t.Error("Profile should have been saved")
	}
}

func TestProfile_String(t *testing.T) {
	p := NewProfile()
	p.OptimalParallelThreshold = 100
	p.OptimalFFTThreshold = 200
	p.OptimalStrassenThreshold = 300
	p.CalibrationN = 1000
	p.CalibrationTime = "1s"

	str := p.String()
	expectedSubstrings := []string{
		"Parallel: 100 bits",
		"FFT: 200 bits",
		"Strassen: 300 bits",
		"CalibrationProfile{",
	}

	for _, s := range expectedSubstrings {
		if !strings.Contains(str, s) {
			t.Errorf("String() missing %q, got: %s", s, str)
		}
	}
}

func TestProfile_SaveProfile_Error(t *testing.T) {
	p := NewProfile()
	// Try to save to a directory that doesn't exist/invalid path
	err := p.SaveProfile("\x00invalid/path/profile.json")
	if err == nil {
		t.Error("Expected error saving to invalid path")
	}
}
