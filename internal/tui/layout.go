package tui

// Layout constants for the TUI dashboard.
const (
	headerHeight          = 1
	footerHeight          = 1
	minBodyHeight         = 4
	LogsPanelWidthPercent = 60
	MetricsPanelHeight    = 7 // compact: top line + 1 data row + borders; expands to ~9 with indicators

	// Adaptive thresholds (R4.10): below these dimensions the layout switches
	// to a compact / single-column mode so panels remain readable on small
	// terminals (e.g. tmux splits, narrow consoles).
	minNormalWidth      = 80 // below: switch to single-column (logs full width, panels stacked).
	minNormalHeight     = 20 // below: shrink metrics panel to keep room for logs + chart.
	compactMetricsHeight = 3 // metrics panel height when height < minNormalHeight.
)

// LayoutManager holds terminal dimensions and provides layout calculations.
type LayoutManager struct {
	width  int
	height int
}

// isNarrow reports whether the terminal is too narrow for the side-by-side
// layout. In narrow mode, the logs panel takes the full width and the
// metrics + chart panels stack vertically below it.
func (l LayoutManager) isNarrow() bool {
	return l.width > 0 && l.width < minNormalWidth
}

// isShort reports whether the terminal is too short for the default
// metrics-panel height, triggering the compact metrics mode.
func (l LayoutManager) isShort() bool {
	return l.height > 0 && l.height < minNormalHeight
}

// bodyHeight returns the available height for the main body panels.
func (l LayoutManager) bodyHeight() int {
	h := l.height - headerHeight - footerHeight
	if h < minBodyHeight {
		h = minBodyHeight
	}
	return h
}

// logsWidth returns the width allocated to the logs panel.
func (l LayoutManager) logsWidth() int {
	if l.isNarrow() {
		return l.width
	}
	return l.width * LogsPanelWidthPercent / 100
}

// rightWidth returns the width allocated to the right column (metrics + chart).
// In narrow mode the right column is stacked below the logs and uses the full
// terminal width.
func (l LayoutManager) rightWidth() int {
	if l.isNarrow() {
		return l.width
	}
	return l.width - l.logsWidth()
}

// baseMetricsHeight returns the unclamped metrics panel height, taking the
// adaptive compact mode into account.
func (l LayoutManager) baseMetricsHeight() int {
	if l.isShort() {
		return compactMetricsHeight
	}
	return MetricsPanelHeight
}

// metricsHeight returns the height allocated to the metrics panel, clamped so
// it never consumes more than half of the available body height (or the
// stacked right column in narrow mode).
func (l LayoutManager) metricsHeight() int {
	avail := l.rightColumnHeight()
	h := l.baseMetricsHeight()
	if h > avail/2 {
		h = avail / 2
	}
	if h < 1 {
		h = 1
	}
	return h
}

// metricsWidth returns the width allocated to the metrics panel.
func (l LayoutManager) metricsWidth() int {
	return l.rightWidth()
}

// chartHeight returns the height allocated to the chart panel.
func (l LayoutManager) chartHeight() int {
	h := l.rightColumnHeight() - l.metricsHeight()
	if h < 1 {
		h = 1
	}
	return h
}

// rightColumnHeight returns the vertical space available for the metrics +
// chart stack. In side-by-side mode this equals the body height; in narrow
// (single-column) mode the body is split between the logs panel on top and
// the stacked right column below.
func (l LayoutManager) rightColumnHeight() int {
	if !l.isNarrow() {
		return l.bodyHeight()
	}
	// Single-column: reserve roughly half the body for the stacked panels,
	// floored at minBodyHeight/2 so neither panel collapses to zero.
	body := l.bodyHeight()
	h := body / 2
	if h < 2 {
		h = 2
	}
	return h
}

// logsHeight returns the height allocated to the logs panel. In side-by-side
// mode it gets the full body; in narrow mode it gets what is left after the
// stacked right column.
func (l LayoutManager) logsHeight() int {
	if !l.isNarrow() {
		return l.bodyHeight()
	}
	h := l.bodyHeight() - l.rightColumnHeight()
	if h < minBodyHeight {
		h = minBodyHeight
	}
	return h
}
