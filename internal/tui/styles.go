package tui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/agbruneau/FibGo/internal/ui"
)

// Style variables for the TUI dashboard.
// Initialized from the ui theme system via initTUIStyles().
var (
	panelStyle         lipgloss.Style
	headerStyle        lipgloss.Style
	titleStyle         lipgloss.Style
	versionStyle       lipgloss.Style
	elapsedStyle       lipgloss.Style
	logTimeStyle       lipgloss.Style
	logAlgoStyle       lipgloss.Style
	logProgressStyle   lipgloss.Style
	logSuccessStyle    lipgloss.Style
	logErrorStyle      lipgloss.Style
	metricLabelStyle   lipgloss.Style
	metricValueStyle   lipgloss.Style
	chartBarStyle      lipgloss.Style
	chartEmptyStyle    lipgloss.Style
	footerKeyStyle     lipgloss.Style
	footerDescStyle    lipgloss.Style
	statusRunningStyle lipgloss.Style
	statusPausedStyle  lipgloss.Style
	statusDoneStyle    lipgloss.Style
	statusErrorStyle   lipgloss.Style
	cpuSparklineStyle  lipgloss.Style
	memSparklineStyle  lipgloss.Style
)

// init builds the TUI styles once at package load so that tests and callers
// that exercise rendering helpers without going through Run() observe a
// consistent, non-zero set of styles. Run() invokes initTUIStyles() again
// after app.Run has resolved the final theme via ui.InitTheme — this second
// call is required because the active theme may have changed (e.g. NO_COLOR,
// FIBCALC_TUI_THEME=high-contrast) between package load and Run().
func init() {
	initTUIStyles()
}

// initTUIStyles rebuilds all TUI styles from the current ui theme.
// Idempotent: safe to call multiple times.
func initTUIStyles() {
	t := ui.GetCurrentTUITheme()

	panelStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Background(t.Bg).
		Foreground(t.Text)

	headerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Accent).
		Background(t.Bg).
		Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Accent)

	versionStyle = lipgloss.NewStyle().
		Foreground(t.Dim)

	elapsedStyle = lipgloss.NewStyle().
		Foreground(t.Accent)

	logTimeStyle = lipgloss.NewStyle().
		Foreground(t.Dim)

	logAlgoStyle = lipgloss.NewStyle().
		Foreground(t.Info)

	logProgressStyle = lipgloss.NewStyle().
		Foreground(t.Accent)

	logSuccessStyle = lipgloss.NewStyle().
		Foreground(t.Success)

	logErrorStyle = lipgloss.NewStyle().
		Foreground(t.Error)

	metricLabelStyle = lipgloss.NewStyle().
		Foreground(t.Dim)

	metricValueStyle = lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true)

	chartBarStyle = lipgloss.NewStyle().
		Foreground(t.Accent)

	chartEmptyStyle = lipgloss.NewStyle().
		Foreground(t.Dim)

	footerKeyStyle = lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true)

	footerDescStyle = lipgloss.NewStyle().
		Foreground(t.Dim)

	statusRunningStyle = lipgloss.NewStyle().
		Foreground(t.Success).
		Bold(true)

	statusPausedStyle = lipgloss.NewStyle().
		Foreground(t.Warning).
		Bold(true)

	statusDoneStyle = lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true)

	statusErrorStyle = lipgloss.NewStyle().
		Foreground(t.Error).
		Bold(true)

	cpuSparklineStyle = lipgloss.NewStyle().
		Foreground(t.Accent)

	memSparklineStyle = lipgloss.NewStyle().
		Foreground(t.Warning)
}
