package config

import (
	"flag"
	"io"
	"strings"
	"testing"
)

// TestParseBoolEnv covers the tri-state parsing, including the fallback to
// the caller-supplied default for unrecognized values.
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
		{"garbage", true, true},
		{"garbage", false, false},
		{"", true, true},
	}
	for _, tc := range cases {
		if got := parseBoolEnv(tc.val, tc.defaultVal); got != tc.want {
			t.Errorf("parseBoolEnv(%q, %v) = %v, want %v", tc.val, tc.defaultVal, got, tc.want)
		}
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
