package cli

import (
	"bytes"
	"io"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/agbru/fibcalc/internal/progress"
	"github.com/briandowns/spinner"
)

// MockSpinner duplicated for advanced tests if needed, but since it is in ui_test.go which is in the same package (cli),
// it should be available when running `go test ./internal/cli`.
// However, if we run `go test ./internal/cli/ui_advanced_test.go ...`, we need to include ui_test.go or redefine it.
// Since we want `go test ./internal/cli` to work, we don't redefine it.

// TestDisplayProgress_LoopCoverage ensures every update sent on the
// progress channel is consumed and that the spinner lifecycle
// (Start / Stop) is honoured by DisplayProgress.
//
// Synchronisation is deterministic: the channel is unbuffered, so each
// send blocks until DisplayProgress receives it — no time.Sleep needed.
// Closing the channel drives DisplayProgress out of its select loop.
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
	progressChan := make(chan progress.ProgressUpdate)
	out := io.Discard

	// A second WaitGroup makes the producer goroutine joinable so the
	// test never returns while the sender is still running (no leaks).
	var sendWG sync.WaitGroup
	sendWG.Add(1)
	go func() {
		defer sendWG.Done()
		// Unbuffered channel: each send rendezvous with DisplayProgress's
		// receive, providing a happens-before edge — the 5th send is
		// observed before close.
		for i := 0; i < 5; i++ {
			progressChan <- progress.ProgressUpdate{
				CalculatorIndex: 0,
				Value:           float64(i) * 0.2,
			}
		}
		close(progressChan)
	}()

	DisplayProgress(&wg, progressChan, 1, out)
	wg.Wait()     // ensure DisplayProgress returned
	sendWG.Wait() // ensure producer returned

	if !mockS.started {
		t.Error("Spinner should have started")
	}
	if !mockS.stopped {
		t.Error("Spinner should have stopped after channel close")
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
