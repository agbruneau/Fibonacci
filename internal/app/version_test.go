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

// TestGetVersionInfo tests the GetVersionInfo function.
func TestGetVersionInfo(t *testing.T) {
	t.Parallel()
	info := GetVersionInfo()

	if info.Version != Version {
		t.Errorf("GetVersionInfo().Version = %s, want %s", info.Version, Version)
	}
	if info.Commit != Commit {
		t.Errorf("GetVersionInfo().Commit = %s, want %s", info.Commit, Commit)
	}
	if info.BuildDate != BuildDate {
		t.Errorf("GetVersionInfo().BuildDate = %s, want %s", info.BuildDate, BuildDate)
	}
	if info.GoVersion != runtime.Version() {
		t.Errorf("GetVersionInfo().GoVersion = %s, want %s", info.GoVersion, runtime.Version())
	}
	if info.OS != runtime.GOOS {
		t.Errorf("GetVersionInfo().OS = %s, want %s", info.OS, runtime.GOOS)
	}
	if info.Arch != runtime.GOARCH {
		t.Errorf("GetVersionInfo().Arch = %s, want %s", info.Arch, runtime.GOARCH)
	}
}
