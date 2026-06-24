// Package apperrors provides tests for application error types.
package apperrors

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
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

func TestTimeoutError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		err         TimeoutError
		expected    string
		checkTypeAs bool
	}{
		{
			name:     "Error returns formatted message",
			err:      TimeoutError{Operation: "fibonacci", Limit: 30 * time.Second},
			expected: `operation "fibonacci" timed out after 30s`,
		},
		{
			name:     "Error with subsecond limit",
			err:      TimeoutError{Operation: "matrix multiply", Limit: 500 * time.Millisecond},
			expected: `operation "matrix multiply" timed out after 500ms`,
		},
		{
			name:        "errors.As works with TimeoutError",
			err:         TimeoutError{Operation: "fft", Limit: 10 * time.Second},
			expected:    `operation "fft" timed out after 10s`,
			checkTypeAs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var err error = tt.err
			if err.Error() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, err.Error())
			}
			if tt.checkTypeAs {
				var timeoutErr TimeoutError
				if !errors.As(err, &timeoutErr) {
					t.Error("expected error to be TimeoutError type")
				}
				if timeoutErr.Operation != tt.err.Operation {
					t.Errorf("expected Operation %q, got %q", tt.err.Operation, timeoutErr.Operation)
				}
				if timeoutErr.Limit != tt.err.Limit {
					t.Errorf("expected Limit %v, got %v", tt.err.Limit, timeoutErr.Limit)
				}
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		err         ValidationError
		expected    string
		checkTypeAs bool
	}{
		{
			name:     "Error returns formatted message",
			err:      ValidationError{Field: "n", Message: "must be non-negative"},
			expected: `validation error for "n": must be non-negative`,
		},
		{
			name:     "Error with different field",
			err:      ValidationError{Field: "threshold", Message: "must be greater than zero"},
			expected: `validation error for "threshold": must be greater than zero`,
		},
		{
			name:        "errors.As works with ValidationError",
			err:         ValidationError{Field: "algorithm", Message: "unknown algorithm"},
			expected:    `validation error for "algorithm": unknown algorithm`,
			checkTypeAs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var err error = tt.err
			if err.Error() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, err.Error())
			}
			if tt.checkTypeAs {
				var validationErr ValidationError
				if !errors.As(err, &validationErr) {
					t.Error("expected error to be ValidationError type")
				}
				if validationErr.Field != tt.err.Field {
					t.Errorf("expected Field %q, got %q", tt.err.Field, validationErr.Field)
				}
				if validationErr.Message != tt.err.Message {
					t.Errorf("expected Message %q, got %q", tt.err.Message, validationErr.Message)
				}
			}
		})
	}
}

func TestMemoryError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		err         MemoryError
		expected    string
		checkTypeAs bool
	}{
		{
			name:     "Error returns formatted message",
			err:      MemoryError{Requested: 1073741824, Available: 536870912, Limit: 1073741824},
			expected: "memory error: requested 1073741824 bytes, available 536870912 bytes (limit: 1073741824)",
		},
		{
			name:     "Error with small values",
			err:      MemoryError{Requested: 1024, Available: 512, Limit: 2048},
			expected: "memory error: requested 1024 bytes, available 512 bytes (limit: 2048)",
		},
		{
			name:        "errors.As works with MemoryError",
			err:         MemoryError{Requested: 4096, Available: 2048, Limit: 8192},
			expected:    "memory error: requested 4096 bytes, available 2048 bytes (limit: 8192)",
			checkTypeAs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var err error = tt.err
			if err.Error() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, err.Error())
			}
			if tt.checkTypeAs {
				var memErr MemoryError
				if !errors.As(err, &memErr) {
					t.Error("expected error to be MemoryError type")
				}
				if memErr.Requested != tt.err.Requested {
					t.Errorf("expected Requested %d, got %d", tt.err.Requested, memErr.Requested)
				}
				if memErr.Available != tt.err.Available {
					t.Errorf("expected Available %d, got %d", tt.err.Available, memErr.Available)
				}
				if memErr.Limit != tt.err.Limit {
					t.Errorf("expected Limit %d, got %d", tt.err.Limit, memErr.Limit)
				}
			}
		})
	}
}

