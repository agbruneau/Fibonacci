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
			var outBuf, errBuf bytes.Buffer
			app := &Application{
				Config: config.AppConfig{
					N:           1_000_000_000,
					MemoryLimit: "1K",
					LastDigits:  tc.lastDigits,
				},
				ErrWriter: &errBuf,
			}

			code := app.validateMemoryBudget(&outBuf)

			if code != apperrors.ExitErrorConfig {
				t.Fatalf("Expected exit code %d (config), got %d", apperrors.ExitErrorConfig, code)
			}
			// ERR-02: error text goes to ErrWriter, stdout stays clean.
			output := errBuf.String()
			if !strings.Contains(output, "exceeds limit") {
				t.Errorf("ErrWriter should mention exceeding the limit. Got:\n%s", output)
			}
			if gotHint := strings.Contains(output, "last-digits"); gotHint != tc.wantHint {
				t.Errorf("last-digits hint presence = %v, want %v. ErrWriter:\n%s",
					gotHint, tc.wantHint, output)
			}
			if outBuf.Len() != 0 {
				t.Errorf("stdout must stay clean on budget failure, got:\n%s", outBuf.String())
			}
		})
	}
}

// TestRunTUI_RejectsOversizedMemoryBudget reproduces audit.md APP-07: runTUI
// launched the dashboard unconditionally, so --memory-limit was silently
// ignored in TUI mode. An over-budget limit must short-circuit before
// tui.Run is ever invoked (which would otherwise hang the test, per the note
// in app_dispatch_test.go) and return the same config exit code the CLI path
// uses.
func TestRunTUI_RejectsOversizedMemoryBudget(t *testing.T) {
	t.Parallel()
	var errBuf bytes.Buffer
	app := &Application{
		Config: config.AppConfig{
			N:           1_000_000_000,
			MemoryLimit: "1K",
			TUI:         true,
		},
		ErrWriter: &errBuf,
	}

	var outBuf bytes.Buffer
	exitCode := app.runTUI(context.Background(), &outBuf)

	if exitCode != apperrors.ExitErrorConfig {
		t.Fatalf("expected exit code %d (config), got %d", apperrors.ExitErrorConfig, exitCode)
	}
	// ERR-02: the diagnostic goes to ErrWriter, stdout stays clean.
	if !strings.Contains(errBuf.String(), "exceeds limit") {
		t.Errorf("expected the memory budget diagnostic on stderr, got:\n%s", errBuf.String())
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

// TestSaveResultIfNeeded_WritesToErrWriter reproduces audit.md APP-16:
// saveResultIfNeeded wrote its failure diagnostic straight to os.Stderr
// instead of the injected a.ErrWriter, so tests/consumers redirecting
// ErrWriter never saw the message. A save failure must be visible on
// a.ErrWriter and must NOT leak onto the real os.Stderr.
func TestSaveResultIfNeeded_WritesToErrWriter(t *testing.T) {
	t.Parallel()
	var errBuf bytes.Buffer
	app := &Application{
		Config:    config.AppConfig{N: 10},
		ErrWriter: &errBuf,
	}
	res := &orchestration.CalculationResult{
		Name:     "fast",
		Result:   big.NewInt(55),
		Duration: 1 * time.Millisecond,
	}
	cfg := cli.OutputConfig{OutputFile: "invalid\x00path/file.txt"}

	if err := app.saveResultIfNeeded(res, cfg); err == nil {
		t.Fatal("expected error for invalid output path")
	}

	if !strings.Contains(errBuf.String(), "Error saving result") {
		t.Errorf("expected diagnostic on a.ErrWriter, got: %q", errBuf.String())
	}
}

// TestRunLastDigits_RejectsOversizedK reproduces audit.md SEC-03: the
// last-digits path dispatches straight to ComputeLastDigits with no bound on
// K, so 10^K allocates ~K*3.32 bits of memory regardless of --memory-limit.
// An oversized K must be rejected (config error, no allocation attempted)
// with the diagnostic written to a.ErrWriter.
func TestRunLastDigits_RejectsOversizedK(t *testing.T) {
	t.Parallel()
	var outBuf, errBuf bytes.Buffer
	app := &Application{
		Config: config.AppConfig{
			N:          1_000_000,
			LastDigits: maxLastDigits + 1,
			Timeout:    1 * time.Minute,
		},
		ErrWriter: &errBuf,
	}

	exitCode := app.runLastDigits(context.Background(), &outBuf)

	if exitCode != apperrors.ExitErrorConfig {
		t.Errorf("Expected exit code %d (config error), got %d", apperrors.ExitErrorConfig, exitCode)
	}
	if !strings.Contains(errBuf.String(), "last-digits") {
		t.Errorf("Expected rejection diagnostic on a.ErrWriter, got: %q", errBuf.String())
	}
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

// TestPresentQuietMismatchWritesStderr pins audit M-06: quiet mode silences
// stdout, not diagnostics. The mismatch guard above used to return exit 3 while
// writing nothing anywhere, leaving a calling script with an empty stdout and a
// bare exit code to interpret. The explanation belongs on stderr, where it
// cannot contaminate a captured stdout.
func TestPresentQuietMismatchWritesStderr(t *testing.T) {
	t.Parallel()
	var errBuf bytes.Buffer
	app := &Application{
		Config:    config.AppConfig{N: 10, Quiet: true},
		ErrWriter: &errBuf,
	}
	results := []orchestration.CalculationResult{
		{Name: "fast", Result: big.NewInt(55), Duration: time.Millisecond},
		{Name: "matrix", Result: big.NewInt(54), Duration: 2 * time.Millisecond}, // divergent
	}

	var outBuf bytes.Buffer
	exitCode := app.analyzeResultsWithOutput(results, cli.OutputConfig{Quiet: true}, &outBuf)

	if exitCode != apperrors.ExitErrorMismatch {
		t.Fatalf("got exit %d, want %d (mismatch)", exitCode, apperrors.ExitErrorMismatch)
	}
	if got := strings.TrimSpace(outBuf.String()); got != "" {
		t.Errorf("stdout must stay empty in quiet mode, got %q", got)
	}
	stderr := testutil.StripAnsiCodes(errBuf.String())
	if !strings.Contains(stderr, orchestration.MismatchMessage) {
		t.Errorf("stderr must explain the mismatch, got %q", stderr)
	}
	if !strings.HasSuffix(errBuf.String(), "\n") {
		t.Error("the diagnostic must be newline-terminated")
	}
}
