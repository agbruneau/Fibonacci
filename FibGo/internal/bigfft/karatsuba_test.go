package bigfft

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// Precision/Correctness Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestKaratsubaSmall verifies Karatsuba multiplication for small numbers.
func TestKaratsubaSmall(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		x, y, want int64
	}{
		{0, 0, 0},
		{1, 1, 1},
		{2, 3, 6},
		{7, 8, 56},
		{100, 200, 20000},
		{12345, 67890, 838102050},
		{-5, 3, -15},
		{-7, -8, 56},
	}

	for _, tc := range testCases {
		x := big.NewInt(tc.x)
		y := big.NewInt(tc.y)
		got := KaratsubaMultiply(x, y)
		want := big.NewInt(tc.want)

		if got.Cmp(want) != 0 {
			t.Errorf("KaratsubaMultiply(%d, %d) = %s, want %s", tc.x, tc.y, got, want)
		}
	}
}

// TestKaratsubaMedium verifies Karatsuba for medium-sized numbers (~1000 bits).
func TestKaratsubaMedium(t *testing.T) {
	t.Parallel()
	// Generate random numbers with ~1000 bits
	x, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 1000))
	y, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 1000))

	got := KaratsubaMultiply(x, y)
	want := new(big.Int).Mul(x, y)

	if got.Cmp(want) != 0 {
		t.Errorf("KaratsubaMedium: result mismatch for %d-bit × %d-bit numbers",
			x.BitLen(), y.BitLen())
	}
}

// TestKaratsubaLarge verifies Karatsuba for large numbers (~100k bits).
func TestKaratsubaLarge(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping large number test in short mode")
	}

	x, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 100000))
	y, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 100000))

	got := KaratsubaMultiply(x, y)
	want := new(big.Int).Mul(x, y)

	if got.Cmp(want) != 0 {
		t.Errorf("KaratsubaLarge: result mismatch for %d-bit × %d-bit numbers",
			x.BitLen(), y.BitLen())
	}
}

// TestKaratsubaVeryLarge tests with very large numbers (~1M bits).
func TestKaratsubaVeryLarge(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping very large number test in short mode")
	}

	x, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 500000))
	y, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 500000))

	got := KaratsubaMultiply(x, y)
	want := new(big.Int).Mul(x, y)

	if got.Cmp(want) != 0 {
		t.Errorf("KaratsubaVeryLarge: result mismatch for %d-bit × %d-bit numbers",
			x.BitLen(), y.BitLen())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Edge Case Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestKaratsubaZero verifies multiplication by zero.
func TestKaratsubaZero(t *testing.T) {
	t.Parallel()
	zero := big.NewInt(0)
	large := new(big.Int).Lsh(big.NewInt(1), 10000)

	testCases := []struct {
		name string
		x, y *big.Int
	}{
		{"zero*zero", zero, zero},
		{"zero*large", zero, large},
		{"large*zero", large, zero},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := KaratsubaMultiply(tc.x, tc.y)
			if got.Cmp(zero) != 0 {
				t.Errorf("Expected 0, got %s", got)
			}
		})
	}
}

// TestKaratsubaOne verifies multiplication by one.
func TestKaratsubaOne(t *testing.T) {
	t.Parallel()
	one := big.NewInt(1)
	x, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 5000))

	got := KaratsubaMultiply(x, one)
	if got.Cmp(x) != 0 {
		t.Errorf("x*1 != x")
	}

	got = KaratsubaMultiply(one, x)
	if got.Cmp(x) != 0 {
		t.Errorf("1*x != x")
	}
}

// TestKaratsubaPowerOfTwo verifies multiplication of powers of two.
func TestKaratsubaPowerOfTwo(t *testing.T) {
	t.Parallel()
	for i := 0; i < 20; i++ {
		x := new(big.Int).Lsh(big.NewInt(1), uint(i*100))
		y := new(big.Int).Lsh(big.NewInt(1), uint(i*50))

		got := KaratsubaMultiply(x, y)
		want := new(big.Int).Mul(x, y)

		if got.Cmp(want) != 0 {
			t.Errorf("Power of two test failed at i=%d", i)
		}
	}
}

