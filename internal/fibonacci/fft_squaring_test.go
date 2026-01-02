package fibonacci

import (
	"crypto/rand"
	"math/big"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// smartSquare Precision Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestSmartSquarePrecisionSmall verifies smartSquare precision for small numbers.
func TestSmartSquarePrecisionSmall(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		x, expected string
	}{
		{"0", "0"},
		{"1", "1"},
		{"2", "4"},
		{"3", "9"},
		{"10", "100"},
		{"123", "15129"},
		{"999", "998001"},
		{"12345", "152399025"},
	}

	for _, tc := range testCases {
		x := new(big.Int)
		x.SetString(tc.x, 10)
		expected := new(big.Int)
		expected.SetString(tc.expected, 10)

		z := new(big.Int)
		result, err := smartSquare(z, x, 0, 0) // threshold 0 means use standard multiplication
		if err != nil {
			t.Fatalf("smartSquare failed: %v", err)
		}

		if result.Cmp(expected) != 0 {
			t.Errorf("smartSquare(%s): expected %s, got %s", tc.x, expected.String(), result.String())
		}
	}
}

// TestSmartSquareWithFFTThreshold tests smartSquare with various thresholds.
func TestSmartSquareWithFFTThreshold(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		x         string
		threshold int
	}{
		{"12345", 0},       // No FFT
		{"12345", 100},     // Threshold higher than number
		{"12345", 10},      // Threshold lower than number
		{"999999999", 0},   // Larger number, no FFT
		{"999999999", 100}, // Larger number with threshold
		{"999999999", 20},  // Larger number, may use FFT
	}

	for _, tc := range testCases {
		x := new(big.Int)
		x.SetString(tc.x, 10)
		expected := new(big.Int).Mul(x, x)

		z := new(big.Int)
		result, err := smartSquare(z, x, tc.threshold, 0)
		if err != nil {
			t.Fatalf("smartSquare failed: %v", err)
		}

		if result.Cmp(expected) != 0 {
			t.Errorf("smartSquare(%s, threshold=%d): expected %s, got %s",
				tc.x, tc.threshold, expected.String(), result.String())
		}
	}
}

// TestSmartSquareLarge tests smartSquare with large numbers.
func TestSmartSquareLarge(t *testing.T) {
	t.Parallel()
	xStr := "123456789012345678901234567890123456789012345678901234567890"

	x := new(big.Int)
	x.SetString(xStr, 10)
	expected := new(big.Int).Mul(x, x)

	// Test with FFT threshold that forces FFT usage
	z := new(big.Int)
	result, err := smartSquare(z, x, 100, 0)
	if err != nil {
		t.Fatalf("smartSquare failed: %v", err)
	}

	if result.Cmp(expected) != 0 {
		t.Errorf("smartSquare large number mismatch:\n  Expected: %s\n  Got:      %s",
			expected.String(), result.String())
	}
}

