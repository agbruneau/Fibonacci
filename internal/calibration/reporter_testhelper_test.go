package calibration

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// testReporter renders calibration narration as plain lines into a writer.
//
// With the presentation port (audit ARC-01) the package emits wording only, and
// this reporter is the plain-text adapter that makes the wording assertable
// without stripping ANSI escapes before matching a threshold value.
// The CLI adapter (internal/cli.CalibrationReporter) is what a user sees.
type testReporter struct {
	out io.Writer
}

func newTestReporter(out io.Writer) *testReporter { return &testReporter{out: out} }

func (r *testReporter) Notice(format string, args ...any) {
	fmt.Fprintf(r.out, "%s\n", fmt.Sprintf(format, args...))
}

func (r *testReporter) Warning(format string, args ...any) {
	fmt.Fprintf(r.out, "Warning: %s\n", fmt.Sprintf(format, args...))
}

func (r *testReporter) Error(format string, args ...any) {
	fmt.Fprintf(r.out, "Error: %s\n", fmt.Sprintf(format, args...))
}

func (r *testReporter) Summary(results []PassResult, best int) {
	var b strings.Builder
	b.WriteString("--- Calibration Summary ---\n")
	for _, res := range results {
		label := fmt.Sprintf("%d bits", res.Threshold)
		if res.Threshold < 0 {
			label = "Sequential"
		}
		switch {
		case res.Err != nil:
			fmt.Fprintf(&b, "  %-12s | N/A\n", label)
		case res.Threshold == best:
			fmt.Fprintf(&b, "  %-12s | %s (Optimal)\n", label, res.Duration.Round(time.Nanosecond))
		default:
			fmt.Fprintf(&b, "  %-12s | %s\n", label, res.Duration.Round(time.Nanosecond))
		}
	}
	fmt.Fprint(r.out, b.String())
}
