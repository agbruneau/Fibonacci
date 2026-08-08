package tui

import (
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/agbruneau/FibGo/internal/orchestration"
)

func TestLogsModel_AddProgressEntry(t *testing.T) {
	t.Parallel()
	logs := NewLogsModel([]string{"Fast Doubling", "Matrix"})
	logs.SetSize(60, 20)

	logs.AddProgressEntry(ProgressMsg{
		CalculatorIndex: 0,
		Value:           0.5,
		AverageProgress: 0.5,
		ETA:             10 * time.Second,
	})

	if len(logs.entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(logs.entries))
	}
	if !strings.Contains(logs.entries[0], "Fast Doubling") {
		t.Error("expected entry to contain algorithm name")
	}
	if !strings.Contains(logs.entries[0], "50.0%") {
		t.Error("expected entry to contain progress percentage")
	}
}

func TestLogsModel_AddProgressEntry_Complete(t *testing.T) {
	t.Parallel()
	logs := NewLogsModel([]string{"Fast Doubling"})
	logs.SetSize(60, 20)

	logs.AddProgressEntry(ProgressMsg{
		CalculatorIndex: 0,
		Value:           1.0,
		AverageProgress: 1.0,
		ETA:             0,
	})

	if !strings.Contains(logs.entries[0], "100% OK") {
		t.Errorf("expected '100%% OK' for completed progress, got %q", logs.entries[0])
	}
}

func TestLogsModel_AddResults(t *testing.T) {
	t.Parallel()
	logs := NewLogsModel([]string{"Fast Doubling", "Matrix"})
	logs.SetSize(60, 20)

	results := []orchestration.CalculationResult{
		{Name: "Fast Doubling", Result: big.NewInt(55), Duration: 100 * time.Millisecond},
		{Name: "Matrix", Result: big.NewInt(55), Duration: 200 * time.Millisecond},
	}
	logs.AddResults(results)

	// Should have separator + header + 2 result lines
	if len(logs.entries) < 4 {
		t.Errorf("expected at least 4 entries, got %d", len(logs.entries))
	}

	joined := strings.Join(logs.entries, "\n")
	if !strings.Contains(joined, "Comparison Summary") {
		t.Error("expected 'Comparison Summary' header")
	}
	if !strings.Contains(joined, "Fast Doubling") {
		t.Error("expected 'Fast Doubling' in results")
	}
	if !strings.Contains(joined, "OK") {
		t.Error("expected 'OK' status for successful results")
	}
}

func TestLogsModel_AddResults_WithError(t *testing.T) {
	t.Parallel()
	logs := NewLogsModel([]string{"Fast Doubling"})
	logs.SetSize(60, 20)

	results := []orchestration.CalculationResult{
		{Name: "Fast Doubling", Err: errors.New("timeout"), Duration: time.Second},
	}
	logs.AddResults(results)

	joined := strings.Join(logs.entries, "\n")
	if !strings.Contains(joined, "FAIL") {
		t.Error("expected 'FAIL' for errored result")
	}
}

func TestLogsModel_AddFinalResult(t *testing.T) {
	t.Parallel()
	logs := NewLogsModel([]string{"Fast Doubling"})
	logs.SetSize(60, 20)

	logs.AddFinalResult(FinalResultMsg{
		Result: orchestration.CalculationResult{
			Name:     "Fast Doubling",
			Result:   big.NewInt(55),
			Duration: 100 * time.Millisecond,
		},
		N:       10,
		Verbose: true,
	})

	joined := strings.Join(logs.entries, "\n")
	if !strings.Contains(joined, "Final Result") {
		t.Error("expected 'Final Result' header")
	}
	if !strings.Contains(joined, "Fast Doubling") {
		t.Error("expected algorithm name in final result")
	}
	if !strings.Contains(joined, "Bits") {
		t.Error("expected bit count in final result")
	}
}

func TestLogsModel_AddFinalResult_NilResult(t *testing.T) {
	t.Parallel()
	logs := NewLogsModel([]string{"Fast Doubling"})
	logs.SetSize(60, 20)

	logs.AddFinalResult(FinalResultMsg{
		Result: orchestration.CalculationResult{
			Name:     "Fast Doubling",
			Result:   nil,
			Duration: 100 * time.Millisecond,
		},
		N: 10,
	})

	joined := strings.Join(logs.entries, "\n")
	// Should not contain Bits line when Result is nil
	if strings.Contains(joined, "Bits") {
		t.Error("expected no 'Bits' line when Result is nil")
	}
}

func TestLogsModel_AddError(t *testing.T) {
	t.Parallel()
	logs := NewLogsModel([]string{})
	logs.SetSize(60, 20)

	logs.AddError(ErrorMsg{
		Err:      errors.New("calculation failed"),
		Duration: 5 * time.Second,
	})

	if len(logs.entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(logs.entries))
	}
	if !strings.Contains(logs.entries[0], "ERROR") {
		t.Error("expected entry to contain 'ERROR'")
	}
	if !strings.Contains(logs.entries[0], "calculation failed") {
		t.Error("expected entry to contain error message")
	}
}

func TestLogsModel_AlgoName_OutOfBounds(t *testing.T) {
	t.Parallel()
	logs := NewLogsModel([]string{"Fast Doubling"})

	// Valid index
	if name := logs.algoName(0); name != "Fast Doubling" {
		t.Errorf("expected 'Fast Doubling', got %q", name)
	}

	// Out of bounds
	if name := logs.algoName(5); name != "Algo 5" {
		t.Errorf("expected fallback 'Algo 5', got %q", name)
	}

	// Negative index
	if name := logs.algoName(-1); name != "Unknown" {
		t.Errorf("expected 'Unknown' for negative index, got %q", name)
	}
}

