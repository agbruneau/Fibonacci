// This file stays package calibration (white-box) purely so its exported
// test doubles are visible to BOTH the white-box test files in this
// directory (which need unexported access) and the black-box
// calibration_test package files (which import "internal/calibration"
// normally). Go compiles internal (package calibration) test files and
// external (package calibration_test) test files into the same test
// binary, so an exported type declared here is reachable from either side
// -- this is the standard way to share a non-trivial fixture across that
// boundary without duplicating its behavior in multiple files.
package calibration

import (
	"context"
	"errors"
	"io"
	"math/big"
	"sync"
	"time"

	apperrors "github.com/agbruneau/FibGo/internal/errors"
	"github.com/agbruneau/FibGo/internal/fibonacci"
	"github.com/agbruneau/FibGo/internal/progress"
)

// errSimulated is the error MockFailingCalculator always returns.
var errSimulated = errors.New("simulated error")

// MockCalculator implements fibonacci.Calculator for calibration tests. Its
// Calculate simulates a duration that improves ("speeds up") when the
// threshold options fall in specific ranges, so tests can assert that the
// calibration/escalation logic actually picks the fastest combination
// rather than an arbitrary one.
type MockCalculator struct {
	name string
}

var _ fibonacci.Calculator = (*MockCalculator)(nil)

// NewMockCalculator builds a MockCalculator with the given name. It exists
// so black-box test files (package calibration_test) can construct one
// without reaching into the unexported name field.
func NewMockCalculator(name string) *MockCalculator {
	return &MockCalculator{name: name}
}

func (m *MockCalculator) Name() string {
	return m.name
}

func (m *MockCalculator) Calculate(ctx context.Context, _ chan<- progress.ProgressUpdate, _ int, _ uint64, opts fibonacci.Options) (*big.Int, error) {
	// Simulate work duration dependent on threshold to test optimization
	// logic. Cumulative speedups ensure the combination of optimal
	// parameters yields the strictly fastest time.
	duration := 100 * time.Millisecond

	// Parallel speedup. Accept a range (2048-8192) to match adaptive
	// environment logic across CPU counts.
	if opts.ParallelThreshold >= 2048 && opts.ParallelThreshold <= 8192 {
		duration -= 40 * time.Millisecond
	}

	// FFT speedup.
	if opts.FFTThreshold >= 750000 && opts.FFTThreshold <= 1000000 {
		duration -= 20 * time.Millisecond
	}

	// Strassen speedup.
	if opts.StrassenThreshold >= 192 && opts.StrassenThreshold <= 512 {
		duration -= 20 * time.Millisecond
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(duration):
	}

	return big.NewInt(1), nil
}

// MockFailingCalculator always returns an error, for testing calibration's
// error handling paths.
type MockFailingCalculator struct{}

var _ fibonacci.Calculator = (*MockFailingCalculator)(nil)

func (m *MockFailingCalculator) Name() string { return "fail" }
func (m *MockFailingCalculator) Calculate(context.Context, chan<- progress.ProgressUpdate, int, uint64, fibonacci.Options) (*big.Int, error) {
	return nil, errSimulated
}

// MockBlockingCalculator simulates a long-running calculation. Calculate
// closes EnteredChan (if set) so callers can synchronize on the
// calculation having actually started, instead of guessing with a fixed
// sleep, then blocks on BlockChan (if set) until the caller releases it.
type MockBlockingCalculator struct {
	BlockChan   chan struct{}
	EnteredChan chan struct{}
}

var _ fibonacci.Calculator = (*MockBlockingCalculator)(nil)

func (m *MockBlockingCalculator) Name() string { return "block" }
func (m *MockBlockingCalculator) Calculate(context.Context, chan<- progress.ProgressUpdate, int, uint64, fibonacci.Options) (*big.Int, error) {
	if m.EnteredChan != nil {
		close(m.EnteredChan)
	}
	if m.BlockChan != nil {
		<-m.BlockChan
	}
	return big.NewInt(1), nil
}

// NoopProgressDisplay drains the progress channel without output, for
// tests that must supply a ProgressDisplayFunc but do not assert on it.
func NoopProgressDisplay(wg *sync.WaitGroup, progressChan <-chan progress.ProgressUpdate, _ int, _ io.Writer) {
	defer wg.Done()
	for range progressChan {
	}
}

// NoopColorProvider returns empty strings for all colors, for tests that
// must supply an apperrors.ColorProvider but do not assert on color codes.
type NoopColorProvider struct{}

var _ apperrors.ColorProvider = NoopColorProvider{}

func (NoopColorProvider) Red() string    { return "" }
func (NoopColorProvider) Yellow() string { return "" }
func (NoopColorProvider) Reset() string  { return "" }
