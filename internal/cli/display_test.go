package cli_test

import (
	"bytes"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agbruneau/FibGo/internal/cli"
	"github.com/agbruneau/FibGo/internal/progress"
	"github.com/agbruneau/FibGo/internal/testutil"
)

// --- Golden tests: exact expected output, ANSI-stripped -------------------
//
// Comparisons run through testutil.StripAnsiCodes, so they are independent
// of whatever the process-global internal/ui theme happens to be -- no
// InitTheme call or cross-test serialization is needed here (see
// provider_test.go for the one test in this package that does need it).

func TestDisplayResult_Golden(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		result    *big.Int
		n         uint64
		duration  time.Duration
		verbose   bool
		details   bool
		showValue bool
		expected  string
	}{
		{
			name:      "Simple Result",
			result:    big.NewInt(55),
			n:         10,
			duration:  1 * time.Millisecond,
			showValue: true,
			expected:  "Result binary size: 6 bits.\n\n--- Calculated value ---\nF(10) = 55\n",
		},
		{
			name:     "Detailed Result",
			result:   big.NewInt(55),
			n:        10,
			duration: 0, // 0 duration -> < 1µs
			details:  true,
			expected: "Result binary size: 6 bits.\n\n--- Detailed result analysis ---\nCalculation time        : < 1µs\nNumber of digits      : 2\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			cli.DisplayResult(tt.result, tt.n, tt.duration, tt.verbose, tt.details, tt.showValue, &buf)
			got := testutil.StripAnsiCodes(buf.String())
			if got != tt.expected {
				t.Errorf("Golden mismatch for %s.\nWant:\n%q\nGot:\n%q", tt.name, tt.expected, got)
			}
		})
	}
}

func TestDisplayQuietResult_Golden(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	cli.DisplayQuietResult(&buf, big.NewInt(12345))
	if want := "12345\n"; buf.String() != want {
		t.Errorf("Golden mismatch quiet. Want %q, Got %q", want, buf.String())
	}
}

// --- Behavioral tests: substring checks ------------------------------------

func TestDisplayResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		result    *big.Int
		n         uint64
		duration  time.Duration
		verbose   bool
		details   bool
		showValue bool
		contains  []string
	}{
		{
			name:     "Details only",
			result:   big.NewInt(12345),
			n:        10,
			duration: time.Millisecond,
			details:  true,
			contains: []string{"Result binary size:", "Detailed result analysis", "Calculation time", "Number of digits"},
		},
		{
			name:      "ShowValue Output",
			result:    big.NewInt(12345),
			n:         10,
			duration:  time.Millisecond,
			showValue: true,
			contains:  []string{"Calculated value", "F(", ") =", "12,345"},
		},
		{
			name:      "Truncated Output",
			result:    new(big.Int).Exp(big.NewInt(10), big.NewInt(200), nil), // Very large number
			n:         100,
			duration:  time.Millisecond,
			showValue: true,
			contains:  []string{"(truncated)", "Tip: use"},
		},
		{
			name:      "Verbose Output",
			result:    new(big.Int).Exp(big.NewInt(10), big.NewInt(200), nil),
			n:         100,
			duration:  time.Millisecond,
			verbose:   true,
			showValue: true,
			contains:  []string{"F(", ") ="}, // Should not contain truncated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			cli.DisplayResult(tt.result, tt.n, tt.duration, tt.verbose, tt.details, tt.showValue, &buf)
			output := buf.String()
			for _, s := range tt.contains {
				if !strings.Contains(output, s) {
					t.Errorf("Expected output to contain %q, but got:\n%s", s, output)
				}
			}
		})
	}
}

// TestDisplayProgress_ZeroCalculators covers the degenerate case where
// NewProgressAggregator returns nil (numCalculators <= 0): DisplayProgress
// must drain the channel and return without ever constructing a spinner.
func TestDisplayProgress_ZeroCalculators(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup
	wg.Add(1)
	progressChan := make(chan progress.ProgressUpdate)
	close(progressChan)

	cli.DisplayProgress(&wg, progressChan, 0, io.Discard)
	wg.Wait()
}

func TestWriteResultToFile(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	testCases := []struct {
		name       string
		outputFile string
		checkFunc  func(t *testing.T, filePath string)
	}{
		{
			name:       "Write decimal result to file",
			outputFile: filepath.Join(tmpDir, "result.txt"),
			checkFunc: func(t *testing.T, filePath string) {
				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}
				contentStr := string(content)
				if !strings.Contains(contentStr, "F(10) =") {
					t.Error("File should contain 'F(10) ='")
				}
				if !strings.Contains(contentStr, "55") {
					t.Error("File should contain result '55'")
				}
			},
		},
		{
			name:       "Empty output file (no write)",
			outputFile: "", // No file should be created
		},
		{
			name:       "Create nested directory",
			outputFile: filepath.Join(tmpDir, "nested", "dir", "result.txt"),
			checkFunc: func(t *testing.T, filePath string) {
				if _, err := os.Stat(filePath); err != nil {
					t.Errorf("File should exist in nested directory: %v", err)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := big.NewInt(55)
			config := cli.OutputConfig{OutputFile: tc.outputFile}

			if err := cli.WriteResultToFile(result, 10, 100*time.Millisecond, "fast", config); err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tc.outputFile != "" && tc.checkFunc != nil {
				tc.checkFunc(t, tc.outputFile)
			}
		})
	}
}

func TestFormatQuietResult(t *testing.T) {
	t.Parallel()
	result := big.NewInt(55)

	t.Run("Decimal format", func(t *testing.T) {
		t.Parallel()
		if output := cli.FormatQuietResult(result); output != "55" {
			t.Errorf("Expected '55', got '%s'", output)
		}
	})

	t.Run("Large number decimal", func(t *testing.T) {
		t.Parallel()
		large := new(big.Int)
		large.SetString("123456789012345678901234567890", 10)
		if output := cli.FormatQuietResult(large); output != large.String() {
			t.Errorf("Expected full decimal string, got '%s'", output)
		}
	})
}

func TestDisplayQuietResult(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	cli.DisplayQuietResult(&buf, big.NewInt(55))
	if output := buf.String(); !strings.Contains(output, "55") {
		t.Errorf("Output should contain '55', got '%s'", output)
	}
}
