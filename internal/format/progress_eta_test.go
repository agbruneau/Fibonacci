// This file stays package format (white-box) because it exercises the
// unexported formatProgressBarWithETA directly. See PLAN.md §3.3a:
// formatProgressBarWithETA is a frozen internal symbol precisely so this
// test keeps compiling. Every other test in this package lives in
// format_test.go and only touches the exported API.
package format

import (
	"strings"
	"testing"
	"time"
)

// TestFormatProgressBarWithETA verifies combined progress and ETA formatting.
func TestFormatProgressBarWithETA(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		progress float64
		eta      time.Duration
		width    int
	}{
		{"zero progress", 0, time.Minute, 10},
		{"fifty percent progress", 0.5, 30 * time.Second, 20},
		{"complete", 1.0, 0, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatProgressBarWithETA(tt.progress, tt.eta, tt.width)

			if !strings.Contains(got, "ETA:") {
				t.Errorf("formatProgressBarWithETA(%v, %v, %d) = %q, want it to contain %q", tt.progress, tt.eta, tt.width, got, "ETA:")
			}
			if !strings.Contains(got, "%") {
				t.Errorf("formatProgressBarWithETA(%v, %v, %d) = %q, want it to contain %q", tt.progress, tt.eta, tt.width, got, "%")
			}
			if !strings.Contains(got, "[") || !strings.Contains(got, "]") {
				t.Errorf("formatProgressBarWithETA(%v, %v, %d) = %q, want progress bar brackets", tt.progress, tt.eta, tt.width, got)
			}
		})
	}
}
