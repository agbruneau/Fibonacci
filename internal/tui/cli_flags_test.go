package tui

import (
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/agbru/fibcalc/internal/config"
	apperrors "github.com/agbru/fibcalc/internal/errors"
	"github.com/agbru/fibcalc/internal/fibonacci"
	"github.com/agbru/fibcalc/internal/orchestration"
	"github.com/agbru/fibcalc/internal/progress"
)

// capturingCalculator captures the parameters passed to Calculate
// so tests can verify config propagation.
type capturingCalculator struct {
	capturedN    uint64
	capturedOpts fibonacci.Options
	result       *big.Int
	name         string
}

func (c *capturingCalculator) Calculate(_ context.Context, progressChan chan<- progress.ProgressUpdate, calcIndex int, n uint64, opts fibonacci.Options) (*big.Int, error) {
	c.capturedN = n
	c.capturedOpts = opts
	if progressChan != nil {
		progressChan <- progress.ProgressUpdate{CalculatorIndex: calcIndex, Value: 1.0}
	}
	return c.result, nil
}

func (c *capturingCalculator) Name() string {
	if c.name != "" {
		return c.name
	}
	return "capturing"
}

// Verify interface compliance.
var _ fibonacci.Calculator = (*capturingCalculator)(nil)

// blockingCalculator blocks until context cancellation.
type blockingCalculator struct{}

func (b blockingCalculator) Calculate(ctx context.Context, _ chan<- progress.ProgressUpdate, _ int, _ uint64, _ fibonacci.Options) (*big.Int, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func (b blockingCalculator) Name() string { return "blocking" }

// Verify interface compliance.
var _ fibonacci.Calculator = blockingCalculator{}

// ---------------------------------------------------------------------------
// Group 1: Config propagation to model
// ---------------------------------------------------------------------------

func TestNewModel_ConfigPropagation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  config.AppConfig
	}{
		{
			name: "Default values",
			cfg: config.AppConfig{
				N:                 config.DefaultN,
				Timeout:           config.DefaultTimeout,
				Algo:              config.DefaultAlgo,
				Threshold:         fibonacci.DefaultParallelThreshold,
				FFTThreshold:      fibonacci.DefaultFFTThreshold,
				StrassenThreshold: fibonacci.DefaultStrassenThreshold,
				TUI:               true,
			},
		},
		{
			name: "Custom N and timeout",
			cfg: config.AppConfig{
				N:       42,
				Timeout: 30 * time.Second,
				Algo:    "fast",
				TUI:     true,
			},
		},
		{
			name: "Custom thresholds",
			cfg: config.AppConfig{
				N:                 100,
				Timeout:           time.Minute,
				Threshold:         8192,
				FFTThreshold:      100_000,
				StrassenThreshold: 2048,
				TUI:               true,
			},
		},
		{
			name: "All display flags true",
			cfg: config.AppConfig{
				N:         10,
				Timeout:   time.Minute,
				Verbose:   true,
				Details:   true,
				ShowValue: true,
				TUI:       true,
			},
		},
		{
			name: "All display flags false",
			cfg: config.AppConfig{
				N:         10,
				Timeout:   time.Minute,
				Verbose:   false,
				Details:   false,
				ShowValue: false,
				TUI:       true,
			},
		},
		{
			name: "N zero edge case",
			cfg: config.AppConfig{
				N:       0,
				Timeout: time.Minute,
				TUI:     true,
			},
		},
		{
			name: "N one",
			cfg: config.AppConfig{
				N:       1,
				Timeout: time.Minute,
				TUI:     true,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := NewModel(context.Background(), nil, tt.cfg, "v1.0.0")
			t.Cleanup(m.cancel)

			if m.config.N != tt.cfg.N {
				t.Errorf("N: want %d, got %d", tt.cfg.N, m.config.N)
			}
			if m.config.Timeout != tt.cfg.Timeout {
				t.Errorf("Timeout: want %v, got %v", tt.cfg.Timeout, m.config.Timeout)
			}
			if m.config.Algo != tt.cfg.Algo {
				t.Errorf("Algo: want %q, got %q", tt.cfg.Algo, m.config.Algo)
			}
			if m.config.Threshold != tt.cfg.Threshold {
				t.Errorf("Threshold: want %d, got %d", tt.cfg.Threshold, m.config.Threshold)
			}
			if m.config.FFTThreshold != tt.cfg.FFTThreshold {
				t.Errorf("FFTThreshold: want %d, got %d", tt.cfg.FFTThreshold, m.config.FFTThreshold)
			}
			if m.config.StrassenThreshold != tt.cfg.StrassenThreshold {
				t.Errorf("StrassenThreshold: want %d, got %d", tt.cfg.StrassenThreshold, m.config.StrassenThreshold)
			}
			if m.config.Verbose != tt.cfg.Verbose {
				t.Errorf("Verbose: want %v, got %v", tt.cfg.Verbose, m.config.Verbose)
			}
			if m.config.Details != tt.cfg.Details {
				t.Errorf("Details: want %v, got %v", tt.cfg.Details, m.config.Details)
			}
			if m.config.ShowValue != tt.cfg.ShowValue {
				t.Errorf("ShowValue: want %v, got %v", tt.cfg.ShowValue, m.config.ShowValue)
			}
			if m.config.TUI != tt.cfg.TUI {
				t.Errorf("TUI: want %v, got %v", tt.cfg.TUI, m.config.TUI)
			}
		})
	}
}

