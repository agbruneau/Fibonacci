// claudemd_guardians_test.go pins CLAUDE.md to the code it describes.
//
// CLAUDE.md anchors every sensitive invariant to a named guardian test.
// Nothing else verifies those anchors: after a rename or deletion the
// documentation keeps pointing at a ghost test, and a future maintainer —
// finding no such test — would conclude the invariant itself is gone and
// feel free to reintroduce the regression it guards against. This test
// fails as soon as the documentation drifts.
package internal_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// citedGuardian matches guardian test names cited in CLAUDE.md. A trailing
// `*` is a prefix wildcard (e.g. TestStateBump_*). The uppercase letter
// after "Test" keeps prose words (testable, tests, ...) out of the set.
var citedGuardian = regexp.MustCompile(`\bTest[A-Z][A-Za-z0-9_]*\*?`)

// testFuncDecl matches test function declarations in _test.go files.
var testFuncDecl = regexp.MustCompile(`(?m)^func (Test[A-Za-z0-9_]+)\s*\(`)

func TestClaudeMdGuardiansExist(t *testing.T) {
	t.Parallel()

	doc, err := os.ReadFile(filepath.Join("..", "CLAUDE.md"))
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}

	citedSet := map[string]struct{}{}
	for _, c := range citedGuardian.FindAllString(string(doc), -1) {
		citedSet[c] = struct{}{}
	}
	// CLAUDE.md currently cites ~24 guardians; a collapse of the extraction
	// regex must fail loudly rather than silently checking nothing.
	if len(citedSet) < 15 {
		t.Fatalf("extracted only %d guardian names from CLAUDE.md; extraction regex is probably broken", len(citedSet))
	}

	declared := declaredTestFuncs(t)

	for name := range citedSet {
		if prefix, isWildcard := strings.CutSuffix(name, "*"); isWildcard {
			found := false
			for d := range declared {
				if strings.HasPrefix(d, prefix) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("CLAUDE.md cites guardian pattern %q but no declared test function matches it", name)
			}
			continue
		}
		if _, ok := declared[name]; !ok {
			t.Errorf("CLAUDE.md cites guardian %q but no such test function is declared", name)
		}
	}
}

// declaredTestFuncs walks the repository for _test.go files and returns
// the set of declared Test function names.
func declaredTestFuncs(t *testing.T) map[string]struct{} {
	t.Helper()
	declared := map[string]struct{}{}
	err := filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range testFuncDecl.FindAllStringSubmatch(string(src), -1) {
			declared[m[1]] = struct{}{}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repository for _test.go files: %v", err)
	}
	if len(declared) == 0 {
		t.Fatal("no test functions found in tree; the walker is broken")
	}
	return declared
}
