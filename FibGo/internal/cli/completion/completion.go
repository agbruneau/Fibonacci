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
	// "ps" is unreachable from the CLI — config.Validate accepts only bash,
	// zsh, fish and powershell — but it is a tested alias of this package's
	// own API, so it stays for callers that use Generate directly (audit L-02
	// candidate, rejected on verification).
	case "powershell", "ps":
		return GeneratePowerShell(out, algorithms)
	default:
		return fmt.Errorf("unsupported shell: %s (accepted values: bash, zsh, fish, powershell)", shell)
	}
}
