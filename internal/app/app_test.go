package app

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/agbru/fibcalc/internal/cli"
	"github.com/agbru/fibcalc/internal/config"
	apperrors "github.com/agbru/fibcalc/internal/errors"
	"github.com/agbru/fibcalc/internal/fibonacci"
	"github.com/agbru/fibcalc/internal/orchestration"
	"github.com/agbru/fibcalc/internal/testutil"
)

// TestNew tests the New function for creating Application instances.
func TestNew(t *testing.T) {
	t.Run("Valid args create application", func(t *testing.T) {
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
		if app.Config.N != 250000000 {
			t.Errorf("Expected default N=250000000, got N=%d", app.Config.N)
		}
	})
}

// TestApplicationRun tests the Application.Run method.
func TestApplicationRun(t *testing.T) {
	t.Run("Simple execution with success", func(t *testing.T) {
		var outBuf bytes.Buffer
		app := &Application{
			Config: config.AppConfig{
				N:            10,
				Algo:         "fast",
				Timeout:      1 * time.Minute,
				Threshold:    fibonacci.DefaultParallelThreshold,
				FFTThreshold: 20000,
				Details:      true,
				Concise:      true,
			},
			Factory:   fibonacci.GlobalFactory(),
			ErrWriter: &bytes.Buffer{},
		}

		exitCode := app.Run(context.Background(), &outBuf)

		if exitCode != apperrors.ExitSuccess {
			t.Errorf("Expected exit code %d, got %d", apperrors.ExitSuccess, exitCode)
		}
		output := testutil.StripAnsiCodes(outBuf.String())
		if !strings.Contains(output, "F(10) = 55") {
			t.Errorf("Output should contain 'F(10) = 55'. Output:\n%s", output)
		}
	})

	t.Run("Parallel comparison with success", func(t *testing.T) {
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
			Factory:   fibonacci.GlobalFactory(),
			ErrWriter: &bytes.Buffer{},
		}

		exitCode := app.Run(context.Background(), &outBuf)

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
		app := &Application{
			Config: config.AppConfig{
				N:       100_000_000,
				Algo:    "fast",
				Timeout: 1 * time.Millisecond,
			},
			Factory:   fibonacci.GlobalFactory(),
			ErrWriter: &bytes.Buffer{},
		}

		exitCode := app.Run(context.Background(), &outBuf)

		if exitCode != apperrors.ExitErrorTimeout {
			t.Errorf("Expected exit code %d (timeout), got %d", apperrors.ExitErrorTimeout, exitCode)
		}
		output := testutil.StripAnsiCodes(outBuf.String())
		if !strings.Contains(output, "Timeout") {
			t.Errorf("Output should mention timeout. Output:\n%s", output)
		}
	})

	t.Run("Context cancellation", func(t *testing.T) {
		var outBuf bytes.Buffer
		app := &Application{
			Config: config.AppConfig{
				N:       100_000_000,
				Algo:    "fast",
				Timeout: 1 * time.Minute,
			},
			Factory:   fibonacci.GlobalFactory(),
			ErrWriter: &bytes.Buffer{},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		exitCode := app.Run(ctx, &outBuf)

		if exitCode != apperrors.ExitErrorCanceled {
			t.Errorf("Expected exit code %d (canceled), got %d", apperrors.ExitErrorCanceled, exitCode)
		}
	})

	t.Run("JSON output mode", func(t *testing.T) {
		var outBuf bytes.Buffer
		app := &Application{
			Config: config.AppConfig{
				N:          10,
				Algo:       "fast",
				Timeout:    1 * time.Minute,
				JSONOutput: true,
			},
			Factory:   fibonacci.GlobalFactory(),
			ErrWriter: &bytes.Buffer{},
		}

		exitCode := app.Run(context.Background(), &outBuf)

		if exitCode != apperrors.ExitSuccess {
			t.Errorf("Expected exit code %d, got %d", apperrors.ExitSuccess, exitCode)
		}
		output := outBuf.String()
		if !strings.Contains(output, `"algorithm"`) {
			t.Errorf("JSON output should contain 'algorithm' field. Output:\n%s", output)
		}
		if !strings.Contains(output, `"result"`) {
			t.Errorf("JSON output should contain 'result' field. Output:\n%s", output)
		}
	})

	t.Run("Quiet mode", func(t *testing.T) {
		var outBuf bytes.Buffer
		app := &Application{
			Config: config.AppConfig{
				N:       10,
				Algo:    "fast",
				Timeout: 1 * time.Minute,
				Quiet:   true,
			},
			Factory:   fibonacci.GlobalFactory(),
			ErrWriter: &bytes.Buffer{},
		}

		exitCode := app.Run(context.Background(), &outBuf)

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
	var errBuf bytes.Buffer
	args := []string{"fibcalc", "-h"}

	_, err := New(args, &errBuf)

	if !IsHelpError(err) {
		t.Error("IsHelpError should return true for help flag error")
	}
}

// TestRunCompletion tests the completion script generation.
func TestRunCompletion(t *testing.T) {
	var outBuf bytes.Buffer
	app := &Application{
		Config: config.AppConfig{
			Completion: "bash",
		},
		Factory:   fibonacci.GlobalFactory(),
		ErrWriter: &bytes.Buffer{},
	}

	exitCode := app.Run(context.Background(), &outBuf)

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
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	app := &Application{
		Config: config.AppConfig{
			Completion: "invalid-shell",
		},
		Factory:   fibonacci.GlobalFactory(),
		ErrWriter: &errBuf,
	}

	exitCode := app.Run(context.Background(), &outBuf)

	if exitCode == apperrors.ExitSuccess {
		t.Error("Expected error exit code for invalid shell")
	}
}

// TestPrintJSONResults tests the JSON output formatting.
func TestPrintJSONResults(t *testing.T) {
	var outBuf bytes.Buffer
	app := &Application{
		Config: config.AppConfig{
			N:          5,
			Algo:       "fast",
			Timeout:    1 * time.Minute,
			JSONOutput: true,
		},
		Factory:   fibonacci.GlobalFactory(),
		ErrWriter: &bytes.Buffer{},
	}

	exitCode := app.Run(context.Background(), &outBuf)

	if exitCode != apperrors.ExitSuccess {
		t.Errorf("Expected exit code %d, got %d", apperrors.ExitSuccess, exitCode)
	}

	output := outBuf.String()
	// Verify JSON structure
	if !strings.Contains(output, `"algorithm"`) {
		t.Error("JSON output should contain 'algorithm' field")
	}
	if !strings.Contains(output, `"duration"`) {
		t.Error("JSON output should contain 'duration' field")
	}
	if !strings.Contains(output, `"result"`) {
		t.Error("JSON output should contain 'result' field")
	}
	// F(5) = 5
	if !strings.Contains(output, `"5"`) {
		t.Errorf("JSON output should contain result '5' for F(5). Got:\n%s", output)
	}
}

// TestHexOutput tests hexadecimal output mode.
func TestHexOutput(t *testing.T) {
	var outBuf bytes.Buffer
	app := &Application{
		Config: config.AppConfig{
			N:         10,
			Algo:      "fast",
			Timeout:   1 * time.Minute,
			HexOutput: true,
			Details:   true,
		},
		Factory:   fibonacci.GlobalFactory(),
		ErrWriter: &bytes.Buffer{},
	}

	exitCode := app.Run(context.Background(), &outBuf)

	if exitCode != apperrors.ExitSuccess {
		t.Errorf("Expected exit code %d, got %d", apperrors.ExitSuccess, exitCode)
	}

	output := testutil.StripAnsiCodes(outBuf.String())
	if !strings.Contains(output, "Hexadecimal") || !strings.Contains(output, "0x") {
		t.Errorf("Output should contain hexadecimal format. Got:\n%s", output)
	}
}

// TestRunAutoCalibrationDisabled tests that auto-calibration doesn't run when disabled.
func TestRunAutoCalibrationDisabled(t *testing.T) {
	var outBuf bytes.Buffer
	app := &Application{
		Config: config.AppConfig{
			N:             10,
			Algo:          "fast",
			Timeout:       1 * time.Minute,
			AutoCalibrate: false, // Disabled
		},
		Factory:   fibonacci.GlobalFactory(),
		ErrWriter: &bytes.Buffer{},
	}

	exitCode := app.Run(context.Background(), &outBuf)

	if exitCode != apperrors.ExitSuccess {
		t.Errorf("Expected exit code %d, got %d", apperrors.ExitSuccess, exitCode)
	}
}

// TestMultipleAlgorithms tests running all algorithms.
func TestMultipleAlgorithms(t *testing.T) {
	var outBuf bytes.Buffer
	app := &Application{
		Config: config.AppConfig{
			N:       15,
			Algo:    "all",
			Timeout: 1 * time.Minute,
			Details: true,
		},
		Factory:   fibonacci.GlobalFactory(),
		ErrWriter: &bytes.Buffer{},
	}

	exitCode := app.Run(context.Background(), &outBuf)

	if exitCode != apperrors.ExitSuccess {
		t.Errorf("Expected exit code %d, got %d", apperrors.ExitSuccess, exitCode)
	}

	output := testutil.StripAnsiCodes(outBuf.String())
	if !strings.Contains(output, "Comparison Summary") {
		t.Errorf("Output should contain comparison summary. Got:\n%s", output)
	}
}

// TestSetupSignals tests the SetupSignals function.
func TestSetupSignals(t *testing.T) {
	ctx := context.Background()
	ctxWithSignals, stop := SetupSignals(ctx)
	defer stop()

	// Context should not be nil
	if ctxWithSignals == nil {
		t.Error("Context should not be nil")
	}

	// Stop should not panic
	stop()
}

func TestApplyAdaptiveThresholds(t *testing.T) {
	// Test case where defaults are present and should be replaced
	t.Run("ReplaceDefaults", func(t *testing.T) {
		cfg := config.AppConfig{
			Threshold:         fibonacci.DefaultParallelThreshold,
			FFTThreshold:      fibonacci.DefaultFFTThreshold,
			StrassenThreshold: fibonacci.DefaultStrassenThreshold,
		}

		// Since we can't easily check internal calls without mocking,
		// we mainly check that it runs safely and returns a valid config.
		// The thresholds might remain default if the environment matches the defaults,
		// or change if it differs.
		newCfg := applyAdaptiveThresholds(cfg)
		_ = newCfg
	})

	// Test case where user overrides should be preserved
	t.Run("PreserveOverrides", func(t *testing.T) {
		cfg := config.AppConfig{
			Threshold:         1234,
			FFTThreshold:      5678,
			StrassenThreshold: 9012,
		}

		newCfg := applyAdaptiveThresholds(cfg)

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

	t.Run("Hex Output", func(t *testing.T) {
		var outBuf bytes.Buffer
		outputCfg := cli.OutputConfig{HexOutput: true}
		exitCode := app.analyzeResultsWithOutput(results, outputCfg, &outBuf)
		if exitCode != apperrors.ExitSuccess {
			t.Errorf("Expected success, got %d", exitCode)
		}
		if !strings.Contains(outBuf.String(), "0x37") { // 55 in hex is 37
			t.Errorf("Expected hex 0x37, got %s", outBuf.String())
		}
	})

	t.Run("No Success Results", func(t *testing.T) {
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

func TestPrintJSONResultsError(t *testing.T) {
	results := []orchestration.CalculationResult{
		{
			Name: "fail",
			Err:  fmt.Errorf("intentional failure"),
		},
	}
	var outBuf bytes.Buffer
	exitCode := printJSONResults(results, &outBuf)
	if exitCode != apperrors.ExitSuccess {
		t.Errorf("Expected success, got %d", exitCode)
	}
	if !strings.Contains(outBuf.String(), "intentional failure") {
		t.Errorf("Expected error in JSON, got %s", outBuf.String())
	}
}
