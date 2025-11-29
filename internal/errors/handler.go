package apperrors

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"
)

// ErrorMessageProvider defines the interface for obtaining localized error messages.
// This abstraction breaks the import cycle with i18n.
type ErrorMessageProvider interface {
	GetMessage(key string) string
}

// ColorProvider defines the interface for obtaining terminal color codes.
// This abstraction breaks the import cycle with cli.
type ColorProvider interface {
	Yellow() string
	Reset() string
}

// DefaultColorProvider provides no color codes (for non-terminal output).
type DefaultColorProvider struct{}

func (d DefaultColorProvider) Yellow() string { return "" }
func (d DefaultColorProvider) Reset() string  { return "" }

// DefaultMessageProvider returns the message key itself as a fallback.
type DefaultMessageProvider struct{}

func (d DefaultMessageProvider) GetMessage(key string) string { return key }

// HandleCalculationError formats and prints error messages related to failed calculations.
// It distinguishes between different error types (timeout, cancellation, generic)
// to provide the user with specific feedback.
//
// Parameters:
//   - err: The error that occurred.
//   - duration: The duration of the calculation before it failed.
//   - out: The io.Writer to which the error message will be written.
//   - colors: Provider for terminal color codes (can be nil for no colors).
//   - messages: Provider for localized messages (can be nil for default keys).
//
// Returns:
//   - int: The appropriate exit code for the error type.
func HandleCalculationError(err error, duration time.Duration, out io.Writer, colors ColorProvider, messages ErrorMessageProvider) int {
	if err == nil {
		return ExitSuccess
	}

	// Use defaults if not provided
	if colors == nil {
		colors = DefaultColorProvider{}
	}
	if messages == nil {
		messages = DefaultMessageProvider{}
	}

	msgSuffix := ""
	if duration > 0 {
		msgSuffix = fmt.Sprintf(" after %s%s%s", colors.Yellow(), duration, colors.Reset())
	}

	if errors.Is(err, context.DeadlineExceeded) {
		fmt.Fprintf(out, "%s\n", messages.GetMessage("StatusTimeout"))
		return ExitErrorTimeout
	}
	if errors.Is(err, context.Canceled) {
		fmt.Fprintf(out, "%s%s%s.%s\n", colors.Yellow(), messages.GetMessage("StatusCanceled"), msgSuffix, colors.Reset())
		return ExitErrorCanceled
	}
	fmt.Fprintf(out, "%s\n", messages.GetMessage("StatusFailure"))
	return ExitErrorGeneric
}
