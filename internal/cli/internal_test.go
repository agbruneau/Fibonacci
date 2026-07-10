// Tests in this file are white-box (package cli) because they exercise
// private symbols the black-box test files in this package cannot reach:
// padSpaces and displayMemoryStats (presenter.go), displayResultWithConfig
// (display.go), and the newSpinner factory var plus the realSpinner type
// (ui.go). See ui_suffix_race_test.go for the CONC-01 ordering guardian,
// kept in its own file since it documents a specific invariant.
package cli

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

	"github.com/agbruneau/FibGo/internal/progress"
	"github.com/briandowns/spinner"
)

func TestPadSpaces(t *testing.T) {
	t.Parallel()
	cases := []struct {
		length int
		want   string
	}{
		{-1, ""},
		{0, ""},
		{1, " "},
		{4, "    "},
	}
	for _, tc := range cases {
		if got := padSpaces(tc.length); got != tc.want {
			t.Errorf("padSpaces(%d) = %q, want %q", tc.length, got, tc.want)
		}
	}
}

func TestDisplayMemoryStats(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name          string
		heapAlloc     uint64
		totalAlloc    uint64
		numGC         uint32
		pauseTotalNs  uint64
		wantSubstring []string
	}{
		{
			name:          "gc disabled branch",
			heapAlloc:     1 << 20,
			totalAlloc:    1 << 22,
			numGC:         0,
			pauseTotalNs:  0,
			wantSubstring: []string{"GC pause total:  0ms (GC disabled)"},
		},
		{
			name:          "gc enabled branch",
			heapAlloc:     2 << 20,
			totalAlloc:    8 << 20,
			numGC:         3,
			pauseTotalNs:  5_000_000, // 5 ms
			wantSubstring: []string{"GC pause total:  5.00ms", "GC cycles:       3"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			displayMemoryStats(tc.heapAlloc, tc.totalAlloc, tc.numGC, tc.pauseTotalNs, &buf)
			for _, want := range tc.wantSubstring {
				if !strings.Contains(buf.String(), want) {
					t.Errorf("displayMemoryStats output missing %q; got:\n%s", want, buf.String())
				}
			}
		})
	}
}

func TestDisplayResultWithConfig(t *testing.T) {
	t.Parallel()
	result := big.NewInt(55)
	tmpDir := t.TempDir()

	t.Run("Quiet mode", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		config := OutputConfig{Quiet: true}
		if err := displayResultWithConfig(&buf, result, 10, 100*time.Millisecond, "fast", config); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), "55") {
			t.Errorf("Quiet output should contain result, got '%s'", buf.String())
		}
	})

	t.Run("Normal mode with file output", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		outputFile := filepath.Join(tmpDir, "test_output.txt")
		config := OutputConfig{OutputFile: outputFile, Quiet: false}
		if err := displayResultWithConfig(&buf, result, 10, 100*time.Millisecond, "fast", config); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if _, err := os.Stat(outputFile); err != nil {
			t.Errorf("Output file should exist: %v", err)
		}
		if !strings.Contains(buf.String(), "Result saved to") {
			t.Errorf("Should show file save message, got '%s'", buf.String())
		}
	})

	t.Run("Quiet mode with file output", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		outputFile := filepath.Join(tmpDir, "quiet_output.txt")
		config := OutputConfig{OutputFile: outputFile, Quiet: true}
		if err := displayResultWithConfig(&buf, result, 10, 100*time.Millisecond, "fast", config); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if _, err := os.Stat(outputFile); err != nil {
			t.Errorf("Output file should exist: %v", err)
		}
		if strings.Contains(buf.String(), "Result saved to") {
			t.Error("Quiet mode should not show file save message")
		}
	})
}

// MockSpinner is a Spinner double used to exercise DisplayProgress without
// a real terminal.
type MockSpinner struct {
	started bool
	stopped bool
	suffix  string
}

func (m *MockSpinner) Start()                { m.started = true }
func (m *MockSpinner) Stop()                 { m.stopped = true }
func (m *MockSpinner) UpdateSuffix(s string) { m.suffix = s }

// newSpinnerMu serializes the two tests below that touch the package-level
// newSpinner factory var: TestDisplayProgress_LoopCoverage replaces it with
// a mock for its duration, TestRealSpinner calls it expecting the real
// spinner.New-backed implementation. Both still declare t.Parallel() to run
// concurrently with the rest of the suite; the mutex only serializes these
// two against each other -- without it, concurrent read/write of a plain
// package-level func var would be a genuine data race.
var newSpinnerMu sync.Mutex

func TestRealSpinner(t *testing.T) {
	t.Parallel()
	newSpinnerMu.Lock()
	defer newSpinnerMu.Unlock()

	rs := newSpinner(spinner.WithColor("fgCyan"))

	// Just verify these methods don't panic.
	rs.Start()
	rs.UpdateSuffix(" test")
	rs.Stop()
}

// TestDisplayProgress_LoopCoverage ensures every update sent on the
// progress channel is consumed and that the spinner lifecycle
// (Start / Stop) is honored by DisplayProgress.
//
// Synchronization is deterministic: the channel is unbuffered, so each
// send blocks until DisplayProgress receives it -- no time.Sleep needed.
// Closing the channel drives DisplayProgress out of its select loop.
func TestDisplayProgress_LoopCoverage(t *testing.T) {
	t.Parallel()
	newSpinnerMu.Lock()
	defer newSpinnerMu.Unlock()

	original := newSpinner
	defer func() { newSpinner = original }()

	mockS := &MockSpinner{}
	newSpinner = func(options ...spinner.Option) Spinner {
		return mockS
	}

	var wg sync.WaitGroup
	wg.Add(1)
	progressChan := make(chan progress.ProgressUpdate)

	// A second WaitGroup makes the producer goroutine joinable so the test
	// never returns while the sender is still running (no leaks).
	var sendWG sync.WaitGroup
	sendWG.Add(1)
	go func() {
		defer sendWG.Done()
		// Unbuffered channel: each send rendezvous with DisplayProgress's
		// receive, providing a happens-before edge -- the 5th send is
		// observed before close.
		for i := 0; i < 5; i++ {
			progressChan <- progress.ProgressUpdate{
				CalculatorIndex: 0,
				Value:           float64(i) * 0.2,
			}
		}
		close(progressChan)
	}()

	DisplayProgress(&wg, progressChan, 1, io.Discard)
	wg.Wait()     // ensure DisplayProgress returned
	sendWG.Wait() // ensure producer returned

	if !mockS.started {
		t.Error("Spinner should have started")
	}
	if !mockS.stopped {
		t.Error("Spinner should have stopped after channel close")
	}
}
