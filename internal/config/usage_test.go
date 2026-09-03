package config

import (
	"bytes"
	"flag"
	"os"
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
		// ERR-06: -version/-V and -h are handled outside the FlagSet
		// (HasVersionFlag / flag.ErrHelp) and must still appear in the help.
		"-version",
		"-h",
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

// TestUsageHonorsMachineAndQuiet pins audit L-05: ui.InitTheme runs after
// parsing, so --machine and -q had no effect on the help text and the usage
// block carried ANSI escapes into a stream the caller asked to keep clean.
func TestUsageHonorsMachineAndQuiet(t *testing.T) {
	// Not parallel: mutates NO_COLOR, which the usage closure reads.
	t.Setenv("NO_COLOR", "")
	os.Unsetenv("NO_COLOR")

	for _, args := range [][]string{
		{"--machine"},
		{"-q"},
		{"--quiet"},
	} {
		name := strings.Join(args, " ")
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			fs := flag.NewFlagSet("fibcalc", flag.ContinueOnError)
			fs.SetOutput(&buf)
			cfg := AppConfig{}
			registerFlags(fs, &cfg, []string{"fast"})
			setCustomUsage(fs)

			if err := fs.Parse(args); err != nil {
				t.Fatalf("Parse(%v) failed: %v", args, err)
			}
			fs.Usage()

			if strings.Contains(buf.String(), "\x1b[") {
				t.Errorf("usage under %q must not emit ANSI escapes, got:\n%q", name, buf.String())
			}
		})
	}

	t.Run("colored by default", func(t *testing.T) {
		var buf bytes.Buffer
		fs := flag.NewFlagSet("fibcalc", flag.ContinueOnError)
		fs.SetOutput(&buf)
		cfg := AppConfig{}
		registerFlags(fs, &cfg, []string{"fast"})
		setCustomUsage(fs)
		if err := fs.Parse([]string{"-n", "10"}); err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
		fs.Usage()

		if !strings.Contains(buf.String(), "\x1b[") {
			t.Error("without --machine/-q the usage should keep the themed output")
		}
	})
}
