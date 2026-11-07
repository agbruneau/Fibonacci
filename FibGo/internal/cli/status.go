// This file renders calculation failures for a terminal reader.

package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/agbruneau/FibGo/internal/apperrors"
	"github.com/agbruneau/FibGo/internal/ui"
)

// WriteCalculationStatus writes the human-facing "Status: …" line for a failed
// calculation and returns the exit code that matches it.
//
// It is the presentation half of calculation-error handling (audit API-04): the
// wording, the color and the writer live here, in the CLI
// adapter, while internal/apperrors keeps only ExitCodeFor. A caller that wants
// just the code — internal/tui, which renders its own error panel — calls
// ExitCodeFor and never reaches this function, instead of passing io.Discard
// and a nil color provider to get a number out of a printer.
//
// Color comes straight from the ui theme, with no ColorProvider indirection:
// ui.Color* already returns empty strings after ui.InitTheme(true), which
// app.Run calls for both --quiet and --machine, so a "use the no-color provider
// under --machine" branch at every call site would select an empty result that
// is already empty.
func WriteCalculationStatus(out io.Writer, err error, duration time.Duration) int {
	if err == nil {
		return apperrors.ExitSuccess
	}

	suffix := ""
	if duration > 0 {
		suffix = fmt.Sprintf(" after %s%s%s", ui.ColorYellow(), duration, ui.ColorReset())
	}

	switch {
	case errors.Is(err, context.DeadlineExceeded):
		fmt.Fprintf(out, "Status: Failure (Timeout). The execution limit was reached%s.\n", suffix)
	case errors.Is(err, context.Canceled):
		fmt.Fprintf(out, "%sStatus: Canceled%s.%s\n", ui.ColorYellow(), suffix, ui.ColorReset())
	default:
		fmt.Fprintf(out, "Status: Failure. An unexpected error occurred: %v\n", err)
	}

	if d := apperrors.CalculationDiagnostic(err); d != "" {
		fmt.Fprintf(out, "Diagnostic: %s\n", d)
	}

	return apperrors.ExitCodeFor(err)
}
