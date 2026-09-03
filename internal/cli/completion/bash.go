package completion

import (
	"fmt"
	"io"
	"slices"
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

// GenerateBash generates a Bash completion script.
func GenerateBash(out io.Writer, algorithms []string) error {
	// One pass over the registry sorts each flag into the case group it
	// belongs to; the groups are then emitted in the original script's order
	// (algo, completion, file, static-with-values), which the goldens pin.
	var opts, filePatterns []string
	var algoCases, completionCases, staticCases []bashCase
	for _, f := range flagRegistry {
		if f.Long != "" {
			opts = append(opts, "--"+f.Long)
		}
		if f.Short != "" {
			opts = append(opts, "-"+f.Short)
		}
		switch {
		case f.IsAlgo:
			algoCases = append(algoCases, bashCase{
				patterns: []string{"--" + f.Long},
				body:     `COMPREPLY=( $(compgen -W "${algorithms}" -- "${cur}") )`,
			})
		case f.IsFile:
			if f.Long != "" {
				filePatterns = append(filePatterns, "--"+f.Long)
			}
			if f.Short != "" {
				filePatterns = append(filePatterns, "-"+f.Short)
			}
		case f.Long == "completion" && len(f.Values) > 0:
			completionCases = append(completionCases, bashStaticValueCase(f))
		case len(f.Values) > 0:
			staticCases = append(staticCases, bashStaticValueCase(f))
		}
	}

	orderedCases := slices.Concat(algoCases, completionCases)
	if len(filePatterns) > 0 {
		orderedCases = append(orderedCases, bashCase{
			patterns: filePatterns,
			body: `# File/directory completion
            COMPREPLY=( $(compgen -f -- "${cur}") )`,
		})
	}
	orderedCases = append(orderedCases, staticCases...)

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
