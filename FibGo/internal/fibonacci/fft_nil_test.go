package fibonacci

import (
	"math/big"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// Nil z Parameter Tests
// ─────────────────────────────────────────────────────────────────────────────
//
// These tests verify that smartMultiply and smartSquare handle nil z parameter
// correctly, as specified by the MultiplicationStrategy interface contract.
// Before the fix, these functions would panic with a nil pointer dereference.

// TestSmartMultiplyNilZ verifies that smartMultiply handles nil z parameter.
// This test would PANIC before the fix was applied.
func TestSmartMultiplyNilZ(t *testing.T) {
	x := big.NewInt(12345)
	y := big.NewInt(67890)
	expected := new(big.Int).Mul(x, y)

	// Test with nil z and threshold 0 (forces non-FFT path)
	result, err := smartMultiply(nil, x, y, 0, 0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Cmp(expected) != 0 {
		t.Errorf("smartMultiply(nil, x, y, 0) = %s, want %s", result.String(), expected.String())
	}

	// Test with nil z and high threshold (still forces non-FFT path for small numbers)
	result2, err := smartMultiply(nil, x, y, 1000000, 0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result2 == nil {
		t.Fatal("Expected non-nil result")
	}
	if result2.Cmp(expected) != 0 {
		t.Errorf("smartMultiply(nil, x, y, 1000000) = %s, want %s", result2.String(), expected.String())
	}
}

// TestSmartSquareNilZ verifies that smartSquare handles nil z parameter.
// This test would PANIC before the fix was applied.
func TestSmartSquareNilZ(t *testing.T) {
	x := big.NewInt(12345)
	expected := new(big.Int).Mul(x, x)

	// Test with nil z and threshold 0 (forces non-FFT path)
	result, err := smartSquare(nil, x, 0, 0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Cmp(expected) != 0 {
		t.Errorf("smartSquare(nil, x, 0) = %s, want %s", result.String(), expected.String())
	}

	// Test with nil z and high threshold (still forces non-FFT path for small numbers)
	result2, err := smartSquare(nil, x, 1000000, 0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result2 == nil {
		t.Fatal("Expected non-nil result")
	}
	if result2.Cmp(expected) != 0 {
		t.Errorf("smartSquare(nil, x, 1000000) = %s, want %s", result2.String(), expected.String())
	}
}

// TestAdaptiveStrategyNilZ verifies that the AdaptiveStrategy correctly handles
// nil z parameter, which is allowed by the MultiplicationStrategy interface.
// This test would PANIC before the fix was applied.
func TestAdaptiveStrategyNilZ(t *testing.T) {
	strategy := &AdaptiveStrategy{}
	opts := Options{FFTThreshold: 0} // Disable FFT to force the standard path

	t.Run("Multiply", func(t *testing.T) {
		x := big.NewInt(999)
		y := big.NewInt(1001)
		expected := new(big.Int).Mul(x, y) // 999 * 1001 = 999999

		result, err := strategy.Multiply(nil, x, y, opts)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("Expected non-nil result")
		}
		if result.Cmp(expected) != 0 {
			t.Errorf("AdaptiveStrategy.Multiply(nil, x, y) = %s, want %s", result.String(), expected.String())
		}
	})

	t.Run("Square", func(t *testing.T) {
		x := big.NewInt(999)
		expected := new(big.Int).Mul(x, x) // 999^2 = 998001

		result, err := strategy.Square(nil, x, opts)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("Expected non-nil result")
		}
		if result.Cmp(expected) != 0 {
			t.Errorf("AdaptiveStrategy.Square(nil, x) = %s, want %s", result.String(), expected.String())
		}
	})
}

// TestStrategyConsistencyWithNilZ verifies that all strategies produce consistent
// results when z is nil. This ensures the fix maintains API consistency.
func TestStrategyConsistencyWithNilZ(t *testing.T) {
	strategies := map[string]MultiplicationStrategy{
		"Adaptive":  &AdaptiveStrategy{},
		"Karatsuba": &KaratsubaStrategy{},
		"FFTOnly":   &FFTOnlyStrategy{},
	}

	x := big.NewInt(123456)
	y := big.NewInt(789012)
	expectedMul := new(big.Int).Mul(x, y)
	expectedSqr := new(big.Int).Mul(x, x)

	opts := Options{FFTThreshold: 0}

	t.Run("Multiply", func(t *testing.T) {
		for name, strategy := range strategies {
			t.Run(name, func(t *testing.T) {
				result, err := strategy.Multiply(nil, x, y, opts)
				if err != nil {
					t.Fatalf("%s.Multiply returned error: %v", name, err)
				}
				if result == nil {
					t.Fatalf("%s.Multiply returned nil result", name)
				}
				if result.Cmp(expectedMul) != 0 {
					t.Errorf("%s.Multiply(nil, x, y) = %s, want %s", name, result.String(), expectedMul.String())
				}
			})
		}
	})

	t.Run("Square", func(t *testing.T) {
		for name, strategy := range strategies {
			t.Run(name, func(t *testing.T) {
				result, err := strategy.Square(nil, x, opts)
				if err != nil {
					t.Fatalf("%s.Square returned error: %v", name, err)
				}
				if result == nil {
					t.Fatalf("%s.Square returned nil result", name)
				}
				if result.Cmp(expectedSqr) != 0 {
					t.Errorf("%s.Square(nil, x) = %s, want %s", name, result.String(), expectedSqr.String())
				}
			})
		}
	})
}
