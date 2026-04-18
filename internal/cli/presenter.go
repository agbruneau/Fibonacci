package cli

import (
	"fmt"
	"io"
	"sync"
	"time"

	apperrors "github.com/agbru/fibcalc/internal/errors"
	"github.com/agbru/fibcalc/internal/format"
	"github.com/agbru/fibcalc/internal/orchestration"
	"github.com/agbru/fibcalc/internal/progress"
	"github.com/agbru/fibcalc/internal/ui"
)

// CLIProgressReporter implements orchestration.ProgressReporter for CLI output.
// It wraps the DisplayProgress function to provide a spinner and progress bar
// display during calculations.
type CLIProgressReporter struct{}

// Verify that CLIProgressReporter implements orchestration.ProgressReporter.
var _ orchestration.ProgressReporter = CLIProgressReporter{}

// DisplayProgress displays a spinner and progress bar for ongoing calculations.
func (CLIProgressReporter) DisplayProgress(wg *sync.WaitGroup, progressChan <-chan progress.ProgressUpdate, numCalculators int, out io.Writer) {
	DisplayProgress(wg, progressChan, numCalculators, out)
}

// CLIResultPresenter implements orchestration.ResultPresenter for CLI output.
// It provides formatted, colorized output for calculation results in the
// command-line interface.
type CLIResultPresenter struct {
	// MachineOutput disables ANSI in error messages (use with [github.com/agbru/fibcalc/internal/ui.InitTheme]).
	MachineOutput bool
}

// Verify interface compliance.
var (
	_ orchestration.ResultPresenter   = CLIResultPresenter{}
	_ orchestration.DurationFormatter = CLIResultPresenter{}
	_ orchestration.ErrorHandler      = CLIResultPresenter{}
)

// PresentComparisonTable displays the comparison summary table with
// algorithm names, durations, and status in a formatted tabular layout.
// Uses manual padding to correctly handle ANSI color codes.
func (CLIResultPresenter) PresentComparisonTable(results []orchestration.CalculationResult, out io.Writer) {
	fmt.Fprintf(out, "\n--- Comparison Summary ---\n")

	// Find the maximum algorithm name width for proper alignment
	maxNameLen := 9     // "Algorithm" header length
	maxDurationLen := 8 // "Duration" header length
	for _, res := range results {
		if len(res.Name) > maxNameLen {
			maxNameLen = len(res.Name)
		}
		duration := format.FormatExecutionDuration(res.Duration)
		if res.Duration == 0 {
			duration = "< 1µs"
		}
		if len(duration) > maxDurationLen {
			maxDurationLen = len(duration)
		}
	}

	// Print header with proper padding
	fmt.Fprintf(out, "%sAlgorithm%s%s   %sDuration%s%s   %sStatus%s\n",
		ui.ColorUnderline(), ui.ColorReset(), padRight("", maxNameLen-9),
		ui.ColorUnderline(), ui.ColorReset(), padRight("", maxDurationLen-8),
		ui.ColorUnderline(), ui.ColorReset())

	// Print each result row
	for _, res := range results {
		var status string
		if res.Err != nil {
			status = fmt.Sprintf("%s❌ Failure (%v)%s", ui.ColorRed(), res.Err, ui.ColorReset())
		} else {
			status = fmt.Sprintf("%s✅ Success%s", ui.ColorGreen(), ui.ColorReset())
		}
		duration := format.FormatExecutionDuration(res.Duration)
		if res.Duration == 0 {
			duration = "< 1µs"
		}
		fmt.Fprintf(out, "%s%s%s%s   %s%s%s%s   %s\n",
			ui.ColorBlue(), res.Name, ui.ColorReset(), padRight("", maxNameLen-len(res.Name)),
			ui.ColorYellow(), duration, ui.ColorReset(), padRight("", maxDurationLen-len(duration)),
			status)
	}
}

// padRight returns a string of spaces with the given length.
func padRight(s string, length int) string {
	if length <= 0 {
		return s
	}
	return s + fmt.Sprintf("%*s", length, "")
}

// PresentResult displays the final calculation result using the CLI's
// DisplayResult function.
func (CLIResultPresenter) PresentResult(result orchestration.CalculationResult, n uint64, verbose, details, showValue bool, out io.Writer) {
	DisplayResult(result.Result, n, result.Duration, verbose, details, showValue, out)
}

// FormatDuration formats a duration for display using the CLI's standard
// duration formatting.
func (CLIResultPresenter) FormatDuration(d time.Duration) string {
	return format.FormatExecutionDuration(d)
}

// HandleError handles calculation errors and returns an appropriate exit code.
func (p CLIResultPresenter) HandleError(err error, duration time.Duration, out io.Writer) int {
	colors := apperrors.ColorProvider(CLIColorProvider{})
	if p.MachineOutput {
		colors = apperrors.DefaultColorProvider{}
	}
	return apperrors.HandleCalculationError(err, duration, out, colors)
}

// DisplayMemoryStats shows memory statistics after a calculation.
func DisplayMemoryStats(heapAlloc, totalAlloc uint64, numGC uint32, pauseTotalNs uint64, out io.Writer) {
	fmt.Fprintf(out, "\nMemory Stats:\n")
	fmt.Fprintf(out, "  Peak heap:       %s\n", format.FormatBytes(heapAlloc))
	fmt.Fprintf(out, "  Total allocated: %s\n", format.FormatBytes(totalAlloc))
	fmt.Fprintf(out, "  GC cycles:       %d\n", numGC)
	if pauseTotalNs > 0 {
		fmt.Fprintf(out, "  GC pause total:  %.2fms\n", float64(pauseTotalNs)/1e6)
	} else {
		fmt.Fprintf(out, "  GC pause total:  0ms (GC disabled)\n")
	}
}
