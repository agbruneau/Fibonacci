package app

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/FibGo/internal/cli"
	"github.com/agbruneau/FibGo/internal/config"
	apperrors "github.com/agbruneau/FibGo/internal/errors"
	"github.com/agbruneau/FibGo/internal/fibonacci"
	"github.com/agbruneau/FibGo/internal/orchestration"
	"github.com/agbruneau/FibGo/internal/progress"
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

// TestPresentQuietAllMismatchReturnsMismatch guards the quiet + comparison-mode
// fast path: when several algorithms succeed but disagree, quiet mode must NOT
// emit a (wrong) value with a success code — it must surface ExitErrorMismatch
// and print nothing on stdout. Regression guard for the silent-wrong-answer gap.
func TestPresentQuietAllMismatchReturnsMismatch(t *testing.T) {
	t.Parallel()
	app := &Application{
		Config:    config.AppConfig{N: 10, Quiet: true},
		ErrWriter: &bytes.Buffer{},
	}
	results := []orchestration.CalculationResult{
		{Name: "fast", Result: big.NewInt(55), Duration: time.Millisecond},
		{Name: "matrix", Result: big.NewInt(54), Duration: 2 * time.Millisecond}, // divergent
	}
	outputCfg := cli.OutputConfig{Quiet: true}

	var outBuf bytes.Buffer
	exitCode := app.analyzeResultsWithOutput(results, outputCfg, &outBuf)

	if exitCode != apperrors.ExitErrorMismatch {
		t.Errorf("quiet+all with divergent results: got exit %d, want %d (mismatch)",
			exitCode, apperrors.ExitErrorMismatch)
	}
	if got := strings.TrimSpace(outBuf.String()); got != "" {
		t.Errorf("quiet mode must not print a value on mismatch, got %q", got)
	}
}

// TestPresentQuietAllConsistentPrintsValue is the companion: when the successful
// algorithms agree, quiet mode prints the value and exits 0 (the mismatch guard
// must not regress the normal consistent path).
func TestPresentQuietAllConsistentPrintsValue(t *testing.T) {
	t.Parallel()
	app := &Application{
		Config:    config.AppConfig{N: 10, Quiet: true},
		ErrWriter: &bytes.Buffer{},
	}
	results := []orchestration.CalculationResult{
		{Name: "fast", Result: big.NewInt(55), Duration: time.Millisecond},
		{Name: "matrix", Result: big.NewInt(55), Duration: 2 * time.Millisecond},
	}
	outputCfg := cli.OutputConfig{Quiet: true}

	var outBuf bytes.Buffer
	exitCode := app.analyzeResultsWithOutput(results, outputCfg, &outBuf)

	if exitCode != apperrors.ExitSuccess {
		t.Errorf("quiet+all with consistent results: got exit %d, want %d (success)",
			exitCode, apperrors.ExitSuccess)
	}
	if !strings.Contains(outBuf.String(), "55") {
		t.Errorf("quiet mode should print the value on success, got %q", outBuf.String())
	}
}

// TestAnalyzeResultsWithOutput_ComparisonModePartialFailure reproduces APP-01:
// in comparison mode with an output file, selectBest captures a pointer into
// the results slice; AnalyzeComparisonResults (invoked from present) then
// sorts that same slice in place, which can leave the pointer aimed at a
// failed slot (Result == nil) by the time save() dereferences it. Mirrors the
// audit.md §3/M1 probe: {failure, slow success, fast success}.
func TestAnalyzeResultsWithOutput_ComparisonModePartialFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	outputFile := dir + "/result.txt"
	app := &Application{
		Config: config.AppConfig{
			N:          10,
			OutputFile: outputFile,
		},
		ErrWriter: &bytes.Buffer{},
	}
	results := []orchestration.CalculationResult{
		{Name: "failure", Result: nil, Duration: time.Millisecond, Err: fmt.Errorf("boom")},
		{Name: "slow", Result: big.NewInt(55), Duration: 20 * time.Millisecond},
		{Name: "fast", Result: big.NewInt(55), Duration: 5 * time.Millisecond},
	}
	outputCfg := cli.OutputConfig{OutputFile: outputFile}

	var outBuf bytes.Buffer
	exitCode := app.analyzeResultsWithOutput(results, outputCfg, &outBuf)

	if exitCode != apperrors.ExitSuccess {
		t.Fatalf("expected exit code %d (success), got %d", apperrors.ExitSuccess, exitCode)
	}
	saved, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if !strings.Contains(string(saved), "# Algorithm: fast") {
		t.Errorf("output file must record the fastest successful algorithm (fast), got:\n%s", saved)
	}
}

