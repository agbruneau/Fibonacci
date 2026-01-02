package bigfft

import (
	"crypto/rand"
	"math/big"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// Squaring Precision Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestSqrPrecisionSmall verifies FFT squaring precision for small numbers.
func TestSqrPrecisionSmall(t *testing.T) {
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
		{"999999999", "999999998000000001"},
	}

	for _, tc := range testCases {
		t.Run(tc.x, func(t *testing.T) {
			t.Parallel()
			x := new(big.Int)
			x.SetString(tc.x, 10)
			expected := new(big.Int)
			expected.SetString(tc.expected, 10)

			result, err := Sqr(x)
			if err != nil {
				t.Fatalf("Sqr failed: %v", err)
			}
			if result.Cmp(expected) != 0 {
				t.Errorf("%s²: expected %s, got %s", tc.x, expected.String(), result.String())
			}
		})
	}
}

// TestSqrPrecisionLarge verifies FFT squaring precision for large numbers.
func TestSqrPrecisionLarge(t *testing.T) {
	t.Parallel()
	xStr := "123456789012345678901234567890123456789012345678901234567890"

	x := new(big.Int)
	x.SetString(xStr, 10)

	// Calculate expected using standard multiplication
	expected := new(big.Int).Mul(x, x)

	// Calculate using our FFT squaring
	result, err := Sqr(x)
	if err != nil {
		t.Fatalf("Sqr failed: %v", err)
	}

	if result.Cmp(expected) != 0 {
		t.Errorf("Large squaring mismatch:\n  Expected: %s\n  Got:      %s",
			expected.String(), result.String())
	}
}

// TestSqrPrecisionVeryLarge tests with numbers large enough to force FFT.
func TestSqrPrecisionVeryLarge(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping very large number test in short mode")
	}

	// Create numbers large enough to definitely use FFT
	// 10000 digits
	xBytes := make([]byte, 5000)

	// Fill with random data
	if _, err := rand.Read(xBytes); err != nil {
		t.Fatalf("Failed to generate random data: %v", err)
	}

	x := new(big.Int).SetBytes(xBytes)

	// Calculate using standard multiplication
	expected := new(big.Int).Mul(x, x)

	// Calculate using FFT squaring
	result, err := Sqr(x)
	if err != nil {
		t.Fatalf("Sqr failed: %v", err)
	}

	if result.Cmp(expected) != 0 {
		t.Errorf("Very large squaring mismatch. Bit length: x=%d", x.BitLen())
	}
}

// TestSqrToPrecision verifies the SqrTo function.
func TestSqrToPrecision(t *testing.T) {
	t.Parallel()
	testCases := []string{
		"123",
		"999999999",
		"12345678901234567890",
		"123456789012345678901234567890123456789012345678901234567890",
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			t.Parallel()
			x := new(big.Int)
			x.SetString(tc, 10)

			expected := new(big.Int).Mul(x, x)

			z := new(big.Int)
			result, err := SqrTo(z, x)
			if err != nil {
				t.Fatalf("SqrTo failed: %v", err)
			}

			if result.Cmp(expected) != 0 {
				t.Errorf("SqrTo(%s): expected %s, got %s",
					tc, expected.String(), result.String())
			}

			// Verify z was used (same pointer)
			if result != z {
				t.Error("SqrTo did not return the destination pointer")
			}
		})
	}
}

// TestSqrToReuseBuffer tests that SqrTo correctly reuses the buffer.
func TestSqrToReuseBuffer(t *testing.T) {
	t.Parallel()
	x := big.NewInt(123456789)

	// Pre-allocate z with some capacity
	z := new(big.Int)
	z.SetInt64(999999999999)

	expected := new(big.Int).Mul(x, x)
	result, err := SqrTo(z, x)
	if err != nil {
		t.Fatalf("SqrTo failed: %v", err)
	}

	if result.Cmp(expected) != 0 {
		t.Errorf("SqrTo reuse failed: expected %s, got %s",
			expected.String(), result.String())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Squaring vs Multiplication Consistency Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestSqrVsMulConsistency verifies that Sqr(x) == Mul(x, x).
func TestSqrVsMulConsistency(t *testing.T) {
	t.Parallel()
	testCases := []string{
		"0",
		"1",
		"2",
		"12345",
		"999999999999999999",
		"123456789012345678901234567890",
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			t.Parallel()
			x := new(big.Int)
			x.SetString(tc, 10)

			sqrResult, err := Sqr(x)
			if err != nil {
				t.Fatalf("Sqr failed: %v", err)
			}
			mulResult, err := Mul(x, x)
			if err != nil {
				t.Fatalf("Mul failed: %v", err)
			}

			if sqrResult.Cmp(mulResult) != 0 {
				t.Errorf("Sqr(%s) != Mul(%s, %s):\n  Sqr: %s\n  Mul: %s",
					tc, tc, tc, sqrResult.String(), mulResult.String())
			}
		})
	}
}

// TestSqrToVsMulToConsistency verifies that SqrTo(z, x) == MulTo(z, x, x).
func TestSqrToVsMulToConsistency(t *testing.T) {
	t.Parallel()
	testCases := []string{
		"123",
		"999999999",
		"12345678901234567890",
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			t.Parallel()
			x := new(big.Int)
			x.SetString(tc, 10)

			z1 := new(big.Int)
			z2 := new(big.Int)

			sqrResult, err := SqrTo(z1, x)
			if err != nil {
				t.Fatalf("SqrTo failed: %v", err)
			}
			mulResult, err := MulTo(z2, x, x)
			if err != nil {
				t.Fatalf("MulTo failed: %v", err)
			}

			if sqrResult.Cmp(mulResult) != 0 {
				t.Errorf("SqrTo vs MulTo inconsistency for %s:\n  SqrTo: %s\n  MulTo: %s",
					tc, sqrResult.String(), mulResult.String())
			}
		})
	}
}