func TestNewModel_ConfigPreservedAfterWindowResize(t *testing.T) {
	t.Parallel()
	cfg := config.AppConfig{
		N:                 42,
		Timeout:           30 * time.Second,
		Algo:              "fast",
		Threshold:         8192,
		FFTThreshold:      200_000,
		StrassenThreshold: 4096,
		Verbose:           true,
		Details:           true,
		ShowValue:         true,
		TUI:               true,
	}
	m := NewModel(context.Background(), nil, cfg, "v1.0.0")
	t.Cleanup(m.cancel)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	result := updated.(Model)

	if result.config.N != 42 {
		t.Errorf("N changed after resize: want 42, got %d", result.config.N)
	}
	if result.config.Threshold != 8192 {
		t.Errorf("Threshold changed after resize: want 8192, got %d", result.config.Threshold)
	}
	if !result.config.Verbose {
		t.Error("Verbose changed to false after resize")
	}
	if result.config.Algo != "fast" {
		t.Errorf("Algo changed after resize: want 'fast', got %q", result.config.Algo)
	}
}

func TestNewModel_ConfigPreservedAfterRestart(t *testing.T) {
	t.Parallel()
	cfg := config.AppConfig{
		N:                 42,
		Timeout:           30 * time.Second,
		Algo:              "fast",
		Threshold:         8192,
		FFTThreshold:      200_000,
		StrassenThreshold: 4096,
		Verbose:           true,
		Details:           true,
		ShowValue:         true,
		TUI:               true,
	}
	calcs := []fibonacci.Calculator{mockCalculator{name: "Fast"}}
	m := NewModel(context.Background(), calcs, cfg, "v1.0.0")
	t.Cleanup(m.cancel)

	// Set size so restart works properly
	sized, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = sized.(Model)

	// Press 'r' to restart
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	result := updated.(Model)

	if result.config.N != 42 {
		t.Errorf("N changed after restart: want 42, got %d", result.config.N)
	}
	if result.config.Threshold != 8192 {
		t.Errorf("Threshold changed after restart: want 8192, got %d", result.config.Threshold)
	}
	if result.config.FFTThreshold != 200_000 {
		t.Errorf("FFTThreshold changed after restart: want 200000, got %d", result.config.FFTThreshold)
	}
	if result.config.StrassenThreshold != 4096 {
		t.Errorf("StrassenThreshold changed after restart: want 4096, got %d", result.config.StrassenThreshold)
	}
	if !result.config.Verbose {
		t.Error("Verbose changed to false after restart")
	}
	if !result.config.Details {
		t.Error("Details changed to false after restart")
	}
	if !result.config.ShowValue {
		t.Error("ShowValue changed to false after restart")
	}
}

