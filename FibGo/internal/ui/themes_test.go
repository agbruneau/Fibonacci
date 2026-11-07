package ui

import (
	"testing"

	"github.com/agbruneau/FibGo/internal/testutil"
)

// These tests mutate NO_COLOR, so they use t.Setenv / testutil.Unsetenv rather
// than os.Setenv with a hand-written defer (audit TST-02). The restore is then
// registered by the framework, it survives a t.Fatal, and — the reason that
// matters most — t.Setenv refuses to run in a parallel test, which is exactly
// the guarantee an environment-mutating test needs.

// TestInitThemeWithNoColorFlag verifies that InitTheme respects the noColor flag.
func TestInitThemeWithNoColorFlag(t *testing.T) {
	originalTheme := GetCurrentTheme()
	t.Cleanup(func() { setCurrentTheme(originalTheme) })

	// Ensure NO_COLOR is not set for this test.
	testutil.Unsetenv(t, "NO_COLOR")

	t.Run("noColor flag true disables colors", func(t *testing.T) {
		InitTheme(true)
		current := GetCurrentTheme()
		if current.Name != "none" {
			t.Errorf("InitTheme(true): got theme %q, want %q", current.Name, "none")
		}
		if current.Primary != "" {
			t.Errorf("InitTheme(true): Primary should be empty, got %q", current.Primary)
		}
	})

	t.Run("noColor flag false uses dark theme", func(t *testing.T) {
		InitTheme(false)
		current := GetCurrentTheme()
		if current.Name != "dark" {
			t.Errorf("InitTheme(false): got theme %q, want %q", current.Name, "dark")
		}
	})
}

// TestInitThemeWithNO_COLOREnv verifies that InitTheme respects NO_COLOR env var.
func TestInitThemeWithNO_COLOREnv(t *testing.T) {
	originalTheme := GetCurrentTheme()
	t.Cleanup(func() { setCurrentTheme(originalTheme) })

	t.Run("NO_COLOR set disables colors", func(t *testing.T) {
		t.Setenv("NO_COLOR", "1")
		InitTheme(false)
		current := GetCurrentTheme()
		if current.Name != "none" {
			t.Errorf("InitTheme with NO_COLOR=1: got theme %q, want %q", current.Name, "none")
		}
	})

	// Presence, not value: InitTheme uses os.LookupEnv, so NO_COLOR= disables
	// colors too. This is why the "not set" case below needs a real unset and
	// not t.Setenv(key, "").
	t.Run("NO_COLOR empty value still disables colors", func(t *testing.T) {
		t.Setenv("NO_COLOR", "")
		InitTheme(false)
		current := GetCurrentTheme()
		if current.Name != "none" {
			t.Errorf("InitTheme with NO_COLOR='': got theme %q, want %q", current.Name, "none")
		}
	})

	t.Run("NO_COLOR not set uses dark theme", func(t *testing.T) {
		testutil.Unsetenv(t, "NO_COLOR")
		InitTheme(false)
		current := GetCurrentTheme()
		if current.Name != "dark" {
			t.Errorf("InitTheme without NO_COLOR: got theme %q, want %q", current.Name, "dark")
		}
	})
}

// TestThemeColors verifies that theme colors are properly defined.
func TestThemeColors(t *testing.T) {
	t.Run("DarkTheme has non-empty colors", func(t *testing.T) {
		if DarkTheme.Primary == "" {
			t.Error("DarkTheme.Primary should not be empty")
		}
		if DarkTheme.Success == "" {
			t.Error("DarkTheme.Success should not be empty")
		}
		if DarkTheme.Error == "" {
			t.Error("DarkTheme.Error should not be empty")
		}
		if DarkTheme.Reset == "" {
			t.Error("DarkTheme.Reset should not be empty")
		}
	})

	t.Run("NoColorTheme has all empty colors", func(t *testing.T) {
		if NoColorTheme.Primary != "" {
			t.Errorf("NoColorTheme.Primary should be empty, got %q", NoColorTheme.Primary)
		}
		if NoColorTheme.Success != "" {
			t.Errorf("NoColorTheme.Success should be empty, got %q", NoColorTheme.Success)
		}
		if NoColorTheme.Error != "" {
			t.Errorf("NoColorTheme.Error should be empty, got %q", NoColorTheme.Error)
		}
		if NoColorTheme.Reset != "" {
			t.Errorf("NoColorTheme.Reset should be empty, got %q", NoColorTheme.Reset)
		}
	})
}

