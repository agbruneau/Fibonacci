package format

import (
	"fmt"
	"strings"
	"time"
)

// FormatETA formats a duration into a human-readable ETA string.
// It adapts the format based on the magnitude of the duration.
//
// Parameters:
//   - eta: The duration to format.
//
// Returns:
//   - string: A formatted string like "< 1s", "2m30s", "1h15m".
func FormatETA(eta time.Duration) string {
	if eta <= 0 {
		return "calculating..."
	}

	if eta < time.Second {
		return "< 1s"
	}

	if eta < time.Minute {
		return fmt.Sprintf("%ds", int(eta.Seconds()))
	}

	if eta < time.Hour {
		minutes := int(eta.Minutes())
		seconds := int(eta.Seconds()) % 60
		if seconds > 0 {
			return fmt.Sprintf("%dm%ds", minutes, seconds)
		}
		return fmt.Sprintf("%dm", minutes)
	}

	hours := int(eta.Hours())
	minutes := int(eta.Minutes()) % 60
	if minutes > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	return fmt.Sprintf("%dh", hours)
}

// ProgressBar generates a string representing a textual progress bar.
//
// Parameters:
//   - progress: The normalized progress value (0.0 to 1.0).
//   - length: The total character width of the progress bar. All callers
//     pass a fixed positive width; a negative length panics (strings.Repeat).
//
// Returns:
//   - string: A string representation of the progress bar.
func ProgressBar(progress float64, length int) string {
	if progress > 1.0 {
		progress = 1.0
	}
	if progress < 0.0 {
		progress = 0.0
	}
	count := int(progress * float64(length))
	return strings.Repeat("█", count) + strings.Repeat("░", length-count)
}

// formatProgressBarWithETA generates a formatted progress string with ETA.
// It combines the progress percentage, visual bar, and time estimate.
//
// Parameters:
//   - progress: The normalized progress value (0.0 to 1.0).
//   - eta: The estimated time remaining.
//   - width: The width of the progress bar in characters.
//
// Returns:
//   - string: A formatted string like "45.00% [####....] ETA: 2m30s".
func formatProgressBarWithETA(progress float64, eta time.Duration, width int) string {
	bar := ProgressBar(progress, width)
	etaStr := FormatETA(eta)
	return fmt.Sprintf("%6.2f%% [%s] ETA: %s", progress*100, bar, etaStr)
}
