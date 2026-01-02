// Package apperrors provides tests for application error types.
package apperrors

import (
	"context"
	"errors"
	"testing"
)

func TestConfigError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		err         error
		expected    string
		checkTypeAs bool
	}{
		{
			name:     "Error returns message",
			err:      ConfigError{Message: "invalid flag value"},
			expected: "invalid flag value",
		},
		{
			name:     "NewConfigError creates formatted error",
			err:      NewConfigError("invalid value %d for flag %s", 42, "--threshold"),
			expected: "invalid value 42 for flag --threshold",
		},
		{
			name:        "ConfigError type assertion",
			err:         NewConfigError("test error"),
			expected:    "test error",
			checkTypeAs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.err.Error() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.err.Error())
			}
			if tt.checkTypeAs {
				var configErr ConfigError
				if !errors.As(tt.err, &configErr) {
					t.Error("expected error to be ConfigError type")
				}
			}
		})
	}
}

func TestCalculationError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		cause       error
		expectedMsg string
		checkIs     error
		checkUnwrap bool
	}{
		{
			name:        "Error returns cause message",
			cause:       errors.New("division by zero"),
			expectedMsg: "division by zero",
		},
		{
			name:        "Unwrap returns cause",
			cause:       errors.New("original error"),
			expectedMsg: "original error",
			checkUnwrap: true,
		},
		{
			name:        "errors.Is works with wrapped error",
			cause:       context.Canceled,
			expectedMsg: "context canceled",
			checkIs:     context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := CalculationError{Cause: tt.cause}

			if err.Error() != tt.expectedMsg {
				t.Errorf("expected %q, got %q", tt.expectedMsg, err.Error())
			}

			if tt.checkUnwrap && err.Unwrap() != tt.cause {
				t.Error("Unwrap should return the original cause")
			}

			if tt.checkIs != nil && !errors.Is(err, tt.checkIs) {
				t.Errorf("errors.Is should find %v in the chain", tt.checkIs)
			}
		})
	}
}

func TestServerError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		message     string
		cause       error
		expectedMsg string
		checkUnwrap bool
		expectNil   bool
		checkTypeAs bool
	}{
		{
			name:        "Error with cause",
			message:     "failed to start",
			cause:       errors.New("connection refused"),
			expectedMsg: "failed to start: connection refused",
		},
		{
			name:        "Error without cause",
			message:     "server stopped",
			cause:       nil,
			expectedMsg: "server stopped",
		},
		{
			name:        "Unwrap returns cause",
			message:     "test",
			cause:       errors.New("network error"),
			expectedMsg: "test: network error",
			checkUnwrap: true,
		},
		{
			name:        "Unwrap returns nil when no cause",
			message:     "test",
			cause:       nil,
			expectedMsg: "test",
			checkUnwrap: true,
			expectNil:   true,
		},
		{
			name:        "NewServerError creates error",
			message:     "cannot listen on port 8080",
			cause:       errors.New("bind failed"),
			expectedMsg: "cannot listen on port 8080: bind failed",
			checkTypeAs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var err error
			if tt.checkTypeAs {
				err = NewServerError(tt.message, tt.cause)
			} else {
				err = ServerError{Message: tt.message, Cause: tt.cause}
			}

			if err.Error() != tt.expectedMsg {
				t.Errorf("expected %q, got %q", tt.expectedMsg, err.Error())
			}

			if tt.checkTypeAs {
				var serverErr ServerError
				if !errors.As(err, &serverErr) {
					t.Error("expected error to be ServerError type")
				}
				if serverErr.Message != tt.message {
					t.Errorf("unexpected message: %s", serverErr.Message)
				}
			}

			if tt.checkUnwrap {
				serverErr := err.(ServerError)
				unwrapped := serverErr.Unwrap()
				if tt.expectNil && unwrapped != nil {
					t.Error("Unwrap should return nil when there's no cause")
				}
				if !tt.expectNil && unwrapped == nil {
					t.Error("Unwrap should return the cause")
				}
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		field       string
		message     string
		value       any
		expectedMsg string
		checkTypeAs bool
	}{
		{
			name:        "Error with field",
			field:       "n",
			message:     "must be positive",
			expectedMsg: "validation error for 'n': must be positive",
		},
		{
			name:        "Error without field",
			field:       "",
			message:     "invalid input",
			expectedMsg: "validation error: invalid input",
		},
		{
			name:        "NewValidationError creates error",
			field:       "algo",
			message:     "unknown algorithm",
			value:       "invalid",
			expectedMsg: "validation error for 'algo': unknown algorithm",
			checkTypeAs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var err error
			if tt.checkTypeAs {
				err = NewValidationError(tt.field, tt.message, tt.value)
			} else {
				err = ValidationError{Field: tt.field, Message: tt.message, Value: tt.value}
			}

			if err.Error() != tt.expectedMsg {
				t.Errorf("expected %q, got %q", tt.expectedMsg, err.Error())
			}

			if tt.checkTypeAs {
				var valErr ValidationError
				if !errors.As(err, &valErr) {
					t.Error("expected error to be ValidationError type")
				}
				if valErr.Field != tt.field || valErr.Value != tt.value {
					t.Errorf("unexpected validation error: %+v", valErr)
				}
			}
		})
	}
}

