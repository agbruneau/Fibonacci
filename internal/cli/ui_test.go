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
	"example.com/fibcalc/internal/testutil"
	"github.com/briandowns/spinner"
)

// MockSpinner is a mock implementation of the Spinner interface for testing.
type MockSpinner struct {
	startCalled bool
	stopCalled  bool
	suffix      string
	mu          sync.Mutex
}

func (ms *MockSpinner) Start() {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.startCalled = true
}

func (ms *MockSpinner) Stop() {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.stopCalled = true
}

func (ms *MockSpinner) UpdateSuffix(suffix string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.suffix = suffix
}

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

// TestDisplayResult checks the formatting of the result output.
func TestDisplayResult(t *testing.T) {
	duration := 123 * time.Millisecond
	result, _ := new(big.Int).SetString("12586269025", 10) // F(50)

	t.Run("Output without details", func(t *testing.T) {
		var buf bytes.Buffer
		DisplayResult(result, 50, duration, false, false, &buf)
		output := testutil.StripAnsiCodes(buf.String())
		if !strings.Contains(output, "Result binary size: 34 bits.") {
			t.Errorf("The basic output is incorrect. Expected: 'Result binary size: 34 bits.', Got: %q", output)
		}
		if !strings.Contains(output, "(Tip: use the -d or --details option") {
			t.Errorf("The basic output should contain help for the details mode. Got: %q", output)
		}
	})

	t.Run("Detailed but non-verbose output (truncation)", func(t *testing.T) {
		var buf bytes.Buffer
		longNumStr := strings.Repeat("1", 101) // String longer than TruncationLimit
		longResult, _ := new(big.Int).SetString(longNumStr, 10)
		DisplayResult(longResult, 500, duration, false, true, &buf)
		output := testutil.StripAnsiCodes(buf.String())

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
		output := testutil.StripAnsiCodes(buf.String())

		if strings.Contains(output, "(truncated)") {
			t.Errorf("The verbose output should not be truncated. Got: %q", output)
		}
		expectedValue := "F(50) =\n12,586,269,025"
		if !strings.Contains(output, expectedValue) {
			t.Errorf("The value in the verbose output is incorrect.\nExpected (containing): %q\nGot: %s", expectedValue, output)
		}
	})
}

// TestDisplayProgress validates the behavior of the progress display.
func TestDisplayProgress(t *testing.T) {
	var buf bytes.Buffer
	var wg sync.WaitGroup
	progressChan := make(chan fibonacci.ProgressUpdate, 10)
	numCalculators := 2
	mock := &MockSpinner{}

	originalNewSpinner := newSpinner
	newSpinner = func(options ...spinner.Option) Spinner {
		return mock
	}
	defer func() { newSpinner = originalNewSpinner }()

	wg.Add(1)
	go DisplayProgress(&wg, progressChan, numCalculators, &buf)

	progressChan <- fibonacci.ProgressUpdate{CalculatorIndex: 0, Value: 0.25}
	progressChan <- fibonacci.ProgressUpdate{CalculatorIndex: 1, Value: 0.50}

	// Give the ticker time to update the suffix.
	time.Sleep(ProgressRefreshRate * 2)

	mock.mu.Lock()
	if !strings.Contains(mock.suffix, "Avg progress") {
		t.Errorf("The spinner suffix should show the 'Avg progress' label. Got: %q", mock.suffix)
	}
	if !strings.Contains(mock.suffix, "37.50%") {
		t.Errorf("The spinner suffix should show the correct average percentage. Got: %q", mock.suffix)
	}
	if !strings.Contains(mock.suffix, "ETA:") {
		t.Errorf("The spinner suffix should show ETA. Got: %q", mock.suffix)
	}
	mock.mu.Unlock()

	close(progressChan)
	wg.Wait()

	if !mock.startCalled {
		t.Error("Spinner.Start() was not called.")
	}
	if !mock.stopCalled {
		t.Error("Spinner.Stop() was not called.")
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if !strings.Contains(mock.suffix, "Calculation finished.") {
		t.Errorf("The final spinner suffix is incorrect. Got: %q", mock.suffix)
	}
}
