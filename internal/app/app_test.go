package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/FibGo/internal/calibration"
	"github.com/agbruneau/FibGo/internal/cli"
	"github.com/agbruneau/FibGo/internal/config"
	apperrors "github.com/agbruneau/FibGo/internal/errors"
	"github.com/agbruneau/FibGo/internal/fibonacci"
	"github.com/agbruneau/FibGo/internal/orchestration"
	"github.com/agbruneau/FibGo/internal/testutil"
)

// Helper to create a test factory with mocked calculator
func createMockFactory(result *big.Int, err error) *fibonacci.TestFactory {
	mockCalc := &fibonacci.MockCalculator{
		Result: result,
		Err:    err,
	}
	// Pre-populate with common algorithms to allow tests to "Create" them
	calcs := map[string]fibonacci.Calculator{
		"fast":   mockCalc,
		"matrix": mockCalc,
		"fft":    mockCalc,
	}
	return fibonacci.NewTestFactory(calcs)
}

// TestNew tests the New function for creating Application instances.
func TestNew(t *testing.T) {
	t.Parallel()
	t.Run("Valid args create application", func(t *testing.T) {
		t.Parallel()
		var errBuf bytes.Buffer
		args := []string{"fibcalc", "-n", "100"}

		app, err := New(args, &errBuf)

		if err != nil {
			t.Fatalf("New() returned unexpected error: %v", err)
		}
		if app == nil {
			t.Fatal("New() returned nil application")
		}
		if app.Config.N != 100 {
			t.Errorf("Expected N=100, got N=%d", app.Config.N)
		}
		if app.Factory == nil {
			t.Error("Factory should not be nil")
		}
	})

	t.Run("Invalid args return error", func(t *testing.T) {
		t.Parallel()
		var errBuf bytes.Buffer
		args := []string{"fibcalc", "-invalid-flag"}

		app, err := New(args, &errBuf)

		if err == nil {
			t.Error("New() should return error for invalid args")
		}
		if app != nil {
			t.Error("New() should return nil application on error")
		}
	})

	t.Run("Help flag returns error", func(t *testing.T) {
		t.Parallel()
		var errBuf bytes.Buffer
		args := []string{"fibcalc", "-h"}

		_, err := New(args, &errBuf)

		if err == nil {
			t.Error("New() should return error for help flag")
		}
		if !IsHelpError(err) {
			t.Error("Error should be a help error")
		}
	})

	t.Run("Empty args slice handled correctly", func(t *testing.T) {
		t.Parallel()
		var errBuf bytes.Buffer
		args := []string{}

		app, err := New(args, &errBuf)

		// Empty args should still work - it will use default program name
		// and empty command args, which should parse to default config
		if err != nil {
			t.Errorf("New() should handle empty args without error, got: %v", err)
		}
		if app == nil {
			t.Fatal("New() should return application even with empty args")
		}
		// Should use default program name
		if app.Config.N != 100000000 {
			t.Errorf("Expected default N=100000000, got N=%d", app.Config.N)
		}
	})

	// SEC-01: a calibration profile is untrusted disk input. IsValid only
	// checks hardware compatibility, never threshold ranges, so a forged
	// profile with a negative threshold must be rejected after being
	// applied to cfg, falling back to config.ApplyAdaptiveThresholds
	// instead of propagating the invalid value.
	t.Run("Forged profile with negative threshold is ignored", func(t *testing.T) {
		t.Parallel()
		var errBuf bytes.Buffer

		tmpDir := t.TempDir()
		profilePath := tmpDir + "/forged.json"

		wordSize := 32 << (^uint(0) >> 63)
		forged := calibration.CalibrationProfile{
			ProfileVersion:           calibration.CurrentProfileVersion,
			NumCPU:                   runtime.NumCPU(),
			GOARCH:                   runtime.GOARCH,
			WordSize:                 wordSize,
			CPUHeuristicKey:          config.CurrentHardwareHeuristicKey(),
			OptimalParallelThreshold: -1,
			OptimalFFTThreshold:      600000,
			OptimalStrassenThreshold: 4096,
			CalibratedAt:             time.Now(),
		}
		profileData, err := json.Marshal(forged)
		if err != nil {
			t.Fatalf("Failed to marshal forged profile: %v", err)
		}
		if err := os.WriteFile(profilePath, profileData, 0o644); err != nil {
			t.Fatalf("Failed to write forged profile: %v", err)
		}

		args := []string{"fibcalc", "-n", "100", "-calibration-profile", profilePath}

		app, err := New(args, &errBuf)
		if err != nil {
			t.Fatalf("New() returned unexpected error: %v", err)
		}

		if app.Config.Threshold < 0 {
			t.Fatalf("forged negative threshold leaked into cfg: Threshold=%d", app.Config.Threshold)
		}

		want := config.ApplyAdaptiveThresholds(config.AppConfig{N: 100})
		if app.Config.Threshold != want.Threshold {
			t.Errorf("Threshold = %d, want fallback ApplyAdaptiveThresholds value %d", app.Config.Threshold, want.Threshold)
		}
	})
}

