package completion

import (
	"fmt"
	"io"
	"strings"
)

// GenerateFish generates a Fish completion script.
func GenerateFish(out io.Writer, algorithms []string) error {
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

	parts = append(parts, fmt.Sprintf("-d '%s'", f.Help))

	switch {
	case f.IsFile:
		parts = append(parts, "-rF")
	case f.IsAlgo:
		parts = append(parts, fmt.Sprintf("-xa '%s all'", algoList))
	case len(f.Values) > 0:
		parts = append(parts, fmt.Sprintf("-xa '%s'", strings.Join(f.Values, " ")))
	case f.ValueName != "":
		// Takes a value but no suggestions (e.g., -n)
		parts = append(parts, "-x")
	}

	return strings.Join(parts, " ")
}
