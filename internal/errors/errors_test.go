// Package apperrors provides tests for application error types.
package apperrors

import (
	"context"
	"errors"
	"strings"
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
	})
	var ce CalculationError
	if !errors.As(err, &ce) {
		t.Fatal("expected CalculationError")
	}
	if !ce.HasDiagnostic() {
		t.Fatal("expected HasDiagnostic true")
	}
	d := ce.FormatDiagnostic()
	for _, want := range []string{"n=1000000", "parallel_bits=4096", "fft_bits=500000", "strassen_bits=3072", "mem_est="} {
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
