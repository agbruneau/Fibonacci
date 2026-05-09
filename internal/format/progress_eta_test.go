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

// TestFormatProgressBarWithETA verifies combined progress and ETA formatting.
func TestFormatProgressBarWithETA(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name        string
		progress    float64
		eta         time.Duration
		width       int
		containsETA bool
		containsPct bool
	}{
		{
			name:        "Zero progress",
			progress:    0,
			eta:         time.Minute,
			width:       10,
			containsETA: true,
			containsPct: true,
		},
		{
			name:        "50% progress",
			progress:    0.5,
			eta:         30 * time.Second,
			width:       20,
			containsETA: true,
			containsPct: true,
		},
		{
			name:        "Complete",
			progress:    1.0,
			eta:         0,
			width:       10,
			containsETA: true,
			containsPct: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := FormatProgressBarWithETA(tc.progress, tc.eta, tc.width)

			if tc.containsETA {
				if !contains(result, "ETA:") {
					t.Errorf("FormatProgressBarWithETA result should contain 'ETA:', got %q", result)
				}
			}
			if tc.containsPct {
				if !contains(result, "%") {
					t.Errorf("FormatProgressBarWithETA result should contain '%%', got %q", result)
				}
			}
			// Should contain progress bar characters
			if !contains(result, "[") || !contains(result, "]") {
				t.Errorf("FormatProgressBarWithETA result should contain progress bar brackets, got %q", result)
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

// TestNewProgressState verifies ProgressState initialization.
func TestNewProgressState(t *testing.T) {
	t.Parallel()
	ps := NewProgressState(3)
	if ps.numCalculators != 3 {
		t.Errorf("numCalculators = %d, want 3", ps.numCalculators)
	}
	if len(ps.progresses) != 3 {
		t.Errorf("progresses length = %d, want 3", len(ps.progresses))
	}
	avg := ps.CalculateAverage()
	if avg != 0 {
		t.Errorf("initial average = %f, want 0", avg)
	}
}

// TestProgressStateUpdate verifies progress updates.
func TestProgressStateUpdate(t *testing.T) {
	t.Parallel()
	ps := NewProgressState(2)
	ps.Update(0, 0.5)
	ps.Update(1, 1.0)
	avg := ps.CalculateAverage()
	if avg != 0.75 {
		t.Errorf("average = %f, want 0.75", avg)
	}
}

// TestProgressStateZeroCalculators verifies edge case with zero calculators.
func TestProgressStateZeroCalculators(t *testing.T) {
	t.Parallel()
	ps := NewProgressState(0)
	avg := ps.CalculateAverage()
	if avg != 0 {
		t.Errorf("average = %f, want 0", avg)
	}
}

// contains is a helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
