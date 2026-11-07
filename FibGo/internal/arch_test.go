// Package internal_test enforces the Clean Architecture hierarchy:
//
//	cmd → app → orchestration → fibonacci → bigfft
//
// internal/config is a sibling of orchestration, not a layer below
// fibonacci: internal/fibonacci does not import it. internal/bigfft is the
// kernel and imports no internal package at all.
//
// The rule is an allowlist (ADR-0014 D4): allowedImports names, for every
// package of the module, every internal package its production code may
// import directly. An import missing from it fails the test, and so does a
// package missing from it, so a new arrow is a reviewed edit to this file
// rather than a side effect of an import line. Entries that no longer exist
// fail too, which keeps the list an exact picture of the graph. It replaced
// six forbidden-import rules, which forbade only what they named.
package internal_test

import (
	"os/exec"
	"slices"
	"strings"
	"testing"
)

const modulePath = "github.com/agbruneau/FibGo/"

// allowedImports maps each package (path relative to the module) to the
// internal packages it may import directly. Test files are not covered: the
// production .Imports list excludes them. The -tags gmp and -tags purego
// builds add no internal edge.
var allowedImports = map[string][]string{
	"cmd/fibcalc": {"internal/app", "internal/apperrors"},
	// The golden generator is an independent oracle: math/big only.
	"cmd/generate-golden": nil,
	"internal":            nil,
	"internal/app": {
		"internal/apperrors", "internal/calibration", "internal/cli", "internal/cli/completion",
		"internal/config", "internal/fibonacci", "internal/fibonacci/memory",
		"internal/orchestration", "internal/tui", "internal/ui",
	},
	// Leaf; a local helper formats byte counts instead of internal/format.
	"internal/apperrors": nil,
	// The kernel.
	"internal/bigfft": nil,
	// Application layer: it decides what to say, internal/cli how it looks,
	// so neither internal/ui nor internal/format (audit ARC-01).
	"internal/calibration": {
		"internal/apperrors", "internal/bigfft", "internal/clock", "internal/config",
		"internal/fibonacci", "internal/progress",
	},
	// Presentation: reaches fibonacci only through orchestration (audit STR-04).
	"internal/cli": {
		"internal/apperrors", "internal/calibration", "internal/config", "internal/format",
		"internal/metrics", "internal/orchestration", "internal/progress", "internal/ui",
	},
	"internal/cli/completion": nil,
	"internal/clock":          nil,
	// Two documented lateral imports (config/doc.go); never fibonacci's root or
	// bigfft, which would pull the computation core into flag parsing.
	"internal/config": {"internal/apperrors", "internal/fibonacci/memory", "internal/ui"},
	"internal/fibonacci": {
		"internal/apperrors", "internal/bigfft", "internal/fibonacci/fibmath",
		"internal/fibonacci/memory", "internal/progress",
	},
	"internal/fibonacci/fibmath": nil,
	"internal/fibonacci/memory":  {"internal/fibonacci/fibmath"},
	"internal/format":            nil,
	"internal/metrics":           nil,
	// Owns progress aggregation; never the presentation-only internal/format
	// (audit APP-10).
	"internal/orchestration": {
		"internal/apperrors", "internal/clock", "internal/fibonacci", "internal/fibonacci/memory",
		"internal/progress",
	},
	"internal/progress": nil,
	"internal/testutil": nil,
	// Presentation: reaches fibonacci only through orchestration.
	"internal/tui": {
		"internal/apperrors", "internal/config", "internal/format", "internal/metrics",
		"internal/orchestration", "internal/progress", "internal/ui",
	},
	"internal/ui": nil,
	"test/e2e":    nil,
}

func TestArchitectureLayering(t *testing.T) {
	t.Parallel()
	graph := moduleImports(t)

	for pkg, imports := range graph {
		allowed, classified := allowedImports[pkg]
		if !classified {
			t.Errorf("package %s is not in allowedImports (internal/arch_test.go): "+
				"add it with the internal imports it may make", pkg)
			continue
		}
		for _, imp := range imports {
			if !slices.Contains(allowed, imp) {
				t.Errorf("architecture violation: %s imports %s, an arrow allowedImports "+
					"(internal/arch_test.go) does not list", pkg, imp)
			}
		}
		for _, imp := range allowed {
			if !slices.Contains(imports, imp) {
				t.Errorf("allowedImports lists %s → %s, which no longer exists: remove it", pkg, imp)
			}
		}
	}
	for pkg := range allowedImports {
		if _, ok := graph[pkg]; !ok {
			t.Errorf("allowedImports lists package %s, which no longer exists: remove it", pkg)
		}
	}
}

// moduleImports returns, for every package of the module, the module-internal
// packages its production code imports directly, both as module-relative paths.
// It shells out to `go list`.
func moduleImports(t *testing.T) map[string][]string {
	t.Helper()
	out, err := exec.Command("go", "list", "-f",
		"{{.ImportPath}}{{range .Imports}} {{.}}{{end}}", modulePath+"...").Output()
	if err != nil {
		t.Fatalf("go list %s... failed: %v", modulePath, err)
	}
	graph := map[string][]string{}
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		pkg := strings.TrimPrefix(fields[0], modulePath)
		graph[pkg] = []string{}
		for _, imp := range fields[1:] {
			if rel, ok := strings.CutPrefix(imp, modulePath); ok {
				graph[pkg] = append(graph[pkg], rel)
			}
		}
	}
	return graph
}
