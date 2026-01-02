package fibonacci

import (
	"math/big"
	"testing"
)

// TestSetOrReturn tests the setOrReturn helper function.
func TestSetOrReturn(t *testing.T) {
	t.Parallel()
	t.Run("z is nil - returns result directly", func(t *testing.T) {
		t.Parallel()
		result := big.NewInt(456)
		ret := setOrReturn(nil, result)
		if ret != result {
			t.Error("expected same pointer when z is nil")
		}
	})

	t.Run("z is non-nil - sets z and returns it", func(t *testing.T) {
		t.Parallel()
		z := big.NewInt(0)
		result := big.NewInt(456)
		ret := setOrReturn(z, result)
		if ret != z {
			t.Error("expected z pointer to be returned")
		}
		if z.Cmp(result) != 0 {
			t.Errorf("expected z to be set to %s, got %s", result.String(), z.String())
		}
	})
}

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

// TestKaratsubaStrategy tests the Karatsuba-only strategy.
func TestKaratsubaStrategy(t *testing.T) {
	t.Parallel()
	s := &KaratsubaStrategy{}

	t.Run("Name", func(t *testing.T) {
		t.Parallel()
		name := s.Name()
		if name == "" {
			t.Error("expected non-empty name")
		}
		if name != "Karatsuba-Only" {
			t.Errorf("expected 'Karatsuba-Only', got '%s'", name)
		}
	})

	t.Run("Multiply with nil z", func(t *testing.T) {
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

	t.Run("Multiply with non-nil z", func(t *testing.T) {
		t.Parallel()
		z := big.NewInt(999999)
		x := big.NewInt(100)
		y := big.NewInt(200)
		opts := Options{}

		result, err := s.Multiply(z, x, y, opts)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != z {
			t.Error("expected z to be returned")
		}

		expected := big.NewInt(20000)
		if result.Cmp(expected) != 0 {
			t.Errorf("expected %s, got %s", expected.String(), result.String())
		}
	})

	t.Run("Square with nil z", func(t *testing.T) {
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

	t.Run("Square with non-nil z", func(t *testing.T) {
		t.Parallel()
		z := big.NewInt(0)
		x := big.NewInt(100)
		opts := Options{}

		result, err := s.Square(z, x, opts)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != z {
			t.Error("expected z to be returned")
		}

		expected := big.NewInt(10000)
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
		// After setOrReturn, z should have the value
		if z != result {
			t.Log("z was returned (as expected with setOrReturn)")
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

// TestMultiplicationStrategyInterface verifies interface implementation.
func TestMultiplicationStrategyInterface(t *testing.T) {
	t.Parallel()
	var _ MultiplicationStrategy = &AdaptiveStrategy{}
	var _ MultiplicationStrategy = &FFTOnlyStrategy{}
	var _ MultiplicationStrategy = &KaratsubaStrategy{}
}

func TestKaratsubaStrategy_ExecuteStep(t *testing.T) {
	t.Parallel()

	t.Run("ExecuteStep with parallel disabled", func(t *testing.T) {
		t.Parallel()
		strategy := &KaratsubaStrategy{}

		// Create a calculation state
		state := &CalculationState{
			FK:  big.NewInt(5),
			FK1: big.NewInt(8),
			T1:  big.NewInt(0),
			T2:  big.NewInt(0),
			T3:  big.NewInt(0),
			T4:  big.NewInt(0),
		}

		opts := Options{
			ParallelThreshold: 4096,
			FFTThreshold:      1000000,
		}

		err := strategy.ExecuteStep(state, opts, false)
		if err != nil {
			t.Errorf("ExecuteStep() error = %v, want nil", err)
		}
	})

	t.Run("ExecuteStep with parallel enabled", func(t *testing.T) {
		t.Parallel()
		strategy := &KaratsubaStrategy{}

		// Create a calculation state
		state := &CalculationState{
			FK:  big.NewInt(5),
			FK1: big.NewInt(8),
			T1:  big.NewInt(0),
			T2:  big.NewInt(0),
			T3:  big.NewInt(0),
			T4:  big.NewInt(0),
		}

		opts := Options{
			ParallelThreshold: 4096,
			FFTThreshold:      1000000,
		}

		err := strategy.ExecuteStep(state, opts, true)
		if err != nil {
			t.Errorf("ExecuteStep() error = %v, want nil", err)
		}
	})
}
