// Package apperrors defines structured application error types,
// allowing for a clear distinction between error classes (configuration,
// calculation, etc.) and for carrying the underlying cause.
//
// Error Wrapping Guidelines:
// This package follows Go's error wrapping conventions using fmt.Errorf with %w.
// All error types implement the Unwrap() method to support errors.Is() and errors.As().
package apperrors

import (
	"context"
	"errors"
	"fmt"
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

// CalculationError encapsulates a calculation error while preserving the
// original cause. This allows for structured error handling and inspection
// of what went wrong during the Fibonacci calculation.
type CalculationError struct {
	// Cause is the underlying error that triggered this calculation error.
	Cause error
}

// Error returns the error message from the underlying cause.
//
// Returns:
//   - string: The error message string from the wrapped error.
func (e CalculationError) Error() string { return e.Cause.Error() }

// Unwrap returns the original wrapped error, allowing for error chain
// inspection (e.g., using errors.Is or errors.As).
//
// Returns:
//   - error: The underlying cause of the CalculationError.
func (e CalculationError) Unwrap() error { return e.Cause }

// ServerError represents errors that occur in the HTTP server component.
// It wraps an underlying error with additional context specific to the server operation.
type ServerError struct {
	// Message is a descriptive message about the server error.
	Message string
	// Cause is the underlying error, if any.
	Cause error
}

// Error returns the error message for a ServerError.
// It combines the descriptive message and the underlying cause if present.
//
// Returns:
//   - string: The complete error message.
func (e ServerError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error.
//
// Returns:
//   - error: The cause of the ServerError, or nil if there is none.
func (e ServerError) Unwrap() error { return e.Cause }

// NewServerError creates a new ServerError with a message and optional cause.
//
// Parameters:
//   - message: A description of the error context.
//   - cause: The underlying error that occurred (can be nil).
//
// Returns:
//   - error: A new ServerError instance.
func NewServerError(message string, cause error) error {
	return ServerError{Message: message, Cause: cause}
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

// ValidationError represents an error due to invalid input validation.
// It is used for API request validation and configuration validation.
type ValidationError struct {
	// Field is the name of the field that failed validation.
	Field string
	// Message describes why validation failed.
	Message string
	// Value is the invalid value (optional, may be nil).
	Value any
}

// Error returns the error message for a ValidationError.
func (e ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error for '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

// NewValidationError creates a new ValidationError.
//
// Parameters:
//   - field: The name of the field that failed validation.
//   - message: A description of why validation failed.
//   - value: The invalid value (optional).
//
// Returns:
//   - error: A new ValidationError instance.
func NewValidationError(field, message string, value any) error {
	return ValidationError{Field: field, Message: message, Value: value}
}
