package tui

import (
	"io"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	apperrors "github.com/agbru/fibcalc/internal/errors"
	"github.com/agbru/fibcalc/internal/format"
	"github.com/agbru/fibcalc/internal/orchestration"
	"github.com/agbru/fibcalc/internal/progress"
)

// programRef is a shared reference to the tea.Program.
// Because bubbletea copies the model on every Update, we need a pointer
// that survives copies so the bridge goroutines can send messages.
type programRef struct {
	mu      sync.RWMutex
	program *tea.Program
}

// SetProgram sets the tea.Program reference (thread-safe).
func (r *programRef) SetProgram(p *tea.Program) {
	r.mu.Lock()
	r.program = p
	r.mu.Unlock()
}

// Send sends a message to the bubbletea program (thread-safe).
func (r *programRef) Send(msg tea.Msg) {
	r.mu.RLock()
	p := r.program
	r.mu.RUnlock()
	if p != nil {
		p.Send(msg)
	}
}

// TUIProgressReporter implements orchestration.ProgressReporter.
// It drains the progress channel and forwards updates as bubbletea messages.
type TUIProgressReporter struct {
	ref *programRef
}

// Verify interface compliance.
var _ orchestration.ProgressReporter = (*TUIProgressReporter)(nil)

// DisplayProgress drains the progress channel and sends ProgressMsg to the TUI.
func (t *TUIProgressReporter) DisplayProgress(wg *sync.WaitGroup, progressChan <-chan progress.ProgressUpdate, numCalculators int, _ io.Writer) {
	defer wg.Done()

	agg := orchestration.NewProgressAggregator(numCalculators)
	if agg == nil {
		orchestration.DrainChannel(progressChan)
		return
	}

	for update := range progressChan {
		ap := agg.Update(update)
		t.ref.Send(ProgressMsg{
			CalculatorIndex: ap.CalculatorIndex,
			Value:           ap.Value,
			AverageProgress: ap.AverageProgress,
			ETA:             ap.ETA,
		})
	}
	t.ref.Send(ProgressDoneMsg{})
}

// TUIResultPresenter implements orchestration.ResultPresenter.
// It sends result messages to the TUI instead of writing to stdout.
type TUIResultPresenter struct {
	ref *programRef
}

// Verify interface compliance.
var (
	_ orchestration.ResultPresenter   = (*TUIResultPresenter)(nil)
	_ orchestration.DurationFormatter = (*TUIResultPresenter)(nil)
	_ orchestration.ErrorHandler      = (*TUIResultPresenter)(nil)
)

// PresentComparisonTable sends comparison results to the TUI.
func (t *TUIResultPresenter) PresentComparisonTable(results []orchestration.CalculationResult, _ io.Writer) {
	t.ref.Send(ComparisonResultsMsg{Results: results})
}

// PresentResult sends the final result to the TUI.
func (t *TUIResultPresenter) PresentResult(result orchestration.CalculationResult, n uint64, verbose, details, showValue bool, _ io.Writer) {
	t.ref.Send(FinalResultMsg{
		Result:    result,
		N:         n,
		Verbose:   verbose,
		Details:   details,
		ShowValue: showValue,
	})
}

// FormatDuration delegates to the CLI formatter.
func (t *TUIResultPresenter) FormatDuration(d time.Duration) string {
	return format.FormatExecutionDuration(d)
}

// HandleError sends an error message to the TUI and returns the exit code.
func (t *TUIResultPresenter) HandleError(err error, duration time.Duration, _ io.Writer) int {
	t.ref.Send(ErrorMsg{Err: err, Duration: duration})
	return apperrors.HandleCalculationError(err, duration, io.Discard, nil)
}
