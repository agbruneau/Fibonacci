package fibonaccitest

import (
	"context"
	"math/big"
	"testing"

	"github.com/agbru/fibcalc/internal/fibonacci"
	"github.com/agbru/fibcalc/internal/progress"
)

func TestCoreStub_Name_default(t *testing.T) {
	t.Parallel()
	var s CoreStub
	if got := s.Name(); got != "stub" {
		t.Errorf("Name() = %q, want stub", got)
	}
}

func TestCoreStub_CalculateCore_default(t *testing.T) {
	t.Parallel()
	var s CoreStub
	res, err := s.CalculateCore(context.Background(), nil, 10, fibonacci.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Int64() != 0 {
		t.Errorf("got %v, want 0", res)
	}
}

func TestCoreStub_WrapAsCalculator(t *testing.T) {
	t.Parallel()
	stub := &CoreStub{
		NameVal: "inject",
		CoreFunc: func(ctx context.Context, reporter progress.ProgressCallback, n uint64, opts fibonacci.Options) (*big.Int, error) {
			return big.NewInt(42), nil
		},
	}
	calc, err := fibonacci.NewCalculator(stub)
	if err != nil {
		t.Fatal(err)
	}
	if calc.Name() != "inject" {
		t.Errorf("Name = %q", calc.Name())
	}
	res, err := calc.Calculate(context.Background(), nil, 0, 100, fibonacci.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Int64() != 42 {
		t.Errorf("result = %v", res)
	}
}
