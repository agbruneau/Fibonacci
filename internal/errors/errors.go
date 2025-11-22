// Package apperrors defines structured application error types,
// allowing for a clear distinction between error classes (configuration,
// calculation, etc.) and for carrying the underlying cause.
package apperrors

import "fmt"

// Application exit codes define the standard exit statuses for the application.
const (
	ExitSuccess       = 0
	ExitErrorGeneric  = 1
	ExitErrorTimeout  = 2
	ExitErrorMismatch = 3
	ExitErrorConfig   = 4
	ExitErrorCanceled = 130
)

// ConfigError represents a user configuration error, such as invalid flags or
// values.
type ConfigError struct {
	Message string
}

// Error returns the error message for a ConfigError.
func (e ConfigError) Error() string { return e.Message }

// NewConfigError creates a new ConfigError with a formatted message.
// It allows for the creation of configuration-specific errors with dynamic
// content.
//
// The format string and arguments are the same as for fmt.Sprintf.
//
// It returns an error of type ConfigError.
func NewConfigError(format string, a ...interface{}) error {
	return ConfigError{Message: fmt.Sprintf(format, a...)}
}

// CalculationError encapsulates a calculation error while preserving the
// original cause. This allows for structured error handling and inspection.
type CalculationError struct {
	Cause error
}

// Error returns the error message from the underlying cause.
func (e CalculationError) Error() string { return e.Cause.Error() }

// Unwrap returns the original wrapped error, allowing for error chain
// inspection.
func (e CalculationError) Unwrap() error { return e.Cause }

// ServerError represents errors that occur in the HTTP server.
type ServerError struct {
	Message string
	Cause   error
}

// Error returns the error message for a ServerError.
func (e ServerError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error.
func (e ServerError) Unwrap() error { return e.Cause }

// NewServerError creates a new ServerError with a message and optional cause.
func NewServerError(message string, cause error) error {
	return ServerError{Message: message, Cause: cause}
}
