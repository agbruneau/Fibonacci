// Package apperrors defines structured application error types,
// allowing for a clear distinction between error classes (configuration,
// calculation, etc.) and for carrying the underlying cause.
//
// Error Wrapping Guidelines:
// This package follows Go's error wrapping conventions using fmt.Errorf with %w.
// All error types implement the Unwrap() method to support errors.Is() and errors.As().
//
// Calculation failures may be wrapped as [CalculationError] with optional
// [CalculationContext] (thresholds, memory estimate, safe config excerpt) for
// diagnostics; see [WrapCalculationError] and [CalculationError.FormatDiagnostic].
package apperrors
