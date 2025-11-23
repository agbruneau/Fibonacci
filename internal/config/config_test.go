package config

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

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
