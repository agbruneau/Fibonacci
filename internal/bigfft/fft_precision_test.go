package bigfft

import (
	"crypto/rand"
	"math/big"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// Precision Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestMulPrecisionSmall verifies FFT multiplication precision for small numbers.
func TestMulPrecisionSmall(t *testing.T) {
	testCases := []struct {
		a, b, expected string
	}{
		{"0", "0", "0"},
		{"1", "1", "1"},
		{"2", "3", "6"},
		{"123", "456", "56088"},
		{"999", "999", "998001"},
		{"12345", "67890", "838102050"},
		{"999999999", "999999999", "999999998000000001"},
	}

	for _, tc := range testCases {
		a := new(big.Int)
		a.SetString(tc.a, 10)
		b := new(big.Int)
		b.SetString(tc.b, 10)
		expected := new(big.Int)
		expected.SetString(tc.expected, 10)

		result := Mul(a, b)
		if result.Cmp(expected) != 0 {
			t.Errorf("%s * %s: expected %s, got %s", tc.a, tc.b, expected.String(), result.String())
		}
	}
}

// TestMulPrecisionLarge verifies FFT multiplication precision for large numbers.
func TestMulPrecisionLarge(t *testing.T) {
	// Create large numbers that will trigger FFT
	aStr := "123456789012345678901234567890123456789012345678901234567890"
	bStr := "987654321098765432109876543210987654321098765432109876543210"

	a := new(big.Int)
	a.SetString(aStr, 10)
	b := new(big.Int)
	b.SetString(bStr, 10)

	// Calculate expected using standard multiplication
	expected := new(big.Int).Mul(a, b)

	// Calculate using our FFT multiplication
	result := Mul(a, b)

	if result.Cmp(expected) != 0 {
		t.Errorf("Large multiplication mismatch:\n  Expected: %s\n  Got:      %s",
			expected.String(), result.String())
	}
}

// TestMulPrecisionVeryLarge tests with numbers large enough to force FFT.
func TestMulPrecisionVeryLarge(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping very large number test in short mode")
	}

	// Create numbers large enough to definitely use FFT
	// 10000 digits each
	aBytes := make([]byte, 5000)
	bBytes := make([]byte, 5000)

	// Fill with random data
	if _, err := rand.Read(aBytes); err != nil {
		t.Fatalf("Failed to generate random data: %v", err)
	}
	if _, err := rand.Read(bBytes); err != nil {
		t.Fatalf("Failed to generate random data: %v", err)
	}

	a := new(big.Int).SetBytes(aBytes)
	b := new(big.Int).SetBytes(bBytes)

	// Calculate using standard multiplication
	expected := new(big.Int).Mul(a, b)

	// Calculate using FFT multiplication
	result := Mul(a, b)

	if result.Cmp(expected) != 0 {
		t.Errorf("Very large multiplication mismatch. Bit lengths: a=%d, b=%d",
			a.BitLen(), b.BitLen())
	}
}

// TestMulPrecisionNegative verifies handling of negative numbers.
func TestMulPrecisionNegative(t *testing.T) {
	testCases := []struct {
		a, b, expected string
	}{
		{"-1", "1", "-1"},
		{"1", "-1", "-1"},
		{"-1", "-1", "1"},
		{"-12345", "67890", "-838102050"},
		{"12345", "-67890", "-838102050"},
		{"-12345", "-67890", "838102050"},
	}

	for _, tc := range testCases {
		a := new(big.Int)
		a.SetString(tc.a, 10)
		b := new(big.Int)
		b.SetString(tc.b, 10)
		expected := new(big.Int)
		expected.SetString(tc.expected, 10)

		result := Mul(a, b)
		if result.Cmp(expected) != 0 {
			t.Errorf("%s * %s: expected %s, got %s", tc.a, tc.b, expected.String(), result.String())
		}
	}
}

