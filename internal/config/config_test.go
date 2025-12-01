package config

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

// TestParseConfig verifies the behavior of the command-line argument parser.
// It checks that valid arguments are correctly parsed into the AppConfig struct,
// and that invalid arguments or values trigger the expected errors.
func TestParseConfig(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectedN     uint64
		expectedAlgo  string
		expectError   bool
		errorContains string
	}{
		{
			name:         "Default values",
			args:         []string{},
			expectedN:    250000000,
			expectedAlgo: "all",
			expectError:  false,
		},
		{
			name:         "Valid flags",
			args:         []string{"-n", "100", "-algo", "matrix"},
			expectedN:    100,
			expectedAlgo: "matrix",
			expectError:  false,
		},
		{
			name:          "Invalid flag",
			args:          []string{"-invalid"},
			expectError:   true,
			errorContains: "flag provided but not defined",
		},
		{
			name:          "Invalid algorithm",
			args:          []string{"-algo", "invalid"},
			expectError:   true,
			errorContains: "unrecognized algorithm",
		},
	}

	availableAlgos := []string{"matrix", "fast"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			cfg, err := ParseConfig("test", tt.args, &buf, availableAlgos)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				if !strings.Contains(buf.String(), tt.errorContains) && !strings.Contains(err.Error(), tt.errorContains) {
					// Check both buffer (flag errors) and returned error (validation errors)
					t.Errorf("expected error containing %q, got output %q and error %v", tt.errorContains, buf.String(), err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if cfg.N != tt.expectedN {
					t.Errorf("expected N %d, got %d", tt.expectedN, cfg.N)
				}
				if cfg.Algo != tt.expectedAlgo {
					t.Errorf("expected Algo %s, got %s", tt.expectedAlgo, cfg.Algo)
				}
			}
		})
	}
}

