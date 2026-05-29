package cli

import (
	"fmt"
	"io"
	"runtime"

	"github.com/agbruneau/FibGo/internal/config"
	"github.com/agbruneau/FibGo/internal/ui"
)

// PrintExecutionConfig displays the current execution configuration to the user.
// It shows the target Fibonacci number, timeout, environment details, and
// optimization thresholds.
//
// Parameters:
//   - cfg: The application configuration.
//   - out: The writer for standard output.
func PrintExecutionConfig(cfg config.AppConfig, out io.Writer) {
	fmt.Fprintf(out, "--- Execution Configuration ---\n")
	fmt.Fprintf(out, "Calculating %sF(%d)%s with a timeout of %s%s%s.\n",
		ui.ColorMagenta(), cfg.N, ui.ColorReset(), ui.ColorYellow(), cfg.Timeout, ui.ColorReset())
	fmt.Fprintf(out, "Environment: %s%d%s logical processors, Go %s%s%s.\n",
		ui.ColorCyan(), runtime.NumCPU(), ui.ColorReset(), ui.ColorCyan(), runtime.Version(), ui.ColorReset())
	fmt.Fprintf(out, "Optimization thresholds: Parallelism=%s%d%s bits, FFT=%s%d%s bits.\n",
		ui.ColorCyan(), cfg.Threshold, ui.ColorReset(), ui.ColorCyan(), cfg.FFTThreshold, ui.ColorReset())
}

// PrintExecutionMode displays the execution mode (single algorithm vs comparison).
// names is the list of human-readable algorithm names that will run.
func PrintExecutionMode(names []string, out io.Writer) {
	var modeDesc string
	if len(names) > 1 {
		modeDesc = "Parallel comparison of all algorithms"
	} else {
		modeDesc = fmt.Sprintf("Single calculation with the %s%s%s algorithm",
			ui.ColorGreen(), names[0], ui.ColorReset())
	}
	fmt.Fprintf(out, "Execution mode: %s.\n", modeDesc)
	fmt.Fprintf(out, "\n--- Starting Execution ---\n")
}
