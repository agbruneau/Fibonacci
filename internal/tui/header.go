package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/agbruneau/FibGo/internal/format"
)

// HeaderModel renders the top bar: title, version, elapsed time.
type HeaderModel struct {
	startTime time.Time
	endTime   time.Time
	version   string
	width     int
}

// NewHeaderModel creates a new header.
func NewHeaderModel(version string) HeaderModel {
	return HeaderModel{
		startTime: time.Now(),
		version:   version,
	}
}

// SetDone freezes the elapsed timer at the current time.
func (h *HeaderModel) SetDone() {
	h.endTime = time.Now()
}

// Reset restarts the elapsed timer.
func (h *HeaderModel) Reset() {
	h.startTime = time.Now()
	h.endTime = time.Time{}
}

// SetWidth updates the available width.
func (h *HeaderModel) SetWidth(w int) {
	h.width = w
}

// View renders the header.
func (h HeaderModel) View() string {
	titleText := "FibGo Monitor"
	if h.version != "" && h.version != "dev" {
		titleText += " " + h.version
	}
	title := titleStyle.Render(titleText)

	pipe := versionStyle.Render(" | ")

	var duration time.Duration
	if !h.endTime.IsZero() {
		duration = h.endTime.Sub(h.startTime)
	} else {
		duration = time.Since(h.startTime)
	}
	elapsed := elapsedStyle.Render(fmt.Sprintf("Elapsed: %s", format.FormatExecutionDuration(duration)))

	leftPart := title + pipe + elapsed
	leftLen := lipgloss.Width(leftPart)

	innerWidth := h.width - 2
	if innerWidth < 0 {
		innerWidth = 0
	}

	gap := innerWidth - leftLen
	if gap < 0 {
		gap = 0
	}

	row := leftPart + spaces(gap)

	return headerStyle.Width(h.width).Render(row)
}

// spaces returns a string of n space characters.
func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = ' '
	}
	return string(b)
}
