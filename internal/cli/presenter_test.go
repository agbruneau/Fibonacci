package cli

import (
	"bytes"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/FibGo/internal/orchestration"
)

// PR#27 / P1-25:
// These tests cover the thin CLI adapter methods in presenter.go. The
// underlying DisplayProgress / DisplayResult / HandleCalculationError are
// tested elsewhere; these tests pin down that the adapter methods wire
// their arguments through and that CLIResultPresenter satisfies the
// three orchestration interfaces it is documented to implement.

func TestCLIResultPresenter_InterfaceCompliance(t *testing.T) {
	t.Parallel()
	var (
		_ orchestration.ResultPresenter = CLIResultPresenter{}
		_ orchestration.ErrorHandler    = CLIResultPresenter{}
	)
}

func TestCLIResultPresenter_PresentComparisonTable(t *testing.T) {
	t.Parallel()
	results := []orchestration.CalculationResult{
		{Name: "Fast Doubling", Duration: 1500 * time.Microsecond, Result: big.NewInt(5)},
		{Name: "Matrix Exp", Duration: 0, Err: errors.New("boom")},
	}
	var buf bytes.Buffer
	CLIResultPresenter{}.PresentComparisonTable(results, &buf)
	out := buf.String()
	for _, want := range []string{"Comparison Summary", "Fast Doubling", "Matrix Exp", "boom", "< 1µs"} {
		if !strings.Contains(out, want) {
			t.Errorf("PresentComparisonTable output missing %q; got:\n%s", want, out)
		}
	}
}

func TestCLIResultPresenter_PresentResult(t *testing.T) {
	t.Parallel()
	res := orchestration.CalculationResult{
		Name:     "Fast Doubling",
		Result:   big.NewInt(21),
		Duration: 5 * time.Millisecond,
	}
	var buf bytes.Buffer
	CLIResultPresenter{}.PresentResult(res, 8, false, false, true, &buf)
	// Adapter success is "produces any output"; the content is pinned down
	// by ui_display_test.go's TestDisplayResult* cases.
	if buf.Len() == 0 {
		t.Fatal("PresentResult produced no output")
	}
}

func TestCLIResultPresenter_HandleError(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name          string
		machineOutput bool
	}{
		{"colored", false},
		{"machine", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			p := CLIResultPresenter{MachineOutput: tc.machineOutput}
			code := p.HandleError(errors.New("boom"), 2*time.Millisecond, &buf)
			// HandleCalculationError returns a non-zero exit code for a
			// plain error; exact value is pinned by errors/handler_test.go.
			if code == 0 {
				t.Errorf("HandleError(err) should return non-zero exit code, got 0")
			}
			if buf.Len() == 0 {
				t.Errorf("HandleError produced no output")
			}
		})
	}
}
