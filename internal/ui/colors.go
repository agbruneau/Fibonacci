// Package ui provides theme and color support for the application's user interface.
package ui

// Color functions return ANSI escape codes from the current theme.
// These functions provide a simple API for consistent color usage across the application.

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
