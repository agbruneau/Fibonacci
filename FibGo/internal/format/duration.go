package format

import (
	"fmt"
	"time"
)

// FormatExecutionDuration formats a time.Duration for display.
// It shows microseconds for durations less than a millisecond, milliseconds for
// durations less than a second, and the default string representation otherwise.
// This approach provides a more human-readable output for short durations.
//
// Parameters:
//   - d: The duration to format.
//
// Returns:
//   - string: A formatted string representing the duration.
func FormatExecutionDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%d\u00b5s", d.Microseconds())
	} else if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return d.String()
}
