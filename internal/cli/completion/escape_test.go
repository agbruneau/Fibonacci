package completion

import (
	"bytes"
	"strings"
	"testing"
)

// adversarialAlgoNames is the canonical set of metacharacter-laden inputs
// used to exercise the shell escape contracts (Audit-PRD E5-R1).
var adversarialAlgoNames = []string{
	`Fast Doubling`,              // space — splits zsh/bash arrays if unquoted
	`evil$(rm -rf /)`,            // command substitution
	"evil`rm -rf /`",             // backticks command substitution
	`evil; echo pwn`,             // semicolon command chaining
	`evil"break"`,                // double quote breaks bash "..."
	`evil'break`,                 // single quote breaks fish/zsh/pwsh '...'
	`evil\nnewline`,              // literal "\n" sequence (no actual newline)
	`evil\backslash`,             // backslash
	"evil\nactual_newline",       // actual newline character
	`Schönhage-Strassen "Fancy"`, // unicode + double quote
	`Fibonacci & friends`,        // ampersand (bash background)
}

func TestEscapeBashDoubleQuoted_NeutralisesInjection(t *testing.T) {
	t.Parallel()
	dangerous := []string{`$`, `"`, "`", `\`}
	for _, in := range adversarialAlgoNames {
		out := escapeBashDoubleQuoted(in)
		// All dangerous chars must be backslash-escaped in the output.
		for _, ch := range dangerous {
			if strings.Contains(in, ch) && !strings.Contains(out, `\`+ch) {
				t.Errorf("escapeBashDoubleQuoted(%q): %q is not backslash-escaped in %q", in, ch, out)
			}
		}
		// Newline must not survive — would break the array assignment line.
		if strings.Contains(out, "\n") {
			t.Errorf("escapeBashDoubleQuoted(%q) leaked newline in %q", in, out)
		}
	}
}

func TestEscapeFishSingleQuoted_NeutralisesInjection(t *testing.T) {
	t.Parallel()
	for _, in := range adversarialAlgoNames {
		out := escapeFishSingleQuoted(in)
		// In fish '...', only \ and ' are special. Both must be escaped.
		// Check by ensuring no unescaped ' remains. An unescaped single
		// quote would close the literal. We verify by counting backslash
		// runs preceding each ' character.
		for i, r := range out {
			if r != '\'' {
				continue
			}
			// Count preceding backslashes.
			j := i - 1
			bs := 0
			for j >= 0 && out[j] == '\\' {
				bs++
				j--
			}
			if bs%2 == 0 {
				t.Errorf("escapeFishSingleQuoted(%q): unescaped ' at index %d in %q", in, i, out)
			}
		}
		if strings.Contains(out, "\n") {
			t.Errorf("escapeFishSingleQuoted(%q) leaked newline in %q", in, out)
		}
	}
}

func TestEscapeZshSingleQuoted_NeutralisesInjection(t *testing.T) {
	t.Parallel()
	for _, in := range adversarialAlgoNames {
		out := escapeZshSingleQuoted(in)
		// The escape rule is ' → '\''. So every ' from input expands to
		// exactly the 4-char sequence "'\\''" in the output. Verify by
		// counting occurrences instead of trying to parse the close-
		// escape-reopen pattern positionally.
		expectedExpansions := strings.Count(in, `'`)
		gotExpansions := strings.Count(out, `'\''`)
		if gotExpansions != expectedExpansions {
			t.Errorf("escapeZshSingleQuoted(%q): expected %d '\\'' expansions, got %d in %q",
				in, expectedExpansions, gotExpansions, out)
		}
		if strings.Contains(out, "\n") {
			t.Errorf("escapeZshSingleQuoted(%q) leaked newline in %q", in, out)
		}
	}
}

