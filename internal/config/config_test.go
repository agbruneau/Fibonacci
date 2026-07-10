package config_test

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	config "github.com/agbruneau/FibGo/internal/config"
	apperrors "github.com/agbruneau/FibGo/internal/errors"
)

// ─────────────────────────────────────────────────────────────────────────────
// ParseConfig: defaults, flag parsing, aliases
// ─────────────────────────────────────────────────────────────────────────────

// TestParseConfigDefaults tests that default values are correctly set.
func TestParseConfigDefaults(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	algos := []string{"fast", "matrix"}

	cfg, err := config.ParseConfig("test", []string{}, &buf, algos)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cfg.N != 100000000 {
		t.Errorf("Default N: expected 100000000, got %d", cfg.N)
	}
	if cfg.Verbose {
		t.Error("Default Verbose should be false")
	}
	if cfg.Details {
		t.Error("Default Details should be false")
	}
	if cfg.Timeout != 5*time.Minute {
		t.Errorf("Default Timeout: expected 5m, got %v", cfg.Timeout)
	}
	if cfg.Algo != "all" {
		t.Errorf("Default Algo: expected 'all', got '%s'", cfg.Algo)
	}
	if cfg.Threshold != 0 {
		t.Errorf("Default Threshold: expected 0, got %d", cfg.Threshold)
	}
	if cfg.FFTThreshold != 0 {
		t.Errorf("Default FFTThreshold: expected 0, got %d", cfg.FFTThreshold)
	}
	if cfg.StrassenThreshold != 0 {
		t.Errorf("Default StrassenThreshold: expected 0, got %d", cfg.StrassenThreshold)
	}
	if cfg.Calibrate {
		t.Error("Default Calibrate should be false")
	}
	if cfg.AutoCalibrate {
		t.Error("Default AutoCalibrate should be false")
	}
}

// TestParseConfigAllFlags tests parsing of all flags.
func TestParseConfigAllFlags(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	algos := []string{"fast", "matrix", "fft"}

	args := []string{
		"-n", "12345",
		"-v",
		"-d",
		"-timeout", "10m",
		"-algo", "matrix",
		"-threshold", "8192",
		"-fft-threshold", "2000000",
		"-strassen-threshold", "512",
		"-calibrate",
		"-auto-calibrate",
		"-calibration-profile", "/path/to/profile.json",
	}

	cfg, err := config.ParseConfig("test", args, &buf, algos)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cfg.N != 12345 {
		t.Errorf("N: expected 12345, got %d", cfg.N)
	}
	if !cfg.Verbose {
		t.Error("Verbose should be true")
	}
	if !cfg.Details {
		t.Error("Details should be true")
	}
	if cfg.Timeout != 10*time.Minute {
		t.Errorf("Timeout: expected 10m, got %v", cfg.Timeout)
	}
	if cfg.Algo != "matrix" {
		t.Errorf("Algo: expected 'matrix', got '%s'", cfg.Algo)
	}
	if cfg.Threshold != 8192 {
		t.Errorf("Threshold: expected 8192, got %d", cfg.Threshold)
	}
	if cfg.FFTThreshold != 2000000 {
		t.Errorf("FFTThreshold: expected 2000000, got %d", cfg.FFTThreshold)
	}
	if cfg.StrassenThreshold != 512 {
		t.Errorf("StrassenThreshold: expected 512, got %d", cfg.StrassenThreshold)
	}
	if !cfg.Calibrate {
		t.Error("Calibrate should be true")
	}
	if !cfg.AutoCalibrate {
		t.Error("AutoCalibrate should be true")
	}
	if cfg.CalibrationProfile != "/path/to/profile.json" {
		t.Errorf("CalibrationProfile: expected '/path/to/profile.json', got '%s'", cfg.CalibrationProfile)
	}
}

// TestParseConfigDetailsAlias tests the -details alias for -d.
func TestParseConfigDetailsAlias(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	algos := []string{"fast"}

	cfg, err := config.ParseConfig("test", []string{"-details"}, &buf, algos)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !cfg.Details {
		t.Error("Details should be true when -details is used")
	}
}

