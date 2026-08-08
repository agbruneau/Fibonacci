package tui

import (
	"context"
	"errors"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	apperrors "github.com/agbruneau/FibGo/internal/errors"
	"github.com/agbruneau/FibGo/internal/metrics"
)

// handleProgress applies a progress update to the chart, metrics and logs.
// No-op when paused, so the UI freezes cleanly while the underlying
// calculation keeps running.
func (m *Model) handleProgress(msg ProgressMsg) {
	if msg.Generation != m.generation {
		return
	}
	if m.paused {
		return
	}
	m.logs.AddProgressEntry(msg)
	m.chart.AddDataPoint(msg.AverageProgress, msg.ETA)
	m.metrics.UpdateProgress(msg.AverageProgress)
	// Refresh live indicators from progress data
	elapsed := time.Since(m.header.startTime)
	m.metrics.UpdateIndicators(metrics.ComputeLive(m.config.N, msg.AverageProgress, elapsed))
}

// handleFinalResult writes the final result to the logs and, when a numeric
// result is available, returns a follow-up command that computes the
// post-calculation indicators off the UI thread.
func (m *Model) handleFinalResult(msg FinalResultMsg) tea.Cmd {
	if msg.Generation != m.generation {
		return nil
	}
	m.logs.AddFinalResult(msg)
	if msg.Result.Result != nil {
		return computeIndicatorsCmd(msg)
	}
	return nil
}

// handleError records an error in the logs and flips header/footer to the
// "done with error" terminal state.
func (m *Model) handleError(msg ErrorMsg) {
	if msg.Generation != m.generation {
		return
	}
	m.logs.AddError(msg)
	m.footer.SetError(true)
	m.done = true
	m.header.SetDone()
	m.footer.SetDone(true)
}

// handleTick fires the periodic sampling commands when running. When paused
// we still re-arm the tick so resuming picks up immediately, but skip the
// expensive sampling work.
func (m *Model) handleTick() tea.Cmd {
	if m.done {
		return nil
	}
	if m.paused {
		return tickCmd()
	}
	return tea.Batch(sampleMemStatsCmd(), sampleSysStatsCmd(), tickCmd())
}

// handleCalculationComplete transitions the model to the terminal "done"
// state for the matching generation. Stale messages from a previous
// generation (after Reset) are dropped.
func (m *Model) handleCalculationComplete(msg CalculationCompleteMsg) {
	if msg.Generation != m.generation {
		return
	}
	m.done = true
	m.exitCode = msg.ExitCode
	m.header.SetDone()
	m.chart.SetDone(time.Since(m.header.startTime))
	m.footer.SetDone(true)
}

// handleContextCancelled marks the model done, maps the cancellation cause to
// the matching process exit code, and asks bubbletea to quit. Stale messages
// from a previous generation are ignored.
func (m *Model) handleContextCancelled(msg ContextCancelledMsg) tea.Cmd {
	if msg.Generation != m.generation {
		return nil
	}
	m.done = true
	switch {
	case errors.Is(msg.Err, context.DeadlineExceeded):
		m.exitCode = apperrors.ExitErrorTimeout
	case errors.Is(msg.Err, context.Canceled):
		m.exitCode = apperrors.ExitErrorCanceled
	}
	m.header.SetDone()
	m.footer.SetDone(true)
	return tea.Quit
}

// handleKey dispatches a key event. Returns the (possibly mutated) model and
// any follow-up command. Keeping the receiver as Model (value) preserves the
// bubbletea Update contract while letting individual cases use pointer
// helpers via &m where they need to mutate state in place.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keymap.Quit):
		if m.cancel != nil {
			m.cancel()
		}
		return m, tea.Quit

	case key.Matches(msg, m.keymap.Pause):
		m.paused = !m.paused
		m.footer.SetPaused(m.paused)
		return m, nil

	case key.Matches(msg, m.keymap.Reset):
		return m.handleReset()

	case key.Matches(msg, m.keymap.Up), key.Matches(msg, m.keymap.Down),
		key.Matches(msg, m.keymap.PageUp), key.Matches(msg, m.keymap.PageDown):
		m.logs.Update(msg)
		return m, nil
	}

	return m, nil
}

// handleReset cancels the current calculation, recreates a fresh context
// generation, resets all UI components and returns the bootstrap commands
// for the new run.
func (m Model) handleReset() (tea.Model, tea.Cmd) {
	if m.cancel != nil {
		m.cancel()
	}

	// Create a new context for the restarted calculation, with its own fresh
	// timeout budget (APP-05) rather than inheriting the original session's
	// absolute deadline via m.parentCtx.
	m.generation++
	ctx, cancel := context.WithTimeout(m.parentCtx, m.config.Timeout)
	m.ctx = ctx
	m.cancel = cancel

	// Reset all UI components.
	m.header.Reset()
	m.logs.Reset()
	m.chart.Reset()
	m.metrics = NewMetricsModel()
	m.metrics.SetSize(m.metricsWidth(), m.metricsHeight())
	m.footer.SetDone(false)
	m.footer.SetError(false)
	m.footer.SetPaused(false)
	m.done = false
	m.paused = false
	m.exitCode = apperrors.ExitSuccess

	// Restart calculation and watchers.
	return m, tea.Batch(
		tickCmd(),
		startCalculationCmd(m.ctx, m.ref, m.calculators, m.config, m.generation),
		watchContextCmd(m.ctx, m.generation),
	)
}
