package cli

import (
	"bytes"
	"io"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/agbru/fibcalc/internal/fibonacci"
	"github.com/briandowns/spinner"
)

// MockSpinner duplicated for advanced tests if needed, but since it is in ui_test.go which is in the same package (cli),
// it should be available when running `go test ./internal/cli`.
// However, if we run `go test ./internal/cli/ui_advanced_test.go ...`, we need to include ui_test.go or redefine it.
// Since we want `go test ./internal/cli` to work, we don't redefine it.

// TestDisplayProgress_LoopCoverage ensures the ticker and updates are processed
func TestDisplayProgress_LoopCoverage(t *testing.T) {
	// Setup mock spinner
	originalNewSpinner := newSpinner
	defer func() { newSpinner = originalNewSpinner }()

	mockS := &MockSpinner{}
	newSpinner = func(options ...spinner.Option) Spinner {
		return mockS
	}

	var wg sync.WaitGroup
	wg.Add(1)
	progressChan := make(chan fibonacci.ProgressUpdate)
	out := io.Discard

	go func() {
		// Send updates
		for i := 0; i < 5; i++ {
			progressChan <- fibonacci.ProgressUpdate{
				CalculatorIndex: 0,
				Value:           float64(i) * 0.2,
			}
			time.Sleep(50 * time.Millisecond) // enough to trigger ticker potentially
		}
		close(progressChan)
	}()

	DisplayProgress(&wg, progressChan, 1, out)
	wg.Wait()

	if !mockS.started {
		t.Error("Spinner should have started")
	}
}

// TestFormatExecutionDuration_MoreCases covers microsecond formatting
func TestFormatExecutionDuration_MoreCases(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{500 * time.Nanosecond, "0µs"},
		{1500 * time.Nanosecond, "1µs"},
		{999 * time.Microsecond, "999µs"},
		{1001 * time.Microsecond, "1ms"},
	}
	for _, c := range cases {
		got := FormatExecutionDuration(c.in)
		if got != c.want {
			t.Errorf("FormatExecutionDuration(%v) = %s, want %s", c.in, got, c.want)
		}
	}
}

// TestDisplayResult_VerySmallDuration covers "< 1µs" case in DisplayResult details
func TestDisplayResult_VerySmallDuration(t *testing.T) {
	var buf bytes.Buffer
	// Test the case where duration is exactly 0, which triggers the "< 1µs" display logic
	DisplayResult(big.NewInt(1), 1, 0, false, true, false, &buf)
	if !bytes.Contains(buf.Bytes(), []byte("< 1µs")) {
		t.Errorf("Expected output to contain '< 1µs', got %s", buf.String())
	}
}

// TestWriteResultToFile_Advanced calls WriteResultToFile with correct args
func TestWriteResultToFile_Advanced(t *testing.T) {
	tmpDir := t.TempDir()
	path := tmpDir + "/result.txt"

	res := big.NewInt(123456789)
	n := uint64(10)
	dur := time.Second
	algo := "test"
	cfg := OutputConfig{OutputFile: path}

	err := WriteResultToFile(res, n, dur, algo, cfg)
	if err != nil {
		t.Fatalf("WriteResultToFile failed: %v", err)
	}
}
