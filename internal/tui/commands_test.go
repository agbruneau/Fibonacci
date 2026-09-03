package tui

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/agbruneau/FibGo/internal/config"
	apperrors "github.com/agbruneau/FibGo/internal/errors"
)

// TestRun_ReportsProgramError pins ERR-04: a p.Run failure must be reported
// on errOut with a generic exit code, never swallowed into a mute exit 1.
//
// Not parallel: swaps the package-level runProgram seam.
func TestRun_ReportsProgramError(t *testing.T) {
	orig := runProgram
	t.Cleanup(func() { runProgram = orig })
	runProgram = func(p *tea.Program) (tea.Model, error) {
		return nil, errors.New("tty unavailable")
	}

	var errBuf bytes.Buffer
	code := Run(context.Background(), nil, config.AppConfig{Timeout: time.Second}, "test", &errBuf)

	if code != apperrors.ExitErrorGeneric {
		t.Fatalf("expected exit %d, got %d", apperrors.ExitErrorGeneric, code)
	}
	if !strings.Contains(errBuf.String(), "tty unavailable") {
		t.Fatalf("p.Run error must reach errOut, got: %q", errBuf.String())
	}
}

// TestRun_InterruptedExitsCanceled pins audit M-07: when stdin is not a TTY,
// ^C reaches bubbletea's signal handler rather than the Quit key binding and
// p.Run returns tea.ErrInterrupted. That is a cancellation, not a TUI failure,
// so it must map to the documented SIGINT code (APP-04) and print nothing on
// errOut.
//
// Not parallel: swaps the package-level runProgram seam.
func TestRun_InterruptedExitsCanceled(t *testing.T) {
	orig := runProgram
	t.Cleanup(func() { runProgram = orig })
	runProgram = func(p *tea.Program) (tea.Model, error) {
		return nil, tea.ErrInterrupted
	}

	var errBuf bytes.Buffer
	code := Run(context.Background(), nil, config.AppConfig{Timeout: time.Second}, "test", &errBuf)

	if code != apperrors.ExitErrorCanceled {
		t.Fatalf("expected exit %d for an interrupted program, got %d", apperrors.ExitErrorCanceled, code)
	}
	if errBuf.Len() != 0 {
		t.Fatalf("interruption is not an error to report, got: %q", errBuf.String())
	}
}
