package main

import (
	"bytes"
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/agbru/fibcalc/internal/app"
	"github.com/agbru/fibcalc/internal/config"
	apperrors "github.com/agbru/fibcalc/internal/errors"
	"github.com/agbru/fibcalc/internal/fibonacci"
	"github.com/agbru/fibcalc/internal/testutil"
)

// TestRun validates the run function with various argument combinations.
// Note: run() delegates to app.New and application.Run.
// Since run() constructs the real application with the global factory,
// these tests are effectively integration tests.
func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		args           []string
		wantExitCode   int
		wantOutContain string
	}{
		{
			name:           "version flag short",
			args:           []string{"fibcalc", "-V"},
			wantExitCode:   apperrors.ExitSuccess,
			wantOutContain: "fibcalc",
		},
		{
			name:           "version flag long",
			args:           []string{"fibcalc", "--version"},
			wantExitCode:   apperrors.ExitSuccess,
			wantOutContain: "fibcalc",
		},
		{
			name:         "help flag",
			args:         []string{"fibcalc", "-h"},
			wantExitCode: apperrors.ExitSuccess,
		},
		{
			name:         "invalid flag",
			args:         []string{"fibcalc", "--invalid-flag"},
			wantExitCode: apperrors.ExitErrorConfig,
		},
		{
			name:           "simple calculation",
			args:           []string{"fibcalc", "-n", "10", "-algo", "fast", "-timeout", "30s", "-q", "-c"},
			wantExitCode:   apperrors.ExitSuccess,
			wantOutContain: "55",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var stdout, stderr bytes.Buffer
			exitCode := run(tc.args, &stdout, &stderr)

			if exitCode != tc.wantExitCode {
				t.Errorf("run() exit code = %d, want %d\nstderr: %s", exitCode, tc.wantExitCode, stderr.String())
			}
			if tc.wantOutContain != "" && !strings.Contains(stdout.String(), tc.wantOutContain) {
				t.Errorf("stdout should contain %q, got: %s", tc.wantOutContain, stdout.String())
			}
		})
	}
}

// TestParseConfig validates the configuration parsing function.
func TestParseConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      []string
		wantErr   bool
		wantN     uint64
		wantAlgo  string
	}{
		{"defaults", []string{}, false, 250_000_000, "all"},
		{"custom N", []string{"-n", "50"}, false, 50, "all"},
		{"algorithm fast", []string{"-algo", "fast"}, false, 250_000_000, "fast"},
		{"algorithm case insensitive", []string{"-algo", "MATRIX"}, false, 250_000_000, "matrix"},
		{"error: negative threshold", []string{"-threshold", "-100"}, true, 0, ""},
		{"error: unknown flag", []string{"-invalid-flag"}, true, 0, ""},
		{"error: unknown algorithm", []string{"-algo", "nonexistent"}, true, 0, ""},
		{"error: invalid timeout", []string{"-timeout", "-5s"}, true, 0, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var errBuf bytes.Buffer
			availableAlgos := fibonacci.GlobalFactory().List()
			cfg, err := config.ParseConfig("test", tc.args, &errBuf, availableAlgos)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.N != tc.wantN {
				t.Errorf("N = %d, want %d", cfg.N, tc.wantN)
			}
			if cfg.Algo != tc.wantAlgo {
				t.Errorf("Algo = %q, want %q", cfg.Algo, tc.wantAlgo)
			}
		})
	}
}

