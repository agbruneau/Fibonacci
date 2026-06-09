package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// buildOnce ensures the binary is built only once across all tests.
var buildOnce sync.Once

// sharedBinPath holds the path to the compiled binary.
var sharedBinPath string

// sharedBuildErr holds any error from the build step.
var sharedBuildErr error

// buildBinary builds the fibcalc binary once and returns the path to it.
// It is safe to call from multiple tests concurrently.
func buildBinary(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		tmpDir := os.TempDir()
		binName := "fibcalc_e2e_test"
		if runtime.GOOS == "windows" {
			binName = "fibcalc_e2e_test.exe"
		}
		sharedBinPath = filepath.Join(tmpDir, binName)
		rootDir := "../.."
		cmd := exec.Command("go", "build", "-o", sharedBinPath, "./cmd/fibcalc")
		cmd.Dir = rootDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		sharedBuildErr = cmd.Run()
	})
	if sharedBuildErr != nil {
		t.Fatalf("Failed to build fibcalc: %v", sharedBuildErr)
	}
	return sharedBinPath
}

// skipShortE2E skips heavy end-to-end tests under `go test -short`.
// A-15: these tests compile the binary and spawn subprocesses; running them
// in -short (used by fast CI smoke and `make test-short`) is wasteful and
// makes the short suite sensitive to build/disk latency.
func skipShortE2E(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("e2e: heavy build+subprocess test skipped in -short mode (A-15)")
	}
}

// TestCLI_E2E verifies the built binary functions correctly
func TestCLI_E2E(t *testing.T) {
	skipShortE2E(t)
	binPath := buildBinary(t)

	tests := []struct {
		name     string
		args     []string
		wantOut  string // substring match (case-insensitive)
		wantCode int
	}{
		{
			name:     "Basic Calculation",
			args:     []string{"-n", "10", "-c"}, // -c to show result
			wantOut:  "F(10) = 55",
			wantCode: 0,
		},
		{
			name:     "Help",
			args:     []string{"--help"},
			wantOut:  "usage", // Case-insensitive pattern
			wantCode: 0,
		},
		{
			name:     "All Algorithms Comparison",
			args:     []string{"-n", "100", "--algo", "all", "-c"},
			wantOut:  "F(100)",
			wantCode: 0,
		},
		{
			name:     "Quiet Mode",
			args:     []string{"-n", "10", "--quiet", "-c"},
			wantOut:  "55",
			wantCode: 0,
		},
		{
			name:     "Very Short Timeout",
			args:     []string{"-n", "10000000", "--timeout", "1ms"},
			wantOut:  "", // may produce error output on stderr
			wantCode: 2,  // non-zero exit code expected (timeout error)
		},
		{
			name:     "Invalid N Zero",
			args:     []string{"-n", "0", "-c"},
			wantOut:  "F(0)",
			wantCode: 0,
		},
		{
			name:     "Large N",
			args:     []string{"-n", "1000", "-c"},
			wantOut:  "F(1000)",
			wantCode: 0,
		},
		{
			name:     "Version Flag",
			args:     []string{"--version"},
			wantOut:  "fibcalc",
			wantCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binPath, tt.args...)
			cmd.Env = append(os.Environ(), "NO_COLOR=1")
			output, err := cmd.CombinedOutput()

			outStr := string(output)

			if tt.wantCode == 0 {
				if err != nil {
					t.Errorf("Command failed unexpectedly: %v\nOutput: %s", err, outStr)
				}
			} else {
				// Expect a non-zero exit code
				if err == nil {
					t.Errorf("Expected non-zero exit code, but command succeeded.\nOutput: %s", outStr)
				} else if exitErr, ok := err.(*exec.ExitError); ok {
					// A-15: exit codes are a stable contract (internal/errors:
					// ExitErrorTimeout=2). Assert exactly instead of accepting
					// any non-zero, so a mapping regression is caught.
					if exitErr.ExitCode() != tt.wantCode {
						t.Errorf("Exit code mismatch: got %d, want %d\nOutput: %s",
							exitErr.ExitCode(), tt.wantCode, outStr)
					}
				}
				// err != nil but not ExitError is also acceptable (e.g., signal kill)
			}

			// Check output substring (skip check if wantOut is empty)
			if tt.wantOut != "" {
				if !strings.Contains(strings.ToLower(outStr), strings.ToLower(tt.wantOut)) {
					t.Errorf("Output missing expected string.\nExpected: %q\nGot:\n%s", tt.wantOut, outStr)
				}
			}
		})
	}
}