// TestAnalyzeResultsWithOutput_AllSucceedSavesFastest is the complementary
// case: when every algorithm succeeds, the output file must still carry the
// Name/Duration of the correct (fastest) algorithm after the in-place sort.
func TestAnalyzeResultsWithOutput_AllSucceedSavesFastest(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	outputFile := dir + "/result.txt"
	app := &Application{
		Config: config.AppConfig{
			N:          10,
			OutputFile: outputFile,
		},
		ErrWriter: &bytes.Buffer{},
	}
	results := []orchestration.CalculationResult{
		{Name: "slow", Result: big.NewInt(55), Duration: 20 * time.Millisecond},
		{Name: "fast", Result: big.NewInt(55), Duration: 5 * time.Millisecond},
	}
	outputCfg := cli.OutputConfig{OutputFile: outputFile}

	var outBuf bytes.Buffer
	exitCode := app.analyzeResultsWithOutput(results, outputCfg, &outBuf)

	if exitCode != apperrors.ExitSuccess {
		t.Fatalf("expected exit code %d (success), got %d", apperrors.ExitSuccess, exitCode)
	}
	saved, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if !strings.Contains(string(saved), "# Algorithm: fast") {
		t.Errorf("output file must record the fastest algorithm (fast), got:\n%s", saved)
	}
	if !strings.Contains(string(saved), "# Duration: 5ms") {
		t.Errorf("output file must record the fastest algorithm's duration (5ms), got:\n%s", saved)
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

// gcSpyCalculator captures the Options it receives so tests can assert on
// GCMode wiring without depending on the shared MockCalculator (which
// discards opts).
type gcSpyCalculator struct {
	capturedOpts fibonacci.Options
}

func (s *gcSpyCalculator) Name() string { return "spy" }

func (s *gcSpyCalculator) Calculate(_ context.Context, progressChan chan<- progress.ProgressUpdate, calcIndex int, _ uint64, opts fibonacci.Options) (*big.Int, error) {
	s.capturedOpts = opts
	if progressChan != nil {
		progressChan <- progress.ProgressUpdate{CalculatorIndex: calcIndex, Value: 1.0}
	}
	return big.NewInt(55), nil
}

// TestExecuteCalculations_WiresGCControl reproduces audit.md M2/FIB-01=APP-02:
// --gc-control is parsed and validated but never copied into the
// fibonacci.Options the calculator receives, so "disabled"/"aggressive" are
// silent no-ops. executeCalculations must forward Config.GCControl as
// Options.GCMode.
func TestExecuteCalculations_WiresGCControl(t *testing.T) {
	t.Parallel()
	spy := &gcSpyCalculator{}
	factory := fibonacci.NewTestFactory(map[string]fibonacci.Calculator{"fast": spy})
	app := &Application{
		Config: config.AppConfig{
			N:         10,
			Algo:      "fast",
			GCControl: "disabled",
			Quiet:     true,
		},
		Factory: factory,
	}

	var outBuf bytes.Buffer
	app.executeCalculations(context.Background(), &outBuf)

	if spy.capturedOpts.GCMode != "disabled" {
		t.Errorf("Options.GCMode: want %q, got %q", "disabled", spy.capturedOpts.GCMode)
	}
}