func TestEscapePowerShellSingleQuoted_NeutralisesInjection(t *testing.T) {
	t.Parallel()
	for _, in := range adversarialAlgoNames {
		out := escapePowerShellSingleQuoted(in)
		// Each ' from input must appear as '' in output.
		// Easy check: count single quotes — output should have exactly
		// 2*input_quotes.
		inQ := strings.Count(in, "'")
		outQ := strings.Count(out, "'")
		if outQ != 2*inQ {
			t.Errorf("escapePowerShellSingleQuoted(%q): expected %d singlequotes in output, got %d (%q)", in, 2*inQ, outQ, out)
		}
		if strings.Contains(out, "\n") {
			t.Errorf("escapePowerShellSingleQuoted(%q) leaked newline in %q", in, out)
		}
	}
}

// TestGenerateBash_AdversarialAlgo verifies that a malicious algorithm
// name embedded into the bash completion script cannot escape its
// "algorithms=..." double-quoted string.
func TestGenerateBash_AdversarialAlgo(t *testing.T) {
	t.Parallel()
	for _, evil := range adversarialAlgoNames {
		var buf bytes.Buffer
		if err := GenerateBash(&buf, []string{"safe", evil}); err != nil {
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
		if err := GenerateZsh(&buf, []string{"safe", evil}); err != nil {
			t.Fatalf("GenerateZsh failed: %v", err)
		}
		script := buf.String()
		// Locate the algorithms=(...) line and confirm it terminates on
		// the same physical line.
		var algoLine string
		for _, line := range strings.Split(script, "\n") {
			t := strings.TrimSpace(line)
			if strings.HasPrefix(t, "algorithms=(") {
				algoLine = t
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
		if err := GenerateFish(&buf, []string{"safe", evil}); err != nil {
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
		if err := GeneratePowerShell(&buf, []string{"safe", evil}); err != nil {
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

// TestFlagFormatters_NeutraliseHelpAndValues extends the shell-escape contract
// to flag Help text and static Values (audit F-014): both are interpolated into
// the same quoting contexts as algorithm names, so they must be escaped too.
// These exercise the per-flag formatters directly with adversarial input, so no
// global flagRegistry mutation is needed.
func TestFlagFormatters_NeutraliseHelpAndValues(t *testing.T) {
	t.Parallel()
	evil := FlagCompletion{
		Long:      "evil",
		ValueName: "x",
		Help:      "pwn' \"$(rm -rf /)`boom`",
		Values:    []string{"v'\"$(rm)", "a b"},
	}

	t.Run("bash_values_escaped", func(t *testing.T) {
		t.Parallel()
		body := bashStaticValueCase(evil).body
		if !strings.Contains(body, `\"`) || !strings.Contains(body, `\$`) {
			t.Errorf("bash value list not escaped via escape.go: %q", body)
		}
	})

	t.Run("zsh_help_and_values_escaped", func(t *testing.T) {
		t.Parallel()
		entry := zshArgEntry(evil)
		// Every literal ' from help+values must be emitted as '\''.
		wantQuotes := strings.Count(evil.Help, "'") + strings.Count(strings.Join(evil.Values, ""), "'")
		if got := strings.Count(entry, `'\''`); got < wantQuotes {
			t.Errorf("zsh entry under-escaped single quotes: got %d, want >= %d: %q", got, wantQuotes, entry)
		}
	})

	t.Run("fish_help_and_values_balanced", func(t *testing.T) {
		t.Parallel()
		line := fishCompleteLine(evil, "safe all")
		stripped := strings.ReplaceAll(line, `\\`, "")
		stripped = strings.ReplaceAll(stripped, `\'`, "")
		if strings.Count(stripped, "'")%2 != 0 {
			t.Errorf("fish line has unbalanced literal quotes after stripping escapes: %q", line)
		}
	})

	t.Run("powershell_values_balanced", func(t *testing.T) {
		t.Parallel()
		entry := psSwitchEntry(evil)
		if strings.Count(entry, "'")%2 != 0 {
			t.Errorf("powershell switch entry has odd single-quote count: %q", entry)
		}
	})
}

// TestGenerate_UnsupportedShellRejected verifies the dispatcher rejects
// unknown shells rather than silently producing garbage. Sanity check.
func TestGenerate_UnsupportedShellRejected(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := Generate(&buf, "nushell", []string{"safe"})
	if err == nil {
		t.Fatal("expected error for unsupported shell, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported shell") {
		t.Errorf("unexpected error message: %v", err)
	}
}