// TestKaratsubaNegative verifies sign handling.
func TestKaratsubaNegative(t *testing.T) {
	t.Parallel()
	x, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 2000))
	y, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 2000))

	// (+) * (+) = (+)
	got := KaratsubaMultiply(x, y)
	want := new(big.Int).Mul(x, y)
	if got.Cmp(want) != 0 {
		t.Errorf("(+)*(+) mismatch")
	}

	// (-) * (+) = (-)
	negX := new(big.Int).Neg(x)
	got = KaratsubaMultiply(negX, y)
	want = new(big.Int).Mul(negX, y)
	if got.Cmp(want) != 0 {
		t.Errorf("(-)*(+) mismatch")
	}

	// (+) * (-) = (-)
	negY := new(big.Int).Neg(y)
	got = KaratsubaMultiply(x, negY)
	want = new(big.Int).Mul(x, negY)
	if got.Cmp(want) != 0 {
		t.Errorf("(+)*(-) mismatch")
	}

	// (-) * (-) = (+)
	got = KaratsubaMultiply(negX, negY)
	want = new(big.Int).Mul(negX, negY)
	if got.Cmp(want) != 0 {
		t.Errorf("(-)*(-) mismatch")
	}
}

// TestKaratsubaAsymmetric tests multiplication of differently sized operands.
func TestKaratsubaAsymmetric(t *testing.T) {
	t.Parallel()
	small := big.NewInt(12345)
	medium, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 1000))
	large, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 10000))

	testCases := []struct {
		name string
		x, y *big.Int
	}{
		{"small*medium", small, medium},
		{"small*large", small, large},
		{"medium*large", medium, large},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := KaratsubaMultiply(tc.x, tc.y)
			want := new(big.Int).Mul(tc.x, tc.y)
			if got.Cmp(want) != 0 {
				t.Errorf("Result mismatch")
			}

			// Test commutativity
			got2 := KaratsubaMultiply(tc.y, tc.x)
			if got2.Cmp(want) != 0 {
				t.Errorf("Commutativity failed")
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Consistency Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestKaratsubaVsBigIntMul compares results with standard library.
func TestKaratsubaVsBigIntMul(t *testing.T) {
	t.Parallel()
	sizes := []int{100, 500, 1000, 2000, 5000, 10000}

	for _, bits := range sizes {
		t.Run(fmt.Sprintf("%dbits", bits), func(t *testing.T) {
			t.Parallel()
			x, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), uint(bits)))
			y, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), uint(bits)))

			got := KaratsubaMultiply(x, y)
			want := new(big.Int).Mul(x, y)

			if got.Cmp(want) != 0 {
				t.Errorf("Mismatch at %d bits", bits)
			}
		})
	}
}

// TestKaratsubaSqrVsMul verifies squaring consistency.
func TestKaratsubaSqrVsMul(t *testing.T) {
	t.Parallel()
	sizes := []int{100, 500, 1000, 5000, 10000}

	for _, bits := range sizes {
		t.Run(fmt.Sprintf("%dbits", bits), func(t *testing.T) {
			t.Parallel()
			x, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), uint(bits)))

			sqr := KaratsubaSqr(x)
			mul := KaratsubaMultiply(x, x)
			std := new(big.Int).Mul(x, x)

			if sqr.Cmp(std) != 0 {
				t.Errorf("KaratsubaSqr mismatch at %d bits", bits)
			}
			if mul.Cmp(std) != 0 {
				t.Errorf("KaratsubaMultiply(x,x) mismatch at %d bits", bits)
			}
		})
	}
}

