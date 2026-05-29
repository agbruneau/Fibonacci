package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/agbruneau/FibGo/internal/format"
)

// ChartModel renders a progress bar, ETA, and system metrics sparklines.
type ChartModel struct {
	averageProgress float64
	eta             time.Duration
	elapsed         time.Duration
	done            bool
	width           int
	height          int

	cpuHistory *RingBuffer
	memHistory *RingBuffer
}

const defaultSparklineCap = 30

// NewChartModel creates a new chart.
func NewChartModel() ChartModel {
	return ChartModel{
		cpuHistory: NewRingBuffer(defaultSparklineCap),
		memHistory: NewRingBuffer(defaultSparklineCap),
	}
}

// SetSize updates dimensions.
func (c *ChartModel) SetSize(w, h int) {
	c.width = w
	c.height = h
	if sw := c.sparklineWidth(); sw > 0 {
		c.cpuHistory.Resize(sw)
		c.memHistory.Resize(sw)
	}
}

// AddDataPoint records a progress sample.
func (c *ChartModel) AddDataPoint(progress float64, avg float64, eta time.Duration) {
	c.averageProgress = avg
	c.eta = eta
}

// UpdateSysStats records a system metrics sample.
func (c *ChartModel) UpdateSysStats(cpuPct, memPct float64) {
	c.cpuHistory.Push(cpuPct)
	c.memHistory.Push(memPct)
}

// SetDone marks the chart as complete with the total elapsed time.
func (c *ChartModel) SetDone(elapsed time.Duration) {
	c.done = true
	c.averageProgress = 1.0
	c.elapsed = elapsed
}

// Reset clears the chart data.
func (c *ChartModel) Reset() {
	c.averageProgress = 0
	c.eta = 0
	c.elapsed = 0
	c.done = false
	c.cpuHistory.Reset()
	c.memHistory.Reset()
}

// View renders the chart panel.
func (c ChartModel) View() string {
	var b strings.Builder

	// Header: "Progress Chart" left, ETA right
	var statusStr string
	if c.done {
		statusStr = fmt.Sprintf("Completed in %s", format.FormatExecutionDuration(c.elapsed))
	} else {
		statusStr = fmt.Sprintf("ETA: %s", format.FormatETA(c.eta))
	}
	titleLeft := metricLabelStyle.Render("  Progress Chart")
	titleRight := elapsedStyle.Render(statusStr + "  ")
	gap := c.width - 4 - lipgloss.Width(titleLeft) - lipgloss.Width(titleRight)
	if gap < 1 {
		gap = 1
	}
	b.WriteString(titleLeft)
	b.WriteString(strings.Repeat(" ", gap))
	b.WriteString(titleRight)
	b.WriteString("\n\n")

	// Render progress bar
	progressBar := c.renderProgressBar()
	if progressBar != "" {
		b.WriteString("  ")
		b.WriteString(progressBar)
	}

	// Render CPU braille chart if space allows
	if c.height >= 8 && c.sparklineWidth() > 0 {
		b.WriteString("\n\n")
		b.WriteString(c.renderBrailleSection())
	}

	return panelStyle.
		Width(c.width - 2).
		Height(c.height - 2).
		Render(b.String())
}

func (c ChartModel) renderProgressBar() string {
	barWidth := c.width - 15 // border + indent + brackets + " 100.0%"
	if barWidth < 5 {
		return ""
	}

	filled := int(c.averageProgress * float64(barWidth))
	if filled < 0 {
		filled = 0
	}
	if filled > barWidth {
		filled = barWidth
	}
	empty := barWidth - filled

	filledStr := chartBarStyle.Render(strings.Repeat("█", filled))
	emptyStr := chartEmptyStyle.Render(strings.Repeat("░", empty))
	pctStr := metricValueStyle.Render(fmt.Sprintf("%5.1f%%", c.averageProgress*100))

	return fmt.Sprintf("[%s%s] %s", filledStr, emptyStr, pctStr)
}

// sparklineWidth computes the number of characters available for the sparkline.
// Line format: "  CPU: xxx.x% [sparkline]" → 16 chars prefix/suffix + 2 border.
func (c ChartModel) sparklineWidth() int {
	w := c.width - 18
	if w < 1 {
		return 0
	}
	return w
}

// renderBrailleSection renders CPU and MEM sparkline indicators.
func (c ChartModel) renderBrailleSection() string {
	var b strings.Builder

	// CPU label: percentage after colon, then sparkline
	cpuPct := c.cpuHistory.Last()
	cpuLabel := fmt.Sprintf("  %s %s [%s]",
		metricLabelStyle.Render("CPU:"),
		metricValueStyle.Render(fmt.Sprintf("%5.1f%%", cpuPct)),
		cpuSparklineStyle.Render(RenderSparkline(c.cpuHistory.Slice())))
	b.WriteString(cpuLabel)

	// MEM label: percentage after colon, then sparkline
	memPct := c.memHistory.Last()
	memLabel := fmt.Sprintf("\n  %s %s [%s]",
		metricLabelStyle.Render("MEM:"),
		metricValueStyle.Render(fmt.Sprintf("%5.1f%%", memPct)),
		memSparklineStyle.Render(RenderSparkline(c.memHistory.Slice())))
	b.WriteString(memLabel)

	return b.String()
}
