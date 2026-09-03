// This file contains environment variable utilities for configuration override.

package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	apperrors "github.com/agbruneau/FibGo/internal/errors"
)

// isFlagSet reports whether any of the named flags was explicitly set on the
// command line. Aliased flags pass both their short and long names, since
// either form counts as the user speaking; a set flag suppresses the matching
// environment override.
func isFlagSet(fs *flag.FlagSet, names ...string) bool {
	found := false
	fs.Visit(func(f *flag.Flag) {
		for _, name := range names {
			if f.Name == name {
				found = true
			}
		}
	})
	return found
}

// envOverride declares a single environment variable override.
// Each entry maps an env key (without the FIBCALC_ prefix) to the CLI flag
// name(s) it corresponds to and a function that applies the env value.
type envOverride struct {
	envKey string
	flags  []string
	// apply mutates the config from the raw env value. It returns a structured
	// ConfigError when the value is explicitly set but unparsable, so a
	// malformed override is surfaced instead of being silently dropped back to
	// the default (which could trigger an O(memory) calculation / OOM).
	apply func(*AppConfig, string) error
}

// override builds an envOverride whose apply parses the raw value with parse
// and stores it through field. A parse failure becomes a ConfigError naming
// the variable (APP-09); it never falls back to the default silently.
func override[T any](envKey string, flags []string, field func(*AppConfig) *T, parse func(string) (T, error)) envOverride {
	return envOverride{envKey: envKey, flags: flags, apply: func(c *AppConfig, v string) error {
		parsed, err := parse(v)
		if err != nil {
			return malformedEnvError(envKey, v, err)
		}
		*field(c) = parsed
		return nil
	}}
}

func parseString(v string) (string, error) { return v, nil }
func parseUint64(v string) (uint64, error) { return strconv.ParseUint(v, 10, 64) }
func flagsOf(names ...string) []string     { return names }

// envOverrides is the declarative table of all environment variable overrides.
// Order matches the original procedural grouping (numeric, duration, string, bool).
var envOverrides = []envOverride{
	// Numeric overrides
	override("N", flagsOf("n"), func(c *AppConfig) *uint64 { return &c.N }, parseUint64),
	override("THRESHOLD", flagsOf("threshold"), func(c *AppConfig) *int { return &c.Threshold }, strconv.Atoi),
	override("FFT_THRESHOLD", flagsOf("fft-threshold"), func(c *AppConfig) *int { return &c.FFTThreshold }, strconv.Atoi),
	override("STRASSEN_THRESHOLD", flagsOf("strassen-threshold"), func(c *AppConfig) *int { return &c.StrassenThreshold }, strconv.Atoi),
	override("LAST_DIGITS", flagsOf("last-digits"), func(c *AppConfig) *int { return &c.LastDigits }, strconv.Atoi),

	// Duration overrides
	override("TIMEOUT", flagsOf("timeout"), func(c *AppConfig) *time.Duration { return &c.Timeout }, time.ParseDuration),

	// String overrides
	override("ALGO", flagsOf("algo"), func(c *AppConfig) *string { return &c.Algo }, parseString),
	override("OUTPUT", flagsOf("output", "o"), func(c *AppConfig) *string { return &c.OutputFile }, parseString),
	override("CALIBRATION_PROFILE", flagsOf("calibration-profile"), func(c *AppConfig) *string { return &c.CalibrationProfile }, parseString),
	override("MEMORY_LIMIT", flagsOf("memory-limit"), func(c *AppConfig) *string { return &c.MemoryLimit }, parseString),
	override("GC_CONTROL", flagsOf("gc-control"), func(c *AppConfig) *string { return &c.GCControl }, parseString),

	// Boolean overrides
	override("VERBOSE", flagsOf("v", "verbose"), func(c *AppConfig) *bool { return &c.Verbose }, parseBoolEnv),
	override("DETAILS", flagsOf("d", "details"), func(c *AppConfig) *bool { return &c.Details }, parseBoolEnv),
	override("QUIET", flagsOf("quiet", "q"), func(c *AppConfig) *bool { return &c.Quiet }, parseBoolEnv),
	override("MACHINE_OUTPUT", flagsOf("machine"), func(c *AppConfig) *bool { return &c.MachineOutput }, parseBoolEnv),
	override("CALIBRATE", flagsOf("calibrate"), func(c *AppConfig) *bool { return &c.Calibrate }, parseBoolEnv),
	override("AUTO_CALIBRATE", flagsOf("auto-calibrate"), func(c *AppConfig) *bool { return &c.AutoCalibrate }, parseBoolEnv),
	override("CALCULATE", flagsOf("calculate", "c"), func(c *AppConfig) *bool { return &c.ShowValue }, parseBoolEnv),
	override("TUI", flagsOf("tui"), func(c *AppConfig) *bool { return &c.TUI }, parseBoolEnv),
	override("DYNAMIC_THRESHOLDS", flagsOf("dynamic-thresholds"), func(c *AppConfig) *bool { return &c.DynamicThresholds }, parseBoolEnv),
}

