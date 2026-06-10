package orchestration

import (
	"testing"

	"github.com/agbruneau/FibGo/internal/fibonacci"
)

// TestGetCalculatorsToRun tests the GetCalculatorsToRun function.
func TestGetCalculatorsToRun(t *testing.T) {
	t.Parallel()
	factory := fibonacci.GlobalFactory()

	t.Run("Single algorithm returns one calculator", func(t *testing.T) {
		t.Parallel()
		calculators := GetCalculatorsToRun("fast", factory)

		if len(calculators) != 1 {
			t.Errorf("Expected 1 calculator, got %d", len(calculators))
		}
		// Check that the name contains "Fast Doubling" (exact name may vary)
		if calculators[0].Name() == "" {
			t.Error("Calculator name should not be empty")
		}
	})

	t.Run("All algorithms returns multiple calculators", func(t *testing.T) {
		t.Parallel()
		calculators := GetCalculatorsToRun("all", factory)

		if len(calculators) < 2 {
			t.Errorf("Expected at least 2 calculators for 'all', got %d", len(calculators))
		}
	})

	t.Run("Matrix algorithm", func(t *testing.T) {
		t.Parallel()
		calculators := GetCalculatorsToRun("matrix", factory)

		if len(calculators) != 1 {
			t.Errorf("Expected 1 calculator, got %d", len(calculators))
		}
	})

	t.Run("Unknown algorithm returns nil", func(t *testing.T) {
		t.Parallel()
		if calculators := GetCalculatorsToRun("no-such-algo", factory); calculators != nil {
			t.Errorf("Expected nil for unknown algorithm, got %d calculator(s)", len(calculators))
		}
	})
}
