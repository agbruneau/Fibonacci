package fibonacci

import (
	"strings"
	"testing"
)

func TestFibCalculator_Name(t *testing.T) {
	t.Parallel()

	t.Run("Name delegates to core calculator", func(t *testing.T) {
		t.Parallel()
		core := &OptimizedFastDoubling{}
		calc := NewCalculator(core)

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

	t.Run("Name with MatrixExponentiation", func(t *testing.T) {
		t.Parallel()
		core := &MatrixExponentiation{}
		calc := NewCalculator(core)

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
		calc := NewCalculator(core)

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
