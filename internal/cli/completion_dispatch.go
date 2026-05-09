package cli

import (
	"io"

	"github.com/agbru/fibcalc/internal/cli/completion"
)

// GenerateCompletion generates a shell completion script for the specified shell.
//
// This is a thin wrapper that delegates to the cli/completion sub-package, which
// holds the centralized flag registry and the per-shell generators (bash, zsh,
// fish, powershell).
//
// Parameters:
//   - out: The writer to output the completion script.
//   - shell: The shell type ("bash", "zsh", "fish", "powershell").
//   - algorithms: List of available algorithm names.
//
// Returns:
//   - error: An error if the shell is not supported.
func GenerateCompletion(out io.Writer, shell string, algorithms []string) error {
	return completion.Generate(out, shell, algorithms)
}
