package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestGenerateCompletion(t *testing.T) {
	t.Parallel()
	algorithms := []string{"fast", "matrix", "fft"}

	testCases := []struct {
		name      string
		shell     string
		expectErr bool
		checkFunc func(t *testing.T, output string)
	}{
		{
			name:      "Bash completion",
			shell:     "bash",
			expectErr: false,
			checkFunc: func(t *testing.T, output string) {
				if !strings.Contains(output, "Bash completion script") {
					t.Error("Bash script should contain 'Bash completion script'")
				}
				if !strings.Contains(output, "fast matrix fft all") {
					t.Error("Bash script should contain algorithm list")
				}
				if !strings.Contains(output, "_fibcalc_completions") {
					t.Error("Bash script should contain completion function")
				}
			},
		},
		{
			name:      "Zsh completion",
			shell:     "zsh",
			expectErr: false,
			checkFunc: func(t *testing.T, output string) {
				if !strings.Contains(output, "Zsh completion script") {
					t.Error("Zsh script should contain 'Zsh completion script'")
				}
				if !strings.Contains(output, "fast matrix fft all") {
					t.Error("Zsh script should contain algorithm list")
				}
				if !strings.Contains(output, "#compdef fibcalc") {
					t.Error("Zsh script should contain compdef directive")
				}
			},
		},
		{
			name:      "Fish completion",
			shell:     "fish",
			expectErr: false,
			checkFunc: func(t *testing.T, output string) {
				if !strings.Contains(output, "Fish completion script") {
					t.Error("Fish script should contain 'Fish completion script'")
				}
				if !strings.Contains(output, "fast matrix fft all") {
					t.Error("Fish script should contain algorithm list")
				}
				if !strings.Contains(output, "complete -c fibcalc") {
					t.Error("Fish script should contain complete commands")
				}
			},
		},
		{
			name:      "PowerShell completion",
			shell:     "powershell",
			expectErr: false,
			checkFunc: func(t *testing.T, output string) {
				if !strings.Contains(output, "PowerShell completion script") {
					t.Error("PowerShell script should contain 'PowerShell completion script'")
				}
				if !strings.Contains(output, "'fast', 'matrix', 'fft', 'all'") {
					t.Error("PowerShell script should contain algorithm list")
				}
				if !strings.Contains(output, "Register-ArgumentCompleter") {
					t.Error("PowerShell script should contain Register-ArgumentCompleter")
				}
			},
		},
		{
			name:      "PowerShell short alias",
			shell:     "ps",
			expectErr: false,
			checkFunc: func(t *testing.T, output string) {
				if !strings.Contains(output, "PowerShell completion script") {
					t.Error("PowerShell script should contain 'PowerShell completion script'")
				}
			},
		},
		{
			name:      "Unsupported shell",
			shell:     "unsupported",
			expectErr: true,
			checkFunc: func(t *testing.T, output string) {
				// No output expected for error case
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := GenerateCompletion(&buf, tc.shell, algorithms)

			if tc.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if !strings.Contains(err.Error(), "unsupported shell") {
					t.Errorf("Error message should mention 'unsupported shell', got: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				output := buf.String()
				if output == "" {
					t.Error("Output should not be empty")
				}
				if tc.checkFunc != nil {
					tc.checkFunc(t, output)
				}
			}
		})
	}
}

func TestGenerateCompletion_EmptyAlgorithms(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := GenerateCompletion(&buf, "bash", []string{})
	if err != nil {
		t.Errorf("Should not error with empty algorithms: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "algorithms=\" all\"") {
		t.Error("Should handle empty algorithm list")
	}
}

func TestGenerateCompletion_MultipleAlgorithms(t *testing.T) {
	t.Parallel()
	algorithms := []string{"fast", "matrix", "fft", "strassen", "optimized"}
	var buf bytes.Buffer
	err := GenerateCompletion(&buf, "bash", algorithms)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	output := buf.String()
	for _, algo := range algorithms {
		if !strings.Contains(output, algo) {
			t.Errorf("Output should contain algorithm '%s'", algo)
		}
	}
}
