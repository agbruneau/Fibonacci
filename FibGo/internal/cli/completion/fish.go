package completion

import (
	"fmt"
	"io"
	"strings"
)

// GenerateFish generates a Fish completion script.
func GenerateFish(out io.Writer, algorithms []string) error {
	lines := []string{
		"# Fish completion script for fibcalc",
		"# Add this to ~/.config/fish/completions/fibcalc.fish",
		"",
		"# Disable file completion by default",
		"complete -c fibcalc -f",
		"",
	}

	// Group flags into sections for comments.
	// The sections mirror the original fish completion output. Any registry
	// flag not claimed by an earlier section falls into "Other" so no flag
	// can be silently dropped by an outdated section list.
	type section struct {
		comment string
		flags   []FlagCompletion
	}

	sections := []section{
		{comment: "# Help and version", flags: filterFlags("help", "version")},
		{comment: "# Main options", flags: filterFlags("n_short", "verbose", "details", "timeout", "algo", "threshold", "fft-threshold", "strassen-threshold")},
		{comment: "# Calibration", flags: filterFlags("calibrate", "auto-calibrate", "calibration-profile")},
		{comment: "# Output options", flags: filterFlags("output", "quiet", "machine")},
		{comment: "# Completion", flags: filterFlags("completion")},
	}

	// Any registry flag not claimed by a section above still needs a
	// completion line, so it doesn't silently vanish when a new flag is
	// added to flagRegistry without updating the section lists here.
	claimed := map[string]bool{}
	for _, sec := range sections {
		for _, f := range sec.flags {
			claimed[flagKey(f)] = true
		}
	}
	var other []FlagCompletion
	for _, f := range flagRegistry {
		if !claimed[flagKey(f)] {
			other = append(other, f)
		}
	}
	if len(other) > 0 {
		sections = append(sections, section{comment: "# Other", flags: other})
	}

	// Use escape-aware joiner: fish single-quoted strings need \\ and \' escaping.
	algoList := formatAlgoListFish(algorithms)

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

	parts = append(parts, fmt.Sprintf("-d '%s'", escapeFishSingleQuoted(f.Help)))

	switch {
	case f.IsFile:
		parts = append(parts, "-rF")
	case f.IsAlgo:
		parts = append(parts, fmt.Sprintf("-xa '%s all'", algoList))
	case len(f.Values) > 0:
		parts = append(parts, fmt.Sprintf("-xa '%s'", formatAlgoListFish(f.Values)))
	case f.ValueName != "":
		// Takes a value but no suggestions (e.g., -n)
		parts = append(parts, "-x")
	}

	return strings.Join(parts, " ")
}
