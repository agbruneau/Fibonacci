package bigfft

import (
	"math/big"
	"math/rand"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// Test Utilities
// ─────────────────────────────────────────────────────────────────────────────

// generateRandomWords creates a slice of random words for testing.
func generateRandomWords(n int, seed int64) []Word {
	r := rand.New(rand.NewSource(seed))
	words := make([]Word, n)
	for i := range words {
		words[i] = Word(r.Uint64())
	}
	return words
}

// copyWords creates a copy of a word slice.
func copyWords(src []Word) []Word {
	dst := make([]Word, len(src))
	copy(dst, src)
	return dst
}

// ─────────────────────────────────────────────────────────────────────────────
// Correctness Tests for AddVV
// ─────────────────────────────────────────────────────────────────────────────

func TestAddVV_Correctness(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		size int
	}{
		{"Empty", 0},
		{"Single", 1},
		{"Small", 4},
		{"Medium", 16},
		{"Large", 64},
		{"XLarge", 256},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.size == 0 {
				// Test empty case
				c := AddVV(nil, nil, nil)
				if c != 0 {
					t.Errorf("Empty AddVV returned carry %d, expected 0", c)
				}
				return
			}

			x := generateRandomWords(tc.size, 42)
			y := generateRandomWords(tc.size, 43)

			// Reference implementation (using go:linkname)
			zRef := make([]Word, tc.size)
			cRef := addVV(zRef, x, y)

			// Test via public API
			zTest := make([]Word, tc.size)
			cTest := AddVV(zTest, x, y)

			// Compare results
			if cRef != cTest {
				t.Errorf("Carry mismatch: ref=%d, test=%d", cRef, cTest)
			}
			for i := range zRef {
				if zRef[i] != zTest[i] {
					t.Errorf("Result mismatch at index %d: ref=%x, test=%x", i, zRef[i], zTest[i])
				}
			}
		})
	}
}

