// Package config owns the AppConfig type, its flag-binding logic, and
// the validation/normalisation rules applied before fibcalc starts a
// calculation. It is the single source of truth for tunable knobs
// (thresholds, timeouts, output options, calibration paths).
//
// Layered responsibilities:
//
//   - Parsing: ParseConfig builds a standard flag.FlagSet and wires it to
//     the AppConfig fields (via registerFlags). Env-var overrides go
//     through internal/config/env.go.
//   - Validation: Validate() rejects nonsensical flag combinations and
//     out-of-range values early (negative thresholds, incompatible
//     --tui/--last-digits/--output combinations, unrecognized gc-control
//     or completion modes). It does not check the memory cap against the
//     estimated working set for the requested N — that budget check runs
//     later, against the actual N, in internal/app (validateMemoryBudget)
//     and internal/fibonacci (the fibonacci.CanCalculate package function).
//   - Adaptive defaults: ApplyAdaptiveThresholds() combines hardware
//     heuristics (DetectHardwareHeuristic, EstimateOptimalParallelThreshold)
//     and the user-tunable ThresholdTuningProfile (see threshold_tuning.go).
//
// Wiring to the threshold package:
//
//	Since Audit-PRD E3-R1, internal/fibonacci/threshold no longer imports
//	internal/config. The intended runtime wiring is for the
//	application layer to translate DefaultThresholdTuning into a
//	threshold.Tuning value and call threshold.SetTuning at startup. This
//	keeps the import arrows pointing toward the kernel (config → memory,
//	threshold → ø) without re-introducing the historic cycle.
//
// Dependencies (downward only):
//
//	internal/config → internal/fibonacci/memory, internal/errors, internal/ui
package config
