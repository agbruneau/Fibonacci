package config

import (
	"fmt"

	apperrors "github.com/agbru/fibcalc/internal/errors"
	"github.com/agbru/fibcalc/internal/fibonacci/memory"
)

// MemoryBudgetReport summarizes the result of a memory budget validation.
// It is returned even when the budget is satisfied so callers can present
// the estimate to the user.
type MemoryBudgetReport struct {
	// Estimate is the predicted memory usage breakdown for F(N).
	Estimate memory.MemoryEstimate
	// LimitBytes is the parsed user-provided limit, in bytes.
	LimitBytes uint64
	// LimitRaw is the original human-readable limit string (e.g. "8G").
	LimitRaw string
}

// MemoryLimitParseError indicates that the user-supplied --memory-limit value
// could not be parsed (e.g. unrecognised suffix or non-numeric).
type MemoryLimitParseError struct {
	// Raw is the offending value as supplied by the user.
	Raw string
	// Cause is the underlying parse error.
	Cause error
}

// Error returns a formatted message describing the parse failure.
func (e MemoryLimitParseError) Error() string {
	return fmt.Sprintf("invalid memory limit %q: %v", e.Raw, e.Cause)
}

// Unwrap exposes the underlying error for errors.Is / errors.As.
func (e MemoryLimitParseError) Unwrap() error { return e.Cause }

// ValidateMemoryBudget checks whether the estimated memory usage for F(cfg.N)
// fits within the configured memory limit. It is a pure function: it performs
// no I/O. Callers are responsible for presenting the report or error.
//
// Returns a zero-value report and nil error when cfg.MemoryLimit is empty
// (no validation requested).
//
// Error contract:
//   - MemoryLimitParseError when --memory-limit cannot be parsed.
//   - apperrors.MemoryError when the estimate exceeds the limit. The report
//     is also returned in this case so callers can format the estimate.
//   - nil when no limit is set or the estimate fits.
func ValidateMemoryBudget(cfg AppConfig) (MemoryBudgetReport, error) {
	if cfg.MemoryLimit == "" {
		return MemoryBudgetReport{}, nil
	}

	limit, err := memory.ParseMemoryLimit(cfg.MemoryLimit)
	if err != nil {
		return MemoryBudgetReport{}, MemoryLimitParseError{Raw: cfg.MemoryLimit, Cause: err}
	}

	est := memory.EstimateMemoryUsage(cfg.N)
	report := MemoryBudgetReport{
		Estimate:   est,
		LimitBytes: limit,
		LimitRaw:   cfg.MemoryLimit,
	}

	if est.TotalBytes > limit {
		return report, apperrors.MemoryError{
			Requested: est.TotalBytes,
			Available: limit,
			Limit:     limit,
		}
	}

	return report, nil
}
