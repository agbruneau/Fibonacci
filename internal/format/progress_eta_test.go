package format

import (
	"testing"
	"time"
)

// TestFormatETA verifies ETA formatting.
func TestFormatETA(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		eta      time.Duration
		expected string
	}{
		{"Zero duration", 0, "calculating..."},
		{"Negative duration", -time.Second, "calculating..."},
		{"Less than a second", 500 * time.Millisecond, "< 1s"},
		{"One second", time.Second, "1s"},
		{"Multiple seconds", 45 * time.Second, "45s"},
		{"One minute", time.Minute, "1m"},
		{"Minutes and seconds", 2*time.Minute + 30*time.Second, "2m30s"},
		{"One hour", time.Hour, "1h"},
		{"Hours and minutes", time.Hour + 15*time.Minute, "1h15m"},
		{"Multiple hours", 3*time.Hour + 45*time.Minute, "3h45m"},
		{"Hours only (no minutes)", 2 * time.Hour, "2h"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := FormatETA(tc.eta)
			if result != tc.expected {
				t.Errorf("FormatETA(%v) = %q, want %q", tc.eta, result, tc.expected)
			}
		})
	}
}

// TestProgressBar verifies progress bar rendering.
func TestProgressBar(t *testing.T) {
	t.Parallel()
	tests := []struct {
		progress float64
		length   int
		expected string
	}{
		{0.0, 10, "░░░░░░░░░░"},
		{0.5, 10, "█████░░░░░"},
		{1.0, 10, "██████████"},
		{1.2, 10, "██████████"},  // Cap at 1.0
		{-0.1, 10, "░░░░░░░░░░"}, // Floor at 0.0
	}

	for _, tt := range tests {
		got := ProgressBar(tt.progress, tt.length)
		if got != tt.expected {
			t.Errorf("ProgressBar(%f, %d) = %s; want %s", tt.progress, tt.length, got, tt.expected)
		}
	}
}

// TestFormatExecutionDuration verifies duration formatting.
func TestFormatExecutionDuration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		d        time.Duration
		expected string
	}{
		{500 * time.Nanosecond, "0µs"},
		{10 * time.Microsecond, "10µs"},
		{10 * time.Millisecond, "10ms"},
		{2 * time.Second, "2s"},
	}

	for _, tt := range tests {
		got := FormatExecutionDuration(tt.d)
		if got != tt.expected {
			t.Errorf("FormatExecutionDuration(%v) = %s; want %s", tt.d, got, tt.expected)
		}
	}
}

// TestFormatNumberString verifies thousand separator formatting.
func TestFormatNumberString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"1", "1"},
		{"12", "12"},
		{"123", "123"},
		{"1234", "1,234"},
		{"123456", "123,456"},
		{"1234567", "1,234,567"},
		{"-1234", "-1,234"},
	}

	for _, tt := range tests {
		got := FormatNumberString(tt.input)
		if got != tt.expected {
			t.Errorf("FormatNumberString(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

// TestFormatBytes verifies human-readable byte formatting across all four
// magnitude branches. This was the only format helper without a unit test
// in this package (it was previously exercised only indirectly through TUI
// rendering).
func TestFormatBytes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1 << 10, "1.0 KB"},
		{1536, "1.5 KB"},
		{1 << 20, "1.0 MB"},
		{5 << 20, "5.0 MB"},
		{1 << 30, "1.0 GB"},
		{3 << 30, "3.0 GB"},
	}

	for _, tt := range tests {
		got := FormatBytes(tt.input)
		if got != tt.expected {
			t.Errorf("FormatBytes(%d) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}
