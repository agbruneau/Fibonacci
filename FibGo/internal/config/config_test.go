package config

import (
	"io"
	"os"
	"testing"
	"time"
)

func TestParseConfig(t *testing.T) {
	availableAlgos := []string{"fast", "matrix", "fft"}

	t.Run("DefaultValues", func(t *testing.T) {
		t.Parallel()
		args := []string{}
		cfg, err := ParseConfig("fibcalc", args, io.Discard, availableAlgos)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if cfg.N != 250000000 {
			t.Errorf("Expected default N 250000000, got %d", cfg.N)
		}
		if cfg.Algo != "all" {
			t.Errorf("Expected default Algo 'all', got %s", cfg.Algo)
		}
		if cfg.Timeout != 5*time.Minute {
			t.Errorf("Expected default Timeout 5m, got %v", cfg.Timeout)
		}
	})

	t.Run("ValidFlags", func(t *testing.T) {
		t.Parallel()
		args := []string{
			"-n", "100",
			"-algo", "fast",
			"-v",
			"-timeout", "10s",
			"-threshold", "5000",
			"-server",
			"-port", "9090",
		}
		cfg, err := ParseConfig("fibcalc", args, io.Discard, availableAlgos)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if cfg.N != 100 {
			t.Errorf("Expected N 100, got %d", cfg.N)
		}
		if cfg.Algo != "fast" {
			t.Errorf("Expected Algo 'fast', got %s", cfg.Algo)
		}
		if !cfg.Verbose {
			t.Error("Expected Verbose true")
		}
		if cfg.Timeout != 10*time.Second {
			t.Errorf("Expected Timeout 10s, got %v", cfg.Timeout)
		}
		if cfg.Threshold != 5000 {
			t.Errorf("Expected Threshold 5000, got %d", cfg.Threshold)
		}
		if !cfg.ServerMode {
			t.Error("Expected ServerMode true")
		}
		if cfg.Port != "9090" {
			t.Errorf("Expected Port 9090, got %s", cfg.Port)
		}
	})

	t.Run("EnvOverrides", func(t *testing.T) {
		// Set env vars
		env := map[string]string{
			"FIBCALC_N":                   "200",
			"FIBCALC_ALGO":                "matrix",
			"FIBCALC_SERVER":              "true",
			"FIBCALC_PORT":                "3000",
			"FIBCALC_TIMEOUT":             "2m",
			"FIBCALC_THRESHOLD":           "1024",
			"FIBCALC_FFT_THRESHOLD":       "5000",
			"FIBCALC_STRASSEN_THRESHOLD":  "128",
			"FIBCALC_VERBOSE":             "true",
			"FIBCALC_DETAILS":             "true",
			"FIBCALC_QUIET":               "true",
			"FIBCALC_HEX":                 "true",
			"FIBCALC_INTERACTIVE":         "true",
			"FIBCALC_NO_COLOR":            "true",
			"FIBCALC_CALIBRATE":           "true",
			"FIBCALC_AUTO_CALIBRATE":      "true",
			"FIBCALC_OUTPUT":              "out.txt",
			"FIBCALC_CALIBRATION_PROFILE": "prof.json",
			"FIBCALC_JSON":                "true",
		}

		for k, v := range env {
			os.Setenv(k, v)
		}
		defer func() {
			for k := range env {
				os.Unsetenv(k)
			}
		}()

		// No flags set, should take from env
		cfg, err := ParseConfig("fibcalc", []string{}, io.Discard, availableAlgos)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if cfg.N != 200 {
			t.Errorf("Expected N 200 from env, got %d", cfg.N)
		}
		if cfg.Algo != "matrix" {
			t.Errorf("Expected Algo 'matrix' from env, got %s", cfg.Algo)
		}
		if !cfg.ServerMode {
			t.Error("Expected ServerMode true from env")
		}
		if cfg.Port != "3000" {
			t.Errorf("Expected Port 3000, got %s", cfg.Port)
		}
		if cfg.Timeout != 2*time.Minute {
			t.Errorf("Expected Timeout 2m, got %v", cfg.Timeout)
		}
		if cfg.Threshold != 1024 {
			t.Errorf("Expected Threshold 1024, got %d", cfg.Threshold)
		}
		if cfg.FFTThreshold != 5000 {
			t.Errorf("Expected FFTThreshold 5000, got %d", cfg.FFTThreshold)
		}
		if cfg.StrassenThreshold != 128 {
			t.Errorf("Expected StrassenThreshold 128, got %d", cfg.StrassenThreshold)
		}
		if !cfg.Verbose {
			t.Error("Expected Verbose true")
		}
		if !cfg.Details {
			t.Error("Expected Details true")
		}
		if !cfg.Quiet {
			t.Error("Expected Quiet true")
		}
		if !cfg.HexOutput {
			t.Error("Expected HexOutput true")
		}
		if !cfg.Interactive {
			t.Error("Expected Interactive true")
		}
		if !cfg.NoColor {
			t.Error("Expected NoColor true")
		}
		if !cfg.Calibrate {
			t.Error("Expected Calibrate true")
		}
		if !cfg.AutoCalibrate {
			t.Error("Expected AutoCalibrate true")
		}
		if cfg.OutputFile != "out.txt" {
			t.Errorf("Expected OutputFile out.txt, got %s", cfg.OutputFile)
		}
		if cfg.CalibrationProfile != "prof.json" {
			t.Errorf("Expected CalibrationProfile prof.json, got %s", cfg.CalibrationProfile)
		}
		if !cfg.JSONOutput {
			t.Error("Expected JSONOutput true")
		}
	})

	t.Run("FlagPrecedenceOverEnv", func(t *testing.T) {
		os.Setenv("FIBCALC_N", "200")
		defer os.Unsetenv("FIBCALC_N")

		// Flag set explicitly
		cfg, err := ParseConfig("fibcalc", []string{"-n", "300"}, io.Discard, availableAlgos)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if cfg.N != 300 {
			t.Errorf("Expected N 300 from flag, got %d", cfg.N)
		}
	})

	t.Run("InvalidFlags", func(t *testing.T) {
		t.Parallel()
		// Unknown flag
		_, err := ParseConfig("fibcalc", []string{"-unknown"}, io.Discard, availableAlgos)
		if err == nil {
			t.Error("Expected error for unknown flag")
		}
	})

	t.Run("ValidationFailure", func(t *testing.T) {
		t.Parallel()
		// Invalid algorithm
		_, err := ParseConfig("fibcalc", []string{"-algo", "invalid"}, io.Discard, availableAlgos)
		if err == nil {
			t.Error("Expected error for invalid algorithm")
		}
	})
}

