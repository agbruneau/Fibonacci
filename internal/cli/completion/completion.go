package completion

import (
	"fmt"
	"io"
)

// Generate generates a shell completion script for the specified shell.
//
// Parameters:
//   - out: The writer to output the completion script.
//   - shell: The shell type ("bash", "zsh", "fish", "powershell").
//   - algorithms: List of available algorithm names.
//
// Returns:
//   - error: An error if the shell is not supported.
func Generate(out io.Writer, shell string, algorithms []string) error {
	switch shell {
	case "bash":
		return GenerateBash(out, algorithms)
	case "zsh":
		return GenerateZsh(out, algorithms)
	case "fish":
		return GenerateFish(out, algorithms)
	case "powershell", "ps":
		return GeneratePowerShell(out, algorithms)
	default:
		return fmt.Errorf("unsupported shell: %s (accepted values: bash, zsh, fish, powershell)", shell)
	}
}
