package completion

import "strings"

// Shell-escape helpers for completion-script generation.
//
// Security contract (Audit-PRD E5):
//
// All four shell generators (bash, zsh, fish, powershell) interpolate
// runtime-supplied algorithm names into the completion script they emit.
// Today those names come from the static internal fibonacci registry, so
// the risk is latent. But a future contributor wiring dynamic sources
// (env vars, config files) into the registry would inadvertently open a
// shell-injection vector if the interpolation lacked escaping.
//
// The functions below centralize per-shell escaping so each generator
// can apply the rule appropriate to its own quoting context. Each helper
// is paired with an adversarial test in escape_test.go covering the
// dangerous metacharacters: $(...), backticks, semicolons, spaces, quotes,
// backslashes, newlines.

// escapeBashDoubleQuoted escapes s so it is safe to splice inside a
// bash double-quoted string. Metacharacters expanded under "..." are
// \, $, ", `, and newline. All other characters pass through unchanged.
func escapeBashDoubleQuoted(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\\', '$', '"', '`':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\n':
			// Splitting algorithms by newline inside the bash array would
			// inject an extra command line. Replace by space — algorithm
			// names containing newlines are nonsensical anyway.
			b.WriteByte(' ')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// escapeFishSingleQuoted escapes s so it is safe to splice inside a
// fish single-quoted string. Per fish docs, only \ and ' are special
// inside '...'.
func escapeFishSingleQuoted(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `'`, `\'`, "\n", ` `)
	return r.Replace(s)
}

// escapeZshSingleQuoted escapes s so it is safe to splice inside a zsh
// single-quoted string. zsh closes the literal on the first '. The
// canonical workaround is to close-escape-reopen: ' → '\”.
func escapeZshSingleQuoted(s string) string {
	r := strings.NewReplacer(`'`, `'\''`, "\n", ` `)
	return r.Replace(s)
}

// escapePowerShellSingleQuoted escapes s for a PowerShell '...' literal.
// PowerShell single quotes are literal except that ' is escaped by
// doubling: ”.
func escapePowerShellSingleQuoted(s string) string {
	r := strings.NewReplacer(`'`, `''`, "\n", ` `)
	return r.Replace(s)
}

// formatAlgoListBash joins algorithm names for inclusion inside a
// bash double-quoted string, escaping each entry.
func formatAlgoListBash(algorithms []string) string {
	escaped := make([]string, len(algorithms))
	for i, a := range algorithms {
		escaped[i] = escapeBashDoubleQuoted(a)
	}
	return strings.Join(escaped, " ")
}

// formatAlgoListZsh joins algorithm names for the zsh array literal,
// individually single-quoting each entry.
func formatAlgoListZsh(algorithms []string) string {
	parts := make([]string, len(algorithms))
	for i, a := range algorithms {
		parts[i] = "'" + escapeZshSingleQuoted(a) + "'"
	}
	return strings.Join(parts, " ")
}

// formatZshValueList escapes and space-joins static completion values for a
// zsh ':name:(...)' value group, which is itself spliced inside the
// single-quoted _arguments entry (so each value follows the '...' escape rule).
func formatZshValueList(values []string) string {
	escaped := make([]string, len(values))
	for i, v := range values {
		escaped[i] = escapeZshSingleQuoted(v)
	}
	return strings.Join(escaped, " ")
}

// formatAlgoListFish joins algorithm names for the fish completion
// argument, escaping each entry for splicing inside '...'.
func formatAlgoListFish(algorithms []string) string {
	escaped := make([]string, len(algorithms))
	for i, a := range algorithms {
		escaped[i] = escapeFishSingleQuoted(a)
	}
	return strings.Join(escaped, " ")
}

// formatAlgoListPowerShell returns the comma-separated, single-quoted
// PowerShell array element list for $fibcalcAlgorithms.
func formatAlgoListPowerShell(algorithms []string) string {
	parts := make([]string, len(algorithms))
	for i, a := range algorithms {
		parts[i] = "'" + escapePowerShellSingleQuoted(a) + "'"
	}
	return strings.Join(parts, ", ")
}