// TestSqrVeryLargeConsistency tests consistency for very large numbers.
func TestSqrVeryLargeConsistency(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping very large consistency test in short mode")
	}

	// Create a number large enough to trigger FFT
	xBytes := make([]byte, 5000)
	if _, err := rand.Read(xBytes); err != nil {
		t.Fatalf("Failed to generate random data: %v", err)
	}

	x := new(big.Int).SetBytes(xBytes)

	sqrResult, err := Sqr(x)
	if err != nil {
		t.Fatalf("Sqr failed: %v", err)
	}
	mulResult, err := Mul(x, x)
	if err != nil {
		t.Fatalf("Mul failed: %v", err)
	}

	if sqrResult.Cmp(mulResult) != 0 {
		t.Errorf("Very large Sqr vs Mul inconsistency. Bit length: %d", x.BitLen())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Edge Case Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestSqrZero verifies squaring of zero.
func TestSqrZero(t *testing.T) {
	t.Parallel()
	zero := big.NewInt(0)

	result, err := Sqr(zero)
	if err != nil {
		t.Fatalf("Sqr failed: %v", err)
	}
	if result.Cmp(zero) != 0 {
		t.Errorf("0² = %s, expected 0", result.String())
	}
}

// TestSqrOne verifies squaring of one.
func TestSqrOne(t *testing.T) {
	t.Parallel()
	one := big.NewInt(1)

	result, err := Sqr(one)
	if err != nil {
		t.Fatalf("Sqr failed: %v", err)
	}
	if result.Cmp(one) != 0 {
		t.Errorf("1² = %s, expected 1", result.String())
	}
}

// TestSqrNegative verifies squaring of negative numbers.
func TestSqrNegative(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		x, expected string
	}{
		{"-1", "1"},
		{"-2", "4"},
		{"-12345", "152399025"},
		{"-999999999", "999999998000000001"},
	}

	for _, tc := range testCases {
		t.Run(tc.x, func(t *testing.T) {
			t.Parallel()
			x := new(big.Int)
			x.SetString(tc.x, 10)
			expected := new(big.Int)
			expected.SetString(tc.expected, 10)

			result, err := Sqr(x)
			if err != nil {
				t.Fatalf("Sqr failed: %v", err)
			}
			if result.Cmp(expected) != 0 {
				t.Errorf("(%s)²: expected %s, got %s", tc.x, expected.String(), result.String())
			}

			// Verify result is always non-negative
			if result.Sign() < 0 {
				t.Errorf("(%s)² produced negative result: %s", tc.x, result.String())
			}
		})
	}
}

// TestSqrPowerOfTwo verifies squaring of powers of two.
func TestSqrPowerOfTwo(t *testing.T) {
	t.Parallel()
	for i := 0; i < 20; i++ {
		x := new(big.Int).Lsh(big.NewInt(1), uint(i)) // 2^i

		result, err := Sqr(x)
		if err != nil {
			t.Fatalf("Sqr failed: %v", err)
		}
		expected := new(big.Int).Lsh(big.NewInt(1), uint(2*i)) // 2^(2i)

		if result.Cmp(expected) != 0 {
			t.Errorf("(2^%d)² = %s, expected 2^%d = %s",
				i, result.String(), 2*i, expected.String())
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Benchmark Tests
// ─────────────────────────────────────────────────────────────────────────────

// BenchmarkSqrSmall benchmarks squaring of small numbers.
func BenchmarkSqrSmall(b *testing.B) {
	b.ReportAllocs()
	x := new(big.Int)
	x.SetString("12345678901234567890", 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Sqr(x)
	}
}

// BenchmarkSqrMedium benchmarks squaring of medium numbers.
func BenchmarkSqrMedium(b *testing.B) {
	b.ReportAllocs()
	xBytes := make([]byte, 1000)
	rand.Read(xBytes)
	x := new(big.Int).SetBytes(xBytes)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Sqr(x)
	}
}

// BenchmarkSqrLarge benchmarks squaring of large numbers.
func BenchmarkSqrLarge(b *testing.B) {
	b.ReportAllocs()
	if testing.Short() {
		b.Skip("Skipping large benchmark in short mode")
	}

	xBytes := make([]byte, 10000)
	rand.Read(xBytes)
	x := new(big.Int).SetBytes(xBytes)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Sqr(x)
	}
}

// BenchmarkSqrToReuse benchmarks SqrTo with buffer reuse.
func BenchmarkSqrToReuse(b *testing.B) {
	b.ReportAllocs()
	x := new(big.Int)
	x.SetString("12345678901234567890123456789012345678901234567890", 10)
	z := new(big.Int)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = SqrTo(z, x)
	}
}

// BenchmarkSqrVsMul compares Sqr performance against Mul(x,x).
func BenchmarkSqrVsMul(b *testing.B) {
	b.ReportAllocs()
	xBytes := make([]byte, 5000)
	rand.Read(xBytes)
	x := new(big.Int).SetBytes(xBytes)

	b.Run("Sqr", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = Sqr(x)
		}
	})

	b.Run("Mul", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Mul(x, x)
		}
	})
}
