package completion

import (
	"fmt"
	"io"
	"strings"
)

// bashGroupValues defines the completion values used in bash for grouped flags.
// Flags sharing the same BashGroup use these values in the bash case statement.
var bashGroupValues = map[string][]string{
	"threshold": {"1024", "2048", "4096", "8192", "16384"},
}

type bashCase struct {
	patterns []string
	body     string
}

func bashStaticValueCase(f FlagCompletion) bashCase {
	return bashCase{
		patterns: []string{"--" + f.Long},
		body:     fmt.Sprintf(`COMPREPLY=( $(compgen -W "%s" -- "${cur}") )`, strings.Join(f.Values, " ")),
	}
}

func collectBashAlgoCases() []bashCase {
	var cases []bashCase
	for _, f := range flagRegistry {
		if f.IsAlgo {
			cases = append(cases, bashCase{
				patterns: []string{"--" + f.Long},
				body:     `COMPREPLY=( $(compgen -W "${algorithms}" -- "${cur}") )`,
			})
		}
	}
	return cases
}

func collectBashCompletionCases() []bashCase {
	var cases []bashCase
	for _, f := range flagRegistry {
		if f.Long == "completion" && len(f.Values) > 0 {
			cases = append(cases, bashStaticValueCase(f))
		}
	}
	return cases
}

func collectBashFileCases() []bashCase {
	var patterns []string
	for _, f := range flagRegistry {
		if !f.IsFile {
			continue
		}
		if f.Long != "" {
			patterns = append(patterns, "--"+f.Long)
		}
		if f.Short != "" {
			patterns = append(patterns, "-"+f.Short)
		}
	}
	if len(patterns) == 0 {
		return nil
	}
	return []bashCase{{
		patterns: patterns,
		body: `# File/directory completion
            COMPREPLY=( $(compgen -f -- "${cur}") )`,
	}}
}

func collectBashStaticCases() []bashCase {
	var cases []bashCase
	for _, f := range flagRegistry {
		if !f.IsAlgo && !f.IsFile && f.BashGroup == "" && f.Long != "completion" && len(f.Values) > 0 {
			cases = append(cases, bashStaticValueCase(f))
		}
	}
	return cases
}

func collectBashGroupedCases() []bashCase {
	var cases []bashCase
	seen := map[string]bool{}
	for _, f := range flagRegistry {
		if f.BashGroup == "" || seen[f.BashGroup] {
			continue
		}
		seen[f.BashGroup] = true
		var patterns []string
		for _, gf := range flagRegistry {
			if gf.BashGroup == f.BashGroup {
				patterns = append(patterns, "--"+gf.Long)
			}
		}
		vals := bashGroupValues[f.BashGroup]
		cases = append(cases, bashCase{
			patterns: patterns,
			body:     fmt.Sprintf(`COMPREPLY=( $(compgen -W "%s" -- "${cur}") )`, strings.Join(vals, " ")),
		})
	}
	return cases
}

// GenerateBash generates a Bash completion script.
func GenerateBash(out io.Writer, algorithms []string) error {
	var opts []string
	for _, f := range flagRegistry {
		if f.Long != "" {
			opts = append(opts, "--"+f.Long)
		}
		if f.Short != "" {
			opts = append(opts, "-"+f.Short)
		}
	}

	// Order: algo, completion, file, static-with-values, grouped (matches the original layout).
	var orderedCases []bashCase
	orderedCases = append(orderedCases, collectBashAlgoCases()...)
	orderedCases = append(orderedCases, collectBashCompletionCases()...)
	orderedCases = append(orderedCases, collectBashFileCases()...)
	orderedCases = append(orderedCases, collectBashStaticCases()...)
	orderedCases = append(orderedCases, collectBashGroupedCases()...)

	var caseBody strings.Builder
	for _, c := range orderedCases {
		caseBody.WriteString("        ")
		caseBody.WriteString(strings.Join(c.patterns, "|"))
		caseBody.WriteString(")\n")
		caseBody.WriteString("            ")
		caseBody.WriteString(c.body)
		caseBody.WriteString("\n            return 0\n            ;;\n")
	}

	// Use escape-aware joiner: algorithms are interpolated into a
	// bash double-quoted string, so special chars must be neutralised.
	algoList := formatAlgoListBash(algorithms)

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
