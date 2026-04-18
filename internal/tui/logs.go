package tui

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/agbru/fibcalc/internal/config"
	"github.com/agbru/fibcalc/internal/format"
	"github.com/agbru/fibcalc/internal/orchestration"
)

const maxLogEntries = 10000

// LogsModel manages the scrollable log panel.
type LogsModel struct {
	viewport   viewport.Model
	entries    []string
	autoScroll bool
	width      int
	height     int
	algoNames  []string // algorithm names for mapping index -> name
}

// NewLogsModel creates a new logs panel.
func NewLogsModel(algoNames []string) LogsModel {
	vp := viewport.New(40, 10)
	return LogsModel{
		viewport:   vp,
		entries:    make([]string, 0, 64),
		autoScroll: true,
		algoNames:  algoNames,
	}
}

// Reset clears all log entries.
func (l *LogsModel) Reset() {
	l.entries = l.entries[:0]
	l.autoScroll = true
	l.updateContent()
}

// SetSize updates the viewport dimensions.
func (l *LogsModel) SetSize(w, h int) {
	l.width = w
	l.height = h
	l.viewport.Width = w - 2
	l.viewport.Height = h - 2
	l.updateContent()
}

// AddExecutionConfig adds the execution configuration summary as initial log entries.
func (l *LogsModel) AddExecutionConfig(cfg config.AppConfig) {
	l.entries = append(l.entries, logAlgoStyle.Render("--- Execution Configuration ---"))
	l.entries = append(l.entries, fmt.Sprintf("  Calculating %s with a timeout of %s.",
		logAlgoStyle.Render(fmt.Sprintf("F(%d)", cfg.N)),
		metricValueStyle.Render(cfg.Timeout.String())))
	l.entries = append(l.entries, fmt.Sprintf("  Environment: %s logical processors, Go %s.",
		metricValueStyle.Render(fmt.Sprintf("%d", runtime.NumCPU())),
		metricValueStyle.Render(runtime.Version())))
	l.entries = append(l.entries, fmt.Sprintf("  Optimization thresholds: Parallelism=%s bits, FFT=%s bits.",
		metricValueStyle.Render(fmt.Sprintf("%d", cfg.Threshold)),
		metricValueStyle.Render(fmt.Sprintf("%d", cfg.FFTThreshold))))

	var modeDesc string
	if len(l.algoNames) > 1 {
		modeDesc = "Parallel comparison of all algorithms"
	} else if len(l.algoNames) == 1 {
		modeDesc = fmt.Sprintf("Single calculation with the %s algorithm", logSuccessStyle.Render(l.algoNames[0]))
	}
	l.entries = append(l.entries, fmt.Sprintf("  Execution mode: %s.", modeDesc))
	l.entries = append(l.entries, "")
	l.updateContent()
}

// AddProgressEntry adds a progress log line.
func (l *LogsModel) AddProgressEntry(msg ProgressMsg) {
	ts := logTimeStyle.Render(time.Now().Format("15:04:05"))
	name := l.algoName(msg.CalculatorIndex)
	algoStr := logAlgoStyle.Render(fmt.Sprintf("%-16s", name))

	var progressStr string
	if msg.Value >= 1.0 {
		progressStr = logSuccessStyle.Render("100% OK")
	} else {
		progressStr = logProgressStyle.Render(fmt.Sprintf("%5.1f%%", msg.Value*100))
	}

	entry := fmt.Sprintf("[%s] %s %s", ts, algoStr, progressStr)
	l.entries = append(l.entries, entry)
	l.trimEntries()
	l.updateContent()
}

// AddResults adds comparison results to the log.
func (l *LogsModel) AddResults(results []orchestration.CalculationResult) {
	l.entries = append(l.entries, "")
	l.entries = append(l.entries, logAlgoStyle.Render("--- Comparison Summary ---"))

	// Find max name and duration widths for column alignment
	maxNameLen := 0
	maxDurLen := 0
	for _, res := range results {
		if len(res.Name) > maxNameLen {
			maxNameLen = len(res.Name)
		}
		dur := format.FormatExecutionDuration(res.Duration)
		if len(dur) > maxDurLen {
			maxDurLen = len(dur)
		}
	}

	nameFmt := fmt.Sprintf("%%-%ds", maxNameLen)
	durFmt := fmt.Sprintf("%%%ds", maxDurLen)

	for _, res := range results {
		var status string
		if res.Err != nil {
			status = logErrorStyle.Render(fmt.Sprintf("FAIL (%v)", res.Err))
		} else {
			status = logSuccessStyle.Render("OK")
		}
		duration := format.FormatExecutionDuration(res.Duration)
		entry := fmt.Sprintf("  %s  %s  %s",
			logAlgoStyle.Render(fmt.Sprintf(nameFmt, res.Name)),
			metricValueStyle.Render(fmt.Sprintf(durFmt, duration)),
			status)
		l.entries = append(l.entries, entry)
	}
	l.trimEntries()
	l.updateContent()
}

// AddFinalResult adds the final result to the log.
func (l *LogsModel) AddFinalResult(msg FinalResultMsg) {
	l.entries = append(l.entries, "")
	l.entries = append(l.entries, logSuccessStyle.Render("--- Final Result ---"))
	l.entries = append(l.entries, fmt.Sprintf("  Algorithm: %s", logAlgoStyle.Render(msg.Result.Name)))
	l.entries = append(l.entries, fmt.Sprintf("  Duration:  %s", metricValueStyle.Render(format.FormatExecutionDuration(msg.Result.Duration))))
	if msg.Result.Result != nil {
		bits := msg.Result.Result.BitLen()
		l.entries = append(l.entries, fmt.Sprintf("  Bits:      %s", metricValueStyle.Render(format.FormatNumberString(fmt.Sprintf("%d", bits)))))
	}
	l.trimEntries()
	l.updateContent()
}

// AddError adds an error entry to the log.
func (l *LogsModel) AddError(msg ErrorMsg) {
	ts := logTimeStyle.Render(time.Now().Format("15:04:05"))
	entry := fmt.Sprintf("[%s] %s", ts, logErrorStyle.Render(fmt.Sprintf("ERROR: %v", msg.Err)))
	l.entries = append(l.entries, entry)
	l.trimEntries()
	l.updateContent()
}

// Update handles viewport keyboard events.
func (l *LogsModel) Update(msg tea.Msg) {
	var cmd tea.Cmd
	l.viewport, cmd = l.viewport.Update(msg)
	_ = cmd

	// If user scrolled up manually, disable auto-scroll
	if l.viewport.AtBottom() {
		l.autoScroll = true
	} else {
		l.autoScroll = false
	}
}

// View renders the logs panel.
func (l LogsModel) View() string {
	return l.renderToHeight(l.height)
}

// renderToHeight renders the logs panel to the specified total height.
func (l LogsModel) renderToHeight(h int) string {
	return panelStyle.
		Width(l.width - 2).
		Height(max(h-2, 0)).
		Render(l.viewport.View())
}

func (l *LogsModel) trimEntries() {
	if len(l.entries) > maxLogEntries {
		l.entries = l.entries[len(l.entries)-maxLogEntries:]
	}
}

func (l *LogsModel) updateContent() {
	content := strings.Join(l.entries, "\n")
	l.viewport.SetContent(content)
	if l.autoScroll {
		l.viewport.GotoBottom()
	}
}

func (l LogsModel) algoName(index int) string {
	if index < 0 {
		return "Unknown"
	}
	if index < len(l.algoNames) {
		return l.algoNames[index]
	}
	return fmt.Sprintf("Algo %d", index)
}
