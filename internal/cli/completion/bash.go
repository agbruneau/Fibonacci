package completion

import (
	"fmt"
	"io"
	"strings"
)

type bashCase struct {
	patterns []string
	body     string
}

func bashStaticValueCase(f FlagCompletion) bashCase {
	return bashCase{
		patterns: []string{"--" + f.Long},
		//nolint:gocritic // %s intentionnel: litteral bash -W, %q=quoting Go non-shell; echappement via escape.go
		body: fmt.Sprintf(`COMPREPLY=( $(compgen -W "%s" -- "${cur}") )`, formatAlgoListBash(f.Values)),
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
		if !f.IsAlgo && !f.IsFile && f.Long != "completion" && len(f.Values) > 0 {
			cases = append(cases, bashStaticValueCase(f))
		}
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

	// Order: algo, completion, file, static-with-values (matches the original layout).
	orderedCases := make([]bashCase, 0, len(flagRegistry))
	orderedCases = append(orderedCases, collectBashAlgoCases()...)
	orderedCases = append(orderedCases, collectBashCompletionCases()...)
	orderedCases = append(orderedCases, collectBashFileCases()...)
	orderedCases = append(orderedCases, collectBashStaticCases()...)

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
	// bash double-quoted string, so special chars must be neutralized.
	//
	// SEC-01 limitation: the escaping only protects the assignment line.
	// The generated script later runs `compgen -W "${algorithms}"`, and
	// compgen -W RE-EXPANDS each word (command substitution included) —
	// a name containing $(...) stored literally would execute there. Safe
	// today because names come from the static compiled registry ONLY;
	// never wire a dynamic source into this list without rethinking the
	// compgen layer (or whitelisting names to [A-Za-z0-9_-] at the source).
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
