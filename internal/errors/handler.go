package apperrors

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"
)

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

// HandleCalculationError formats and prints error messages related to failed calculations.
// It distinguishes between different error types (timeout, cancellation, generic)
// to provide the user with specific feedback.
//
// Parameters:
//   - err: The error that occurred.
//   - duration: The duration of the calculation before it failed.
//   - out: The io.Writer to which the error message will be written.
//   - colors: Provider for terminal color codes (can be nil for no colors).
//
// Returns:
//   - int: The appropriate exit code for the error type.
func HandleCalculationError(err error, duration time.Duration, out io.Writer, colors ColorProvider) int {
	if err == nil {
		return ExitSuccess
	}

	// Use defaults if not provided
	if colors == nil {
		colors = DefaultColorProvider{}
	}

	msgSuffix := ""
	if duration > 0 {
		msgSuffix = fmt.Sprintf(" after %s%s%s", colors.Yellow(), duration, colors.Reset())
	}

	if errors.Is(err, context.DeadlineExceeded) {
		fmt.Fprintf(out, "Status: Failure (Timeout). The execution limit was reached%s.\n", msgSuffix)
		return ExitErrorTimeout
	}
	if errors.Is(err, context.Canceled) {
		fmt.Fprintf(out, "%sStatus: Canceled%s.%s\n", colors.Yellow(), msgSuffix, colors.Reset())
		return ExitErrorCanceled
	}
	fmt.Fprintf(out, "Status: Failure. An unexpected error occurred: %v\n", err)
	return ExitErrorGeneric
}
