package config

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestParseConfig(t *testing.T) {
	availableAlgos := []string{"fast", "matrix", "fft"}

	tests := []struct {
		name           string
		args           []string
		expectedN      uint64
		expectedAlgo   string
		expectError    bool
		expectedErrMsg string
	}{
		{
			name:         "Default values",
			args:         []string{},
			expectedN:    250000000, // Default value from code
			expectedAlgo: "all",
		},
		{
			name:         "Valid explicit values",
			args:         []string{"-n", "100", "-algo", "fast", "-timeout", "10s"},
			expectedN:    100,
			expectedAlgo: "fast",
		},
		{
			name:           "Invalid Algo",
			args:           []string{"-algo", "invalid"},
			expectError:    true,
			expectedErrMsg: "unrecognized algorithm",
		},
		{
			name:           "Negative Threshold",
			args:           []string{"-threshold", "-1"},
			expectError:    true,
			expectedErrMsg: "parallelism threshold cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var errBuf bytes.Buffer
			cfg, err := ParseConfig("test", tt.args, &errBuf, availableAlgos)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				if tt.expectedErrMsg != "" && !strings.Contains(err.Error(), tt.expectedErrMsg) && !strings.Contains(errBuf.String(), tt.expectedErrMsg) {
					t.Errorf("Expected error message containing '%s', got error '%v' and output '%s'", tt.expectedErrMsg, err, errBuf.String())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if cfg.N != tt.expectedN {
					t.Errorf("Expected N=%d, got %d", tt.expectedN, cfg.N)
				}
				if cfg.Algo != tt.expectedAlgo {
					t.Errorf("Expected Algo=%s, got %s", tt.expectedAlgo, cfg.Algo)
				}
			}
		})
	}
}

func TestConfigValidation(t *testing.T) {
	availableAlgos := []string{"fast"}

	t.Run("Timeout Validation", func(t *testing.T) {
		cfg := AppConfig{Timeout: 0, Algo: "fast"}
		err := cfg.Validate(availableAlgos)
		if err == nil {
			t.Error("Expected error for zero timeout")
		}
	})

	t.Run("FFT Threshold Validation", func(t *testing.T) {
		cfg := AppConfig{Timeout: time.Second, Algo: "fast", FFTThreshold: -1}
		err := cfg.Validate(availableAlgos)
		if err == nil {
			t.Error("Expected error for negative FFT threshold")
		}
	})
}
