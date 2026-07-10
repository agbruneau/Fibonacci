package config

import (
	"bytes"
	"os"
	"testing"
	"time"
)

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
		EnvPrefix + "VERBOSE",
		EnvPrefix + "QUIET",
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
		os.Setenv(EnvPrefix+"VERBOSE", "yes")
		os.Setenv(EnvPrefix+"QUIET", "0")

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
		if !cfg.Verbose {
			t.Error("expected Verbose=true")
		}
	})

	t.Run("invalid environment values rejected", func(t *testing.T) {
		// A-09: an explicitly-set but unparsable override is a hard config
		// error rather than a silent fallback to the default (which could
		// trigger an O(memory) calculation / OOM).
		os.Setenv(EnvPrefix+"N", "notanumber")
		os.Setenv(EnvPrefix+"THRESHOLD", "invalid")
		os.Setenv(EnvPrefix+"TIMEOUT", "notaduration")

		var buf bytes.Buffer
		_, err := ParseConfig("test", []string{}, &buf, []string{"fast", "matrix", "fft"})
		if err == nil {
			t.Fatal("expected error for malformed env override, got nil")
		}
	})
}
