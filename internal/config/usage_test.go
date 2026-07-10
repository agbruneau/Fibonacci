package config

import (
	"bytes"
	"flag"
	"strings"
	"testing"
)

// TestSetCustomUsage_NoColor pins the NO_COLOR contract: the usage text must
// carry no ANSI escapes while keeping the header, flag list and meaningful
// defaults. Uses t.Setenv (process-global env), so no t.Parallel.
func TestSetCustomUsage_NoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	fs := flag.NewFlagSet("fibcalc-usage-test", flag.ContinueOnError)
	var cfg AppConfig
	registerFlags(fs, &cfg, []string{"fast", "matrix"})
	setCustomUsage(fs)

	var buf bytes.Buffer
	fs.SetOutput(&buf)
	fs.Usage()

	out := buf.String()
	if strings.Contains(out, "\x1b[") {
		t.Error("usage output contains ANSI escape sequences despite NO_COLOR")
	}
	for _, want := range []string{
		"Fibonacci Calculator",
		"Usage:",
		"fibcalc-usage-test",
		"Flags:",
		"-algo",
		"(default all)",
		"(default 5m0s)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("usage output missing %q\noutput:\n%s", want, out)
		}
	}
	// Zero/false defaults are intentionally hidden to keep the listing readable.
	if strings.Contains(out, "(default false)") || strings.Contains(out, "(default 0)") {
		t.Error("usage output should not print zero/false defaults")
	}
}
