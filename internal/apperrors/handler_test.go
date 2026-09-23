package apperrors

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// These tests cover the pure error-to-exit-code mapping this package is
// responsible for. The presentation half lives in internal/cli (audit API-04 /
// ARC-02) and is covered by cli.TestWriteCalculationStatus.

func TestExitCodeFor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil is success", nil, ExitSuccess},
		{"deadline is timeout", context.DeadlineExceeded, ExitErrorTimeout},
		{"cancellation is 130", context.Canceled, ExitErrorCanceled},
		{"anything else is generic", fmt.Errorf("random error"), ExitErrorGeneric},

		// The mapping must look through the chain, not at the concrete type:
		// every calculation error reaches the edge wrapped in context.
		{
			"wrapped deadline is still a timeout",
			WrapCalculationError(context.DeadlineExceeded, CalculationContext{N: 7}),
			ExitErrorTimeout,
		},
		{
			"wrapped cancellation is still 130",
			WrapCalculationError(context.Canceled, CalculationContext{FFTThresholdBits: 1000}),
			ExitErrorCanceled,
		},
		{
			"wrapped generic stays generic",
			WrapCalculationError(fmt.Errorf("boom"), CalculationContext{}),
			ExitErrorGeneric,
		},
		{
			"doubly wrapped deadline",
			fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", context.DeadlineExceeded)),
			ExitErrorTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ExitCodeFor(tt.err); got != tt.want {
				t.Errorf("ExitCodeFor(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}

func TestCalculationDiagnostic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		want     string
		contains []string
	}{
		{name: "nil has no diagnostic", err: nil, want: ""},
		{name: "plain error has no diagnostic", err: fmt.Errorf("boom"), want: ""},
		{
			// An empty context carries nothing worth printing; the presenter
			// relies on the empty string to omit the Diagnostic line entirely.
			name: "empty context yields nothing",
			err:  WrapCalculationError(fmt.Errorf("boom"), CalculationContext{}),
			want: "",
		},
		{
			name: "populated context is rendered",
			err: WrapCalculationError(fmt.Errorf("compute failed"), CalculationContext{
				N:                   42,
				MemoryEstimateBytes: 1024,
			}),
			contains: []string{"n=42", "parallel_bits=auto", "fft_bits=auto", "strassen_bits=auto", "mem_est="},
		},
		{
			name:     "explicit threshold appears instead of auto",
			err:      WrapCalculationError(context.Canceled, CalculationContext{FFTThresholdBits: 1000}),
			contains: []string{"fft_bits=1000", "parallel_bits=auto"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := CalculationDiagnostic(tt.err)

			if len(tt.contains) == 0 {
				if got != tt.want {
					t.Errorf("CalculationDiagnostic() = %q, want %q", got, tt.want)
				}
				return
			}
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("CalculationDiagnostic() = %q, missing %q", got, want)
				}
			}
		})
	}
}
