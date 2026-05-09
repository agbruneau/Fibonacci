// This file defines the CalibrationStrategy abstraction used by
// AutoCalibrateWithProfile to escalate from a fast micro-benchmark
// based estimate to a full calibration sweep when the fast pass does
// not produce a confident enough result.
//
// R3.3: previously AutoCalibrateWithProfile inlined a two-tier
// (quick → full) decision tree. Extracting a Strategy interface gives
// each tier a self-contained implementation file (strategy_fast.go,
// strategy_complete.go) and lets the orchestrator stay focused on the
// escalation policy alone.

package calibration

import (
	"context"
	"io"

	"github.com/agbru/fibcalc/internal/config"
	"github.com/agbru/fibcalc/internal/fibonacci"
)

// Confidence is a calibration-quality score in the closed range [0, 1].
// 0 means "no useful signal" (the strategy effectively failed silently);
// 1 means "fully measured, persist as authoritative". The orchestrator
// in AutoCalibrateWithProfile escalates from FastStrategy to
// CompleteStrategy when the reported confidence is below
// EscalationConfidenceThreshold.
type Confidence float64

// EscalationConfidenceThreshold is the minimum FastStrategy confidence
// the orchestrator accepts before deciding the fast pass is good
// enough. Below this value AutoCalibrateWithProfile falls through to
// CompleteStrategy. The value mirrors the historical inline check
// (microResults.Confidence >= 0.5) so behaviour is preserved bit-for-bit.
const EscalationConfidenceThreshold Confidence = 0.5

// StrategyOptions bundles everything a CalibrationStrategy needs to
// run. It is built once by AutoCalibrateWithProfile from the caller's
// inputs and passed to each tried strategy.
//
// Fields are shared across strategies on purpose: a strategy is free to
// ignore any field that does not apply to it (e.g. FastStrategy ignores
// CalculatorRegistry because micro-benchmarks call bigfft directly).
type StrategyOptions struct {
	// BaseConfig is the configuration the calling application started
	// with. Strategies do not mutate it; they return updated values
	// through the produced *CalibrationProfile.
	BaseConfig config.AppConfig

	// CalculatorRegistry is the set of fibonacci.Calculator
	// implementations available to the strategy. CompleteStrategy
	// requires the "fast" entry; the optional "matrix" entry, when
	// present, lets it also tune the Strassen threshold.
	CalculatorRegistry map[string]fibonacci.Calculator

	// Out is the writer used for human-facing progress messages.
	// Strategies should keep their output minimal; the orchestrator is
	// responsible for the final summary line.
	Out io.Writer
}

// CalibrationStrategy is the abstraction shared by every calibration
// tier. Implementations must be safe to call once with a fresh
// StrategyOptions. They are not expected to be safe for concurrent
// reuse across goroutines — AutoCalibrateWithProfile invokes them
// sequentially.
//
// The returned *CalibrationProfile must carry the optimal threshold
// values the strategy discovered (parallel/FFT/Strassen). When a
// strategy cannot produce useful data it must return a non-nil error
// and (profile, confidence) values are ignored by the orchestrator.
type CalibrationStrategy interface {
	// Name returns a short human-readable identifier used in log
	// output ("fast", "complete"). Stable across runs.
	Name() string

	// Calibrate runs the strategy. It must honour ctx cancellation.
	// On success it returns a populated *CalibrationProfile, the
	// associated Confidence score, and a nil error. On failure it
	// returns (nil, 0, err) so the orchestrator can escalate.
	Calibrate(ctx context.Context, opts StrategyOptions) (*CalibrationProfile, Confidence, error)
}
