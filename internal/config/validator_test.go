package config

import (
	"errors"
	"strings"
	"testing"

	apperrors "github.com/agbruneau/FibGo/internal/errors"
)

func TestValidateMemoryBudget_NoLimit(t *testing.T) {
	t.Parallel()
	report, err := ValidateMemoryBudget(AppConfig{N: 1_000_000, MemoryLimit: ""})
	if err != nil {
		t.Fatalf("expected nil error when no limit is set, got %v", err)
	}
	if report != (MemoryBudgetReport{}) {
		t.Errorf("expected zero-value report when no limit is set, got %+v", report)
	}
}

func TestValidateMemoryBudget_ParseError(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{"abc", "12X", "??G"} {
		t.Run(raw, func(t *testing.T) {
			t.Parallel()
			_, err := ValidateMemoryBudget(AppConfig{N: 1000, MemoryLimit: raw})
			if err == nil {
				t.Fatalf("expected parse error for limit %q, got nil", raw)
			}
			var perr MemoryLimitParseError
			if !errors.As(err, &perr) {
				t.Fatalf("expected MemoryLimitParseError, got %T: %v", err, err)
			}
			if perr.Raw != raw {
				t.Errorf("Raw = %q, want %q", perr.Raw, raw)
			}
			// Unwrap must expose the underlying parse failure for errors.Is/As.
			if perr.Unwrap() == nil {
				t.Error("Unwrap() = nil, want underlying cause")
			}
			if !strings.Contains(perr.Error(), raw) {
				t.Errorf("Error() = %q, should mention the offending value %q", perr.Error(), raw)
			}
		})
	}
}

func TestValidateMemoryBudget_ExceedsLimit(t *testing.T) {
	t.Parallel()
	const limitBytes = uint64(1 << 20) // "1M"
	report, err := ValidateMemoryBudget(AppConfig{N: 10_000_000, MemoryLimit: "1M"})
	if err == nil {
		t.Fatal("expected MemoryError for F(10M) under a 1M budget, got nil")
	}
	var merr apperrors.MemoryError
	if !errors.As(err, &merr) {
		t.Fatalf("expected apperrors.MemoryError, got %T: %v", err, err)
	}
	if merr.Limit != limitBytes || merr.Available != limitBytes {
		t.Errorf("Limit/Available = %d/%d, want both %d", merr.Limit, merr.Available, limitBytes)
	}
	if merr.Requested != report.Estimate.TotalBytes {
		t.Errorf("Requested = %d, want estimate total %d", merr.Requested, report.Estimate.TotalBytes)
	}
	// The report must stay usable on failure so callers can format the estimate.
	if report.LimitBytes != limitBytes || report.LimitRaw != "1M" {
		t.Errorf("report limit = %d/%q, want %d/%q", report.LimitBytes, report.LimitRaw, limitBytes, "1M")
	}
	if report.Estimate.TotalBytes <= limitBytes {
		t.Errorf("estimate %d should exceed limit %d", report.Estimate.TotalBytes, limitBytes)
	}
}

func TestValidateMemoryBudget_Fits(t *testing.T) {
	t.Parallel()
	const limitBytes = uint64(1 << 30) // "1G"
	report, err := ValidateMemoryBudget(AppConfig{N: 1000, MemoryLimit: "1G"})
	if err != nil {
		t.Fatalf("expected nil error for F(1000) under a 1G budget, got %v", err)
	}
	if report.LimitBytes != limitBytes || report.LimitRaw != "1G" {
		t.Errorf("report limit = %d/%q, want %d/%q", report.LimitBytes, report.LimitRaw, limitBytes, "1G")
	}
	if report.Estimate.TotalBytes == 0 || report.Estimate.TotalBytes > limitBytes {
		t.Errorf("estimate total = %d, want non-zero and within limit %d", report.Estimate.TotalBytes, limitBytes)
	}
}
