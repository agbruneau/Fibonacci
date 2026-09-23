package config

import (
	"bytes"
	"testing"
	"time"
)

// TestParseConfigEnvironmentVariables tests environment variable parsing.
//
// t.Setenv throughout (audit TST-02): it restores the previous value — set or
// unset — automatically, with no hand-written save/restore preamble, and it
// refuses to run in a parallel test, so the
// mutation can never overlap a test that reads the same variables.
func TestParseConfigEnvironmentVariables(t *testing.T) {
	t.Run("all environment variables set", func(t *testing.T) {
		t.Setenv(EnvPrefix+"N", "999")
		t.Setenv(EnvPrefix+"THRESHOLD", "1111")
		t.Setenv(EnvPrefix+"FFT_THRESHOLD", "2222")
		t.Setenv(EnvPrefix+"STRASSEN_THRESHOLD", "3333")
		t.Setenv(EnvPrefix+"TIMEOUT", "10m")
		t.Setenv(EnvPrefix+"ALGO", "fast")
		t.Setenv(EnvPrefix+"VERBOSE", "yes")
		t.Setenv(EnvPrefix+"QUIET", "0")

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
		t.Setenv(EnvPrefix+"N", "notanumber")
		t.Setenv(EnvPrefix+"THRESHOLD", "invalid")
		t.Setenv(EnvPrefix+"TIMEOUT", "notaduration")

		var buf bytes.Buffer
		_, err := ParseConfig("test", []string{}, &buf, []string{"fast", "matrix", "fft"})
		if err == nil {
			t.Fatal("expected error for malformed env override, got nil")
		}
	})
}
