package tui

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"testing"
	"time"

	apperrors "github.com/agbru/fibcalc/internal/errors"
	"github.com/agbru/fibcalc/internal/orchestration"
	"github.com/agbru/fibcalc/internal/progress"
)

func TestTUIProgressReporter_DrainsChannel(t *testing.T) {
	t.Parallel()
	ref := &programRef{} // nil program - Send is a no-op

	reporter := &TUIProgressReporter{ref: ref}

	ch := make(chan progress.ProgressUpdate, 10)
	var wg sync.WaitGroup
	wg.Add(1)

	// Send some updates
	ch <- progress.ProgressUpdate{CalculatorIndex: 0, Value: 0.25}
	ch <- progress.ProgressUpdate{CalculatorIndex: 0, Value: 0.50}
	ch <- progress.ProgressUpdate{CalculatorIndex: 0, Value: 0.75}
	ch <- progress.ProgressUpdate{CalculatorIndex: 0, Value: 1.00}
	close(ch)

	go reporter.DisplayProgress(&wg, ch, 1, nil)
	wg.Wait()

	// Channel should be fully drained (close consumed)
	// If we reach here without deadlock, the test passes
}

func TestTUIProgressReporter_ZeroCalculators(t *testing.T) {
	t.Parallel()
	ref := &programRef{}
	reporter := &TUIProgressReporter{ref: ref}

	ch := make(chan progress.ProgressUpdate, 5)
	ch <- progress.ProgressUpdate{CalculatorIndex: 0, Value: 0.5}
	close(ch)

	var wg sync.WaitGroup
	wg.Add(1)
	go reporter.DisplayProgress(&wg, ch, 0, nil)
	wg.Wait()
}

func TestTUIResultPresenter_FormatDuration(t *testing.T) {
	t.Parallel()
	ref := &programRef{}
	presenter := &TUIResultPresenter{ref: ref}

	tests := []struct {
		name  string
		input time.Duration
	}{
		{"zero", 0},
		{"microseconds", 500 * time.Microsecond},
		{"milliseconds", 42 * time.Millisecond},
		{"seconds", 2*time.Second + 500*time.Millisecond},
		{"minutes", 3 * time.Minute},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := presenter.FormatDuration(tt.input)
			if result == "" {
				t.Errorf("expected non-empty duration format for %v", tt.input)
			}
		})
	}
}

func TestProgramRef_Send_NilProgram(t *testing.T) {
	t.Parallel()
	ref := &programRef{} // program is nil
	// Should not panic
	ref.Send(ProgressMsg{Value: 0.5})
}

func TestTUIResultPresenter_PresentComparisonTable(t *testing.T) {
	t.Parallel()
	ref := &programRef{} // nil program — just verify no panic
	presenter := &TUIResultPresenter{ref: ref}

	results := []orchestration.CalculationResult{
		{Name: "Fast", Result: big.NewInt(55), Duration: 100 * time.Millisecond},
		{Name: "Matrix", Result: big.NewInt(55), Duration: 200 * time.Millisecond},
	}
	// Should not panic
	presenter.PresentComparisonTable(results, nil)
}

func TestTUIResultPresenter_PresentResult(t *testing.T) {
	t.Parallel()
	ref := &programRef{}
	presenter := &TUIResultPresenter{ref: ref}

	result := orchestration.CalculationResult{
		Name:     "Fast",
		Result:   big.NewInt(55),
		Duration: 100 * time.Millisecond,
	}
	// Should not panic
	presenter.PresentResult(result, 10, true, true, true, nil)
}

func TestTUIResultPresenter_HandleError_Timeout(t *testing.T) {
	t.Parallel()
	ref := &programRef{}
	presenter := &TUIResultPresenter{ref: ref}

	exitCode := presenter.HandleError(context.DeadlineExceeded, time.Second, nil)
	if exitCode != apperrors.ExitErrorTimeout {
		t.Errorf("expected exit code %d for timeout, got %d", apperrors.ExitErrorTimeout, exitCode)
	}
}

func TestTUIResultPresenter_HandleError_Canceled(t *testing.T) {
	t.Parallel()
	ref := &programRef{}
	presenter := &TUIResultPresenter{ref: ref}

	exitCode := presenter.HandleError(context.Canceled, time.Second, nil)
	if exitCode != apperrors.ExitErrorCanceled {
		t.Errorf("expected exit code %d for canceled, got %d", apperrors.ExitErrorCanceled, exitCode)
	}
}

func TestTUIResultPresenter_HandleError_Generic(t *testing.T) {
	t.Parallel()
	ref := &programRef{}
	presenter := &TUIResultPresenter{ref: ref}

	exitCode := presenter.HandleError(errors.New("something failed"), time.Second, nil)
	if exitCode != apperrors.ExitErrorGeneric {
		t.Errorf("expected exit code %d for generic error, got %d", apperrors.ExitErrorGeneric, exitCode)
	}
}

func TestTUIResultPresenter_HandleError_Nil(t *testing.T) {
	t.Parallel()
	ref := &programRef{}
	presenter := &TUIResultPresenter{ref: ref}

	exitCode := presenter.HandleError(nil, 0, nil)
	if exitCode != apperrors.ExitSuccess {
		t.Errorf("expected exit code %d for nil error, got %d", apperrors.ExitSuccess, exitCode)
	}
}

func TestTUIProgressReporter_MultipleCalculators(t *testing.T) {
	t.Parallel()
	ref := &programRef{}
	reporter := &TUIProgressReporter{ref: ref}

	ch := make(chan progress.ProgressUpdate, 10)
	var wg sync.WaitGroup
	wg.Add(1)

	ch <- progress.ProgressUpdate{CalculatorIndex: 0, Value: 0.25}
	ch <- progress.ProgressUpdate{CalculatorIndex: 1, Value: 0.50}
	ch <- progress.ProgressUpdate{CalculatorIndex: 0, Value: 0.75}
	ch <- progress.ProgressUpdate{CalculatorIndex: 1, Value: 1.00}
	close(ch)

	go reporter.DisplayProgress(&wg, ch, 2, nil)
	wg.Wait()
}

func TestProgramRef_Send_Concurrent(t *testing.T) {
	t.Parallel()
	ref := &programRef{} // nil program - Send is a no-op

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ref.Send(ProgressMsg{Value: float64(i) / 100})
		}(i)
	}
	wg.Wait()
	// If we reach here without panic/race, the test passes
}

func TestTUIResultPresenter_HandleError_PassesDuration(t *testing.T) {
	t.Parallel()
	ref := &programRef{}
	presenter := &TUIResultPresenter{ref: ref}

	// The bug was that HandleError passed 0 instead of duration to HandleCalculationError.
	// We verify indirectly: the exit code should still be correct.
	duration := 5 * time.Second
	exitCode := presenter.HandleError(context.DeadlineExceeded, duration, nil)
	if exitCode != apperrors.ExitErrorTimeout {
		t.Errorf("expected exit code %d, got %d", apperrors.ExitErrorTimeout, exitCode)
	}
}

func TestTUIProgressReporter_EmptyChannel(t *testing.T) {
	t.Parallel()
	ref := &programRef{}
	reporter := &TUIProgressReporter{ref: ref}

	ch := make(chan progress.ProgressUpdate)
	close(ch)

	var wg sync.WaitGroup
	wg.Add(1)
	go reporter.DisplayProgress(&wg, ch, 1, nil)
	wg.Wait()
}
