package internal_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Audit identifiers cited in comments must resolve (audit DOC-01).
//
// Roughly 350 of them appear across the production sources — FIB-02, OVR-10,
// M-01, A2-04, R4.2 — and until 2026-09-07 not one resolved to anything in the
// tree: the reports that defined them were deleted once their work was done, so
// a reader had to run `git log -S` to decode a three-word note.
//
// The fix is not to strip them. Most sit beside a justification already written
// out in full, and removing them would be a large diff that buys nothing. The
// fix is to keep them resolvable, which this test enforces: every prefix used in
// the code must be listed in docs/audits/INDEX.md, and that file says where each
// family is defined.
//
// A grep in the two gate scripts would have needed writing twice, in two shells;
// a test runs wherever `go test` does, including CI.

// auditIDPattern matches the identifier shapes this repository uses:
//
//	LETTERS-digits   FIB-02, OVR-10, A2-04, GATE-01, P1-04
//	Rn.n             R4.2, R3.9
//
// It deliberately does not match ADR-0012 (handled below) or ordinary hyphenated
// words, which are lower case.
var auditIDPattern = regexp.MustCompile(`\b([A-Z][A-Z0-9]{0,5})-\d{1,3}\b|\b(R)\d\.\d{1,2}\b`)

// notAnAuditID lists upper-case hyphenated tokens that this pattern matches but
// that are not audit identifiers: standards, instruction sets, and the algebraic
// notation the FFT code uses (K-1, P0-1, MSB-1).
var notAnAuditID = map[string]string{
	"UTF": "encoding",
	"ISO": "standard",
	"RFC": "standard",
	"SHA": "hash",
	"AES": "cipher",
	"CVE": "vulnerability database",
	"GO":  "language version",
	"FNV": "hash",
	"LRU": "eviction policy",
	"AVX": "x86 instruction set (AVX-512)",
	"MSB": "bit index arithmetic (MSB-1)",
	"K":   "polynomial degree arithmetic (x^(K-1))",
	"P0":  "polynomial coefficient (P0-1)",
}

func TestAuditIdentifiersResolve(t *testing.T) {
	t.Parallel()

	index, err := os.ReadFile(filepath.Join("..", "docs", "audits", "INDEX.md"))
	if err != nil {
		t.Fatalf("docs/audits/INDEX.md is what makes these identifiers resolvable: %v", err)
	}
	known := string(index)

	root := ".."
	unknown := map[string][]string{}

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip vendored, generated and agent-scratch trees.
			switch info.Name() {
			case ".git", ".claude", "build", "testdata", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, line := range strings.Split(string(src), "\n") {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "//") {
				continue
			}
			for _, m := range auditIDPattern.FindAllStringSubmatch(trimmed, -1) {
				prefix := m[1]
				if prefix == "" {
					prefix = m[2] // the Rn.n shape
				}
				if _, notAnID := notAnAuditID[prefix]; notAnID {
					continue
				}
				// INDEX.md lists families as `PREFIX-` or `Rn.`; a prefix is
				// resolvable when the index mentions it at all.
				if strings.Contains(known, "`"+prefix+"-`") ||
					strings.Contains(known, "`"+prefix+"1.`") ||
					strings.Contains(known, prefix+"-") {
					continue
				}
				rel, _ := filepath.Rel(root, path)
				unknown[prefix] = append(unknown[prefix], rel)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the source tree: %v", err)
	}

	if len(unknown) == 0 {
		return
	}

	prefixes := make([]string, 0, len(unknown))
	for p := range unknown {
		prefixes = append(prefixes, p)
	}
	sort.Strings(prefixes)

	var b strings.Builder
	b.WriteString("audit identifiers used in comments but not listed in docs/audits/INDEX.md:\n")
	for _, p := range prefixes {
		files := unknown[p]
		if len(files) > 3 {
			files = append(files[:3], "…")
		}
		b.WriteString("  " + p + "-  (" + strings.Join(files, ", ") + ")\n")
	}
	b.WriteString("\nAdd the family to the index, or state the reason in words instead of a code.")
	t.Error(b.String())
}
