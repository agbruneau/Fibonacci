package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"example.com/fibcalc/internal/fibonacci"
)

// TestParseConfig validates the configuration parsing function.
func TestParseConfig(t *testing.T) {
	var errorSink bytes.Buffer

	testCases := []struct {
		name         string
		args         []string
		expectErr    bool
		expectedN    uint64
		expectedAlgo string
	}{
		{"Nominal case (defaults)", []string{}, false, 250000000, "all"},
		{"Specifying N", []string{"-n", "50"}, false, 50, "all"},
		{"Specifying the algorithm", []string{"-algo", "fast"}, false, 250000000, "fast"},
		{"Specifying the algorithm (case-insensitive)", []string{"-algo", "MATRIX"}, false, 250000000, "matrix"},
		{"Error case: negative threshold", []string{"-threshold", "-100"}, true, 0, ""},
		{"Error case: unknown argument", []string{"-invalid-flag"}, true, 0, ""},
		{"Error case: unknown algorithm", []string{"-algo", "nonexistent"}, true, 0, ""},
		{"Error case: invalid timeout", []string{"-timeout", "-5s"}, true, 0, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config, err := parseConfig("test", tc.args, &errorSink)

			if tc.expectErr {
				if err == nil {
					t.Error("An error was expected, but none was returned.")
				}
			} else {
				if err != nil {
					t.Errorf("An unexpected error was returned: %v", err)
				}
				if config.N != tc.expectedN {
					t.Errorf("Incorrect N field in config. Expected: %d, Got: %d", tc.expectedN, config.N)
				}
				if config.Algo != tc.expectedAlgo {
					t.Errorf("Incorrect Algo field in config. Expected: %q, Got: %q", tc.expectedAlgo, config.Algo)
				}
			}
		})
	}
}

// TestRunFunction validates the behavior of the main orchestration function `run`.
func TestRunFunction(t *testing.T) {

	t.Run("Simple execution with success", func(t *testing.T) {
		var buf bytes.Buffer
		config := AppConfig{N: 10, Algo: "fast", Timeout: 1 * time.Minute, Threshold: fibonacci.DefaultParallelThreshold, FFTThreshold: 20000, Details: true}
		exitCode := run(context.Background(), config, &buf)

		if exitCode != ExitSuccess {
			t.Errorf("Incorrect exit code. Expected: %d, Got: %d", ExitSuccess, exitCode)
		}
		output := buf.String()
		if !strings.Contains(output, "F(10) = 55") {
			t.Errorf("The detailed output does not contain the expected result 'F(10) = 55'. Output:\n%s", output)
		}
	})

	t.Run("Parallel comparison with success", func(t *testing.T) {
		var buf bytes.Buffer
		config := AppConfig{N: 20, Algo: "all", Timeout: 1 * time.Minute, Threshold: fibonacci.DefaultParallelThreshold, FFTThreshold: 20000, Details: true}
		exitCode := run(context.Background(), config, &buf)

		if exitCode != ExitSuccess {
			t.Errorf("Incorrect exit code. Expected: %d, Got: %d", ExitSuccess, exitCode)
		}
		output := buf.String()
		if !strings.Contains(output, "Comparison Summary") || !strings.Contains(output, "Global Status: Success") {
			t.Errorf("The comparison mode output is incorrect. Output:\n%s", output)
		}
		if !strings.Contains(output, "Calculation time") {
			t.Errorf("The detailed output should contain the calculation time. Output:\n%s", output)
		}
	})

	t.Run("Failure due to timeout", func(t *testing.T) {
		var buf bytes.Buffer
		config := AppConfig{N: 100_000_000, Algo: "fast", Timeout: 1 * time.Millisecond}
		exitCode := run(context.Background(), config, &buf)

		if exitCode != ExitErrorTimeout {
			t.Errorf("Incorrect exit code for a timeout. Expected: %d, Got: %d", ExitErrorTimeout, exitCode)
		}
		output := buf.String()
		if !strings.Contains(output, "Failure (Timeout)") {
			t.Errorf("The output should explicitly mention the timeout failure. Output:\n%s", output)
		}
	})

	t.Run("Failure due to context cancellation", func(t *testing.T) {
		var buf bytes.Buffer
		config := AppConfig{N: 100_000_000, Algo: "fast", Timeout: 1 * time.Minute}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		exitCode := run(ctx, config, &buf)

		if exitCode != ExitErrorCanceled {
			t.Errorf("Incorrect exit code for a cancellation. Expected: %d, Got: %d", ExitErrorCanceled, exitCode)
		}
		output := buf.String()
		if !strings.Contains(output, "Status: Canceled") {
			t.Errorf("The output should explicitly mention the cancellation. Output:\n%s", output)
		}
	})
}