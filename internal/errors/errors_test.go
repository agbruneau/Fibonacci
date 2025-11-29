// Package apperrors provides tests for application error types.
package apperrors

import (
	"context"
	"errors"
	"testing"
)

func TestConfigError(t *testing.T) {
	t.Run("Error returns message", func(t *testing.T) {
		err := ConfigError{Message: "invalid flag value"}
		if err.Error() != "invalid flag value" {
			t.Errorf("expected 'invalid flag value', got %q", err.Error())
		}
	})

	t.Run("NewConfigError creates formatted error", func(t *testing.T) {
		err := NewConfigError("invalid value %d for flag %s", 42, "--threshold")
		expected := "invalid value 42 for flag --threshold"
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("ConfigError type assertion", func(t *testing.T) {
		err := NewConfigError("test error")
		var configErr ConfigError
		if !errors.As(err, &configErr) {
			t.Error("expected error to be ConfigError type")
		}
	})
}

func TestCalculationError(t *testing.T) {
	t.Run("Error returns cause message", func(t *testing.T) {
		cause := errors.New("division by zero")
		err := CalculationError{Cause: cause}
		if err.Error() != "division by zero" {
			t.Errorf("expected 'division by zero', got %q", err.Error())
		}
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		cause := errors.New("original error")
		err := CalculationError{Cause: cause}
		if err.Unwrap() != cause {
			t.Error("Unwrap should return the original cause")
		}
	})

	t.Run("errors.Is works with wrapped error", func(t *testing.T) {
		cause := context.Canceled
		err := CalculationError{Cause: cause}
		if !errors.Is(err, context.Canceled) {
			t.Error("errors.Is should find context.Canceled in the chain")
		}
	})
}

func TestServerError(t *testing.T) {
	t.Run("Error with cause", func(t *testing.T) {
		cause := errors.New("connection refused")
		err := ServerError{Message: "failed to start", Cause: cause}
		expected := "failed to start: connection refused"
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("Error without cause", func(t *testing.T) {
		err := ServerError{Message: "server stopped"}
		if err.Error() != "server stopped" {
			t.Errorf("expected 'server stopped', got %q", err.Error())
		}
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		cause := errors.New("network error")
		err := ServerError{Message: "test", Cause: cause}
		if err.Unwrap() != cause {
			t.Error("Unwrap should return the cause")
		}
	})

	t.Run("Unwrap returns nil when no cause", func(t *testing.T) {
		err := ServerError{Message: "test"}
		if err.Unwrap() != nil {
			t.Error("Unwrap should return nil when there's no cause")
		}
	})

	t.Run("NewServerError creates error", func(t *testing.T) {
		cause := errors.New("bind failed")
		err := NewServerError("cannot listen on port 8080", cause)
		var serverErr ServerError
		if !errors.As(err, &serverErr) {
			t.Error("expected error to be ServerError type")
		}
		if serverErr.Message != "cannot listen on port 8080" {
			t.Errorf("unexpected message: %s", serverErr.Message)
		}
	})
}

func TestValidationError(t *testing.T) {
	t.Run("Error with field", func(t *testing.T) {
		err := ValidationError{Field: "n", Message: "must be positive"}
		expected := "validation error for 'n': must be positive"
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("Error without field", func(t *testing.T) {
		err := ValidationError{Message: "invalid input"}
		expected := "validation error: invalid input"
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("NewValidationError creates error", func(t *testing.T) {
		err := NewValidationError("algo", "unknown algorithm", "invalid")
		var valErr ValidationError
		if !errors.As(err, &valErr) {
			t.Error("expected error to be ValidationError type")
		}
		if valErr.Field != "algo" || valErr.Value != "invalid" {
			t.Errorf("unexpected validation error: %+v", valErr)
		}
	})
}

func TestWrapError(t *testing.T) {
	t.Run("wraps error with context", func(t *testing.T) {
		original := errors.New("file not found")
		wrapped := WrapError(original, "failed to load config")
		if wrapped == nil {
			t.Fatal("wrapped error should not be nil")
		}
		expected := "failed to load config: file not found"
		if wrapped.Error() != expected {
			t.Errorf("expected %q, got %q", expected, wrapped.Error())
		}
	})

	t.Run("preserves error chain", func(t *testing.T) {
		original := context.DeadlineExceeded
		wrapped := WrapError(original, "operation timed out")
		if !errors.Is(wrapped, context.DeadlineExceeded) {
			t.Error("wrapped error should preserve the original in the chain")
		}
	})

	t.Run("returns nil for nil error", func(t *testing.T) {
		wrapped := WrapError(nil, "some context")
		if wrapped != nil {
			t.Error("WrapError(nil, ...) should return nil")
		}
	})

	t.Run("supports format arguments", func(t *testing.T) {
		original := errors.New("connection reset")
		wrapped := WrapError(original, "failed to connect to %s:%d", "localhost", 8080)
		expected := "failed to connect to localhost:8080: connection reset"
		if wrapped.Error() != expected {
			t.Errorf("expected %q, got %q", expected, wrapped.Error())
		}
	})
}

func TestIsContextError(t *testing.T) {
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
			result := IsContextError(tt.err)
			if result != tt.expected {
				t.Errorf("IsContextError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestExitCodes(t *testing.T) {
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