// TestKaratsubaMultiplyTo verifies the buffer reuse variant.
func TestKaratsubaMultiplyTo(t *testing.T) {
	t.Parallel()
	x, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 5000))
	y, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 5000))
	want := new(big.Int).Mul(x, y)

	// Test with pre-allocated buffer
	z := new(big.Int)
	got := KaratsubaMultiplyTo(z, x, y)

	if got != z {
		t.Errorf("KaratsubaMultiplyTo should return the same pointer")
	}
	if got.Cmp(want) != 0 {
		t.Errorf("KaratsubaMultiplyTo result mismatch")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Threshold Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestKaratsubaThreshold tests threshold configuration.
func TestKaratsubaThreshold(t *testing.T) {
	t.Parallel()
	original := GetKaratsubaThreshold()
	defer SetKaratsubaThreshold(original)

	// Test with different thresholds
	thresholds := []int{8, 16, 32, 64, 128}

	x, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 5000))
	y, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 5000))
	want := new(big.Int).Mul(x, y)

	for _, threshold := range thresholds {
		SetKaratsubaThreshold(threshold)
		got := KaratsubaMultiply(x, y)
		if got.Cmp(want) != 0 {
			t.Errorf("Mismatch at threshold=%d", threshold)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Benchmarks
// ─────────────────────────────────────────────────────────────────────────────

// BenchmarkKaratsubaSmall benchmarks small number multiplication.
func BenchmarkKaratsubaSmall(b *testing.B) {
	x, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 512))
	y, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 512))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		KaratsubaMultiply(x, y)
	}
}

// BenchmarkKaratsubaMedium benchmarks medium number multiplication.
func BenchmarkKaratsubaMedium(b *testing.B) {
	x, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 10000))
	y, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 10000))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		KaratsubaMultiply(x, y)
	}
}

// BenchmarkKaratsubaLarge benchmarks large number multiplication.
func BenchmarkKaratsubaLarge(b *testing.B) {
	x, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 100000))
	y, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 100000))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		KaratsubaMultiply(x, y)
	}
}

// BenchmarkKaratsubaVsStdlib compares Karatsuba with big.Int.Mul at various sizes.
func BenchmarkKaratsubaVsStdlib(b *testing.B) {
	sizes := []int{512, 2048, 10000, 50000, 100000}

	for _, bits := range sizes {
		x, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), uint(bits)))
		y, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), uint(bits)))

		b.Run(fmt.Sprintf("Karatsuba_%dbits", bits), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				KaratsubaMultiply(x, y)
			}
		})

		b.Run(fmt.Sprintf("BigIntMul_%dbits", bits), func(b *testing.B) {
			z := new(big.Int)
			for i := 0; i < b.N; i++ {
				z.Mul(x, y)
			}
		})
	}
}

// BenchmarkKaratsubaSqr benchmarks squaring.
func BenchmarkKaratsubaSqr(b *testing.B) {
	sizes := []int{1000, 10000, 50000}

	for _, bits := range sizes {
		x, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), uint(bits)))

		b.Run(fmt.Sprintf("KaratsubaSqr_%dbits", bits), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				KaratsubaSqr(x)
			}
		})

		b.Run(fmt.Sprintf("BigIntMul_%dbits", bits), func(b *testing.B) {
			z := new(big.Int)
			for i := 0; i < b.N; i++ {
				z.Mul(x, x)
			}
		})
	}
}

// BenchmarkKaratsubaMemory benchmarks memory allocations.
func BenchmarkKaratsubaMemory(b *testing.B) {
	x, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 50000))
	y, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 50000))

	b.Run("Karatsuba", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			KaratsubaMultiply(x, y)
		}
	})

	b.Run("KaratsubaTo", func(b *testing.B) {
		z := new(big.Int)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			KaratsubaMultiplyTo(z, x, y)
		}
	})

	b.Run("BigIntMul", func(b *testing.B) {
		z := new(big.Int)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			z.Mul(x, y)
		}
	})
}
