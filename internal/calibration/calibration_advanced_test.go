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

	"github.com/agbru/fibcalc/internal/config"
	"github.com/agbru/fibcalc/internal/fibonacci"
	"github.com/agbru/fibcalc/internal/progress"
)

// MockFailingCalculator simulates calculation errors
type MockFailingCalculator struct{}

func (m *MockFailingCalculator) Name() string { return "fail" }
func (m *MockFailingCalculator) Calculate(ctx context.Context, progressChan chan<- progress.ProgressUpdate, calcIndex int, n uint64, opts fibonacci.Options) (*big.Int, error) {
	return nil, errors.New("simulated error")
}

// MockBlockingCalculator simulates a long running calculation
type MockBlockingCalculator struct {
	BlockChan chan struct{}
}

func (m *MockBlockingCalculator) Name() string { return "block" }
func (m *MockBlockingCalculator) Calculate(ctx context.Context, progressChan chan<- progress.ProgressUpdate, calcIndex int, n uint64, opts fibonacci.Options) (*big.Int, error) {
	if m.BlockChan != nil {
		<-m.BlockChan
	}
	return big.NewInt(1), nil
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
	registry := map[string]fibonacci.Calculator{
		"fast": &MockBlockingCalculator{BlockChan: blockChan},
	}

	opts := CalibrationOptions{
		LoadProfile: false,
		SaveProfile: false,
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context shortly after start
	go func() {
		time.Sleep(10 * time.Millisecond)
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
	err := p.SaveProfile("/invalid/path/profile.json")
	if err == nil {
		t.Error("Expected error saving to invalid path")
	}
}

