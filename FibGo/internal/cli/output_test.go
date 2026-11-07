package cli

import (
	"bytes"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// errWriter fails every write; exercises the error path of writeResult.
type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

func TestWriteResult_FailingWriter(t *testing.T) {
	t.Parallel()
	err := writeResult(errWriter{}, big.NewInt(55), 10, 100*time.Millisecond, "fast")
	if err == nil {
		t.Fatal("writeResult must surface write errors, got nil")
	}
}

// TestWriteLastDigitsToFile pins the --last-digits output file (audit
// 2026-11-07): the leading zeros of the K digits survive, the header names the
// mode, and an empty path writes nothing.
func TestWriteLastDigitsToFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "nested", "last.txt")

	if err := WriteLastDigitsToFile("00875", 1000, 3*time.Millisecond, OutputConfig{OutputFile: path}); err != nil {
		t.Fatalf("WriteLastDigitsToFile: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("output file not written: %v", err)
	}
	for _, want := range []string{"# N: 1000\n", "# Digits: 5 (the last 5 of F(1000))\n", "\nF(1000) mod 10^5 =\n00875\n", "--last-digits"} {
		if !strings.Contains(string(content), want) {
			t.Errorf("output file lacks %q:\n%s", want, content)
		}
	}

	if err := WriteLastDigitsToFile("1", 1, 0, OutputConfig{}); err != nil {
		t.Errorf("an empty OutputFile must be a no-op, got %v", err)
	}
	if err := writeLastDigits(errWriter{}, "5", 5, 0); err == nil {
		t.Error("writeLastDigits must surface write errors, got nil")
	}
}

func TestWriteResultToFile(t *testing.T) {
	t.Parallel()
	// Create temporary directory
	tmpDir := t.TempDir()

	testCases := []struct {
		name        string
		outputFile  string
		expectError bool
		checkFunc   func(t *testing.T, filePath string)
	}{
		{
			name:        "Write decimal result to file",
			outputFile:  filepath.Join(tmpDir, "result.txt"),
			expectError: false,
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
			name:        "Empty output file (no write)",
			outputFile:  "",
			expectError: false,
			checkFunc:   nil, // No file should be created
		},
		{
			name:        "Create nested directory",
			outputFile:  filepath.Join(tmpDir, "nested", "dir", "result.txt"),
			expectError: false,
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
			config := OutputConfig{
				OutputFile: tc.outputFile,
			}

			err := WriteResultToFile(result, 10, 100*time.Millisecond, "fast", config)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tc.outputFile != "" && tc.checkFunc != nil {
					tc.checkFunc(t, tc.outputFile)
				}
			}
		})
	}
}

func TestDisplayQuietResult(t *testing.T) {
	t.Parallel()
	result := big.NewInt(55)

	t.Run("Decimal output", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		DisplayQuietResult(&buf, result)
		output := buf.String()
		if !strings.Contains(output, "55") {
			t.Errorf("Output should contain '55', got '%s'", output)
		}
	})
}
