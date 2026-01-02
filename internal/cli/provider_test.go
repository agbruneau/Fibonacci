package cli

import (
	"os"
	"testing"

	"github.com/agbru/fibcalc/internal/ui"
)

func TestCLIColorProvider(t *testing.T) {
	// Save and temporarily unset NO_COLOR to test with colors enabled
	// This is necessary because InitTheme respects the NO_COLOR environment
	// variable (per no-color.org spec), which may be set in the test environment
	noColorVal, hadNoColor := os.LookupEnv("NO_COLOR")
	if hadNoColor {
		os.Unsetenv("NO_COLOR")
		defer func() {
			if hadNoColor {
				os.Setenv("NO_COLOR", noColorVal)
			}
		}()
	}

	// Initialize theme to ensure we get codes
	ui.InitTheme(false)

	provider := CLIColorProvider{}

	// Test Yellow
	if provider.Yellow() == "" {
		t.Error("Yellow should return a color code when colors are enabled")
	}
	// We just want to ensure it calls the function
	_ = provider.Yellow()

	// Test Reset
	_ = provider.Reset()

	// Test with NoColor
	ui.InitTheme(true)
	if provider.Yellow() != "" {
		t.Error("Yellow should be empty when NoColor is true")
	}
	if provider.Reset() != "" {
		t.Error("Reset should be empty when NoColor is true")
	}
}