// TestVerboseFlagAlias covers the -v/-verbose aliases.
func TestVerboseFlagAlias(t *testing.T) {
	t.Parallel()
	availableAlgos := []string{"fast", "matrix", "fft"}

	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"short form -v", []string{"-v"}, true},
		{"long form --verbose", []string{"-verbose"}, true},
		{"no verbose flag", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := config.ParseConfig("test", tt.args, io.Discard, availableAlgos)
			if err != nil {
				t.Fatalf("ParseConfig failed: %v", err)
			}
			if cfg.Verbose != tt.want {
				t.Errorf("Verbose = %v, want %v", cfg.Verbose, tt.want)
			}
		})
	}
}

// TestMachineFlag covers the --machine flag (machine-readable, no ANSI).
func TestMachineFlag(t *testing.T) {
	t.Parallel()
	cfg, err := config.ParseConfig("fibcalc", []string{"--machine"}, io.Discard, []string{"fast", "matrix", "fft"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !cfg.MachineOutput {
		t.Error("Expected MachineOutput true")
	}
}

// TestTUIFlag covers the --tui flag: CLI, default and env-override forms.
func TestTUIFlag(t *testing.T) {
	availableAlgos := []string{"fast", "matrix", "fft"}

	t.Run("--tui flag", func(t *testing.T) {
		t.Parallel()
		cfg, err := config.ParseConfig("test", []string{"-tui"}, io.Discard, availableAlgos)
		if err != nil {
			t.Fatalf("ParseConfig failed: %v", err)
		}
		if !cfg.TUI {
			t.Error("TUI should be true when --tui flag is set")
		}
	})

	t.Run("default TUI is false", func(t *testing.T) {
		t.Parallel()
		cfg, err := config.ParseConfig("test", []string{}, io.Discard, availableAlgos)
		if err != nil {
			t.Fatalf("ParseConfig failed: %v", err)
		}
		if cfg.TUI {
			t.Error("TUI should be false by default")
		}
	})

	// Not t.Parallel(): mutates the process environment (FIBCALC_TUI).
	t.Run("FIBCALC_TUI env override", func(t *testing.T) {
		t.Setenv("FIBCALC_TUI", "true")

		cfg, err := config.ParseConfig("test", []string{}, io.Discard, availableAlgos)
		if err != nil {
			t.Fatalf("ParseConfig failed: %v", err)
		}
		if !cfg.TUI {
			t.Error("TUI should be true when FIBCALC_TUI=true")
		}
	})
}

// TestParseConfigAlgoCaseInsensitivity tests that algo is lowercased.
func TestParseConfigAlgoCaseInsensitivity(t *testing.T) {
	t.Parallel()
	algos := []string{"fast", "matrix"}

	testCases := []struct {
		input    string
		expected string
	}{
		{"FAST", "fast"},
		{"Fast", "fast"},
		{"fAsT", "fast"},
		{"MATRIX", "matrix"},
		{"Matrix", "matrix"},
		{"ALL", "all"},
		{"All", "all"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			cfg, err := config.ParseConfig("test", []string{"-algo", tc.input}, &buf, algos)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if cfg.Algo != tc.expected {
				t.Errorf("Algo: expected '%s', got '%s'", tc.expected, cfg.Algo)
			}
		})
	}
}

// TestParseConfigLargeN tests parsing of very large N values.
func TestParseConfigLargeN(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	algos := []string{"fast"}

	cfg, err := config.ParseConfig("test", []string{"-n", "18446744073709551615"}, &buf, algos)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cfg.N != 18446744073709551615 {
		t.Errorf("N: expected max uint64, got %d", cfg.N)
	}
}

// TestParseConfigZeroN tests that N=0 is valid.
func TestParseConfigZeroN(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	algos := []string{"fast"}

	cfg, err := config.ParseConfig("test", []string{"-n", "0"}, &buf, algos)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cfg.N != 0 {
		t.Errorf("N: expected 0, got %d", cfg.N)
	}
}

// TestParseConfigTimeoutFormats tests various timeout format strings.
func TestParseConfigTimeoutFormats(t *testing.T) {
	t.Parallel()
	algos := []string{"fast"}

	testCases := []struct {
		input    string
		expected time.Duration
	}{
		{"1ns", 1 * time.Nanosecond},
		{"1s", 1 * time.Second},
		{"30s", 30 * time.Second},
		{"1m", 1 * time.Minute},
		{"5m", 5 * time.Minute},
		{"1h", 1 * time.Hour},
		{"1m30s", 90 * time.Second},
		{"1h30m", 90 * time.Minute},
		{"500ms", 500 * time.Millisecond},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			cfg, err := config.ParseConfig("test", []string{"-timeout", tc.input}, &buf, algos)
			if err != nil {
				t.Fatalf("Unexpected error for timeout '%s': %v", tc.input, err)
			}
			if cfg.Timeout != tc.expected {
				t.Errorf("Timeout: expected %v, got %v", tc.expected, cfg.Timeout)
			}
		})
	}
}

