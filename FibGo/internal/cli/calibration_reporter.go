// This file implements calibration's presentation port for the terminal.

package cli

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/agbruneau/FibGo/internal/calibration"
	"github.com/agbruneau/FibGo/internal/format"
	"github.com/agbruneau/FibGo/internal/ui"
)

// CalibrationReporter renders calibration narration for a terminal reader. It
// is the CLI-side implementation of calibration.Reporter (audit ARC-01).
//
// The colors and the summary table live here so that internal/calibration, an
// application-layer package, does no presentation and imports neither
// internal/ui nor internal/format. The package owns the wording; this adapter
// owns the decision of how it looks.
//
// Emphasis is per line, by message kind, rather than per value: a
// recommendation is a green line, with no threshold picked out in a second
// color. The table keeps its per-column coloring, which is where the
// highlighting actually helps.
type CalibrationReporter struct {
	out io.Writer
}

// Compile-time check that the adapter satisfies the port.
var _ calibration.Reporter = (*CalibrationReporter)(nil)

// NewCalibrationReporter returns a reporter writing to out.
func NewCalibrationReporter(out io.Writer) *CalibrationReporter {
	return &CalibrationReporter{out: out}
}

// Notice renders ordinary progress in the theme's success color.
func (r *CalibrationReporter) Notice(msg string, args ...any) {
	fmt.Fprintf(r.out, "%s%s%s\n", ui.ColorGreen(), fmt.Sprintf(msg, args...), ui.ColorReset())
}

// Warning renders a recoverable problem in the theme's warning color.
func (r *CalibrationReporter) Warning(msg string, args ...any) {
	fmt.Fprintf(r.out, "%sWarning: %s%s\n", ui.ColorYellow(), fmt.Sprintf(msg, args...), ui.ColorReset())
}

// Error renders a run-ending condition in the theme's error color.
func (r *CalibrationReporter) Error(msg string, args ...any) {
	fmt.Fprintf(r.out, "%sError: %s%s\n", ui.ColorRed(), fmt.Sprintf(msg, args...), ui.ColorReset())
}

// Summary renders the completed sweep as an aligned table, one row per
// threshold, with the winning row marked.
//
// Moved here verbatim from calibration/io.go:printCalibrationResults, which is
// what made that package depend on text/tabwriter, internal/format and
// internal/ui to produce a table it had no business formatting.
func (r *CalibrationReporter) Summary(results []calibration.PassResult, best int) {
	fmt.Fprintf(r.out, "\n--- Calibration Summary ---\n")

	tw := tabwriter.NewWriter(r.out, 0, 0, 3, ' ', 0)
	fmt.Fprintf(tw, "  %sThreshold%s    │ %sExecution Time%s\n",
		ui.ColorUnderline(), ui.ColorReset(), ui.ColorUnderline(), ui.ColorReset())
	fmt.Fprintf(tw, "  %s┼%s\n", strings.Repeat("─", 14), strings.Repeat("─", 25))

	for _, res := range results {
		thresholdLabel := fmt.Sprintf("%d bits", res.Threshold)
		if res.Threshold < 0 {
			thresholdLabel = "Sequential"
		}

		durationStr := fmt.Sprintf("%sN/A%s", ui.ColorRed(), ui.ColorReset())
		if res.Err == nil {
			durationStr = format.FormatExecutionDuration(res.Duration)
			if res.Duration == 0 {
				durationStr = "< 1µs"
			}
		}

		highlight := ""
		if res.Threshold == best && res.Err == nil {
			highlight = fmt.Sprintf(" %s(Optimal)%s", ui.ColorGreen(), ui.ColorReset())
		}

		fmt.Fprintf(tw, "  %s%-12s%s │ %s%s%s%s\n",
			ui.ColorCyan(), thresholdLabel, ui.ColorReset(),
			ui.ColorYellow(), durationStr, ui.ColorReset(), highlight)
	}

	// tabwriter.Flush can only fail when the underlying io.Writer fails. Report
	// it, but do not propagate: this is a cosmetic summary printer and its
	// caller has no recovery beyond what the writer has already done.
	if err := tw.Flush(); err != nil {
		fmt.Fprintf(r.out, "%scalibration summary: tabwriter flush failed: %v%s\n",
			ui.ColorRed(), err, ui.ColorReset())
	}
}
