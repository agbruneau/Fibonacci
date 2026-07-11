package tui

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"

	"github.com/agbruneau/FibGo/internal/config"
	apperrors "github.com/agbruneau/FibGo/internal/errors"
	"github.com/agbruneau/FibGo/internal/metrics"
	"github.com/agbruneau/FibGo/internal/orchestration"
)

// runProgram is a seam so tests can inject p.Run failures (ERR-04) without
// needing a real TTY.
var runProgram = func(p *tea.Program) (tea.Model, error) { return p.Run() }

// Run is the public entry point for the TUI mode.
// It creates the bubbletea program, runs it, and returns the exit code.
// Program-level failures (e.g. TTY unavailable) are reported on errOut.
func Run(ctx context.Context, calculators []orchestration.Calculator, cfg config.AppConfig, version string, errOut io.Writer) int {
	// Rebuild styles from the current ui theme (set by app.Run via InitTheme).
	initTUIStyles()

	model := NewModel(ctx, calculators, cfg, version)
	defer model.cancel()

	p := tea.NewProgram(model, tea.WithAltScreen())
	// Inject the program reference before running so bridge goroutines can Send.
	model.ref.SetProgram(p)

	finalModel, err := runProgram(p)
	if err != nil {
		fmt.Fprintf(errOut, "TUI error: %v\n", err)
		return apperrors.ExitErrorGeneric
	}

	if m, ok := finalModel.(Model); ok {
		m.cancel()
		return m.exitCode
	}
	return apperrors.ExitSuccess
}

// startCalculationCmd returns a tea.Cmd that launches the orchestration.
func startCalculationCmd(ref *programRef, ctx context.Context, calculators []orchestration.Calculator, cfg config.AppConfig, gen uint64) tea.Cmd {
	return func() tea.Msg {
		progressReporter := &TUIProgressReporter{ref: ref, gen: gen}
		presenter := &TUIResultPresenter{ref: ref, gen: gen}

		opts := orchestration.Options{
			ParallelThreshold: cfg.Threshold,
			FFTThreshold:      cfg.FFTThreshold,
			StrassenThreshold: cfg.StrassenThreshold,
			GCMode:            cfg.GCControl,
		}
		results := orchestration.ExecuteCalculations(ctx, orchestration.ExecutionConfig{
			Calculators:      calculators,
			N:                cfg.N,
			Opts:             opts,
			ProgressReporter: progressReporter,
			Out:              io.Discard,
		})
		presOpts := orchestration.PresentationOptions{
			N:         cfg.N,
			Verbose:   cfg.Verbose,
			Details:   cfg.Details,
			ShowValue: cfg.ShowValue,
		}
		exitCode := orchestration.AnalyzeComparisonResults(results, presOpts, presenter, presenter, io.Discard, io.Discard)

		return CalculationCompleteMsg{ExitCode: exitCode, Generation: gen}
	}
}

// tickCmd returns a command that sends a TickMsg after 500ms.
func tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// sampleMemStatsCmd reads runtime memory stats and returns a MemStatsMsg.
func sampleMemStatsCmd() tea.Cmd {
	return func() tea.Msg {
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		return MemStatsMsg{
			Alloc:        ms.Alloc,
			HeapSys:      ms.HeapSys,
			NumGC:        ms.NumGC,
			PauseTotalNs: ms.PauseTotalNs,
			NumGoroutine: runtime.NumGoroutine(),
		}
	}
}

// sampleSysStatsCmd reads system-wide CPU and memory stats and returns a
// SysStatsMsg. CPU uses interval=0 (delta since last call); zero values on
// error. Inlined from the former internal/metrics/system package — tui was
// its only caller (audit Fable5 DEAD-05).
func sampleSysStatsCmd() tea.Cmd {
	return func() tea.Msg {
		var msg SysStatsMsg
		if cpuPcts, err := cpu.Percent(0, false); err == nil && len(cpuPcts) > 0 {
			msg.CPUPercent = cpuPcts[0]
		}
		if vmem, err := mem.VirtualMemory(); err == nil && vmem != nil {
			msg.MemPercent = vmem.UsedPercent
		}
		return msg
	}
}

// computeIndicatorsCmd returns a tea.Cmd that computes post-calculation
// indicators asynchronously, ensuring no impact on the UI thread.
func computeIndicatorsCmd(msg FinalResultMsg) tea.Cmd {
	return func() tea.Msg {
		ind := metrics.Compute(msg.Result.Result, msg.N, msg.Result.Duration)
		return IndicatorsMsg{Indicators: ind, Generation: msg.Generation}
	}
}

// watchContextCmd waits for context cancellation and sends a message.
func watchContextCmd(ctx context.Context, gen uint64) tea.Cmd {
	return func() tea.Msg {
		<-ctx.Done()
		return ContextCancelledMsg{Err: ctx.Err(), Generation: gen}
	}
}
