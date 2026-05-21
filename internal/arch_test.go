// Package internal_test enforces the Clean Architecture hierarchy
// revendiquée dans CLAUDE.md :
//
//	cmd → app → orchestration → fibonacci/bigfft → config/errors
//
// The test inspects the go list -deps graph at runtime and fails if a
// forbidden upward import is introduced. It is the runtime sentinel
// for Audit-PRD Epic E3-R4 (S3-T5).
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
		importer: "github.com/agbru/fibcalc/internal/fibonacci/threshold",
		forbid: []string{
			"github.com/agbru/fibcalc/internal/config",
		},
	},
	{
		// internal/errors is a leaf utility package and must not depend
		// on internal/format (presentation concern). See Audit-PRD E3-R2.
		importer: "github.com/agbru/fibcalc/internal/errors",
		forbid: []string{
			"github.com/agbru/fibcalc/internal/format",
		},
	},
	{
		// Production code in internal/tui must reach fibonacci only via
		// internal/orchestration ; the direct import is forbidden. Tests
		// inside internal/tui are tolerated (the production .Imports
		// query excludes _test.go files).
		importer: "github.com/agbru/fibcalc/internal/tui",
		forbid: []string{
			"github.com/agbru/fibcalc/internal/fibonacci",
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
							"hierarchy documented in CLAUDE.md and Audit - Global - FibGo.md §3.3.",
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