func TestWrapError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		original    error
		format      string
		args        []any
		expectedMsg string
		expectNil   bool
		checkIs     error
	}{
		{
			name:        "wraps error with context",
			original:    errors.New("file not found"),
			format:      "failed to load config",
			expectedMsg: "failed to load config: file not found",
		},
		{
			name:        "preserves error chain",
			original:    context.DeadlineExceeded,
			format:      "operation timed out",
			expectedMsg: "operation timed out: context deadline exceeded",
			checkIs:     context.DeadlineExceeded,
		},
		{
			name:      "returns nil for nil error",
			original:  nil,
			format:    "some context",
			expectNil: true,
		},
		{
			name:        "supports format arguments",
			original:    errors.New("connection reset"),
			format:      "failed to connect to %s:%d",
			args:        []any{"localhost", 8080},
			expectedMsg: "failed to connect to localhost:8080: connection reset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			wrapped := WrapError(tt.original, tt.format, tt.args...)

			if tt.expectNil {
				if wrapped != nil {
					t.Error("WrapError(nil, ...) should return nil")
				}
				return
			}

			if wrapped == nil {
				t.Fatal("wrapped error should not be nil")
			}

			if wrapped.Error() != tt.expectedMsg {
				t.Errorf("expected %q, got %q", tt.expectedMsg, wrapped.Error())
			}

			if tt.checkIs != nil && !errors.Is(wrapped, tt.checkIs) {
				t.Errorf("wrapped error should preserve %v in the chain", tt.checkIs)
			}
		})
	}
}

func TestIsContextError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"context.Canceled", context.Canceled, true},
		{"context.DeadlineExceeded", context.DeadlineExceeded, true},
		{"wrapped context.Canceled", WrapError(context.Canceled, "operation canceled"), true},
		{"regular error", errors.New("some error"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := IsContextError(tt.err)
			if result != tt.expected {
				t.Errorf("IsContextError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestExitCodes(t *testing.T) {
	t.Parallel()
	// Verify exit codes are distinct and match expected values
	codes := map[string]int{
		"ExitSuccess":       ExitSuccess,
		"ExitErrorGeneric":  ExitErrorGeneric,
		"ExitErrorTimeout":  ExitErrorTimeout,
		"ExitErrorMismatch": ExitErrorMismatch,
		"ExitErrorConfig":   ExitErrorConfig,
		"ExitErrorCanceled": ExitErrorCanceled,
	}

	// Check expected values
	if ExitSuccess != 0 {
		t.Errorf("ExitSuccess should be 0, got %d", ExitSuccess)
	}
	if ExitErrorCanceled != 130 {
		t.Errorf("ExitErrorCanceled should be 130 (SIGINT convention), got %d", ExitErrorCanceled)
	}

	// Check all codes are unique
	seen := make(map[int]string)
	for name, code := range codes {
		if existing, ok := seen[code]; ok {
			t.Errorf("duplicate exit code %d: %s and %s", code, existing, name)
		}
		seen[code] = name
	}
}