// ---------------------------------------------------------------------------
// Group 2: Algorithm selection
// ---------------------------------------------------------------------------

func TestModel_CalculatorSelection(t *testing.T) {
	t.Parallel()
	cfg := config.AppConfig{N: 1000, Timeout: time.Minute}

	tests := []struct {
		name      string
		calcs     []fibonacci.Calculator
		wantCount int
		wantNames []string
	}{
		{
			name:      "Single calculator",
			calcs:     []fibonacci.Calculator{mockCalculator{name: "Fast Doubling"}},
			wantCount: 1,
			wantNames: []string{"Fast Doubling"},
		},
		{
			name: "Two calculators",
			calcs: []fibonacci.Calculator{
				mockCalculator{name: "Fast Doubling"},
				mockCalculator{name: "Matrix"},
			},
			wantCount: 2,
			wantNames: []string{"Fast Doubling", "Matrix"},
		},
		{
			name: "Three calculators",
			calcs: []fibonacci.Calculator{
				mockCalculator{name: "Fast Doubling"},
				mockCalculator{name: "Matrix"},
				mockCalculator{name: "FFT"},
			},
			wantCount: 3,
			wantNames: []string{"Fast Doubling", "Matrix", "FFT"},
		},
		{
			name:      "Nil calculators",
			calcs:     nil,
			wantCount: 0,
			wantNames: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := NewModel(context.Background(), tt.calcs, cfg, "v1.0.0")
			t.Cleanup(m.cancel)

			if len(m.calculators) != tt.wantCount {
				t.Errorf("calculator count: want %d, got %d", tt.wantCount, len(m.calculators))
			}
			for i, wantName := range tt.wantNames {
				if m.calculators[i].Name() != wantName {
					t.Errorf("calculator[%d] name: want %q, got %q", i, wantName, m.calculators[i].Name())
				}
			}
		})
	}
}

func TestModel_CalculatorsPreservedAfterRestart(t *testing.T) {
	t.Parallel()
	calcs := []fibonacci.Calculator{
		mockCalculator{name: "Fast Doubling"},
		mockCalculator{name: "Matrix"},
	}
	cfg := config.AppConfig{N: 1000, Timeout: time.Minute}
	m := NewModel(context.Background(), calcs, cfg, "v1.0.0")
	t.Cleanup(m.cancel)

	// Set size and mark done
	sized, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = sized.(Model)
	m.done = true

	// Press 'r' to restart
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	result := updated.(Model)

	if len(result.calculators) != 2 {
		t.Fatalf("calculator count after restart: want 2, got %d", len(result.calculators))
	}
	if result.calculators[0].Name() != "Fast Doubling" {
		t.Errorf("calculator[0] name after restart: want 'Fast Doubling', got %q", result.calculators[0].Name())
	}
	if result.calculators[1].Name() != "Matrix" {
		t.Errorf("calculator[1] name after restart: want 'Matrix', got %q", result.calculators[1].Name())
	}
}

// ---------------------------------------------------------------------------
// Group 3: startCalculationCmd with different configs
// ---------------------------------------------------------------------------

