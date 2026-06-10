package app

import (
	"testing"

	apperrors "github.com/agbruneau/FibGo/internal/errors"
)

// TestExitActionCode verifies the mapping between ExitAction values and the
// POSIX exit codes defined in internal/errors, including the defensive
// fallback for out-of-range values.
func TestExitActionCode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		action ExitAction
		want   int
	}{
		{"Success", ActionSuccess, apperrors.ExitSuccess},
		{"VersionHandled maps to success code", ActionVersionHandled, apperrors.ExitSuccess},
		{"Error", ActionError, apperrors.ExitErrorGeneric},
		{"Timeout", ActionTimeout, apperrors.ExitErrorTimeout},
		{"Mismatch", ActionMismatch, apperrors.ExitErrorMismatch},
		{"Config", ActionConfig, apperrors.ExitErrorConfig},
		{"Canceled", ActionCanceled, apperrors.ExitErrorCanceled},
		{"Unknown action falls back to generic error", ExitAction(999), apperrors.ExitErrorGeneric},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.action.Code(); got != tc.want {
				t.Errorf("ExitAction(%d).Code() = %d, want %d", tc.action, got, tc.want)
			}
		})
	}
}

// TestExitActionShouldExit pins the sentinel semantics that replaced the
// old exitVersion = -1: only ActionVersionHandled must suppress os.Exit.
func TestExitActionShouldExit(t *testing.T) {
	t.Parallel()
	exiting := []ExitAction{
		ActionSuccess, ActionError, ActionTimeout,
		ActionMismatch, ActionConfig, ActionCanceled,
	}
	for _, a := range exiting {
		if !a.ShouldExit() {
			t.Errorf("ExitAction(%d).ShouldExit() = false, want true", a)
		}
	}
	if ActionVersionHandled.ShouldExit() {
		t.Error("ActionVersionHandled.ShouldExit() = true, want false (main must not call os.Exit)")
	}
}

// TestExitActionFromCode covers every POSIX code produced by the internal
// helpers, plus the fallback for unexpected codes.
func TestExitActionFromCode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		code int
		want ExitAction
	}{
		{"Success", apperrors.ExitSuccess, ActionSuccess},
		{"Timeout", apperrors.ExitErrorTimeout, ActionTimeout},
		{"Mismatch", apperrors.ExitErrorMismatch, ActionMismatch},
		{"Config", apperrors.ExitErrorConfig, ActionConfig},
		{"Canceled", apperrors.ExitErrorCanceled, ActionCanceled},
		{"Generic", apperrors.ExitErrorGeneric, ActionError},
		{"Unknown code falls back to ActionError", 42, ActionError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := exitActionFromCode(tc.code); got != tc.want {
				t.Errorf("exitActionFromCode(%d) = %d, want %d", tc.code, got, tc.want)
			}
		})
	}
}

// TestExitActionRoundTrip pins the documented 1:1 mapping invariant:
// converting an action to its POSIX code and back must be the identity.
// ActionVersionHandled is excluded by design (its code 0 is shared with
// ActionSuccess).
func TestExitActionRoundTrip(t *testing.T) {
	t.Parallel()
	actions := []ExitAction{
		ActionSuccess, ActionError, ActionTimeout,
		ActionMismatch, ActionConfig, ActionCanceled,
	}
	for _, a := range actions {
		if got := exitActionFromCode(a.Code()); got != a {
			t.Errorf("exitActionFromCode(ExitAction(%d).Code()) = %d, want identity", a, got)
		}
	}
}
