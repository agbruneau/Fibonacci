package cli

import (
	"fmt"
	"io"
	"strings"
)

// FlagCompletion describes a CLI flag for shell completion generation.
// All shell completion functions generate from this registry, so adding
// a new flag only requires appending to flagRegistry.
type FlagCompletion struct {
	Long      string   // long flag name without "--" (e.g., "help")
	Short     string   // short flag without "-" (e.g., "h")
	Help      string   // description text
	Values    []string // suggested completion values (nil = boolean/no suggestions)
	ValueName string   // label for the value in zsh (e.g., "number", "duration")
	IsFile    bool     // true if the flag takes a file path
	IsAlgo    bool     // true if values come from algorithm list (dynamic)
	BashGroup string   // flags with same non-empty BashGroup share a bash case entry
}

// flagRegistry is the central list of all CLI flags for completion generation.
// The order matches the original completion output for each shell.
var flagRegistry = []FlagCompletion{
	{Long: "help", Short: "h", Help: "Show help message"},
	{Long: "version", Short: "V", Help: "Show version information"},
	{Short: "n", Help: "Fibonacci index to calculate", ValueName: "number"},
	{Short: "v", Help: "Display full result value"},
	{Long: "details", Short: "d", Help: "Show performance details"},
	{Long: "timeout", Help: "Maximum execution time", Values: []string{"1m", "5m", "10m", "30m", "1h"}, ValueName: "duration"},
	{Long: "algo", Help: "Algorithm to use", IsAlgo: true, ValueName: "algorithm"},
	{Long: "threshold", Help: "Parallelism threshold in bits", Values: []string{"1024", "2048", "4096", "8192", "16384"}, ValueName: "bits", BashGroup: "threshold"},
	{Long: "fft-threshold", Help: "FFT threshold in bits", Values: []string{"100000", "500000", "1000000"}, ValueName: "bits", BashGroup: "threshold"},
	{Long: "strassen-threshold", Help: "Strassen threshold", Values: []string{"1024", "2048", "3072", "4096"}, ValueName: "bits", BashGroup: "threshold"},
	{Long: "calibrate", Help: "Run calibration mode"},
	{Long: "auto-calibrate", Help: "Enable auto-calibration"},
	{Long: "calibration-profile", Help: "Calibration profile file", IsFile: true, ValueName: "file"},
	{Long: "output", Short: "o", Help: "Output file path", IsFile: true, ValueName: "file"},
	{Long: "quiet", Short: "q", Help: "Quiet mode for scripts"},
	{Long: "machine", Help: "Machine-readable output (no ANSI colors)"},
	{Long: "completion", Help: "Generate completion script", Values: []string{"bash", "zsh", "fish", "powershell"}, ValueName: "shell"},
}

// bashGroupValues defines the completion values used in bash for grouped flags.
// Flags sharing the same BashGroup use these values in the bash case statement.
var bashGroupValues = map[string][]string{
	"threshold": {"1024", "2048", "4096", "8192", "16384"},
}

// zshHelpOverrides provides shell-specific help text overrides for zsh.
// Some flags have slightly different descriptions in zsh's _arguments format.
var zshHelpOverrides = map[string]string{
	"n":                  "Index n of Fibonacci number",
	"strassen-threshold": "Strassen threshold in bits",
}

// GenerateCompletion generates a shell completion script for the specified shell.
//
// Parameters:
//   - out: The writer to output the completion script.
//   - shell: The shell type ("bash", "zsh", "fish", "powershell").
//   - algorithms: List of available algorithm names.
//
// Returns:
//   - error: An error if the shell is not supported.
func GenerateCompletion(out io.Writer, shell string, algorithms []string) error {
	switch shell {
	case "bash":
		return generateBashCompletion(out, algorithms)
	case "zsh":
		return generateZshCompletion(out, algorithms)
	case "fish":
		return generateFishCompletion(out, algorithms)
	case "powershell", "ps":
		return generatePowerShellCompletion(out, algorithms)
	default:
		return fmt.Errorf("unsupported shell: %s (accepted values: bash, zsh, fish, powershell)", shell)
	}
}

