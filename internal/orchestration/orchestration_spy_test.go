package orchestration

import (
	"context"
	"io"
	"math/big"
	"testing"

	"github.com/agbru/fibcalc/internal/config"
	"github.com/agbru/fibcalc/internal/fibonacci"
)

// TestExecuteCalculationsRespectsStrassenConfig verifies that the orchestration layer
// correctly passes the StrassenThreshold from the AppConfig to the calculator Options.
// This proves that setting the global default in main.go is redundant and safe to remove.
func TestExecuteCalculationsRespectsStrassenConfig(t *testing.T) {
	t.Parallel()
	// 1. Ensure global default is NOT what we are testing for.
	// Default is 256. We'll verify that we can pass a different value (e.g. 1000)
	// and it is used.
	// Since we can't inspect the internal execution, we rely on the fact that
	// ExecuteCalculations creates Options.

	// We can create a SpyCalculator that implements fibonacci.Calculator interface.

	spy := &SpyCalculator{}
	calculators := []fibonacci.Calculator{spy}

	cfg := config.AppConfig{
		N:                 10,
		StrassenThreshold: 12345, // Unique value to verify
		Algo:              "matrix",
	}

	ExecuteCalculations(context.Background(), calculators, cfg, io.Discard)

	if spy.capturedOpts.StrassenThreshold != 12345 {
		t.Errorf("ExecuteCalculations failed to pass StrassenThreshold. Expected 12345, got %d", spy.capturedOpts.StrassenThreshold)
	}
}

type SpyCalculator struct {
	capturedOpts fibonacci.Options
}

func (s *SpyCalculator) Calculate(ctx context.Context, progressChan chan<- fibonacci.ProgressUpdate, calcIndex int, n uint64, opts fibonacci.Options) (*big.Int, error) {
	s.capturedOpts = opts
	return big.NewInt(55), nil // F(10)
}

func (s *SpyCalculator) Name() string {
	return "Spy"
}
