package tui

import (
	"strings"
	"testing"
	"time"
)

// TestHeaderModel_View_FrozenAfterDone requires direct field access
// (startTime/endTime are unexported): the elapsed text must be computed from
// the frozen endTime, not time.Since, so this pins the exact duration string
// deterministically rather than tolerating wall-clock jitter.
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