// TestCLI_InvalidFlags verifies that invalid flags produce non-zero exit codes.
func TestCLI_InvalidFlags(t *testing.T) {
	skipShortE2E(t)
	binPath := buildBinary(t)

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "Invalid Algorithm",
			args: []string{"-n", "10", "--algo", "nonexistent"},
		},
		{
			name: "Negative Timeout",
			args: []string{"-n", "10", "--timeout", "-1s"},
		},
		{
			name: "Unknown Flag",
			args: []string{"--not-a-real-flag"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binPath, tt.args...)
			cmd.Env = append(os.Environ(), "NO_COLOR=1")
			output, err := cmd.CombinedOutput()
			outStr := string(output)

			if err == nil {
				t.Errorf("Expected non-zero exit code for invalid flags, but command succeeded.\nOutput: %s", outStr)
				return
			}
			// Verify it's a non-zero exit code (config errors should be non-zero)
			if exitErr, ok := err.(*exec.ExitError); ok {
				if exitErr.ExitCode() == 0 {
					t.Errorf("Expected non-zero exit code, got 0.\nOutput: %s", outStr)
				}
			}
		})
	}
}

// TestCLI_OutputFile verifies that --output creates a file containing the result.
func TestCLI_OutputFile(t *testing.T) {
	skipShortE2E(t)
	binPath := buildBinary(t)

	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "result.txt")

	cmd := exec.Command(binPath, "-n", "50", "--output", outFile, "-c")
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, string(output))
	}

	// Verify the output file was created
	info, err := os.Stat(outFile)
	if err != nil {
		t.Fatalf("Output file was not created: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("Output file is empty")
	}

	// Read file contents and verify it contains the result
	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}
	contentStr := string(content)

	// F(50) = 12586269025
	if !strings.Contains(contentStr, "12586269025") {
		t.Errorf("Output file does not contain expected result.\nGot:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "F(50)") {
		t.Errorf("Output file does not contain F(50) header.\nGot:\n%s", contentStr)
	}
}

// TestCLI_TimeoutLargeN verifies that a very short timeout with a huge N triggers timeout behavior.
func TestCLI_TimeoutLargeN(t *testing.T) {
	skipShortE2E(t)
	binPath := buildBinary(t)

	// Use an extremely large N with a very short timeout to guarantee timeout
	cmd := exec.Command(binPath, "-n", "999999999", "--timeout", "1ms")
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	_, err := cmd.CombinedOutput()

	if err == nil {
		t.Error("Expected non-zero exit code for timeout, but command succeeded")
		return
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		// A-15: timeout maps to ExitErrorTimeout=2 (internal/errors). Assert it.
		if exitErr.ExitCode() != 2 {
			t.Errorf("Timeout exit code: got %d, want 2 (ExitErrorTimeout)", exitErr.ExitCode())
		}
	}
}

