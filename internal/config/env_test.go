package config

import (
	"flag"
	"io"
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
