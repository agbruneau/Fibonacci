package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/agbruneau/FibGo/internal/config"
	apperrors "github.com/agbruneau/FibGo/internal/errors"
	"github.com/agbruneau/FibGo/internal/orchestration"
)

// ExecutionState holds the execution-related fields of a TUI session.
type ExecutionState struct {
	ctx         context.Context
	cancel      context.CancelFunc
	calculators []orchestration.Calculator
	generation  uint64
	done        bool
	exitCode    int
}

// Model is the root bubbletea model for the TUI dashboard. It owns the five
// sub-components (header, logs, metrics, chart, footer) plus the shared
// execution state and layout dimensions; per-message handling lives in
// handlers.go and command builders live in commands.go so this file can stay
// focused on the router/composition concern.
type Model struct {
	header  HeaderModel
	logs    LogsModel
	metrics MetricsModel
	chart   ChartModel
	footer  FooterModel

	keymap KeyMap

	ExecutionState
	LayoutManager

	parentCtx context.Context
	config    config.AppConfig
	ref       *programRef
	paused    bool
}

// NewModel creates a new TUI model.
func NewModel(parentCtx context.Context, calculators []orchestration.Calculator, cfg config.AppConfig, version string) Model {
	algoNames := make([]string, len(calculators))
	for i, c := range calculators {
		algoNames[i] = c.Name()
	}

	ctx, cancel := context.WithCancel(parentCtx)

	logs := NewLogsModel(algoNames)
	logs.AddExecutionConfig(cfg)

	return Model{
		header:  NewHeaderModel(version),
		logs:    logs,
		metrics: NewMetricsModel(),
		chart:   NewChartModel(),
		footer:  NewFooterModel(),
		keymap:  DefaultKeyMap(),
		ExecutionState: ExecutionState{
			ctx:         ctx,
			cancel:      cancel,
			calculators: calculators,
			exitCode:    apperrors.ExitSuccess,
		},
		parentCtx: parentCtx,
		config:    cfg,
		ref:       &programRef{},
	}
}

// Init returns the initial commands.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		startCalculationCmd(m.ref, m.ctx, m.calculators, m.config, m.generation),
		watchContextCmd(m.ctx, m.generation),
	)
}

// Update is the bubbletea router: it dispatches each message type to a
// dedicated handler in handlers.go and returns the resulting model + cmd.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layoutPanels()
		return m, nil

	case ProgressMsg:
		m.handleProgress(msg)
		return m, nil

	case ProgressDoneMsg:
		// A-19: intentional drain sink. Completion is driven by
		// CalculationCompleteMsg; ProgressDoneMsg only signals that the
		// progress channel closed and carries no state transition. Kept as an
		// explicit no-op case so the closed-channel signal is consumed rather
		// than falling through to the default handler.
		return m, nil

	case ComparisonResultsMsg:
		m.logs.AddResults(msg.Results)
		return m, nil

	case FinalResultMsg:
		return m, m.handleFinalResult(msg)

	case IndicatorsMsg:
		m.metrics.UpdateIndicators(msg.Indicators)
		return m, nil

	case ErrorMsg:
		m.handleError(msg)
		return m, nil

	case TickMsg:
		return m, m.handleTick()

	case MemStatsMsg:
		m.metrics.UpdateMemStats(msg)
		return m, nil

	case SysStatsMsg:
		m.chart.UpdateSysStats(msg.CPUPercent, msg.MemPercent)
		return m, nil

	case CalculationCompleteMsg:
		m.handleCalculationComplete(msg)
		return m, nil

	case ContextCancelledMsg:
		return m, m.handleContextCancelled(msg)
	}

	return m, nil
}

// View renders the entire dashboard by composing each component's View().
// Composition adapts to terminal size (R4.10): narrow terminals stack the
// logs panel on top of the metrics + chart column instead of side-by-side.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	header := m.header.View()
	footer := m.footer.View()

	metrics := m.metrics.View()
	chart := m.chart.View()
	rightCol := lipgloss.JoinVertical(lipgloss.Left, metrics, chart)

	var body string
	if m.isNarrow() {
		// Single-column: logs on top, metrics + chart stacked below.
		logs := m.logs.renderToHeight(m.logsHeight())
		body = lipgloss.JoinVertical(lipgloss.Left, logs, rightCol)
	} else {
		// Side-by-side: logs on the left, right column on the right.
		logs := m.logs.renderToHeight(lipgloss.Height(rightCol))
		body = lipgloss.JoinHorizontal(lipgloss.Top, logs, rightCol)
	}

	// Full layout: header + body + footer.
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

// layoutPanels propagates the current terminal dimensions to every component
// using the LayoutManager's percentage/clamp rules. Adaptive behaviour
// (single-column / compact metrics) is encapsulated in LayoutManager.
func (m *Model) layoutPanels() {
	m.header.SetWidth(m.width)
	m.footer.SetWidth(m.width)
	m.logs.SetSize(m.logsWidth(), m.logsHeight())
	m.metrics.SetSize(m.rightWidth(), m.metricsHeight())
	m.chart.SetSize(m.rightWidth(), m.chartHeight())
}
