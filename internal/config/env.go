// Package config provides the configuration management for the fibcalc application.
// This file contains environment variable utilities for configuration override.
package config

import (
	"flag"
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

// applyEnvOverrides applies environment variable values to the configuration
// for any flags that were not explicitly set on the command line.
// This implements the priority: CLI flags > Environment variables > Defaults.
//
// Supported environment variables:
//   - FIBCALC_N: Index of the Fibonacci number to calculate (uint64)
//   - FIBCALC_ALGO: Algorithm to use (string: fast, matrix, fft, all)
//   - FIBCALC_PORT: Port for server mode (string)
//   - FIBCALC_TIMEOUT: Calculation timeout (duration: "5m", "30s")
//   - FIBCALC_THRESHOLD: Parallelism threshold in bits (int)
//   - FIBCALC_FFT_THRESHOLD: FFT multiplication threshold in bits (int)
//   - FIBCALC_STRASSEN_THRESHOLD: Strassen algorithm threshold in bits (int)
//   - FIBCALC_SERVER: Enable server mode (bool: true/false, 1/0, yes/no)
//   - FIBCALC_JSON: Enable JSON output (bool)
//   - FIBCALC_VERBOSE: Enable verbose output (bool)
//   - FIBCALC_QUIET: Enable quiet mode (bool)
//   - FIBCALC_HEX: Enable hexadecimal output (bool)
//   - FIBCALC_INTERACTIVE: Enable interactive REPL mode (bool)
//   - FIBCALC_NO_COLOR: Disable colored output (bool)
//   - FIBCALC_OUTPUT: Output file path (string)
//   - FIBCALC_CALIBRATION_PROFILE: Path to calibration profile (string)
func applyEnvOverrides(config *AppConfig, fs *flag.FlagSet) {
	applyNumericOverrides(config, fs)
	applyDurationOverrides(config, fs)
	applyStringOverrides(config, fs)
	applyBooleanOverrides(config, fs)
}

func applyNumericOverrides(config *AppConfig, fs *flag.FlagSet) {
	if !isFlagSet(fs, "n") {
		config.N = getEnvUint64("N", config.N)
	}
	if !isFlagSet(fs, "threshold") {
		config.Threshold = getEnvInt("THRESHOLD", config.Threshold)
	}
	if !isFlagSet(fs, "fft-threshold") {
		config.FFTThreshold = getEnvInt("FFT_THRESHOLD", config.FFTThreshold)
	}
	if !isFlagSet(fs, "strassen-threshold") {
		config.StrassenThreshold = getEnvInt("STRASSEN_THRESHOLD", config.StrassenThreshold)
	}
}

func applyDurationOverrides(config *AppConfig, fs *flag.FlagSet) {
	if !isFlagSet(fs, "timeout") {
		config.Timeout = getEnvDuration("TIMEOUT", config.Timeout)
	}
}

func applyStringOverrides(config *AppConfig, fs *flag.FlagSet) {
	if !isFlagSet(fs, "algo") {
		config.Algo = getEnvString("ALGO", config.Algo)
	}
	if !isFlagSet(fs, "port") {
		config.Port = getEnvString("PORT", config.Port)
	}
	if !isFlagSet(fs, "output") && !isFlagSet(fs, "o") {
		config.OutputFile = getEnvString("OUTPUT", config.OutputFile)
	}
	if !isFlagSet(fs, "calibration-profile") {
		config.CalibrationProfile = getEnvString("CALIBRATION_PROFILE", config.CalibrationProfile)
	}
}

func applyBooleanOverrides(config *AppConfig, fs *flag.FlagSet) {
	if !isFlagSet(fs, "server") {
		config.ServerMode = getEnvBool("SERVER", config.ServerMode)
	}
	if !isFlagSet(fs, "json") {
		config.JSONOutput = getEnvBool("JSON", config.JSONOutput)
	}
	if !isFlagSet(fs, "v") {
		config.Verbose = getEnvBool("VERBOSE", config.Verbose)
	}
	if !isFlagSet(fs, "d") && !isFlagSet(fs, "details") {
		config.Details = getEnvBool("DETAILS", config.Details)
	}
	if !isFlagSet(fs, "quiet") && !isFlagSet(fs, "q") {
		config.Quiet = getEnvBool("QUIET", config.Quiet)
	}
	if !isFlagSet(fs, "hex") {
		config.HexOutput = getEnvBool("HEX", config.HexOutput)
	}
	if !isFlagSet(fs, "interactive") {
		config.Interactive = getEnvBool("INTERACTIVE", config.Interactive)
	}
	if !isFlagSet(fs, "no-color") {
		config.NoColor = getEnvBool("NO_COLOR", config.NoColor)
	}
	if !isFlagSet(fs, "calibrate") {
		config.Calibrate = getEnvBool("CALIBRATE", config.Calibrate)
	}
	if !isFlagSet(fs, "auto-calibrate") {
		config.AutoCalibrate = getEnvBool("AUTO_CALIBRATE", config.AutoCalibrate)
	}
	if !isFlagSet(fs, "calculate") && !isFlagSet(fs, "c") {
		config.Concise = getEnvBool("CALCULATE", config.Concise)
	}
}
