package orchestration

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/agbruneau/FibGo/internal/fibonacci"
	"github.com/agbruneau/FibGo/internal/progress"
)

// Contract tests: invariants that should hold for orchestration + Calculator
// implementations (modest N, fast to run). See docs/TESTING.md.

// coreStub is a configurable fibonacci.CoreCalculator for these contract
// tests. Moved here from the former internal/fibonacci/fibonaccitest package,
// whose only consumer was this file (audit Fable5 DEAD-06).
type coreStub struct {
	// nameVal is returned by Name when non-empty.
	nameVal string
	// coreFunc implements CalculateCore when set.
	coreFunc func(ctx context.Context, reporter progress.ProgressCallback, n uint64, opts fibonacci.Options) (*big.Int, error)
}

// Name returns nameVal or "stub" if empty.
func (s *coreStub) Name() string {
	if s.nameVal != "" {
		return s.nameVal
	}
	return "stub"
}

// CalculateCore delegates to coreFunc or returns 0, nil.
func (s *coreStub) CalculateCore(ctx context.Context, reporter progress.ProgressCallback, n uint64, opts fibonacci.Options) (*big.Int, error) {
	if s.coreFunc != nil {
		return s.coreFunc(ctx, reporter, n, opts)
	}
	return big.NewInt(0), nil
}

func TestExecuteCalculations_contextCancelBeforeCompletion(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	core := &coreStub{
		nameVal: "slow",
		coreFunc: func(ctx context.Context, _ progress.ProgressCallback, n uint64, _ fibonacci.Options) (*big.Int, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}
	calc, err := fibonacci.NewCalculator(core)
	if err != nil {
		t.Fatal(err)
	}

	results := ExecuteCalculations(ctx, ExecutionConfig{
		Calculators:      []fibonacci.Calculator{calc},
		N:                200,
		Opts:             fibonacci.Options{},
		ProgressReporter: NullProgressReporter{},
		Out:              &DiscardWriter{},
	})
	if len(results) != 1 {
		t.Fatalf("len(results) = %d", len(results))
	}
	if results[0].Err == nil {
		t.Fatal("expected error from canceled context")
	}
	if !errors.Is(results[0].Err, context.Canceled) {
		t.Errorf("expected context cancellation in error chain, got %v", results[0].Err)
	}
}

func TestExecuteCalculations_progressChannelClosedAndDrained(t *testing.T) {
	t.Parallel()
	core := &coreStub{
		coreFunc: func(ctx context.Context, reporter progress.ProgressCallback, n uint64, _ fibonacci.Options) (*big.Int, error) {
			reporter(0.5)
			return big.NewInt(1), nil
		},
	}
	calc, err := fibonacci.NewCalculator(core)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		ExecuteCalculations(context.Background(), ExecutionConfig{
			Calculators:      []fibonacci.Calculator{calc},
			N:                200,
			Opts:             fibonacci.Options{},
			ProgressReporter: NullProgressReporter{},
			Out:              &DiscardWriter{},
		})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("orchestration blocked — progress channel likely not closed or display goroutine stuck")
	}
}

func TestExecuteCalculations_noPanicFromCalculatorError(t *testing.T) {
	t.Parallel()
	core := &coreStub{
		coreFunc: func(context.Context, progress.ProgressCallback, uint64, fibonacci.Options) (*big.Int, error) {
			return nil, errors.New("expected failure")
		},
	}
	calc, err := fibonacci.NewCalculator(core)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("unexpected panic: %v", r)
		}
	}()
	results := ExecuteCalculations(context.Background(), ExecutionConfig{
		Calculators:      []fibonacci.Calculator{calc},
		N:                200,
		Opts:             fibonacci.Options{},
		ProgressReporter: NullProgressReporter{},
		Out:              &DiscardWriter{},
	})
	if len(results) != 1 || results[0].Err == nil {
		t.Fatalf("expected wrapped error, got %+v", results)
	}
}
