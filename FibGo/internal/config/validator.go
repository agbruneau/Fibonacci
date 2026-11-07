package config

import (
	"fmt"

	"github.com/agbruneau/FibGo/internal/apperrors"
	"github.com/agbruneau/FibGo/internal/fibonacci/memory"
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
// could not be parsed (e.g. unrecognized suffix or non-numeric).
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

// IndexTooLargeError reports an n whose full F(n) cannot be computed on this
// platform at all: its memory estimate exceeds memory.AddressableHeapBytes.
// It is refused whether or not --memory-limit is set (audit 2026-11-07); before,
// such an n reached the arena allocation and crashed the process there.
type IndexTooLargeError struct {
	// N is the requested index.
	N uint64
	// EstimateBytes is memory.EstimateMemoryUsage(N).TotalBytes.
	EstimateBytes uint64
}

// Error returns a one-line description naming the estimate and the ceiling.
func (e IndexTooLargeError) Error() string {
	return fmt.Sprintf("F(%d) cannot be computed in full on this platform: its memory estimate (%s) exceeds the %s the Go runtime can address",
		e.N, memory.FormatBytes(e.EstimateBytes), memory.FormatBytes(memory.AddressableHeapBytes))
}

// ValidateMemoryBudget checks whether the estimated memory usage for F(cfg.N)
// fits within the configured memory limit. It is a pure function: it performs
// no I/O. Callers are responsible for presenting the report or error.
//
// Returns a zero-value report and nil error when cfg.MemoryLimit is empty
// (no validation requested) and the estimate is addressable.
//
// Error contract:
//   - IndexTooLargeError when the estimate exceeds memory.AddressableHeapBytes,
//     with or without a limit; the report carries the estimate.
//   - MemoryLimitParseError when --memory-limit cannot be parsed.
//   - apperrors.MemoryError when the estimate exceeds the limit. The report
//     is also returned in this case so callers can format the estimate.
//   - nil when no limit is set or the estimate fits.
func ValidateMemoryBudget(cfg AppConfig) (MemoryBudgetReport, error) {
	est := memory.EstimateMemoryUsage(cfg.N)
	if !memory.IsAddressable(est) {
		return MemoryBudgetReport{Estimate: est}, IndexTooLargeError{N: cfg.N, EstimateBytes: est.TotalBytes}
	}

	if cfg.MemoryLimit == "" {
		return MemoryBudgetReport{}, nil
	}

	limit, err := memory.ParseMemoryLimit(cfg.MemoryLimit)
	if err != nil {
		return MemoryBudgetReport{}, MemoryLimitParseError{Raw: cfg.MemoryLimit, Cause: err}
	}

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
