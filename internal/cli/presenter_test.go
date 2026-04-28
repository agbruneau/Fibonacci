package cli

import (
	"bytes"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/agbru/fibcalc/internal/orchestration"
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

func TestCLIResultPresenter_FormatDuration(t *testing.T) {
	t.Parallel()
	p := CLIResultPresenter{}
	if p.FormatDuration(0) == "" {
		t.Fatal("FormatDuration(0) returned empty string")
	}
	if p.FormatDuration(123*time.Millisecond) == p.FormatDuration(456*time.Millisecond) {
		t.Fatal("FormatDuration does not differentiate distinct inputs")
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
		tc := tc
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

func TestDisplayMemoryStats(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name          string
		heapAlloc     uint64
		totalAlloc    uint64
		numGC         uint32
		pauseTotalNs  uint64
		wantSubstring []string
	}{
		{
			name:          "gc disabled branch",
			heapAlloc:     1 << 20,
			totalAlloc:    1 << 22,
			numGC:         0,
			pauseTotalNs:  0,
			wantSubstring: []string{"GC pause total:  0ms (GC disabled)"},
		},
		{
			name:          "gc enabled branch",
			heapAlloc:     2 << 20,
			totalAlloc:    8 << 20,
			numGC:         3,
			pauseTotalNs:  5_000_000, // 5 ms
			wantSubstring: []string{"GC pause total:  5.00ms", "GC cycles:       3"},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			DisplayMemoryStats(tc.heapAlloc, tc.totalAlloc, tc.numGC, tc.pauseTotalNs, &buf)
			for _, want := range tc.wantSubstring {
				if !strings.Contains(buf.String(), want) {
					t.Errorf("DisplayMemoryStats output missing %q; got:\n%s", want, buf.String())
				}
			}
		})
	}
}

func TestPadSpaces(t *testing.T) {
	t.Parallel()
	cases := []struct {
		length int
		want   string
	}{
		{-1, ""},
		{0, ""},
		{1, " "},
		{4, "    "},
	}
	for _, tc := range cases {
		if got := padSpaces(tc.length); got != tc.want {
			t.Errorf("padSpaces(%d) = %q, want %q", tc.length, got, tc.want)
		}
	}
}
