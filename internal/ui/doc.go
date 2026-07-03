// Package ui provides theme and color support for the application's user interface.
//
// It exposes two coordinated theme systems sharing a single active state:
//
//   - CLI themes ([Theme], [DarkTheme], [NoColorTheme])
//     hold ANSI escape codes consumed by the CLI/calibration/app layers via the
//     [ColorRed], [ColorGreen], … helpers and [GetCurrentTheme].
//   - TUI themes ([TUITheme], [DarkTUITheme], [HighContrastTUITheme], [NoColorTUITheme])
//     hold lipgloss-compatible colors used by the Bubble Tea dashboard via
//     [GetCurrentTUITheme].
//
// Both systems honor the NO_COLOR environment variable and the --no-color flag
// through [InitTheme] (per https://no-color.org/). The TUI palette can be further
// switched to a high-contrast variant via FIBCALC_TUI_THEME=high-contrast.
//
// This package is designed to be a shared dependency for packages that need
// color output, reducing coupling between business logic and presentation.
package ui
