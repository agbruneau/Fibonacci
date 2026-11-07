package apperrors

import (
	"context"
	"errors"
)

// ExitCodeFor maps a calculation error to the POSIX exit code the binary should
// return: ExitSuccess for nil, ExitErrorTimeout for a deadline, ExitErrorCanceled
// for a cancellation (SIGINT included), ExitErrorGeneric for anything else.
//
// It is a pure function of the error, which is the whole point of it existing
// (audit API-04 / ARC-02). What stood here was HandleCalculationError: it took
// an io.Writer and a ColorProvider, wrote colored "Status: …" lines, and
// returned the exit code as a side effect. That put terminal presentation in a
// leaf package the whole tree depends on, and it forced this package to define
// a ColorProvider interface whose own comment said it existed "to break the
// import cycle with cli" — an interface invented to work around a layering
// mistake rather than to express a behavior.
//
// The cost was visible at the call sites. internal/tui wanted only the number,
// so it called HandleCalculationError(err, d, io.Discard, nil) and threw the
// formatted text away. The CLI wanted the text, and had to pass a provider that
// duplicated what ui.Color* already returns. Presentation now lives in
// internal/cli (WriteCalculationStatus); this package answers the one question
// that is genuinely its own.
func ExitCodeFor(err error) int {
	switch {
	case err == nil:
		return ExitSuccess
	case errors.Is(err, context.DeadlineExceeded):
		return ExitErrorTimeout
	case errors.Is(err, context.Canceled):
		return ExitErrorCanceled
	default:
		return ExitErrorGeneric
	}
}

// CalculationDiagnostic returns the diagnostic text carried by a
// CalculationError, or the empty string when the error chain has none.
//
// The formatting of that text belongs to CalculationError itself; this is only
// the lookup, so a presenter does not have to know the concrete type.
func CalculationDiagnostic(err error) string {
	var ce CalculationError
	if !errors.As(err, &ce) || !ce.HasDiagnostic() {
		return ""
	}
	return ce.FormatDiagnostic()
}
