package apperrors

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/agbru/fibcalc/internal/format"
)

// Application exit codes define the standard exit statuses for the application.
// These codes are used to signal the outcome of the program execution to the OS.
const (
	ExitSuccess       = 0   // Indicates successful execution.
	ExitErrorGeneric  = 1   // Indicates a generic error.
	ExitErrorTimeout  = 2   // Indicates the operation timed out.
	ExitErrorMismatch = 3   // Indicates a result mismatch between algorithms.
	ExitErrorConfig   = 4   // Indicates a configuration error.
	ExitErrorCanceled = 130 // Indicates the operation was canceled (e.g., SIGINT).
)

// ConfigError represents a user configuration error, such as invalid flags or
// values. It indicates that the application cannot proceed due to incorrect user input.
type ConfigError struct {
	// Message explains the specific configuration error.
	Message string
}

// Error returns the error message for a ConfigError.
//
// Returns:
//   - string: The error message string.
func (e ConfigError) Error() string { return e.Message }

// NewConfigError creates a new ConfigError with a formatted message.
// It allows for the creation of configuration-specific errors with dynamic
// content.
//
// Parameters:
//   - format: A format string (see fmt.Sprintf).
//   - a: Arguments to be formatted into the string.
//
// Returns:
//   - error: A new ConfigError instance containing the formatted message.
func NewConfigError(format string, a ...any) error {
	return ConfigError{Message: fmt.Sprintf(format, a...)}
}

// CalculationContext holds optional diagnostic metadata for a failed calculation.
// Zero values in a field mean “omit” except thresholds where 0 means “auto” when
// the context is considered present (see [CalculationError.HasDiagnostic]).
type CalculationContext struct {
	// N is the Fibonacci index for the failed run.
	N uint64
	// ParallelThresholdBits is the parallelism threshold in bits (0 = auto).
	ParallelThresholdBits int
	// FFTThresholdBits is the FFT multiplication threshold in bits (0 = auto).
	FFTThresholdBits int
	// StrassenThresholdBits is the Strassen matrix threshold in bits (0 = auto).
	StrassenThresholdBits int
	// MemoryEstimateBytes is an approximate memory budget for F(N) (best-effort).
	MemoryEstimateBytes uint64
	// ConfigExcerpt is a single-line, non-sensitive summary (no secrets or tokens).
	ConfigExcerpt string
}

// CalculationError encapsulates a calculation error while preserving the
// original cause. This allows for structured error handling and inspection
// of what went wrong during the Fibonacci calculation.
type CalculationError struct {
	// Cause is the underlying error that triggered this calculation error.
	Cause error
	CalculationContext
}

// Error returns the error message from the underlying cause.
//
// Returns:
//   - string: The error message string from the wrapped error.
func (e CalculationError) Error() string {
	if e.Cause == nil {
		return "calculation error"
	}
	return e.Cause.Error()
}

// Unwrap returns the original wrapped error, allowing for error chain
// inspection (e.g., using errors.Is or errors.As).
//
// Returns:
//   - error: The underlying cause of the CalculationError.
func (e CalculationError) Unwrap() error { return e.Cause }

// HasDiagnostic reports whether optional diagnostic fields should be shown.
func (e CalculationError) HasDiagnostic() bool {
	if e.ConfigExcerpt != "" || e.MemoryEstimateBytes != 0 {
		return true
	}
	if e.N != 0 {
		return true
	}
	return e.ParallelThresholdBits != 0 || e.FFTThresholdBits != 0 || e.StrassenThresholdBits != 0
}

func formatThresholdBits(v int) string {
	if v == 0 {
		return "auto"
	}
	return strconv.Itoa(v)
}

func sanitizeConfigExcerpt(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	const max = 200
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}

// FormatDiagnostic returns a single-line, machine-readable diagnostic summary
// (no ANSI). Empty if [CalculationError.HasDiagnostic] is false.
func (e CalculationError) FormatDiagnostic() string {
	if !e.HasDiagnostic() {
		return ""
	}
	var b strings.Builder
	if e.N != 0 {
		fmt.Fprintf(&b, "n=%d", e.N)
	}
	if b.Len() > 0 {
		b.WriteString("; ")
	}
	fmt.Fprintf(&b, "parallel_bits=%s; fft_bits=%s; strassen_bits=%s",
		formatThresholdBits(e.ParallelThresholdBits),
		formatThresholdBits(e.FFTThresholdBits),
		formatThresholdBits(e.StrassenThresholdBits))
	if e.MemoryEstimateBytes != 0 {
		fmt.Fprintf(&b, "; mem_est=%s", format.FormatBytes(e.MemoryEstimateBytes))
	}
	if ex := sanitizeConfigExcerpt(e.ConfigExcerpt); ex != "" {
		fmt.Fprintf(&b, "; config=%s", ex)
	}
	return b.String()
}

// WrapCalculationError attaches a [CalculationContext] to a calculation failure.
// It returns nil if cause is nil.
func WrapCalculationError(cause error, ctx CalculationContext) error {
	if cause == nil {
		return nil
	}
	return CalculationError{Cause: cause, CalculationContext: ctx}
}

// TimeoutError represents a calculation timeout. It captures the operation
// name and the duration limit that was exceeded.
type TimeoutError struct {
	// Operation is the name of the operation that timed out.
	Operation string
	// Limit is the duration after which the operation was considered timed out.
	Limit time.Duration
}

// Error returns a formatted message describing the timeout.
//
// Returns:
//   - string: The error message string.
func (e TimeoutError) Error() string {
	return fmt.Sprintf("operation %q timed out after %s", e.Operation, e.Limit)
}

// ValidationError represents an input validation failure. It identifies which
// field failed validation and provides a human-readable explanation.
type ValidationError struct {
	// Field is the name of the field that failed validation.
	Field string
	// Message explains the validation failure.
	Message string
}

// Error returns a formatted message describing the validation failure.
//
// Returns:
//   - string: The error message string.
func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error for %q: %s", e.Field, e.Message)
}

// MemoryError represents a memory limit exceeded condition. It captures the
// requested, available, and limit memory values for diagnostic purposes.
type MemoryError struct {
	// Requested is the number of bytes the operation needed.
	Requested uint64
	// Available is the number of bytes currently available.
	Available uint64
	// Limit is the configured memory limit in bytes.
	Limit uint64
}

// Error returns a formatted message describing the memory error.
//
// Returns:
//   - string: The error message string.
func (e MemoryError) Error() string {
	return fmt.Sprintf("memory error: requested %d bytes, available %d bytes (limit: %d)", e.Requested, e.Available, e.Limit)
}

// WrapError wraps an error with additional context using fmt.Errorf and %w.
// This allows the wrapped error to be unwrapped with errors.Unwrap() and
// checked with errors.Is() and errors.As().
//
// Parameters:
//   - err: The error to wrap.
//   - format: A format string for the context message.
//   - args: Arguments for the format string.
//
// Returns:
//   - error: The wrapped error, or nil if err is nil.
func WrapError(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	message := fmt.Sprintf(format, args...)
	return fmt.Errorf("%s: %w", message, err)
}

// IsContextError checks if the error is a context cancellation or deadline exceeded error.
//
// Parameters:
//   - err: The error to check.
//
// Returns:
//   - bool: true if the error is a context error.
func IsContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