func TestStartCalculationCmd_ConfigPassthrough(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		n                 uint64
		threshold         int
		fftThreshold      int
		strassenThreshold int
	}{
		{
			name:              "Default thresholds",
			n:                 10,
			threshold:         fibonacci.DefaultParallelThreshold,
			fftThreshold:      fibonacci.DefaultFFTThreshold,
			strassenThreshold: fibonacci.DefaultStrassenThreshold,
		},
		{
			name:              "Custom thresholds",
			n:                 50,
			threshold:         8192,
			fftThreshold:      200_000,
			strassenThreshold: 4096,
		},
		{
			name:              "N zero",
			n:                 0,
			threshold:         fibonacci.DefaultParallelThreshold,
			fftThreshold:      fibonacci.DefaultFFTThreshold,
			strassenThreshold: fibonacci.DefaultStrassenThreshold,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			capture := &capturingCalculator{result: big.NewInt(55)}
			ref := &programRef{}
			ctx := context.Background()
			cfg := config.AppConfig{
				N:                 tt.n,
				Timeout:           time.Minute,
				Threshold:         tt.threshold,
				FFTThreshold:      tt.fftThreshold,
				StrassenThreshold: tt.strassenThreshold,
			}

			cmd := startCalculationCmd(ref, ctx, []fibonacci.Calculator{capture}, cfg, 0)
			msg := cmd()

			complete, ok := msg.(CalculationCompleteMsg)
			if !ok {
				t.Fatalf("expected CalculationCompleteMsg, got %T", msg)
			}
			if complete.ExitCode != apperrors.ExitSuccess {
				t.Errorf("exit code: want %d, got %d", apperrors.ExitSuccess, complete.ExitCode)
			}
			if capture.capturedN != tt.n {
				t.Errorf("captured N: want %d, got %d", tt.n, capture.capturedN)
			}
			if capture.capturedOpts.ParallelThreshold != tt.threshold {
				t.Errorf("ParallelThreshold: want %d, got %d", tt.threshold, capture.capturedOpts.ParallelThreshold)
			}
			if capture.capturedOpts.FFTThreshold != tt.fftThreshold {
				t.Errorf("FFTThreshold: want %d, got %d", tt.fftThreshold, capture.capturedOpts.FFTThreshold)
			}
			if capture.capturedOpts.StrassenThreshold != tt.strassenThreshold {
				t.Errorf("StrassenThreshold: want %d, got %d", tt.strassenThreshold, capture.capturedOpts.StrassenThreshold)
			}
		})
	}
}

func TestStartCalculationCmd_ExitCodes(t *testing.T) {
	t.Parallel()

	t.Run("Successful calculation", func(t *testing.T) {
		t.Parallel()
		ref := &programRef{}
		calc := mockCalculator{name: "Fast"}
		cfg := config.AppConfig{N: 10, Timeout: time.Minute}

		cmd := startCalculationCmd(ref, context.Background(), []fibonacci.Calculator{calc}, cfg, 0)
		msg := cmd()

		complete, ok := msg.(CalculationCompleteMsg)
		if !ok {
			t.Fatalf("expected CalculationCompleteMsg, got %T", msg)
		}
		if complete.ExitCode != apperrors.ExitSuccess {
			t.Errorf("exit code: want %d, got %d", apperrors.ExitSuccess, complete.ExitCode)
		}
	})

	t.Run("Timeout", func(t *testing.T) {
		t.Parallel()
		ref := &programRef{}
		calc := blockingCalculator{}
		cfg := config.AppConfig{N: 100_000_000, Timeout: time.Minute}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		cmd := startCalculationCmd(ref, ctx, []fibonacci.Calculator{calc}, cfg, 0)
		msg := cmd()

		complete, ok := msg.(CalculationCompleteMsg)
		if !ok {
			t.Fatalf("expected CalculationCompleteMsg, got %T", msg)
		}
		if complete.ExitCode != apperrors.ExitErrorTimeout {
			t.Errorf("exit code: want %d (timeout), got %d", apperrors.ExitErrorTimeout, complete.ExitCode)
		}
	})
}

func TestStartCalculationCmd_Generation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		generation uint64
	}{
		{"Generation 0", 0},
		{"Generation 1", 1},
		{"Generation 42", 42},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ref := &programRef{}
			calc := mockCalculator{name: "Fast"}
			cfg := config.AppConfig{N: 10, Timeout: time.Minute}

			cmd := startCalculationCmd(ref, context.Background(), []fibonacci.Calculator{calc}, cfg, tt.generation)
			msg := cmd()

			complete, ok := msg.(CalculationCompleteMsg)
			if !ok {
				t.Fatalf("expected CalculationCompleteMsg, got %T", msg)
			}
			if complete.Generation != tt.generation {
				t.Errorf("generation: want %d, got %d", tt.generation, complete.Generation)
			}
		})
	}
}