// TestSmartSquareVeryLarge tests with numbers large enough to benefit from FFT.
func TestSmartSquareVeryLarge(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping very large number test in short mode")
	}

	xBytes := make([]byte, 5000)
	if _, err := rand.Read(xBytes); err != nil {
		t.Fatalf("Failed to generate random data: %v", err)
	}

	x := new(big.Int).SetBytes(xBytes)
	expected := new(big.Int).Mul(x, x)

	// Test with FFT threshold
	z := new(big.Int)
	result, err := smartSquare(z, x, 1000, 0)
	if err != nil {
		t.Fatalf("smartSquare failed: %v", err)
	}

	if result.Cmp(expected) != 0 {
		t.Errorf("smartSquare very large number mismatch. Bit length: %d", x.BitLen())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// smartSquare vs smartMultiply Consistency Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestSmartSquareVsSmartMultiplyConsistency verifies that smartSquare(z, x) == smartMultiply(z, x, x).
func TestSmartSquareVsSmartMultiplyConsistency(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		x         string
		threshold int
	}{
		{"0", 0},
		{"1", 0},
		{"12345", 0},
		{"12345", 100},
		{"999999999999999999", 0},
		{"999999999999999999", 1000},
	}

	for _, tc := range testCases {
		x := new(big.Int)
		x.SetString(tc.x, 10)

		z1 := new(big.Int)
		z2 := new(big.Int)

		sqrResult, err := smartSquare(z1, x, tc.threshold, 0)
		if err != nil {
			t.Fatalf("smartSquare failed: %v", err)
		}
		mulResult, err := smartMultiply(z2, x, x, tc.threshold, 0)
		if err != nil {
			t.Fatalf("smartMultiply failed: %v", err)
		}

		if sqrResult.Cmp(mulResult) != 0 {
			t.Errorf("smartSquare vs smartMultiply inconsistency for %s (threshold=%d):\n  smartSquare: %s\n  smartMultiply: %s",
				tc.x, tc.threshold, sqrResult.String(), mulResult.String())
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// sqrFFT Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestSqrFFTPrecision verifies sqrFFT precision.
func TestSqrFFTPrecision(t *testing.T) {
	t.Parallel()
	testCases := []string{
		"12345",
		"999999999",
		"12345678901234567890",
		"123456789012345678901234567890123456789012345678901234567890",
	}

	for _, tc := range testCases {
		x := new(big.Int)
		x.SetString(tc, 10)
		expected := new(big.Int).Mul(x, x)

		result, err := sqrFFT(x)
		if err != nil {
			t.Fatalf("sqrFFT failed: %v", err)
		}

		if result.Cmp(expected) != 0 {
			t.Errorf("sqrFFT(%s): expected %s, got %s",
				tc, expected.String(), result.String())
		}
	}
}

// TestSqrFFTVsMulFFTConsistency verifies that sqrFFT(x) == mulFFT(x, x).
func TestSqrFFTVsMulFFTConsistency(t *testing.T) {
	t.Parallel()
	testCases := []string{
		"12345",
		"999999999",
		"12345678901234567890",
	}

	for _, tc := range testCases {
		x := new(big.Int)
		x.SetString(tc, 10)

		sqrResult, err := sqrFFT(x)
		if err != nil {
			t.Fatalf("sqrFFT failed: %v", err)
		}
		mulResult, err := mulFFT(x, x)
		if err != nil {
			t.Fatalf("mulFFT failed: %v", err)
		}

		if sqrResult.Cmp(mulResult) != 0 {
			t.Errorf("sqrFFT vs mulFFT inconsistency for %s:\n  sqrFFT: %s\n  mulFFT: %s",
				tc, sqrResult.String(), mulResult.String())
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Edge Case Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestSmartSquareZero verifies squaring of zero.
func TestSmartSquareZero(t *testing.T) {
	t.Parallel()
	zero := big.NewInt(0)
	z := new(big.Int)

	result, err := smartSquare(z, zero, 0, 0)
	if err != nil {
		t.Fatalf("smartSquare failed: %v", err)
	}
	if result.Cmp(zero) != 0 {
		t.Errorf("smartSquare(0) = %s, expected 0", result.String())
	}
}

// TestSmartSquareNegative verifies squaring of negative numbers.
func TestSmartSquareNegative(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		x, expected string
	}{
		{"-1", "1"},
		{"-2", "4"},
		{"-12345", "152399025"},
	}

	for _, tc := range testCases {
		x := new(big.Int)
		x.SetString(tc.x, 10)
		expected := new(big.Int)
		expected.SetString(tc.expected, 10)

		z := new(big.Int)
		result, err := smartSquare(z, x, 0, 0)
		if err != nil {
			t.Fatalf("smartSquare failed: %v", err)
		}

		if result.Cmp(expected) != 0 {
			t.Errorf("smartSquare(%s): expected %s, got %s",
				tc.x, expected.String(), result.String())
		}

		// Verify result is always non-negative
		if result.Sign() < 0 {
			t.Errorf("smartSquare(%s) produced negative result: %s", tc.x, result.String())
		}
	}
}

// TestSmartSquareBufferReuse verifies that z buffer is reused.
func TestSmartSquareBufferReuse(t *testing.T) {
	t.Parallel()
	x := big.NewInt(12345)

	// Pre-allocate z with some capacity
	z := new(big.Int)
	z.SetInt64(999999999999)

	expected := new(big.Int).Mul(x, x)
	result, err := smartSquare(z, x, 0, 0)
	if err != nil {
		t.Fatalf("smartSquare failed: %v", err)
	}

	if result.Cmp(expected) != 0 {
		t.Errorf("smartSquare buffer reuse failed: expected %s, got %s",
			expected.String(), result.String())
	}

	// Verify z was used (same pointer)
	if result != z {
		t.Error("smartSquare did not return the destination pointer")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Benchmark Tests
// ─────────────────────────────────────────────────────────────────────────────

// BenchmarkSmartSquareSmall benchmarks smartSquare for small numbers.
func BenchmarkSmartSquareSmall(b *testing.B) {
	x := new(big.Int)
	x.SetString("12345678901234567890", 10)
	z := new(big.Int)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		smartSquare(z, x, 0, 0)
	}
}

// BenchmarkSmartSquareMedium benchmarks smartSquare for medium numbers.
func BenchmarkSmartSquareMedium(b *testing.B) {
	xBytes := make([]byte, 1000)
	rand.Read(xBytes)
	x := new(big.Int).SetBytes(xBytes)
	z := new(big.Int)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		smartSquare(z, x, 0, 0)
	}
}

// BenchmarkSmartSquareLarge benchmarks smartSquare for large numbers.
func BenchmarkSmartSquareLarge(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping large benchmark in short mode")
	}

	xBytes := make([]byte, 10000)
	rand.Read(xBytes)
	x := new(big.Int).SetBytes(xBytes)
	z := new(big.Int)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		smartSquare(z, x, 1000, 0)
	}
}

// BenchmarkSmartSquareVsSmartMultiply compares smartSquare performance against smartMultiply(x,x).
func BenchmarkSmartSquareVsSmartMultiply(b *testing.B) {
	xBytes := make([]byte, 5000)
	rand.Read(xBytes)
	x := new(big.Int).SetBytes(xBytes)

	b.Run("smartSquare", func(b *testing.B) {
		z := new(big.Int)
		for i := 0; i < b.N; i++ {
			smartSquare(z, x, 1000, 0)
		}
	})

	b.Run("smartMultiply", func(b *testing.B) {
		z := new(big.Int)
		for i := 0; i < b.N; i++ {
			smartMultiply(z, x, x, 1000, 0)
		}
	})
}
