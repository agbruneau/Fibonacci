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
