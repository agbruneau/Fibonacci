package orchestration

import (
	"context"
	"errors"
	"io"
	"math/big"
	"testing"
	"time"

	apperrors "github.com/agbru/fibcalc/internal/errors"
	"github.com/agbru/fibcalc/internal/fibonacci"
	"github.com/agbru/fibcalc/internal/progress"
)

// MockResultPresenter is a mock implementation of ResultPresenter for testing.
type MockResultPresenter struct{}

func (MockResultPresenter) PresentComparisonTable(results []CalculationResult, out io.Writer) {}
func (MockResultPresenter) PresentResult(result CalculationResult, n uint64, verbose, details, showValue bool, out io.Writer) {
}
func (MockResultPresenter) HandleError(err error, duration time.Duration, out io.Writer) int {
	return apperrors.ExitErrorGeneric
}

// MockCalculator is a mock implementation of fibonacci.Calculator
// used for testing the orchestration logic without invoking real algorithms.
type MockCalculator struct {
	NameFunc      func() string
	CalculateFunc func(ctx context.Context, reporter progress.ProgressCallback, index int, n uint64, opts fibonacci.Options) (*big.Int, error)
}

// Name returns the mocked name of the calculator.
func (m *MockCalculator) Name() string {
	if m.NameFunc != nil {
		return m.NameFunc()
	}
	return "Mock"
}

// Calculate invokes the mocked CalculateFunc.
func (m *MockCalculator) Calculate(ctx context.Context, progressChan chan<- progress.ProgressUpdate, index int, n uint64, opts fibonacci.Options) (*big.Int, error) {
	if m.CalculateFunc != nil {
		// Create a dummy reporter that sends to the channel
		reporter := func(pct float64) {
			if progressChan != nil {
				progressChan <- progress.ProgressUpdate{CalculatorIndex: index, Value: pct}
			}
		}
		return m.CalculateFunc(ctx, reporter, index, n, opts)
	}
	return big.NewInt(0), nil
}

// TestExecuteCalculations verifies that the orchestrator correctly runs calculators
// and aggregates their results.
func TestExecuteCalculations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		calculators []fibonacci.Calculator
		expectedLen int
		expectError bool
	}{
		{
			name: "Single success",
			calculators: []fibonacci.Calculator{
				&MockCalculator{
					CalculateFunc: func(ctx context.Context, reporter progress.ProgressCallback, index int, n uint64, opts fibonacci.Options) (*big.Int, error) {
						return big.NewInt(1), nil
					},
				},
			},
			expectedLen: 1,
			expectError: false,
		},
		{
			name: "Single failure",
			calculators: []fibonacci.Calculator{
				&MockCalculator{
					CalculateFunc: func(ctx context.Context, reporter progress.ProgressCallback, index int, n uint64, opts fibonacci.Options) (*big.Int, error) {
						return nil, errors.New("mock error")
					},
				},
			},
			expectedLen: 1,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			results := ExecuteCalculations(context.Background(), ExecutionConfig{
				Calculators:      tt.calculators,
				N:                0,
				Opts:             fibonacci.Options{},
				ProgressReporter: NullProgressReporter{},
				Out:              &DiscardWriter{},
			})
			if len(results) != tt.expectedLen {
				t.Errorf("expected %d results, got %d", tt.expectedLen, len(results))
			}
			if tt.expectError {
				if results[0].Err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if results[0].Err != nil {
					t.Errorf("unexpected error: %v", results[0].Err)
				}
			}
		})
	}
}

// TestAnalyzeComparisonResults verifies the logic for comparing results from
// multiple algorithms. It checks for consistent results, handling of failures,
// and detection of mismatches.
func TestAnalyzeComparisonResults(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		results        []CalculationResult
		expectedStatus int
	}{
		{
			name: "All success",
			results: []CalculationResult{
				{Name: "A", Result: big.NewInt(5), Duration: time.Millisecond, Err: nil},
				{Name: "B", Result: big.NewInt(5), Duration: time.Millisecond, Err: nil},
			},
			expectedStatus: apperrors.ExitSuccess,
		},
		{
			name: "Mismatch",
			results: []CalculationResult{
				{Name: "A", Result: big.NewInt(5), Duration: time.Millisecond, Err: nil},
				{Name: "B", Result: big.NewInt(6), Duration: time.Millisecond, Err: nil},
			},
			expectedStatus: apperrors.ExitErrorMismatch,
		},
		{
			name: "All failure",
			results: []CalculationResult{
				{Name: "A", Result: nil, Duration: time.Millisecond, Err: errors.New("fail")},
				{Name: "B", Result: nil, Duration: time.Millisecond, Err: errors.New("fail")},
			},
			expectedStatus: apperrors.ExitErrorGeneric,
		},
		{
			name: "Mixed success/failure",
			results: []CalculationResult{
				{Name: "A", Result: big.NewInt(5), Duration: time.Millisecond, Err: nil},
				{Name: "B", Result: nil, Duration: time.Millisecond, Err: errors.New("fail")},
			},
			expectedStatus: apperrors.ExitSuccess,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			status := AnalyzeComparisonResults(tt.results, PresentationOptions{}, MockResultPresenter{}, MockResultPresenter{}, &DiscardWriter{})
			if status != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, status)
			}
		})
	}
}

