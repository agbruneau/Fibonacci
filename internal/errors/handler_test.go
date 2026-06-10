package apperrors

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

type MockColorProvider struct{}

func (m MockColorProvider) Yellow() string { return "[YELLOW]" }
func (m MockColorProvider) Reset() string  { return "[RESET]" }

func TestHandleCalculationError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		err          error
		duration     time.Duration
		colors       ColorProvider
		expectedCode int
		expectedMsg  string
	}{
		{
			name:         "No Error",
			err:          nil,
			expectedCode: ExitSuccess,
			expectedMsg:  "",
		},
		{
			name:         "Timeout Error",
			err:          context.DeadlineExceeded,
			duration:     1 * time.Second,
			colors:       MockColorProvider{},
			expectedCode: ExitErrorTimeout,
			expectedMsg:  "Status: Failure (Timeout). The execution limit was reached after [YELLOW]1s[RESET].",
		},
		{
			name:         "Canceled Error",
			err:          context.Canceled,
			duration:     500 * time.Millisecond,
			colors:       MockColorProvider{},
			expectedCode: ExitErrorCanceled,
			expectedMsg:  "[YELLOW]Status: Canceled after [YELLOW]500ms[RESET].[RESET]",
		},
		{
			name:         "Generic Error",
			err:          fmt.Errorf("random error"),
			expectedCode: ExitErrorGeneric,
			expectedMsg:  "Status: Failure. An unexpected error occurred: random error",
		},
		{
			name: "Generic Error with calculation diagnostic",
			err: WrapCalculationError(fmt.Errorf("compute failed"), CalculationContext{
				N:                     42,
				ParallelThresholdBits: 0,
				FFTThresholdBits:      0,
				StrassenThresholdBits: 0,
				MemoryEstimateBytes:   1024,
			}),
			expectedCode: ExitErrorGeneric,
			expectedMsg:  "Status: Failure. An unexpected error occurred: compute failed\nDiagnostic: n=42; parallel_bits=auto; fft_bits=auto; strassen_bits=auto; mem_est=",
		},
		{
			name:         "Default Colors",
			err:          context.DeadlineExceeded,
			duration:     1 * time.Second,
			colors:       nil,
			expectedCode: ExitErrorTimeout,
			expectedMsg:  "Status: Failure (Timeout). The execution limit was reached after 1s.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := new(bytes.Buffer)
			code := HandleCalculationError(tt.err, tt.duration, out, tt.colors)

			if code != tt.expectedCode {
				t.Errorf("HandleCalculationError() code = %v, want %v", code, tt.expectedCode)
			}

			if tt.expectedMsg != "" && !strings.Contains(out.String(), tt.expectedMsg) {
				t.Errorf("HandleCalculationError() output = %q, want %q", out.String(), tt.expectedMsg)
			}
		})
	}
}

func TestHandleCalculationError_DiagnosticPaths(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		err          error
		duration     time.Duration
		expectedCode int
		contains     []string
		notContains  []string
	}{
		{
			name:         "timeout with diagnostic",
			err:          WrapCalculationError(context.DeadlineExceeded, CalculationContext{N: 7}),
			duration:     time.Second,
			expectedCode: ExitErrorTimeout,
			contains: []string{
				"Status: Failure (Timeout).",
				"Diagnostic: n=7; parallel_bits=auto; fft_bits=auto; strassen_bits=auto",
			},
		},
		{
			name:         "canceled with diagnostic",
			err:          WrapCalculationError(context.Canceled, CalculationContext{FFTThresholdBits: 1000}),
			duration:     time.Second,
			expectedCode: ExitErrorCanceled,
			contains: []string{
				"Status: Canceled",
				"Diagnostic: parallel_bits=auto; fft_bits=1000; strassen_bits=auto",
			},
		},
		{
			// An empty context must not emit a Diagnostic line.
			name:         "calculation error without diagnostic",
			err:          WrapCalculationError(fmt.Errorf("boom"), CalculationContext{}),
			expectedCode: ExitErrorGeneric,
			contains:     []string{"boom"},
			notContains:  []string{"Diagnostic:"},
		},
		{
			name:         "zero duration omits suffix",
			err:          context.DeadlineExceeded,
			duration:     0,
			expectedCode: ExitErrorTimeout,
			contains:     []string{"The execution limit was reached.\n"},
			notContains:  []string{" after "},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := new(bytes.Buffer)
			code := HandleCalculationError(tt.err, tt.duration, out, nil)

			if code != tt.expectedCode {
				t.Errorf("HandleCalculationError() code = %v, want %v", code, tt.expectedCode)
			}
			for _, want := range tt.contains {
				if !strings.Contains(out.String(), want) {
					t.Errorf("output %q missing %q", out.String(), want)
				}
			}
			for _, banned := range tt.notContains {
				if strings.Contains(out.String(), banned) {
					t.Errorf("output %q must not contain %q", out.String(), banned)
				}
			}
		})
	}
}

func TestDefaultColorProvider(t *testing.T) {
	t.Parallel()
	p := DefaultColorProvider{}
	if p.Yellow() != "" {
		t.Error("DefaultColorProvider.Yellow should return empty string")
	}
	if p.Reset() != "" {
		t.Error("DefaultColorProvider.Reset should return empty string")
	}
}