func TestLogsModel_View(t *testing.T) {
	t.Parallel()
	logs := NewLogsModel([]string{"Fast Doubling"})
	logs.SetSize(60, 20)

	logs.AddProgressEntry(ProgressMsg{CalculatorIndex: 0, Value: 0.5})

	view := logs.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestLogsModel_AutoScroll(t *testing.T) {
	t.Parallel()
	logs := NewLogsModel([]string{"Fast"})
	logs.SetSize(60, 10)

	// Add many entries to overflow viewport
	for i := 0; i < 50; i++ {
		logs.AddProgressEntry(ProgressMsg{CalculatorIndex: 0, Value: float64(i) / 50})
	}

	// autoScroll should still be true (GotoBottom called)
	if !logs.autoScroll {
		t.Error("expected autoScroll to be true after adding entries")
	}
}

func TestLogsModel_Update_ScrollKeys(t *testing.T) {
	t.Parallel()
	logs := NewLogsModel([]string{"Fast"})
	logs.SetSize(60, 10)

	// Add content
	for i := 0; i < 30; i++ {
		logs.AddProgressEntry(ProgressMsg{CalculatorIndex: 0, Value: float64(i) / 30})
	}

	// Scroll up - should work without panic
	logs.Update(tea.KeyMsg{Type: tea.KeyUp})
	logs.Update(tea.KeyMsg{Type: tea.KeyDown})
	logs.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	logs.Update(tea.KeyMsg{Type: tea.KeyPgDown})
}

func TestLogsModel_AddProgressEntry_BoundedGrowth(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	logs := NewLogsModel([]string{"Fast"})
	logs.SetSize(60, 20)

	// Add more entries than the limit
	for i := 0; i < maxLogEntries+100; i++ {
		logs.AddProgressEntry(ProgressMsg{CalculatorIndex: 0, Value: float64(i%100) / 100})
	}

	if len(logs.entries) > maxLogEntries {
		t.Errorf("expected entries to be bounded at %d, got %d", maxLogEntries, len(logs.entries))
	}
}

func TestLogsModel_algoName_NegativeProducesUnknown(t *testing.T) {
	t.Parallel()
	logs := NewLogsModel([]string{"Fast Doubling", "Matrix"})

	tests := []struct {
		index    int
		expected string
	}{
		{-1, "Unknown"},
		{-100, "Unknown"},
		{0, "Fast Doubling"},
		{1, "Matrix"},
		{2, "Algo 2"},
		{99, "Algo 99"},
	}

	for _, tt := range tests {
		got := logs.algoName(tt.index)
		if got != tt.expected {
			t.Errorf("algoName(%d) = %q, want %q", tt.index, got, tt.expected)
		}
	}
}

func TestLogsModel_AutoScroll_DisablesOnScrollUp(t *testing.T) {
	t.Parallel()
	logs := NewLogsModel([]string{"Fast"})
	logs.SetSize(60, 5)

	// Add many entries to overflow the viewport. With height=5 the viewport
	// body is 3 lines, so any content > 3 lines exceeds it — 100 entries is
	// more than enough.
	for i := 0; i < 100; i++ {
		logs.AddProgressEntry(ProgressMsg{CalculatorIndex: 0, Value: float64(i) / 100})
	}

	if !logs.autoScroll {
		t.Fatal("precondition: autoScroll should be true")
	}

	// Force the viewport into "overflowing content" state. GotoBottom in
	// updateContent may not always pin AtBottom() in a deterministic way
	// when the underlying bubbles viewport has not yet measured line counts
	// (e.g. when only the body width is known at call time). Running a
	// no-op Update after we've queued 100 entries lets the model commit its
	// measured state, which is the precondition of the scroll-up check.
	logs.viewport.GotoBottom()
	if !logs.viewport.AtBottom() {
		t.Fatal("precondition: viewport should be at bottom after GotoBottom")
	}

	// Now scroll up one page.
	logs.Update(tea.KeyMsg{Type: tea.KeyPgUp})

	// After PgUp on overflowing content, AtBottom must become false and
	// autoScroll must be disabled. Both assertions replace the previous
	// t.Skip escape hatch (P3-01): the test is now deterministic.
	if logs.viewport.AtBottom() {
		t.Fatal("viewport still AtBottom after PgUp; content should overflow height=5")
	}
	if logs.autoScroll {
		t.Error("expected autoScroll to be false after scrolling up")
	}
}

func TestLogsModel_AddProgressEntry_StressTest(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	logs := NewLogsModel([]string{"Fast"})
	logs.SetSize(60, 20)

	// 10000 rapid updates
	for i := 0; i < 10000; i++ {
		logs.AddProgressEntry(ProgressMsg{CalculatorIndex: 0, Value: float64(i%100) / 100})
	}

	if len(logs.entries) > maxLogEntries {
		t.Errorf("entries exceeded max: %d > %d", len(logs.entries), maxLogEntries)
	}
}

func TestLogsModel_SetSize(t *testing.T) {
	t.Parallel()
	logs := NewLogsModel([]string{"Fast"})
	logs.SetSize(80, 30)

	if logs.width != 80 {
		t.Errorf("expected width 80, got %d", logs.width)
	}
	if logs.height != 30 {
		t.Errorf("expected height 30, got %d", logs.height)
	}
	// viewport dimensions should be inner (minus borders)
	if logs.viewport.Width != 78 {
		t.Errorf("expected viewport width 78, got %d", logs.viewport.Width)
	}
	if logs.viewport.Height != 28 {
		t.Errorf("expected viewport height 28, got %d", logs.viewport.Height)
	}
}
