package config

import (
	"flag"
	"io"
	"strings"
	"testing"
)

// TestParseBoolEnv covers the tri-state parsing, including the loud error for
// unrecognized values (APP-09: malformed input must not be silently dropped
// back to the default).
func TestParseBoolEnv(t *testing.T) {
	t.Parallel()
	cases := []struct {
		val  string
		want bool
	}{
		{"true", true},
		{"1", true},
		{"YES", true},
		{"false", false},
		{"0", false},
		{"No", false},
	}
	for _, tc := range cases {
		got, err := parseBoolEnv(tc.val)
		if err != nil {
			t.Errorf("parseBoolEnv(%q) unexpected error: %v", tc.val, err)
		}
		if got != tc.want {
			t.Errorf("parseBoolEnv(%q) = %v, want %v", tc.val, got, tc.want)
		}
	}
}

// TestParseBoolEnv_Malformed verifies that an unrecognized value surfaces an
// error instead of silently falling back to the default (APP-09).
func TestParseBoolEnv_Malformed(t *testing.T) {
	t.Parallel()
	if _, err := parseBoolEnv("garbage"); err == nil {
		t.Fatal("expected error for malformed bool env value, got nil")
	}
}

// TestEnvOverride_MalformedBool verifies that a malformed boolean env
// override surfaces a structured config error through ParseConfig (APP-09).
func TestEnvOverride_MalformedBool(t *testing.T) {
	algos := []string{"fast", "matrix"}
	t.Setenv(EnvPrefix+"VERBOSE", "notabool")

	_, err := ParseConfig("test", []string{}, io.Discard, algos)
	if err == nil {
		t.Fatal("expected error for malformed FIBCALC_VERBOSE, got nil")
	}
}

// TestEnvOverride_LastDigitsAndGCControl verifies that FIBCALC_LAST_DIGITS
// and FIBCALC_GC_CONTROL are honored (APP-08).
func TestEnvOverride_LastDigitsAndGCControl(t *testing.T) {
	algos := []string{"fast", "matrix"}
	t.Setenv(EnvPrefix+"LAST_DIGITS", "42")
	t.Setenv(EnvPrefix+"GC_CONTROL", "aggressive")

	cfg, err := ParseConfig("test", []string{}, io.Discard, algos)
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if cfg.LastDigits != 42 {
		t.Errorf("LastDigits = %d, want 42 (env override)", cfg.LastDigits)
	}
	if cfg.GCControl != "aggressive" {
		t.Errorf("GCControl = %q, want %q (env override)", cfg.GCControl, "aggressive")
	}
}

// TestEnvOverride_MemoryLimitAndCalculate covers the MEMORY_LIMIT (string) and
// CALCULATE (bool) overrides plus the CLI > env priority. Uses t.Setenv
// (process-global env), so no t.Parallel.
func TestEnvOverride_MemoryLimitAndCalculate(t *testing.T) {
	algos := []string{"fast", "matrix"}
	t.Setenv(EnvPrefix+"MEMORY_LIMIT", "2G")
	t.Setenv(EnvPrefix+"CALCULATE", "yes")

	cfg, err := ParseConfig("test", []string{}, io.Discard, algos)
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if cfg.MemoryLimit != "2G" {
		t.Errorf("MemoryLimit = %q, want %q (env override)", cfg.MemoryLimit, "2G")
	}
	if !cfg.ShowValue {
		t.Error("ShowValue = false, want true (FIBCALC_CALCULATE=yes)")
	}

	// An explicit CLI flag must win over the environment value.
	cfg, err = ParseConfig("test", []string{"-memory-limit", "512M"}, io.Discard, algos)
	if err != nil {
		t.Fatalf("ParseConfig with flags: %v", err)
	}
	if cfg.MemoryLimit != "512M" {
		t.Errorf("MemoryLimit = %q, want %q (CLI wins over env)", cfg.MemoryLimit, "512M")
	}
}

// TestValidateEnvOverrides_Errors exercises every structural-inconsistency
// branch of validateEnvOverrides. It swaps the package-level envOverrides
// table, so it must not run in parallel with tests that read it (ParseConfig,
// the integrity test); the original table is restored via t.Cleanup.
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
