package cli

import (
	"bytes"
	"testing"

	"github.com/agbru/fibcalc/internal/config"
)

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

	t.Run("Single calculator mode", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		PrintExecutionMode([]string{"Fast Doubling"}, &buf)
		if buf.Len() == 0 {
			t.Error("PrintExecutionMode should produce output")
		}
	})

	t.Run("Multiple calculators mode", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		PrintExecutionMode([]string{"Fast Doubling", "Matrix Exp", "FFT"}, &buf)
		if buf.Len() == 0 {
			t.Error("PrintExecutionMode should produce output for multiple calculators")
		}
	})
}