// TestParseConfigBoundaryValues tests edge cases for numeric flag values,
// each an explicit CLI zero (as opposed to an unset/default zero).
func TestParseConfigBoundaryValues(t *testing.T) {
	t.Parallel()
	algos := []string{"fast"}

	testCases := []struct {
		name string
		args []string
	}{
		{"ThresholdZero", []string{"-threshold", "0"}},
		{"FFTThresholdZero", []string{"-fft-threshold", "0"}},
		{"StrassenThresholdZero", []string{"-strassen-threshold", "0"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			if _, err := config.ParseConfig("test", tc.args, &buf, algos); err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ParseConfig: error handling
// ─────────────────────────────────────────────────────────────────────────────

// TestParseConfigInvalidFlags tests handling of invalid flags.
func TestParseConfigInvalidFlags(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		args []string
	}{
		{"UnknownFlag", []string{"-unknown"}},
		{"InvalidNValue", []string{"-n", "notanumber"}},
		{"InvalidTimeout", []string{"-timeout", "invalid"}},
		{"InvalidThreshold", []string{"-threshold", "abc"}},
		{"MissingFlagValue", []string{"-n"}},
	}

	algos := []string{"fast"}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			_, err := config.ParseConfig("test", tc.args, &buf, algos)
			if err == nil {
				t.Error("Expected error for invalid flags")
			}
		})
	}
}

// TestParseConfigValidationErrors tests that validation errors are reported,
// including to the error writer, through the full ParseConfig pipeline.
func TestParseConfigValidationErrors(t *testing.T) {
	t.Parallel()
	algos := []string{"fast"}

	testCases := []struct {
		name          string
		args          []string
		errorContains string
	}{
		{
			"InvalidAlgo",
			[]string{"-algo", "nonexistent"},
			"unrecognized algorithm",
		},
		{
			"NegativeThreshold",
			[]string{"-threshold", "-1"},
			"", // Just needs to error
		},
		{
			"NegativeFFTThreshold",
			[]string{"-fft-threshold", "-1"},
			"",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			_, err := config.ParseConfig("test", tc.args, &buf, algos)
			if err == nil {
				t.Error("Expected validation error")
			}
			if tc.errorContains != "" && !strings.Contains(buf.String(), tc.errorContains) {
				t.Errorf("Expected error containing '%s', got: %s", tc.errorContains, buf.String())
			}
		})
	}
}

// TestParseConfig_ValidationFailureReturnsTypedError verifies (APP-14) that
// ParseConfig propagates Validate's typed ConfigError rather than replacing
// it with a generic error.
func TestParseConfig_ValidationFailureReturnsTypedError(t *testing.T) {
	t.Parallel()
	_, err := config.ParseConfig("fibcalc", []string{"-algo", "invalid"}, io.Discard, []string{"fast", "matrix", "fft"})
	var cfgErr apperrors.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Errorf("expected error chain to contain apperrors.ConfigError, got %T: %v", err, err)
	}
}

// TestParseConfigHelpFlag tests that -h/-help returns an error (flag.ErrHelp).
func TestParseConfigHelpFlag(t *testing.T) {
	t.Parallel()
	algos := []string{"fast"}

	helpFlags := []string{"-h", "-help", "--help"}

	for _, flag := range helpFlags {
		t.Run(flag, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			_, err := config.ParseConfig("test", []string{flag}, &buf, algos)
			if err == nil {
				t.Error("Expected error for help flag")
			}
		})
	}
}

