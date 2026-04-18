package tui

import (
	"strings"
	"testing"
)

func TestHeaderModel_View_ContainsTitle(t *testing.T) {
	t.Parallel()
	h := NewHeaderModel("v1.0.0")
	h.SetWidth(80)

	view := h.View()
	if !strings.Contains(view, "FibGo Monitor") {
		t.Error("expected header to contain 'FibGo Monitor'")
	}
}

func TestHeaderModel_View_ContainsVersion(t *testing.T) {
	t.Parallel()
	h := NewHeaderModel("v2.3.4")
	h.SetWidth(80)

	view := h.View()
	if !strings.Contains(view, "v2.3.4") {
		t.Error("expected header to contain version string")
	}
}

func TestHeaderModel_View_ContainsElapsed(t *testing.T) {
	t.Parallel()
	h := NewHeaderModel("v1.0.0")
	h.SetWidth(80)

	view := h.View()
	if !strings.Contains(view, "Elapsed") {
		t.Error("expected header to contain 'Elapsed'")
	}
}

func TestHeaderModel_View_NarrowWidth(t *testing.T) {
	t.Parallel()
	h := NewHeaderModel("v1.0.0")
	h.SetWidth(10)

	// Should not panic even with very narrow width
	view := h.View()
	if len(view) == 0 {
		t.Error("expected non-empty view even with narrow width")
	}
}

func TestHeaderModel_View_ZeroWidth(t *testing.T) {
	t.Parallel()
	h := NewHeaderModel("v1.0.0")
	h.SetWidth(0)

	// Should not panic
	view := h.View()
	_ = view
}

func TestSpaces(t *testing.T) {
	t.Parallel()
	tests := []struct {
		n    int
		want string
	}{
		{0, ""},
		{-1, ""},
		{1, " "},
		{5, "     "},
	}
	for _, tt := range tests {
		got := spaces(tt.n)
		if got != tt.want {
			t.Errorf("spaces(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}
