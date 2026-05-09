package ui

import (
	"os"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
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

	// OrangeTheme is an orange-dominant dark theme matching the TUI palette.
	// Uses warm orange tones for a btop-inspired aesthetic.
	OrangeTheme = Theme{
		Name:      "orange",
		Primary:   "\033[38;5;208m", // Orange
		Secondary: "\033[38;5;245m", // Grey
		Success:   "\033[38;5;82m",  // Bright green
		Warning:   "\033[38;5;214m", // Light orange
		Error:     "\033[38;5;196m", // Red
		Info:      "\033[38;5;69m",  // Blue
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

// TUITheme defines lipgloss-compatible colors for the TUI dashboard.
// Each field is a lipgloss.TerminalColor suitable for use with
// lipgloss.Style.Foreground() and Background().
type TUITheme struct {
	Bg      lipgloss.TerminalColor
	Text    lipgloss.TerminalColor
	Border  lipgloss.TerminalColor
	Accent  lipgloss.TerminalColor
	Success lipgloss.TerminalColor
	Warning lipgloss.TerminalColor
	Error   lipgloss.TerminalColor
	Dim     lipgloss.TerminalColor
	Info    lipgloss.TerminalColor
}

var (
	// DarkTUITheme is the orange-dominant btop-inspired TUI palette.
	DarkTUITheme = TUITheme{
		Bg:      lipgloss.Color("#000000"),
		Text:    lipgloss.Color("#E0E0E0"),
		Border:  lipgloss.Color("#FF6600"),
		Accent:  lipgloss.Color("#FF8C00"),
		Success: lipgloss.Color("#9ece6a"),
		Warning: lipgloss.Color("#FFB347"),
		Error:   lipgloss.Color("#FF4444"),
		Dim:     lipgloss.Color("#666666"),
		Info:    lipgloss.Color("#4488FF"),
	}

	// NoColorTUITheme disables all TUI colors.
	// lipgloss.NoColor{} renders text with the terminal's default colors.
	NoColorTUITheme = TUITheme{
		Bg:      lipgloss.NoColor{},
		Text:    lipgloss.NoColor{},
		Border:  lipgloss.NoColor{},
		Accent:  lipgloss.NoColor{},
		Success: lipgloss.NoColor{},
		Warning: lipgloss.NoColor{},
		Error:   lipgloss.NoColor{},
		Dim:     lipgloss.NoColor{},
		Info:    lipgloss.NoColor{},
	}

	// HighContrastTUITheme is a black/white/yellow palette for low-vision and
	// high-contrast terminal profiles (WCAG-oriented, not a certification).
	HighContrastTUITheme = TUITheme{
		Bg:      lipgloss.Color("#000000"),
		Text:    lipgloss.Color("#FFFFFF"),
		Border:  lipgloss.Color("#FFFF00"),
		Accent:  lipgloss.Color("#FFFF00"),
		Success: lipgloss.Color("#00FF00"),
		Warning: lipgloss.Color("#FFAA00"),
		Error:   lipgloss.Color("#FF3333"),
		Dim:     lipgloss.Color("#CCCCCC"),
		Info:    lipgloss.Color("#66CCFF"),
	}
)

// GetCurrentTUITheme returns the TUI theme matching the currently active theme.
// When NoColorTheme is active, returns NoColorTUITheme. If the environment
// variable FIBCALC_TUI_THEME is set to "high-contrast" (case-insensitive),
// returns HighContrastTUITheme; otherwise DarkTUITheme.
func GetCurrentTUITheme() TUITheme {
	themeMutex.RLock()
	defer themeMutex.RUnlock()

	if currentTheme.Name == "none" {
		return NoColorTUITheme
	}
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("FIBCALC_TUI_THEME"))); v == "high-contrast" || v == "highcontrast" {
		return HighContrastTUITheme
	}
	return DarkTUITheme
}

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
// Valid names are: "dark", "light", "orange", "none".
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
	case "orange":
		currentTheme = OrangeTheme
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

// ---------------------------------------------------------------------------
// CLI color helpers
// ---------------------------------------------------------------------------
// The Color* functions below return ANSI escape codes from the currently
// active CLI theme. They are the primary API used by CLI presentation code
// (cli/, calibration/, app/) for consistent coloring. Each call is
// thread-safe via GetCurrentTheme.

// ColorReset returns the reset escape code from the current theme.
func ColorReset() string { return GetCurrentTheme().Reset }

// ColorRed returns the error color from the current theme.
func ColorRed() string { return GetCurrentTheme().Error }

// ColorGreen returns the success color from the current theme.
func ColorGreen() string { return GetCurrentTheme().Success }

// ColorYellow returns the warning color from the current theme.
func ColorYellow() string { return GetCurrentTheme().Warning }

// ColorBlue returns the primary color from the current theme.
func ColorBlue() string { return GetCurrentTheme().Primary }

// ColorMagenta returns the info color from the current theme.
func ColorMagenta() string { return GetCurrentTheme().Info }

// ColorCyan returns the secondary color from the current theme.
func ColorCyan() string { return GetCurrentTheme().Secondary }

// ColorBold returns the bold escape code from the current theme.
func ColorBold() string { return GetCurrentTheme().Bold }

// ColorUnderline returns the underline escape code from the current theme.
func ColorUnderline() string { return GetCurrentTheme().Underline }
