package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	apperrors "github.com/agbruneau/FibGo/internal/errors"
)

// TestEnvOverridesIntegrity guards against silent drift between the
// envOverrides table and the canonical CLI FlagSet built by registerFlags.
// Any new flag that gains an env override (or any rename) must keep both in
// sync; this test fails loudly if they don't.
func TestEnvOverridesIntegrity(t *testing.T) {
	t.Parallel()
	fs := flag.NewFlagSet("integrity", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var cfg AppConfig
	registerFlags(fs, &cfg, []string{"fast", "matrix"})

	if err := validateEnvOverrides(fs); err != nil {
		t.Fatalf("envOverrides table is inconsistent with the CLI FlagSet: %v", err)
	}
}

// TestEnvOverride_MalformedReturnsError verifies that an explicitly-set but
// unparsable environment override surfaces a structured config error instead
// of being silently swallowed (which previously left a default value that
// could trigger an O(memory) calculation / OOM).
func TestEnvOverride_MalformedReturnsError(t *testing.T) {
	algos := []string{"fast", "matrix", "fft"}

	tests := []struct {
		name   string
		envKey string
		envVal string
	}{
		{"malformed N", EnvPrefix + "N", "abc"},
		{"malformed TIMEOUT", EnvPrefix + "TIMEOUT", "xyz"},
		{"malformed THRESHOLD", EnvPrefix + "THRESHOLD", "notanint"},
		{"malformed FFT_THRESHOLD", EnvPrefix + "FFT_THRESHOLD", "1.5"},
		{"malformed STRASSEN_THRESHOLD", EnvPrefix + "STRASSEN_THRESHOLD", "??"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old, had := os.LookupEnv(tt.envKey)
			os.Setenv(tt.envKey, tt.envVal)
			defer func() {
				if had {
					os.Setenv(tt.envKey, old)
				} else {
					os.Unsetenv(tt.envKey)
				}
			}()

			_, err := ParseConfig("test", []string{}, io.Discard, algos)
			if err == nil {
				t.Fatalf("expected error for %s=%q, got nil", tt.envKey, tt.envVal)
			}
			var cfgErr apperrors.ConfigError
			if !errors.As(err, &cfgErr) {
				t.Errorf("expected error chain to contain apperrors.ConfigError, got %T: %v", err, err)
			}
		})
	}
}

func TestIsFlagSetAny(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setFlags []string
		check    []string
		want     bool
	}{
		{
			name:     "no flags set",
			setFlags: []string{},
			check:    []string{"a", "b"},
			want:     false,
		},
		{
			name:     "first alias set",
			setFlags: []string{"a"},
			check:    []string{"a", "b"},
			want:     true,
		},
		{
			name:     "second alias set",
			setFlags: []string{"b"},
			check:    []string{"a", "b"},
			want:     true,
		},
		{
			name:     "both set",
			setFlags: []string{"a", "b"},
			check:    []string{"a", "b"},
			want:     true,
		},
		{
			name:     "different flag set",
			setFlags: []string{"c"},
			check:    []string{"a", "b"},
			want:     false,
		},
		{
			name:     "single flag check - set",
			setFlags: []string{"verbose"},
			check:    []string{"verbose"},
			want:     true,
		},
		{
			name:     "single flag check - not set",
			setFlags: []string{},
			check:    []string{"verbose"},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			var val bool
			fs.BoolVar(&val, "a", false, "")
			fs.BoolVar(&val, "b", false, "")
			fs.BoolVar(&val, "c", false, "")
			fs.BoolVar(&val, "verbose", false, "")

			args := make([]string, 0, len(tt.setFlags))
			for _, f := range tt.setFlags {
				args = append(args, "-"+f)
			}
			if err := fs.Parse(args); err != nil {
				t.Fatalf("Failed to parse flags: %v", err)
			}

			if got := isFlagSet(fs, tt.check...); got != tt.want {
				t.Errorf("isFlagSet(%v) = %v, want %v", tt.check, got, tt.want)
			}
		})
	}
}

func TestIsFlagSet(t *testing.T) {
	t.Parallel()

	t.Run("flag is set", func(t *testing.T) {
		t.Parallel()
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		var val bool
		fs.BoolVar(&val, "test", false, "")
		if err := fs.Parse([]string{"-test"}); err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}
		if !isFlagSet(fs, "test") {
			t.Error("Expected flag to be set")
		}
	})

	t.Run("flag is not set", func(t *testing.T) {
		t.Parallel()
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		var val bool
		fs.BoolVar(&val, "test", false, "")
		if err := fs.Parse([]string{}); err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}
		if isFlagSet(fs, "test") {
			t.Error("Expected flag to not be set")
		}
	})
}

// validateEnvOverrides verifies the structural consistency of the envOverrides
// table against the canonical CLI FlagSet. It catches the silent inconsistency
// risk inherent to the dual declaration (FlagSet in registerFlags + table in
// env.go):
//
//  1. Every envKey is non-empty, free of the FIBCALC_ prefix, and unique.
//  2. Every flag name referenced in an entry's flags slice exists on fs.
//  3. No entry has an empty flags slice (would short-circuit isFlagSet and
//     always apply the env override, defeating CLI > env priority).
//  4. apply is non-nil.
//
// It lives in the test files because nothing in production calls it.
func validateEnvOverrides(fs *flag.FlagSet) error {
	known := make(map[string]struct{})
	fs.VisitAll(func(f *flag.Flag) {
		known[f.Name] = struct{}{}
	})

	seen := make(map[string]struct{}, len(envOverrides))
	for _, o := range envOverrides {
		if o.envKey == "" {
			return fmt.Errorf("envOverrides: entry has empty envKey")
		}
		if strings.HasPrefix(o.envKey, EnvPrefix) {
			return fmt.Errorf("envOverrides: envKey %q must not include the %s prefix (added automatically)", o.envKey, EnvPrefix)
		}
		if _, dup := seen[o.envKey]; dup {
			return fmt.Errorf("envOverrides: duplicate envKey %q", o.envKey)
		}
		seen[o.envKey] = struct{}{}

		if len(o.flags) == 0 {
			return fmt.Errorf("envOverrides: entry %q has empty flags slice", o.envKey)
		}
		for _, name := range o.flags {
			if _, ok := known[name]; !ok {
				return fmt.Errorf("envOverrides: entry %q references unknown flag %q (not registered on FlagSet)", o.envKey, name)
			}
		}
		if o.apply == nil {
			return fmt.Errorf("envOverrides: entry %q has nil apply function", o.envKey)
		}
	}
	return nil
}