// TestAppConfig_Validate ensures that the Validate method correctly identifies
// semantic errors in the configuration, such as negative thresholds or
// invalid timeout values.
func TestAppConfig_Validate(t *testing.T) {
	tests := []struct {
		name           string
		config         AppConfig
		availableAlgos []string
		expectError    bool
	}{
		{
			name: "Valid config",
			config: AppConfig{
				Timeout:      time.Minute,
				Threshold:    100,
				FFTThreshold: 100,
				Algo:         "matrix",
			},
			availableAlgos: []string{"matrix"},
			expectError:    false,
		},
		{
			name: "Invalid timeout",
			config: AppConfig{
				Timeout: 0,
			},
			expectError: true,
		},
		{
			name: "Negative threshold",
			config: AppConfig{
				Timeout:   time.Minute,
				Threshold: -1,
			},
			expectError: true,
		},
		{
			name: "Negative FFT threshold",
			config: AppConfig{
				Timeout:      time.Minute,
				Threshold:    100,
				FFTThreshold: -1,
			},
			expectError: true,
		},
		{
			name: "Invalid algorithm",
			config: AppConfig{
				Timeout:      time.Minute,
				Threshold:    100,
				FFTThreshold: 100,
				Algo:         "invalid",
			},
			availableAlgos: []string{"matrix"},
			expectError:    true,
		},
		{
			name: "Valid 'all' algorithm",
			config: AppConfig{
				Timeout:      time.Minute,
				Threshold:    100,
				FFTThreshold: 100,
				Algo:         "all",
			},
			availableAlgos: []string{"matrix"},
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate(tt.availableAlgos)
			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Environment Variable Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestEnvVariableOverrides verifies that environment variables are correctly
// applied when CLI flags are not explicitly set.
func TestEnvVariableOverrides(t *testing.T) {
	availableAlgos := []string{"matrix", "fast", "fft"}

	// Helper to set and clean environment variables
	setEnv := func(key, value string) func() {
		os.Setenv(EnvPrefix+key, value)
		return func() { os.Unsetenv(EnvPrefix + key) }
	}

	t.Run("N from environment", func(t *testing.T) {
		cleanup := setEnv("N", "12345")
		defer cleanup()

		var buf bytes.Buffer
		cfg, err := ParseConfig("test", []string{}, &buf, availableAlgos)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.N != 12345 {
			t.Errorf("expected N=12345, got %d", cfg.N)
		}
	})

	t.Run("PORT from environment", func(t *testing.T) {
		cleanup := setEnv("PORT", "9090")
		defer cleanup()

		var buf bytes.Buffer
		cfg, err := ParseConfig("test", []string{}, &buf, availableAlgos)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Port != "9090" {
			t.Errorf("expected Port=9090, got %s", cfg.Port)
		}
	})

	t.Run("ALGO from environment", func(t *testing.T) {
		cleanup := setEnv("ALGO", "fast")
		defer cleanup()

		var buf bytes.Buffer
		cfg, err := ParseConfig("test", []string{}, &buf, availableAlgos)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Algo != "fast" {
			t.Errorf("expected Algo=fast, got %s", cfg.Algo)
		}
	})

	t.Run("TIMEOUT from environment", func(t *testing.T) {
		cleanup := setEnv("TIMEOUT", "10m")
		defer cleanup()

		var buf bytes.Buffer
		cfg, err := ParseConfig("test", []string{}, &buf, availableAlgos)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := 10 * time.Minute
		if cfg.Timeout != expected {
			t.Errorf("expected Timeout=%v, got %v", expected, cfg.Timeout)
		}
	})

	t.Run("Boolean SERVER from environment", func(t *testing.T) {
		cleanup := setEnv("SERVER", "true")
		defer cleanup()

		var buf bytes.Buffer
		cfg, err := ParseConfig("test", []string{}, &buf, availableAlgos)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !cfg.ServerMode {
			t.Error("expected ServerMode=true, got false")
		}
	})

	t.Run("THRESHOLD from environment", func(t *testing.T) {
		cleanup := setEnv("THRESHOLD", "8192")
		defer cleanup()

		var buf bytes.Buffer
		cfg, err := ParseConfig("test", []string{}, &buf, availableAlgos)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Threshold != 8192 {
			t.Errorf("expected Threshold=8192, got %d", cfg.Threshold)
		}
	})

	t.Run("FFT_THRESHOLD from environment", func(t *testing.T) {
		cleanup := setEnv("FFT_THRESHOLD", "500000")
		defer cleanup()

		var buf bytes.Buffer
		cfg, err := ParseConfig("test", []string{}, &buf, availableAlgos)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.FFTThreshold != 500000 {
			t.Errorf("expected FFTThreshold=500000, got %d", cfg.FFTThreshold)
		}
	})
}

// TestCLIPriorityOverEnv verifies that CLI flags take precedence over
// environment variables when both are provided.
func TestCLIPriorityOverEnv(t *testing.T) {
	availableAlgos := []string{"matrix", "fast", "fft"}

	// Set environment variables
	os.Setenv(EnvPrefix+"N", "99999")
	os.Setenv(EnvPrefix+"PORT", "9090")
	os.Setenv(EnvPrefix+"ALGO", "matrix")
	defer func() {
		os.Unsetenv(EnvPrefix + "N")
		os.Unsetenv(EnvPrefix + "PORT")
		os.Unsetenv(EnvPrefix + "ALGO")
	}()

	// CLI flags should override environment variables
	var buf bytes.Buffer
	cfg, err := ParseConfig("test", []string{"-n", "100", "-port", "3000", "-algo", "fast"}, &buf, availableAlgos)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.N != 100 {
		t.Errorf("CLI should override ENV: expected N=100, got %d", cfg.N)
	}
	if cfg.Port != "3000" {
		t.Errorf("CLI should override ENV: expected Port=3000, got %s", cfg.Port)
	}
	if cfg.Algo != "fast" {
		t.Errorf("CLI should override ENV: expected Algo=fast, got %s", cfg.Algo)
	}
}

// TestEnvVariableTypes verifies that the environment variable parsing
// functions correctly handle different types and invalid values.
func TestEnvVariableTypes(t *testing.T) {
	t.Run("getEnvString", func(t *testing.T) {
		os.Setenv(EnvPrefix+"TEST_STRING", "hello")
		defer os.Unsetenv(EnvPrefix + "TEST_STRING")

		val := getEnvString("TEST_STRING", "default")
		if val != "hello" {
			t.Errorf("expected 'hello', got '%s'", val)
		}

		val = getEnvString("NONEXISTENT", "default")
		if val != "default" {
			t.Errorf("expected 'default', got '%s'", val)
		}
	})

	t.Run("getEnvUint64", func(t *testing.T) {
		os.Setenv(EnvPrefix+"TEST_UINT64", "12345")
		defer os.Unsetenv(EnvPrefix + "TEST_UINT64")

		val := getEnvUint64("TEST_UINT64", 0)
		if val != 12345 {
			t.Errorf("expected 12345, got %d", val)
		}

		// Test invalid value - should return default
		os.Setenv(EnvPrefix+"TEST_UINT64", "invalid")
		val = getEnvUint64("TEST_UINT64", 999)
		if val != 999 {
			t.Errorf("expected default 999 for invalid input, got %d", val)
		}
	})

	t.Run("getEnvInt", func(t *testing.T) {
		os.Setenv(EnvPrefix+"TEST_INT", "42")
		defer os.Unsetenv(EnvPrefix + "TEST_INT")

		val := getEnvInt("TEST_INT", 0)
		if val != 42 {
			t.Errorf("expected 42, got %d", val)
		}

		// Test negative value
		os.Setenv(EnvPrefix+"TEST_INT", "-10")
		val = getEnvInt("TEST_INT", 0)
		if val != -10 {
			t.Errorf("expected -10, got %d", val)
		}
	})

	t.Run("getEnvBool", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected bool
		}{
			{"true", true},
			{"TRUE", true},
			{"True", true},
			{"1", true},
			{"yes", true},
			{"YES", true},
			{"false", false},
			{"FALSE", false},
			{"0", false},
			{"no", false},
			{"NO", false},
		}

		for _, tc := range testCases {
			os.Setenv(EnvPrefix+"TEST_BOOL", tc.input)
			val := getEnvBool("TEST_BOOL", !tc.expected)
			if val != tc.expected {
				t.Errorf("getEnvBool(%q): expected %v, got %v", tc.input, tc.expected, val)
			}
		}
		os.Unsetenv(EnvPrefix + "TEST_BOOL")

		// Test default
		val := getEnvBool("NONEXISTENT", true)
		if val != true {
			t.Error("expected default true, got false")
		}
	})

	t.Run("getEnvDuration", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected time.Duration
		}{
			{"5m", 5 * time.Minute},
			{"30s", 30 * time.Second},
			{"1h30m", 90 * time.Minute},
			{"100ms", 100 * time.Millisecond},
		}

		for _, tc := range testCases {
			os.Setenv(EnvPrefix+"TEST_DURATION", tc.input)
			val := getEnvDuration("TEST_DURATION", 0)
			if val != tc.expected {
				t.Errorf("getEnvDuration(%q): expected %v, got %v", tc.input, tc.expected, val)
			}
		}
		os.Unsetenv(EnvPrefix + "TEST_DURATION")

		// Test invalid duration - should return default
		os.Setenv(EnvPrefix+"TEST_DURATION", "invalid")
		val := getEnvDuration("TEST_DURATION", time.Minute)
		if val != time.Minute {
			t.Errorf("expected default 1m for invalid input, got %v", val)
		}
		os.Unsetenv(EnvPrefix + "TEST_DURATION")
	})
}