// TestParseConfig_UsageOutput_NoColor pins the NO_COLOR contract for the
// flag usage text printed on -h: no ANSI escapes, header/flags present,
// zero/false defaults hidden. Exercised through the public ParseConfig
// entry point (which wires the custom usage function internally) rather
// than the private setCustomUsage/registerFlags, so the assertion holds
// regardless of how the usage rendering internals are structured.
func TestParseConfig_UsageOutput_NoColor(t *testing.T) {
	// Not t.Parallel(): mutates the process environment (NO_COLOR).
	t.Setenv("NO_COLOR", "1")

	var buf bytes.Buffer
	_, err := config.ParseConfig("fibcalc-usage-test", []string{"-h"}, &buf, []string{"fast", "matrix"})
	if err == nil {
		t.Fatal("expected error for -h (flag.ErrHelp)")
	}

	out := buf.String()
	if strings.Contains(out, "\x1b[") {
		t.Error("usage output contains ANSI escape sequences despite NO_COLOR")
	}
	for _, want := range []string{
		"Fibonacci Calculator",
		"Usage:",
		"fibcalc-usage-test",
		"Flags:",
		"-algo",
		"(default all)",
		"(default 5m0s)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("usage output missing %q\noutput:\n%s", want, out)
		}
	}
	// Zero/false defaults are intentionally hidden to keep the listing readable.
	if strings.Contains(out, "(default false)") || strings.Contains(out, "(default 0)") {
		t.Error("usage output should not print zero/false defaults")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Environment variable overrides (priority: CLI > env > default)
// ─────────────────────────────────────────────────────────────────────────────

// TestParseConfig_EnvOverrides exercises every FIBCALC_* environment
// override (except FIBCALC_TUI, covered by TestTUIFlag) in isolation:
// when no corresponding flag is set, the env value must be reflected in
// the parsed AppConfig.
func TestParseConfig_EnvOverrides(t *testing.T) {
	algos := []string{"fast", "matrix", "fft"}

	tests := []struct {
		name   string
		envKey string
		envVal string
		check  func(config.AppConfig) bool
	}{
		{"N", "N", "999", func(c config.AppConfig) bool { return c.N == 999 }},
		{"Threshold", "THRESHOLD", "1111", func(c config.AppConfig) bool { return c.Threshold == 1111 }},
		{"FFTThreshold", "FFT_THRESHOLD", "2222", func(c config.AppConfig) bool { return c.FFTThreshold == 2222 }},
		{"StrassenThreshold", "STRASSEN_THRESHOLD", "3333", func(c config.AppConfig) bool { return c.StrassenThreshold == 3333 }},
		{"LastDigits", "LAST_DIGITS", "42", func(c config.AppConfig) bool { return c.LastDigits == 42 }},
		{"Timeout", "TIMEOUT", "10m", func(c config.AppConfig) bool { return c.Timeout == 10*time.Minute }},
		{"Algo", "ALGO", "fast", func(c config.AppConfig) bool { return c.Algo == "fast" }},
		{"OutputFile", "OUTPUT", "out.txt", func(c config.AppConfig) bool { return c.OutputFile == "out.txt" }},
		{"CalibrationProfile", "CALIBRATION_PROFILE", "prof.json", func(c config.AppConfig) bool { return c.CalibrationProfile == "prof.json" }},
		{"MemoryLimit", "MEMORY_LIMIT", "2G", func(c config.AppConfig) bool { return c.MemoryLimit == "2G" }},
		{"GCControl", "GC_CONTROL", "aggressive", func(c config.AppConfig) bool { return c.GCControl == "aggressive" }},
		{"Verbose", "VERBOSE", "yes", func(c config.AppConfig) bool { return c.Verbose }},
		{"Details", "DETAILS", "true", func(c config.AppConfig) bool { return c.Details }},
		{"Quiet", "QUIET", "true", func(c config.AppConfig) bool { return c.Quiet }},
		{"MachineOutput", "MACHINE_OUTPUT", "true", func(c config.AppConfig) bool { return c.MachineOutput }},
		{"Calibrate", "CALIBRATE", "true", func(c config.AppConfig) bool { return c.Calibrate }},
		{"AutoCalibrate", "AUTO_CALIBRATE", "true", func(c config.AppConfig) bool { return c.AutoCalibrate }},
		{"ShowValue", "CALCULATE", "yes", func(c config.AppConfig) bool { return c.ShowValue }},
	}

	for _, tt := range tests {
		// Not t.Parallel(): each subtest mutates a process-global FIBCALC_*
		// environment variable via t.Setenv. Go's test runner completes all
		// non-parallel tests (and their t.Cleanup env restoration) before any
		// t.Parallel() test body executes, so the parallel tests elsewhere in
		// this package that assume a clean environment (e.g.
		// TestParseConfigDefaults) are never exposed to a leaked override.
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(config.EnvPrefix+tt.envKey, tt.envVal)
			cfg, err := config.ParseConfig("test", []string{}, io.Discard, algos)
			if err != nil {
				t.Fatalf("ParseConfig: %v", err)
			}
			if !tt.check(cfg) {
				t.Errorf("%s%s=%q: override not applied, got %+v", config.EnvPrefix, tt.envKey, tt.envVal, cfg)
			}
		})
	}
}

