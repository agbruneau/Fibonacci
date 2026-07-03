package calibration

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/agbruneau/FibGo/internal/config"
	"github.com/agbruneau/FibGo/internal/format"
	"github.com/agbruneau/FibGo/internal/ui"
)

// printCalibrationResults formats and prints the calibration results table.
func printCalibrationResults(out io.Writer, results []calibrationResult, bestThreshold int) {
	fmt.Fprintf(out, "\n--- Calibration Summary ---\n")
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	fmt.Fprintf(tw, "  %sThreshold%s    │ %sExecution Time%s\n", ui.ColorUnderline(), ui.ColorReset(), ui.ColorUnderline(), ui.ColorReset())
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
		if res.Threshold == bestThreshold && res.Err == nil {
			highlight = fmt.Sprintf(" %s(Optimal)%s", ui.ColorGreen(), ui.ColorReset())
		}
		fmt.Fprintf(tw, "  %s%-12s%s │ %s%s%s%s\n", ui.ColorCyan(), thresholdLabel, ui.ColorReset(), ui.ColorYellow(), durationStr, ui.ColorReset(), highlight)
	}
	// tabwriter.Flush can only fail when the underlying io.Writer fails. We
	// report the failure to stderr but do not propagate: this is a cosmetic
	// summary printer invoked from a CLI command and its caller has no
	// meaningful recovery beyond what the writer itself has already done.
	if err := tw.Flush(); err != nil {
		fmt.Fprintf(out, "%scalibration summary: tabwriter flush failed: %v%s\n",
			ui.ColorRed(), err, ui.ColorReset())
	}
}

// printCalibrationOutput prints the calibration results.
//
// Parameters:
//   - cfg: The updated configuration with calibration results.
//   - out: The writer for output.
func printCalibrationOutput(cfg config.AppConfig, out io.Writer) {
	fmt.Fprintf(out, "%sAuto-calibration%s: parallelism=%s%d%s bits, FFT=%s%d%s bits, Strassen=%s%d%s bits\n",
		ui.ColorGreen(), ui.ColorReset(),
		ui.ColorYellow(), cfg.Threshold, ui.ColorReset(),
		ui.ColorYellow(), cfg.FFTThreshold, ui.ColorReset(),
		ui.ColorYellow(), cfg.StrassenThreshold, ui.ColorReset())
}
