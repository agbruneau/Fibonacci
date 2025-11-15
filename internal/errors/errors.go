// Package apperrors defines structured application error types,
// allowing for a clear distinction between error classes (configuration,
// calculation, etc.) and for carrying the underlying cause.
package apperrors

import "fmt"

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