// malformedEnvError builds a structured ConfigError for an environment
// variable that was explicitly set but could not be parsed.
func malformedEnvError(envKey, value string, cause error) error {
	return apperrors.NewConfigError(
		"invalid value %q for environment variable %s%s: %v",
		value, EnvPrefix, envKey, cause)
}

// parseBoolEnv parses a boolean environment variable value.
// Accepts "true", "1", "yes" as true; "false", "0", "no" as false (case-insensitive).
// Anything else is an error, never a silent fallback to the default (APP-09).
func parseBoolEnv(val string) (bool, error) {
	switch strings.ToLower(val) {
	case "true", "1", "yes":
		return true, nil
	case "false", "0", "no":
		return false, nil
	}
	return false, fmt.Errorf("unrecognized boolean value %q (expected true/1/yes or false/0/no)", val)
}

// envProvided reports whether the FIBCALC_-prefixed variable for key carries a
// value. It mirrors the emptiness test applyEnvOverrides uses, so a variable
// set to the empty string counts as absent in both places.
func envProvided(key string) bool {
	return os.Getenv(EnvPrefix+key) != ""
}

// markExplicitThresholds records which of the three thresholds the user pinned,
// so calibration can fill only the ones left to the tool (audit M-03).
//
// A threshold counts as explicit when its flag appears on the command line or
// when its environment variable is set, which together are exactly the two ways
// a value can arrive that is not the tool's own choice. It must run after
// applyEnvOverrides so the two agree on what the environment provided.
func markExplicitThresholds(config *AppConfig, fs *flag.FlagSet) {
	config.ThresholdExplicit = isFlagSet(fs, "threshold") || envProvided("THRESHOLD")
	config.FFTThresholdExplicit = isFlagSet(fs, "fft-threshold") || envProvided("FFT_THRESHOLD")
	config.StrassenThresholdExplicit = isFlagSet(fs, "strassen-threshold") || envProvided("STRASSEN_THRESHOLD")
}

// applyEnvOverrides applies environment variable values to the configuration
// for any flags that were not explicitly set on the command line.
// This implements the priority: CLI flags > Environment variables > Defaults.
//
// Supported environment variables (all prefixed with FIBCALC_):
//   - N, ALGO, TIMEOUT, THRESHOLD, FFT_THRESHOLD, STRASSEN_THRESHOLD, LAST_DIGITS,
//     VERBOSE, DETAILS, QUIET, MACHINE_OUTPUT, CALIBRATE, AUTO_CALIBRATE, CALCULATE,
//     OUTPUT, CALIBRATION_PROFILE, MEMORY_LIMIT, GC_CONTROL, TUI, DYNAMIC_THRESHOLDS
//   - FIBCALC_TUI_THEME: TUI palette (read by ui.GetCurrentTUITheme), e.g. high-contrast
//
// It returns a structured ConfigError (without mutating further) the first
// time an explicitly-set environment variable holds an unparsable value, so
// that a malformed override is rejected loudly instead of silently falling
// back to the default.
func applyEnvOverrides(config *AppConfig, fs *flag.FlagSet) error {
	for _, o := range envOverrides {
		if isFlagSet(fs, o.flags...) {
			continue
		}
		if val := os.Getenv(EnvPrefix + o.envKey); val != "" {
			if err := o.apply(config, val); err != nil {
				return err
			}
		}
	}
	return nil
}
