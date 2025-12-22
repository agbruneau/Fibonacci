package cli

import (
	"bytes"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriteResultToFile(t *testing.T) {
	t.Parallel()
	// Create temporary directory
	tmpDir := t.TempDir()

	testCases := []struct {
		name        string
		outputFile  string
		hexOutput   bool
		expectError bool
		checkFunc   func(t *testing.T, filePath string)
	}{
		{
			name:        "Write decimal result to file",
			outputFile:  filepath.Join(tmpDir, "result.txt"),
			hexOutput:   false,
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
				if strings.Contains(contentStr, "0x") {
					t.Error("File should not contain hexadecimal prefix")
				}
			},
		},
		{
			name:        "Write hex result to file",
			outputFile:  filepath.Join(tmpDir, "result_hex.txt"),
			hexOutput:   true,
			expectError: false,
			checkFunc: func(t *testing.T, filePath string) {
				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}
				contentStr := string(content)
				if !strings.Contains(contentStr, "[hex]") {
					t.Error("File should contain '[hex]' marker")
				}
				if !strings.Contains(contentStr, "0x") {
					t.Error("File should contain hexadecimal prefix")
				}
			},
		},
		{
			name:        "Empty output file (no write)",
			outputFile:  "",
			hexOutput:   false,
			expectError: false,
			checkFunc:   nil, // No file should be created
		},
		{
			name:        "Create nested directory",
			outputFile:  filepath.Join(tmpDir, "nested", "dir", "result.txt"),
			hexOutput:   false,
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
				HexOutput:  tc.hexOutput,
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
		output := FormatQuietResult(result, 10, 100*time.Millisecond, false)
		if output != "55" {
			t.Errorf("Expected '55', got '%s'", output)
		}
	})

	t.Run("Hexadecimal format", func(t *testing.T) {
		t.Parallel()
		output := FormatQuietResult(result, 10, 100*time.Millisecond, true)
		if !strings.HasPrefix(output, "0x") {
			t.Errorf("Expected hex output to start with '0x', got '%s'", output)
		}
		if !strings.Contains(output, "37") { // 55 in hex is 0x37
			t.Errorf("Expected hex output to contain '37', got '%s'", output)
		}
	})

	t.Run("Large number decimal", func(t *testing.T) {
		t.Parallel()
		large := new(big.Int)
		large.SetString("123456789012345678901234567890", 10)
		output := FormatQuietResult(large, 100, 1*time.Second, false)
		if output != large.String() {
			t.Errorf("Expected full decimal string, got '%s'", output)
		}
	})

	t.Run("Large number hex", func(t *testing.T) {
		t.Parallel()
		large := new(big.Int)
		large.SetString("123456789012345678901234567890", 10)
		output := FormatQuietResult(large, 100, 1*time.Second, true)
		if !strings.HasPrefix(output, "0x") {
			t.Errorf("Expected hex output to start with '0x', got '%s'", output)
		}
	})
}

func TestDisplayQuietResult(t *testing.T) {
	t.Parallel()
	result := big.NewInt(55)

	t.Run("Decimal output", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		DisplayQuietResult(&buf, result, 10, 100*time.Millisecond, false)
		output := buf.String()
		if !strings.Contains(output, "55") {
			t.Errorf("Output should contain '55', got '%s'", output)
		}
		if strings.Contains(output, "0x") {
			t.Error("Decimal output should not contain '0x'")
		}
	})

	t.Run("Hex output", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		DisplayQuietResult(&buf, result, 10, 100*time.Millisecond, true)
		output := buf.String()
		if !strings.HasPrefix(strings.TrimSpace(output), "0x") {
			t.Errorf("Hex output should start with '0x', got '%s'", output)
		}
	})
}

func TestDisplayResultWithConfig(t *testing.T) {
	t.Parallel()
	result := big.NewInt(55)
	tmpDir := t.TempDir()

	t.Run("Quiet mode", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		config := OutputConfig{
			Quiet: true,
		}
		err := DisplayResultWithConfig(&buf, result, 10, 100*time.Millisecond, "fast", config)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, "55") {
			t.Errorf("Quiet output should contain result, got '%s'", output)
		}
	})

	t.Run("Quiet mode with hex", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		config := OutputConfig{
			Quiet:     true,
			HexOutput: true,
		}
		err := DisplayResultWithConfig(&buf, result, 10, 100*time.Millisecond, "fast", config)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		output := buf.String()
		if !strings.HasPrefix(strings.TrimSpace(output), "0x") {
			t.Errorf("Quiet hex output should start with '0x', got '%s'", output)
		}
	})

	t.Run("Normal mode with file output", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		outputFile := filepath.Join(tmpDir, "test_output.txt")
		config := OutputConfig{
			OutputFile: outputFile,
			Quiet:      false,
		}
		err := DisplayResultWithConfig(&buf, result, 10, 100*time.Millisecond, "fast", config)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		// Check that file was created
		if _, err := os.Stat(outputFile); err != nil {
			t.Errorf("Output file should exist: %v", err)
		}
		// Check that success message was printed
		output := buf.String()
		if !strings.Contains(output, "Result saved to") {
			t.Errorf("Should show file save message, got '%s'", output)
		}
	})

	t.Run("Quiet mode with file output", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		outputFile := filepath.Join(tmpDir, "quiet_output.txt")
		config := OutputConfig{
			OutputFile: outputFile,
			Quiet:      true,
		}
		err := DisplayResultWithConfig(&buf, result, 10, 100*time.Millisecond, "fast", config)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		// Check that file was created
		if _, err := os.Stat(outputFile); err != nil {
			t.Errorf("Output file should exist: %v", err)
		}
		// In quiet mode, file save message should not appear
		output := buf.String()
		if strings.Contains(output, "Result saved to") {
			t.Error("Quiet mode should not show file save message")
		}
	})

	t.Run("Hex output in normal mode", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		config := OutputConfig{
			HexOutput: true,
			Quiet:     false,
			Verbose:   false,
		}
		err := DisplayResultWithConfig(&buf, result, 10, 100*time.Millisecond, "fast", config)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, "Hexadecimal format") {
			t.Errorf("Should show hex format section, got '%s'", output)
		}
	})
}
