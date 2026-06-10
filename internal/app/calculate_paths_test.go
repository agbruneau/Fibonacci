package app

import (
	"bytes"
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/FibGo/internal/cli"
	"github.com/agbruneau/FibGo/internal/config"
	apperrors "github.com/agbruneau/FibGo/internal/errors"
	"github.com/agbruneau/FibGo/internal/orchestration"
	"github.com/agbruneau/FibGo/internal/testutil"
)

// TestRunLastDigitsCalculationError covers the error branch of runLastDigits:
// a pre-canceled context makes orchestration.ComputeLastDigits fail with
// context.Canceled, which must map to the canceled exit code and write a
// uniform "Status: Canceled" message to the error stream. Both color
// selections (CLI colors vs plain machine-output colors) are exercised.
func TestRunLastDigitsCalculationError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		machineOutput bool
	}{
		{"CLI color provider", false},
		{"Machine output uses plain color provider", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var outBuf, errBuf bytes.Buffer
			app := &Application{
				Config: config.AppConfig{
					N:             1_000_000,
					LastDigits:    5,
					Timeout:       1 * time.Minute,
					MachineOutput: tc.machineOutput,
				},
				Factory:   createMockFactory(big.NewInt(55), nil),
				ErrWriter: &errBuf,
			}

			ctx, cancel := context.WithCancel(context.Background())
			cancel() // ComputeLastDigits checks ctx.Err() upfront.

			exitCode := app.runLastDigits(ctx, &outBuf)

			if exitCode != apperrors.ExitErrorCanceled {
				t.Errorf("Expected exit code %d (canceled), got %d",
					apperrors.ExitErrorCanceled, exitCode)
			}
			errOut := testutil.StripAnsiCodes(errBuf.String())
			if !strings.Contains(errOut, "Status: Canceled") {
				t.Errorf("Error output should contain 'Status: Canceled'. Got:\n%s", errOut)
			}
		})
	}
}

// TestAnalyzeResultsSaveFailureReturnsGeneric verifies that a failing save
// (invalid output path) downgrades an otherwise successful run to the
// generic error exit code and suppresses the success notice.
func TestAnalyzeResultsSaveFailureReturnsGeneric(t *testing.T) {
	t.Parallel()
	// A null byte makes the path invalid on every platform.
	invalidPath := "invalid\x00path/result.txt"
	app := &Application{
		Config: config.AppConfig{
			N:          10,
			OutputFile: invalidPath,
		},
		ErrWriter: &bytes.Buffer{},
	}
	results := []orchestration.CalculationResult{
		{Name: "fast", Result: big.NewInt(55), Duration: time.Millisecond},
	}

	var outBuf bytes.Buffer
	outputCfg := cli.OutputConfig{OutputFile: invalidPath}

	exitCode := app.analyzeResultsWithOutput(results, outputCfg, &outBuf)

	if exitCode != apperrors.ExitErrorGeneric {
		t.Errorf("Expected exit code %d (generic) when save fails, got %d",
			apperrors.ExitErrorGeneric, exitCode)
	}
	output := testutil.StripAnsiCodes(outBuf.String())
	if strings.Contains(output, "Result saved to") {
		t.Errorf("Success notice must not be printed when save fails. Got:\n%s", output)
	}
}

// TestValidateMemoryBudgetSuggestion pins the presentation branch on budget
// overrun: the --last-digits hint must only appear when the user is not
// already in last-digits mode.
func TestValidateMemoryBudgetSuggestion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		lastDigits int
		wantHint   bool
	}{
		{"Full computation suggests last-digits", 0, true},
		{"Last-digits mode omits the hint", 7, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var outBuf bytes.Buffer
			app := &Application{
				Config: config.AppConfig{
					N:           1_000_000_000,
					MemoryLimit: "1K",
					LastDigits:  tc.lastDigits,
				},
				ErrWriter: &bytes.Buffer{},
			}

			code := app.validateMemoryBudget(&outBuf)

			if code != apperrors.ExitErrorConfig {
				t.Fatalf("Expected exit code %d (config), got %d", apperrors.ExitErrorConfig, code)
			}
			output := outBuf.String()
			if !strings.Contains(output, "exceeds limit") {
				t.Errorf("Output should mention exceeding the limit. Got:\n%s", output)
			}
			if gotHint := strings.Contains(output, "last-digits"); gotHint != tc.wantHint {
				t.Errorf("last-digits hint presence = %v, want %v. Output:\n%s",
					gotHint, tc.wantHint, output)
			}
		})
	}
}
