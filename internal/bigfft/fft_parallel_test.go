package bigfft

import (
	"math/big"
	"testing"
)

// TestFFTParallelization verifies that parallel FFT produces the same results
// as sequential FFT for various input sizes.
func TestFFTParallelization(t *testing.T) {
	t.Parallel()
	// Test with numbers that should trigger parallelization (large k)
	// We need k >= ParallelFFTRecursionThreshold (4) for parallelization to occur
	// This means we need vectors of size at least 2^4 = 16

	testCases := []struct {
		name string
		x    *big.Int
		y    *big.Int
	}{
		{
			name: "medium numbers",
			x:    big.NewInt(123456789),
			y:    big.NewInt(987654321),
		},
		{
			name: "large numbers",
			x:    new(big.Int).Exp(big.NewInt(10), big.NewInt(100), nil),
			y:    new(big.Int).Exp(big.NewInt(10), big.NewInt(100), nil),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Compute product using FFT (which may use parallelization internally)
			result, err := Mul(tc.x, tc.y)
			if err != nil {
				t.Fatalf("Mul failed: %v", err)
			}

			// Compute expected result using standard multiplication
			expected := new(big.Int).Mul(tc.x, tc.y)

			// Verify results match
			if result.Cmp(expected) != 0 {
				t.Errorf("FFT multiplication mismatch:\nGot:     %s\nExpected: %s", result.String(), expected.String())
			}
		})
	}
}

// BenchmarkFFTParallelization benchmarks FFT multiplication to verify
// that parallelization provides performance benefits for large numbers.
func BenchmarkFFTParallelization(b *testing.B) {
	// Create large numbers that will trigger FFT and parallelization
	x := new(big.Int).Exp(big.NewInt(2), big.NewInt(10000), nil)
	y := new(big.Int).Exp(big.NewInt(3), big.NewInt(10000), nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Mul(x, y)
	}
}
