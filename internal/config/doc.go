// Package config provides configuration management for the fibcalc application.
// It defines the data structure, handles CLI flag parsing, and validates values.
// Adaptive defaults (thresholds at 0) combine [ApplyAdaptiveThresholds] with
// hardware heuristics ([DetectHardwareHeuristic], [EstimateOptimalParallelThreshold]).
package config
