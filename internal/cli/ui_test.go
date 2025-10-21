package cli

import (
	"bytes"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/fibcalc/internal/fibonacci"
)

// TestFormatNumberString validates the number formatting function.
func TestFormatNumberString(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"Empty string", "", ""},
		{"Single-digit number", "1", "1"},
		{"Three-digit number", "123", "123"},
		{"Four-digit number", "1234", "1,234"},
		{"Six-digit number", "123456", "123,456"},
		{"Seven-digit number", "1234567", "1,234,567"},
		{"Negative number", "-1234567", "-1,234,567"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatNumberString(tc.input); got != tc.expected {
				t.Errorf("formatNumberString(%q) = %q; want %q", tc.input, got, tc.expected)
			}
		})
	}
}

// TestProgressBar validates the progress bar generation.
func TestProgressBar(t *testing.T) {
	testCases := []struct {
		name     string
		progress float64
		length   int
		expected string
	}{
		{"Zero progress (0%)", 0.0, 10, "░░░░░░░░░░"},
		{"Partial progress (50%)", 0.5, 10, "█████░░░░░"},
		{"Full progress (100%)", 1.0, 10, "██████████"},
		{"25% progress on a 20-char bar", 0.25, 20, "█████░░░░░░░░░░░░░░░"},
		{"Edge case: progress > 100%", 1.1, 10, "██████████"},
		{"Edge case: progress < 0%", -0.1, 10, "░░░░░░░░░░"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := progressBar(tc.progress, tc.length); got != tc.expected {
				t.Errorf("progressBar(%.2f, %d) = %q; want %q", tc.progress, tc.length, got, tc.expected)
			}
		})
	}
}

// TestDisplayResult checks the formatting of the result output.
func TestDisplayResult(t *testing.T) {
	duration := 123 * time.Millisecond
	result, _ := new(big.Int).SetString("12586269025", 10) // F(50)

	t.Run("Output without details", func(t *testing.T) {
		var buf bytes.Buffer
		DisplayResult(result, 50, duration, false, false, &buf)
		output := buf.String()
		if !strings.Contains(output, "Binary Size of the Result: 34 bits.") {
			t.Errorf("The basic output is incorrect. Expected: 'Binary Size of the Result: 34 bits.', Got: %q", output)
		}
		if !strings.Contains(output, "(Use the -d or --details option") {
			t.Errorf("The basic output should contain help for the details mode. Got: %q", output)
		}
	})

	t.Run("Detailed but non-verbose output (truncation)", func(t *testing.T) {
		var buf bytes.Buffer
		longNumStr := strings.Repeat("1", 101) // String longer than TruncationLimit
		longResult, _ := new(big.Int).SetString(longNumStr, 10)
		DisplayResult(longResult, 500, duration, false, true, &buf)
		output := buf.String()

		if !strings.Contains(output, "(truncated)") {
			t.Errorf("The detailed non-verbose output should be truncated. Got: %q", output)
		}
		expectedTruncated := fmt.Sprintf("F(500) (truncated) = %s...%s", longNumStr[:DisplayEdges], longNumStr[len(longNumStr)-DisplayEdges:])
		if !strings.Contains(output, expectedTruncated) {
			t.Errorf("The truncated output format is incorrect.\nExpected (containing): %q\nGot: %s", expectedTruncated, output)
		}
	})

	t.Run("Detailed and verbose output (full)", func(t *testing.T) {
		var buf bytes.Buffer
		DisplayResult(result, 50, duration, true, true, &buf)
		output := buf.String()

		if strings.Contains(output, "(truncated)") {
			t.Errorf("The verbose output should not be truncated. Got: %q", output)
		}
		expectedValue := "F(50) =\n12,586,269,025"
		if !strings.Contains(output, expectedValue) {
			t.Errorf("The value in the verbose output is incorrect.\nExpected (containing): %q\nGot: %s", expectedValue, output)
		}
	})
}

// TestDisplayAggregateProgress validates the behavior of the progress consumer.
func TestDisplayAggregateProgress(t *testing.T) {
	var buf bytes.Buffer
	var wg sync.WaitGroup
	progressChan := make(chan fibonacci.ProgressUpdate, 10)
	numCalculators := 2

	wg.Add(1)
	go DisplayAggregateProgress(&wg, progressChan, numCalculators, &buf)

	progressChan <- fibonacci.ProgressUpdate{CalculatorIndex: 0, Value: 0.25}
	progressChan <- fibonacci.ProgressUpdate{CalculatorIndex: 1, Value: 0.50}

	time.Sleep(ProgressRefreshRate * 2)

	close(progressChan)
	wg.Wait()

	output := buf.String()
	expectedFinalLine := fmt.Sprintf("Average Progress :  37.50%% [%s]", progressBar(0.375, ProgressBarWidth))

	lines := strings.Split(strings.TrimSpace(output), "\n")
	lastLine := ""
	if len(lines) > 0 {
		lastLineWithControl := lines[len(lines)-1]
		if finalCR := strings.LastIndex(lastLineWithControl, "\r"); finalCR != -1 {
			lastLine = lastLineWithControl[finalCR+1:]
		} else {
			lastLine = lastLineWithControl
		}
		lastLine = strings.TrimPrefix(lastLine, "\033[K")
	}

	if lastLine != expectedFinalLine {
		t.Errorf("The final line of the progress bar is incorrect.\nExpected: %q\nGot : %q", expectedFinalLine, lastLine)
	}
}