// TestCLI_Completion verifies that shell completion scripts are generated for all supported shells.
func TestCLI_Completion(t *testing.T) {
	skipShortE2E(t)
	binPath := buildBinary(t)

	shells := []struct {
		name    string
		shell   string
		wantOut string // substring to verify in the completion output
	}{
		{
			name:    "Bash",
			shell:   "bash",
			wantOut: "complete",
		},
		{
			name:    "Zsh",
			shell:   "zsh",
			wantOut: "compdef",
		},
		{
			name:    "Fish",
			shell:   "fish",
			wantOut: "complete -c fibcalc",
		},
		{
			name:    "PowerShell",
			shell:   "powershell",
			wantOut: "Register-ArgumentCompleter",
		},
	}

	for _, tt := range shells {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binPath, "--completion", tt.shell)
			cmd.Env = append(os.Environ(), "NO_COLOR=1")
			output, err := cmd.CombinedOutput()
			outStr := string(output)

			if err != nil {
				t.Errorf("Completion generation failed for %s: %v\nOutput: %s", tt.shell, err, outStr)
				return
			}

			if len(strings.TrimSpace(outStr)) == 0 {
				t.Errorf("Completion output for %s is empty", tt.shell)
				return
			}

			if !strings.Contains(outStr, tt.wantOut) {
				t.Errorf("Completion output for %s missing expected string %q.\nGot:\n%s", tt.shell, tt.wantOut, outStr)
			}
		})
	}
}

// TestCLI_LastDigits verifies the --last-digits flag computes partial results.
func TestCLI_LastDigits(t *testing.T) {
	skipShortE2E(t)
	binPath := buildBinary(t)

	tests := []struct {
		name     string
		args     []string
		wantOut  string
		wantCode int
	}{
		{
			name:     "Last 5 Digits of F(100)",
			args:     []string{"-n", "100", "--last-digits", "5", "--quiet"},
			wantOut:  "15075", // F(100) ends in ...15075
			wantCode: 0,
		},
		{
			name:     "Last 3 Digits of F(10)",
			args:     []string{"-n", "10", "--last-digits", "3", "--quiet"},
			wantOut:  "055", // F(10) = 55, padded to 3 digits = 055
			wantCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binPath, tt.args...)
			cmd.Env = append(os.Environ(), "NO_COLOR=1")
			output, err := cmd.CombinedOutput()
			outStr := strings.TrimSpace(string(output))

			if tt.wantCode == 0 {
				if err != nil {
					t.Errorf("Command failed unexpectedly: %v\nOutput: %s", err, outStr)
					return
				}
			}

			if tt.wantOut != "" {
				if !strings.Contains(outStr, tt.wantOut) {
					t.Errorf("Output missing expected string.\nExpected: %q\nGot: %q", tt.wantOut, outStr)
				}
			}
		})
	}
}

// TestCLI_LastDigitsBoundaries covers the --last-digits edges that the
// nominal cases above do not: K larger than the digit count (zero-padded),
// and negative K (rejected by config validation with a non-zero exit).
func TestCLI_LastDigitsBoundaries(t *testing.T) {
	skipShortE2E(t)
	binPath := buildBinary(t)

	t.Run("K Exceeds Digit Count Is Zero Padded", func(t *testing.T) {
		// F(10) = 55; asking for the last 10 digits must zero-pad to 0000000055.
		cmd := exec.Command(binPath, "-n", "10", "--last-digits", "10", "--quiet")
		cmd.Env = append(os.Environ(), "NO_COLOR=1")
		output, err := cmd.CombinedOutput()
		outStr := strings.TrimSpace(string(output))
		if err != nil {
			t.Fatalf("Command failed unexpectedly: %v\nOutput: %s", err, outStr)
		}
		if !strings.Contains(outStr, "0000000055") {
			t.Errorf("Expected zero-padded result 0000000055, got: %q", outStr)
		}
	})

	t.Run("Negative K Is Rejected", func(t *testing.T) {
		cmd := exec.Command(binPath, "-n", "10", "--last-digits", "-5")
		cmd.Env = append(os.Environ(), "NO_COLOR=1")
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Errorf("Expected non-zero exit code for negative --last-digits.\nOutput: %s", string(output))
		}
	})
}

