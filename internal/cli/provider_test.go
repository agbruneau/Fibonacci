package cli_test

import (
	"os"
	"testing"

	"github.com/agbruneau/FibGo/internal/cli"
	"github.com/agbruneau/FibGo/internal/ui"
)

// TestCLIColorProvider is the only test in this package that mutates the
// process-global NO_COLOR env var and the internal/ui active theme; both
// are saved and restored via defer so it can still declare t.Parallel()
// without leaking state into whatever else runs in the same test binary.
// No other test in this package asserts on exact color content -- they
// check ANSI-stripped or color-agnostic substrings -- so no additional
// serialization against sibling tests is required.
func TestCLIColorProvider(t *testing.T) {
	t.Parallel()

	// NO_COLOR (per no-color.org) may be set in the test environment;
	// unset it so the "colors enabled" branch below is actually exercised,
	// and restore it unconditionally afterwards.
	noColorVal, hadNoColor := os.LookupEnv("NO_COLOR")
	if hadNoColor {
		os.Unsetenv("NO_COLOR")
		defer os.Setenv("NO_COLOR", noColorVal)
	}
	savedTheme := ui.GetCurrentTheme()
	defer ui.SetCurrentTheme(savedTheme)

	provider := cli.CLIColorProvider{}

	ui.InitTheme(false)
	if provider.Yellow() == "" {
		t.Error("Yellow should return a color code when colors are enabled")
	}

	ui.InitTheme(true)
	if provider.Yellow() != "" {
		t.Error("Yellow should be empty when NoColor is true")
	}
	if provider.Reset() != "" {
		t.Error("Reset should be empty when NoColor is true")
	}
}
