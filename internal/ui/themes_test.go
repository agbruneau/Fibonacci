package ui_test

import (
	"os"
	"testing"

	"github.com/agbruneau/FibGo/internal/ui"
)

// TestInitThemeWithNoColorFlag verifies that InitTheme respects the noColor flag.
func TestInitThemeWithNoColorFlag(t *testing.T) {
	// Save original theme and env to restore after test
	originalTheme := ui.GetCurrentTheme()
	originalNoColor := os.Getenv("NO_COLOR")
	defer func() {
		ui.SetCurrentTheme(originalTheme)
		if originalNoColor == "" {
			os.Unsetenv("NO_COLOR")
		} else {
			os.Setenv("NO_COLOR", originalNoColor)
		}
	}()

	// Ensure NO_COLOR is not set for this test
	os.Unsetenv("NO_COLOR")

	t.Run("noColor flag true disables colors", func(t *testing.T) {
		ui.InitTheme(true)
		current := ui.GetCurrentTheme()
		if current.Name != "none" {
			t.Errorf("InitTheme(true): got theme %q, want %q", current.Name, "none")
		}
		if current.Primary != "" {
			t.Errorf("InitTheme(true): Primary should be empty, got %q", current.Primary)
		}
	})

	t.Run("noColor flag false uses dark theme", func(t *testing.T) {
		ui.InitTheme(false)
		current := ui.GetCurrentTheme()
		if current.Name != "dark" {
			t.Errorf("InitTheme(false): got theme %q, want %q", current.Name, "dark")
		}
	})
}

// TestInitThemeWithNO_COLOREnv verifies that InitTheme respects NO_COLOR env var.
func TestInitThemeWithNO_COLOREnv(t *testing.T) {
	// Save original theme and env to restore after test
	originalTheme := ui.GetCurrentTheme()
	originalNoColor := os.Getenv("NO_COLOR")
	defer func() {
		ui.SetCurrentTheme(originalTheme)
		if originalNoColor == "" {
			os.Unsetenv("NO_COLOR")
		} else {
			os.Setenv("NO_COLOR", originalNoColor)
		}
	}()

	t.Run("NO_COLOR set disables colors", func(t *testing.T) {
		os.Setenv("NO_COLOR", "1")
		ui.InitTheme(false)
		current := ui.GetCurrentTheme()
		if current.Name != "none" {
			t.Errorf("InitTheme with NO_COLOR=1: got theme %q, want %q", current.Name, "none")
		}
	})

	t.Run("NO_COLOR empty value still disables colors", func(t *testing.T) {
		os.Setenv("NO_COLOR", "")
		ui.InitTheme(false)
		current := ui.GetCurrentTheme()
		if current.Name != "none" {
			t.Errorf("InitTheme with NO_COLOR='': got theme %q, want %q", current.Name, "none")
		}
	})

	t.Run("NO_COLOR not set uses dark theme", func(t *testing.T) {
		os.Unsetenv("NO_COLOR")
		ui.InitTheme(false)
		current := ui.GetCurrentTheme()
		if current.Name != "dark" {
			t.Errorf("InitTheme without NO_COLOR: got theme %q, want %q", current.Name, "dark")
		}
	})
}

// TestThemeColors verifies that theme colors are properly defined.
func TestThemeColors(t *testing.T) {
	t.Run("DarkTheme has non-empty colors", func(t *testing.T) {
		if ui.DarkTheme.Primary == "" {
			t.Error("DarkTheme.Primary should not be empty")
		}
		if ui.DarkTheme.Success == "" {
			t.Error("DarkTheme.Success should not be empty")
		}
		if ui.DarkTheme.Error == "" {
			t.Error("DarkTheme.Error should not be empty")
		}
		if ui.DarkTheme.Reset == "" {
			t.Error("DarkTheme.Reset should not be empty")
		}
	})

	t.Run("NoColorTheme has all empty colors", func(t *testing.T) {
		if ui.NoColorTheme.Primary != "" {
			t.Errorf("NoColorTheme.Primary should be empty, got %q", ui.NoColorTheme.Primary)
		}
		if ui.NoColorTheme.Success != "" {
			t.Errorf("NoColorTheme.Success should be empty, got %q", ui.NoColorTheme.Success)
		}
		if ui.NoColorTheme.Error != "" {
			t.Errorf("NoColorTheme.Error should be empty, got %q", ui.NoColorTheme.Error)
		}
		if ui.NoColorTheme.Reset != "" {
			t.Errorf("NoColorTheme.Reset should be empty, got %q", ui.NoColorTheme.Reset)
		}
	})
}

