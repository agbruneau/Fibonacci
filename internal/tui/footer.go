package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// FooterModel renders the bottom status bar.
type FooterModel struct {
	paused bool
	done   bool
	hasErr bool
	width  int
}

// NewFooterModel creates a new footer.
func NewFooterModel() FooterModel {
	return FooterModel{}
}

// SetWidth updates the available width.
func (f *FooterModel) SetWidth(w int) {
	f.width = w
}

// SetPaused sets the paused state.
func (f *FooterModel) SetPaused(p bool) {
	f.paused = p
}

// SetDone sets the done state.
func (f *FooterModel) SetDone(d bool) {
	f.done = d
}

// SetError sets the error state.
func (f *FooterModel) SetError(e bool) {
	f.hasErr = e
}

// View renders the footer.
func (f FooterModel) View() string {
	shortcuts := fmt.Sprintf(
		"%s: %s   %s: %s   %s: %s",
		footerKeyStyle.Render("q"), footerDescStyle.Render("Quit"),
		footerKeyStyle.Render("r"), footerDescStyle.Render("Restart"),
		footerKeyStyle.Render("space"), footerDescStyle.Render("Pause/Resume"),
	)

	var status string
	switch {
	case f.hasErr:
		status = statusErrorStyle.Render("[!] Status: Error")
	case f.done:
		status = statusDoneStyle.Render("[OK] Status: Done")
	case f.paused:
		status = statusPausedStyle.Render("[||] Status: Paused")
	default:
		status = statusRunningStyle.Render("[>] Status: Running")
	}

	innerWidth := f.width - 2
	if innerWidth < 0 {
		innerWidth = 0
	}

	shortcutsWidth := lipgloss.Width(shortcuts)
	statusWidth := lipgloss.Width(status)
	gap := innerWidth - shortcutsWidth - statusWidth
	if gap < 2 {
		gap = 2
	}

	row := shortcuts + strings.Repeat(" ", gap) + status

	return headerStyle.Width(f.width).Render(row)
}
