package tui

import (
	"strings"
	"testing"
	"time"
)

func TestChartModel_AddDataPoint(t *testing.T) {
	t.Parallel()
	chart := NewChartModel()
	chart.SetSize(50, 10)

	chart.AddDataPoint(0.25, 30*time.Second)
	chart.AddDataPoint(0.50, 20*time.Second)
	chart.AddDataPoint(0.75, 10*time.Second)

	if chart.averageProgress != 0.75 {
		t.Errorf("expected average 0.75, got %f", chart.averageProgress)
	}
}

func TestChartModel_Reset(t *testing.T) {
	t.Parallel()
	chart := NewChartModel()
	chart.AddDataPoint(0.5, 10*time.Second)
	chart.AddDataPoint(0.8, 5*time.Second)
	chart.UpdateSysStats(25.0, 60.0)

	chart.Reset()

	if chart.averageProgress != 0 {
		t.Errorf("expected 0 average after reset, got %f", chart.averageProgress)
	}
	if chart.cpuHistory.Len() != 0 {
		t.Error("expected cpuHistory to be empty after reset")
	}
	if chart.memHistory.Len() != 0 {
		t.Error("expected memHistory to be empty after reset")
	}
}

// TestChartModel_RenderProgressBar covers the progress-driven shape of the
// bar: filled/empty block presence, plus the percentage label where it is
// meaningful. renderProgressBar clamps the filled/empty *block count* to
// [0,barWidth], but the percentage label itself is NOT clamped (it renders
// raw averageProgress*100) — wantPercent is left empty for the two clamp
// cases to avoid asserting that unclamped-label behavior here.
func TestChartModel_RenderProgressBar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		progress    float64
		wantFilled  bool
		wantEmpty   bool
		wantPercent string
	}{
		{"50 percent", 0.5, true, true, "50.0%"},
		{"zero percent", 0.0, false, true, "0.0%"},
		{"full", 1.0, true, false, "100.0%"},
		{"clamp negative", -0.5, false, true, ""},
		{"clamp overflow", 1.5, true, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			chart := NewChartModel()
			chart.SetSize(50, 10)
			chart.AddDataPoint(tt.progress, 0)

			bar := chart.renderProgressBar()
			if got := strings.Contains(bar, "█"); got != tt.wantFilled {
				t.Errorf("filled block present = %v, want %v (bar=%q)", got, tt.wantFilled, bar)
			}
			if got := strings.Contains(bar, "░"); got != tt.wantEmpty {
				t.Errorf("empty block present = %v, want %v (bar=%q)", got, tt.wantEmpty, bar)
			}
			if tt.wantPercent != "" && !strings.Contains(bar, tt.wantPercent) {
				t.Errorf("expected bar to show %q, got %q", tt.wantPercent, bar)
			}
		})
	}
}

func TestChartModel_RenderProgressBar_TooNarrow(t *testing.T) {
	t.Parallel()
	chart := NewChartModel()
	chart.SetSize(10, 5) // too narrow for a progress bar

	bar := chart.renderProgressBar()
	if bar != "" {
		t.Error("expected empty progress bar for very narrow chart")
	}
}

func TestChartModel_UpdateSysStats(t *testing.T) {
	t.Parallel()
	chart := NewChartModel()
	chart.SetSize(50, 15)

	chart.UpdateSysStats(25.0, 60.0)
	chart.UpdateSysStats(30.0, 62.0)

	if chart.cpuHistory.Len() != 2 {
		t.Errorf("expected 2 cpu samples, got %d", chart.cpuHistory.Len())
	}
	if chart.memHistory.Len() != 2 {
		t.Errorf("expected 2 mem samples, got %d", chart.memHistory.Len())
	}
	if chart.cpuHistory.Last() != 30.0 {
		t.Errorf("expected last cpu 30.0, got %f", chart.cpuHistory.Last())
	}
	if chart.memHistory.Last() != 62.0 {
		t.Errorf("expected last mem 62.0, got %f", chart.memHistory.Last())
	}
}

func TestChartModel_SetSize_ResizesBuffers(t *testing.T) {
	t.Parallel()
	chart := NewChartModel()
	chart.SetSize(50, 15)

	expectedWidth := 50 - 18 // sparklineWidth
	if chart.cpuHistory.Cap() != expectedWidth {
		t.Errorf("expected cpu buffer cap %d, got %d", expectedWidth, chart.cpuHistory.Cap())
	}
	if chart.memHistory.Cap() != expectedWidth {
		t.Errorf("expected mem buffer cap %d, got %d", expectedWidth, chart.memHistory.Cap())
	}
}
