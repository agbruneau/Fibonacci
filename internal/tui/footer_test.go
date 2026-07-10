package tui_test

import (
	"strings"
	"testing"

	"github.com/agbruneau/FibGo/internal/tui"
)

func TestFooterModel_View_Status(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		paused       bool
		done         bool
		hasErr       bool
		wantContains string
	}{
		{"Running by default", false, false, false, "Running"},
		{"Paused", true, false, false, "Paused"},
		{"Done", false, true, false, "Done"},
		{"Error takes precedence over Done", false, true, true, "Error"},
		{"Error takes precedence over Done and Paused", true, true, true, "Error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := tui.NewFooterModel()
			f.SetWidth(80)
			f.SetPaused(tt.paused)
			f.SetDone(tt.done)
			f.SetError(tt.hasErr)

			view := f.View()
			if !strings.Contains(view, tt.wantContains) {
				t.Errorf("expected footer to show %q status, got %q", tt.wantContains, view)
			}
		})
	}
}

func TestFooterModel_View_Shortcuts(t *testing.T) {
	t.Parallel()
	f := tui.NewFooterModel()
	f.SetWidth(120)

	view := f.View()
	if !strings.Contains(view, "Quit") {
		t.Error("expected footer to contain 'Quit' shortcut")
	}
	if !strings.Contains(view, "Restart") {
		t.Error("expected footer to contain 'Restart' shortcut")
	}
	if !strings.Contains(view, "Pause") {
		t.Error("expected footer to contain 'Pause' shortcut")
	}
}

func TestFooterModel_View_EdgeWidths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		width int
	}{
		{"Narrow width", 5},
		{"Negative width", -1},
		{"Zero width", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := tui.NewFooterModel()
			f.SetWidth(tt.width)

			// Should not panic.
			view := f.View()
			if len(view) == 0 {
				t.Errorf("expected non-empty view for width %d", tt.width)
			}
		})
	}
}
