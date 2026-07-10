package completion

import (
	"strings"
	"testing"
)

// adversarialAlgoNames is the canonical set of metacharacter-laden inputs
// used to exercise the shell escape contracts (Audit-PRD E5-R1).
//
// A copy of this fixture also lives in generate_test.go (package
// completion_test): the escaper functions exercised here are unexported,
// so their direct unit tests must stay white-box, while the paired
// end-to-end Generate* tests only touch the exported API and were moved
// to a black-box file to shrink the surface coupled to internals.
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

// TestEscapeZshArgSpec_NeutralisesArgMetachars covers the zsh _arguments grammar
// layer: ':', '[' and ']' in help text or values must be backslash-escaped so
// they cannot terminate a description early or open a spurious field, while the
// single-quote shell layer still holds. Guards the latent injection class the
// package security contract pledges to keep closed.
func TestEscapeZshArgSpec_NeutralisesArgMetachars(t *testing.T) {
	t.Parallel()
	cases := []string{
		"a:b",                  // field separator
		"desc]break",           // would close a [description] early
		"open[bracket",         // would open a spurious group
		"all: at [once]",       // mixed
		"quote'inside",         // single-quote layer must still hold
		"line\nactual_newline", // newline must not survive
	}
	for _, in := range cases {
		out := escapeZshArgSpec(in)
		for i := 0; i < len(out); i++ {
			if c := out[i]; c == ':' || c == '[' || c == ']' {
				if i == 0 || out[i-1] != '\\' {
					t.Errorf("escapeZshArgSpec(%q): unescaped %q at %d in %q", in, string(c), i, out)
				}
			}
		}
		if strings.Contains(out, "\n") {
			t.Errorf("escapeZshArgSpec(%q) leaked newline in %q", in, out)
		}
		if got, want := strings.Count(out, `'\''`), strings.Count(in, "'"); got != want {
			t.Errorf("escapeZshArgSpec(%q): single-quote not close-reopened (%d/%d) in %q", in, got, want, out)
		}
	}
}

// TestZshArgEntry_EscapesArgMetachars confirms the escaper is wired into the
// real _arguments entry builder for both help text and value lists.
func TestZshArgEntry_EscapesArgMetachars(t *testing.T) {
	t.Parallel()
	entry := zshArgEntry(FlagCompletion{
		Long:      "evil",
		ValueName: "name",
		Help:      "desc] and : and [brackets]",
		Values:    []string{"a:b", "c]d"},
	})
	for _, want := range []string{`desc\]`, `\: and \[brackets\]`, `a\:b`, `c\]d`} {
		if !strings.Contains(entry, want) {
			t.Errorf("zshArgEntry did not escape payload %q; entry: %q", want, entry)
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
