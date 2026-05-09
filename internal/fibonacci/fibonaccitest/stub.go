package fibonaccitest

import (
	"context"
	"math/big"

	"github.com/agbru/fibcalc/internal/fibonacci"
	"github.com/agbru/fibcalc/internal/progress"
)

// CoreStub is a configurable [fibonacci.CoreCalculator] for tests.
type CoreStub struct {
	// NameVal is returned by Name when non-empty.
	NameVal string
	// CoreFunc implements CalculateCore when set.
	CoreFunc func(ctx context.Context, reporter progress.ProgressCallback, n uint64, opts fibonacci.Options) (*big.Int, error)
}

// Name returns NameVal or "stub" if empty.
func (s *CoreStub) Name() string {
	if s.NameVal != "" {
		return s.NameVal
	}
	return "stub"
}

// CalculateCore delegates to CoreFunc or returns 0, nil.
func (s *CoreStub) CalculateCore(ctx context.Context, reporter progress.ProgressCallback, n uint64, opts fibonacci.Options) (*big.Int, error) {
	if s.CoreFunc != nil {
		return s.CoreFunc(ctx, reporter, n, opts)
	}
	return big.NewInt(0), nil
}
