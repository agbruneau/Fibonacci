package completion

import (
	"fmt"
	"io"
	"strings"
)

// GeneratePowerShell generates a PowerShell completion script.
func GeneratePowerShell(out io.Writer, algorithms []string) error {
	// Build $options entries from registry
	var optionEntries []string
	for _, f := range flagRegistry {
		if f.Short != "" {
			optionEntries = append(optionEntries, fmt.Sprintf(
				"        @{Name = '-%s'; Description = '%s' }", f.Short, escapePowerShellSingleQuoted(f.Help)))
		}
		if f.Long != "" {
			optionEntries = append(optionEntries, fmt.Sprintf(
				"        @{Name = '--%s'; Description = '%s' }", f.Long, escapePowerShellSingleQuoted(f.Help)))
		}
	}

	// Build context-aware switch entries.
	// Only algo and non-grouped flags with static values get context-aware completion.
	// Grouped flags (e.g., threshold variants) are omitted to match original behavior.
	// Order: algo, then non-algo value flags in reverse registry order (completion before timeout).
	var switchEntries []string

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

	// Format algorithm list for PowerShell with escape-aware single-quote
	// doubling so metacharacters in algorithm names cannot break the array.
	psAlgoList := formatAlgoListPowerShell(algorithms)

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

// psSwitchEntry builds a single PowerShell switch entry for a flag with static values.
func psSwitchEntry(f FlagCompletion) string {
	var quotedVals []string
	for _, v := range f.Values {
		quotedVals = append(quotedVals, fmt.Sprintf("'%s'", escapePowerShellSingleQuoted(v)))
	}
	return fmt.Sprintf(`        '--%s' {
            @(%s) | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
            }
            return
        }`, f.Long, strings.Join(quotedVals, ", "))
}
