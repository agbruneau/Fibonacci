package cli

import (
	"bytes"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agbruneau/FibGo/internal/progress"
)

// TestDisplayProgress_LoopCoverage ensures every update sent on the
// progress channel is consumed and that the final line is written when the
// channel closes.
//
// DisplayProgress draws on its own ticker loop, with no spinner dependency to
// mock (audit DEP-02), so the assertion is on the output rather than on a
// mock's Start/Stop bookkeeping.
//
// Synchronization is deterministic: the channel is unbuffered, so each
// send blocks until DisplayProgress receives it — no time.Sleep needed.
// Closing the channel drives DisplayProgress out of its select loop.
func TestDisplayProgress_LoopCoverage(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	progressChan := make(chan progress.ProgressUpdate)
	var out bytes.Buffer

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

	DisplayProgress(&wg, progressChan, 1, &out)
	wg.Wait()     // ensure DisplayProgress returned
	sendWG.Wait() // ensure producer returned

	got := out.String()
	if !strings.Contains(got, "Progress:") {
		t.Errorf("no final progress line written; got %q", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("final line is not terminated; got %q", got)
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
