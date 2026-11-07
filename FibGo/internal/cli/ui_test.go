package cli

import (
	"bytes"
	"io"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agbruneau/FibGo/internal/progress"
	"github.com/agbruneau/FibGo/internal/ui"
)

// MockSpinner is retained only for tests that still reference it.
type MockSpinner struct {
	started bool
	stopped bool
	suffix  string
}

func (m *MockSpinner) Start() {
	m.started = true
}

func (m *MockSpinner) Stop() {
	m.stopped = true
}

func (m *MockSpinner) UpdateSuffix(suffix string) {
	m.suffix = suffix
}

func TestDisplayResult(t *testing.T) {
	// Initialize theme
	ui.InitTheme(false)

	tests := []struct {
		name      string
		result    *big.Int
		n         uint64
		duration  time.Duration
		verbose   bool
		details   bool
		showValue bool
		contains  []string
	}{
		{
			name:      "Details only",
			result:    big.NewInt(12345),
			n:         10,
			duration:  time.Millisecond,
			verbose:   false,
			details:   true,
			showValue: false,
			contains:  []string{"Result binary size:", "Detailed result analysis", "Calculation time", "Number of digits"},
		},
		{
			name:      "ShowValue Output",
			result:    big.NewInt(12345),
			n:         10,
			duration:  time.Millisecond,
			verbose:   false,
			details:   false,
			showValue: true,
			contains:  []string{"Calculated value", "F(", ") =", "12,345"},
		},
		{
			name:      "Truncated Output",
			result:    new(big.Int).Exp(big.NewInt(10), big.NewInt(200), nil), // Very large number
			n:         100,
			duration:  time.Millisecond,
			verbose:   false,
			details:   false,
			showValue: true,
			contains:  []string{"(truncated)", "Tip: use"},
		},
		{
			name:      "Verbose Output",
			result:    new(big.Int).Exp(big.NewInt(10), big.NewInt(200), nil),
			n:         100,
			duration:  time.Millisecond,
			verbose:   true,
			details:   false,
			showValue: true,
			contains:  []string{"F(", ") ="}, // Should not contain truncated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			DisplayResult(tt.result, tt.n, tt.duration, tt.verbose, tt.details, tt.showValue, &buf)
			output := buf.String()
			for _, s := range tt.contains {
				if !strings.Contains(output, s) {
					t.Errorf("Expected output to contain %q, but got:\n%s", s, output)
				}
			}
		})
	}
}

// progressLine draws in place with a carriage return and erases what it drew.
// The animation must advance, and a shorter line must not leave a tail of the
// previous one behind — the ETA field shrinks from "ETA: 1m20s" to "ETA: < 1s".
func TestProgressLine(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	p := newProgressLine(&out)

	p.Draw(" first frame, quite long")
	first := out.String()
	if !strings.HasPrefix(first, "\r") {
		t.Errorf("Draw did not return to column 0: %q", first)
	}
	if !strings.Contains(first, "first frame") {
		t.Errorf("Draw did not write its text: %q", first)
	}

	out.Reset()
	p.Draw(" first frame, quite long")
	if second := out.String(); second == first {
		t.Error("the animation frame did not advance between draws")
	}

	out.Reset()
	p.Draw(" short")
	padded := out.String()
	if len([]rune(padded)) < len([]rune(" first frame, quite long")) {
		t.Errorf("a shorter line was not padded over the previous one: %q", padded)
	}

	out.Reset()
	p.Clear()
	cleared := out.String()
	if !strings.HasPrefix(cleared, "\r") || !strings.HasSuffix(cleared, "\r") {
		t.Errorf("Clear must erase and return to column 0: %q", cleared)
	}

	// Clearing twice must not emit a second erase.
	out.Reset()
	p.Clear()
	if out.Len() != 0 {
		t.Errorf("Clear on an already-cleared line wrote %q", out.String())
	}
}

func TestDisplayProgress_ZeroCalculators(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	progressChan := make(chan progress.ProgressUpdate)
	close(progressChan)

	DisplayProgress(&wg, progressChan, 0, io.Discard)
	wg.Wait()
	// Should return immediately, coverage check
}