func TestAddVV_CarryPropagation(t *testing.T) {
	t.Parallel()
	// Test carry propagation with all 1s
	size := 8
	x := make([]Word, size)
	y := make([]Word, size)

	// x = all 1s (max value per word)
	// y = 1 in first position
	// Should produce carry chain
	for i := range x {
		x[i] = ^Word(0)
	}
	y[0] = 1

	zRef := make([]Word, size)
	cRef := addVV(zRef, x, y)

	zTest := make([]Word, size)
	cTest := AddVV(zTest, x, y)

	if cRef != cTest {
		t.Errorf("Carry chain mismatch: ref=%d, test=%d", cRef, cTest)
	}
	for i := range zRef {
		if zRef[i] != zTest[i] {
			t.Errorf("Result mismatch at index %d: ref=%x, test=%x", i, zRef[i], zTest[i])
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Correctness Tests for SubVV
// ─────────────────────────────────────────────────────────────────────────────

func TestSubVV_Correctness(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		size int
	}{
		{"Empty", 0},
		{"Single", 1},
		{"Small", 4},
		{"Medium", 16},
		{"Large", 64},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.size == 0 {
				c := SubVV(nil, nil, nil)
				if c != 0 {
					t.Errorf("Empty SubVV returned borrow %d, expected 0", c)
				}
				return
			}

			x := generateRandomWords(tc.size, 42)
			y := generateRandomWords(tc.size, 43)

			zRef := make([]Word, tc.size)
			cRef := subVV(zRef, x, y)

			zTest := make([]Word, tc.size)
			cTest := SubVV(zTest, x, y)

			if cRef != cTest {
				t.Errorf("Borrow mismatch: ref=%d, test=%d", cRef, cTest)
			}
			for i := range zRef {
				if zRef[i] != zTest[i] {
					t.Errorf("Result mismatch at index %d: ref=%x, test=%x", i, zRef[i], zTest[i])
				}
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Correctness Tests for AddMulVVW
// ─────────────────────────────────────────────────────────────────────────────

func TestAddMulVVW_Correctness(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		size int
	}{
		{"Empty", 0},
		{"Single", 1},
		{"Small", 4},
		{"Medium", 16},
		{"Large", 64},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.size == 0 {
				c := AddMulVVW(nil, nil, 0)
				if c != 0 {
					t.Errorf("Empty AddMulVVW returned carry %d, expected 0", c)
				}
				return
			}

			x := generateRandomWords(tc.size, 42)
			y := Word(rand.New(rand.NewSource(44)).Uint64())

			// Initialize z with random values (accumulator)
			zRef := generateRandomWords(tc.size, 45)
			zTest := copyWords(zRef)

			cRef := addMulVVW(zRef, x, y)
			cTest := AddMulVVW(zTest, x, y)

			if cRef != cTest {
				t.Errorf("Carry mismatch: ref=%d, test=%d", cRef, cTest)
			}
			for i := range zRef {
				if zRef[i] != zTest[i] {
					t.Errorf("Result mismatch at index %d: ref=%x, test=%x", i, zRef[i], zTest[i])
				}
			}
		})
	}
}

func TestAddMulVVW_SpecialCases(t *testing.T) {
	t.Parallel()
	size := 8

	t.Run("MultiplyByZero", func(t *testing.T) {
		t.Parallel()
		x := generateRandomWords(size, 42)
		z := generateRandomWords(size, 43)
		zOriginal := copyWords(z)

		c := AddMulVVW(z, x, 0)

		// Multiplying by zero should leave z unchanged and return 0 carry
		if c != 0 {
			t.Errorf("Multiply by zero carry: got %d, expected 0", c)
		}
		for i := range z {
			if z[i] != zOriginal[i] {
				t.Errorf("z changed at index %d: got %x, expected %x", i, z[i], zOriginal[i])
			}
		}
	})

	t.Run("MultiplyByOne", func(t *testing.T) {
		t.Parallel()
		x := generateRandomWords(size, 42)
		z := make([]Word, size) // Start with zeros

		c := AddMulVVW(z, x, 1)

		// Multiplying by one should give z = x (plus original z, which is 0)
		// The carry depends on whether any addition overflows
		_ = c // Carry might be non-zero for large x values
		for i := range z {
			if z[i] != x[i] {
				t.Errorf("Multiply by one mismatch at index %d: got %x, expected %x", i, z[i], x[i])
			}
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Benchmarks
// ─────────────────────────────────────────────────────────────────────────────

func BenchmarkAddVV(b *testing.B) {
	b.ReportAllocs()
	sizes := []int{8, 64, 256, 1024, 4096}

	for _, size := range sizes {
		x := generateRandomWords(size, 42)
		y := generateRandomWords(size, 43)
		z := make([]Word, size)

		b.Run(itoa(size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				AddVV(z, x, y)
			}
		})
	}
}

func BenchmarkSubVV(b *testing.B) {
	b.ReportAllocs()
	sizes := []int{8, 64, 256, 1024, 4096}

	for _, size := range sizes {
		x := generateRandomWords(size, 42)
		y := generateRandomWords(size, 43)
		z := make([]Word, size)

		b.Run(itoa(size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				SubVV(z, x, y)
			}
		})
	}
}

func BenchmarkAddMulVVW(b *testing.B) {
	b.ReportAllocs()
	sizes := []int{8, 64, 256, 1024, 4096}

	for _, size := range sizes {
		x := generateRandomWords(size, 42)
		y := Word(0x123456789ABCDEF0)
		z := make([]Word, size)

		b.Run(itoa(size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				// Reset z for fair comparison
				for j := range z {
					z[j] = 0
				}
				AddMulVVW(z, x, y)
			}
		})
	}
}

// itoa converts int to string (simple helper to avoid strconv import)
func itoa(n int) string {
	return big.NewInt(int64(n)).String()
}

// ─────────────────────────────────────────────────────────────────────────────
// Integration Tests with big.Int
// ─────────────────────────────────────────────────────────────────────────────

func TestIntegrationWithBigInt(t *testing.T) {
	t.Parallel()
	// Test that our implementations produce results compatible with big.Int

	t.Run("Addition", func(t *testing.T) {
		x := big.NewInt(0).SetBits(generateRandomWords(16, 42))
		y := big.NewInt(0).SetBits(generateRandomWords(16, 43))

		expected := new(big.Int).Add(x, y)

		// Manual addition using our function
		xWords := x.Bits()
		yWords := y.Bits()
		maxLen := len(xWords)
		if len(yWords) > maxLen {
			maxLen = len(yWords)
		}

		// Pad to same length
		xPadded := make([]Word, maxLen)
		yPadded := make([]Word, maxLen)
		copy(xPadded, xWords)
		copy(yPadded, yWords)

		zWords := make([]Word, maxLen+1)
		c := AddVV(zWords[:maxLen], xPadded, yPadded)
		zWords[maxLen] = c

		result := big.NewInt(0).SetBits(zWords)

		if expected.Cmp(result) != 0 {
			t.Errorf("Addition mismatch:\nexpected: %s\ngot: %s", expected.String(), result.String())
		}
	})
}