// TestApplicationRun tests the Application.Run method.
// Optimized to use MockCalculator via TestFactory.
func TestApplicationRun(t *testing.T) {
	t.Parallel()
	// Reusable factory for success cases
	successFactory := createMockFactory(big.NewInt(55), nil)

	t.Run("Simple execution with success", func(t *testing.T) {
		t.Parallel()
		var outBuf bytes.Buffer
		app := &Application{
			Config: config.AppConfig{
				N:            10,
				Algo:         "fast",
				Timeout:      1 * time.Minute,
				Threshold:    fibonacci.DefaultParallelThreshold,
				FFTThreshold: 20000,
				Details:      true,
				ShowValue:    true,
			},
			Factory:   successFactory,
			ErrWriter: &bytes.Buffer{},
		}

		exitCode := app.Run(context.Background(), &outBuf).Code()

		if exitCode != apperrors.ExitSuccess {
			t.Errorf("Expected exit code %d, got %d", apperrors.ExitSuccess, exitCode)
		}
		output := testutil.StripAnsiCodes(outBuf.String())
		if !strings.Contains(output, "F(10) = 55") {
			t.Errorf("Output should contain 'F(10) = 55'. Output:\n%s", output)
		}
	})

	t.Run("Parallel comparison with success", func(t *testing.T) {
		t.Parallel()
		var outBuf bytes.Buffer
		app := &Application{
			Config: config.AppConfig{
				N:            20,
				Algo:         "all",
				Timeout:      1 * time.Minute,
				Threshold:    fibonacci.DefaultParallelThreshold,
				FFTThreshold: 20000,
				Details:      true,
			},
			Factory:   successFactory,
			ErrWriter: &bytes.Buffer{},
		}

		exitCode := app.Run(context.Background(), &outBuf).Code()

		if exitCode != apperrors.ExitSuccess {
			t.Errorf("Expected exit code %d, got %d", apperrors.ExitSuccess, exitCode)
		}
		output := testutil.StripAnsiCodes(outBuf.String())
		if !strings.Contains(output, "Comparison Summary") {
			t.Errorf("Output should contain 'Comparison Summary'. Output:\n%s", output)
		}
		if !strings.Contains(output, "Global Status: Success") {
			t.Errorf("Output should contain 'Global Status: Success'. Output:\n%s", output)
		}
	})

	t.Run("Timeout failure", func(t *testing.T) {
		var outBuf bytes.Buffer

		// Mock blocking calculator to respect context timeout
		mockCalc := &fibonacci.MockCalculator{
			Fn: func(ctx context.Context, n uint64) (*big.Int, error) {
				<-ctx.Done()
				return nil, ctx.Err()
			},
		}
		factory := fibonacci.NewTestFactory(map[string]fibonacci.Calculator{"fast": mockCalc})

		app := &Application{
			Config: config.AppConfig{
				N:       100_000_000,
				Algo:    "fast",
				Timeout: 1 * time.Millisecond,
			},
			Factory:   factory,
			ErrWriter: &bytes.Buffer{},
		}

		exitCode := app.Run(context.Background(), &outBuf).Code()

		if exitCode != apperrors.ExitErrorTimeout {
			t.Errorf("Expected exit code %d (timeout), got %d", apperrors.ExitErrorTimeout, exitCode)
		}
		output := testutil.StripAnsiCodes(outBuf.String())
		if !strings.Contains(output, "Timeout") {
			t.Errorf("Output should mention timeout. Output:\n%s", output)
		}
	})

	t.Run("Context cancellation", func(t *testing.T) {
		t.Parallel()
		var outBuf bytes.Buffer

		// Mock blocking calculator
		mockCalc := &fibonacci.MockCalculator{
			Fn: func(ctx context.Context, n uint64) (*big.Int, error) {
				<-ctx.Done()
				return nil, ctx.Err()
			},
		}
		factory := fibonacci.NewTestFactory(map[string]fibonacci.Calculator{"fast": mockCalc})

		app := &Application{
			Config: config.AppConfig{
				N:       100_000_000,
				Algo:    "fast",
				Timeout: 1 * time.Minute,
			},
			Factory:   factory,
			ErrWriter: &bytes.Buffer{},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		exitCode := app.Run(ctx, &outBuf).Code()

		if exitCode != apperrors.ExitErrorCanceled {
			t.Errorf("Expected exit code %d (canceled), got %d", apperrors.ExitErrorCanceled, exitCode)
		}
	})

	t.Run("Quiet mode", func(t *testing.T) {
		t.Parallel()
		var outBuf bytes.Buffer
		app := &Application{
			Config: config.AppConfig{
				N:       10,
				Algo:    "fast",
				Timeout: 1 * time.Minute,
				Quiet:   true,
			},
			Factory:   successFactory,
			ErrWriter: &bytes.Buffer{},
		}

		exitCode := app.Run(context.Background(), &outBuf).Code()

		if exitCode != apperrors.ExitSuccess {
			t.Errorf("Expected exit code %d, got %d", apperrors.ExitSuccess, exitCode)
		}
		output := outBuf.String()
		// Quiet mode should output just the result
		if !strings.Contains(output, "55") {
			t.Errorf("Quiet output should contain the result '55'. Output:\n%s", output)
		}
	})
}

// TestIsHelpError tests the IsHelpError function.
func TestIsHelpError(t *testing.T) {
	t.Parallel()
	var errBuf bytes.Buffer
	args := []string{"fibcalc", "-h"}

	_, err := New(args, &errBuf)

	if !IsHelpError(err) {
		t.Error("IsHelpError should return true for help flag error")
	}
}

// TestRunCompletion tests the completion script generation.
func TestRunCompletion(t *testing.T) {
	t.Parallel()
	var outBuf bytes.Buffer
	app := &Application{
		Config: config.AppConfig{
			Completion: "bash",
		},
		Factory:   fibonacci.GlobalFactory(),
		ErrWriter: &bytes.Buffer{},
	}

	exitCode := app.Run(context.Background(), &outBuf).Code()

	if exitCode != apperrors.ExitSuccess {
		t.Errorf("Expected exit code %d, got %d", apperrors.ExitSuccess, exitCode)
	}
	output := outBuf.String()
	if !strings.Contains(output, "bash") && !strings.Contains(output, "complete") {
		t.Errorf("Output should contain bash completion script. Got:\n%s", output)
	}
}

// TestRunCompletionInvalid tests invalid completion shell.
func TestRunCompletionInvalid(t *testing.T) {
	t.Parallel()
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	app := &Application{
		Config: config.AppConfig{
			Completion: "invalid-shell",
		},
		Factory:   fibonacci.GlobalFactory(),
		ErrWriter: &errBuf,
	}

	exitCode := app.Run(context.Background(), &outBuf).Code()

	if exitCode == apperrors.ExitSuccess {
		t.Error("Expected error exit code for invalid shell")
	}
}

// TestRunAutoCalibrationDisabled tests that auto-calibration doesn't run when disabled.
func TestRunAutoCalibrationDisabled(t *testing.T) {
	t.Parallel()
	var outBuf bytes.Buffer
	factory := createMockFactory(big.NewInt(55), nil)
	app := &Application{
		Config: config.AppConfig{
			N:             10,
			Algo:          "fast",
			Timeout:       1 * time.Minute,
			AutoCalibrate: false, // Disabled
		},
		Factory:   factory,
		ErrWriter: &bytes.Buffer{},
	}

	exitCode := app.Run(context.Background(), &outBuf).Code()

	if exitCode != apperrors.ExitSuccess {
		t.Errorf("Expected exit code %d, got %d", apperrors.ExitSuccess, exitCode)
	}
}

// TestMultipleAlgorithms tests running all algorithms.
func TestMultipleAlgorithms(t *testing.T) {
	t.Parallel()
	var outBuf bytes.Buffer
	factory := createMockFactory(big.NewInt(55), nil)
	app := &Application{
		Config: config.AppConfig{
			N:       15,
			Algo:    "all",
			Timeout: 1 * time.Minute,
			Details: true,
		},
		Factory:   factory,
		ErrWriter: &bytes.Buffer{},
	}

	exitCode := app.Run(context.Background(), &outBuf).Code()

	if exitCode != apperrors.ExitSuccess {
		t.Errorf("Expected exit code %d, got %d", apperrors.ExitSuccess, exitCode)
	}

	output := testutil.StripAnsiCodes(outBuf.String())
	if !strings.Contains(output, "Comparison Summary") {
		t.Errorf("Output should contain comparison summary. Got:\n%s", output)
	}
}

func TestApplyAdaptiveThresholds(t *testing.T) {
	t.Parallel()
	// Test case where defaults are present and should be replaced
	t.Run("ReplaceDefaults", func(t *testing.T) {
		t.Parallel()
		cfg := config.AppConfig{
			Threshold:         fibonacci.DefaultParallelThreshold,
			FFTThreshold:      fibonacci.DefaultFFTThreshold,
			StrassenThreshold: fibonacci.DefaultStrassenThreshold,
		}

		// Since we can't easily check internal calls without mocking,
		// we mainly check that it runs safely and returns a valid config.
		// The thresholds might remain default if the environment matches the defaults,
		// or change if it differs.
		newCfg := config.ApplyAdaptiveThresholds(cfg)
		_ = newCfg
	})

	// Test case where user overrides should be preserved
	t.Run("PreserveOverrides", func(t *testing.T) {
		t.Parallel()
		cfg := config.AppConfig{
			Threshold:         1234,
			FFTThreshold:      5678,
			StrassenThreshold: 9012,
		}

		newCfg := config.ApplyAdaptiveThresholds(cfg)

		if newCfg.Threshold != 1234 {
			t.Errorf("Threshold changed, want %d, got %d", 1234, newCfg.Threshold)
		}
		if newCfg.FFTThreshold != 5678 {
			t.Errorf("FFTThreshold changed, want %d, got %d", 5678, newCfg.FFTThreshold)
		}
		if newCfg.StrassenThreshold != 9012 {
			t.Errorf("StrassenThreshold changed, want %d, got %d", 9012, newCfg.StrassenThreshold)
		}
	})
}

func TestAnalyzeResultsWithOutputFile(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	outputPath := strings.ReplaceAll(tmpDir+"/result.txt", "\\", "/")

	app := &Application{
		Config: config.AppConfig{
			N:          10,
			OutputFile: outputPath,
		},
		Factory:   fibonacci.GlobalFactory(),
		ErrWriter: &bytes.Buffer{},
	}

	results := []orchestration.CalculationResult{
		{
			Name:     "fast",
			Result:   big.NewInt(55),
			Duration: 1 * time.Millisecond,
			Err:      nil,
		},
	}

	var outBuf bytes.Buffer
	outputCfg := cli.OutputConfig{
		OutputFile: outputPath,
	}

	exitCode := app.analyzeResultsWithOutput(results, outputCfg, &outBuf)
	if exitCode != apperrors.ExitSuccess {
		t.Errorf("Expected exit code %d, got %d", apperrors.ExitSuccess, exitCode)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("Output file %s was not created", outputPath)
	}
}

func TestAnalyzeResultsWithOutputVariety(t *testing.T) {
	t.Parallel()
	app := &Application{
		Config:    config.AppConfig{N: 10},
		ErrWriter: &bytes.Buffer{},
	}

	results := []orchestration.CalculationResult{
		{
			Name:     "fast",
			Result:   big.NewInt(55),
			Duration: 1 * time.Millisecond,
		},
	}

	t.Run("Quiet Mode", func(t *testing.T) {
		t.Parallel()
		var outBuf bytes.Buffer
		outputCfg := cli.OutputConfig{Quiet: true}
		exitCode := app.analyzeResultsWithOutput(results, outputCfg, &outBuf)
		if exitCode != apperrors.ExitSuccess {
			t.Errorf("Expected success, got %d", exitCode)
		}
		if !strings.Contains(outBuf.String(), "55") {
			t.Errorf("Expected output 55, got %s", outBuf.String())
		}
	})

	t.Run("No Success Results", func(t *testing.T) {
		t.Parallel()
		var outBuf bytes.Buffer
		resultsErr := []orchestration.CalculationResult{
			{Name: "err", Err: fmt.Errorf("some error")},
		}
		outputCfg := cli.OutputConfig{}
		exitCode := app.analyzeResultsWithOutput(resultsErr, outputCfg, &outBuf)
		if exitCode == apperrors.ExitSuccess {
			t.Error("Expected error exit code")
		}
	})
}

// TestRunCalibration tests the runCalibration method.
func TestRunCalibration(t *testing.T) {
	t.Parallel()

	t.Run("Calibration runs successfully", func(t *testing.T) {
		t.Parallel()
		var outBuf bytes.Buffer
		factory := createMockFactory(big.NewInt(55), nil)

		app := &Application{
			Config: config.AppConfig{
				Calibrate: true,
				Timeout:   5 * time.Second,
			},
			Factory:   factory,
			ErrWriter: &bytes.Buffer{},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		exitCode := app.runCalibration(ctx, &outBuf)

		// Calibration may succeed or timeout, both are valid
		if exitCode != apperrors.ExitSuccess &&
			exitCode != apperrors.ExitErrorTimeout &&
			exitCode != apperrors.ExitErrorCanceled {
			t.Errorf("Expected exit code %d, %d, or %d, got %d",
				apperrors.ExitSuccess, apperrors.ExitErrorTimeout,
				apperrors.ExitErrorCanceled, exitCode)
		}
	})

	t.Run("Calibration with context cancellation", func(t *testing.T) {
		t.Parallel()
		var outBuf bytes.Buffer
		factory := createMockFactory(big.NewInt(55), nil)

		app := &Application{
			Config: config.AppConfig{
				Calibrate: true,
				Timeout:   1 * time.Minute,
			},
			Factory:   factory,
			ErrWriter: &bytes.Buffer{},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		exitCode := app.runCalibration(ctx, &outBuf)

		if exitCode != apperrors.ExitErrorCanceled {
			t.Errorf("Expected exit code %d (canceled), got %d",
				apperrors.ExitErrorCanceled, exitCode)
		}
	})
}

// TestRunAutoCalibrationIfEnabled tests the runAutoCalibrationIfEnabled method.
func TestRunAutoCalibrationIfEnabled(t *testing.T) {
	t.Parallel()

	t.Run("Auto-calibration enabled and succeeds", func(t *testing.T) {
		t.Parallel()
		var outBuf bytes.Buffer
		factory := createMockFactory(big.NewInt(55), nil)

		// Create a temporary calibration profile with known values
		// This ensures predictable results since mock calculators don't produce
		// meaningful timing measurements for the micro-benchmark calibration
		tmpDir := t.TempDir()
		profilePath := tmpDir + "/calibration.json"

		// Create a valid profile that matches current hardware
		wordSize := 32 << (^uint(0) >> 63)
		profile := calibration.CalibrationProfile{
			ProfileVersion:           calibration.CurrentProfileVersion,
			NumCPU:                   runtime.NumCPU(),
			GOARCH:                   runtime.GOARCH,
			WordSize:                 wordSize,
			CPUHeuristicKey:          config.CurrentHardwareHeuristicKey(),
			OptimalParallelThreshold: 8192,
			OptimalFFTThreshold:      600000,
			OptimalStrassenThreshold: 4096,
			CalibratedAt:             time.Now(),
		}
		profileData, err := json.Marshal(profile)
		if err != nil {
			t.Fatalf("Failed to marshal profile: %v", err)
		}
		if err := os.WriteFile(profilePath, profileData, 0o644); err != nil {
			t.Fatalf("Failed to create test profile: %v", err)
		}

		originalCfg := config.AppConfig{
			AutoCalibrate:      true,
			Timeout:            5 * time.Second,
			Threshold:          4096,
			CalibrationProfile: profilePath,
		}

		app := &Application{
			Config:    originalCfg,
			Factory:   factory,
			ErrWriter: &bytes.Buffer{},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		updatedCfg := app.runAutoCalibrationIfEnabled(ctx, &outBuf)

		// Config should be updated with the cached profile values
		if updatedCfg.Threshold != 8192 {
			t.Errorf("Expected Threshold=8192 from cached profile, got %d", updatedCfg.Threshold)
		}
		if updatedCfg.FFTThreshold != 600000 {
			t.Errorf("Expected FFTThreshold=600000 from cached profile, got %d", updatedCfg.FFTThreshold)
		}
	})

	t.Run("Auto-calibration enabled but fails", func(t *testing.T) {
		t.Parallel()
		var outBuf bytes.Buffer
		// Use a factory with no calculators to force failure
		emptyFactory := fibonacci.NewTestFactory(map[string]fibonacci.Calculator{})

		// Use a temporary profile path to avoid loading existing profiles
		tmpProfile := t.TempDir() + "/profile.json"

		originalCfg := config.AppConfig{
			AutoCalibrate:      true,
			Timeout:            1 * time.Second,
			Threshold:          4096,
			CalibrationProfile: tmpProfile,
		}

		app := &Application{
			Config:    originalCfg,
			Factory:   emptyFactory,
			ErrWriter: &bytes.Buffer{},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		updatedCfg := app.runAutoCalibrationIfEnabled(ctx, &outBuf)

		// Config should remain unchanged if calibration fails
		if updatedCfg.Threshold != originalCfg.Threshold {
			t.Errorf("Threshold should remain %d when calibration fails, got %d",
				originalCfg.Threshold, updatedCfg.Threshold)
		}
	})

	t.Run("Auto-calibration disabled", func(t *testing.T) {
		t.Parallel()
		var outBuf bytes.Buffer
		factory := createMockFactory(big.NewInt(55), nil)

		originalCfg := config.AppConfig{
			AutoCalibrate: false,
			Threshold:     4096,
		}

		app := &Application{
			Config:    originalCfg,
			Factory:   factory,
			ErrWriter: &bytes.Buffer{},
		}

		updatedCfg := app.runAutoCalibrationIfEnabled(context.Background(), &outBuf)

		// Config should remain unchanged when auto-calibration is disabled
		if updatedCfg.Threshold != originalCfg.Threshold {
			t.Errorf("Threshold should remain %d when auto-calibration is disabled, got %d",
				originalCfg.Threshold, updatedCfg.Threshold)
		}
	})
}

// TestRunAllModes tests the Run method with all different modes.
func TestRunAllModes(t *testing.T) {
	t.Parallel()

	t.Run("Calibration mode", func(t *testing.T) {
		t.Parallel()
		var outBuf bytes.Buffer
		factory := createMockFactory(big.NewInt(55), nil)

		app := &Application{
			Config: config.AppConfig{
				Calibrate: true,
				Timeout:   2 * time.Second,
			},
			Factory:   factory,
			ErrWriter: &bytes.Buffer{},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		exitCode := app.Run(ctx, &outBuf).Code()

		if exitCode != apperrors.ExitSuccess &&
			exitCode != apperrors.ExitErrorTimeout &&
			exitCode != apperrors.ExitErrorCanceled {
			t.Errorf("Expected exit code %d, %d, or %d, got %d",
				apperrors.ExitSuccess, apperrors.ExitErrorTimeout,
				apperrors.ExitErrorCanceled, exitCode)
		}
	})
}

// TestWithFactory tests the WithFactory option for dependency injection.
func TestWithFactory(t *testing.T) {
	t.Parallel()

	t.Run("Custom factory is used", func(t *testing.T) {
		t.Parallel()
		var errBuf bytes.Buffer
		customFactory := createMockFactory(big.NewInt(42), nil)
		args := []string{"fibcalc", "-n", "10"}

		app, err := New(args, &errBuf, WithFactory(customFactory))

		if err != nil {
			t.Fatalf("New() returned unexpected error: %v", err)
		}
		if app.Factory != customFactory {
			t.Error("Expected custom factory to be set")
		}
	})

	t.Run("Default factory when WithFactory not provided", func(t *testing.T) {
		t.Parallel()
		var errBuf bytes.Buffer
		args := []string{"fibcalc", "-n", "10"}

		app, err := New(args, &errBuf)

		if err != nil {
			t.Fatalf("New() returned unexpected error: %v", err)
		}
		if app.Factory == nil {
			t.Error("Factory should be set to default when WithFactory not provided")
		}
	})
}

// TestApplyAdaptiveThresholdsZeroValues tests that zero-value thresholds
// trigger the adaptive estimation paths.
func TestApplyAdaptiveThresholdsZeroValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  config.AppConfig
	}{
		{
			name: "All zero thresholds are replaced",
			cfg: config.AppConfig{
				Threshold:         0,
				FFTThreshold:      0,
				StrassenThreshold: 0,
			},
		},
		{
			name: "Only Threshold zero",
			cfg: config.AppConfig{
				Threshold:         0,
				FFTThreshold:      999999,
				StrassenThreshold: 5555,
			},
		},
		{
			name: "Only FFTThreshold zero",
			cfg: config.AppConfig{
				Threshold:         1111,
				FFTThreshold:      0,
				StrassenThreshold: 2222,
			},
		},
		{
			name: "Only StrassenThreshold zero",
			cfg: config.AppConfig{
				Threshold:         3333,
				FFTThreshold:      4444,
				StrassenThreshold: 0,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := config.ApplyAdaptiveThresholds(tc.cfg)

			// Zero thresholds should be replaced with positive values
			if tc.cfg.Threshold == 0 && result.Threshold == 0 {
				t.Error("Expected Threshold to be set to a positive value when input is 0")
			}
			if tc.cfg.FFTThreshold == 0 && result.FFTThreshold == 0 {
				t.Error("Expected FFTThreshold to be set to a positive value when input is 0")
			}
			if tc.cfg.StrassenThreshold == 0 && result.StrassenThreshold == 0 {
				t.Error("Expected StrassenThreshold to be set to a positive value when input is 0")
			}

			// Non-zero thresholds should be preserved
			if tc.cfg.Threshold != 0 && result.Threshold != tc.cfg.Threshold {
				t.Errorf("Non-zero Threshold should be preserved: want %d, got %d",
					tc.cfg.Threshold, result.Threshold)
			}
			if tc.cfg.FFTThreshold != 0 && result.FFTThreshold != tc.cfg.FFTThreshold {
				t.Errorf("Non-zero FFTThreshold should be preserved: want %d, got %d",
					tc.cfg.FFTThreshold, result.FFTThreshold)
			}
			if tc.cfg.StrassenThreshold != 0 && result.StrassenThreshold != tc.cfg.StrassenThreshold {
				t.Errorf("Non-zero StrassenThreshold should be preserved: want %d, got %d",
					tc.cfg.StrassenThreshold, result.StrassenThreshold)
			}
		})
	}
}

// TestRunLastDigits tests the runLastDigits method for computing
// the last K decimal digits of F(N).
func TestRunLastDigits(t *testing.T) {
	t.Parallel()

	t.Run("Compute last 5 digits of F(100)", func(t *testing.T) {
		t.Parallel()
		var outBuf bytes.Buffer
		factory := createMockFactory(big.NewInt(55), nil)

		app := &Application{
			Config: config.AppConfig{
				N:          100,
				LastDigits: 5,
				Timeout:    1 * time.Minute,
			},
			Factory:   factory,
			ErrWriter: &bytes.Buffer{},
		}

		exitCode := app.runLastDigits(context.Background(), &outBuf)

		if exitCode != apperrors.ExitSuccess {
			t.Errorf("Expected exit code %d, got %d", apperrors.ExitSuccess, exitCode)
		}
		output := outBuf.String()
		// F(100) = 354224848179261915075, last 5 digits = 15075
		if !strings.Contains(output, "15075") {
			t.Errorf("Expected output to contain '15075'. Output:\n%s", output)
		}
	})

	t.Run("Quiet mode outputs only digits", func(t *testing.T) {
		t.Parallel()
		var outBuf bytes.Buffer
		factory := createMockFactory(big.NewInt(55), nil)

		app := &Application{
			Config: config.AppConfig{
				N:          100,
				LastDigits: 5,
				Timeout:    1 * time.Minute,
				Quiet:      true,
			},
			Factory:   factory,
			ErrWriter: &bytes.Buffer{},
		}

		exitCode := app.runLastDigits(context.Background(), &outBuf)

		if exitCode != apperrors.ExitSuccess {
			t.Errorf("Expected exit code %d, got %d", apperrors.ExitSuccess, exitCode)
		}
		output := strings.TrimSpace(outBuf.String())
		if output != "15075" {
			t.Errorf("Expected quiet output '15075', got '%s'", output)
		}
	})

	t.Run("Last digits with timeout", func(t *testing.T) {
		t.Parallel()
		var outBuf bytes.Buffer
		factory := createMockFactory(big.NewInt(55), nil)

		app := &Application{
			Config: config.AppConfig{
				N:          10,
				LastDigits: 3,
				Timeout:    1 * time.Minute,
			},
			Factory:   factory,
			ErrWriter: &bytes.Buffer{},
		}

		exitCode := app.runLastDigits(context.Background(), &outBuf)

		if exitCode != apperrors.ExitSuccess {
			t.Errorf("Expected exit code %d, got %d", apperrors.ExitSuccess, exitCode)
		}
		// F(10) = 55, last 3 digits = "055" (zero-padded)
		output := outBuf.String()
		if !strings.Contains(output, "055") {
			t.Errorf("Expected output to contain '055'. Output:\n%s", output)
		}
	})
}

// TestRunLastDigitsViaRun tests the last-digits mode dispatched through Run.
func TestRunLastDigitsViaRun(t *testing.T) {
	t.Parallel()
	var outBuf bytes.Buffer
	factory := createMockFactory(big.NewInt(55), nil)

	app := &Application{
		Config: config.AppConfig{
			N:          100,
			Algo:       "fast",
			LastDigits: 5,
			Timeout:    1 * time.Minute,
		},
		Factory:   factory,
		ErrWriter: &bytes.Buffer{},
	}

	exitCode := app.Run(context.Background(), &outBuf).Code()

	if exitCode != apperrors.ExitSuccess {
		t.Errorf("Expected exit code %d, got %d", apperrors.ExitSuccess, exitCode)
	}
	output := outBuf.String()
	if !strings.Contains(output, "15075") {
		t.Errorf("Expected output to contain '15075'. Output:\n%s", output)
	}
}

// TestRunCalculateMemoryLimit tests the memory limit validation paths
// in runCalculate.
func TestRunCalculateMemoryLimit(t *testing.T) {
	t.Parallel()

	t.Run("Invalid memory limit format", func(t *testing.T) {
		t.Parallel()
		var outBuf bytes.Buffer
		factory := createMockFactory(big.NewInt(55), nil)

		app := &Application{
			Config: config.AppConfig{
				N:           10,
				Algo:        "fast",
				Timeout:     1 * time.Minute,
				MemoryLimit: "not-a-number",
			},
			Factory:   factory,
			ErrWriter: &bytes.Buffer{},
		}

		exitCode := app.Run(context.Background(), &outBuf).Code()

		if exitCode != apperrors.ExitErrorConfig {
			t.Errorf("Expected exit code %d (config error), got %d",
				apperrors.ExitErrorConfig, exitCode)
		}
		output := outBuf.String()
		if !strings.Contains(output, "Invalid --memory-limit") {
			t.Errorf("Expected output to mention invalid memory limit. Output:\n%s", output)
		}
	})

	t.Run("Memory limit exceeded", func(t *testing.T) {
		t.Parallel()
		var outBuf bytes.Buffer
		factory := createMockFactory(big.NewInt(55), nil)

		// Use a very large N to ensure estimated memory exceeds a tiny limit
		app := &Application{
			Config: config.AppConfig{
				N:           1_000_000_000,
				Algo:        "fast",
				Timeout:     1 * time.Minute,
				MemoryLimit: "1K",
			},
			Factory:   factory,
			ErrWriter: &bytes.Buffer{},
		}

		exitCode := app.Run(context.Background(), &outBuf).Code()

		if exitCode != apperrors.ExitErrorConfig {
			t.Errorf("Expected exit code %d (config error), got %d",
				apperrors.ExitErrorConfig, exitCode)
		}
		output := outBuf.String()
		if !strings.Contains(output, "exceeds limit") {
			t.Errorf("Expected output to mention exceeding limit. Output:\n%s", output)
		}
		// Should suggest --last-digits
		if !strings.Contains(output, "last-digits") {
			t.Errorf("Expected output to suggest --last-digits. Output:\n%s", output)
		}
	})

	t.Run("Memory limit sufficient", func(t *testing.T) {
		t.Parallel()
		var outBuf bytes.Buffer
		factory := createMockFactory(big.NewInt(55), nil)

		app := &Application{
			Config: config.AppConfig{
				N:           10,
				Algo:        "fast",
				Timeout:     1 * time.Minute,
				MemoryLimit: "8G",
			},
			Factory:   factory,
			ErrWriter: &bytes.Buffer{},
		}

		exitCode := app.Run(context.Background(), &outBuf).Code()

		if exitCode != apperrors.ExitSuccess {
			t.Errorf("Expected exit code %d, got %d", apperrors.ExitSuccess, exitCode)
		}
		output := outBuf.String()
		if !strings.Contains(output, "Memory estimate") {
			t.Errorf("Expected output to show memory estimate. Output:\n%s", output)
		}
	})

	t.Run("Memory limit sufficient quiet mode", func(t *testing.T) {
		t.Parallel()
		var outBuf bytes.Buffer
		factory := createMockFactory(big.NewInt(55), nil)

		app := &Application{
			Config: config.AppConfig{
				N:           10,
				Algo:        "fast",
				Timeout:     1 * time.Minute,
				MemoryLimit: "8G",
				Quiet:       true,
			},
			Factory:   factory,
			ErrWriter: &bytes.Buffer{},
		}

		exitCode := app.Run(context.Background(), &outBuf).Code()

		if exitCode != apperrors.ExitSuccess {
			t.Errorf("Expected exit code %d, got %d", apperrors.ExitSuccess, exitCode)
		}
		// In quiet mode, the memory estimate line should not appear
		output := outBuf.String()
		if strings.Contains(output, "Memory estimate") {
			t.Errorf("Quiet mode should not show memory estimate. Output:\n%s", output)
		}
	})
}

// TestAnalyzeResultsQuietModeWithOutputFile tests quiet mode output
// with file saving in analyzeResultsWithOutput.
func TestAnalyzeResultsQuietModeWithOutputFile(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	outputPath := strings.ReplaceAll(tmpDir+"/quiet_result.txt", "\\", "/")

	app := &Application{
		Config: config.AppConfig{
			N:          10,
			OutputFile: outputPath,
		},
		ErrWriter: &bytes.Buffer{},
	}

	results := []orchestration.CalculationResult{
		{
			Name:     "fast",
			Result:   big.NewInt(55),
			Duration: 1 * time.Millisecond,
		},
	}

	var outBuf bytes.Buffer
	outputCfg := cli.OutputConfig{
		Quiet:      true,
		OutputFile: outputPath,
	}

	exitCode := app.analyzeResultsWithOutput(results, outputCfg, &outBuf)
	if exitCode != apperrors.ExitSuccess {
		t.Errorf("Expected exit code %d, got %d", apperrors.ExitSuccess, exitCode)
	}

	// Verify result was printed
	if !strings.Contains(outBuf.String(), "55") {
		t.Errorf("Expected quiet output to contain '55'. Got:\n%s", outBuf.String())
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("Output file %s was not created", outputPath)
	}
}

// TestSaveResultIfNeeded tests the saveResultIfNeeded helper.
func TestSaveResultIfNeeded(t *testing.T) {
	t.Parallel()

	t.Run("No output file does nothing", func(t *testing.T) {
		t.Parallel()
		app := &Application{
			Config:    config.AppConfig{N: 10},
			ErrWriter: &bytes.Buffer{},
		}
		res := &orchestration.CalculationResult{
			Name:     "fast",
			Result:   big.NewInt(55),
			Duration: 1 * time.Millisecond,
		}
		err := app.saveResultIfNeeded(res, cli.OutputConfig{})
		if err != nil {
			t.Errorf("Expected nil error for empty output file, got: %v", err)
		}
	})

	t.Run("Invalid output path returns error", func(t *testing.T) {
		t.Parallel()
		app := &Application{
			Config:    config.AppConfig{N: 10},
			ErrWriter: &bytes.Buffer{},
		}
		res := &orchestration.CalculationResult{
			Name:     "fast",
			Result:   big.NewInt(55),
			Duration: 1 * time.Millisecond,
		}
		// Use a path with a null byte which is invalid on all platforms
		cfg := cli.OutputConfig{OutputFile: "invalid\x00path/file.txt"}
		err := app.saveResultIfNeeded(res, cfg)
		if err == nil {
			t.Error("Expected error for invalid output path")
		}
	})
}

// TestFindBestResult tests the findBestResult helper function.
func TestFindBestResult(t *testing.T) {
	t.Parallel()

	t.Run("All errors returns nil", func(t *testing.T) {
		t.Parallel()
		results := []orchestration.CalculationResult{
			{Name: "a", Err: fmt.Errorf("error a")},
			{Name: "b", Err: fmt.Errorf("error b")},
		}
		best := findBestResult(results)
		if best != nil {
			t.Error("Expected nil for all-error results")
		}
	})

	t.Run("Selects fastest successful result", func(t *testing.T) {
		t.Parallel()
		results := []orchestration.CalculationResult{
			{Name: "slow", Result: big.NewInt(55), Duration: 100 * time.Millisecond},
			{Name: "fast", Result: big.NewInt(55), Duration: 10 * time.Millisecond},
			{Name: "err", Err: fmt.Errorf("failed")},
		}
		best := findBestResult(results)
		if best == nil {
			t.Fatal("Expected non-nil result")
		}
		if best.Name != "fast" {
			t.Errorf("Expected fastest result 'fast', got '%s'", best.Name)
		}
	})

	t.Run("Empty results returns nil", func(t *testing.T) {
		t.Parallel()
		best := findBestResult(nil)
		if best != nil {
			t.Error("Expected nil for nil results")
		}
	})
}

// TestNewWithCustomFactory tests creating an Application with
// a custom factory via the WithFactory option.
func TestNewWithCustomFactory(t *testing.T) {
	t.Parallel()
	var errBuf bytes.Buffer
	customFactory := createMockFactory(big.NewInt(42), nil)
	args := []string{"fibcalc", "-n", "50"}

	app, err := New(args, &errBuf, WithFactory(customFactory))

	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}
	if app == nil {
		t.Fatal("New() returned nil application")
	}
	if app.Factory != customFactory {
		t.Error("Expected custom factory to be used")
	}

	// Verify it can run successfully with the custom factory
	var outBuf bytes.Buffer
	exitCode := app.Run(context.Background(), &outBuf).Code()
	if exitCode != apperrors.ExitSuccess {
		t.Errorf("Expected exit code %d, got %d", apperrors.ExitSuccess, exitCode)
	}
}

// TestRunCalculateVerboseMode tests the verbose flag path in runCalculate.
func TestRunCalculateVerboseMode(t *testing.T) {
	t.Parallel()
	var outBuf bytes.Buffer
	factory := createMockFactory(big.NewInt(55), nil)

	app := &Application{
		Config: config.AppConfig{
			N:         10,
			Algo:      "fast",
			Timeout:   1 * time.Minute,
			Verbose:   true,
			Details:   true,
			ShowValue: true,
		},
		Factory:   factory,
		ErrWriter: &bytes.Buffer{},
	}

	exitCode := app.Run(context.Background(), &outBuf).Code()

	if exitCode != apperrors.ExitSuccess {
		t.Errorf("Expected exit code %d, got %d", apperrors.ExitSuccess, exitCode)
	}
	output := testutil.StripAnsiCodes(outBuf.String())
	if !strings.Contains(output, "55") {
		t.Errorf("Verbose output should contain the result. Output:\n%s", output)
	}
}

// TestRunCalculateCalculatorError tests that calculator errors are handled.
func TestRunCalculateCalculatorError(t *testing.T) {
	t.Parallel()
	var outBuf bytes.Buffer
	factory := createMockFactory(nil, fmt.Errorf("calculation failed"))

	app := &Application{
		Config: config.AppConfig{
			N:       10,
			Algo:    "fast",
			Timeout: 1 * time.Minute,
		},
		Factory:   factory,
		ErrWriter: &bytes.Buffer{},
	}

	exitCode := app.Run(context.Background(), &outBuf).Code()

	// Should return an error exit code
	if exitCode == apperrors.ExitSuccess {
		t.Error("Expected non-success exit code for calculator error")
	}
}
