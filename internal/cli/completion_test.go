package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestGenerateCompletion(t *testing.T) {
	tests := []struct {
		shell    string
		contains string
	}{
		{"bash", "_fibcalc_completions()"},
		{"zsh", "#compdef fibcalc"},
		{"fish", "complete -c fibcalc"},
		{"powershell", "Register-ArgumentCompleter"},
		{"unknown", "unsupported shell"}, // Should error or return nothing?
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			var buf bytes.Buffer
			// Dummy algorithms list
			algos := []string{"algo1", "algo2"}
			err := GenerateCompletion(&buf, tt.shell, algos)
			if tt.shell == "unknown" {
				if err == nil {
					t.Error("Expected error for unknown shell")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !strings.Contains(buf.String(), tt.contains) {
					t.Errorf("Output for %s should contain %q", tt.shell, tt.contains)
				}
				// Check if algorithms are included
				if !strings.Contains(buf.String(), "algo1") {
					t.Error("Output should contain algorithms")
				}
			}
		})
	}
}
