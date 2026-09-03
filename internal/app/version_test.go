package app

import (
	"bytes"
	"runtime"
	"strings"
	"testing"
)

// TestHasVersionFlag tests the HasVersionFlag function.
func TestHasVersionFlag(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		args     []string
		expected bool
	}{
		{"Empty args", []string{}, false},
		{"No version flag", []string{"-n", "100"}, false},
		{"Long version flag", []string{"--version"}, true},
		{"Short version flag", []string{"-V"}, true},
		{"Version flag with dash", []string{"-version"}, true},
		{"Version flag in middle", []string{"-n", "100", "--version", "-algo", "fast"}, true},
		{"Version flag at end", []string{"-n", "100", "--version"}, true},
		{"Similar but not version", []string{"--verbose"}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := HasVersionFlag(tc.args)
			if result != tc.expected {
				t.Errorf("HasVersionFlag(%v) = %v, want %v", tc.args, result, tc.expected)
			}
		})
	}
}

// TestPrintVersion tests the PrintVersion function.
func TestPrintVersion(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	PrintVersion(&buf)

	output := buf.String()

	// Check that output contains expected components
	if !strings.Contains(output, "fibcalc") {
		t.Error("PrintVersion output should contain 'fibcalc'")
	}
	if !strings.Contains(output, Version) {
		t.Errorf("PrintVersion output should contain version '%s'", Version)
	}
	if !strings.Contains(output, "Commit:") {
		t.Error("PrintVersion output should contain 'Commit:'")
	}
	if !strings.Contains(output, "Built:") {
		t.Error("PrintVersion output should contain 'Built:'")
	}
	if !strings.Contains(output, "Go version:") {
		t.Error("PrintVersion output should contain 'Go version:'")
	}
	if !strings.Contains(output, runtime.Version()) {
		t.Errorf("PrintVersion output should contain Go version '%s'", runtime.Version())
	}
	if !strings.Contains(output, "OS/Arch:") {
		t.Error("PrintVersion output should contain 'OS/Arch:'")
	}
}

// TestHasVersionFlagStopsAtTerminator pins audit L-04: "--" ends option
// parsing by POSIX convention, so a literal "--version" after it is an
// operand, not a request for the version banner.
func TestHasVersionFlagStopsAtTerminator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"before terminator", []string{"-n", "10", "--version"}, true},
		{"after terminator", []string{"-n", "10", "--", "--version"}, false},
		{"terminator only", []string{"--"}, false},
		{"short form before terminator", []string{"-V", "--"}, true},
		{"single dash form", []string{"-version"}, true},
		{"absent", []string{"-n", "10"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := HasVersionFlag(tt.args); got != tt.want {
				t.Errorf("HasVersionFlag(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}