// TestParseConfig_CLIPrecedenceOverEnv verifies that an explicitly-set CLI
// flag always wins over its FIBCALC_* environment counterpart.
func TestParseConfig_CLIPrecedenceOverEnv(t *testing.T) {
	algos := []string{"fast", "matrix", "fft"}

	tests := []struct {
		name    string
		envKey  string
		envVal  string
		cliArgs []string
		check   func(config.AppConfig) bool
	}{
		{"N", "N", "200", []string{"-n", "300"}, func(c config.AppConfig) bool { return c.N == 300 }},
		{"MemoryLimit", "MEMORY_LIMIT", "2G", []string{"-memory-limit", "512M"}, func(c config.AppConfig) bool { return c.MemoryLimit == "512M" }},
	}

	for _, tt := range tests {
		// Not t.Parallel(): mutates the process environment (see
		// TestParseConfig_EnvOverrides for the ordering argument).
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(config.EnvPrefix+tt.envKey, tt.envVal)
			cfg, err := config.ParseConfig("test", tt.cliArgs, io.Discard, algos)
			if err != nil {
				t.Fatalf("ParseConfig: %v", err)
			}
			if !tt.check(cfg) {
				t.Errorf("%s: CLI flag did not take precedence over env, got %+v", tt.name, cfg)
			}
		})
	}
}