// formatAlgoList joins algorithm names with space separators.
func formatAlgoList(algorithms []string) string {
	return strings.Join(algorithms, " ")
}

// flagKey returns the identifier used for lookups: Long name if present, else Short.
func flagKey(f FlagCompletion) string {
	if f.Long != "" {
		return f.Long
	}
	return f.Short
}

// generateBashCompletion generates a Bash completion script.
func generateBashCompletion(out io.Writer, algorithms []string) error {
	// Build opts string from registry
	var opts []string
	for _, f := range flagRegistry {
		if f.Long != "" {
			opts = append(opts, "--"+f.Long)
		}
		if f.Short != "" {
			opts = append(opts, "-"+f.Short)
		}
	}

	// Build case entries from registry.
	// Order: algo, completion, file, timeout, threshold (matches original layout).
	type caseEntry struct {
		patterns []string
		body     string
	}
	bashCaseEntry := func(f FlagCompletion) caseEntry {
		return caseEntry{
			patterns: []string{"--" + f.Long},
			body:     fmt.Sprintf(`COMPREPLY=( $(compgen -W "%s" -- "${cur}") )`, strings.Join(f.Values, " ")),
		}
	}
	var orderedCases []caseEntry

	// 1. Algo flags
	for _, f := range flagRegistry {
		if f.IsAlgo {
			orderedCases = append(orderedCases, caseEntry{
				patterns: []string{"--" + f.Long},
				body:     `COMPREPLY=( $(compgen -W "${algorithms}" -- "${cur}") )`,
			})
		}
	}

	// 2. Completion flag (static values, comes before file/timeout)
	for _, f := range flagRegistry {
		if f.Long == "completion" && len(f.Values) > 0 {
			orderedCases = append(orderedCases, bashCaseEntry(f))
		}
	}

	// 3. File completion flags
	var filePatterns []string
	for _, f := range flagRegistry {
		if f.IsFile {
			if f.Long != "" {
				filePatterns = append(filePatterns, "--"+f.Long)
			}
			if f.Short != "" {
				filePatterns = append(filePatterns, "-"+f.Short)
			}
		}
	}
	if len(filePatterns) > 0 {
		orderedCases = append(orderedCases, caseEntry{
			patterns: filePatterns,
			body: `# File/directory completion
            COMPREPLY=( $(compgen -f -- "${cur}") )`,
		})
	}

	// 4. Remaining flags with static values (non-algo, non-file, non-grouped, non-completion)
	for _, f := range flagRegistry {
		if !f.IsAlgo && !f.IsFile && f.BashGroup == "" && f.Long != "completion" && len(f.Values) > 0 {
			orderedCases = append(orderedCases, bashCaseEntry(f))
		}
	}

	// 5. Grouped flags (threshold group)
	seenGroups := map[string]bool{}
	for _, f := range flagRegistry {
		if f.BashGroup != "" && !seenGroups[f.BashGroup] {
			seenGroups[f.BashGroup] = true
			var patterns []string
			for _, gf := range flagRegistry {
				if gf.BashGroup == f.BashGroup {
					patterns = append(patterns, "--"+gf.Long)
				}
			}
			vals := bashGroupValues[f.BashGroup]
			orderedCases = append(orderedCases, caseEntry{
				patterns: patterns,
				body:     fmt.Sprintf(`COMPREPLY=( $(compgen -W "%s" -- "${cur}") )`, strings.Join(vals, " ")),
			})
		}
	}

	// Format case entries
	var caseBody strings.Builder
	for _, c := range orderedCases {
		caseBody.WriteString("        ")
		caseBody.WriteString(strings.Join(c.patterns, "|"))
		caseBody.WriteString(")\n")
		caseBody.WriteString("            ")
		caseBody.WriteString(c.body)
		caseBody.WriteString("\n            return 0\n            ;;\n")
	}

	algoList := formatAlgoList(algorithms)

	script := fmt.Sprintf(`# Bash completion script for fibcalc
# Add this to your ~/.bashrc or ~/.bash_completion

_fibcalc_completions() {
    local cur prev opts algorithms
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    # Main options
    opts="%s"

    # Available algorithms
    algorithms="%s all"

    case "${prev}" in
%s    esac

    if [[ "${cur}" == -* ]]; then
        COMPREPLY=( $(compgen -W "${opts}" -- "${cur}") )
        return 0
    fi
}

complete -F _fibcalc_completions fibcalc
`, strings.Join(opts, " "), algoList, caseBody.String())

	_, err := fmt.Fprint(out, script)
	if err != nil {
		return fmt.Errorf("completion bash generation failed: %w", err)
	}
	return nil
}

