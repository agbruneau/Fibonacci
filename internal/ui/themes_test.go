package ui

import (
	"os"
	"testing"
)

// TestSetTheme verifies that SetTheme correctly switches between themes.
func TestSetTheme(t *testing.T) {
	// Save original theme to restore after test
	originalTheme := GetCurrentTheme()
	defer func() { SetCurrentTheme(originalTheme) }()

	testCases := []struct {
		name          string
		themeName     string
		expectedTheme Theme
	}{
		{"Set dark theme", "dark", DarkTheme},
		{"Set light theme", "light", LightTheme},
		{"Set none theme", "none", NoColorTheme},
		{"Unknown theme defaults to dark", "unknown", DarkTheme},
		{"Empty string defaults to dark", "", DarkTheme},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			SetTheme(tc.themeName)
			current := GetCurrentTheme()
			if current.Name != tc.expectedTheme.Name {
				t.Errorf("SetTheme(%q): got theme %q, want %q",
					tc.themeName, current.Name, tc.expectedTheme.Name)
			}
		})
	}
}

// TestInitThemeWithNoColorFlag verifies that InitTheme respects the noColor flag.
func TestInitThemeWithNoColorFlag(t *testing.T) {
	// Save original theme and env to restore after test
	originalTheme := GetCurrentTheme()
	originalNoColor := os.Getenv("NO_COLOR")
	defer func() {
		SetCurrentTheme(originalTheme)
		if originalNoColor == "" {
			os.Unsetenv("NO_COLOR")
		} else {
			os.Setenv("NO_COLOR", originalNoColor)
		}
	}()

	// Ensure NO_COLOR is not set for this test
	os.Unsetenv("NO_COLOR")

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
	// Save original theme and env to restore after test
	originalTheme := GetCurrentTheme()
	originalNoColor := os.Getenv("NO_COLOR")
	defer func() {
		SetCurrentTheme(originalTheme)
		if originalNoColor == "" {
			os.Unsetenv("NO_COLOR")
		} else {
			os.Setenv("NO_COLOR", originalNoColor)
		}
	}()

	t.Run("NO_COLOR set disables colors", func(t *testing.T) {
		os.Setenv("NO_COLOR", "1")
		InitTheme(false)
		current := GetCurrentTheme()
		if current.Name != "none" {
			t.Errorf("InitTheme with NO_COLOR=1: got theme %q, want %q", current.Name, "none")
		}
	})

	t.Run("NO_COLOR empty value still disables colors", func(t *testing.T) {
		os.Setenv("NO_COLOR", "")
		InitTheme(false)
		current := GetCurrentTheme()
		if current.Name != "none" {
			t.Errorf("InitTheme with NO_COLOR='': got theme %q, want %q", current.Name, "none")
		}
	})

	t.Run("NO_COLOR not set uses dark theme", func(t *testing.T) {
		os.Unsetenv("NO_COLOR")
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

	t.Run("LightTheme has non-empty colors", func(t *testing.T) {
		if LightTheme.Primary == "" {
			t.Error("LightTheme.Primary should not be empty")
		}
		if LightTheme.Success == "" {
			t.Error("LightTheme.Success should not be empty")
		}
		if LightTheme.Error == "" {
			t.Error("LightTheme.Error should not be empty")
		}
		if LightTheme.Reset == "" {
			t.Error("LightTheme.Reset should not be empty")
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
	defer func() { SetCurrentTheme(originalTheme) }()

	t.Run("Color functions with DarkTheme", func(t *testing.T) {
		SetTheme("dark")
		if ColorReset() != DarkTheme.Reset {
			t.Errorf("ColorReset() = %q, want %q", ColorReset(), DarkTheme.Reset)
		}
		if ColorGreen() != DarkTheme.Success {
			t.Errorf("ColorGreen() = %q, want %q", ColorGreen(), DarkTheme.Success)
		}
		if ColorRed() != DarkTheme.Error {
			t.Errorf("ColorRed() = %q, want %q", ColorRed(), DarkTheme.Error)
		}
	})

	t.Run("Color functions with NoColorTheme", func(t *testing.T) {
		SetTheme("none")
		if ColorReset() != "" {
			t.Errorf("ColorReset() with none theme should be empty, got %q", ColorReset())
		}
		if ColorGreen() != "" {
			t.Errorf("ColorGreen() with none theme should be empty, got %q", ColorGreen())
		}
		if ColorRed() != "" {
			t.Errorf("ColorRed() with none theme should be empty, got %q", ColorRed())
		}
	})
}
