package app

import apperrors "github.com/agbru/fibcalc/internal/errors"

// ExitAction enumerates the high-level outcomes of running the application.
//
// It replaces the fragile sentinel `exitVersion = -1` previously used in
// cmd/fibcalc/main.go to mean "do not call os.Exit". Callers (typically
// main) use ShouldExit to decide whether to invoke os.Exit, and Code to
// obtain the POSIX exit status.
//
// Action* values map 1:1 to the POSIX exit codes defined in
// internal/errors so that the observed exit status of the binary is
// unchanged by the refactor.
type ExitAction int

const (
	// ActionSuccess indicates a successful run; POSIX code 0.
	ActionSuccess ExitAction = iota
	// ActionError indicates a generic failure; POSIX code 1.
	ActionError
	// ActionVersionHandled indicates that --version was handled and the
	// caller MUST NOT call os.Exit (preserves the original "-1" sentinel
	// semantics: main simply returns, and the process exits with the
	// runtime default of 0).
	ActionVersionHandled
	// ActionTimeout indicates the operation exceeded its deadline; POSIX
	// code 2.
	ActionTimeout
	// ActionMismatch indicates that the parallel-algorithm comparison
	// produced divergent results; POSIX code 3.
	ActionMismatch
	// ActionConfig indicates a user configuration error; POSIX code 4.
	ActionConfig
	// ActionCanceled indicates the operation was canceled by signal
	// (e.g. SIGINT); POSIX code 130.
	ActionCanceled
)

// Code returns the POSIX exit code corresponding to the action.
//
// ActionVersionHandled returns 0, which is the natural process exit
// status when main returns without calling os.Exit; this matches the
// observed pre-refactor behaviour.
func (a ExitAction) Code() int {
	switch a {
	case ActionSuccess, ActionVersionHandled:
		return apperrors.ExitSuccess
	case ActionError:
		return apperrors.ExitErrorGeneric
	case ActionTimeout:
		return apperrors.ExitErrorTimeout
	case ActionMismatch:
		return apperrors.ExitErrorMismatch
	case ActionConfig:
		return apperrors.ExitErrorConfig
	case ActionCanceled:
		return apperrors.ExitErrorCanceled
	default:
		return apperrors.ExitErrorGeneric
	}
}

// ShouldExit reports whether main should invoke os.Exit with Code.
// It returns false only for ActionVersionHandled, preserving the original
// sentinel behaviour where the version path simply returned from main.
func (a ExitAction) ShouldExit() bool {
	return a != ActionVersionHandled
}

// exitActionFromCode converts a POSIX exit code produced by an internal
// helper into the matching ExitAction so that the typed API exposed by
// Application.Run is consistent with the observed exit status.
func exitActionFromCode(code int) ExitAction {
	switch code {
	case apperrors.ExitSuccess:
		return ActionSuccess
	case apperrors.ExitErrorTimeout:
		return ActionTimeout
	case apperrors.ExitErrorMismatch:
		return ActionMismatch
	case apperrors.ExitErrorConfig:
		return ActionConfig
	case apperrors.ExitErrorCanceled:
		return ActionCanceled
	default:
		return ActionError
	}
}
