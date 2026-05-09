package fibonacci

import (
	"math/big"
	"testing"
)

// TestKaratsubaIntegration verifies that smartMultiply correctly selects
// between math/big (Karatsuba) and FFT tiers based on the FFT threshold.
func TestKaratsubaIntegration(t *testing.T) {
	t.Parallel()
	x := new(big.Int).Lsh(big.NewInt(1), 5000) // ~5000 bits
	y := new(big.Int).Lsh(big.NewInt(1), 5000)

	expected := new(big.Int).Mul(x, y)

	t.Run("MathBigTier", func(t *testing.T) {
		// FFT threshold high — forces math/big path
		z := new(big.Int)
		result, err := smartMultiply(z, x, y, 1000000)
		if err != nil {
			t.Fatalf("smartMultiply failed: %v", err)
		}

		if result.Cmp(expected) != 0 {
			t.Errorf("Result mismatch in math/big tier")
		}
	})

	t.Run("FFTTier", func(t *testing.T) {
		// FFT threshold low — forces FFT path
		z := new(big.Int)
		result, err := smartMultiply(z, x, y, 1024)
		if err != nil {
			t.Fatalf("smartMultiply failed: %v", err)
		}

		if result.Cmp(expected) != 0 {
			t.Errorf("Result mismatch in FFT tier")
		}
	})
}
