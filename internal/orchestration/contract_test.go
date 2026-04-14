package orchestration

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	apperrors "github.com/agbru/fibcalc/internal/errors"
	"github.com/agbru/fibcalc/internal/fibonacci"
	"github.com/agbru/fibcalc/internal/fibonacci/fibonaccitest"
)

// Contract tests: invariants that should hold for orchestration + Calculator implementations
// (modest N, fast CI). See docs/INNOVEPLAN.md P2-b.

func TestExecuteCalculations_contextCancelBeforeCompletion(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	core := &fibonaccitest.CoreStub{
		NameVal: "slow",
		CoreFunc: func(ctx context.Context, _ fibonacci.ProgressCallback, n uint64, _ fibonacci.Options) (*big.Int, error) {
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
	if !apperrors.IsContextError(results[0].Err) && !errors.Is(results[0].Err, context.Canceled) {
		t.Errorf("expected context cancellation in error chain, got %v", results[0].Err)
	}
}

func TestExecuteCalculations_progressChannelClosedAndDrained(t *testing.T) {
	t.Parallel()
	core := &fibonaccitest.CoreStub{
		CoreFunc: func(ctx context.Context, reporter fibonacci.ProgressCallback, n uint64, _ fibonacci.Options) (*big.Int, error) {
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
	core := &fibonaccitest.CoreStub{
		CoreFunc: func(context.Context, fibonacci.ProgressCallback, uint64, fibonacci.Options) (*big.Int, error) {
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
