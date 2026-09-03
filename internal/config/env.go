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

// ─────────────────────────────────────────────────────────────────────────────
// Environment Variable Utilities
// ─────────────────────────────────────────────────────────────────────────────

// isFlagSet checks if a flag was explicitly set on the command line.
// This is used to determine whether to apply environment variable overrides.
func isFlagSet(fs *flag.FlagSet, name string) bool {
	found := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// isFlagSetAny checks if any of the specified flags were explicitly set.
// This is useful for aliased flags where either the short or long form may be used.
func isFlagSetAny(fs *flag.FlagSet, names ...string) bool {
	for _, name := range names {
		if isFlagSet(fs, name) {
			return true
		}
	}
	return false
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

// envOverrides is the declarative table of all environment variable overrides.
// Order matches the original procedural grouping (numeric, duration, string, bool).
var envOverrides = []envOverride{
	// Numeric overrides
	{"N", []string{"n"}, func(c *AppConfig, v string) error {
		parsed, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return malformedEnvError("N", v, err)
		}
		c.N = parsed
		return nil
	}},
	{"THRESHOLD", []string{"threshold"}, func(c *AppConfig, v string) error {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return malformedEnvError("THRESHOLD", v, err)
		}
		c.Threshold = parsed
		return nil
	}},
	{"FFT_THRESHOLD", []string{"fft-threshold"}, func(c *AppConfig, v string) error {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return malformedEnvError("FFT_THRESHOLD", v, err)
		}
		c.FFTThreshold = parsed
		return nil
	}},
	{"STRASSEN_THRESHOLD", []string{"strassen-threshold"}, func(c *AppConfig, v string) error {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return malformedEnvError("STRASSEN_THRESHOLD", v, err)
		}
		c.StrassenThreshold = parsed
		return nil
	}},
	{"LAST_DIGITS", []string{"last-digits"}, func(c *AppConfig, v string) error {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return malformedEnvError("LAST_DIGITS", v, err)
		}
		c.LastDigits = parsed
		return nil
	}},

	// Duration overrides
	{"TIMEOUT", []string{"timeout"}, func(c *AppConfig, v string) error {
		parsed, err := time.ParseDuration(v)
		if err != nil {
			return malformedEnvError("TIMEOUT", v, err)
		}
		c.Timeout = parsed
		return nil
	}},

	// String overrides
	{"ALGO", []string{"algo"}, func(c *AppConfig, v string) error {
		c.Algo = v
		return nil
	}},
	{"OUTPUT", []string{"output", "o"}, func(c *AppConfig, v string) error {
		c.OutputFile = v
		return nil
	}},
	{"CALIBRATION_PROFILE", []string{"calibration-profile"}, func(c *AppConfig, v string) error {
		c.CalibrationProfile = v
		return nil
	}},
	{"MEMORY_LIMIT", []string{"memory-limit"}, func(c *AppConfig, v string) error {
		c.MemoryLimit = v
		return nil
	}},
	{"GC_CONTROL", []string{"gc-control"}, func(c *AppConfig, v string) error {
		c.GCControl = v
		return nil
	}},

	// Boolean overrides
	{"VERBOSE", []string{"v", "verbose"}, func(c *AppConfig, v string) error {
		parsed, err := parseBoolEnv(v, c.Verbose)
		if err != nil {
			return malformedEnvError("VERBOSE", v, err)
		}
		c.Verbose = parsed
		return nil
	}},
	{"DETAILS", []string{"d", "details"}, func(c *AppConfig, v string) error {
		parsed, err := parseBoolEnv(v, c.Details)
		if err != nil {
			return malformedEnvError("DETAILS", v, err)
		}
		c.Details = parsed
		return nil
	}},
	{"QUIET", []string{"quiet", "q"}, func(c *AppConfig, v string) error {
		parsed, err := parseBoolEnv(v, c.Quiet)
		if err != nil {
			return malformedEnvError("QUIET", v, err)
		}
		c.Quiet = parsed
		return nil
	}},
	{"MACHINE_OUTPUT", []string{"machine"}, func(c *AppConfig, v string) error {
		parsed, err := parseBoolEnv(v, c.MachineOutput)
		if err != nil {
			return malformedEnvError("MACHINE_OUTPUT", v, err)
		}
		c.MachineOutput = parsed
		return nil
	}},
	{"CALIBRATE", []string{"calibrate"}, func(c *AppConfig, v string) error {
		parsed, err := parseBoolEnv(v, c.Calibrate)
		if err != nil {
			return malformedEnvError("CALIBRATE", v, err)
		}
		c.Calibrate = parsed
		return nil
	}},
	{"AUTO_CALIBRATE", []string{"auto-calibrate"}, func(c *AppConfig, v string) error {
		parsed, err := parseBoolEnv(v, c.AutoCalibrate)
		if err != nil {
			return malformedEnvError("AUTO_CALIBRATE", v, err)
		}
		c.AutoCalibrate = parsed
		return nil
	}},
	{"CALCULATE", []string{"calculate", "c"}, func(c *AppConfig, v string) error {
		parsed, err := parseBoolEnv(v, c.ShowValue)
		if err != nil {
			return malformedEnvError("CALCULATE", v, err)
		}
		c.ShowValue = parsed
		return nil
	}},
	{"TUI", []string{"tui"}, func(c *AppConfig, v string) error {
		parsed, err := parseBoolEnv(v, c.TUI)
		if err != nil {
			return malformedEnvError("TUI", v, err)
		}
		c.TUI = parsed
		return nil
	}},
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
// Returns a malformedEnvError if the value is not recognized, instead of
// silently falling back to defaultVal (APP-09).
func parseBoolEnv(val string, defaultVal bool) (bool, error) {
	switch strings.ToLower(val) {
	case "true", "1", "yes":
		return true, nil
	case "false", "0", "no":
		return false, nil
	}
	return defaultVal, fmt.Errorf("unrecognized boolean value %q (expected true/1/yes or false/0/no)", val)
}