func TestStartCalculationCmd_NoCalculators(t *testing.T) {
	t.Parallel()
	ref := &programRef{}
	cfg := config.AppConfig{N: 10, Timeout: time.Minute}

	cmd := startCalculationCmd(ref, context.Background(), []fibonacci.Calculator{}, cfg, 0)
	msg := cmd()

	_, ok := msg.(CalculationCompleteMsg)
	if !ok {
		t.Fatalf("expected CalculationCompleteMsg, got %T", msg)
	}
}

// ---------------------------------------------------------------------------
// Group 4: Display flags (Verbose, Details, ShowValue)
// ---------------------------------------------------------------------------

func TestStartCalculationCmd_DisplayFlagsInConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		verbose   bool
		details   bool
		showValue bool
	}{
		{"Verbose only", true, false, false},
		{"Details only", false, true, false},
		{"ShowValue only", false, false, true},
		{"All display flags true", true, true, true},
		{"All display flags false", false, false, false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ref := &programRef{}
			calc := mockCalculator{name: "Fast"}
			cfg := config.AppConfig{
				N:         10,
				Timeout:   time.Minute,
				Verbose:   tt.verbose,
				Details:   tt.details,
				ShowValue: tt.showValue,
			}

			cmd := startCalculationCmd(ref, context.Background(), []fibonacci.Calculator{calc}, cfg, 0)
			msg := cmd()

			complete, ok := msg.(CalculationCompleteMsg)
			if !ok {
				t.Fatalf("expected CalculationCompleteMsg, got %T", msg)
			}
			if complete.ExitCode != apperrors.ExitSuccess {
				t.Errorf("exit code: want %d, got %d", apperrors.ExitSuccess, complete.ExitCode)
			}
		})
	}
}

func TestFinalResultMsg_DisplayFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		verbose   bool
		details   bool
		showValue bool
	}{
		{"All true", true, true, true},
		{"All false", false, false, false},
		{"Verbose only", true, false, false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := config.AppConfig{N: 10, Timeout: time.Minute}
			m := NewModel(context.Background(), nil, cfg, "v1.0.0")
			t.Cleanup(m.cancel)

			// Set size first
			sized, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
			m = sized.(Model)

			msg := FinalResultMsg{
				Result: orchestration.CalculationResult{
					Name:     "Fast",
					Result:   big.NewInt(55),
					Duration: 50 * time.Millisecond,
				},
				N:         10,
				Verbose:   tt.verbose,
				Details:   tt.details,
				ShowValue: tt.showValue,
			}

			updated, _ := m.Update(msg)
			result := updated.(Model)

			if len(result.logs.entries) == 0 {
				t.Error("expected log entries after FinalResultMsg")
			}

			// Verify that "Final Result" appears in log entries
			found := false
			for _, entry := range result.logs.entries {
				if strings.Contains(entry, "Final Result") {
					found = true
					break
				}
			}
			if !found {
				t.Error("expected log entries to contain 'Final Result'")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Group 5: Timeout
// ---------------------------------------------------------------------------

func TestStartCalculationCmd_Timeout(t *testing.T) {
	t.Parallel()
	ref := &programRef{}
	calc := blockingCalculator{}
	cfg := config.AppConfig{N: 100_000_000, Timeout: time.Minute}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	cmd := startCalculationCmd(ref, ctx, []fibonacci.Calculator{calc}, cfg, 0)
	msg := cmd()

	complete, ok := msg.(CalculationCompleteMsg)
	if !ok {
		t.Fatalf("expected CalculationCompleteMsg, got %T", msg)
	}
	if complete.ExitCode != apperrors.ExitErrorTimeout {
		t.Errorf("exit code: want %d (timeout), got %d", apperrors.ExitErrorTimeout, complete.ExitCode)
	}
}

func TestModel_TimeoutContextPropagation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	cmd := watchContextCmd(ctx, 5)
	if cmd == nil {
		t.Fatal("expected non-nil command from watchContextCmd")
	}

	done := make(chan tea.Msg, 1)
	go func() {
		done <- cmd()
	}()

	select {
	case msg := <-done:
		ccMsg, ok := msg.(ContextCancelledMsg)
		if !ok {
			t.Fatalf("expected ContextCancelledMsg, got %T", msg)
		}
		if ccMsg.Generation != 5 {
			t.Errorf("generation: want 5, got %d", ccMsg.Generation)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for ContextCancelledMsg")
	}
}

// ---------------------------------------------------------------------------
// Group 6: Edge cases
// ---------------------------------------------------------------------------

func TestStartCalculationCmd_SmallN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		n      uint64
		result *big.Int
	}{
		{"N=0", 0, big.NewInt(0)},
		{"N=1", 1, big.NewInt(1)},
		{"N=2", 2, big.NewInt(1)},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			capture := &capturingCalculator{result: tt.result}
			ref := &programRef{}
			cfg := config.AppConfig{N: tt.n, Timeout: time.Minute}

			cmd := startCalculationCmd(ref, context.Background(), []fibonacci.Calculator{capture}, cfg, 0)
			msg := cmd()

			complete, ok := msg.(CalculationCompleteMsg)
			if !ok {
				t.Fatalf("expected CalculationCompleteMsg, got %T", msg)
			}
			if complete.ExitCode != apperrors.ExitSuccess {
				t.Errorf("exit code: want %d, got %d", apperrors.ExitSuccess, complete.ExitCode)
			}
			if capture.capturedN != tt.n {
				t.Errorf("captured N: want %d, got %d", tt.n, capture.capturedN)
			}
		})
	}
}

