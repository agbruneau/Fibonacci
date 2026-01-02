package config

import (
	"bytes"
	"os"
	"testing"
	"time"
)

// TestToCalculationOptions tests the ToCalculationOptions method.
func TestToCalculationOptions(t *testing.T) {
	cfg := AppConfig{
		Threshold:         1234,
		FFTThreshold:      5678,
		StrassenThreshold: 9012,
	}

	opts := cfg.ToCalculationOptions()

	if opts.ParallelThreshold != 1234 {
		t.Errorf("expected ParallelThreshold=1234, got %d", opts.ParallelThreshold)
	}
	if opts.FFTThreshold != 5678 {
		t.Errorf("expected FFTThreshold=5678, got %d", opts.FFTThreshold)
	}
	if opts.StrassenThreshold != 9012 {
		t.Errorf("expected StrassenThreshold=9012, got %d", opts.StrassenThreshold)
	}
}

// TestParseConfigEnvironmentVariables tests environment variable parsing.
func TestParseConfigEnvironmentVariables(t *testing.T) {
	// Save and defer restore of environment
	oldEnv := make(map[string]string)
	envVars := []string{
		EnvPrefix + "N",
		EnvPrefix + "THRESHOLD",
		EnvPrefix + "FFT_THRESHOLD",
		EnvPrefix + "STRASSEN_THRESHOLD",
		EnvPrefix + "TIMEOUT",
		EnvPrefix + "ALGO",
		EnvPrefix + "PORT",
		EnvPrefix + "SERVER",
		EnvPrefix + "JSON",
		EnvPrefix + "VERBOSE",
		EnvPrefix + "QUIET",
		EnvPrefix + "HEX",
		EnvPrefix + "NO_COLOR",
	}

	for _, key := range envVars {
		if val, ok := os.LookupEnv(key); ok {
			oldEnv[key] = val
		}
	}

	defer func() {
		for _, key := range envVars {
			if val, ok := oldEnv[key]; ok {
				os.Setenv(key, val)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	t.Run("all environment variables set", func(t *testing.T) {
		os.Setenv(EnvPrefix+"N", "999")
		os.Setenv(EnvPrefix+"THRESHOLD", "1111")
		os.Setenv(EnvPrefix+"FFT_THRESHOLD", "2222")
		os.Setenv(EnvPrefix+"STRASSEN_THRESHOLD", "3333")
		os.Setenv(EnvPrefix+"TIMEOUT", "10m")
		os.Setenv(EnvPrefix+"ALGO", "fast")
		os.Setenv(EnvPrefix+"PORT", "9999")
		os.Setenv(EnvPrefix+"SERVER", "true")
		os.Setenv(EnvPrefix+"JSON", "1")
		os.Setenv(EnvPrefix+"VERBOSE", "yes")
		os.Setenv(EnvPrefix+"QUIET", "0")
		os.Setenv(EnvPrefix+"HEX", "false")
		os.Setenv(EnvPrefix+"NO_COLOR", "no")

		var buf bytes.Buffer
		cfg, err := ParseConfig("test", []string{}, &buf, []string{"fast", "matrix", "fft"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.N != 999 {
			t.Errorf("expected N=999, got %d", cfg.N)
		}
		if cfg.Threshold != 1111 {
			t.Errorf("expected Threshold=1111, got %d", cfg.Threshold)
		}
		if cfg.FFTThreshold != 2222 {
			t.Errorf("expected FFTThreshold=2222, got %d", cfg.FFTThreshold)
		}
		if cfg.StrassenThreshold != 3333 {
			t.Errorf("expected StrassenThreshold=3333, got %d", cfg.StrassenThreshold)
		}
		if cfg.Timeout != 10*time.Minute {
			t.Errorf("expected Timeout=10m, got %v", cfg.Timeout)
		}
		if cfg.Algo != "fast" {
			t.Errorf("expected Algo=fast, got %s", cfg.Algo)
		}
		if cfg.Port != "9999" {
			t.Errorf("expected Port=9999, got %s", cfg.Port)
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
	})

	t.Run("invalid environment values ignored", func(t *testing.T) {
		os.Setenv(EnvPrefix+"N", "notanumber")
		os.Setenv(EnvPrefix+"THRESHOLD", "invalid")
		os.Setenv(EnvPrefix+"TIMEOUT", "notaduration")

		var buf bytes.Buffer
		cfg, err := ParseConfig("test", []string{}, &buf, []string{"fast", "matrix", "fft"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should use defaults for invalid values
		if cfg.N != DefaultN {
			t.Errorf("expected default N=%d, got %d", DefaultN, cfg.N)
		}
		if cfg.Threshold != DefaultThreshold {
			t.Errorf("expected default Threshold=%d, got %d", DefaultThreshold, cfg.Threshold)
		}
		if cfg.Timeout != DefaultTimeout {
			t.Errorf("expected default Timeout=%v, got %v", DefaultTimeout, cfg.Timeout)
		}
	})
}

// TestGetEnvHelpers tests environment variable helper functions.
func TestGetEnvHelpers(t *testing.T) {
	// Save environment
	oldVal := os.Getenv(EnvPrefix + "TEST")
	defer func() {
		if oldVal != "" {
			os.Setenv(EnvPrefix+"TEST", oldVal)
		} else {
			os.Unsetenv(EnvPrefix + "TEST")
		}
	}()

	t.Run("getEnvString", func(t *testing.T) {
		os.Unsetenv(EnvPrefix + "TEST")
		if val := getEnvString("TEST", "default"); val != "default" {
			t.Errorf("expected default, got %s", val)
		}

		os.Setenv(EnvPrefix+"TEST", "custom")
		if val := getEnvString("TEST", "default"); val != "custom" {
			t.Errorf("expected custom, got %s", val)
		}
	})

	t.Run("getEnvUint64", func(t *testing.T) {
		os.Unsetenv(EnvPrefix + "TEST")
		if val := getEnvUint64("TEST", 100); val != 100 {
			t.Errorf("expected 100, got %d", val)
		}

		os.Setenv(EnvPrefix+"TEST", "200")
		if val := getEnvUint64("TEST", 100); val != 200 {
			t.Errorf("expected 200, got %d", val)
		}

		os.Setenv(EnvPrefix+"TEST", "invalid")
		if val := getEnvUint64("TEST", 100); val != 100 {
			t.Errorf("expected default 100 for invalid, got %d", val)
		}
	})

	t.Run("getEnvInt", func(t *testing.T) {
		os.Unsetenv(EnvPrefix + "TEST")
		if val := getEnvInt("TEST", 50); val != 50 {
			t.Errorf("expected 50, got %d", val)
		}

		os.Setenv(EnvPrefix+"TEST", "75")
		if val := getEnvInt("TEST", 50); val != 75 {
			t.Errorf("expected 75, got %d", val)
		}
	})

	t.Run("getEnvBool", func(t *testing.T) {
		os.Unsetenv(EnvPrefix + "TEST")
		if val := getEnvBool("TEST", true); !val {
			t.Error("expected true default")
		}

		testCases := []struct {
			env    string
			expect bool
		}{
			{"true", true},
			{"TRUE", true},
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
			os.Setenv(EnvPrefix+"TEST", tc.env)
			if val := getEnvBool("TEST", !tc.expect); val != tc.expect {
				t.Errorf("for %s expected %v, got %v", tc.env, tc.expect, val)
			}
		}
	})

	t.Run("getEnvDuration", func(t *testing.T) {
		os.Unsetenv(EnvPrefix + "TEST")
		if val := getEnvDuration("TEST", time.Minute); val != time.Minute {
			t.Errorf("expected 1m default, got %v", val)
		}

		os.Setenv(EnvPrefix+"TEST", "30s")
		if val := getEnvDuration("TEST", time.Minute); val != 30*time.Second {
			t.Errorf("expected 30s, got %v", val)
		}

		os.Setenv(EnvPrefix+"TEST", "invalid")
		if val := getEnvDuration("TEST", time.Minute); val != time.Minute {
			t.Errorf("expected default 1m for invalid, got %v", val)
		}
	})
}
