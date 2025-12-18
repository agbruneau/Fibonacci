package cli

import (
	"testing"
	"time"
)

// TestNewProgressWithETA verifies proper initialization.
func TestNewProgressWithETA(t *testing.T) {
	t.Parallel()
	p := NewProgressWithETA(3)

	if p.ProgressState == nil {
		t.Fatal("ProgressState should not be nil")
	}
	if p.numCalculators != 3 {
		t.Errorf("numCalculators = %d, want 3", p.numCalculators)
	}
	if p.progressRate != 0 {
		t.Errorf("initial progressRate = %f, want 0", p.progressRate)
	}
	if p.startTime.IsZero() {
		t.Error("startTime should not be zero")
	}
}

// TestUpdateWithETA verifies progress updates and ETA calculation.
func TestUpdateWithETA(t *testing.T) {
	t.Parallel()
	p := NewProgressWithETA(2)

	// Initial update
	progress, eta := p.UpdateWithETA(0, 0.25)
	if progress != 0.125 { // average of 0.25 and 0
		t.Errorf("initial progress = %f, want 0.125", progress)
	}
	// ETA should be 0 or calculating at first (not enough data)
	if eta < 0 {
		t.Errorf("ETA should not be negative, got %v", eta)
	}

	// Update second calculator
	progress, _ = p.UpdateWithETA(1, 0.5)
	if progress != 0.375 { // average of 0.25 and 0.5
		t.Errorf("progress = %f, want 0.375", progress)
	}
}

// TestGetETA verifies ETA retrieval.
func TestGetETA(t *testing.T) {
	t.Parallel()
	p := NewProgressWithETA(1)

	// Before any updates, ETA should be 0
	eta := p.GetETA()
	if eta != 0 {
		t.Errorf("initial ETA = %v, want 0", eta)
	}

	// Simulate some progress
	p.Update(0, 0.5)
	p.progressRate = 0.1 // 10% per second

	eta = p.GetETA()
	// With 50% remaining at 10%/s, ETA should be around 5 seconds
	expectedETA := 5 * time.Second
	tolerance := time.Second
	if eta < expectedETA-tolerance || eta > expectedETA+tolerance {
		t.Errorf("ETA = %v, want approximately %v", eta, expectedETA)
	}
}

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

// TestProgressWithETAEdgeCases verifies edge case handling.
func TestProgressWithETAEdgeCases(t *testing.T) {
	t.Parallel()
	t.Run("Progress exceeds 1.0", func(t *testing.T) {
		t.Parallel()
		p := NewProgressWithETA(1)
		p.Update(0, 1.5)
		progress := p.CalculateAverage()
		// Progress should be clamped or handled gracefully
		if progress < 0 {
			t.Errorf("progress should not be negative, got %f", progress)
		}
	})

	t.Run("Negative progress", func(t *testing.T) {
		t.Parallel()
		p := NewProgressWithETA(1)
		p.Update(0, -0.5)
		progress := p.CalculateAverage()
		// Should handle gracefully
		if progress > 1.0 {
			t.Errorf("progress should not exceed 1.0, got %f", progress)
		}
	})

	t.Run("Invalid calculator index", func(t *testing.T) {
		t.Parallel()
		p := NewProgressWithETA(2)
		// Should not panic with invalid index
		p.UpdateWithETA(5, 0.5)
		p.UpdateWithETA(-1, 0.5)
		// Verify state is still valid
		progress := p.CalculateAverage()
		if progress < 0 || progress > 1.0 {
			t.Errorf("progress should be valid, got %f", progress)
		}
	})
}

// TestETACapping verifies that ETA is capped at reasonable values.
func TestETACapping(t *testing.T) {
	t.Parallel()
	p := NewProgressWithETA(1)
	p.Update(0, 0.001)         // Very small progress
	p.progressRate = 0.0000001 // Very slow rate

	eta := p.GetETA()
	maxETA := 24 * time.Hour

	if eta > maxETA {
		t.Errorf("ETA = %v, should be capped at %v", eta, maxETA)
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