func TestConfigValidate(t *testing.T) {
	t.Parallel()
	availableAlgos := []string{"fast", "matrix"}

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()
		c := AppConfig{Timeout: 1 * time.Second, Threshold: 10, FFTThreshold: 10, Algo: "fast"}
		if err := c.Validate(availableAlgos); err != nil {
			t.Errorf("Unexpected validation error: %v", err)
		}
	})

	t.Run("InvalidTimeout", func(t *testing.T) {
		t.Parallel()
		c := AppConfig{Timeout: 0, Threshold: 10, FFTThreshold: 10, Algo: "fast"}
		if err := c.Validate(availableAlgos); err == nil {
			t.Error("Expected error for zero timeout")
		}
	})

	t.Run("InvalidThreshold", func(t *testing.T) {
		t.Parallel()
		c := AppConfig{Timeout: 1 * time.Second, Threshold: -1, FFTThreshold: 10, Algo: "fast"}
		if err := c.Validate(availableAlgos); err == nil {
			t.Error("Expected error for negative threshold")
		}
	})

	t.Run("InvalidFFTThreshold", func(t *testing.T) {
		t.Parallel()
		c := AppConfig{Timeout: 1 * time.Second, Threshold: 10, FFTThreshold: -1, Algo: "fast"}
		if err := c.Validate(availableAlgos); err == nil {
			t.Error("Expected error for negative FFT threshold")
		}
	})

	t.Run("InvalidAlgo", func(t *testing.T) {
		t.Parallel()
		c := AppConfig{Timeout: 1 * time.Second, Threshold: 10, FFTThreshold: 10, Algo: "unknown"}
		if err := c.Validate(availableAlgos); err == nil {
			t.Error("Expected error for unknown algorithm")
		}
	})

	t.Run("AlgoAll", func(t *testing.T) {
		t.Parallel()
		c := AppConfig{Timeout: 1 * time.Second, Threshold: 10, FFTThreshold: 10, Algo: "all"}
		if err := c.Validate(availableAlgos); err != nil {
			t.Error("Algo 'all' should be valid")
		}
	})
}

func TestEnvHelpers(t *testing.T) {
	prefix := EnvPrefix

	t.Run("getEnvString", func(t *testing.T) {
		key := "TEST_STRING"
		os.Setenv(prefix+key, "value")
		defer os.Unsetenv(prefix + key)
		if val := getEnvString(key, "default"); val != "value" {
			t.Errorf("Expected 'value', got '%s'", val)
		}
		if val := getEnvString("NONEXISTENT", "default"); val != "default" {
			t.Errorf("Expected 'default', got '%s'", val)
		}
	})

	t.Run("getEnvUint64", func(t *testing.T) {
		key := "TEST_UINT"
		os.Setenv(prefix+key, "123")
		defer os.Unsetenv(prefix + key)
		if val := getEnvUint64(key, 0); val != 123 {
			t.Errorf("Expected 123, got %d", val)
		}
		// Invalid
		os.Setenv(prefix+"INVALID", "abc")
		defer os.Unsetenv(prefix + "INVALID")
		if val := getEnvUint64("INVALID", 999); val != 999 {
			t.Errorf("Expected default 999 for invalid input, got %d", val)
		}
	})

	t.Run("getEnvInt", func(t *testing.T) {
		key := "TEST_INT"
		os.Setenv(prefix+key, "-123")
		defer os.Unsetenv(prefix + key)
		if val := getEnvInt(key, 0); val != -123 {
			t.Errorf("Expected -123, got %d", val)
		}
	})

	t.Run("getEnvBool", func(t *testing.T) {
		key := "TEST_BOOL"
		os.Setenv(prefix+key, "true")
		defer os.Unsetenv(prefix + key)
		if val := getEnvBool(key, false); !val {
			t.Error("Expected true")
		}

		os.Setenv(prefix+key, "0")
		if val := getEnvBool(key, true); val {
			t.Error("Expected false for '0'")
		}

		os.Setenv(prefix+key, "invalid")
		if val := getEnvBool(key, true); !val {
			t.Error("Expected default true for invalid input")
		}
	})

	t.Run("getEnvDuration", func(t *testing.T) {
		key := "TEST_DURATION"
		os.Setenv(prefix+key, "1h")
		defer os.Unsetenv(prefix + key)
		if val := getEnvDuration(key, 0); val != time.Hour {
			t.Errorf("Expected 1h, got %v", val)
		}
	})
}
