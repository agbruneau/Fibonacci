package cli

import (
	"testing"
)

func TestCLIColorProvider(t *testing.T) {
	// Initialize theme to ensure we get codes
	InitTheme(false)

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
	InitTheme(true)
	if provider.Yellow() != "" {
		t.Error("Yellow should be empty when NoColor is true")
	}
	if provider.Reset() != "" {
		t.Error("Reset should be empty when NoColor is true")
	}
}
