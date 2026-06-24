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

	// Boolean overrides
	{"VERBOSE", []string{"v", "verbose"}, func(c *AppConfig, v string) error {
		c.Verbose = parseBoolEnv(v, c.Verbose)
		return nil
	}},
	{"DETAILS", []string{"d", "details"}, func(c *AppConfig, v string) error {
		c.Details = parseBoolEnv(v, c.Details)
		return nil
	}},
	{"QUIET", []string{"quiet", "q"}, func(c *AppConfig, v string) error {
		c.Quiet = parseBoolEnv(v, c.Quiet)
		return nil
	}},
	{"MACHINE_OUTPUT", []string{"machine"}, func(c *AppConfig, v string) error {
		c.MachineOutput = parseBoolEnv(v, c.MachineOutput)
		return nil
	}},
	{"CALIBRATE", []string{"calibrate"}, func(c *AppConfig, v string) error {
		c.Calibrate = parseBoolEnv(v, c.Calibrate)
		return nil
	}},
	{"AUTO_CALIBRATE", []string{"auto-calibrate"}, func(c *AppConfig, v string) error {
		c.AutoCalibrate = parseBoolEnv(v, c.AutoCalibrate)
		return nil
	}},
	{"CALCULATE", []string{"calculate", "c"}, func(c *AppConfig, v string) error {
		c.ShowValue = parseBoolEnv(v, c.ShowValue)
		return nil
	}},
	{"TUI", []string{"tui"}, func(c *AppConfig, v string) error {
		c.TUI = parseBoolEnv(v, c.TUI)
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
// Returns defaultVal if the value is not recognized.
func parseBoolEnv(val string, defaultVal bool) bool {
	switch strings.ToLower(val) {
	case "true", "1", "yes":
		return true
	case "false", "0", "no":
		return false
	}
	return defaultVal
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

// applyEnvOverrides applies environment variable values to the configuration
// for any flags that were not explicitly set on the command line.
// This implements the priority: CLI flags > Environment variables > Defaults.
//
// Supported environment variables (all prefixed with FIBCALC_):
//   - N, ALGO, TIMEOUT, THRESHOLD, FFT_THRESHOLD, STRASSEN_THRESHOLD,
//     VERBOSE, DETAILS, QUIET, MACHINE_OUTPUT, CALIBRATE, AUTO_CALIBRATE, CALCULATE,
//     OUTPUT, CALIBRATION_PROFILE, MEMORY_LIMIT, TUI
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
