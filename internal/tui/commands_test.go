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