// TestMulToPrecision verifies the MulTo function.
func TestMulToPrecision(t *testing.T) {
	testCases := []struct {
		a, b string
	}{
		{"123", "456"},
		{"999999999", "999999999"},
		{"12345678901234567890", "98765432109876543210"},
	}

	for _, tc := range testCases {
		a := new(big.Int)
		a.SetString(tc.a, 10)
		b := new(big.Int)
		b.SetString(tc.b, 10)

		expected := new(big.Int).Mul(a, b)

		z := new(big.Int)
		result := MulTo(z, a, b)

		if result.Cmp(expected) != 0 {
			t.Errorf("MulTo(%s, %s): expected %s, got %s",
				tc.a, tc.b, expected.String(), result.String())
		}

		// Verify z was used (same pointer)
		if result != z {
			t.Error("MulTo did not return the destination pointer")
		}
	}
}

// TestMulToReuseBuffer tests that MulTo correctly reuses the buffer.
func TestMulToReuseBuffer(t *testing.T) {
	a := big.NewInt(123456)
	b := big.NewInt(789012)

	// Pre-allocate z with some capacity
	z := new(big.Int)
	z.SetInt64(999999999999)

	expected := new(big.Int).Mul(a, b)
	result := MulTo(z, a, b)

	if result.Cmp(expected) != 0 {
		t.Errorf("MulTo reuse failed: expected %s, got %s",
			expected.String(), result.String())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Commutativity and Associativity Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestMulCommutativity verifies a * b = b * a.
func TestMulCommutativity(t *testing.T) {
	testCases := []struct {
		a, b string
	}{
		{"12345", "67890"},
		{"999999999999999999", "888888888888888888"},
		{"123456789012345678901234567890", "987654321098765432109876543210"},
	}

	for _, tc := range testCases {
		a := new(big.Int)
		a.SetString(tc.a, 10)
		b := new(big.Int)
		b.SetString(tc.b, 10)

		ab := Mul(a, b)
		ba := Mul(b, a)

		if ab.Cmp(ba) != 0 {
			t.Errorf("Commutativity violated: %s * %s != %s * %s",
				tc.a, tc.b, tc.b, tc.a)
		}
	}
}

// TestMulAssociativity verifies (a * b) * c = a * (b * c).
func TestMulAssociativity(t *testing.T) {
	a := new(big.Int)
	a.SetString("12345678901234567890", 10)
	b := new(big.Int)
	b.SetString("98765432109876543210", 10)
	c := new(big.Int)
	c.SetString("11111111111111111111", 10)

	// (a * b) * c
	ab := Mul(a, b)
	abc1 := Mul(ab, c)

	// a * (b * c)
	bc := Mul(b, c)
	abc2 := Mul(a, bc)

	if abc1.Cmp(abc2) != 0 {
		t.Errorf("Associativity violated:\n  (a*b)*c = %s\n  a*(b*c) = %s",
			abc1.String(), abc2.String())
	}
}

// TestMulDistributivity verifies a * (b + c) = a*b + a*c.
func TestMulDistributivity(t *testing.T) {
	a := new(big.Int)
	a.SetString("12345678901234567890", 10)
	b := new(big.Int)
	b.SetString("98765432109876543210", 10)
	c := new(big.Int)
	c.SetString("11111111111111111111", 10)

	// a * (b + c)
	bPlusC := new(big.Int).Add(b, c)
	left := Mul(a, bPlusC)

	// a*b + a*c
	ab := Mul(a, b)
	ac := Mul(a, c)
	right := new(big.Int).Add(ab, ac)

	if left.Cmp(right) != 0 {
		t.Errorf("Distributivity violated:\n  a*(b+c) = %s\n  a*b+a*c = %s",
			left.String(), right.String())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Edge Case Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestMulByZero verifies multiplication by zero.
func TestMulByZero(t *testing.T) {
	zero := big.NewInt(0)

	testCases := []string{
		"1",
		"12345",
		"999999999999999999999999999999",
	}

	for _, tc := range testCases {
		a := new(big.Int)
		a.SetString(tc, 10)

		result := Mul(a, zero)
		if result.Cmp(zero) != 0 {
			t.Errorf("%s * 0 = %s, expected 0", tc, result.String())
		}

		result = Mul(zero, a)
		if result.Cmp(zero) != 0 {
			t.Errorf("0 * %s = %s, expected 0", tc, result.String())
		}
	}
}

// TestMulByOne verifies multiplication by one.
func TestMulByOne(t *testing.T) {
	one := big.NewInt(1)

	testCases := []string{
		"0",
		"1",
		"12345",
		"999999999999999999999999999999",
	}

	for _, tc := range testCases {
		a := new(big.Int)
		a.SetString(tc, 10)

		result := Mul(a, one)
		if result.Cmp(a) != 0 {
			t.Errorf("%s * 1 = %s, expected %s", tc, result.String(), a.String())
		}

		result = Mul(one, a)
		if result.Cmp(a) != 0 {
			t.Errorf("1 * %s = %s, expected %s", tc, result.String(), a.String())
		}
	}
}

// TestMulPowerOfTwo verifies multiplication by powers of two.
func TestMulPowerOfTwo(t *testing.T) {
	a := new(big.Int)
	a.SetString("12345678901234567890", 10)

	for i := 0; i < 20; i++ {
		powerOfTwo := new(big.Int).Lsh(big.NewInt(1), uint(i))

		result := Mul(a, powerOfTwo)
		expected := new(big.Int).Lsh(a, uint(i))

		if result.Cmp(expected) != 0 {
			t.Errorf("Multiplication by 2^%d failed:\n  Expected: %s\n  Got:      %s",
				i, expected.String(), result.String())
		}
	}
}

// TestMulSquaring verifies that a * a produces correct squares.
func TestMulSquaring(t *testing.T) {
	testCases := []string{
		"2",
		"10",
		"123",
		"12345",
		"123456789",
		"12345678901234567890",
	}

	for _, tc := range testCases {
		a := new(big.Int)
		a.SetString(tc, 10)

		result := Mul(a, a)
		expected := new(big.Int).Mul(a, a)

		if result.Cmp(expected) != 0 {
			t.Errorf("Square of %s failed:\n  Expected: %s\n  Got:      %s",
				tc, expected.String(), result.String())
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Benchmark Tests
// ─────────────────────────────────────────────────────────────────────────────

// BenchmarkMulSmall benchmarks multiplication of small numbers.
func BenchmarkMulSmall(b *testing.B) {
	a := new(big.Int)
	a.SetString("12345678901234567890", 10)
	bInt := new(big.Int)
	bInt.SetString("98765432109876543210", 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Mul(a, bInt)
	}
}

// BenchmarkMulMedium benchmarks multiplication of medium numbers.
func BenchmarkMulMedium(b *testing.B) {
	aBytes := make([]byte, 1000)
	bBytes := make([]byte, 1000)
	rand.Read(aBytes)
	rand.Read(bBytes)

	a := new(big.Int).SetBytes(aBytes)
	bInt := new(big.Int).SetBytes(bBytes)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Mul(a, bInt)
	}
}

// BenchmarkMulLarge benchmarks multiplication of large numbers.
func BenchmarkMulLarge(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping large benchmark in short mode")
	}

	aBytes := make([]byte, 10000)
	bBytes := make([]byte, 10000)
	rand.Read(aBytes)
	rand.Read(bBytes)

	a := new(big.Int).SetBytes(aBytes)
	bInt := new(big.Int).SetBytes(bBytes)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Mul(a, bInt)
	}
}

// BenchmarkMulToReuse benchmarks MulTo with buffer reuse.
func BenchmarkMulToReuse(b *testing.B) {
	a := new(big.Int)
	a.SetString("12345678901234567890123456789012345678901234567890", 10)
	bInt := new(big.Int)
	bInt.SetString("98765432109876543210987654321098765432109876543210", 10)

	z := new(big.Int)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MulTo(z, a, bInt)
	}
}
