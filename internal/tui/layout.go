package tui

// Layout constants for the TUI dashboard.
const (
	headerHeight          = 1
	footerHeight          = 1
	minBodyHeight         = 4
	LogsPanelWidthPercent = 60
	MetricsPanelHeight    = 7 // compact: top line + 1 data row + borders; expands to ~9 with indicators
)

// LayoutManager holds terminal dimensions and provides layout calculations.
type LayoutManager struct {
	width  int
	height int
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
	return l.width * LogsPanelWidthPercent / 100
}

// rightWidth returns the width allocated to the right column (metrics + chart).
func (l LayoutManager) rightWidth() int {
	return l.width - l.logsWidth()
}

// metricsHeight returns the height allocated to the metrics panel.
func (l LayoutManager) metricsHeight() int {
	body := l.bodyHeight()
	h := MetricsPanelHeight
	if h > body/2 {
		h = body / 2
	}
	return h
}

// metricsWidth returns the width allocated to the metrics panel.
func (l LayoutManager) metricsWidth() int {
	return l.rightWidth()
}

// chartHeight returns the height allocated to the chart panel.
func (l LayoutManager) chartHeight() int {
	return l.bodyHeight() - l.metricsHeight()
}
