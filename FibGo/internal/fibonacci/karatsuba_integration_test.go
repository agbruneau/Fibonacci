package fibonacci

import (
	"math/big"
	"testing"
)

// TestKaratsubaIntegration verifies that smartMultiply correctly enters the
// Karatsuba tier when thresholds are set appropriately.
func TestKaratsubaIntegration(t *testing.T) {
	x := new(big.Int).Lsh(big.NewInt(1), 5000) // ~5000 bits
	y := new(big.Int).Lsh(big.NewInt(1), 5000)

	expected := new(big.Int).Mul(x, y)

	t.Run("KaratsubaTierActive", func(t *testing.T) {
		// FFT Threshold high (off), Karatsuba Threshold low (on)
		fftThreshold := 1000000
		karatsubaThreshold := 1024

		z := new(big.Int)
		result, err := smartMultiply(z, x, y, fftThreshold, karatsubaThreshold)
		if err != nil {
			t.Fatalf("smartMultiply failed: %v", err)
		}

		if result.Cmp(expected) != 0 {
			t.Errorf("Result mismatch in Karatsuba tier")
		}
	})

	t.Run("StandardTierActive", func(t *testing.T) {
		// Both thresholds high (off)
		fftThreshold := 1000000
		karatsubaThreshold := 1000000

		z := new(big.Int)
		result, err := smartMultiply(z, x, y, fftThreshold, karatsubaThreshold)
		if err != nil {
			t.Fatalf("smartMultiply failed: %v", err)
		}

		if result.Cmp(expected) != 0 {
			t.Errorf("Result mismatch in Standard tier")
		}
	})

	t.Run("FFTTierActive", func(t *testing.T) {
		// FFT Threshold low (on)
		fftThreshold := 1024
		karatsubaThreshold := 0

		z := new(big.Int)
		result, err := smartMultiply(z, x, y, fftThreshold, karatsubaThreshold)
		if err != nil {
			t.Fatalf("smartMultiply failed: %v", err)
		}

		if result.Cmp(expected) != 0 {
			t.Errorf("Result mismatch in FFT tier")
		}
	})
}
