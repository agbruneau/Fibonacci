// This file covers env.go's private surface (isFlagSet, isFlagSetAny,
// validateEnvOverrides, parseBoolEnv, the envOverrides table itself). Tests
// that only exercise the public ParseConfig/AppConfig surface live in the
// black-box config_test.go instead.

package config

import (
	"flag"
	"io"
	"strings"
	"testing"
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

			if got := isFlagSetAny(fs, tt.check...); got != tt.want {
				t.Errorf("isFlagSetAny(%v) = %v, want %v", tt.check, got, tt.want)
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

// TestParseBoolEnv covers the tri-state parsing, including the loud error for
// unrecognized values (APP-09: malformed input must not be silently dropped
// back to the default).
func TestParseBoolEnv(t *testing.T) {
	t.Parallel()
	cases := []struct {
		val        string
		defaultVal bool
		want       bool
	}{
		{"true", false, true},
		{"1", false, true},
		{"YES", false, true},
		{"false", true, false},
		{"0", true, false},
		{"No", true, false},
	}
	for _, tc := range cases {
		got, err := parseBoolEnv(tc.val, tc.defaultVal)
		if err != nil {
			t.Errorf("parseBoolEnv(%q, %v) unexpected error: %v", tc.val, tc.defaultVal, err)
		}
		if got != tc.want {
			t.Errorf("parseBoolEnv(%q, %v) = %v, want %v", tc.val, tc.defaultVal, got, tc.want)
		}
	}
}

// TestParseBoolEnv_Malformed verifies that an unrecognized value surfaces an
// error instead of silently falling back to the default (APP-09).
func TestParseBoolEnv_Malformed(t *testing.T) {
	t.Parallel()
	if _, err := parseBoolEnv("garbage", true); err == nil {
		t.Fatal("expected error for malformed bool env value, got nil")
	}
}

// TestValidateEnvOverrides_Errors exercises every structural-inconsistency
// branch of validateEnvOverrides. It swaps the package-level envOverrides
// table, so it must not run in parallel with tests that read it (ParseConfig,
// the integrity test above); the original table is restored via t.Cleanup.
// Not t.Parallel(): mutates package-level state; Go's test runner completes
// all non-parallel tests (including this one's t.Cleanup restoration) before
// any t.Parallel() test body executes, so ParseConfig-based tests elsewhere
// in this package are never exposed to the swapped table.
func TestValidateEnvOverrides_Errors(t *testing.T) {
	orig := envOverrides
	t.Cleanup(func() { envOverrides = orig })

	fs := flag.NewFlagSet("verrs", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var dummy string
	fs.StringVar(&dummy, "x", "", "")
	noop := func(*AppConfig, string) error { return nil }

	cases := []struct {
		name    string
		table   []envOverride
		wantSub string
	}{
		{
			name:    "empty envKey",
			table:   []envOverride{{envKey: "", flags: []string{"x"}, apply: noop}},
			wantSub: "empty envKey",
		},
		{
			name:    "prefixed envKey",
			table:   []envOverride{{envKey: EnvPrefix + "N", flags: []string{"x"}, apply: noop}},
			wantSub: "must not include",
		},
		{
			name: "duplicate envKey",
			table: []envOverride{
				{envKey: "A", flags: []string{"x"}, apply: noop},
				{envKey: "A", flags: []string{"x"}, apply: noop},
			},
			wantSub: "duplicate envKey",
		},
		{
			name:    "empty flags slice",
			table:   []envOverride{{envKey: "A", flags: nil, apply: noop}},
			wantSub: "empty flags slice",
		},
		{
			name:    "unknown flag reference",
			table:   []envOverride{{envKey: "A", flags: []string{"missing"}, apply: noop}},
			wantSub: "unknown flag",
		},
		{
			name:    "nil apply",
			table:   []envOverride{{envKey: "A", flags: []string{"x"}, apply: nil}},
			wantSub: "nil apply",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			envOverrides = tc.table
			err := validateEnvOverrides(fs)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantSub)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantSub)
			}
		})
	}
}
