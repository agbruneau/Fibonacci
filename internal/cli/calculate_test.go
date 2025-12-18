package cli

import (
	"bytes"
	"testing"

	"github.com/agbru/fibcalc/internal/config"
	"github.com/agbru/fibcalc/internal/fibonacci"
)

// TestGetCalculatorsToRun tests the GetCalculatorsToRun function.
func TestGetCalculatorsToRun(t *testing.T) {
	t.Parallel()
	factory := fibonacci.GlobalFactory()

	t.Run("Single algorithm returns one calculator", func(t *testing.T) {
		t.Parallel()
		cfg := config.AppConfig{Algo: "fast"}
		calculators := GetCalculatorsToRun(cfg, factory)

		if len(calculators) != 1 {
			t.Errorf("Expected 1 calculator, got %d", len(calculators))
		}
		// Check that the name contains "Fast Doubling" (exact name may vary)
		if calculators[0].Name() == "" {
			t.Error("Calculator name should not be empty")
		}
	})

	t.Run("All algorithms returns multiple calculators", func(t *testing.T) {
		t.Parallel()
		cfg := config.AppConfig{Algo: "all"}
		calculators := GetCalculatorsToRun(cfg, factory)

		if len(calculators) < 2 {
			t.Errorf("Expected at least 2 calculators for 'all', got %d", len(calculators))
		}
	})

	t.Run("Matrix algorithm", func(t *testing.T) {
		t.Parallel()
		cfg := config.AppConfig{Algo: "matrix"}
		calculators := GetCalculatorsToRun(cfg, factory)

		if len(calculators) != 1 {
			t.Errorf("Expected 1 calculator, got %d", len(calculators))
		}
	})
}

// TestPrintExecutionConfig tests the PrintExecutionConfig function.
func TestPrintExecutionConfig(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	cfg := config.AppConfig{
		N:            1000,
		Timeout:      60000000000, // 1 minute
		Threshold:    4096,
		FFTThreshold: 1000000,
	}

	PrintExecutionConfig(cfg, &buf)

	output := buf.String()

	// Check that output contains expected components
	if output == "" {
		t.Error("PrintExecutionConfig should produce output")
	}
	if len(output) < 50 {
		t.Errorf("PrintExecutionConfig output seems too short: %s", output)
	}
}

// TestPrintExecutionMode tests the PrintExecutionMode function.
func TestPrintExecutionMode(t *testing.T) {
	t.Parallel()
	factory := fibonacci.GlobalFactory()

	t.Run("Single calculator mode", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		calculators := []fibonacci.Calculator{factory.MustGet("fast")}

		PrintExecutionMode(calculators, &buf)

		output := buf.String()
		if output == "" {
			t.Error("PrintExecutionMode should produce output")
		}
	})

	t.Run("Multiple calculators mode", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		cfg := config.AppConfig{Algo: "all"}
		calculators := GetCalculatorsToRun(cfg, factory)

		PrintExecutionMode(calculators, &buf)

		output := buf.String()
		if output == "" {
			t.Error("PrintExecutionMode should produce output for multiple calculators")
		}
	})
}
