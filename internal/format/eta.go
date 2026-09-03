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
//   - length: The total character width of the progress bar.
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
	var builder strings.Builder
	builder.Grow(length)
	for i := 0; i < length; i++ {
		if i < count {
			builder.WriteRune('█')
		} else {
			builder.WriteRune('░')
		}
	}
	return builder.String()
}
