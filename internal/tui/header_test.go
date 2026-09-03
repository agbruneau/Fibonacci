package tui

import (
	"strings"
	"testing"
	"time"
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
	if view == "" {
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

func TestHeaderModel_View_FrozenAfterDone(t *testing.T) {
	t.Parallel()
	h := NewHeaderModel("v1.0.0")
	h.SetWidth(80)
	h.SetDone()
	// Pin the start exactly 90s before the frozen end: the elapsed text must
	// be computed from endTime ("1m30s" exactly), not from time.Since, which
	// would add wall-clock jitter and render e.g. "1m30.0001s".
	h.startTime = h.endTime.Add(-90 * time.Second)

	view := h.View()
	if !strings.Contains(view, "Elapsed: 1m30s") {
		t.Errorf("expected frozen elapsed 'Elapsed: 1m30s', got %q", view)
	}
}
