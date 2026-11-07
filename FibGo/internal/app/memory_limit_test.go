package app

import (
	"bytes"
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/agbruneau/FibGo/internal/apperrors"
	"github.com/agbruneau/FibGo/internal/config"
	"github.com/agbruneau/FibGo/internal/fibonacci"
	"github.com/agbruneau/FibGo/internal/progress"
)

// Regression tests for audit MEM-01.
//
// fibonacci.Options.MemoryLimitBytes and fibonacci.CanCalculate are documented
// as defense in depth behind config.ValidateMemoryBudget — a guard that holds
// "even when the validator is bypassed". It could not: no production caller
// ever set the field, so from the binary the guard was unreachable and only
// programmatic callers and the calculator's own unit tests could trip it.
//
// The tests below pin the wiring at the two seams that were empty: the CLI
// path, where the value must reach the Calculator, and the TUI path, where it
// must be on the config handed to tui.Run.

// optionsSpy records the Options it is called with. MockCalculator discards
// them, which is why it cannot cover this.
type optionsSpy struct {
	captured fibonacci.Options
}

func (s *optionsSpy) Name() string { return "spy" }

func (s *optionsSpy) Calculate(_ context.Context, progressChan chan<- progress.ProgressUpdate, calcIndex int, _ uint64, opts fibonacci.Options) (*big.Int, error) {
	s.captured = opts
	if progressChan != nil {
		progressChan <- progress.ProgressUpdate{CalculatorIndex: calcIndex, Value: 1.0}
	}
	return big.NewInt(55), nil
}

func newSpyApp(t *testing.T, memoryLimit string) (*Application, *optionsSpy) {
	t.Helper()
	spy := &optionsSpy{}
	return &Application{
		Config: config.AppConfig{
			N:           10,
			Algo:        "fast",
			Timeout:     time.Minute,
			MemoryLimit: memoryLimit,
			Quiet:       true,
		},
		Factory: fibonacci.NewTestFactory(map[string]fibonacci.Calculator{
			"fast": spy,
		}),
		ErrWriter: &bytes.Buffer{},
	}, spy
}

func TestMemoryLimitReachesCalculator(t *testing.T) {
	app, spy := newSpyApp(t, "8G")

	if code := app.runCalculate(context.Background(), &bytes.Buffer{}); code != apperrors.ExitSuccess {
		t.Fatalf("runCalculate exit code = %d, want %d", code, apperrors.ExitSuccess)
	}

	const want = uint64(8) << 30
	if spy.captured.MemoryLimitBytes != want {
		t.Errorf("Options.MemoryLimitBytes = %d, want %d", spy.captured.MemoryLimitBytes, want)
	}
}

// No --memory-limit must leave the guard disabled (0 means "no check"), not
// accidentally clamp the calculation to some default.
func TestNoMemoryLimitLeavesGuardDisabled(t *testing.T) {
	app, spy := newSpyApp(t, "")

	if code := app.runCalculate(context.Background(), &bytes.Buffer{}); code != apperrors.ExitSuccess {
		t.Fatalf("runCalculate exit code = %d, want %d", code, apperrors.ExitSuccess)
	}

	if spy.captured.MemoryLimitBytes != 0 {
		t.Errorf("Options.MemoryLimitBytes = %d, want 0 (guard disabled)", spy.captured.MemoryLimitBytes)
	}
}

// runTUI validates the budget and then hands a.Config to tui.Run, which builds
// its own Options per generation. Driving bubbletea headless is out of scope
// here, so pin the seam instead: after validation the parsed limit must be on
// the config the TUI receives.
func TestMemoryLimitResolvedOnConfigBeforeTUI(t *testing.T) {
	app, _ := newSpyApp(t, "512M")

	if code := app.validateMemoryBudget(&bytes.Buffer{}); code != apperrors.ExitSuccess {
		t.Fatalf("validateMemoryBudget exit code = %d, want %d", code, apperrors.ExitSuccess)
	}

	const want = uint64(512) << 20
	if app.Config.MemoryLimitBytes != want {
		t.Errorf("Config.MemoryLimitBytes = %d, want %d", app.Config.MemoryLimitBytes, want)
	}
}

// A limit smaller than the estimate must be refused before any calculator runs,
// with the config exit code.
func TestMemoryLimitBelowEstimateIsRefused(t *testing.T) {
	spy := &optionsSpy{}
	app := &Application{
		Config: config.AppConfig{
			N:           100_000_000,
			Algo:        "fast",
			Timeout:     time.Minute,
			MemoryLimit: "1K",
			Quiet:       true,
		},
		Factory: fibonacci.NewTestFactory(map[string]fibonacci.Calculator{
			"fast": spy,
		}),
		ErrWriter: &bytes.Buffer{},
	}

	if code := app.runCalculate(context.Background(), &bytes.Buffer{}); code != apperrors.ExitErrorConfig {
		t.Errorf("exit code = %d, want %d (config error)", code, apperrors.ExitErrorConfig)
	}
	if spy.captured.MemoryLimitBytes != 0 {
		t.Error("a calculator ran despite the budget being exceeded")
	}
}