// TestMultipleEnvVariables verifies that multiple environment variables
// can be set and applied together.
func TestMultipleEnvVariables(t *testing.T) {
	availableAlgos := []string{"matrix", "fast", "fft"}

	// Set multiple environment variables
	envVars := map[string]string{
		"N":             "50000",
		"ALGO":          "fft",
		"PORT":          "7777",
		"THRESHOLD":     "2048",
		"FFT_THRESHOLD": "100000",
		"SERVER":        "true",
		"JSON":          "yes",
		"VERBOSE":       "1",
	}

	for k, v := range envVars {
		os.Setenv(EnvPrefix+k, v)
	}
	defer func() {
		for k := range envVars {
			os.Unsetenv(EnvPrefix + k)
		}
	}()

	var buf bytes.Buffer
	cfg, err := ParseConfig("test", []string{}, &buf, availableAlgos)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.N != 50000 {
		t.Errorf("expected N=50000, got %d", cfg.N)
	}
	if cfg.Algo != "fft" {
		t.Errorf("expected Algo=fft, got %s", cfg.Algo)
	}
	if cfg.Port != "7777" {
		t.Errorf("expected Port=7777, got %s", cfg.Port)
	}
	if cfg.Threshold != 2048 {
		t.Errorf("expected Threshold=2048, got %d", cfg.Threshold)
	}
	if cfg.FFTThreshold != 100000 {
		t.Errorf("expected FFTThreshold=100000, got %d", cfg.FFTThreshold)
	}
	if !cfg.ServerMode {
		t.Error("expected ServerMode=true")
	}
	if !cfg.JSONOutput {
		t.Error("expected JSONOutput=true")
	}
	if !cfg.Verbose {
		t.Error("expected Verbose=true")
	}
}
