// This file contains environment variable utilities for configuration override.

package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// Environment Variable Utilities
// ─────────────────────────────────────────────────────────────────────────────

// getEnvString returns the value of the environment variable with the given key
// (prefixed with EnvPrefix), or the default value if not set.
func getEnvString(key, defaultVal string) string {
	if val := os.Getenv(EnvPrefix + key); val != "" {
		return val
	}
	return defaultVal
}

// getEnvUint64 returns the value of the environment variable with the given key
// (prefixed with EnvPrefix) parsed as uint64, or the default value if not set
// or invalid.
func getEnvUint64(key string, defaultVal uint64) uint64 {
	if val := os.Getenv(EnvPrefix + key); val != "" {
		if parsed, err := strconv.ParseUint(val, 10, 64); err == nil {
			return parsed
		}
	}
	return defaultVal
}

// getEnvInt returns the value of the environment variable with the given key
// (prefixed with EnvPrefix) parsed as int, or the default value if not set
// or invalid.
func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(EnvPrefix + key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return defaultVal
}

// getEnvBool returns the value of the environment variable with the given key
// (prefixed with EnvPrefix) parsed as bool, or the default value if not set.
// Accepts "true", "1", "yes" as true; "false", "0", "no" as false (case-insensitive).
func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(EnvPrefix + key); val != "" {
		switch strings.ToLower(val) {
		case "true", "1", "yes":
			return true
		case "false", "0", "no":
			return false
		}
	}
	return defaultVal
}

// getEnvDuration returns the value of the environment variable with the given key
// (prefixed with EnvPrefix) parsed as time.Duration, or the default value if not
// set or invalid. Accepts formats like "5m", "30s", "1h30m".
func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(EnvPrefix + key); val != "" {
		if parsed, err := time.ParseDuration(val); err == nil {
			return parsed
		}
	}
	return defaultVal
}

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
	apply  func(*AppConfig, string)
}

// envOverrides is the declarative table of all environment variable overrides.
// Order matches the original procedural grouping (numeric, duration, string, bool).
var envOverrides = []envOverride{
	// Numeric overrides
	{"N", []string{"n"}, func(c *AppConfig, v string) {
		if parsed, err := strconv.ParseUint(v, 10, 64); err == nil {
			c.N = parsed
		}
	}},
	{"THRESHOLD", []string{"threshold"}, func(c *AppConfig, v string) {
		if parsed, err := strconv.Atoi(v); err == nil {
			c.Threshold = parsed
		}
	}},
	{"FFT_THRESHOLD", []string{"fft-threshold"}, func(c *AppConfig, v string) {
		if parsed, err := strconv.Atoi(v); err == nil {
			c.FFTThreshold = parsed
		}
	}},
	{"STRASSEN_THRESHOLD", []string{"strassen-threshold"}, func(c *AppConfig, v string) {
		if parsed, err := strconv.Atoi(v); err == nil {
			c.StrassenThreshold = parsed
		}
	}},

	// Duration overrides
	{"TIMEOUT", []string{"timeout"}, func(c *AppConfig, v string) {
		if parsed, err := time.ParseDuration(v); err == nil {
			c.Timeout = parsed
		}
	}},

	// String overrides
	{"ALGO", []string{"algo"}, func(c *AppConfig, v string) {
		c.Algo = v
	}},
	{"OUTPUT", []string{"output", "o"}, func(c *AppConfig, v string) {
		c.OutputFile = v
	}},
	{"CALIBRATION_PROFILE", []string{"calibration-profile"}, func(c *AppConfig, v string) {
		c.CalibrationProfile = v
	}},
	{"MEMORY_LIMIT", []string{"memory-limit"}, func(c *AppConfig, v string) {
		c.MemoryLimit = v
	}},

	// Boolean overrides
	{"VERBOSE", []string{"v", "verbose"}, func(c *AppConfig, v string) {
		c.Verbose = parseBoolEnv(v, c.Verbose)
	}},
	{"DETAILS", []string{"d", "details"}, func(c *AppConfig, v string) {
		c.Details = parseBoolEnv(v, c.Details)
	}},
	{"QUIET", []string{"quiet", "q"}, func(c *AppConfig, v string) {
		c.Quiet = parseBoolEnv(v, c.Quiet)
	}},
	{"MACHINE_OUTPUT", []string{"machine"}, func(c *AppConfig, v string) {
		c.MachineOutput = parseBoolEnv(v, c.MachineOutput)
	}},
	{"CALIBRATE", []string{"calibrate"}, func(c *AppConfig, v string) {
		c.Calibrate = parseBoolEnv(v, c.Calibrate)
	}},
	{"AUTO_CALIBRATE", []string{"auto-calibrate"}, func(c *AppConfig, v string) {
		c.AutoCalibrate = parseBoolEnv(v, c.AutoCalibrate)
	}},
	{"CALCULATE", []string{"calculate", "c"}, func(c *AppConfig, v string) {
		c.ShowValue = parseBoolEnv(v, c.ShowValue)
	}},
	{"TUI", []string{"tui"}, func(c *AppConfig, v string) {
		c.TUI = parseBoolEnv(v, c.TUI)
	}},
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
func applyEnvOverrides(config *AppConfig, fs *flag.FlagSet) {
	for _, o := range envOverrides {
		if isFlagSetAny(fs, o.flags...) {
			continue
		}
		if val := os.Getenv(EnvPrefix + o.envKey); val != "" {
			o.apply(config, val)
		}
	}
}
