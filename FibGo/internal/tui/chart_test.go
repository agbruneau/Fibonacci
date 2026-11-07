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

func TestChartModel_View(t *testing.T) {
	t.Parallel()
	chart := NewChartModel()
	chart.SetSize(50, 10)

	chart.AddDataPoint(0.3, 20*time.Second)
	chart.AddDataPoint(0.6, 10*time.Second)

	view := chart.View()
	if !strings.Contains(view, "Progress Chart") {
		t.Error("expected view to contain 'Progress Chart'")
	}
	if !strings.Contains(view, "ETA:") {
		t.Error("expected view to contain ETA")
	}
}

func TestChartModel_RenderProgressBar(t *testing.T) {
	t.Parallel()
	chart := NewChartModel()
	chart.SetSize(50, 10)
	chart.AddDataPoint(0.5, 10*time.Second)

	bar := chart.renderProgressBar()
	if !strings.Contains(bar, "█") {
		t.Error("expected progress bar to contain filled block character")
	}
	if !strings.Contains(bar, "░") {
		t.Error("expected progress bar to contain empty block character")
	}
	if !strings.Contains(bar, "50.0%") {
		t.Error("expected progress bar to show 50.0%")
	}
}

func TestChartModel_RenderProgressBar_Zero(t *testing.T) {
	t.Parallel()
	chart := NewChartModel()
	chart.SetSize(50, 10)
	chart.AddDataPoint(0.0, 0)

	bar := chart.renderProgressBar()
	if !strings.Contains(bar, "░") {
		t.Error("expected progress bar to contain empty blocks at 0%")
	}
	if !strings.Contains(bar, "0.0%") {
		t.Error("expected progress bar to show 0.0%")
	}
}

func TestChartModel_RenderProgressBar_Full(t *testing.T) {
	t.Parallel()
	chart := NewChartModel()
	chart.SetSize(50, 10)
	chart.AddDataPoint(1.0, 0)

	bar := chart.renderProgressBar()
	if !strings.Contains(bar, "█") {
		t.Error("expected progress bar to contain filled blocks at 100%")
	}
	if !strings.Contains(bar, "100.0%") {
		t.Error("expected progress bar to show 100.0%")
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

func TestChartModel_View_ContainsProgressBar(t *testing.T) {
	t.Parallel()
	chart := NewChartModel()
	chart.SetSize(50, 15)
	chart.AddDataPoint(0.65, 5*time.Second)

	view := chart.View()
	if !strings.Contains(view, "█") {
		t.Error("expected view to contain progress bar filled character")
	}
	if !strings.Contains(view, "65.0%") {
		t.Error("expected view to contain progress percentage")
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

func TestChartModel_View_ContainsSparklines(t *testing.T) {
	t.Parallel()
	chart := NewChartModel()
	chart.SetSize(50, 15) // height >= 8, braille chart visible

	chart.UpdateSysStats(50.0, 75.0)
	chart.UpdateSysStats(60.0, 80.0)

	view := chart.View()
	if !strings.Contains(view, "CPU") {
		t.Error("expected view to contain 'CPU' label in braille section")
	}
	if !strings.Contains(view, "MEM") {
		t.Error("expected view to contain 'MEM' sparkline label")
	}
}

func TestChartModel_View_HidesSparklines_SmallHeight(t *testing.T) {
	t.Parallel()
	chart := NewChartModel()
	chart.SetSize(50, 6) // height < 8, braille chart hidden

	chart.UpdateSysStats(50.0, 75.0)

	view := chart.View()
	if strings.Contains(view, "CPU") {
		t.Error("expected braille chart to be hidden for small height")
	}
}

func TestChartModel_View_Done(t *testing.T) {
	t.Parallel()
	chart := NewChartModel()
	chart.SetSize(50, 10)
	chart.SetDone(2 * time.Second)

	view := chart.View()
	if !strings.Contains(view, "Completed in 2s") {
		t.Errorf("expected done view to show the frozen total duration, got %q", view)
	}
	if strings.Contains(view, "ETA:") {
		t.Error("expected done view to drop the ETA header")
	}
	// SetDone must also force the bar to 100%.
	if !strings.Contains(view, "100.0%") {
		t.Error("expected done view to show a full progress bar")
	}
}

func TestChartModel_RenderProgressBar_ClampNegative(t *testing.T) {
	t.Parallel()
	chart := NewChartModel()
	chart.SetSize(50, 10)
	chart.AddDataPoint(-0.5, 0)

	bar := chart.renderProgressBar()
	if strings.Contains(bar, "█") {
		t.Error("expected no filled blocks for negative progress")
	}
	if !strings.Contains(bar, "░") {
		t.Error("expected a fully empty bar for negative progress")
	}
}

func TestChartModel_RenderProgressBar_ClampOverflow(t *testing.T) {
	t.Parallel()
	chart := NewChartModel()
	chart.SetSize(50, 10)
	chart.AddDataPoint(1.5, 0)

	bar := chart.renderProgressBar()
	if strings.Contains(bar, "░") {
		t.Error("expected no empty blocks when progress overflows 100%")
	}
	if !strings.Contains(bar, "█") {
		t.Error("expected a fully filled bar when progress overflows 100%")
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
