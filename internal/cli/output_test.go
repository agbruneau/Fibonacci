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

func TestFormatQuietResult(t *testing.T) {
	t.Parallel()
	result := big.NewInt(55)

	t.Run("Decimal format", func(t *testing.T) {
		t.Parallel()
		output := FormatQuietResult(result)
		if output != "55" {
			t.Errorf("Expected '55', got '%s'", output)
		}
	})

	t.Run("Large number decimal", func(t *testing.T) {
		t.Parallel()
		large := new(big.Int)
		large.SetString("123456789012345678901234567890", 10)
		output := FormatQuietResult(large)
		if output != large.String() {
			t.Errorf("Expected full decimal string, got '%s'", output)
		}
	})
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
