package tui_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/agbruneau/FibGo/internal/tui"
)

func TestLogsModel_View(t *testing.T) {
	t.Parallel()
	logs := tui.NewLogsModel([]string{"Fast Doubling"})
	logs.SetSize(60, 20)

	logs.AddProgressEntry(tui.ProgressMsg{CalculatorIndex: 0, Value: 0.5})

	view := logs.View()
	if len(view) == 0 {
		t.Error("expected non-empty view")
	}
}

func TestLogsModel_Update_ScrollKeys(t *testing.T) {
	t.Parallel()
	logs := tui.NewLogsModel([]string{"Fast"})
	logs.SetSize(60, 10)

	// Add content
	for i := 0; i < 30; i++ {
		logs.AddProgressEntry(tui.ProgressMsg{CalculatorIndex: 0, Value: float64(i) / 30})
	}

	// Scroll up - should work without panic
	logs.Update(tea.KeyMsg{Type: tea.KeyUp})
	logs.Update(tea.KeyMsg{Type: tea.KeyDown})
	logs.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	logs.Update(tea.KeyMsg{Type: tea.KeyPgDown})
}
