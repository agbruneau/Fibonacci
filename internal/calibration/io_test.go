package calibration

import (
	"bytes"
	"strings"
	"testing"

	"github.com/agbru/fibcalc/internal/config"
)

func TestPrintCalibrationOutput(t *testing.T) {
	t.Parallel()

	t.Run("Print calibration results", func(t *testing.T) {
		t.Parallel()
		var outBuf bytes.Buffer

		cfg := config.AppConfig{
			Threshold:         4096,
			FFTThreshold:      1000000,
			StrassenThreshold: 256,
		}

		printCalibrationOutput(cfg, &outBuf)

		output := outBuf.String()
		if !strings.Contains(output, "Auto-calibration") {
			t.Error("Output should contain 'Auto-calibration'")
		}
		if !strings.Contains(output, "4096") {
			t.Error("Output should contain threshold value 4096")
		}
		if !strings.Contains(output, "1000000") {
			t.Error("Output should contain FFT threshold value 1000000")
		}
		if !strings.Contains(output, "256") {
			t.Error("Output should contain Strassen threshold value 256")
		}
	})

	t.Run("Print with zero thresholds", func(t *testing.T) {
		t.Parallel()
		var outBuf bytes.Buffer

		cfg := config.AppConfig{
			Threshold:         0,
			FFTThreshold:      0,
			StrassenThreshold: 0,
		}

		printCalibrationOutput(cfg, &outBuf)

		output := outBuf.String()
		if !strings.Contains(output, "Auto-calibration") {
			t.Error("Output should contain 'Auto-calibration'")
		}
		// Should still print even with zero values
		if len(output) == 0 {
			t.Error("Output should not be empty")
		}
	})
}
