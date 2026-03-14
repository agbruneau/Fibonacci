package config

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// Exhaustive Validation Tests
// ─────────────────────────────────────────────────────────────────────────────

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
			cfg := AppConfig{
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
			cfg := AppConfig{
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
			cfg := AppConfig{
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
			cfg := AppConfig{
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
	cfg := AppConfig{
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

// TestValidateCombinedErrors tests configs with multiple errors.
func TestValidateCombinedErrors(t *testing.T) {
	t.Parallel()
	algos := []string{"fast"}

	// Multiple issues - validation should catch at least one
	cfg := AppConfig{
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

// ─────────────────────────────────────────────────────────────────────────────
// ParseConfig Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestParseConfigDefaults tests that default values are correctly set.
func TestParseConfigDefaults(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	algos := []string{"fast", "matrix"}

	cfg, err := ParseConfig("test", []string{}, &buf, algos)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify defaults
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

	cfg, err := ParseConfig("test", args, &buf, algos)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify all values
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

	cfg, err := ParseConfig("test", []string{"-details"}, &buf, algos)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !cfg.Details {
		t.Error("Details should be true when -details is used")
	}
}

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
			var buf bytes.Buffer
			_, err := ParseConfig("test", tc.args, &buf, algos)
			if err == nil {
				t.Error("Expected error for invalid flags")
			}
		})
	}
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
			var buf bytes.Buffer
			cfg, err := ParseConfig("test", []string{"-algo", tc.input}, &buf, algos)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if cfg.Algo != tc.expected {
				t.Errorf("Algo: expected '%s', got '%s'", tc.expected, cfg.Algo)
			}
		})
	}
}

// TestParseConfigValidationErrors tests that validation errors are reported.
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
			var buf bytes.Buffer
			_, err := ParseConfig("test", tc.args, &buf, algos)
			if err == nil {
				t.Error("Expected validation error")
			}
			if tc.errorContains != "" && !strings.Contains(buf.String(), tc.errorContains) {
				t.Errorf("Expected error containing '%s', got: %s", tc.errorContains, buf.String())
			}
		})
	}
}

// TestParseConfigLargeN tests parsing of very large N values.
func TestParseConfigLargeN(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	algos := []string{"fast"}

	// Test with max uint64
	cfg, err := ParseConfig("test", []string{"-n", "18446744073709551615"}, &buf, algos)
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

	cfg, err := ParseConfig("test", []string{"-n", "0"}, &buf, algos)
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
			var buf bytes.Buffer
			cfg, err := ParseConfig("test", []string{"-timeout", tc.input}, &buf, algos)
			if err != nil {
				t.Fatalf("Unexpected error for timeout '%s': %v", tc.input, err)
			}
			if cfg.Timeout != tc.expected {
				t.Errorf("Timeout: expected %v, got %v", tc.expected, cfg.Timeout)
			}
		})
	}
}

// TestParseConfigHelpFlag tests that -h/-help returns flag.ErrHelp.
func TestParseConfigHelpFlag(t *testing.T) {
	t.Parallel()
	algos := []string{"fast"}

	helpFlags := []string{"-h", "-help", "--help"}

	for _, flag := range helpFlags {
		t.Run(flag, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			_, err := ParseConfig("test", []string{flag}, &buf, algos)
			// flag.ErrHelp is returned for help flags
			if err == nil {
				t.Error("Expected error for help flag")
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Environment Variable Tests
// ─────────────────────────────────────────────────────────────────────────────


// ─────────────────────────────────────────────────────────────────────────────
// Boundary Value Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestParseConfigBoundaryValues tests edge cases for numeric values.
func TestParseConfigBoundaryValues(t *testing.T) {
	t.Parallel()
	algos := []string{"fast"}

	testCases := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{"ThresholdZero", []string{"-threshold", "0"}, false},
		{"FFTThresholdZero", []string{"-fft-threshold", "0"}, false},
		{"StrassenThresholdZero", []string{"-strassen-threshold", "0"}, false},
		{"NZero", []string{"-n", "0"}, false},
		{"TimeoutMinimum", []string{"-timeout", "1ns"}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			_, err := ParseConfig("test", tc.args, &buf, algos)
			if tc.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
