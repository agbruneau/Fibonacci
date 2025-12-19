package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestCLI_E2E verifies the built binary functions correctly
func TestCLI_E2E(t *testing.T) {
	// Build the binary
	tmpDir := t.TempDir()
	binName := "fibcalc"
	if runtime.GOOS == "windows" {
		binName = "fibcalc.exe"
	}
	binPath := filepath.Join(tmpDir, binName)

	// Adjust build path assuming we are running from repo root
	// We need to find absolute path to cmd/fibcalc
	// go test is run from test/e2e usually if we do `go test ./test/e2e`
	// but user instructions say "Create test/e2e/cli_e2e_test.go"
	// We will assume "go test ./test/e2e/..." runs from module root in context of paths,
	// but `go build` needs correct package path.

	// We need to use the absolute path or relative from where go test is run.
	// When running `go test ./test/e2e/...` from root, CWD is root.
	// But `go build ./cmd/fibcalc` works from root.
	// Wait, the error `stat /app/test/e2e/cmd/fibcalc: directory not found` suggests
	// `go test` changes CWD to the test package directory.

	// Let's find the module root.
	// We are in test/e2e
	rootDir := "../.."

	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/fibcalc")
	cmd.Dir = rootDir // Execute build from repo root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build fibcalc: %v", err)
	}

	tests := []struct {
		name     string
		args     []string
		wantOut  string // substring match (case-insensitive)
		wantCode int
	}{
		{
			name:     "Basic Calculation",
			args:     []string{"-n", "10", "-c", "--no-color"}, // -c to show result, no-color for matching
			wantOut:  "F(10) = 55",
			wantCode: 0,
		},
		{
			name:     "JSON Output",
			args:     []string{"-n", "10", "--json"},
			wantOut:  `"result": "55"`, // JSON string "55"
			wantCode: 0,
		},
		{
			name:     "Help",
			args:     []string{"--help"},
			wantOut:  "usage", // Case-insensitive pattern
			wantCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binPath, tt.args...)
			output, err := cmd.CombinedOutput()

			if tt.wantCode == 0 && err != nil {
				t.Errorf("Command failed: %v\nOutput: %s", err, output)
			} else if tt.wantCode != 0 {
				if err == nil {
					t.Errorf("Expected command to fail with code %d", tt.wantCode)
				}
				// Exit code check requires casting err to ExitError
			}

			outStr := string(output)
			// Use case-insensitive matching for help output
			if !strings.Contains(strings.ToLower(outStr), strings.ToLower(tt.wantOut)) {
				t.Errorf("Output missing expected string.\nExpected: %q\nGot:\n%s", tt.wantOut, outStr)
			}
		})
	}
}
