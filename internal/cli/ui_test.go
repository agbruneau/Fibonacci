package cli

import (
	"bytes"
	"io"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agbru/fibcalc/internal/progress"
	"github.com/agbru/fibcalc/internal/ui"
	"github.com/briandowns/spinner"
)

// MockSpinner for testing
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

func TestRealSpinner(t *testing.T) {
	t.Parallel()
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	rs := &realSpinner{s}

	// Just verify these methods don't panic
	rs.Start()
	rs.UpdateSuffix(" test")
	rs.Stop()
}

func TestColors(t *testing.T) {
	// Initialize with false (colors enabled if terminal supports)
	ui.InitTheme(false)

	// Just call them to ensure coverage - use ui package directly
	_ = ui.ColorReset()
	_ = ui.ColorRed()
	_ = ui.ColorGreen()
	_ = ui.ColorYellow()
	_ = ui.ColorBlue()
	_ = ui.ColorMagenta()
	_ = ui.ColorCyan()
	_ = ui.ColorBold()
	_ = ui.ColorUnderline()
}

func TestDisplayProgress(t *testing.T) {
	// Override newSpinner to use mock
	// Note: We can't easily override newSpinner since it's a var but local to the package?
	// Ah, it IS a var in ui.go: var newSpinner = func...
	// So we can override it!

	originalNewSpinner := newSpinner
	defer func() { newSpinner = originalNewSpinner }()

	mockS := &MockSpinner{}
	newSpinner = func(options ...spinner.Option) Spinner {
		return mockS
	}

	var wg sync.WaitGroup
	wg.Add(1)

	progressChan := make(chan progress.ProgressUpdate)
	out := io.Discard // Discard output

	go func() {
		// Send some updates
		progressChan <- progress.ProgressUpdate{CalculatorIndex: 0, Value: 0.5}
		time.Sleep(10 * time.Millisecond)
		close(progressChan)
	}()

	DisplayProgress(&wg, progressChan, 1, out)
	wg.Wait()

	if !mockS.started {
		t.Error("Spinner should have started")
	}
	if !mockS.stopped {
		t.Error("Spinner should have stopped")
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
