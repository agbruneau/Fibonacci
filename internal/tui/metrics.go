package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/agbruneau/FibGo/internal/format"
	"github.com/agbruneau/FibGo/internal/metrics"
)

// EMA smoothing constants for speed calculation.
const (
	EMASmoothFactor = 0.7
	EMADecayFactor  = 0.3
)

// MetricsModel displays runtime memory and performance metrics.
type MetricsModel struct {
	alloc        uint64
	heapSys      uint64
	numGC        uint32
	pauseTotalNs uint64
	numGoroutine int
	speed        float64 // progress per second
	lastProgress float64
	lastUpdate   time.Time
	indicators   *metrics.Indicators
	width        int
	height       int
}

// NewMetricsModel creates a new metrics panel.
func NewMetricsModel() MetricsModel {
	return MetricsModel{
		lastUpdate: time.Now(),
	}
}

// SetSize updates dimensions.
func (m *MetricsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// UpdateMemStats updates memory statistics.
func (m *MetricsModel) UpdateMemStats(msg MemStatsMsg) {
	m.alloc = msg.Alloc
	m.heapSys = msg.HeapSys
	m.numGC = msg.NumGC
	m.pauseTotalNs = msg.PauseTotalNs
	m.numGoroutine = msg.NumGoroutine
}

// UpdateProgress updates the speed metric.
func (m *MetricsModel) UpdateProgress(progress float64) {
	now := time.Now()
	dt := now.Sub(m.lastUpdate).Seconds()
	if dt > 0.05 {
		dp := progress - m.lastProgress
		if dp > 0 {
			instantSpeed := dp / dt
			if m.speed > 0 {
				m.speed = EMASmoothFactor*m.speed + EMADecayFactor*instantSpeed
			} else {
				m.speed = instantSpeed
			}
		}
		m.lastProgress = progress
		m.lastUpdate = now
	}
}

// UpdateIndicators stores the post-calculation indicators.
func (m *MetricsModel) UpdateIndicators(ind *metrics.Indicators) {
	m.indicators = ind
}

// View renders the metrics panel.
func (m MetricsModel) View() string {
	var rows strings.Builder

	// Compact top line: Heap: X / Y | GC: N (Xms)
	heapStr := metricValueStyle.Render(format.FormatBytes(m.alloc) + " / " + format.FormatBytes(m.heapSys))
	gcPauseStr := metricValueStyle.Render(fmt.Sprintf("%d (%.1fms)", m.numGC, float64(m.pauseTotalNs)/1e6))
	pipe := metricLabelStyle.Render(" | ")
	topLine := fmt.Sprintf("  %s %s%s%s %s",
		metricLabelStyle.Render("Heap:"), heapStr,
		pipe,
		metricLabelStyle.Render("GC:"), gcPauseStr)
	rows.WriteString(topLine)

	colWidth := (m.width - 6) / 2

	leftCol := []string{
		formatMetricCol("Speed:", format.FormatETA(time.Duration(float64(time.Second)/max(m.speed, 0.001)))+"/calc", colWidth),
	}
	rightCol := []string{
		formatMetricCol("Goroutines:", fmt.Sprintf("%d", m.numGoroutine), colWidth),
	}

	if m.indicators != nil {
		parity := "odd"
		if m.indicators.IsEven {
			parity = "even"
		}
		leftCol = append(leftCol,
			formatMetricCol("Bits/s:", metrics.FormatBitsPerSecond(m.indicators.BitsPerSecond), colWidth),
			formatMetricCol("Steps:", fmt.Sprintf("%d (%.1f/s)", m.indicators.DoublingSteps, m.indicators.StepsPerSecond), colWidth),
		)
		rightCol = append(rightCol,
			formatMetricCol("Digits/s:", metrics.FormatDigitsPerSecond(m.indicators.DigitsPerSecond), colWidth),
			formatMetricCol("Parity:", parity, colWidth),
		)
	}

	for i := range leftCol {
		rows.WriteString("\n")
		rows.WriteString(leftCol[i])
		rows.WriteString(rightCol[i])
	}

	return panelStyle.
		Width(m.width - 2).
		Height(m.height - 2).
		Render(rows.String())
}

func formatMetricCol(label, value string, colWidth int) string {
	cell := fmt.Sprintf(" %s %s",
		metricLabelStyle.Render(fmt.Sprintf("%-12s", label)),
		metricValueStyle.Render(value))
	// Pad to fixed column width using lipgloss-aware width
	visible := lipgloss.Width(cell)
	if visible < colWidth {
		cell += strings.Repeat(" ", colWidth-visible)
	}
	return cell
}
