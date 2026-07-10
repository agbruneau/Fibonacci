// Package format_test exercises internal/format's exported API only. It is
// the black-box counterpart to progress_eta_test.go, which stays in package
// format because it must call the unexported formatProgressBarWithETA.
package format_test

import (
	"testing"
	"time"

	"github.com/agbruneau/FibGo/internal/format"
)

func TestFormatETA(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		eta  time.Duration
		want string
	}{
		{"zero duration", 0, "calculating..."},
		{"negative duration", -time.Second, "calculating..."},
		{"less than a second", 500 * time.Millisecond, "< 1s"},
		{"one second", time.Second, "1s"},
		{"multiple seconds", 45 * time.Second, "45s"},
		{"one minute", time.Minute, "1m"},
		{"minutes and seconds", 2*time.Minute + 30*time.Second, "2m30s"},
		{"one hour", time.Hour, "1h"},
		{"hours and minutes", time.Hour + 15*time.Minute, "1h15m"},
		{"multiple hours", 3*time.Hour + 45*time.Minute, "3h45m"},
		{"hours only, no minutes", 2 * time.Hour, "2h"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := format.FormatETA(tt.eta); got != tt.want {
				t.Errorf("FormatETA(%v) = %q, want %q", tt.eta, got, tt.want)
			}
		})
	}
}

func TestProgressBar(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		progress float64
		length   int
		want     string
	}{
		{"empty", 0.0, 10, "░░░░░░░░░░"},
		{"half", 0.5, 10, "█████░░░░░"},
		{"full", 1.0, 10, "██████████"},
		{"capped at 1.0", 1.2, 10, "██████████"},
		{"floored at 0.0", -0.1, 10, "░░░░░░░░░░"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := format.ProgressBar(tt.progress, tt.length); got != tt.want {
				t.Errorf("ProgressBar(%v, %d) = %q, want %q", tt.progress, tt.length, got, tt.want)
			}
		})
	}
}

func TestFormatExecutionDuration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"sub-microsecond rounds down", 500 * time.Nanosecond, "0µs"},
		{"microseconds", 10 * time.Microsecond, "10µs"},
		{"milliseconds", 10 * time.Millisecond, "10ms"},
		{"seconds", 2 * time.Second, "2s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := format.FormatExecutionDuration(tt.d); got != tt.want {
				t.Errorf("FormatExecutionDuration(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func TestFormatNumberString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"one digit", "1", "1"},
		{"two digits", "12", "12"},
		{"three digits", "123", "123"},
		{"four digits", "1234", "1,234"},
		{"six digits", "123456", "123,456"},
		{"seven digits", "1234567", "1,234,567"},
		{"negative", "-1234", "-1,234"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := format.FormatNumberString(tt.input); got != tt.want {
				t.Errorf("FormatNumberString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input uint64
		want  string
	}{
		{"zero bytes", 0, "0 B"},
		{"sub-KB", 512, "512 B"},
		{"exactly 1 KB", 1 << 10, "1.0 KB"},
		{"fractional KB", 1536, "1.5 KB"},
		{"exactly 1 MB", 1 << 20, "1.0 MB"},
		{"multi MB", 5 << 20, "5.0 MB"},
		{"exactly 1 GB", 1 << 30, "1.0 GB"},
		{"multi GB", 3 << 30, "3.0 GB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := format.FormatBytes(tt.input); got != tt.want {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
