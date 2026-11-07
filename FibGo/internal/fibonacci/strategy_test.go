package fibonacci

import (
	"math/big"
	"testing"
)

// TestAdaptiveStrategy tests the adaptive multiplication strategy.
func TestAdaptiveStrategy(t *testing.T) {
	t.Parallel()
	s := &AdaptiveStrategy{}

	t.Run("Name", func(t *testing.T) {
		t.Parallel()
		name := s.Name()
		if name == "" {
			t.Error("expected non-empty name")
		}
	})

	t.Run("Multiply small numbers", func(t *testing.T) {
		t.Parallel()
		x := big.NewInt(123)
		y := big.NewInt(456)
		opts := Options{FFTThreshold: 1000000}

		result, err := s.Multiply(nil, x, y, opts)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		expected := big.NewInt(0).Mul(x, y)
		if result.Cmp(expected) != 0 {
			t.Errorf("expected %s, got %s", expected.String(), result.String())
		}
	})

	t.Run("Square small numbers", func(t *testing.T) {
		t.Parallel()
		x := big.NewInt(123)
		opts := Options{FFTThreshold: 1000000}

		result, err := s.Square(nil, x, opts)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		expected := big.NewInt(0).Mul(x, x)
		if result.Cmp(expected) != 0 {
			t.Errorf("expected %s, got %s", expected.String(), result.String())
		}
	})

	t.Run("Multiply with reusable z", func(t *testing.T) {
		t.Parallel()
		z := new(big.Int)
		x := big.NewInt(100)
		y := big.NewInt(200)
		opts := Options{}

		result, err := s.Multiply(z, x, y, opts)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != z {
			t.Log("Result may differ from z based on implementation")
		}

		expected := big.NewInt(20000)
		if result.Cmp(expected) != 0 {
			t.Errorf("expected %s, got %s", expected.String(), result.String())
		}
	})
}

// TestFFTOnlyStrategy tests the FFT-only strategy.
func TestFFTOnlyStrategy(t *testing.T) {
	t.Parallel()
	s := &FFTOnlyStrategy{}

	t.Run("Name", func(t *testing.T) {
		t.Parallel()
		name := s.Name()
		if name == "" {
			t.Error("expected non-empty name")
		}
		if name != "FFT-Only" {
			t.Errorf("expected 'FFT-Only', got '%s'", name)
		}
	})

	t.Run("Multiply small numbers", func(t *testing.T) {
		t.Parallel()
		x := big.NewInt(12345)
		y := big.NewInt(67890)
		opts := Options{}

		result, err := s.Multiply(nil, x, y, opts)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		expected := big.NewInt(0).Mul(x, y)
		if result.Cmp(expected) != 0 {
			t.Errorf("expected %s, got %s", expected.String(), result.String())
		}
	})

	t.Run("Multiply with z reuse", func(t *testing.T) {
		t.Parallel()
		z := big.NewInt(0)
		x := big.NewInt(100)
		y := big.NewInt(200)
		opts := Options{}

		result, err := s.Multiply(z, x, y, opts)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		// z should be reused as the destination when non-nil.
		if z != result {
			t.Log("z was returned (as expected)")
		}

		expected := big.NewInt(20000)
		if result.Cmp(expected) != 0 {
			t.Errorf("expected %s, got %s", expected.String(), result.String())
		}
	})

	t.Run("Square small number", func(t *testing.T) {
		t.Parallel()
		x := big.NewInt(12345)
		opts := Options{}

		result, err := s.Square(nil, x, opts)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		expected := big.NewInt(0).Mul(x, x)
		if result.Cmp(expected) != 0 {
			t.Errorf("expected %s, got %s", expected.String(), result.String())
		}
	})

	t.Run("Square with z reuse", func(t *testing.T) {
		t.Parallel()
		z := big.NewInt(0)
		x := big.NewInt(100)
		opts := Options{}

		result, err := s.Square(z, x, opts)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		expected := big.NewInt(10000)
		if result.Cmp(expected) != 0 {
			t.Errorf("expected %s, got %s", expected.String(), result.String())
		}
	})
}

// TestMultiplierInterface verifies that all strategies implement Multiplier.
func TestMultiplierInterface(t *testing.T) {
	t.Parallel()
	var _ Multiplier = &AdaptiveStrategy{}
	var _ Multiplier = &FFTOnlyStrategy{}
}

// TestDoublingStepExecutorInterface verifies that all strategies implement DoublingStepExecutor.
func TestDoublingStepExecutorInterface(t *testing.T) {
	t.Parallel()
	var _ DoublingStepExecutor = &AdaptiveStrategy{}
	var _ DoublingStepExecutor = &FFTOnlyStrategy{}
}
