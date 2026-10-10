package cli

import (
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/agbruneau/FibGo/internal/format"
	"github.com/agbruneau/FibGo/internal/orchestration"
	"github.com/agbruneau/FibGo/internal/ui"
)

// CLIResultPresenter implements orchestration.ResultPresenter for CLI output.
// It provides formatted, colorized output for calculation results in the
// command-line interface.
type CLIResultPresenter struct {
	// MachineOutput disables ANSI in error messages (use with [github.com/agbruneau/FibGo/internal/ui.InitTheme]).
	MachineOutput bool
}

// Verify interface compliance.
var (
	_ orchestration.ResultPresenter = CLIResultPresenter{}
	_ orchestration.ErrorHandler    = CLIResultPresenter{}
)

// PresentComparisonTable displays the comparison summary table with
// algorithm names, durations, and status in a formatted tabular layout.
// Uses manual padding to correctly handle ANSI color codes.
func (CLIResultPresenter) PresentComparisonTable(results []orchestration.CalculationResult, out io.Writer) {
	fmt.Fprintf(out, "\n--- Comparison Summary ---\n")

	// Find the maximum algorithm name width for proper alignment
	maxNameLen := 9     // "Algorithm" header length
	maxDurationLen := 8 // "Duration" header length
	durations := make([]string, len(results))
	for i, res := range results {
		if len(res.Name) > maxNameLen {
			maxNameLen = len(res.Name)
		}
		durations[i] = format.FormatExecutionDuration(res.Duration)
		if res.Duration == 0 {
			durations[i] = "< 1µs"
		}
		if utf8.RuneCountInString(durations[i]) > maxDurationLen {
			maxDurationLen = utf8.RuneCountInString(durations[i])
		}
	}
	pad := func(n int) string { return strings.Repeat(" ", max(n, 0)) }

	// Print header with proper padding
	fmt.Fprintf(out, "%sAlgorithm%s%s   %sDuration%s%s   %sStatus%s\n",
		ui.ColorUnderline(), ui.ColorReset(), pad(maxNameLen-9),
		ui.ColorUnderline(), ui.ColorReset(), pad(maxDurationLen-8),
		ui.ColorUnderline(), ui.ColorReset())

	// Print each result row
	for i, res := range results {
		var status string
		if res.Err != nil {
			status = fmt.Sprintf("%s❌ Failure (%v)%s", ui.ColorRed(), res.Err, ui.ColorReset())
		} else {
			status = fmt.Sprintf("%s✅ Success%s", ui.ColorGreen(), ui.ColorReset())
		}
		fmt.Fprintf(out, "%s%s%s%s   %s%s%s%s   %s\n",
			ui.ColorBlue(), res.Name, ui.ColorReset(), pad(maxNameLen-len(res.Name)),
			ui.ColorYellow(), durations[i], ui.ColorReset(), pad(maxDurationLen-utf8.RuneCountInString(durations[i])),
			status)
	}
}

// PresentResult displays the final calculation result using the CLI's
// DisplayResult function.
func (CLIResultPresenter) PresentResult(result orchestration.CalculationResult, n uint64, verbose, details, showValue bool, out io.Writer) {
	DisplayResult(result.Result, n, result.Duration, verbose, details, showValue, out)
}

// HandleError handles calculation errors and returns an appropriate exit code.
//
// MachineOutput is not consulted here: app.Run passes --machine to
// ui.InitTheme, which installs the no-color theme, so ui.Color* already
// returns empty strings in that mode. The former "pick the no-color provider
// under --machine" branch selected an empty result that was already empty
// (audit API-04).
func (CLIResultPresenter) HandleError(err error, duration time.Duration, out io.Writer) int {
	return WriteCalculationStatus(out, err, duration)
}
