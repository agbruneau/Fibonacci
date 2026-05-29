package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/FibGo/internal/format"
)

func TestMetricsModel_UpdateMemStats(t *testing.T) {
	t.Parallel()
	m := NewMetricsModel()

	msg := MemStatsMsg{
		Alloc:        1024 * 1024 * 50, // 50 MB
		NumGC:        10,
		NumGoroutine: 8,
	}
	m.UpdateMemStats(msg)

	if m.alloc != msg.Alloc {
		t.Errorf("expected alloc %d, got %d", msg.Alloc, m.alloc)
	}
	if m.numGC != msg.NumGC {
		t.Errorf("expected numGC %d, got %d", msg.NumGC, m.numGC)
	}
	if m.numGoroutine != msg.NumGoroutine {
		t.Errorf("expected numGoroutine %d, got %d", msg.NumGoroutine, m.numGoroutine)
	}
}

func TestMetricsModel_UpdateProgress(t *testing.T) {
	t.Parallel()
	m := NewMetricsModel()
	// Force the lastUpdate back in time to ensure dt > 0.05
	m.lastUpdate = time.Now().Add(-1 * time.Second)

	m.UpdateProgress(0.5)
	if m.speed <= 0 {
		t.Error("expected positive speed after progress update")
	}
	if m.lastProgress != 0.5 {
		t.Errorf("expected lastProgress 0.5, got %f", m.lastProgress)
	}
}

func TestMetricsModel_UpdateProgress_Smoothing(t *testing.T) {
	t.Parallel()
	m := NewMetricsModel()
	m.lastUpdate = time.Now().Add(-1 * time.Second)

	// First update: dp=0.3 over ~1s → speed ≈ 0.3
	m.UpdateProgress(0.3)
	firstSpeed := m.speed

	if firstSpeed <= 0 {
		t.Fatal("precondition: first speed should be positive")
	}

	// Second update: dp=0.5 over ~0.5s → instant speed ≈ 1.0
	// Smoothed: 0.7*0.3 + 0.3*1.0 = 0.51 ≠ 0.3
	m.lastUpdate = time.Now().Add(-500 * time.Millisecond)
	m.UpdateProgress(0.8)

	if m.speed <= 0 {
		t.Error("expected positive speed after second update")
	}
	if m.speed == firstSpeed {
		t.Error("expected speed to change after second update with different rate")
	}
}

func TestMetricsModel_UpdateProgress_TooFast(t *testing.T) {
	t.Parallel()
	m := NewMetricsModel()
	// lastUpdate is now, so dt < 0.05 — should not update speed
	m.UpdateProgress(0.5)

	if m.speed != 0 {
		t.Errorf("expected speed to remain 0 when dt < 0.05, got %f", m.speed)
	}
}

func TestMetricsModel_UpdateProgress_NoForward(t *testing.T) {
	t.Parallel()
	m := NewMetricsModel()
	m.lastUpdate = time.Now().Add(-1 * time.Second)
	m.lastProgress = 0.5

	// Same progress (dp = 0) should not update speed
	m.UpdateProgress(0.5)

	if m.speed != 0 {
		t.Errorf("expected speed to remain 0 when no forward progress, got %f", m.speed)
	}
}

func TestMetricsModel_View(t *testing.T) {
	t.Parallel()
	m := NewMetricsModel()
	m.SetSize(40, 15)

	m.UpdateMemStats(MemStatsMsg{
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
		tt := tt
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := format.FormatBytes(tt.input)
			if !strings.Contains(got, tt.contains) {
				t.Errorf("format.FormatBytes(%d) = %q, want to contain %q", tt.input, got, tt.contains)
			}
		})
	}
}

func TestMetricsModel_UpdateProgress_RapidUpdates(t *testing.T) {
	t.Parallel()
	m := NewMetricsModel()
	m.lastUpdate = time.Now().Add(-1 * time.Second)

	// 1000 rapid updates with increasing progress
	for i := 0; i < 1000; i++ {
		m.lastUpdate = time.Now().Add(-100 * time.Millisecond)
		m.UpdateProgress(float64(i) / 1000.0)
	}

	if m.speed <= 0 {
		t.Error("expected positive speed after many updates")
	}
	if m.lastProgress == 0 {
		t.Error("expected non-zero lastProgress after many updates")
	}
}

func TestMetricsModel_SetSize(t *testing.T) {
	t.Parallel()
	m := NewMetricsModel()
	m.SetSize(50, 20)

	if m.width != 50 {
		t.Errorf("expected width 50, got %d", m.width)
	}
	if m.height != 20 {
		t.Errorf("expected height 20, got %d", m.height)
	}
}

func TestFormatMetricCol(t *testing.T) {
	t.Parallel()
	col := formatMetricCol("Memory:", "50.0 MB", 30)
	if !strings.Contains(col, "Memory") {
		t.Error("expected column to contain label")
	}
	if !strings.Contains(col, "50.0 MB") {
		t.Error("expected column to contain value")
	}
}
