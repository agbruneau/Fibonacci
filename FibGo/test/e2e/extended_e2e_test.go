package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCLI_GoldenValues verifies the correctness of results for moderate N.
func TestCLI_GoldenValues(t *testing.T) {
	skipShortE2E(t)
	binPath := buildBinary(t)

	tests := []struct {
		n    string
		want string
	}{
		{
			n:    "1000",
			want: "43466557686937456435688527675040625802564660517371780402481729089536555417949051890403879840079255169295922593080322634775209689623239873322471161642996440906533187938298969649928516003704476137795166849228875",
		},
		{
			n:    "100",
			want: "354224848179261915075",
		},
	}

	for _, tt := range tests {
		t.Run("N="+tt.n, func(t *testing.T) {
			t.Parallel()
			// Use quiet mode to get just the number
			cmd := exec.Command(binPath, "-n", tt.n, "--quiet", "-c")
			cmd.Env = append(os.Environ(), "NO_COLOR=1")
			output, err := cmd.Output()
			if err != nil {
				t.Fatalf("Command failed: %v", err)
			}

			got := strings.TrimSpace(string(output))
			if got != tt.want {
				t.Errorf("Result mismatch for N=%s.\nWant: %s\nGot:  %s", tt.n, tt.want, got)
			}
		})
	}
}

// TestCLI_ErrorCases verifies that invalid inputs produce appropriate error codes.
func TestCLI_ErrorCases(t *testing.T) {
	skipShortE2E(t)
	binPath := buildBinary(t)

	tests := []struct {
		name     string
		args     []string
		wantCode int
	}{
		{
			name:     "Negative N",
			args:     []string{"-n", "-1"},
			wantCode: 4, // ExitErrorConfig: flag parse / validation error
		},
		{
			name:     "Non-numeric N",
			args:     []string{"-n", "abc"},
			wantCode: 4, // ExitErrorConfig
		},
		{
			name:     "Invalid Output Path",
			args:     []string{"-n", "10", "--output", "\x00invalid/path.txt"},
			wantCode: 1, // ExitErrorGeneric: runtime file write error
		},
		{
			name:     "Extremely Large N",
			args:     []string{"-n", "100000000000000000000", "-c"}, // 1e20 > math.MaxUint64 (~1.84e19)
			wantCode: 4,                                             // ExitErrorConfig
		},
		// Audit 2026-11-07: these four used to exit 2 on a runtime panic
		// ("makeslice: len out of range" in the arena allocation) or to exit 0
		// after computing the default F(100000000) with the flags ignored.
		{
			name:     "Unaddressable N",
			args:     []string{"-n", "18446744073709551615", "-q"},
			wantCode: 4, // ExitErrorConfig: beyond memory.AddressableHeapBytes
		},
		{
			name:     "Unaddressable N under a huge memory limit",
			args:     []string{"-n", "10000000000000000", "-algo", "fast", "-memory-limit", "999999999G", "-q"},
			wantCode: 4,
		},
		{
			name:     "Positional argument",
			args:     []string{"100"},
			wantCode: 4,
		},
		{
			name:     "Double dash before flags",
			args:     []string{"--", "-n", "100", "-algo", "bogus"},
			wantCode: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := exec.Command(binPath, tt.args...)
			cmd.Env = append(os.Environ(), "NO_COLOR=1")
			err := cmd.Run()
			if err == nil {
				t.Fatal("Expected error, but command succeeded")
			}

			if exitErr, ok := err.(*exec.ExitError); ok {
				if exitErr.ExitCode() != tt.wantCode {
					t.Errorf("Expected exit code %d, got %d", tt.wantCode, exitErr.ExitCode())
				}
			}
		})
	}
}

// TestCLI_ModesCombination verifies combined use of flags.
func TestCLI_ModesCombination(t *testing.T) {
	skipShortE2E(t)
	binPath := buildBinary(t)
	t.Parallel()

	// Combination of --quiet, --last-digits and --algo
	cmd := exec.Command(binPath, "-n", "1000", "--quiet", "--last-digits", "10", "--algo", "matrix")
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	got := strings.TrimSpace(string(output))
	// F(1000) ends in ...166849228875, so its last 10 decimal digits are
	// 6849228875.
	want := "6849228875"

	if got != want {
		t.Errorf("Combined mode mismatch.\nWant: %s\nGot:  %s", want, got)
	}
}

// TestCLI_LastDigitsBeyondTheFullRange checks the two sides of the audit
// 2026-11-07 fixes in last-digits mode: an n refused for a full calculation is
// still computable there (O(K) memory), and --output is written.
func TestCLI_LastDigitsBeyondTheFullRange(t *testing.T) {
	skipShortE2E(t)
	binPath := buildBinary(t)
	t.Parallel()

	outFile := filepath.Join(t.TempDir(), "last.txt")
	cmd := exec.Command(binPath, "-n", "18446744073709551615", "--last-digits", "8", "-q", "-o", outFile)
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}
	digits := strings.TrimSpace(string(output))
	if len(digits) != 8 {
		t.Fatalf("want 8 digits on stdout, got %q", digits)
	}
	saved, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("--output not written in last-digits mode: %v", err)
	}
	if !strings.Contains(string(saved), "mod 10^8 =\n"+digits+"\n") {
		t.Errorf("output file should hold the digits printed on stdout (%s), got:\n%s", digits, saved)
	}
}

// TestCLI_Formatting verifies that the output has thousands separators.
func TestCLI_Formatting(t *testing.T) {
	skipShortE2E(t)
	binPath := buildBinary(t)
	t.Parallel()

	// Use non-quiet mode, which should format the number
	// Use N=70 (size between N=50 and N=100) to ensure formatting but avoid truncation
	cmd := exec.Command(binPath, "-n", "70", "-c")
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	outStr := string(output)
	// F(70) is 190,392,490,709,135 once grouped with thousands separators.
	if !strings.Contains(outStr, "190,392,490,709,135") {
		t.Errorf("Result not correctly formatted with commas.\nGot:\n%s", outStr)
	}
}
