package tui

import (
	"time"

	"github.com/agbruneau/FibGo/internal/metrics"
	"github.com/agbruneau/FibGo/internal/orchestration"
)

// ProgressMsg carries a progress update from a calculator to the TUI.
type ProgressMsg struct {
	CalculatorIndex int
	Value           float64
	AverageProgress float64
	ETA             time.Duration
	// Generation tags the run that emitted this update so the model can drop
	// messages from a previous run still in flight after a Restart.
	Generation uint64
}

// ProgressDoneMsg signals that the progress channel has been closed.
type ProgressDoneMsg struct{}

// ComparisonResultsMsg carries comparison results for display in the logs.
type ComparisonResultsMsg struct {
	Results    []orchestration.CalculationResult
	Generation uint64
}

// FinalResultMsg carries the final calculation result for display.
type FinalResultMsg struct {
	Result     orchestration.CalculationResult
	N          uint64
	Verbose    bool
	Details    bool
	ShowValue  bool
	Generation uint64
}

// ErrorMsg carries an error from the calculation.
type ErrorMsg struct {
	Err        error
	Duration   time.Duration
	Generation uint64
}

// TickMsg triggers periodic metric sampling.
type TickMsg time.Time

// MemStatsMsg carries runtime memory statistics.
type MemStatsMsg struct {
	Alloc        uint64
	HeapSys      uint64
	NumGC        uint32
	PauseTotalNs uint64
	NumGoroutine int
}

// CalculationCompleteMsg signals that all calculations have finished.
type CalculationCompleteMsg struct {
	ExitCode   int
	Generation uint64
}

// SysStatsMsg carries system-wide CPU and memory usage percentages.
type SysStatsMsg struct {
	CPUPercent float64 // 0.0 .. 100.0
	MemPercent float64 // 0.0 .. 100.0
}

// IndicatorsMsg carries post-calculation indicators of interest for display.
type IndicatorsMsg struct {
	Indicators *metrics.Indicators
	// Generation tags the run that emitted this message so the model can drop
	// stragglers from a previous run still in flight after a Reset.
	Generation uint64
}

// ContextCancelledMsg signals that the context was canceled.
type ContextCancelledMsg struct {
	Err        error
	Generation uint64
}
