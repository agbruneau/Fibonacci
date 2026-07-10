package tui_test

import (
	"strings"
	"testing"

	"github.com/agbruneau/FibGo/internal/tui"
)

func TestHeaderModel_View(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		version      string
		wantContains string
	}{
		{"contains title", "v1.0.0", "FibGo Monitor"},
		{"contains version", "v2.3.4", "v2.3.4"},
		{"contains elapsed", "v1.0.0", "Elapsed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := tui.NewHeaderModel(tt.version)
			h.SetWidth(80)

			view := h.View()
			if !strings.Contains(view, tt.wantContains) {
				t.Errorf("expected header to contain %q, got %q", tt.wantContains, view)
			}
		})
	}
}

func TestHeaderModel_View_NarrowWidth(t *testing.T) {
	t.Parallel()
	h := tui.NewHeaderModel("v1.0.0")
	h.SetWidth(10)

	// Should not panic even with very narrow width
	view := h.View()
	if len(view) == 0 {
		t.Error("expected non-empty view even with narrow width")
	}
}

func TestHeaderModel_View_ZeroWidth(t *testing.T) {
	t.Parallel()
	h := tui.NewHeaderModel("v1.0.0")
	h.SetWidth(0)

	// Should not panic
	view := h.View()
	_ = view
}