func TestNewErrorTypes_ErrorsAsWithWrapping(t *testing.T) {
	t.Parallel()

	t.Run("TimeoutError wrapped in CalculationError", func(t *testing.T) {
		t.Parallel()
		inner := TimeoutError{Operation: "fibonacci", Limit: 5 * time.Second}
		err := CalculationError{Cause: inner}

		var timeoutErr TimeoutError
		if !errors.As(err, &timeoutErr) {
			t.Error("errors.As should find TimeoutError through CalculationError")
		}
	})

	t.Run("ValidationError wrapped with WrapError", func(t *testing.T) {
		t.Parallel()
		inner := ValidationError{Field: "n", Message: "too large"}
		err := WrapError(inner, "config check failed")

		var validationErr ValidationError
		if !errors.As(err, &validationErr) {
			t.Error("errors.As should find ValidationError through WrapError")
		}
	})

	t.Run("MemoryError wrapped in CalculationError", func(t *testing.T) {
		t.Parallel()
		inner := MemoryError{Requested: 4096, Available: 1024, Limit: 2048}
		err := CalculationError{Cause: inner}

		var memErr MemoryError
		if !errors.As(err, &memErr) {
			t.Error("errors.As should find MemoryError through CalculationError")
		}
	})
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

func TestWrapCalculationError_nil(t *testing.T) {
	t.Parallel()
	if WrapCalculationError(nil, CalculationContext{N: 1}) != nil {
		t.Fatal("expected nil when cause is nil")
	}
}

func TestCalculationError_FormatDiagnostic(t *testing.T) {
	t.Parallel()
	err := WrapCalculationError(errors.New("fail"), CalculationContext{
		N:                     1_000_000,
		ParallelThresholdBits: 4096,
		FFTThresholdBits:      500_000,
		StrassenThresholdBits: 3072,
		MemoryEstimateBytes:   1024 * 1024,
		ConfigExcerpt:         "algo=fast",
	})
	var ce CalculationError
	if !errors.As(err, &ce) {
		t.Fatal("expected CalculationError")
	}
	if !ce.HasDiagnostic() {
		t.Fatal("expected HasDiagnostic true")
	}
	d := ce.FormatDiagnostic()
	for _, want := range []string{"n=1000000", "parallel_bits=4096", "fft_bits=500000", "strassen_bits=3072", "mem_est=", "config=algo=fast"} {
		if !strings.Contains(d, want) {
			t.Errorf("FormatDiagnostic() missing %q in %q", want, d)
		}
	}
}

func TestCalculationError_Error_NilCause(t *testing.T) {
	t.Parallel()
	// Zero-value CalculationError must still produce a usable message.
	var err CalculationError
	if got := err.Error(); got != "calculation error" {
		t.Errorf("Error() with nil cause = %q, want %q", got, "calculation error")
	}
}

func TestCalculationError_HasDiagnostic(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		ctx  CalculationContext
		want bool
	}{
		{"zero context", CalculationContext{}, false},
		{"config excerpt only", CalculationContext{ConfigExcerpt: "algo=fast"}, true},
		{"memory estimate only", CalculationContext{MemoryEstimateBytes: 1}, true},
		{"n only", CalculationContext{N: 42}, true},
		{"parallel threshold only", CalculationContext{ParallelThresholdBits: 4096}, true},
		{"fft threshold only", CalculationContext{FFTThresholdBits: 500_000}, true},
		{"strassen threshold only", CalculationContext{StrassenThresholdBits: 3072}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := CalculationError{Cause: errors.New("x"), CalculationContext: tt.ctx}
			if got := err.HasDiagnostic(); got != tt.want {
				t.Errorf("HasDiagnostic() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculationError_FormatDiagnostic_branches(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		ctx  CalculationContext
		want string
	}{
		{
			name: "empty without diagnostic",
			ctx:  CalculationContext{},
			want: "",
		},
		{
			// N=0 must not emit an "n=" field nor a leading separator.
			name: "thresholds only, no n prefix",
			ctx:  CalculationContext{ParallelThresholdBits: 4096},
			want: "parallel_bits=4096; fft_bits=auto; strassen_bits=auto",
		},
		{
			name: "n only renders auto thresholds",
			ctx:  CalculationContext{N: 10},
			want: "n=10; parallel_bits=auto; fft_bits=auto; strassen_bits=auto",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := CalculationError{Cause: errors.New("x"), CalculationContext: tt.ctx}
			if got := err.FormatDiagnostic(); got != tt.want {
				t.Errorf("FormatDiagnostic() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSanitizeConfigExcerpt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text unchanged", "algo=fast", "algo=fast"},
		{"newlines flattened and trimmed", "  line1\nline2  ", "line1 line2"},
		{"exactly max length untouched", strings.Repeat("a", 200), strings.Repeat("a", 200)},
		{"over max length truncated with ellipsis", strings.Repeat("a", 250), strings.Repeat("a", 200) + "…"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := sanitizeConfigExcerpt(tt.input); got != tt.want {
				t.Errorf("sanitizeConfigExcerpt(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestSanitizeConfigExcerpt_RuneBoundary ensures truncation never splits a
// multibyte rune: a >maxLen string whose 200th byte lands inside a rune must be
// cut back to the rune boundary, yielding valid UTF-8.
func TestSanitizeConfigExcerpt_RuneBoundary(t *testing.T) {
	t.Parallel()
	input := strings.Repeat("a", 199) + "好" // 199 + 3 bytes = 202 > 200; byte 200 is mid-rune
	got := sanitizeConfigExcerpt(input)
	if !utf8.ValidString(got) {
		t.Fatalf("sanitizeConfigExcerpt produced invalid UTF-8: %q", got)
	}
	if want := strings.Repeat("a", 199) + "…"; got != want {
		t.Errorf("sanitizeConfigExcerpt(...) = %q, want %q", got, want)
	}
}

func TestFormatBytesLocal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		bytes uint64
		want  string
	}{
		{"zero bytes", 0, "0 B"},
		{"below KB boundary", 1023, "1023 B"},
		{"KB boundary", 1 << 10, "1.0 KB"},
		{"fractional KB", 1536, "1.5 KB"},
		{"MB boundary", 1 << 20, "1.0 MB"},
		{"GB boundary", 1 << 30, "1.0 GB"},
		{"fractional GB", 3<<30 + 1<<29, "3.5 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := formatBytesLocal(tt.bytes); got != tt.want {
				t.Errorf("formatBytesLocal(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestCalculationError_FormatDiagnostic_sanitizesExcerpt(t *testing.T) {
	t.Parallel()
	err := WrapCalculationError(errors.New("x"), CalculationContext{
		N:             10,
		ConfigExcerpt: "line1\nline2\tsecret",
	})
	var ce CalculationError
	if !errors.As(err, &ce) {
		t.Fatal("expected CalculationError")
	}
	d := ce.FormatDiagnostic()
	if strings.Contains(d, "\n") {
		t.Errorf("diagnostic should not contain newlines: %q", d)
	}
}