// TestParseConfig_MalformedEnvRejected verifies (APP-09) that an
// explicitly-set but unparsable environment override is rejected with a
// structured config error, rather than silently falling back to the
// default (which could trigger an O(memory) calculation / OOM).
func TestParseConfig_MalformedEnvRejected(t *testing.T) {
	algos := []string{"fast", "matrix", "fft"}

	tests := []struct {
		name   string
		envKey string
		envVal string
	}{
		{"N", "N", "abc"},
		{"TIMEOUT", "TIMEOUT", "xyz"},
		{"THRESHOLD", "THRESHOLD", "notanint"},
		{"FFT_THRESHOLD", "FFT_THRESHOLD", "1.5"},
		{"STRASSEN_THRESHOLD", "STRASSEN_THRESHOLD", "??"},
		{"VERBOSE", "VERBOSE", "notabool"},
	}

	for _, tt := range tests {
		// Not t.Parallel(): mutates the process environment (see
		// TestParseConfig_EnvOverrides for the ordering argument).
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(config.EnvPrefix+tt.envKey, tt.envVal)
			_, err := config.ParseConfig("test", []string{}, io.Discard, algos)
			if err == nil {
				t.Fatalf("expected error for %s%s=%q, got nil", config.EnvPrefix, tt.envKey, tt.envVal)
			}
			var cfgErr apperrors.ConfigError
			if !errors.As(err, &cfgErr) {
				t.Errorf("expected error chain to contain apperrors.ConfigError, got %T: %v", err, err)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// AppConfig.Validate
// ─────────────────────────────────────────────────────────────────────────────

// base returns a minimally-valid AppConfig for targeted validation tests.
func base() config.AppConfig {
	return config.AppConfig{Timeout: 1 * time.Second, Threshold: 10, FFTThreshold: 10, Algo: "fast"}
}

// TestValidateTimeout tests all timeout validation scenarios.
func TestValidateTimeout(t *testing.T) {
	t.Parallel()
	algos := []string{"fast", "matrix"}

	testCases := []struct {
		name        string
		timeout     time.Duration
		expectError bool
	}{
		{"ZeroTimeout", 0, true},
		{"NegativeTimeout", -1 * time.Second, true},
		{"MinPositiveTimeout", 1 * time.Nanosecond, false},
		{"OneSecondTimeout", 1 * time.Second, false},
		{"OneMinuteTimeout", 1 * time.Minute, false},
		{"OneHourTimeout", 1 * time.Hour, false},
		{"VeryLargeTimeout", 24 * time.Hour, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := config.AppConfig{
				Timeout:      tc.timeout,
				Threshold:    100,
				FFTThreshold: 100,
				Algo:         "fast",
			}

			err := cfg.Validate(algos)
			if tc.expectError && err == nil {
				t.Error("Expected validation error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

// TestValidateThreshold tests all threshold validation scenarios.
func TestValidateThreshold(t *testing.T) {
	t.Parallel()
	algos := []string{"fast", "matrix"}

	testCases := []struct {
		name        string
		threshold   int
		expectError bool
	}{
		{"NegativeThreshold", -1, true},
		{"LargeNegativeThreshold", -1000000, true},
		{"ZeroThreshold", 0, false},
		{"SmallThreshold", 1, false},
		{"DefaultThreshold", 4096, false},
		{"LargeThreshold", 1000000, false},
		{"VeryLargeThreshold", 2147483647, false}, // Max int32
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := config.AppConfig{
				Timeout:      time.Minute,
				Threshold:    tc.threshold,
				FFTThreshold: 100,
				Algo:         "fast",
			}

			err := cfg.Validate(algos)
			if tc.expectError && err == nil {
				t.Error("Expected validation error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

// TestValidateFFTThreshold tests FFT threshold validation scenarios.
func TestValidateFFTThreshold(t *testing.T) {
	t.Parallel()
	algos := []string{"fast", "matrix"}

	testCases := []struct {
		name         string
		fftThreshold int
		expectError  bool
	}{
		{"NegativeFFTThreshold", -1, true},
		{"LargeNegativeFFTThreshold", -1000000, true},
		{"ZeroFFTThreshold", 0, false},
		{"SmallFFTThreshold", 1, false},
		{"DefaultFFTThreshold", 500000, false},
		{"LargeFFTThreshold", 10000000, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := config.AppConfig{
				Timeout:      time.Minute,
				Threshold:    100,
				FFTThreshold: tc.fftThreshold,
				Algo:         "fast",
			}

			err := cfg.Validate(algos)
			if tc.expectError && err == nil {
				t.Error("Expected validation error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

// TestValidateAlgorithm tests all algorithm validation scenarios.
func TestValidateAlgorithm(t *testing.T) {
	t.Parallel()
	algos := []string{"fast", "matrix", "fft"}

	testCases := []struct {
		name        string
		algo        string
		expectError bool
	}{
		{"AllAlgo", "all", false},
		{"FastAlgo", "fast", false},
		{"MatrixAlgo", "matrix", false},
		{"FFTAlgo", "fft", false},
		{"UnknownAlgo", "unknown", true},
		{"EmptyAlgo", "", true},
		{"CaseSensitive", "FAST", true}, // Note: ParseConfig lowercases
		{"PartialMatch", "fas", true},
		{"ExtraChars", "fast ", true},
		{"InvalidChars", "fast!", true},
		{"Numeric", "123", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := config.AppConfig{
				Timeout:      time.Minute,
				Threshold:    100,
				FFTThreshold: 100,
				Algo:         tc.algo,
			}

			err := cfg.Validate(algos)
			if tc.expectError && err == nil {
				t.Error("Expected validation error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

// TestValidateEmptyAvailableAlgos tests validation with empty algo list.
func TestValidateEmptyAvailableAlgos(t *testing.T) {
	t.Parallel()
	cfg := config.AppConfig{
		Timeout:      time.Minute,
		Threshold:    100,
		FFTThreshold: 100,
		Algo:         "all",
	}

	// "all" should be valid even with empty available algos
	err := cfg.Validate([]string{})
	if err != nil {
		t.Errorf("'all' should be valid regardless of available algos: %v", err)
	}

	// Specific algo should fail
	cfg.Algo = "fast"
	err = cfg.Validate([]string{})
	if err == nil {
		t.Error("Expected error for specific algo with empty available list")
	}
}

// TestValidateCombinedErrors tests configs with multiple simultaneous errors.
func TestValidateCombinedErrors(t *testing.T) {
	t.Parallel()
	algos := []string{"fast"}

	cfg := config.AppConfig{
		Timeout:      0,             // Invalid
		Threshold:    -1,            // Invalid
		FFTThreshold: -1,            // Invalid
		Algo:         "nonexistent", // Invalid
	}

	err := cfg.Validate(algos)
	if err == nil {
		t.Error("Expected validation error for config with multiple issues")
	}
}

func TestValidate_RejectsNegativeLastDigits(t *testing.T) {
	t.Parallel()
	algos := []string{"fast", "matrix"}

	t.Run("negative is rejected", func(t *testing.T) {
		t.Parallel()
		c := base()
		c.LastDigits = -5
		if err := c.Validate(algos); err == nil {
			t.Error("Expected error for negative LastDigits")
		}
	})

	t.Run("zero is valid (disabled)", func(t *testing.T) {
		t.Parallel()
		c := base()
		c.LastDigits = 0
		if err := c.Validate(algos); err != nil {
			t.Errorf("LastDigits=0 should be valid (disabled), got: %v", err)
		}
	})

	t.Run("positive is valid", func(t *testing.T) {
		t.Parallel()
		c := base()
		c.LastDigits = 10
		if err := c.Validate(algos); err != nil {
			t.Errorf("LastDigits=10 should be valid, got: %v", err)
		}
	})
}

func TestValidate_RejectsNegativeStrassen(t *testing.T) {
	t.Parallel()
	algos := []string{"fast", "matrix"}

	t.Run("negative is rejected", func(t *testing.T) {
		t.Parallel()
		c := base()
		c.StrassenThreshold = -1
		if err := c.Validate(algos); err == nil {
			t.Error("Expected error for negative StrassenThreshold")
		}
	})

	t.Run("zero is valid (auto)", func(t *testing.T) {
		t.Parallel()
		c := base()
		c.StrassenThreshold = 0
		if err := c.Validate(algos); err != nil {
			t.Errorf("StrassenThreshold=0 should be valid (auto), got: %v", err)
		}
	})
}

func TestValidate_RejectsIncompatibleTUICombinations(t *testing.T) {
	t.Parallel()
	algos := []string{"fast", "matrix"}

	assertConfigError := func(t *testing.T, err error) {
		t.Helper()
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		var cfgErr apperrors.ConfigError
		if !errors.As(err, &cfgErr) {
			t.Errorf("expected apperrors.ConfigError, got %T: %v", err, err)
		}
	}

	t.Run("tui + last-digits rejected", func(t *testing.T) {
		t.Parallel()
		c := base()
		c.TUI = true
		c.LastDigits = 10
		assertConfigError(t, c.Validate(algos))
	})

	t.Run("tui + output rejected", func(t *testing.T) {
		t.Parallel()
		c := base()
		c.TUI = true
		c.OutputFile = "result.txt"
		assertConfigError(t, c.Validate(algos))
	})

	t.Run("tui alone is valid", func(t *testing.T) {
		t.Parallel()
		c := base()
		c.TUI = true
		if err := c.Validate(algos); err != nil {
			t.Errorf("tui alone should be valid, got: %v", err)
		}
	})

	t.Run("last-digits + output without tui is valid", func(t *testing.T) {
		t.Parallel()
		c := base()
		c.LastDigits = 10
		c.OutputFile = "result.txt"
		if err := c.Validate(algos); err != nil {
			t.Errorf("last-digits + output without tui should be valid, got: %v", err)
		}
	})
}

func TestValidate_RejectsUnknownGCControl(t *testing.T) {
	t.Parallel()
	algos := []string{"fast", "matrix"}

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"auto is valid", "auto", false},
		{"aggressive is valid", "aggressive", false},
		{"disabled is valid", "disabled", false},
		{"empty is tolerated (effective default)", "", false},
		{"unknown is rejected", "turbo", true},
		{"case mismatch is rejected", "Auto", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := base()
			c.GCControl = tt.value
			err := c.Validate(algos)
			if tt.wantErr && err == nil {
				t.Errorf("GCControl=%q: expected error, got nil", tt.value)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("GCControl=%q: expected no error, got: %v", tt.value, err)
			}
		})
	}
}

func TestValidate_RejectsUnknownCompletion(t *testing.T) {
	t.Parallel()
	algos := []string{"fast", "matrix"}

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty is valid", "", false},
		{"bash is valid", "bash", false},
		{"zsh is valid", "zsh", false},
		{"fish is valid", "fish", false},
		{"powershell is valid", "powershell", false},
		{"unknown is rejected", "tcsh", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := base()
			c.Completion = tt.value
			err := c.Validate(algos)
			if tt.wantErr && err == nil {
				t.Errorf("Completion=%q: expected error, got nil", tt.value)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Completion=%q: expected no error, got: %v", tt.value, err)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ApplyAdaptiveThresholds
// ─────────────────────────────────────────────────────────────────────────────

// TestApplyAdaptiveThresholds_ZeroDefaults verifies that zero values are
// replaced by heuristic estimates while non-zero overrides are preserved.
func TestApplyAdaptiveThresholds_ZeroDefaults(t *testing.T) {
	t.Parallel()

	cfg := config.AppConfig{
		Threshold:         0,
		FFTThreshold:      0,
		StrassenThreshold: 0,
	}
	got := config.ApplyAdaptiveThresholds(cfg)

	if got.Threshold <= 0 {
		t.Errorf("Threshold: expected a positive heuristic value when zero, got %d", got.Threshold)
	}
	if got.FFTThreshold <= 0 {
		t.Errorf("FFTThreshold: expected a positive heuristic value when zero, got %d", got.FFTThreshold)
	}
	if got.StrassenThreshold <= 0 {
		t.Errorf("StrassenThreshold: expected a positive heuristic value when zero, got %d", got.StrassenThreshold)
	}
}

// TestApplyAdaptiveThresholds_PreservesOverrides verifies that explicit
// non-zero values are passed through unchanged.
func TestApplyAdaptiveThresholds_PreservesOverrides(t *testing.T) {
	t.Parallel()

	cfg := config.AppConfig{
		Threshold:         1234,
		FFTThreshold:      567890,
		StrassenThreshold: 42,
	}
	got := config.ApplyAdaptiveThresholds(cfg)

	if got.Threshold != 1234 {
		t.Errorf("Threshold override not preserved: want 1234, got %d", got.Threshold)
	}
	if got.FFTThreshold != 567890 {
		t.Errorf("FFTThreshold override not preserved: want 567890, got %d", got.FFTThreshold)
	}
	if got.StrassenThreshold != 42 {
		t.Errorf("StrassenThreshold override not preserved: want 42, got %d", got.StrassenThreshold)
	}
}

// TestApplyAdaptiveThresholds_PartialOverride verifies that only zero-valued
// fields are adapted while explicit fields remain untouched.
func TestApplyAdaptiveThresholds_PartialOverride(t *testing.T) {
	t.Parallel()

	cfg := config.AppConfig{
		Threshold:         99, // explicit
		FFTThreshold:      0,  // zero → adapted
		StrassenThreshold: 77, // explicit
	}
	got := config.ApplyAdaptiveThresholds(cfg)

	if got.Threshold != 99 {
		t.Errorf("Threshold: want 99, got %d", got.Threshold)
	}
	if got.FFTThreshold <= 0 {
		t.Errorf("FFTThreshold: expected adapted value, got %d", got.FFTThreshold)
	}
	if got.StrassenThreshold != 77 {
		t.Errorf("StrassenThreshold: want 77, got %d", got.StrassenThreshold)
	}
}
