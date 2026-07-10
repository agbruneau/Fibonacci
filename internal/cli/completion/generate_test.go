package completion_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/agbruneau/FibGo/internal/cli/completion"
)

// adversarialAlgoNames is a black-box copy of the fixture in escape_test.go
// (package completion, white-box): these tests only call the exported
// Generate* functions, so they live outside the internal package, but they
// still need the same metacharacter-laden inputs to exercise the escaping
// contract end to end. See escape_test.go for the per-input rationale.
var adversarialAlgoNames = []string{
	`Fast Doubling`,
	`evil$(rm -rf /)`,
	"evil`rm -rf /`",
	`evil; echo pwn`,
	`evil"break"`,
	`evil'break`,
	`evil\nnewline`,
	`evil\backslash`,
	"evil\nactual_newline",
	`Schönhage-Strassen "Fancy"`,
	`Fibonacci & friends`,
}

// TestGenerateBash_AdversarialAlgo verifies that a malicious algorithm
// name embedded into the bash completion script cannot escape its
// "algorithms=..." double-quoted string.
func TestGenerateBash_AdversarialAlgo(t *testing.T) {
	t.Parallel()
	for _, evil := range adversarialAlgoNames {
		var buf bytes.Buffer
		if err := completion.GenerateBash(&buf, []string{"safe", evil}); err != nil {
			t.Fatalf("GenerateBash failed: %v", err)
		}
		script := buf.String()
		// The algorithms= line must remain a single line and properly
		// terminated. We isolate it and check that its outer double
		// quotes still balance.
		var algoLine string
		for _, line := range strings.Split(script, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "algorithms=") {
				algoLine = line
				break
			}
		}
		if algoLine == "" {
			t.Fatalf("algorithms= line missing from generated bash script: %q", script)
		}
		// Count unescaped double quotes — must be exactly 2.
		nUnescaped := 0
		for i := 0; i < len(algoLine); i++ {
			if algoLine[i] != '"' {
				continue
			}
			// Count preceding backslashes.
			bs := 0
			for j := i - 1; j >= 0 && algoLine[j] == '\\'; j-- {
				bs++
			}
			if bs%2 == 0 {
				nUnescaped++
			}
		}
		if nUnescaped != 2 {
			t.Errorf("for %q: expected 2 unescaped double quotes in algorithms= line, got %d: %q", evil, nUnescaped, algoLine)
		}
	}
}

// TestGenerateZsh_AdversarialAlgo verifies that a malicious algorithm
// name cannot escape its zsh single-quoted array element.
func TestGenerateZsh_AdversarialAlgo(t *testing.T) {
	t.Parallel()
	for _, evil := range adversarialAlgoNames {
		var buf bytes.Buffer
		if err := completion.GenerateZsh(&buf, []string{"safe", evil}); err != nil {
			t.Fatalf("GenerateZsh failed: %v", err)
		}
		script := buf.String()
		// Locate the algorithms=(...) line and confirm it terminates on
		// the same physical line.
		var algoLine string
		for _, line := range strings.Split(script, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "algorithms=(") {
				algoLine = trimmed
				break
			}
		}
		if algoLine == "" {
			t.Fatalf("algorithms=( ... ) line missing from zsh script for %q: %q", evil, script)
		}
		if !strings.HasSuffix(algoLine, ")") {
			t.Errorf("for %q: zsh algorithms array not terminated on its own line: %q", evil, algoLine)
		}
	}
}

// TestGenerateFish_AdversarialAlgo verifies fish completion isolates
// adversarial algorithm names inside its `-xa '...'` quoted argument.
// Fish escapes ' inside '...' with \', so a naïve count of ' characters
// will be misled by \'. We strip escaped quotes first then count.
func TestGenerateFish_AdversarialAlgo(t *testing.T) {
	t.Parallel()
	for _, evil := range adversarialAlgoNames {
		var buf bytes.Buffer
		if err := completion.GenerateFish(&buf, []string{"safe", evil}); err != nil {
			t.Fatalf("GenerateFish failed: %v", err)
		}
		for _, line := range strings.Split(buf.String(), "\n") {
			// Strip out escaped quotes (\\' and \\\\) so only literal
			// shell-level quotes remain.
			stripped := strings.ReplaceAll(line, `\\`, "")
			stripped = strings.ReplaceAll(stripped, `\'`, "")
			if strings.Count(stripped, "'")%2 != 0 {
				t.Errorf("for %q: fish line has unbalanced literal quotes after stripping escapes: %q", evil, line)
			}
		}
	}
}

// TestGeneratePowerShell_AdversarialAlgo verifies PowerShell completion
// doubles single quotes correctly when emitting the $fibcalcAlgorithms array.
func TestGeneratePowerShell_AdversarialAlgo(t *testing.T) {
	t.Parallel()
	for _, evil := range adversarialAlgoNames {
		var buf bytes.Buffer
		if err := completion.GeneratePowerShell(&buf, []string{"safe", evil}); err != nil {
			t.Fatalf("GeneratePowerShell failed: %v", err)
		}
		// Each PowerShell line should have an even number of single
		// quotes — by virtue of '' for embedded quotes.
		for _, line := range strings.Split(buf.String(), "\n") {
			if strings.Count(line, "'")%2 != 0 {
				t.Errorf("for %q: powershell line has odd single-quote count: %q", evil, line)
			}
		}
	}
}

// TestGenerate_UnsupportedShellRejected verifies the dispatcher rejects
// unknown shells rather than silently producing garbage. Sanity check.
func TestGenerate_UnsupportedShellRejected(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := completion.Generate(&buf, "nushell", []string{"safe"})
	if err == nil {
		t.Fatal("expected error for unsupported shell, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported shell") {
		t.Errorf("unexpected error message: %v", err)
	}
}
