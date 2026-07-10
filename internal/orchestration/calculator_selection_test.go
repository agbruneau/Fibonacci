package orchestration_test

import (
	"testing"

	"github.com/agbruneau/FibGo/internal/fibonacci"
	"github.com/agbruneau/FibGo/internal/orchestration"
)

// TestGetCalculatorsToRun tests the GetCalculatorsToRun function.
func TestGetCalculatorsToRun(t *testing.T) {
	t.Parallel()
	factory := fibonacci.NewDefaultFactory()

	tests := []struct {
		name    string
		algo    string
		wantLen int
		wantNil bool
	}{
		{name: "single algorithm returns one calculator", algo: "fast", wantLen: 1},
		{name: "matrix algorithm", algo: "matrix", wantLen: 1},
		{name: "unknown algorithm returns nil", algo: "no-such-algo", wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			calculators := orchestration.GetCalculatorsToRun(tt.algo, factory)
			if tt.wantNil {
				if calculators != nil {
					t.Errorf("expected nil for %q, got %d calculator(s)", tt.algo, len(calculators))
				}
				return
			}
			if len(calculators) != tt.wantLen {
				t.Errorf("algo %q: expected %d calculator(s), got %d", tt.algo, tt.wantLen, len(calculators))
			}
			// Check that the name is populated (exact name may vary).
			if calculators[0].Name() == "" {
				t.Error("calculator name should not be empty")
			}
		})
	}

	t.Run("all algorithms returns multiple calculators", func(t *testing.T) {
		t.Parallel()
		calculators := orchestration.GetCalculatorsToRun("all", factory)

		if len(calculators) < 2 {
			t.Errorf("Expected at least 2 calculators for 'all', got %d", len(calculators))
		}
	})
}
