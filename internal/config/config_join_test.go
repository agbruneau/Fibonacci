package config

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/FibGo/internal/apperrors"
)

// Validate reports every problem, not just the first (audit API-07 / CFG-01).
// Returning on the first failing check would make a command line with two bad
// flags take two runs to fix.
func TestValidateReportsEveryProblem(t *testing.T) {
	t.Parallel()

	cfg := AppConfig{
		Timeout:           0,       // problem 1
		Threshold:         -5,      // problem 2
		FFTThreshold:      -5,      // problem 3
		StrassenThreshold: -1,      // problem 4
		Algo:              "nope",  // problem 5
		GCControl:         "loud",  // problem 6
		LogLevel:          "shout", // problem 7
	}

	err := cfg.Validate([]string{"fast", "matrix"})
	if err == nil {
		t.Fatal("Validate accepted a configuration with seven problems")
	}

	msg := err.Error()
	for _, want := range []string{
		"timeout value must be strictly positive",
		"parallelism threshold must be",
		"FFT threshold must be",
		"Strassen threshold cannot be negative",
		"unrecognized algorithm",
		"unrecognized gc-control mode",
		"unrecognized log-level",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("joined error is missing %q:\n%s", want, msg)
		}
	}

	// errors.Join separates elements with newlines; one line per problem.
	if lines := strings.Count(strings.TrimSpace(msg), "\n") + 1; lines != 7 {
		t.Errorf("joined error has %d lines, want 7:\n%s", lines, msg)
	}
}

// Joining must not break the typed-error contract: callers use errors.As to
// distinguish a configuration error from any other failure, and the exit-code
// mapping depends on it.
func TestValidateJoinedErrorStaysTyped(t *testing.T) {
	t.Parallel()

	cfg := AppConfig{Timeout: 0, Algo: "nope"}

	err := cfg.Validate([]string{"fast"})
	var cfgErr apperrors.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Errorf("errors.As found no ConfigError in the joined chain: %T %v", err, err)
	}
}

// A valid configuration must still produce a nil error: errors.Join returns nil
// for an all-nil list, so there is no "empty join" sentinel to leak.
func TestValidateReturnsNilWhenValid(t *testing.T) {
	t.Parallel()

	cfg := AppConfig{
		Timeout:      time.Second,
		Threshold:    10,
		FFTThreshold: 10,
		Algo:         "fast",
	}
	if err := cfg.Validate([]string{"fast", "matrix"}); err != nil {
		t.Errorf("Validate rejected a valid configuration: %v", err)
	}
}
