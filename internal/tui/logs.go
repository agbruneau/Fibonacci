package tui

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/agbruneau/FibGo/internal/config"
	"github.com/agbruneau/FibGo/internal/format"
	"github.com/agbruneau/FibGo/internal/orchestration"
)

const maxLogEntries = 10000

// LogsModel manages the scrollable log panel.
//
// Storage layout (audit task R4.8): entries are pushed into a fixed-capacity
// Ring[string] (`buffer`). The `entries` field is a chronological snapshot
// (oldest first) refreshed in updateContent and exposed for the white-box
// tests that read len(entries) / entries[i] / strings.Join(entries, "\n").
// Compared to the previous append-then-trim implementation this caps the
// underlying allocation at maxLogEntries strings rather than letting `append`
// grow the backing array unboundedly between trims.
type LogsModel struct {
	viewport   viewport.Model
	buffer     *Ring[string]
	entries    []string
	autoScroll bool
	dirty      bool // entries changed since the last viewport flush
	width      int
	height     int
	algoNames  []string // algorithm names for mapping index -> name
}

// NewLogsModel creates a new logs panel.
func NewLogsModel(algoNames []string) LogsModel {
	vp := viewport.New(40, 10)
	return LogsModel{
		viewport:   vp,
		buffer:     NewRing[string](maxLogEntries),
		autoScroll: true,
		algoNames:  algoNames,
	}
}

// Reset clears all log entries.
func (l *LogsModel) Reset() {
	l.buffer.Reset()
	l.entries = nil
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
	l.buffer.Push(logAlgoStyle.Render("--- Execution Configuration ---"))
	l.buffer.Push(fmt.Sprintf("  Calculating %s with a timeout of %s.",
		logAlgoStyle.Render(fmt.Sprintf("F(%d)", cfg.N)),
		metricValueStyle.Render(cfg.Timeout.String())))
	l.buffer.Push(fmt.Sprintf("  Environment: %s logical processors, Go %s.",
		metricValueStyle.Render(fmt.Sprintf("%d", runtime.NumCPU())),
		metricValueStyle.Render(runtime.Version())))
	l.buffer.Push(fmt.Sprintf("  Optimization thresholds: Parallelism=%s bits, FFT=%s bits.",
		metricValueStyle.Render(fmt.Sprintf("%d", cfg.Threshold)),
		metricValueStyle.Render(fmt.Sprintf("%d", cfg.FFTThreshold))))

	var modeDesc string
	if len(l.algoNames) > 1 {
		modeDesc = "Parallel comparison of all algorithms"
	} else if len(l.algoNames) == 1 {
		modeDesc = fmt.Sprintf("Single calculation with the %s algorithm", logSuccessStyle.Render(l.algoNames[0]))
	}
	l.buffer.Push(fmt.Sprintf("  Execution mode: %s.", modeDesc))
	l.buffer.Push("")
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
	l.buffer.Push(entry)
	l.updateContent()
}

// AddResults adds comparison results to the log.
func (l *LogsModel) AddResults(results []orchestration.CalculationResult) {
	l.buffer.Push("")
	l.buffer.Push(logAlgoStyle.Render("--- Comparison Summary ---"))

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
		l.buffer.Push(entry)
	}
	l.updateContent()
}

// AddFinalResult adds the final result to the log.
func (l *LogsModel) AddFinalResult(msg FinalResultMsg) {
	l.buffer.Push("")
	l.buffer.Push(logSuccessStyle.Render("--- Final Result ---"))
	l.buffer.Push(fmt.Sprintf("  Algorithm: %s", logAlgoStyle.Render(msg.Result.Name)))
	l.buffer.Push(fmt.Sprintf("  Duration:  %s", metricValueStyle.Render(format.FormatExecutionDuration(msg.Result.Duration))))
	if msg.Result.Result != nil {
		bits := msg.Result.Result.BitLen()
		l.buffer.Push(fmt.Sprintf("  Bits:      %s", metricValueStyle.Render(format.FormatNumberString(fmt.Sprintf("%d", bits)))))
	}
	l.updateContent()
}

// AddError adds an error entry to the log.
func (l *LogsModel) AddError(msg ErrorMsg) {
	ts := logTimeStyle.Render(time.Now().Format("15:04:05"))
	entry := fmt.Sprintf("[%s] %s", ts, logErrorStyle.Render(fmt.Sprintf("ERROR: %v", msg.Err)))
	l.buffer.Push(entry)
	l.updateContent()
}

// Update handles viewport keyboard events. It flushes any pending content
// first so the viewport's measured line count (and therefore AtBottom) is
// accurate before a scroll key is applied — the eager path got this for free
// because every push re-set the content synchronously.
func (l *LogsModel) Update(msg tea.Msg) {
	l.flushContent()

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
//
// The receiver is a value copy (the bubbletea View contract). flushContent
// materialises the deferred join + SetContent on that copy, so the rendered
// frame always reflects the latest entries without the per-push O(N) cost.
// Mutating the copy is intentional: the persisted viewport scroll state is
// committed separately in Update (which also flushes), so View stays pure.
func (l LogsModel) renderToHeight(h int) string {
	l.flushContent()
	return panelStyle.
		Width(l.width - 2).
		Height(max(h-2, 0)).
		Render(l.viewport.View())
}

// updateContent refreshes the chronological snapshot from the ring buffer and
// marks the rendered viewport content stale. It is the cheap half of the
// former eager updateContent: the white-box tests and the orchestration layer
// rely on LogsModel.entries being current immediately after every push, but
// the expensive strings.Join + viewport.SetContent (O(N) over up to
// maxLogEntries lines) is deferred to flushContent at render time.
//
// Before A-22 this method also joined and re-set the viewport on every push,
// so a long session paid O(N) per progress message — O(N²) overall. Now each
// push only pays the chronological slice; the join happens once per rendered
// frame regardless of how many entries were pushed in between.
func (l *LogsModel) updateContent() {
	l.entries = l.buffer.Slice()
	l.dirty = true
}

// flushContent materialises the deferred viewport content. It is idempotent:
// when nothing has been pushed since the last flush (dirty == false) it is a
// no-op, so calling it on every render frame is cheap. autoScroll is honoured
// here exactly as the eager path did, preserving the observable scroll
// behaviour (GotoBottom after each content change while pinned to bottom).
func (l *LogsModel) flushContent() {
	if !l.dirty {
		return
	}
	l.dirty = false
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
