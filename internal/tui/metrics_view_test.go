package tui_test

import (
	"strings"
	"testing"

	"github.com/agbruneau/FibGo/internal/format"
	"github.com/agbruneau/FibGo/internal/metrics"
	"github.com/agbruneau/FibGo/internal/tui"
)

func TestMetricsModel_View(t *testing.T) {
	t.Parallel()
	m := tui.NewMetricsModel()
	m.SetSize(40, 15)

	m.UpdateMemStats(tui.MemStatsMsg{
		Alloc:        1024 * 1024 * 50,
		NumGC:        10,
		NumGoroutine: 8,
	})

	view := m.View()
	if !strings.Contains(view, "Heap") {
		t.Error("expected view to contain 'Heap' label")
	}
	if !strings.Contains(view, "GC") {
		t.Error("expected view to contain 'GC' label")
	}
	if !strings.Contains(view, "Speed") {
		t.Error("expected view to contain 'Speed' label")
	}
	if !strings.Contains(view, "Goroutines") {
		t.Error("expected view to contain 'Goroutines' label")
	}
}

func TestMetricsModel_View_WithIndicators(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		isEven bool
		parity string
	}{
		{"even result", true, "even"},
		{"odd result", false, "odd"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := tui.NewMetricsModel()
			m.SetSize(60, 12)
			m.UpdateIndicators(&metrics.Indicators{
				BitsPerSecond:   1.5e6,
				DigitsPerSecond: 4.2e5,
				DoublingSteps:   20,
				StepsPerSecond:  3.5,
				IsEven:          tt.isEven,
			})

			view := m.View()
			for _, label := range []string{"Bits/s:", "Steps:", "Digits/s:", "Parity:"} {
				if !strings.Contains(view, label) {
					t.Errorf("expected view with indicators to contain %q", label)
				}
			}
			if !strings.Contains(view, tt.parity) {
				t.Errorf("expected parity %q in view", tt.parity)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    uint64
		contains string
	}{
		{"bytes", 512, "512 B"},
		{"kilobytes", 1024 * 5, "5.0 KB"},
		{"megabytes", 1024 * 1024 * 50, "50.0 MB"},
		{"gigabytes", 1024 * 1024 * 1024 * 2, "2.0 GB"},
		{"zero", 0, "0 B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := format.FormatBytes(tt.input)
			if !strings.Contains(got, tt.contains) {
				t.Errorf("format.FormatBytes(%d) = %q, want to contain %q", tt.input, got, tt.contains)
			}
		})
	}
}

func TestFormatBytes_Boundaries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    uint64
		contains string
	}{
		{"exact_1KB", 1024, "1.0 KB"},
		{"exact_1MB", 1024 * 1024, "1.0 MB"},
		{"exact_1GB", 1024 * 1024 * 1024, "1.0 GB"},
		{"just_below_KB", 1023, "1023 B"},
		{"just_below_MB", 1024*1024 - 1, "KB"},
		{"just_below_GB", 1024*1024*1024 - 1, "MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := format.FormatBytes(tt.input)
			if !strings.Contains(got, tt.contains) {
				t.Errorf("format.FormatBytes(%d) = %q, want to contain %q", tt.input, got, tt.contains)
			}
		})
	}
}
