package tui_test

import (
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/FibGo/internal/tui"
)

func TestChartModel_View(t *testing.T) {
	t.Parallel()
	chart := tui.NewChartModel()
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

func TestChartModel_View_ContainsProgressBar(t *testing.T) {
	t.Parallel()
	chart := tui.NewChartModel()
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

func TestChartModel_View_ContainsSparklines(t *testing.T) {
	t.Parallel()
	chart := tui.NewChartModel()
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
	chart := tui.NewChartModel()
	chart.SetSize(50, 6) // height < 8, braille chart hidden

	chart.UpdateSysStats(50.0, 75.0)

	view := chart.View()
	if strings.Contains(view, "CPU") {
		t.Error("expected braille chart to be hidden for small height")
	}
}

func TestChartModel_View_Done(t *testing.T) {
	t.Parallel()
	chart := tui.NewChartModel()
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