// validateEnvOverrides verifies the structural consistency of the envOverrides
// table against the canonical CLI FlagSet. It catches the silent inconsistency
// risk inherent to the dual declaration (FlagSet in registerFlags + table here):
//
//  1. Every envKey is non-empty, free of the FIBCALC_ prefix, and unique.
//  2. Every flag name referenced in an entry's flags slice exists on fs.
//  3. No entry has an empty flags slice (would short-circuit isFlagSetAny and
//     always apply the env override, defeating CLI > env priority).
//  4. apply is non-nil.
//
// It is intended to be called from tests. Returning an error (rather than
// panicking) keeps init-time behavior unaffected.
func validateEnvOverrides(fs *flag.FlagSet) error {
	known := make(map[string]struct{})
	fs.VisitAll(func(f *flag.Flag) {
		known[f.Name] = struct{}{}
	})

	seen := make(map[string]struct{}, len(envOverrides))
	for _, o := range envOverrides {
		if o.envKey == "" {
			return fmt.Errorf("envOverrides: entry has empty envKey")
		}
		if strings.HasPrefix(o.envKey, EnvPrefix) {
			return fmt.Errorf("envOverrides: envKey %q must not include the %s prefix (added automatically)", o.envKey, EnvPrefix)
		}
		if _, dup := seen[o.envKey]; dup {
			return fmt.Errorf("envOverrides: duplicate envKey %q", o.envKey)
		}
		seen[o.envKey] = struct{}{}

		if len(o.flags) == 0 {
			return fmt.Errorf("envOverrides: entry %q has empty flags slice", o.envKey)
		}
		for _, name := range o.flags {
			if _, ok := known[name]; !ok {
				return fmt.Errorf("envOverrides: entry %q references unknown flag %q (not registered on FlagSet)", o.envKey, name)
			}
		}
		if o.apply == nil {
			return fmt.Errorf("envOverrides: entry %q has nil apply function", o.envKey)
		}
	}
	return nil
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
	config.ThresholdExplicit = isFlagSetAny(fs, "threshold") || envProvided("THRESHOLD")
	config.FFTThresholdExplicit = isFlagSetAny(fs, "fft-threshold") || envProvided("FFT_THRESHOLD")
	config.StrassenThresholdExplicit = isFlagSetAny(fs, "strassen-threshold") || envProvided("STRASSEN_THRESHOLD")
}

// applyEnvOverrides applies environment variable values to the configuration
// for any flags that were not explicitly set on the command line.
// This implements the priority: CLI flags > Environment variables > Defaults.
//
// Supported environment variables (all prefixed with FIBCALC_):
//   - N, ALGO, TIMEOUT, THRESHOLD, FFT_THRESHOLD, STRASSEN_THRESHOLD, LAST_DIGITS,
//     VERBOSE, DETAILS, QUIET, MACHINE_OUTPUT, CALIBRATE, AUTO_CALIBRATE, CALCULATE,
//     OUTPUT, CALIBRATION_PROFILE, MEMORY_LIMIT, GC_CONTROL, TUI
//   - FIBCALC_TUI_THEME: TUI palette (read by ui.GetCurrentTUITheme), e.g. high-contrast
//
// It returns a structured ConfigError (without mutating further) the first
// time an explicitly-set environment variable holds an unparsable value, so
// that a malformed override is rejected loudly instead of silently falling
// back to the default.
func applyEnvOverrides(config *AppConfig, fs *flag.FlagSet) error {
	for _, o := range envOverrides {
		if isFlagSetAny(fs, o.flags...) {
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
