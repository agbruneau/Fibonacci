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
	tmpDir := t.TempDir()
	result := big.NewInt(12345)
	n := uint64(10)
	duration := 100 * time.Millisecond
	algo := "test-algo"

	t.Run("Decimal", func(t *testing.T) {
		outputPath := filepath.Join(tmpDir, "result.txt")
		cfg := OutputConfig{OutputFile: outputPath}

		err := WriteResultToFile(result, n, duration, algo, cfg)
		if err != nil {
			t.Fatalf("WriteResultToFile failed: %v", err)
		}

		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatal(err)
		}
		s := string(content)
		if !strings.Contains(s, "12345") {
			t.Error("File should contain result")
		}
		if !strings.Contains(s, "F(10) =") {
			t.Error("File should contain N")
		}
	})

	t.Run("Hex", func(t *testing.T) {
		outputPath := filepath.Join(tmpDir, "result_hex.txt")
		cfg := OutputConfig{OutputFile: outputPath, HexOutput: true}

		err := WriteResultToFile(result, n, duration, algo, cfg)
		if err != nil {
			t.Fatalf("WriteResultToFile failed: %v", err)
		}

		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatal(err)
		}
		s := string(content)
		if !strings.Contains(s, "0x3039") { // 12345 in hex
			t.Error("File should contain hex result")
		}
	})

	t.Run("CreateDirectory", func(t *testing.T) {
		outputPath := filepath.Join(tmpDir, "subdir", "result.txt")
		cfg := OutputConfig{OutputFile: outputPath}

		err := WriteResultToFile(result, n, duration, algo, cfg)
		if err != nil {
			t.Fatalf("WriteResultToFile failed to create dir: %v", err)
		}
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Error("File was not created in subdir")
		}
	})

	t.Run("NoOutput", func(t *testing.T) {
		err := WriteResultToFile(result, n, duration, algo, OutputConfig{})
		if err != nil {
			t.Error("Should succeed doing nothing")
		}
	})
}

func TestFormatQuietResult(t *testing.T) {
	res := big.NewInt(255)

	s := FormatQuietResult(res, 10, time.Second, false)
	if s != "255" {
		t.Errorf("Expected '255', got '%s'", s)
	}

	s = FormatQuietResult(res, 10, time.Second, true)
	if s != "0xff" {
		t.Errorf("Expected '0xff', got '%s'", s)
	}
}

func TestDisplayQuietResult(t *testing.T) {
	var buf bytes.Buffer
	DisplayQuietResult(&buf, big.NewInt(10), 5, time.Second, false)
	if strings.TrimSpace(buf.String()) != "10" {
		t.Errorf("Expected '10', got '%s'", buf.String())
	}
}

func TestDisplayResultWithConfig(t *testing.T) {
	result := big.NewInt(12345)
	n := uint64(10)
	duration := time.Millisecond
	algo := "test"

	t.Run("Quiet", func(t *testing.T) {
		var buf bytes.Buffer
		cfg := OutputConfig{Quiet: true}
		err := DisplayResultWithConfig(&buf, result, n, duration, algo, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(buf.String()) != "12345" {
			t.Errorf("Expected quiet output '12345', got '%s'", buf.String())
		}
	})

	t.Run("HexVerbose", func(t *testing.T) {
		var buf bytes.Buffer
		cfg := OutputConfig{HexOutput: true, Verbose: true}
		err := DisplayResultWithConfig(&buf, result, n, duration, algo, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), "0x3039") {
			t.Error("Expected hex output")
		}
	})

	t.Run("HexTruncated", func(t *testing.T) {
		var buf bytes.Buffer
		// Make a big number
		bigRes := new(big.Int).Lsh(big.NewInt(1), 1000)
		cfg := OutputConfig{HexOutput: true, Verbose: false}
		err := DisplayResultWithConfig(&buf, bigRes, n, duration, algo, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), "...") {
			t.Error("Expected truncated hex output")
		}
	})

	t.Run("WithFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		outFile := filepath.Join(tmpDir, "out.txt")
		var buf bytes.Buffer
		cfg := OutputConfig{OutputFile: outFile}

		err := DisplayResultWithConfig(&buf, result, n, duration, algo, cfg)
		if err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(buf.String(), "Result saved to") {
			t.Error("Expected save confirmation")
		}
		if _, err := os.Stat(outFile); os.IsNotExist(err) {
			t.Error("File not created")
		}
	})
}
