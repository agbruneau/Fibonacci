package completion

import (
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
