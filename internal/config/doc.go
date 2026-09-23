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
//     heuristics (DetectHardwareHeuristic, EstimateOptimalParallelThreshold).
//
// Dependencies (downward only):
//
//	internal/config → internal/fibonacci/memory, internal/apperrors, internal/ui
package config
