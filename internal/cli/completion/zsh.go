package completion

import (
	"fmt"
	"io"
	"strings"
)

// zshHelpOverrides provides shell-specific help text overrides for zsh.
// Some flags have slightly different descriptions in zsh's _arguments format.
var zshHelpOverrides = map[string]string{
	"n":                  "Index n of Fibonacci number",
	"strassen-threshold": "Strassen threshold in bits",
}

// GenerateZsh generates a Zsh completion script.
func GenerateZsh(out io.Writer, algorithms []string) error {
	// Build _arguments entries from registry
	var args []string
	for _, f := range flagRegistry {
		args = append(args, zshArgEntry(f))
	}

	// Use escape-aware joiner: zsh array elements must each be single-quoted
	// to survive metacharacters in the algorithm name.
	algoList := formatAlgoListZsh(algorithms)

	script := fmt.Sprintf(`#compdef fibcalc

# Zsh completion script for fibcalc
# Add this to your ~/.zshrc or place in $fpath

_fibcalc() {
    local -a algorithms
    algorithms=(%s 'all')

    _arguments -s \
%s
}

_fibcalc "$@"
`, algoList, strings.Join(args, " \\\n"))

	_, err := fmt.Fprint(out, script)
	if err != nil {
		return fmt.Errorf("completion zsh generation failed: %w", err)
	}
	return nil
}

// zshHelp returns the help text for a flag in zsh, using an override if available.
func zshHelp(f FlagCompletion) string {
	key := flagKey(f)
	if override, ok := zshHelpOverrides[key]; ok {
		return override
	}
	return f.Help
}

// zshArgEntry formats a single FlagCompletion as a zsh _arguments entry.
func zshArgEntry(f FlagCompletion) string {
	help := zshHelp(f)

	// Build the value suffix
	valueSuffix := ""
	switch {
	case f.IsFile:
		valueSuffix = fmt.Sprintf(":%s:_files", f.ValueName)
	case f.IsAlgo:
		valueSuffix = fmt.Sprintf(":%s:($algorithms)", f.ValueName)
	case len(f.Values) > 0:
		valueSuffix = fmt.Sprintf(":%s:(%s)", f.ValueName, strings.Join(f.Values, " "))
	case f.ValueName != "":
		// Value-taking flag with no suggestions (e.g., -n)
		valueSuffix = fmt.Sprintf(":%s:", f.ValueName)
	}

	if f.Long != "" && f.Short != "" {
		// Has both short and long form
		return fmt.Sprintf("        '(-%s --%s)'{-%s,--%s}'[%s]%s'",
			f.Short, f.Long, f.Short, f.Long, help, valueSuffix)
	}
	if f.Long != "" {
		return fmt.Sprintf("        '--%s[%s]%s'", f.Long, help, valueSuffix)
	}
	// Short only
	return fmt.Sprintf("        '-%s[%s]%s'", f.Short, help, valueSuffix)
}