// TestApplicationRun validates the Application.Run method behavior.
// Optimized to use MockCalculator via TestFactory, avoiding expensive calculations.
func TestApplicationRun(t *testing.T) {
	t.Parallel()

	// Mock factory helper
	createMockFactory := func(result *big.Int, err error) *fibonacci.TestFactory {
		mockCalc := &fibonacci.MockCalculator{
			Result: result,
			Err:    err,
		}
		// Create a map with mocked calculators matching typical names
		// or at least "fast", "matrix", "fft" as needed by tests.
		// For "all", it iterates all calculators in the factory.
		calcs := map[string]fibonacci.Calculator{
			"fast":   mockCalc,
			"matrix": mockCalc,
		}
		return fibonacci.NewTestFactory(calcs)
	}

	t.Run("single algorithm success", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		factory := createMockFactory(big.NewInt(55), nil)

		application := &app.Application{
			Config: config.AppConfig{
				N:            10,
				Algo:         "fast",
				Timeout:      time.Minute,
				Threshold:    fibonacci.DefaultParallelThreshold,
				FFTThreshold: 20000,
				Details:      true,
				Concise:      true,
			},
			Factory:   factory,
			ErrWriter: &bytes.Buffer{},
		}

		exitCode := application.Run(context.Background(), &buf)

		if exitCode != apperrors.ExitSuccess {
			t.Errorf("exit code = %d, want %d", exitCode, apperrors.ExitSuccess)
		}
		output := testutil.StripAnsiCodes(buf.String())
		if !strings.Contains(output, "F(10) = 55") {
			t.Errorf("output should contain 'F(10) = 55', got:\n%s", output)
		}
	})

	t.Run("comparison mode success", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		factory := createMockFactory(big.NewInt(55), nil)

		application := &app.Application{
			Config: config.AppConfig{
				N:            20,
				Algo:         "all",
				Timeout:      time.Minute,
				Threshold:    fibonacci.DefaultParallelThreshold,
				FFTThreshold: 20000,
				Details:      true,
			},
			Factory:   factory,
			ErrWriter: &bytes.Buffer{},
		}

		exitCode := application.Run(context.Background(), &buf)

		if exitCode != apperrors.ExitSuccess {
			t.Errorf("exit code = %d, want %d", exitCode, apperrors.ExitSuccess)
		}
		output := testutil.StripAnsiCodes(buf.String())
		if !strings.Contains(output, "Comparison Summary") || !strings.Contains(output, "Global Status: Success") {
			t.Errorf("comparison output incorrect:\n%s", output)
		}
		// Since we use a mock, calculation time might be 0 or very small, but it should be printed.
		if !strings.Contains(output, "Calculation time") {
			t.Errorf("output should contain calculation time:\n%s", output)
		}
	})

	t.Run("timeout failure", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		// Mock a calculator that hangs or simulates timeout error?
		// Application.Run handles context timeout, but if we pass a very short timeout
		// in config, Application.Run creates a context with that timeout.
		// The MockCalculator checks context? No, the MockCalculator implementation above
		// just returns.
		// So we need the MockCalculator to respect context or return the expected error.
		// However, the test sets Timeout: time.Millisecond.
		// Application.Run creates context.
		// Then it calls calculator.Calculate(ctx, ...).
		// If MockCalculator returns immediately, we might not trigger timeout *inside* the calc.
		// But Application.Run wraps the call.

		// Let's make the mock actually block or return context error.
		mockCalc := &fibonacci.MockCalculator{
			Fn: func(ctx context.Context, n uint64) (*big.Int, error) {
				// Simulate work longer than timeout
				select {
				case <-time.After(50 * time.Millisecond):
					return big.NewInt(0), nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			},
		}
		calcs := map[string]fibonacci.Calculator{"fast": mockCalc}
		factory := fibonacci.NewTestFactory(calcs)

		application := &app.Application{
			Config: config.AppConfig{
				N:       100_000_000,
				Algo:    "fast",
				Timeout: time.Millisecond, // Very short timeout
			},
			Factory:   factory,
			ErrWriter: &bytes.Buffer{},
		}

		exitCode := application.Run(context.Background(), &buf)

		if exitCode != apperrors.ExitErrorTimeout {
			t.Errorf("exit code = %d, want %d", exitCode, apperrors.ExitErrorTimeout)
		}
		output := testutil.StripAnsiCodes(buf.String())
		if !strings.Contains(output, "Status: Failure (Timeout)") {
			t.Errorf("output should mention timeout failure:\n%s", output)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		// Mock that blocks until cancel
		mockCalc := &fibonacci.MockCalculator{
			Fn: func(ctx context.Context, n uint64) (*big.Int, error) {
				<-ctx.Done()
				return nil, ctx.Err()
			},
		}
		factory := fibonacci.NewTestFactory(map[string]fibonacci.Calculator{"fast": mockCalc})

		application := &app.Application{
			Config: config.AppConfig{
				N:       100_000_000,
				Algo:    "fast",
				Timeout: time.Minute,
			},
			Factory:   factory,
			ErrWriter: &bytes.Buffer{},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		exitCode := application.Run(ctx, &buf)

		if exitCode != apperrors.ExitErrorCanceled {
			t.Errorf("exit code = %d, want %d", exitCode, apperrors.ExitErrorCanceled)
		}
		output := testutil.StripAnsiCodes(buf.String())
		if !strings.Contains(output, "Status: Canceled") {
			t.Errorf("output should mention cancellation:\n%s", output)
		}
	})
}

// TestVersionFlag tests version flag detection.
func TestVersionFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"no flag", []string{"-n", "100"}, false},
		{"long flag", []string{"--version"}, true},
		{"short flag", []string{"-V"}, true},
		{"middle position", []string{"-n", "100", "--version"}, true},
		{"empty args", []string{}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := app.HasVersionFlag(tc.args)
			if got != tc.want {
				t.Errorf("HasVersionFlag(%v) = %v, want %v", tc.args, got, tc.want)
			}
		})
	}
}