// TestCLI_EnvOverrides verifies the documented configuration priority
// (CLI flags > FIBCALC_* environment variables > defaults) through the real
// binary, and that a malformed override is rejected loudly rather than
// silently falling back to the default.
func TestCLI_EnvOverrides(t *testing.T) {
	skipShortE2E(t)
	binPath := buildBinary(t)

	t.Run("Env N Applies When Flag Absent", func(t *testing.T) {
		cmd := exec.Command(binPath, "--quiet", "-c")
		cmd.Env = append(os.Environ(), "NO_COLOR=1", "FIBCALC_N=12")
		output, err := cmd.CombinedOutput()
		outStr := strings.TrimSpace(string(output))
		if err != nil {
			t.Fatalf("Command failed unexpectedly: %v\nOutput: %s", err, outStr)
		}
		if !strings.Contains(outStr, "144") { // F(12) = 144
			t.Errorf("Expected F(12)=144 from FIBCALC_N override, got: %q", outStr)
		}
	})

	t.Run("CLI Flag Wins Over Env", func(t *testing.T) {
		cmd := exec.Command(binPath, "-n", "10", "--quiet", "-c")
		cmd.Env = append(os.Environ(), "NO_COLOR=1", "FIBCALC_N=12")
		output, err := cmd.CombinedOutput()
		outStr := strings.TrimSpace(string(output))
		if err != nil {
			t.Fatalf("Command failed unexpectedly: %v\nOutput: %s", err, outStr)
		}
		if !strings.Contains(outStr, "55") { // F(10) = 55: CLI takes priority
			t.Errorf("Expected F(10)=55 (CLI > env), got: %q", outStr)
		}
		if strings.Contains(outStr, "144") {
			t.Errorf("Env override leaked through despite explicit -n flag, got: %q", outStr)
		}
	})

	t.Run("Malformed Env Value Is Rejected", func(t *testing.T) {
		cmd := exec.Command(binPath, "--quiet")
		cmd.Env = append(os.Environ(), "NO_COLOR=1", "FIBCALC_N=notanumber")
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Errorf("Expected non-zero exit code for malformed FIBCALC_N.\nOutput: %s", string(output))
		}
	})
}

// TestCLI_CompareMode verifies running specific algorithms with --algo flag.
func TestCLI_CompareMode(t *testing.T) {
	skipShortE2E(t)
	binPath := buildBinary(t)

	tests := []struct {
		name     string
		args     []string
		wantOut  string
		wantCode int
	}{
		{
			name:     "Fast Doubling Only",
			args:     []string{"-n", "50", "--algo", "fast", "-c"},
			wantOut:  "12,586,269,025",
			wantCode: 0,
		},
		{
			name:     "Matrix Only",
			args:     []string{"-n", "50", "--algo", "matrix", "-c"},
			wantOut:  "12,586,269,025",
			wantCode: 0,
		},
		{
			name:     "FFT Only",
			args:     []string{"-n", "50", "--algo", "fft", "-c"},
			wantOut:  "12,586,269,025",
			wantCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binPath, tt.args...)
			cmd.Env = append(os.Environ(), "NO_COLOR=1")
			output, err := cmd.CombinedOutput()
			outStr := string(output)

			if tt.wantCode == 0 {
				if err != nil {
					t.Errorf("Command failed unexpectedly: %v\nOutput: %s", err, outStr)
					return
				}
			}

			if tt.wantOut != "" {
				if !strings.Contains(outStr, tt.wantOut) {
					t.Errorf("Output missing expected string.\nExpected: %q\nGot:\n%s", tt.wantOut, outStr)
				}
			}
		})
	}
}

// TestCLI_VersionDetails verifies the version output contains expected fields.
func TestCLI_VersionDetails(t *testing.T) {
	skipShortE2E(t)
	binPath := buildBinary(t)

	cmd := exec.Command(binPath, "--version")
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	output, err := cmd.CombinedOutput()
	outStr := string(output)

	if err != nil {
		t.Fatalf("Version command failed: %v\nOutput: %s", err, outStr)
	}

	// Version output should contain these fields
	expected := []string{
		"fibcalc",
		"Go version:",
		"OS/Arch:",
	}
	for _, want := range expected {
		if !strings.Contains(outStr, want) {
			t.Errorf("Version output missing %q.\nGot:\n%s", want, outStr)
		}
	}
}
