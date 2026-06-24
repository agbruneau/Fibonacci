package main

import (
	"bytes"
	"io"
	"os"
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

	if action.Code() != 4 {
		t.Errorf("Expected exit code 4 (config), got %d", action.Code())
	}
}

func TestRun_ConfigErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		args       []string
		wantStderr string
	}{
		{"unknown flag", []string{"fibcalc", "--invalid-flag-xyz"}, "flag provided but not defined"},
		{"negative n", []string{"fibcalc", "-n=-5"}, "invalid value"},
		{"n overflows uint64", []string{"fibcalc", "-n", "18446744073709551616"}, "invalid value"},
		{"malformed timeout", []string{"fibcalc", "-timeout", "potato"}, "invalid value"},
		{"zero timeout", []string{"fibcalc", "-timeout", "0s"}, "timeout value must be strictly positive"},
		{"unknown algorithm", []string{"fibcalc", "--algo", "bogus"}, "unrecognized algorithm"},
		{"unknown gc-control mode", []string{"fibcalc", "--gc-control", "bogus"}, "unrecognized gc-control mode"},
		{"negative last-digits", []string{"fibcalc", "--last-digits=-1"}, "last-digits cannot be negative"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			action := run(tt.args, &stdout, &stderr)

			if action != app.ActionConfig {
				t.Errorf("Expected ActionConfig, got %v", action)
			}
			if action.Code() != 4 {
				t.Errorf("Expected exit code 4 (config), got %d", action.Code())
			}
			if !action.ShouldExit() {
				t.Error("Error action should require os.Exit")
			}
			if !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("stderr should contain %q, got:\n%s", tt.wantStderr, stderr.String())
			}
		})
	}
}

// TestRun_TimeoutExitAction verifies that a deadline expiring before the
// calculation completes surfaces as ActionTimeout (POSIX code 2) rather
// than a generic error.
func TestRun_TimeoutExitAction(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	action := run([]string{"fibcalc", "-n", "10000000", "--algo", "fast", "-timeout", "1ns", "--quiet"}, &stdout, &stderr)

	if action != app.ActionTimeout {
		t.Fatalf("Expected ActionTimeout, got %v (code %d). stdout:\n%s\nstderr:\n%s",
			action, action.Code(), stdout.String(), stderr.String())
	}
	if action.Code() != 2 {
		t.Errorf("Expected exit code 2, got %d", action.Code())
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

// TestMainFunction_VersionPath exercises main() itself on the --version
// path, the only path where main returns instead of calling os.Exit
// (ActionVersionHandled.ShouldExit() == false). If a regression made the
// version action exit-worthy again, main would call os.Exit(0) here and
// abort the test binary, which `go test` reports as a failure. The
// remaining os.Exit branch of main can only be observed via subprocess
// tests below.
//
// Deliberately NOT parallel: it swaps the process-global os.Args and
// os.Stdout, which must be restored before any other test observes them.
func TestMainFunction_VersionPath(t *testing.T) {
	origArgs := os.Args
	origStdout := os.Stdout
	t.Cleanup(func() {
		os.Args = origArgs
		os.Stdout = origStdout
	})

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Args = []string{"fibcalc", "--version"}
	os.Stdout = w

	main()

	os.Stdout = origStdout
	if cerr := w.Close(); cerr != nil {
		t.Fatalf("closing pipe writer: %v", cerr)
	}
	out, rerr := io.ReadAll(r)
	if rerr != nil {
		t.Fatalf("reading captured stdout: %v", rerr)
	}

	got := string(out)
	if !strings.Contains(got, "fibcalc") || !strings.Contains(got, "Go version:") {
		t.Errorf("main() with --version should print version info, got:\n%s", got)
	}
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

	t.Run("invalid flag exits 4", func(t *testing.T) {
		t.Parallel()
		_, code := runBinary(t, bin, "--invalid-flag-xyz")
		if code != 4 {
			t.Errorf("Expected exit code 4 (config), got %d", code)
		}
	})

	t.Run("successful calculation exits 0", func(t *testing.T) {
		t.Parallel()
		_, code := runBinary(t, bin, "-n", "10", "--quiet")
		if code != 0 {
			t.Errorf("Expected exit code 0, got %d", code)
		}
	})

	t.Run("negative n exits 4", func(t *testing.T) {
		t.Parallel()
		out, code := runBinary(t, bin, "-n=-5")
		if code != 4 {
			t.Errorf("Expected exit code 4 (config), got %d. Output:\n%s", code, out)
		}
		if !strings.Contains(out, "invalid value") {
			t.Errorf("Output should report the invalid value, got:\n%s", out)
		}
	})

	t.Run("zero timeout exits 4", func(t *testing.T) {
		t.Parallel()
		out, code := runBinary(t, bin, "-timeout", "0s")
		if code != 4 {
			t.Errorf("Expected exit code 4 (config), got %d. Output:\n%s", code, out)
		}
		if !strings.Contains(out, "timeout value must be strictly positive") {
			t.Errorf("Output should report the timeout configuration error, got:\n%s", out)
		}
	})

	t.Run("unknown algorithm exits 4", func(t *testing.T) {
		t.Parallel()
		out, code := runBinary(t, bin, "--algo", "bogus")
		if code != 4 {
			t.Errorf("Expected exit code 4 (config), got %d. Output:\n%s", code, out)
		}
		if !strings.Contains(out, "unrecognized algorithm") {
			t.Errorf("Output should report the unknown algorithm, got:\n%s", out)
		}
	})

	t.Run("runtime timeout exits 2", func(t *testing.T) {
		t.Parallel()
		out, code := runBinary(t, bin, "-n", "10000000", "--algo", "fast", "-timeout", "1ns", "--quiet")
		if code != 2 {
			t.Errorf("Expected exit code 2 (timeout), got %d. Output:\n%s", code, out)
		}
	})
}
