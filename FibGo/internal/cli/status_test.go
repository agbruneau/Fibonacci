package cli

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/FibGo/internal/apperrors"
	"github.com/agbruneau/FibGo/internal/testutil"
	"github.com/agbruneau/FibGo/internal/ui"
)

// The presentation half of the former apperrors.HandleCalculationError, which
// moved here with audit API-04. The wording assertions came with it; the color
// ones are new, because color is now taken from the ui theme rather than from
// an injected ColorProvider.

func TestWriteCalculationStatus(t *testing.T) {
	// Not parallel: ui.InitTheme mutates the process-global theme.
	testutil.Unsetenv(t, "NO_COLOR")
	original := ui.GetCurrentTheme()
	t.Cleanup(func() { ui.InitTheme(false); _ = original })
	ui.InitTheme(true) // no-color theme: assert on wording, not escapes

	tests := []struct {
		name        string
		err         error
		duration    time.Duration
		wantCode    int
		contains    []string
		notContains []string
	}{
		{
			name:     "nil error writes nothing and succeeds",
			err:      nil,
			wantCode: apperrors.ExitSuccess,
		},
		{
			name:     "timeout",
			err:      context.DeadlineExceeded,
			duration: time.Second,
			wantCode: apperrors.ExitErrorTimeout,
			contains: []string{"Status: Failure (Timeout). The execution limit was reached after 1s."},
		},
		{
			name:     "cancellation",
			err:      context.Canceled,
			duration: 500 * time.Millisecond,
			wantCode: apperrors.ExitErrorCanceled,
			contains: []string{"Status: Canceled after 500ms."},
		},
		{
			name:     "generic",
			err:      fmt.Errorf("random error"),
			wantCode: apperrors.ExitErrorGeneric,
			contains: []string{"Status: Failure. An unexpected error occurred: random error"},
		},
		{
			name: "diagnostic is appended when the error carries one",
			err: apperrors.WrapCalculationError(fmt.Errorf("compute failed"), apperrors.CalculationContext{
				N:                   42,
				MemoryEstimateBytes: 1024,
			}),
			wantCode: apperrors.ExitErrorGeneric,
			contains: []string{
				"Status: Failure. An unexpected error occurred: compute failed",
				"Diagnostic: n=42;",
			},
		},
		{
			name:        "an empty context emits no Diagnostic line",
			err:         apperrors.WrapCalculationError(fmt.Errorf("boom"), apperrors.CalculationContext{}),
			wantCode:    apperrors.ExitErrorGeneric,
			contains:    []string{"boom"},
			notContains: []string{"Diagnostic:"},
		},
		{
			name:        "zero duration omits the suffix",
			err:         context.DeadlineExceeded,
			duration:    0,
			wantCode:    apperrors.ExitErrorTimeout,
			contains:    []string{"The execution limit was reached.\n"},
			notContains: []string{" after "},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			code := WriteCalculationStatus(&out, tt.err, tt.duration)

			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d", code, tt.wantCode)
			}
			if tt.err == nil && out.Len() != 0 {
				t.Errorf("nil error wrote %q, want nothing", out.String())
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

// Color comes from the active ui theme, with no provider in between. This is
// what makes the former "pick the no-color provider under --machine" branch
// unnecessary: app.Run passes --machine to ui.InitTheme, which is the same
// switch.
func TestWriteCalculationStatus_ColorFollowsTheme(t *testing.T) {
	testutil.Unsetenv(t, "NO_COLOR")
	t.Cleanup(func() { ui.InitTheme(false) })

	ui.InitTheme(false) // colors on
	var colored bytes.Buffer
	WriteCalculationStatus(&colored, context.Canceled, time.Second)

	ui.InitTheme(true) // colors off, as under --machine or --quiet
	var plain bytes.Buffer
	WriteCalculationStatus(&plain, context.Canceled, time.Second)

	if !strings.Contains(colored.String(), "\x1b[") {
		t.Errorf("themed output carries no escape codes: %q", colored.String())
	}
	if strings.Contains(plain.String(), "\x1b[") {
		t.Errorf("no-color theme still emitted escape codes: %q", plain.String())
	}
	if testutil.StripAnsiCodes(colored.String()) != plain.String() {
		t.Errorf("wording differs between themes:\n colored=%q\n plain=%q",
			testutil.StripAnsiCodes(colored.String()), plain.String())
	}
}
