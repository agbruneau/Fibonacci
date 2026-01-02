// Package ui provides theme and color support for the application's user interface.
// It defines color schemes and provides ANSI escape code functions for consistent
// styling across the CLI and other presentation layers.
//
// This package is designed to be a shared dependency for packages that need
// color output, reducing coupling between business logic and presentation.
package ui

import (
	"os"
	"sync"
)

// Theme defines a color scheme for UI output.
// Each field contains an ANSI escape code for the corresponding color category.
type Theme struct {
	// Name is the identifier of the theme.
	Name string
	// Primary is the main accent color for important elements.
	Primary string
	// Secondary is used for less prominent elements.
	Secondary string
	// Success indicates positive outcomes or completed operations.
	Success string
	// Warning is used for caution messages or non-critical issues.
	Warning string
	// Error indicates failures or critical issues.
	Error string
	// Info is used for informational messages.
	Info string
	// Bold is the escape code for bold text.
	Bold string
	// Underline is the escape code for underlined text.
	Underline string
	// Reset clears all formatting.
	Reset string
}

var (
	// DarkTheme is optimized for dark terminal backgrounds.
	// Uses bright, vibrant colors for good contrast.
	DarkTheme = Theme{
		Name:      "dark",
		Primary:   "\033[38;5;39m",  // Bright blue
		Secondary: "\033[38;5;245m", // Grey
		Success:   "\033[38;5;82m",  // Bright green
		Warning:   "\033[38;5;220m", // Yellow
		Error:     "\033[38;5;196m", // Red
		Info:      "\033[38;5;141m", // Purple
		Bold:      "\033[1m",
		Underline: "\033[4m",
		Reset:     "\033[0m",
	}

	// LightTheme is optimized for light terminal backgrounds.
	// Uses darker colors for better readability.
	LightTheme = Theme{
		Name:      "light",
		Primary:   "\033[38;5;27m",  // Dark blue
		Secondary: "\033[38;5;240m", // Dark grey
		Success:   "\033[38;5;28m",  // Dark green
		Warning:   "\033[38;5;130m", // Orange
		Error:     "\033[38;5;124m", // Dark red
		Info:      "\033[38;5;54m",  // Dark purple
		Bold:      "\033[1m",
		Underline: "\033[4m",
		Reset:     "\033[0m",
	}

	// NoColorTheme disables all color output.
	// Used when NO_COLOR is set or --no-color flag is provided.
	NoColorTheme = Theme{
		Name:      "none",
		Primary:   "",
		Secondary: "",
		Success:   "",
		Warning:   "",
		Error:     "",
		Info:      "",
		Bold:      "",
		Underline: "",
		Reset:     "",
	}

	// currentTheme is the active theme used throughout the application.
	// Defaults to DarkTheme but can be changed via SetTheme or InitTheme.
	currentTheme = DarkTheme
	themeMutex   sync.RWMutex
)

// GetCurrentTheme returns the currently active theme in a thread-safe manner.
func GetCurrentTheme() Theme {
	themeMutex.RLock()
	defer themeMutex.RUnlock()
	return currentTheme
}

// SetCurrentTheme sets the currently active theme in a thread-safe manner.
// This is primarily used for testing purposes to restore state.
func SetCurrentTheme(t Theme) {
	themeMutex.Lock()
	defer themeMutex.Unlock()
	currentTheme = t
}

// SetTheme changes the active theme by name.
// Valid names are: "dark", "light", "none".
// Unknown names default to dark theme.
//
// Parameters:
//   - name: The name of the theme to activate.
func SetTheme(name string) {
	themeMutex.Lock()
	defer themeMutex.Unlock()

	switch name {
	case "dark":
		currentTheme = DarkTheme
	case "light":
		currentTheme = LightTheme
	case "none":
		currentTheme = NoColorTheme
	default:
		currentTheme = DarkTheme
	}
}

// InitTheme initializes the theme based on the noColor flag and environment.
// It respects the NO_COLOR environment variable (https://no-color.org/) for
// accessibility. If noColor is true or NO_COLOR is set, colors are disabled.
//
// Parameters:
//   - noColor: If true, disables all color output regardless of environment.
func InitTheme(noColor bool) {
	themeMutex.Lock()
	defer themeMutex.Unlock()

	// Check --no-color flag first
	if noColor {
		currentTheme = NoColorTheme
		return
	}

	// Check NO_COLOR environment variable
	// Any non-empty value disables colors (per no-color.org spec)
	if _, exists := os.LookupEnv("NO_COLOR"); exists {
		currentTheme = NoColorTheme
		return
	}

	// Default to dark theme
	currentTheme = DarkTheme
}