// DiscardWriter is a helper that implements io.Writer and discards all data.
type DiscardWriter struct{}

func (d *DiscardWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

// TestExecuteCalculations_MultiSuccess covers the errgroup branch (>1 calculator)
// where every calculator succeeds.
func TestExecuteCalculations_MultiSuccess(t *testing.T) {
	t.Parallel()
	calcs := []fibonacci.Calculator{
		&MockCalculator{
			NameFunc: func() string { return "A" },
			CalculateFunc: func(ctx context.Context, reporter progress.ProgressCallback, index int, n uint64, opts fibonacci.Options) (*big.Int, error) {
				return big.NewInt(11), nil
			},
		},
		&MockCalculator{
			NameFunc: func() string { return "B" },
			CalculateFunc: func(ctx context.Context, reporter progress.ProgressCallback, index int, n uint64, opts fibonacci.Options) (*big.Int, error) {
				return big.NewInt(22), nil
			},
		},
	}
	results := ExecuteCalculations(context.Background(), ExecutionConfig{
		Calculators:      calcs,
		N:                10,
		Opts:             fibonacci.Options{},
		ProgressReporter: NullProgressReporter{},
		Out:              &DiscardWriter{},
	})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for i, r := range results {
		if r.Err != nil {
			t.Errorf("results[%d].Err = %v, want nil", i, r.Err)
		}
		if r.Result == nil {
			t.Errorf("results[%d].Result = nil", i)
		}
	}
	if results[0].Name != "A" || results[1].Name != "B" {
		t.Errorf("result name order wrong: got %q, %q", results[0].Name, results[1].Name)
	}
}

// TestExecuteCalculations_PartialFailure covers the errgroup branch where one
// calculator returns an error and the sibling still gets to record its result.
// errgroup propagates the first error to cancel siblings; we assert that:
//   - the failing slot has Err != nil
//   - successful slots still have a recorded result (or, if they observed the
//     cancellation, an error)
func TestExecuteCalculations_PartialFailure(t *testing.T) {
	t.Parallel()
	calcs := []fibonacci.Calculator{
		&MockCalculator{
			NameFunc: func() string { return "Fail" },
			CalculateFunc: func(ctx context.Context, reporter progress.ProgressCallback, index int, n uint64, opts fibonacci.Options) (*big.Int, error) {
				return nil, errors.New("boom")
			},
		},
		&MockCalculator{
			NameFunc: func() string { return "Slow" },
			CalculateFunc: func(ctx context.Context, reporter progress.ProgressCallback, index int, n uint64, opts fibonacci.Options) (*big.Int, error) {
				// Wait briefly so the failing goroutine has time to error first
				// and propagate cancellation through the errgroup context.
				select {
				case <-time.After(50 * time.Millisecond):
					return big.NewInt(99), nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			},
		},
	}
	results := ExecuteCalculations(context.Background(), ExecutionConfig{
		Calculators:      calcs,
		N:                10,
		Opts:             fibonacci.Options{},
		ProgressReporter: NullProgressReporter{},
		Out:              &DiscardWriter{},
	})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Err == nil {
		t.Errorf("results[0] (failing calc) Err = nil, want non-nil")
	}
	// Either the slow goroutine completed before cancellation propagated, or it
	// observed ctx.Done(). Both outcomes are valid; what matters is that a
	// per-slot result was recorded — no goroutine leak / lost write.
	if results[1].Name != "Slow" {
		t.Errorf("results[1].Name = %q, want Slow", results[1].Name)
	}
}

// TestExecuteCalculations_ContextCancellation verifies that when the caller
// cancels the parent context, all calculators observe Done() and return an
// error, and every slot is filled (no leaked goroutine).
func TestExecuteCalculations_ContextCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())

	started := make(chan struct{}, 2)
	calcs := []fibonacci.Calculator{
		&MockCalculator{
			NameFunc: func() string { return "A" },
			CalculateFunc: func(ctx context.Context, reporter progress.ProgressCallback, index int, n uint64, opts fibonacci.Options) (*big.Int, error) {
				started <- struct{}{}
				<-ctx.Done()
				return nil, ctx.Err()
			},
		},
		&MockCalculator{
			NameFunc: func() string { return "B" },
			CalculateFunc: func(ctx context.Context, reporter progress.ProgressCallback, index int, n uint64, opts fibonacci.Options) (*big.Int, error) {
				started <- struct{}{}
				<-ctx.Done()
				return nil, ctx.Err()
			},
		},
	}

	// Cancel after both goroutines have begun blocking on ctx.Done().
	go func() {
		<-started
		<-started
		cancel()
	}()

	results := ExecuteCalculations(ctx, ExecutionConfig{
		Calculators:      calcs,
		N:                10,
		Opts:             fibonacci.Options{},
		ProgressReporter: NullProgressReporter{},
		Out:              &DiscardWriter{},
	})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for i, r := range results {
		if r.Err == nil {
			t.Errorf("results[%d].Err = nil, want context error", i)
		}
	}
}
