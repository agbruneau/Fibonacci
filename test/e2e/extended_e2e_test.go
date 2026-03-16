package e2e

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestCLI_GoldenValues verifies the correctness of results for moderate N.
func TestCLI_GoldenValues(t *testing.T) {
	binPath := buildBinary(t)

	tests := []struct {
		n       string
		want    string
		isQuiet bool
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
	binPath := buildBinary(t)

	tests := []struct {
		name     string
		args     []string
		wantCode int
	}{
		{
			name:     "Negative N",
			args:     []string{"-n", "-1"},
			wantCode: 1, // Validation error
		},
		{
			name:     "Non-numeric N",
			args:     []string{"-n", "abc"},
			wantCode: 1,
		},
		{
			name:     "Invalid Output Path",
			args:     []string{"-n", "10", "--output", "\x00invalid/path.txt"},
			wantCode: 1,
		},
		{
			name:     "Extremely Large N",
			args:     []string{"-n", "10000000000000000000", "-c"}, // N > max uint64? No, max uint64 is ~1.8e19. 1e20 is more.
			wantCode: 1,
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
				if exitErr.ExitCode() == 0 {
					t.Errorf("Expected non-zero exit code, got 0")
				}
			}
		})
	}
}

// TestCLI_ModesCombination verifies combined use of flags.
func TestCLI_ModesCombination(t *testing.T) {
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
	want := "228875" // Last digits of F(1000) are ...228875, padded to 10? 
	// Wait, F(1000) result ends in 228875. Padded to 10 digits would be 166849228875? 
	// No, let's see. F(1000) = ...298969649928516003704476137795166849228875
	// Last 10 digits: 166849228875? No, 66849228875 is 11 digits.
	// 5166849228875 -> 13 digits.
	// ...166849228875
	
	// Let's check exactly.
	// Last 10: 66849228875? No.
	// 1 6 6 8 4 9 2 2 8 8 7 5
	// counts from right: 5(1) 7(2) 8(3) 8(4) 2(5) 2(6) 9(7) 4(8) 8(9) 6(10)
	// So 6849228875.
	want = "6849228875"
	
	if got != want {
		t.Errorf("Combined mode mismatch.\nWant: %s\nGot:  %s", want, got)
	}
}

// TestCLI_Formatting verifies that the output has thousands separators.
func TestCLI_Formatting(t *testing.T) {
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
	// F(70) = 190,392,490,709,135
	if !strings.Contains(outStr, "190,392,490,709,135") {
		t.Errorf("Result not correctly formatted with commas.\nGot:\n%s", outStr)
	}
}
