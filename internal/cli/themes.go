// Package cli provides theme support for the command-line interface.
// It allows for customizable color schemes and supports automatic detection
// of the NO_COLOR environment variable for accessibility.
package cli

import (
	"os"
)

// Theme defines a color scheme for CLI output.
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
		Secondary: "\033[38;5;245m", // Gray
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
		Secondary: "\033[38;5;240m", // Dark gray
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

	// CurrentTheme is the active theme used throughout the CLI.
	// Defaults to DarkTheme but can be changed via SetTheme or InitTheme.
	CurrentTheme = DarkTheme
)

// SetTheme changes the active theme by name.
// Valid names are: "dark", "light", "none".
// Unknown names default to dark theme.
//
// Parameters:
//   - name: The name of the theme to activate.
func SetTheme(name string) {
	switch name {
	case "dark":
		CurrentTheme = DarkTheme
	case "light":
		CurrentTheme = LightTheme
	case "none":
		CurrentTheme = NoColorTheme
	default:
		CurrentTheme = DarkTheme
	}
}

// InitTheme initializes the theme based on the noColor flag and environment.
// It respects the NO_COLOR environment variable (https://no-color.org/) for
// accessibility. If noColor is true or NO_COLOR is set, colors are disabled.
//
// Parameters:
//   - noColor: If true, disables all color output regardless of environment.
func InitTheme(noColor bool) {
	// Check --no-color flag first
	if noColor {
		CurrentTheme = NoColorTheme
		return
	}

	// Check NO_COLOR environment variable
	// Any non-empty value disables colors (per no-color.org spec)
	if _, exists := os.LookupEnv("NO_COLOR"); exists {
		CurrentTheme = NoColorTheme
		return
	}

	// Default to dark theme
	CurrentTheme = DarkTheme
}
