package orchestration

import (
	"context"
	"errors"
	"testing"
)

// TestComputeLastDigits exercises the pure last-digits modular computation.
func TestComputeLastDigits(t *testing.T) {
	t.Parallel()

	t.Run("F(10) last 3 digits is 055 (zero-padded)", func(t *testing.T) {
		t.Parallel()
		got, err := ComputeLastDigits(context.Background(), 10, 3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// F(10) = 55, last 3 digits zero-padded = "055"
		if got.Digits != "055" {
			t.Errorf("expected digits %q, got %q", "055", got.Digits)
		}
		if got.Value == nil || got.Value.Int64() != 55 {
			t.Errorf("expected value 55, got %v", got.Value)
		}
	})

	t.Run("F(100) last 5 digits is 15075", func(t *testing.T) {
		t.Parallel()
		got, err := ComputeLastDigits(context.Background(), 100, 5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// F(100) = 354224848179261915075
		if got.Digits != "15075" {
			t.Errorf("expected digits %q, got %q", "15075", got.Digits)
		}
	})

	t.Run("F(0) last 4 digits is 0000", func(t *testing.T) {
		t.Parallel()
		got, err := ComputeLastDigits(context.Background(), 0, 4)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Digits != "0000" {
			t.Errorf("expected digits %q, got %q", "0000", got.Digits)
		}
	})

	t.Run("invalid k returns error", func(t *testing.T) {
		t.Parallel()
		if _, err := ComputeLastDigits(context.Background(), 10, 0); err == nil {
			t.Error("expected error for k=0")
		}
		if _, err := ComputeLastDigits(context.Background(), 10, -1); err == nil {
			t.Error("expected error for k<0")
		}
	})

	t.Run("cancelled context is reported", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := ComputeLastDigits(ctx, 100, 3)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})
}
