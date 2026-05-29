package main

import (
	"bytes"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/agbruneau/FibGo/internal/app"
)

// --- Unit tests calling run() directly (instrumented for coverage) ---

func TestRun_Version(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
	}{
		{"long flag", []string{"fibcalc", "--version"}},
		{"short flag", []string{"fibcalc", "-V"}},
		{"dash flag", []string{"fibcalc", "-version"}},
		{"version with other args", []string{"fibcalc", "-n", "100", "--version"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			action := run(tt.args, &stdout, &stderr)

			if action != app.ActionVersionHandled {
				t.Errorf("Expected ActionVersionHandled, got %v", action)
			}
			if action.ShouldExit() {
				t.Error("ActionVersionHandled.ShouldExit() should be false")
			}
			out := stdout.String()
			if !strings.Contains(out, "fibcalc") {
				t.Errorf("Version output should contain 'fibcalc', got:\n%s", out)
			}
			if !strings.Contains(out, "Commit:") {
				t.Errorf("Version output should contain 'Commit:', got:\n%s", out)
			}
			if !strings.Contains(out, "Go version:") {
				t.Errorf("Version output should contain 'Go version:', got:\n%s", out)
			}
			if !strings.Contains(out, "OS/Arch:") {
				t.Errorf("Version output should contain 'OS/Arch:', got:\n%s", out)
			}
		})
	}
}

func TestRun_Help(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
	}{
		{"long flag", []string{"fibcalc", "--help"}},
		{"short flag", []string{"fibcalc", "-h"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			action := run(tt.args, &stdout, &stderr)

			if action.Code() != 0 {
				t.Errorf("Expected exit code 0, got %d", action.Code())
			}
			// Help text goes to stderr via flag package
			combined := stdout.String() + stderr.String()
			lower := strings.ToLower(combined)
			if !strings.Contains(lower, "usage") {
				t.Errorf("Help output should contain 'usage', got:\n%s", combined)
			}
		})
	}
}

func TestRun_InvalidFlag(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	action := run([]string{"fibcalc", "--invalid-flag-xyz"}, &stdout, &stderr)

	if action.Code() != 1 {
		t.Errorf("Expected exit code 1, got %d", action.Code())
	}
}

func TestRun_Calculation(t *testing.T) {
	t.Parallel()

	t.Run("small N with show-value", func(t *testing.T) {
		t.Parallel()
		var stdout, stderr bytes.Buffer
		code := run([]string{"fibcalc", "-n", "10", "-c"}, &stdout, &stderr).Code()

		if code != 0 {
			t.Errorf("Expected exit code 0, got %d. Output:\n%s", code, stdout.String())
		}
		if !strings.Contains(stdout.String(), "55") {
			t.Errorf("Output should contain F(10)=55, got:\n%s", stdout.String())
		}
	})

	t.Run("quiet mode", func(t *testing.T) {
		t.Parallel()
		var stdout, stderr bytes.Buffer
		code := run([]string{"fibcalc", "-n", "10", "--quiet"}, &stdout, &stderr).Code()

		if code != 0 {
			t.Errorf("Expected exit code 0, got %d. Output:\n%s", code, stdout.String())
		}
		if !strings.Contains(stdout.String(), "55") {
			t.Errorf("Quiet output should contain '55', got:\n%s", stdout.String())
		}
	})

	t.Run("specific algorithms", func(t *testing.T) {
		t.Parallel()
		for _, algo := range []string{"fast", "matrix", "fft"} {
			t.Run(algo, func(t *testing.T) {
				t.Parallel()
				var stdout, stderr bytes.Buffer
				code := run([]string{"fibcalc", "-n", "10", "--algo", algo, "--quiet"}, &stdout, &stderr).Code()

				if code != 0 {
					t.Errorf("Expected exit code 0 for algo %s, got %d", algo, code)
				}
				if !strings.Contains(stdout.String(), "55") {
					t.Errorf("Output for algo %s should contain '55', got:\n%s", algo, stdout.String())
				}
			})
		}
	})

	t.Run("all algorithms comparison", func(t *testing.T) {
		t.Parallel()
		var stdout, stderr bytes.Buffer
		code := run([]string{"fibcalc", "-n", "10", "--algo", "all"}, &stdout, &stderr).Code()

		if code != 0 {
			t.Errorf("Expected exit code 0, got %d. Output:\n%s", code, stdout.String())
		}
		lower := strings.ToLower(stdout.String())
		if !strings.Contains(lower, "comparison") || !strings.Contains(lower, "success") {
			t.Errorf("Comparison output should contain 'comparison' and 'success', got:\n%s", stdout.String())
		}
	})
}

func TestRun_Completion(t *testing.T) {
	t.Parallel()

	t.Run("bash completion", func(t *testing.T) {
		t.Parallel()
		var stdout, stderr bytes.Buffer
		code := run([]string{"fibcalc", "--completion", "bash"}, &stdout, &stderr).Code()

		if code != 0 {
			t.Errorf("Expected exit code 0, got %d", code)
		}
		lower := strings.ToLower(stdout.String())
		if !strings.Contains(lower, "bash") && !strings.Contains(lower, "complete") && !strings.Contains(lower, "fibcalc") {
			t.Errorf("Bash completion output seems invalid, got:\n%s", stdout.String())
		}
	})

	t.Run("invalid shell", func(t *testing.T) {
		t.Parallel()
		var stdout, stderr bytes.Buffer
		code := run([]string{"fibcalc", "--completion", "invalid-shell"}, &stdout, &stderr).Code()

		if code == 0 {
			t.Error("Expected non-zero exit code for invalid shell completion")
		}
	})
}

// --- Subprocess tests for os.Exit behavior in main() ---

// testBinaryPath builds the binary once and returns its path.
func testBinaryPath(t *testing.T) string {
	t.Helper()
	binName := "fibcalc_test_bin"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := t.TempDir() + "/" + binName

	cmd := exec.Command("go", "build", "-o", binPath, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build test binary: %v\n%s", err, out)
	}
	return binPath
}

// runBinary executes the built binary with the given args and returns
// combined stdout+stderr output and the exit code.
func runBinary(t *testing.T, binPath string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("Unexpected error running binary: %v", err)
		}
	}
	return string(out), exitCode
}

func TestMain_ExitCodes(t *testing.T) {
	t.Parallel()
	bin := testBinaryPath(t)

	t.Run("version exits 0", func(t *testing.T) {
		t.Parallel()
		_, code := runBinary(t, bin, "--version")
		if code != 0 {
			t.Errorf("Expected exit code 0, got %d", code)
		}
	})

	t.Run("help exits 0", func(t *testing.T) {
		t.Parallel()
		_, code := runBinary(t, bin, "--help")
		if code != 0 {
			t.Errorf("Expected exit code 0, got %d", code)
		}
	})

	t.Run("invalid flag exits 1", func(t *testing.T) {
		t.Parallel()
		_, code := runBinary(t, bin, "--invalid-flag-xyz")
		if code != 1 {
			t.Errorf("Expected exit code 1, got %d", code)
		}
	})

	t.Run("successful calculation exits 0", func(t *testing.T) {
		t.Parallel()
		_, code := runBinary(t, bin, "-n", "10", "--quiet")
		if code != 0 {
			t.Errorf("Expected exit code 0, got %d", code)
		}
	})
}