// generateZshCompletion generates a Zsh completion script.
func generateZshCompletion(out io.Writer, algorithms []string) error {
	// Build _arguments entries from registry
	var args []string
	for _, f := range flagRegistry {
		args = append(args, zshArgEntry(f))
	}

	algoList := formatAlgoList(algorithms)

	script := fmt.Sprintf(`#compdef fibcalc

# Zsh completion script for fibcalc
# Add this to your ~/.zshrc or place in $fpath

_fibcalc() {
    local -a algorithms
    algorithms=(%s all)

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
	if f.IsFile {
		valueSuffix = fmt.Sprintf(":%s:_files", f.ValueName)
	} else if f.IsAlgo {
		valueSuffix = fmt.Sprintf(":%s:($algorithms)", f.ValueName)
	} else if len(f.Values) > 0 {
		valueSuffix = fmt.Sprintf(":%s:(%s)", f.ValueName, strings.Join(f.Values, " "))
	} else if f.ValueName != "" {
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

// generateFishCompletion generates a Fish completion script.
func generateFishCompletion(out io.Writer, algorithms []string) error {
	var lines []string

	lines = append(lines, "# Fish completion script for fibcalc")
	lines = append(lines, "# Add this to ~/.config/fish/completions/fibcalc.fish")
	lines = append(lines, "")
	lines = append(lines, "# Disable file completion by default")
	lines = append(lines, "complete -c fibcalc -f")
	lines = append(lines, "")

	// Group flags into sections for comments.
	// The sections mirror the original fish completion output.
	type section struct {
		comment string
		flags   []FlagCompletion
	}

	sections := []section{
		{comment: "# Help and version", flags: filterFlags("help", "version")},
		{comment: "# Main options", flags: filterFlags("n_short", "v_short", "details", "timeout", "algo", "threshold", "fft-threshold", "strassen-threshold")},
		{comment: "# Calibration", flags: filterFlags("calibrate", "auto-calibrate", "calibration-profile")},
		{comment: "# Output options", flags: filterFlags("output", "quiet", "machine")},
		{comment: "# Completion", flags: filterFlags("completion")},
	}

	algoList := formatAlgoList(algorithms)

	for _, sec := range sections {
		lines = append(lines, sec.comment)
		for _, f := range sec.flags {
			lines = append(lines, fishCompleteLine(f, algoList))
		}
		lines = append(lines, "")
	}

	script := strings.Join(lines, "\n")

	_, err := fmt.Fprint(out, script)
	if err != nil {
		return fmt.Errorf("completion fish generation failed: %w", err)
	}
	return nil
}

// filterFlags returns flags from the registry matching the given identifiers.
// An identifier is a Long name, or "X_short" to match a flag by Short name only.
func filterFlags(ids ...string) []FlagCompletion {
	var result []FlagCompletion
	for _, id := range ids {
		if strings.HasSuffix(id, "_short") {
			short := strings.TrimSuffix(id, "_short")
			for _, f := range flagRegistry {
				if f.Short == short && f.Long == "" {
					result = append(result, f)
					break
				}
			}
		} else {
			for _, f := range flagRegistry {
				if f.Long == id {
					result = append(result, f)
					break
				}
			}
		}
	}
	return result
}

// fishCompleteLine formats a single FlagCompletion as a fish complete command.
func fishCompleteLine(f FlagCompletion, algoList string) string {
	var parts []string
	parts = append(parts, "complete -c fibcalc")

	if f.Short != "" {
		parts = append(parts, fmt.Sprintf("-s %s", f.Short))
	}
	if f.Long != "" {
		parts = append(parts, fmt.Sprintf("-l %s", f.Long))
	}

	parts = append(parts, fmt.Sprintf("-d '%s'", f.Help))

	if f.IsFile {
		parts = append(parts, "-rF")
	} else if f.IsAlgo {
		parts = append(parts, fmt.Sprintf("-xa '%s all'", algoList))
	} else if len(f.Values) > 0 {
		parts = append(parts, fmt.Sprintf("-xa '%s'", strings.Join(f.Values, " ")))
	} else if f.ValueName != "" {
		// Takes a value but no suggestions (e.g., -n)
		parts = append(parts, "-x")
	}

	return strings.Join(parts, " ")
}

// generatePowerShellCompletion generates a PowerShell completion script.
func generatePowerShellCompletion(out io.Writer, algorithms []string) error {
	// Build $options entries from registry
	var optionEntries []string
	for _, f := range flagRegistry {
		if f.Short != "" {
			optionEntries = append(optionEntries, fmt.Sprintf(
				"        @{Name = '-%s'; Description = '%s' }", f.Short, f.Help))
		}
		if f.Long != "" {
			optionEntries = append(optionEntries, fmt.Sprintf(
				"        @{Name = '--%s'; Description = '%s' }", f.Long, f.Help))
		}
	}

	// Build context-aware switch entries.
	// Only algo and non-grouped flags with static values get context-aware completion.
	// Grouped flags (e.g., threshold variants) are omitted to match original behavior.
	// Order: algo, then non-algo value flags in reverse registry order (completion before timeout).
	var switchEntries []string

	psSwitchEntry := func(f FlagCompletion) string {
		var quotedVals []string
		for _, v := range f.Values {
			quotedVals = append(quotedVals, fmt.Sprintf("'%s'", v))
		}
		return fmt.Sprintf(`        '--%s' {
            @(%s) | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
            }
            return
        }`, f.Long, strings.Join(quotedVals, ", "))
	}

	// Algo flags first
	for _, f := range flagRegistry {
		if f.IsAlgo {
			switchEntries = append(switchEntries, fmt.Sprintf(`        '--%s' {
            $fibcalcAlgorithms | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
            }
            return
        }`, f.Long))
		}
	}

	// Non-algo value flags in reverse registry order (completion before timeout)
	var psValueFlags []FlagCompletion
	for _, f := range flagRegistry {
		if !f.IsAlgo && !f.IsFile && f.BashGroup == "" && len(f.Values) > 0 {
			psValueFlags = append(psValueFlags, f)
		}
	}
	for i := len(psValueFlags) - 1; i >= 0; i-- {
		switchEntries = append(switchEntries, psSwitchEntry(psValueFlags[i]))
	}

	// Format algorithm list for PowerShell
	psAlgoList := ""
	for i, algo := range algorithms {
		if i > 0 {
			psAlgoList += ", "
		}
		psAlgoList += fmt.Sprintf("'%s'", algo)
	}

	script := fmt.Sprintf(`# PowerShell completion script for fibcalc
# Add this to your $PROFILE

$fibcalcAlgorithms = @(%s, 'all')

Register-ArgumentCompleter -CommandName 'fibcalc' -Native -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)

    $options = @(
%s
    )

    $elements = $commandAst.CommandElements
    $lastElement = if ($elements.Count -gt 1) { $elements[-1].ToString() } else { '' }
    $prevElement = if ($elements.Count -gt 2) { $elements[-2].ToString() } else { '' }

    # Context-aware completions
    switch ($prevElement) {
%s
    }

    # Default: show options
    $options | Where-Object { $_.Name -like "$wordToComplete*" } | ForEach-Object {
        [System.Management.Automation.CompletionResult]::new($_.Name, $_.Name, 'ParameterName', $_.Description)
    }
}
`, psAlgoList, strings.Join(optionEntries, "\n"), strings.Join(switchEntries, "\n"))

	_, err := fmt.Fprint(out, script)
	return err
}
