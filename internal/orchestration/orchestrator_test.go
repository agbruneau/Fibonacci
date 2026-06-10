package orchestration

import (
	"context"
	"errors"
	"io"
	"math/big"
	"sync"
	"testing"
	"time"

	apperrors "github.com/agbruneau/FibGo/internal/errors"
	"github.com/agbruneau/FibGo/internal/fibonacci"
	"github.com/agbruneau/FibGo/internal/progress"
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

// capturingPresenter records what the orchestrator hands to the presentation
// layer so tests can assert on ordering and winner selection.
type capturingPresenter struct {
	table     []CalculationResult
	presented []CalculationResult
}

func (c *capturingPresenter) PresentComparisonTable(results []CalculationResult, _ io.Writer) {
	c.table = append([]CalculationResult(nil), results...)
}

func (c *capturingPresenter) PresentResult(result CalculationResult, _ uint64, _, _, _ bool, _ io.Writer) {
	c.presented = append(c.presented, result)
}

// capturingErrorHandler records the error it receives and returns a fixed code
// so tests can verify the delegation contract of AnalyzeComparisonResults.
type capturingErrorHandler struct {
	got  error
	code int
}

func (h *capturingErrorHandler) HandleError(err error, _ time.Duration, _ io.Writer) int {
	h.got = err
	return h.code
}

// TestExecuteCalculations_WrapsErrorWithCalculationContext verifies the error
// aggregation contract: a calculator failure must surface as a CalculationError
// carrying the run parameters, without losing the original cause.
func TestExecuteCalculations_WrapsErrorWithCalculationContext(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("backend failure")
	calc := &MockCalculator{
		CalculateFunc: func(context.Context, progress.ProgressCallback, int, uint64, fibonacci.Options) (*big.Int, error) {
			return nil, sentinel
		},
	}

	results := ExecuteCalculations(context.Background(), ExecutionConfig{
		Calculators:      []fibonacci.Calculator{calc},
		N:                42,
		Opts:             fibonacci.Options{ParallelThreshold: 7, FFTThreshold: 9, StrassenThreshold: 11},
		ProgressReporter: NullProgressReporter{},
		Out:              io.Discard,
	})

	err := results[0].Err
	if !errors.Is(err, sentinel) {
		t.Fatalf("wrapped error chain lost the cause: %v", err)
	}
	var calcErr apperrors.CalculationError
	if !errors.As(err, &calcErr) {
		t.Fatalf("error is %T, want apperrors.CalculationError in the chain", err)
	}
	if calcErr.N != 42 {
		t.Errorf("CalculationError.N = %d, want 42", calcErr.N)
	}
	if calcErr.ParallelThresholdBits != 7 || calcErr.FFTThresholdBits != 9 || calcErr.StrassenThresholdBits != 11 {
		t.Errorf("threshold context = (%d, %d, %d), want (7, 9, 11)",
			calcErr.ParallelThresholdBits, calcErr.FFTThresholdBits, calcErr.StrassenThresholdBits)
	}
	if calcErr.MemoryEstimateBytes == 0 {
		t.Error("CalculationError.MemoryEstimateBytes = 0, want a non-zero estimate")
	}
}

// TestExecuteCalculations_ErrgroupCancelsSiblings proves cross-calculator
// cancellation (P0-08) deterministically: the second calculator blocks solely
// on the errgroup-derived context, so the only way ExecuteCalculations can
// return is the sibling failure propagating through the shared context.
func TestExecuteCalculations_ErrgroupCancelsSiblings(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	calcs := []fibonacci.Calculator{
		&MockCalculator{
			NameFunc: func() string { return "Failing" },
			CalculateFunc: func(context.Context, progress.ProgressCallback, int, uint64, fibonacci.Options) (*big.Int, error) {
				return nil, boom
			},
		},
		&MockCalculator{
			NameFunc: func() string { return "Blocked" },
			CalculateFunc: func(ctx context.Context, _ progress.ProgressCallback, _ int, _ uint64, _ fibonacci.Options) (*big.Int, error) {
				// Parent context is Background: unblocking here can only come
				// from errgroup cancellation triggered by the sibling error.
				<-ctx.Done()
				return nil, ctx.Err()
			},
		},
	}

	var results []CalculationResult
	done := make(chan struct{})
	go func() {
		defer close(done)
		results = ExecuteCalculations(context.Background(), ExecutionConfig{
			Calculators:      calcs,
			N:                10,
			Opts:             fibonacci.Options{},
			ProgressReporter: NullProgressReporter{},
			Out:              io.Discard,
		})
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("ExecuteCalculations did not return: sibling cancellation through errgroup is broken")
	}

	if !errors.Is(results[0].Err, boom) {
		t.Errorf("results[0].Err = %v, want chain containing %v", results[0].Err, boom)
	}
	if !errors.Is(results[1].Err, context.Canceled) {
		t.Errorf("results[1].Err = %v, want chain containing context.Canceled", results[1].Err)
	}
}

// TestExecuteCalculations_ProgressReporterFuncReceivesUpdates exercises the
// ProgressReporterFunc adapter end to end: the orchestrator must forward the
// calculator count, the configured writer and every progress update, and must
// only return once the display goroutine has drained the closed channel.
func TestExecuteCalculations_ProgressReporterFuncReceivesUpdates(t *testing.T) {
	t.Parallel()
	var (
		got    []progress.ProgressUpdate
		gotNum int
		gotOut io.Writer
	)
	reporter := ProgressReporterFunc(func(wg *sync.WaitGroup, ch <-chan progress.ProgressUpdate, numCalculators int, out io.Writer) {
		defer wg.Done()
		gotNum = numCalculators
		gotOut = out
		for u := range ch {
			got = append(got, u)
		}
	})

	calc := &MockCalculator{
		CalculateFunc: func(_ context.Context, report progress.ProgressCallback, _ int, _ uint64, _ fibonacci.Options) (*big.Int, error) {
			report(0.5)
			report(1.0)
			return big.NewInt(1), nil
		},
	}

	ExecuteCalculations(context.Background(), ExecutionConfig{
		Calculators:      []fibonacci.Calculator{calc},
		N:                1,
		Opts:             fibonacci.Options{},
		ProgressReporter: reporter,
		Out:              io.Discard,
	})

	// ExecuteCalculations returns only after displayWg.Wait(), so reading the
	// state captured by the reporter goroutine is race-free by construction.
	if gotNum != 1 {
		t.Errorf("reporter numCalculators = %d, want 1", gotNum)
	}
	if gotOut != io.Discard {
		t.Errorf("reporter out = %v, want cfg.Out (io.Discard)", gotOut)
	}
	want := []float64{0.5, 1.0}
	if len(got) != len(want) {
		t.Fatalf("reporter received %d updates, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].Value != w || got[i].CalculatorIndex != 0 {
			t.Errorf("update[%d] = %+v, want {CalculatorIndex:0 Value:%v}", i, got[i], w)
		}
	}
}

// TestAnalyzeComparisonResults_SortsAndPresentsFastestSuccess pins the sort
// contract (successes before failures, then ascending duration) and verifies
// the fastest successful result is the one handed to PresentResult.
func TestAnalyzeComparisonResults_SortsAndPresentsFastestSuccess(t *testing.T) {
	t.Parallel()
	results := []CalculationResult{
		{Name: "SlowOK", Result: big.NewInt(5), Duration: 30 * time.Millisecond},
		{Name: "FailedFast", Result: nil, Duration: time.Millisecond, Err: errors.New("x")},
		{Name: "FastOK", Result: big.NewInt(5), Duration: 10 * time.Millisecond},
	}
	pres := &capturingPresenter{}

	status := AnalyzeComparisonResults(results, PresentationOptions{N: 7}, pres, &capturingErrorHandler{}, &DiscardWriter{})

	if status != apperrors.ExitSuccess {
		t.Fatalf("status = %d, want %d", status, apperrors.ExitSuccess)
	}
	// FailedFast has the smallest duration but must still sort last.
	wantOrder := []string{"FastOK", "SlowOK", "FailedFast"}
	for i, name := range wantOrder {
		if results[i].Name != name {
			t.Errorf("results[%d].Name = %q, want %q (successes first, then by duration)", i, results[i].Name, name)
		}
	}
	if len(pres.table) != 3 || pres.table[0].Name != "FastOK" {
		t.Errorf("comparison table got %d entries starting with %q, want 3 starting with FastOK", len(pres.table), firstName(pres.table))
	}
	if len(pres.presented) != 1 || pres.presented[0].Name != "FastOK" {
		t.Errorf("PresentResult got %+v, want exactly the fastest success FastOK", pres.presented)
	}
}

// TestAnalyzeComparisonResults_AllFailureDelegatesFirstError verifies the
// global-failure path: the first error in sorted order is delegated to the
// error handler, whose exit code becomes the return value, and no final
// result is presented.
func TestAnalyzeComparisonResults_AllFailureDelegatesFirstError(t *testing.T) {
	t.Parallel()
	errA := errors.New("err-A")
	errB := errors.New("err-B")
	results := []CalculationResult{
		{Name: "B", Err: errB, Duration: 20 * time.Millisecond},
		{Name: "A", Err: errA, Duration: 10 * time.Millisecond},
	}
	pres := &capturingPresenter{}
	handler := &capturingErrorHandler{code: 42}

	status := AnalyzeComparisonResults(results, PresentationOptions{}, pres, handler, &DiscardWriter{})

	if status != 42 {
		t.Errorf("status = %d, want the error handler's code 42", status)
	}
	if !errors.Is(handler.got, errA) {
		t.Errorf("HandleError received %v, want first error after duration sort (%v)", handler.got, errA)
	}
	if len(pres.presented) != 0 {
		t.Errorf("PresentResult called with %+v, want no call when every algorithm failed", pres.presented)
	}
	if len(pres.table) != 2 {
		t.Errorf("comparison table got %d entries, want 2 even on global failure", len(pres.table))
	}
}

// firstName returns the first result name or a placeholder for error messages.
func firstName(results []CalculationResult) string {
	if len(results) == 0 {
		return "<empty>"
	}
	return results[0].Name
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
