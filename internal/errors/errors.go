// Package apperrors defines structured application error types,
// allowing for a clear distinction between error classes (configuration,
// calculation, etc.) and for carrying the underlying cause.
package apperrors

import "fmt"

// ConfigError represents a user configuration error (flags, invalid values, etc.).
type ConfigError struct {
    Message string
}

func (e ConfigError) Error() string { return e.Message }

func NewConfigError(format string, a ...interface{}) error {
    return ConfigError{Message: fmt.Sprintf(format, a...)}
}

// CalculationError allows for encapsulating a calculation error while preserving
// the original cause (unwrap).
type CalculationError struct {
    Cause error
}

func (e CalculationError) Error() string { return e.Cause.Error() }
func (e CalculationError) Unwrap() error { return e.Cause }
