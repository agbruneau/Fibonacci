package tui

import (
	"strings"
	"testing"
)

func TestFooterModel_View_Running(t *testing.T) {
	t.Parallel()
	f := NewFooterModel()
	f.SetWidth(80)

	view := f.View()
	if !strings.Contains(view, "Running") {
		t.Error("expected footer to show 'Running' status by default")
	}
}

func TestFooterModel_View_Paused(t *testing.T) {
	t.Parallel()
	f := NewFooterModel()
	f.SetWidth(80)
	f.SetPaused(true)

	view := f.View()
	if !strings.Contains(view, "Paused") {
		t.Error("expected footer to show 'Paused' status")
	}
}

func TestFooterModel_View_Done(t *testing.T) {
	t.Parallel()
	f := NewFooterModel()
	f.SetWidth(80)
	f.SetDone(true)

	view := f.View()
	if !strings.Contains(view, "Done") {
		t.Error("expected footer to show 'Done' status")
	}
}

func TestFooterModel_View_Error(t *testing.T) {
	t.Parallel()
	f := NewFooterModel()
	f.SetWidth(80)
	f.SetError(true)
	f.SetDone(true)

	view := f.View()
	if !strings.Contains(view, "Error") {
		t.Error("expected footer to show 'Error' status")
	}
}

func TestFooterModel_View_ErrorPrecedence(t *testing.T) {
	t.Parallel()
	// Error should take precedence over Done and Paused
	f := NewFooterModel()
	f.SetWidth(80)
	f.SetError(true)
	f.SetDone(true)
	f.SetPaused(true)

	view := f.View()
	if !strings.Contains(view, "Error") {
		t.Error("expected 'Error' to take precedence")
	}
}

func TestFooterModel_View_Shortcuts(t *testing.T) {
	t.Parallel()
	f := NewFooterModel()
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

func TestFooterModel_View_NarrowWidth(t *testing.T) {
	t.Parallel()
	f := NewFooterModel()
	f.SetWidth(5)

	// Should not panic
	view := f.View()
	if view == "" {
		t.Error("expected non-empty view even with narrow width")
	}
}

func TestFooterModel_SetWidth_Negative(t *testing.T) {
	t.Parallel()
	f := NewFooterModel()
	f.SetWidth(-1)

	// Should not panic
	view := f.View()
	if view == "" {
		t.Error("expected non-empty view even with negative width")
	}
}

func TestFooterModel_SetWidth_Zero(t *testing.T) {
	t.Parallel()
	f := NewFooterModel()
	f.SetWidth(0)

	// Should not panic
	view := f.View()
	if view == "" {
		t.Error("expected non-empty view even with zero width")
	}
}
