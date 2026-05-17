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

		if cfg.N != 100000000 {
			t.Errorf("Expected default N 100000000, got %d", cfg.N)
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
	})

	t.Run("MachineFlag", func(t *testing.T) {
		t.Parallel()
		cfg, err := ParseConfig("fibcalc", []string{"--machine"}, io.Discard, availableAlgos)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !cfg.MachineOutput {
			t.Error("Expected MachineOutput true")
		}
	})

	t.Run("EnvOverrides", func(t *testing.T) {
		// Set env vars
		env := map[string]string{
			"FIBCALC_N":                   "200",
			"FIBCALC_ALGO":                "matrix",
			"FIBCALC_TIMEOUT":             "2m",
			"FIBCALC_THRESHOLD":           "1024",
			"FIBCALC_FFT_THRESHOLD":       "5000",
			"FIBCALC_STRASSEN_THRESHOLD":  "128",
			"FIBCALC_VERBOSE":             "true",
			"FIBCALC_DETAILS":             "true",
			"FIBCALC_QUIET":               "true",
			"FIBCALC_MACHINE_OUTPUT":      "true",
			"FIBCALC_CALIBRATE":           "true",
			"FIBCALC_AUTO_CALIBRATE":      "true",
			"FIBCALC_OUTPUT":              "out.txt",
			"FIBCALC_CALIBRATION_PROFILE": "prof.json",
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
		if !cfg.MachineOutput {
			t.Error("Expected MachineOutput true")
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

// base returns a minimally-valid AppConfig for targeted validation tests.
func base() AppConfig {
	return AppConfig{Timeout: 1 * time.Second, Threshold: 10, FFTThreshold: 10, Algo: "fast"}
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

func TestVerboseFlagAlias(t *testing.T) {
	t.Parallel()
	availableAlgos := []string{"fast", "matrix", "fft"}

	tests := []struct {
		name string
		args []string
		want bool
	}{
		{
			name: "short form -v",
			args: []string{"-v"},
			want: true,
		},
		{
			name: "long form --verbose",
			args: []string{"-verbose"},
			want: true,
		},
		{
			name: "no verbose flag",
			args: []string{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := ParseConfig("test", tt.args, io.Discard, availableAlgos)
			if err != nil {
				t.Fatalf("ParseConfig failed: %v", err)
			}
			if cfg.Verbose != tt.want {
				t.Errorf("Verbose = %v, want %v", cfg.Verbose, tt.want)
			}
		})
	}
}

func TestTUIFlag(t *testing.T) {
	t.Parallel()
	availableAlgos := []string{"fast", "matrix", "fft"}

	t.Run("--tui flag", func(t *testing.T) {
		t.Parallel()
		cfg, err := ParseConfig("test", []string{"-tui"}, io.Discard, availableAlgos)
		if err != nil {
			t.Fatalf("ParseConfig failed: %v", err)
		}
		if !cfg.TUI {
			t.Error("TUI should be true when --tui flag is set")
		}
	})

	t.Run("default TUI is false", func(t *testing.T) {
		t.Parallel()
		cfg, err := ParseConfig("test", []string{}, io.Discard, availableAlgos)
		if err != nil {
			t.Fatalf("ParseConfig failed: %v", err)
		}
		if cfg.TUI {
			t.Error("TUI should be false by default")
		}
	})

	t.Run("FIBCALC_TUI env override", func(t *testing.T) {
		os.Setenv("FIBCALC_TUI", "true")
		defer os.Unsetenv("FIBCALC_TUI")

		cfg, err := ParseConfig("test", []string{}, io.Discard, availableAlgos)
		if err != nil {
			t.Fatalf("ParseConfig failed: %v", err)
		}
		if !cfg.TUI {
			t.Error("TUI should be true when FIBCALC_TUI=true")
		}
	})
}