func TestStartCalculationCmd_ZeroThresholds(t *testing.T) {
	t.Parallel()
	capture := &capturingCalculator{result: big.NewInt(55)}
	ref := &programRef{}
	cfg := config.AppConfig{
		N:                 10,
		Timeout:           time.Minute,
		Threshold:         0,
		FFTThreshold:      0,
		StrassenThreshold: 0,
	}

	cmd := startCalculationCmd(ref, context.Background(), []fibonacci.Calculator{capture}, cfg, 0)
	msg := cmd()

	complete, ok := msg.(CalculationCompleteMsg)
	if !ok {
		t.Fatalf("expected CalculationCompleteMsg, got %T", msg)
	}
	if complete.ExitCode != apperrors.ExitSuccess {
		t.Errorf("exit code: want %d, got %d", apperrors.ExitSuccess, complete.ExitCode)
	}
	if capture.capturedOpts.ParallelThreshold != 0 {
		t.Errorf("ParallelThreshold: want 0, got %d", capture.capturedOpts.ParallelThreshold)
	}
	if capture.capturedOpts.FFTThreshold != 0 {
		t.Errorf("FFTThreshold: want 0, got %d", capture.capturedOpts.FFTThreshold)
	}
	if capture.capturedOpts.StrassenThreshold != 0 {
		t.Errorf("StrassenThreshold: want 0, got %d", capture.capturedOpts.StrassenThreshold)
	}
}

func TestNewModel_AlgoConfigStored(t *testing.T) {
	t.Parallel()

	tests := []struct {
		algo string
	}{
		{"fast"},
		{"all"},
		{"matrix"},
		{"fft"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.algo, func(t *testing.T) {
			t.Parallel()
			cfg := config.AppConfig{
				N:       10,
				Timeout: time.Minute,
				Algo:    tt.algo,
				TUI:     true,
			}
			m := NewModel(context.Background(), nil, cfg, "v1.0.0")
			t.Cleanup(m.cancel)

			if m.config.Algo != tt.algo {
				t.Errorf("Algo: want %q, got %q", tt.algo, m.config.Algo)
			}
		})
	}
}
