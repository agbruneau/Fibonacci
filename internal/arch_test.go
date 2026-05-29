// Package internal_test enforces the Clean Architecture hierarchy
// revendiquée dans CLAUDE.md :
//
//	cmd → app → orchestration → fibonacci/bigfft → config/errors
//
// The test inspects the go list -deps graph at runtime and fails if a
// forbidden upward import is introduced. It guards the three upward
// arrows broken by the May-2026 hardening:
//   - internal/fibonacci/threshold → internal/config
//   - internal/errors → internal/format
//   - internal/tui → internal/fibonacci (production code only)
package internal_test

import (
	"os/exec"
	"strings"
	"testing"
)

// forbiddenImport describes an arrow that the architecture forbids.
// Direct (level-1) imports only — transitive imports through legitimate
// downward arrows (e.g. tui → orchestration → fibonacci) are allowed.
type forbiddenImport struct {
	importer string   // package whose direct imports we audit (full module path)
	forbid   []string // sub-paths that must not appear in importer's direct imports
}

// architectureRules encodes the inverted-tree invariant. Each rule says:
// `importer` MUST NOT directly import any of the `forbid` entries. The
// intent is to keep the dependency arrows pointing toward the kernel
// (cmd → app → orchestration → fibonacci/bigfft → config/errors) without
// being too rigid about transitive paths, which legitimately allow a TUI
// to reach domain types via the orchestration façade.
var architectureRules = []forbiddenImport{
	{
		// internal/fibonacci/threshold is a leaf inside fibonacci ; it
		// must not reach upward to internal/config (which would close
		// a cycle via config → fibonacci/memory).
		importer: "github.com/agbruneau/FibGo/internal/fibonacci/threshold",
		forbid: []string{
			"github.com/agbruneau/FibGo/internal/config",
		},
	},
	{
		// internal/errors is a leaf utility package and must not depend
		// on internal/format (presentation concern). A local
		// formatBytesLocal helper covers the byte-count need.
		importer: "github.com/agbruneau/FibGo/internal/errors",
		forbid: []string{
			"github.com/agbruneau/FibGo/internal/format",
		},
	},
	{
		// Production code in internal/tui must reach fibonacci only via
		// internal/orchestration ; the direct import is forbidden. Tests
		// inside internal/tui are tolerated (the production .Imports
		// query excludes _test.go files).
		importer: "github.com/agbruneau/FibGo/internal/tui",
		forbid: []string{
			"github.com/agbruneau/FibGo/internal/fibonacci",
		},
	},
}

func TestArchitectureLayering(t *testing.T) {
	t.Parallel()
	for _, rule := range architectureRules {
		rule := rule
		t.Run(rule.importer, func(t *testing.T) {
			t.Parallel()
			imports := directImports(t, rule.importer)
			for _, forbidden := range rule.forbid {
				if _, found := imports[forbidden]; found {
					t.Errorf(
						"architecture violation: %s directly imports %s; "+
							"this upward arrow is forbidden by the Clean Architecture "+
							"hierarchy documented in CLAUDE.md.",
						rule.importer, forbidden,
					)
				}
			}
		})
	}
}

// directImports returns the set of packages that pkg imports directly
// in its production code (test files are excluded). It shells out to
// `go list -f '{{.Imports}}'`.
func directImports(t *testing.T, pkg string) map[string]struct{} {
	t.Helper()
	cmd := exec.Command("go", "list", "-f", "{{range .Imports}}{{.}}\n{{end}}", pkg)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list %s failed: %v", pkg, err)
	}
	imports := map[string]struct{}{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		imports[line] = struct{}{}
	}
	return imports
}
