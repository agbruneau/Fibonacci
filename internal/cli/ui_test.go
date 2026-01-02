package cli

import (
	"bytes"
	"io"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agbru/fibcalc/internal/fibonacci"
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

func TestFormatExecutionDuration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		d        time.Duration
		expected string
	}{
		{500 * time.Nanosecond, "0µs"}, // Truncates
		{10 * time.Microsecond, "10µs"},
		{10 * time.Millisecond, "10ms"},
		{2 * time.Second, "2s"},
	}

	for _, tt := range tests {
		got := FormatExecutionDuration(tt.d)
		if got != tt.expected {
			t.Errorf("FormatExecutionDuration(%v) = %s; want %s", tt.d, got, tt.expected)
		}
	}
}

func TestProgressBar(t *testing.T) {
	t.Parallel()
	tests := []struct {
		progress float64
		length   int
		contains string
	}{
		{0.0, 10, "░░░░░░░░░░"},
		{0.5, 10, "█████░░░░░"},
		{1.0, 10, "██████████"},
		{1.2, 10, "██████████"},  // Cap at 1.0
		{-0.1, 10, "░░░░░░░░░░"}, // Floor at 0.0
	}

	for _, tt := range tests {
		got := progressBar(tt.progress, tt.length)
		if got != tt.contains {
			t.Errorf("progressBar(%f, %d) = %s; want %s", tt.progress, tt.length, got, tt.contains)
		}
	}
}

func TestDisplayResult(t *testing.T) {
	// Initialize theme
	ui.InitTheme(false)

	tests := []struct {
		name     string
		result   *big.Int
		n        uint64
		duration time.Duration
		verbose  bool
		details  bool
		concise  bool
		contains []string
	}{
		{
			name:     "Details only",
			result:   big.NewInt(12345),
			n:        10,
			duration: time.Millisecond,
			verbose:  false,
			details:  true,
			concise:  false,
			contains: []string{"Result binary size:", "Detailed result analysis", "Calculation time", "Number of digits"},
		},
		{
			name:     "Concise Output",
			result:   big.NewInt(12345),
			n:        10,
			duration: time.Millisecond,
			verbose:  false,
			details:  false,
			concise:  true,
			contains: []string{"Calculated value", "F(", ") =", "12,345"},
		},
		{
			name:     "Truncated Output",
			result:   new(big.Int).Exp(big.NewInt(10), big.NewInt(200), nil), // Very large number
			n:        100,
			duration: time.Millisecond,
			verbose:  false,
			details:  false,
			concise:  true,
			contains: []string{"(truncated)", "Tip: use"},
		},
		{
			name:     "Verbose Output",
			result:   new(big.Int).Exp(big.NewInt(10), big.NewInt(200), nil),
			n:        100,
			duration: time.Millisecond,
			verbose:  true,
			details:  false,
			concise:  true,
			contains: []string{"F(", ") ="}, // Should not contain truncated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			DisplayResult(tt.result, tt.n, tt.duration, tt.verbose, tt.details, tt.concise, &buf)
			output := buf.String()
			for _, s := range tt.contains {
				if !strings.Contains(output, s) {
					t.Errorf("Expected output to contain %q, but got:\n%s", s, output)
				}
			}
		})
	}
}

func TestFormatNumberString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"1", "1"},
		{"12", "12"},
		{"123", "123"},
		{"1234", "1,234"},
		{"123456", "123,456"},
		{"1234567", "1,234,567"},
		{"-1234", "-1,234"},
	}

	for _, tt := range tests {
		got := formatNumberString(tt.input)
		if got != tt.expected {
			t.Errorf("formatNumberString(%q) = %q; want %q", tt.input, got, tt.expected)
		}
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

	// Just call them to ensure coverage
	_ = ColorReset()
	_ = ColorRed()
	_ = ColorGreen()
	_ = ColorYellow()
	_ = ColorBlue()
	_ = ColorMagenta()
	_ = ColorCyan()
	_ = ColorBold()
	_ = ColorUnderline()
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

	progressChan := make(chan fibonacci.ProgressUpdate)
	out := io.Discard // Discard output

	go func() {
		// Send some updates
		progressChan <- fibonacci.ProgressUpdate{CalculatorIndex: 0, Value: 0.5}
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
	progressChan := make(chan fibonacci.ProgressUpdate)
	close(progressChan)

	DisplayProgress(&wg, progressChan, 0, io.Discard)
	wg.Wait()
	// Should return immediately, coverage check
}