// TestColorFunctions verifies that color functions return current theme values.
func TestColorFunctions(t *testing.T) {
	// Save original theme to restore after test
	originalTheme := GetCurrentTheme()
	defer func() { setCurrentTheme(originalTheme) }()

	t.Run("Color functions with DarkTheme", func(t *testing.T) {
		setCurrentTheme(DarkTheme)
		if ColorReset() != DarkTheme.Reset {
			t.Errorf("ColorReset() = %q, want %q", ColorReset(), DarkTheme.Reset)
		}
		if ColorGreen() != DarkTheme.Success {
			t.Errorf("ColorGreen() = %q, want %q", ColorGreen(), DarkTheme.Success)
		}
		if ColorRed() != DarkTheme.Error {
			t.Errorf("ColorRed() = %q, want %q", ColorRed(), DarkTheme.Error)
		}
		if ColorYellow() != DarkTheme.Warning {
			t.Errorf("ColorYellow() = %q, want %q", ColorYellow(), DarkTheme.Warning)
		}
		if ColorBlue() != DarkTheme.Primary {
			t.Errorf("ColorBlue() = %q, want %q", ColorBlue(), DarkTheme.Primary)
		}
		if ColorMagenta() != DarkTheme.Info {
			t.Errorf("ColorMagenta() = %q, want %q", ColorMagenta(), DarkTheme.Info)
		}
		if ColorCyan() != DarkTheme.Secondary {
			t.Errorf("ColorCyan() = %q, want %q", ColorCyan(), DarkTheme.Secondary)
		}
		if ColorBold() != DarkTheme.Bold {
			t.Errorf("ColorBold() = %q, want %q", ColorBold(), DarkTheme.Bold)
		}
		if ColorUnderline() != DarkTheme.Underline {
			t.Errorf("ColorUnderline() = %q, want %q", ColorUnderline(), DarkTheme.Underline)
		}
	})

	t.Run("Color functions with NoColorTheme", func(t *testing.T) {
		setCurrentTheme(NoColorTheme)
		if ColorReset() != "" {
			t.Errorf("ColorReset() with none theme should be empty, got %q", ColorReset())
		}
		if ColorGreen() != "" {
			t.Errorf("ColorGreen() with none theme should be empty, got %q", ColorGreen())
		}
		if ColorRed() != "" {
			t.Errorf("ColorRed() with none theme should be empty, got %q", ColorRed())
		}
		if ColorYellow() != "" {
			t.Errorf("ColorYellow() with none theme should be empty, got %q", ColorYellow())
		}
		if ColorBlue() != "" {
			t.Errorf("ColorBlue() with none theme should be empty, got %q", ColorBlue())
		}
		if ColorMagenta() != "" {
			t.Errorf("ColorMagenta() with none theme should be empty, got %q", ColorMagenta())
		}
		if ColorCyan() != "" {
			t.Errorf("ColorCyan() with none theme should be empty, got %q", ColorCyan())
		}
		if ColorBold() != "" {
			t.Errorf("ColorBold() with none theme should be empty, got %q", ColorBold())
		}
		if ColorUnderline() != "" {
			t.Errorf("ColorUnderline() with none theme should be empty, got %q", ColorUnderline())
		}
	})
}

// The palette is chosen by name, not by reading FIBCALC_TUI_THEME here
// (audit CFG-02): internal/config parses that variable like every other one.
// This test therefore needs no environment at all, which is the improvement.
func TestTUIThemeFor(t *testing.T) {
	t.Parallel()

	orig := GetCurrentTheme()
	t.Cleanup(func() { setCurrentTheme(orig) })
	setCurrentTheme(DarkTheme)

	tests := []struct {
		name string
		want TUITheme
	}{
		{"high-contrast", HighContrastTUITheme},
		{"highcontrast", HighContrastTUITheme},
		{"HIGH-CONTRAST", HighContrastTUITheme},
		{"  high-contrast  ", HighContrastTUITheme},
		{"", DarkTUITheme},
		{"dark", DarkTUITheme},
		{"nonsense", DarkTUITheme},
	}
	for _, tt := range tests {
		if got := TUIThemeFor(tt.name); got != tt.want {
			t.Errorf("TUIThemeFor(%q) = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

// The active color theme wins over the requested palette: under NO_COLOR or
// --machine there is nothing to color, whatever the caller asked for.
func TestTUIThemeFor_NoColorWins(t *testing.T) {
	orig := GetCurrentTheme()
	t.Cleanup(func() { setCurrentTheme(orig) })

	setCurrentTheme(NoColorTheme)
	if got := TUIThemeFor("high-contrast"); got != NoColorTUITheme {
		t.Errorf("TUIThemeFor with the no-color theme active = %+v, want NoColorTUITheme", got)
	}
}

// NoColorRequested follows the no-color.org convention: presence decides,
// including an empty value.
func TestNoColorRequested(t *testing.T) {
	testutil.Unsetenv(t, "NO_COLOR")
	if NoColorRequested() {
		t.Error("NoColorRequested() is true with NO_COLOR unset")
	}

	t.Setenv("NO_COLOR", "")
	if !NoColorRequested() {
		t.Error("NoColorRequested() is false with NO_COLOR set to the empty string")
	}

	t.Setenv("NO_COLOR", "1")
	if !NoColorRequested() {
		t.Error("NoColorRequested() is false with NO_COLOR=1")
	}
}
