package completion

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/agbruneau/FibGo/internal/config"
)

// completionOnly lists registry entries that intentionally have no counterpart
// in config.registerFlags because they are handled outside the FlagSet:
// --version/-V via app.HasVersionFlag and --help/-h via flag.ErrHelp.
var completionOnly = map[string]bool{"help": true, "h": true, "version": true, "V": true}

// TestFlagRegistryInSyncWithConfig is the guard that keeps shell completion from
// silently drifting from the registered CLI flag set. Every flag in config (the
// canonical source) must be completable, and every completion entry must map to
// a registered flag (or be an explicitly allowlisted completion-only flag). Before
// this guard, six shipped flags (verbose, calculate/c, tui, last-digits,
// memory-limit, gc-control) were missing from completion with nothing to catch
// the divergence.
func TestFlagRegistryInSyncWithConfig(t *testing.T) {
	t.Parallel()

	registry := map[string]bool{}
	for _, f := range flagRegistry {
		if f.Long != "" {
			registry[f.Long] = true
		}
		if f.Short != "" {
			registry[f.Short] = true
		}
	}

	registered := map[string]bool{}
	for _, name := range config.FlagNames() {
		registered[name] = true
	}

	for name := range registered {
		if !registry[name] {
			t.Errorf("flag %q is registered in config but missing from completion.flagRegistry", name)
		}
	}
	for name := range registry {
		if !registered[name] && !completionOnly[name] {
			t.Errorf("completion.flagRegistry has %q with no matching config flag (stale entry)", name)
		}
	}
}

// TestRegistrySubsetOfGeneratedScript is the guard that closes the whole bug
// class behind APP-03: a flag present in flagRegistry but silently dropped by
// a shell generator (as fish dropped six flags via a hand-maintained section
// filter). For every shell, every registry entry must have its short (-x) and
// long (--xxx) forms appear literally in the generated script.
func TestRegistrySubsetOfGeneratedScript(t *testing.T) {
	t.Parallel()

	generators := map[string]func(*bytes.Buffer) error{
		"bash": func(buf *bytes.Buffer) error { return GenerateBash(buf, []string{"fast-doubling"}) },
		"zsh":  func(buf *bytes.Buffer) error { return GenerateZsh(buf, []string{"fast-doubling"}) },
		"fish": func(buf *bytes.Buffer) error { return GenerateFish(buf, []string{"fast-doubling"}) },
		"powershell": func(buf *bytes.Buffer) error {
			return GeneratePowerShell(buf, []string{"fast-doubling"})
		},
	}

	for shell, gen := range generators {
		t.Run(shell, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			if err := gen(&buf); err != nil {
				t.Fatalf("generate %s failed: %v", shell, err)
			}
			script := buf.String()

			// Long/short flags are rendered either concatenated ("--long",
			// "-x", used by bash/zsh/powershell) or space-separated ("-l
			// long", "-s x", used by fish). Accept either form.
			for _, f := range flagRegistry {
				if f.Long != "" && !strings.Contains(script, "--"+f.Long) && !strings.Contains(script, "-l "+f.Long) {
					t.Errorf("%s: registry flag --%s missing from generated script", shell, f.Long)
				}
				if f.Short != "" && !strings.Contains(script, "-"+f.Short) && !strings.Contains(script, "-s "+f.Short) {
					t.Errorf("%s: registry flag -%s missing from generated script", shell, f.Short)
				}
			}
		})
	}
}

// TestBashPerFlagValues guards APP-11: threshold-family flags in bash must
// each offer their own registry values, not a shared BashGroup value set.
// Before the fix, --fft-threshold and --strassen-threshold both offered the
// "threshold" group's 1024/2048/4096/8192/16384 instead of their own values.
func TestBashPerFlagValues(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	if err := GenerateBash(&buf, []string{"fast-doubling"}); err != nil {
		t.Fatalf("GenerateBash failed: %v", err)
	}
	script := buf.String()

	for _, f := range flagRegistry {
		if len(f.Values) == 0 || f.IsAlgo || f.Long == "completion" {
			continue
		}
		want := fmt.Sprintf(`--%s)
            COMPREPLY=( $(compgen -W "%s" -- "${cur}") )`, f.Long, strings.Join(f.Values, " "))
		if !strings.Contains(script, want) {
			t.Errorf("bash: --%s does not offer its own registry values %v; script:\n%s", f.Long, f.Values, script)
		}
	}
}
