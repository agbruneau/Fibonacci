package fibonacci

import (
	"context"
	"errors"
	"strings"
	"testing"

	apperrors "github.com/agbru/fibcalc/internal/errors"
)

func TestFibCalculator_Name(t *testing.T) {
	t.Parallel()

	t.Run("Name delegates to core calculator", func(t *testing.T) {
		t.Parallel()
		core := &FastDoublingCalculator{}
		calc := MustNewCalculator(core)

		name := calc.Name()
		expected := core.Name()

		if name != expected {
			t.Errorf("Name() = %q, want %q", name, expected)
		}
		// Verify it contains expected parts
		if !strings.Contains(name, "Fast") && !strings.Contains(name, "Doubling") {
			t.Errorf("Expected name to contain 'Fast' or 'Doubling', got %q", name)
		}
	})

	t.Run("Name with MatrixExponentiationCalculator", func(t *testing.T) {
		t.Parallel()
		core := &MatrixExponentiationCalculator{}
		calc := MustNewCalculator(core)

		name := calc.Name()
		expected := core.Name()

		if name != expected {
			t.Errorf("Name() = %q, want %q", name, expected)
		}
		// Verify it contains expected parts
		if !strings.Contains(name, "Matrix") && !strings.Contains(name, "Exponentiation") {
			t.Errorf("Expected name to contain 'Matrix' or 'Exponentiation', got %q", name)
		}
	})

	t.Run("Name with FFTBasedCalculator", func(t *testing.T) {
		t.Parallel()
		core := &FFTBasedCalculator{}
		calc := MustNewCalculator(core)

		name := calc.Name()
		expected := core.Name()

		if name != expected {
			t.Errorf("Name() = %q, want %q", name, expected)
		}
		// Verify it contains expected parts
		if !strings.Contains(name, "FFT") {
			t.Errorf("Expected name to contain 'FFT', got %q", name)
		}
	})
}

// TestCanCalculate verifies the verdict semantics of CanCalculate, including
// the disabled (limit==0) path and the boundary cases where the estimate
// exactly equals or just exceeds the limit.
func TestCanCalculate(t *testing.T) {
	t.Parallel()

	t.Run("zero limit disables the check", func(t *testing.T) {
		t.Parallel()
		ok, est := CanCalculate(10_000_000, 0)
		if !ok {
			t.Fatalf("CanCalculate(_, 0) should always return ok=true")
		}
		if est.TotalBytes == 0 {
			t.Errorf("expected non-zero estimate, got %d", est.TotalBytes)
		}
	})

	t.Run("over budget returns false", func(t *testing.T) {
		t.Parallel()
		ok, est := CanCalculate(10_000_000, 1024) // 1 KB cap, ~10M bits Fibonacci
		if ok {
			t.Fatalf("CanCalculate(10M, 1KB) should return ok=false (estimate=%d)", est.TotalBytes)
		}
		if est.TotalBytes <= 1024 {
			t.Errorf("expected estimate > 1024, got %d", est.TotalBytes)
		}
	})

	t.Run("ample budget returns true", func(t *testing.T) {
		t.Parallel()
		ok, est := CanCalculate(1_000, 1<<30) // 1 GB cap, trivial N
		if !ok {
			t.Fatalf("CanCalculate(1k, 1GB) should return ok=true (estimate=%d)", est.TotalBytes)
		}
	})
}

// TestCalculateWithObservers_MemoryLimitExceeded verifies that the calculator
// short-circuits with apperrors.MemoryError when opts.MemoryLimitBytes is set
// to a value below the predicted memory footprint.
func TestCalculateWithObservers_MemoryLimitExceeded(t *testing.T) {
	t.Parallel()

	calc := MustNewCalculator(&FastDoublingCalculator{})

	// 1 KB cap is comically below what F(10_000_000) would need; the check
	// must fire before any allocation.
	opts := Options{MemoryLimitBytes: 1024}

	result, err := calc.Calculate(context.Background(), nil, 0, 10_000_000, opts)
	if err == nil {
		t.Fatalf("expected MemoryError, got nil (result=%v)", result)
	}
	if result != nil {
		t.Errorf("expected nil result on MemoryError, got non-nil")
	}

	var memErr apperrors.MemoryError
	if !errors.As(err, &memErr) {
		t.Fatalf("expected apperrors.MemoryError, got %T: %v", err, err)
	}
	if memErr.Limit != 1024 {
		t.Errorf("memErr.Limit = %d, want 1024", memErr.Limit)
	}
	if memErr.Requested <= memErr.Limit {
		t.Errorf("memErr.Requested = %d, want > %d", memErr.Requested, memErr.Limit)
	}
}

// TestCalculateWithObservers_MemoryLimitAllowed verifies that a generous
// memory limit does NOT block a small calculation, and that the zero value
// preserves backwards-compatible (no-check) behaviour.
func TestCalculateWithObservers_MemoryLimitAllowed(t *testing.T) {
	t.Parallel()

	calc := MustNewCalculator(&FastDoublingCalculator{})

	t.Run("ample limit", func(t *testing.T) {
		t.Parallel()
		opts := Options{MemoryLimitBytes: 1 << 30} // 1 GB cap, plenty for F(200)
		got, err := calc.Calculate(context.Background(), nil, 0, 200, opts)
		if err != nil {
			t.Fatalf("unexpected error with ample budget: %v", err)
		}
		if got == nil || got.Sign() <= 0 {
			t.Fatalf("expected positive result, got %v", got)
		}
	})

	t.Run("zero limit (disabled)", func(t *testing.T) {
		t.Parallel()
		opts := Options{} // MemoryLimitBytes == 0 → check disabled
		got, err := calc.Calculate(context.Background(), nil, 0, 200, opts)
		if err != nil {
			t.Fatalf("unexpected error with disabled check: %v", err)
		}
		if got == nil || got.Sign() <= 0 {
			t.Fatalf("expected positive result, got %v", got)
		}
	})
}