// TestColorFunctions verifies that color functions return current theme values.
func TestColorFunctions(t *testing.T) {
	// Save original theme to restore after test
	originalTheme := ui.GetCurrentTheme()
	defer func() { ui.SetCurrentTheme(originalTheme) }()

	t.Run("Color functions with DarkTheme", func(t *testing.T) {
		ui.SetCurrentTheme(ui.DarkTheme)
		if ui.ColorReset() != ui.DarkTheme.Reset {
			t.Errorf("ColorReset() = %q, want %q", ui.ColorReset(), ui.DarkTheme.Reset)
		}
		if ui.ColorGreen() != ui.DarkTheme.Success {
			t.Errorf("ColorGreen() = %q, want %q", ui.ColorGreen(), ui.DarkTheme.Success)
		}
		if ui.ColorRed() != ui.DarkTheme.Error {
			t.Errorf("ColorRed() = %q, want %q", ui.ColorRed(), ui.DarkTheme.Error)
		}
		if ui.ColorYellow() != ui.DarkTheme.Warning {
			t.Errorf("ColorYellow() = %q, want %q", ui.ColorYellow(), ui.DarkTheme.Warning)
		}
		if ui.ColorBlue() != ui.DarkTheme.Primary {
			t.Errorf("ColorBlue() = %q, want %q", ui.ColorBlue(), ui.DarkTheme.Primary)
		}
		if ui.ColorMagenta() != ui.DarkTheme.Info {
			t.Errorf("ColorMagenta() = %q, want %q", ui.ColorMagenta(), ui.DarkTheme.Info)
		}
		if ui.ColorCyan() != ui.DarkTheme.Secondary {
			t.Errorf("ColorCyan() = %q, want %q", ui.ColorCyan(), ui.DarkTheme.Secondary)
		}
		if ui.ColorBold() != ui.DarkTheme.Bold {
			t.Errorf("ColorBold() = %q, want %q", ui.ColorBold(), ui.DarkTheme.Bold)
		}
		if ui.ColorUnderline() != ui.DarkTheme.Underline {
			t.Errorf("ColorUnderline() = %q, want %q", ui.ColorUnderline(), ui.DarkTheme.Underline)
		}
	})

	t.Run("Color functions with NoColorTheme", func(t *testing.T) {
		ui.SetCurrentTheme(ui.NoColorTheme)
		if ui.ColorReset() != "" {
			t.Errorf("ColorReset() with none theme should be empty, got %q", ui.ColorReset())
		}
		if ui.ColorGreen() != "" {
			t.Errorf("ColorGreen() with none theme should be empty, got %q", ui.ColorGreen())
		}
		if ui.ColorRed() != "" {
			t.Errorf("ColorRed() with none theme should be empty, got %q", ui.ColorRed())
		}
		if ui.ColorYellow() != "" {
			t.Errorf("ColorYellow() with none theme should be empty, got %q", ui.ColorYellow())
		}
		if ui.ColorBlue() != "" {
			t.Errorf("ColorBlue() with none theme should be empty, got %q", ui.ColorBlue())
		}
		if ui.ColorMagenta() != "" {
			t.Errorf("ColorMagenta() with none theme should be empty, got %q", ui.ColorMagenta())
		}
		if ui.ColorCyan() != "" {
			t.Errorf("ColorCyan() with none theme should be empty, got %q", ui.ColorCyan())
		}
		if ui.ColorBold() != "" {
			t.Errorf("ColorBold() with none theme should be empty, got %q", ui.ColorBold())
		}
		if ui.ColorUnderline() != "" {
			t.Errorf("ColorUnderline() with none theme should be empty, got %q", ui.ColorUnderline())
		}
	})
}

func TestGetCurrentTUITheme_HighContrastEnv(t *testing.T) {
	orig := ui.GetCurrentTheme()
	t.Cleanup(func() { ui.SetCurrentTheme(orig) })

	ui.SetCurrentTheme(ui.DarkTheme)
	t.Setenv("FIBCALC_TUI_THEME", "high-contrast")
	ht := ui.GetCurrentTUITheme()
	if ht != ui.HighContrastTUITheme {
		t.Errorf("GetCurrentTUITheme() with FIBCALC_TUI_THEME=high-contrast: got %+v, want HighContrastTUITheme", ht)
	}

	t.Setenv("FIBCALC_TUI_THEME", "")
	if got := ui.GetCurrentTUITheme(); got != ui.DarkTUITheme {
		t.Errorf("with empty FIBCALC_TUI_THEME: got %+v, want DarkTUITheme", got)
	}
}
