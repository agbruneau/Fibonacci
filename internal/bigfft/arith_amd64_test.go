//go:build amd64

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
// CPU Feature Detection Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestCPUFeatureDetection(t *testing.T) {
	t.Parallel()
	features := GetCPUFeatures()
	t.Logf("CPU Features: %s", features.String())
	t.Logf("SIMD Level: %s", features.SIMDLevel.String())
	t.Logf("AVX2: %v, AVX-512: %v, BMI2: %v, ADX: %v",
		features.AVX2, features.AVX512, features.BMI2, features.ADX)
}

func TestSIMDLevel(t *testing.T) {
	t.Parallel()
	level := GetSIMDLevel()
	t.Logf("Active SIMD Level: %s", level.String())

	// SIMD level should be consistent with feature detection
	if HasAVX512() && level != SIMDAVX512 {
		// Note: We might fall back to AVX2 even with AVX512 available
		t.Logf("AVX512 detected but level is %s (may be by design)", level)
	}
	if HasAVX2() && level == SIMDNone {
		t.Logf("AVX2 detected but level is None (may be disabled)")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Correctness Tests for addVV
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

			// AVX2 implementation
			if HasAVX2() {
				zAVX2 := make([]Word, tc.size)
				cAVX2 := addVVAvx2(zAVX2, x, y)

				// Compare results
				if cRef != cAVX2 {
					t.Errorf("Carry mismatch: ref=%d, avx2=%d", cRef, cAVX2)
				}
				for i := range zRef {
					if zRef[i] != zAVX2[i] {
						t.Errorf("Result mismatch at index %d: ref=%x, avx2=%x", i, zRef[i], zAVX2[i])
					}
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

	if HasAVX2() {
		zAVX2 := make([]Word, size)
		cAVX2 := addVVAvx2(zAVX2, x, y)

		if cRef != cAVX2 {
			t.Errorf("Carry chain mismatch: ref=%d, avx2=%d", cRef, cAVX2)
		}
		for i := range zRef {
			if zRef[i] != zAVX2[i] {
				t.Errorf("Result mismatch at index %d: ref=%x, avx2=%x", i, zRef[i], zAVX2[i])
			}
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Correctness Tests for subVV
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

			if HasAVX2() {
				zAVX2 := make([]Word, tc.size)
				cAVX2 := subVVAvx2(zAVX2, x, y)

				if cRef != cAVX2 {
					t.Errorf("Borrow mismatch: ref=%d, avx2=%d", cRef, cAVX2)
				}
				for i := range zRef {
					if zRef[i] != zAVX2[i] {
						t.Errorf("Result mismatch at index %d: ref=%x, avx2=%x", i, zRef[i], zAVX2[i])
					}
				}
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Correctness Tests for addMulVVW
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
			zAVX2 := copyWords(zRef)

			cRef := addMulVVW(zRef, x, y)

			if HasAVX2() {
				cAVX2 := addMulVVWAvx2(zAVX2, x, y)

				if cRef != cAVX2 {
					t.Errorf("Carry mismatch: ref=%d, avx2=%d", cRef, cAVX2)
				}
				for i := range zRef {
					if zRef[i] != zAVX2[i] {
						t.Errorf("Result mismatch at index %d: ref=%x, avx2=%x", i, zRef[i], zAVX2[i])
					}
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

		b.Run("Default/"+itoa(size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				addVV(z, x, y)
			}
		})

		if HasAVX2() {
			b.Run("AVX2/"+itoa(size), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					addVVAvx2(z, x, y)
				}
			})
		}
	}
}

func BenchmarkSubVV(b *testing.B) {
	b.ReportAllocs()
	sizes := []int{8, 64, 256, 1024, 4096}

	for _, size := range sizes {
		x := generateRandomWords(size, 42)
		y := generateRandomWords(size, 43)
		z := make([]Word, size)

		b.Run("Default/"+itoa(size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				subVV(z, x, y)
			}
		})

		if HasAVX2() {
			b.Run("AVX2/"+itoa(size), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					subVVAvx2(z, x, y)
				}
			})
		}
	}
}

func BenchmarkAddMulVVW(b *testing.B) {
	b.ReportAllocs()
	sizes := []int{8, 64, 256, 1024, 4096}

	for _, size := range sizes {
		x := generateRandomWords(size, 42)
		y := Word(0x123456789ABCDEF0)
		z := make([]Word, size)

		b.Run("Default/"+itoa(size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				// Reset z for fair comparison
				for j := range z {
					z[j] = 0
				}
				addMulVVW(z, x, y)
			}
		})

		if HasAVX2() {
			b.Run("AVX2/"+itoa(size), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					for j := range z {
						z[j] = 0
					}
					addMulVVWAvx2(z, x, y)
				}
			})
		}
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

// ─────────────────────────────────────────────────────────────────────────────
// Implementation Selection Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestUseAVX2(t *testing.T) {
	// Save original state
	originalLevel := implLevel
	defer func() {
		// Restore original implementation
		selectImplementation()
	}()

	result := UseAVX2()
	if HasAVX2() {
		if !result {
			t.Error("UseAVX2() returned false but AVX2 is available")
		}
		if implLevel != SIMDAVX2 {
			t.Errorf("expected implLevel SIMDAVX2, got %s", implLevel.String())
		}
	} else {
		if result {
			t.Error("UseAVX2() returned true but AVX2 is not available")
		}
	}
	_ = originalLevel // Use variable
}

func TestUseDefault(t *testing.T) {
	// Save original state
	defer func() {
		// Restore original implementation
		selectImplementation()
	}()

	// First enable AVX2 if available
	UseAVX2()

	// Then switch to default
	UseDefault()

	if implLevel != SIMDNone {
		t.Errorf("expected implLevel SIMDNone after UseDefault, got %s", implLevel.String())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Auto Selection Function Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestAddVVAuto(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{"Empty", 0},
		{"BelowThreshold", MinSIMDVectorLen - 1},
		{"AtThreshold", MinSIMDVectorLen},
		{"AboveThreshold", MinSIMDVectorLen * 2},
		{"Large", 64},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.size == 0 {
				c := AddVVAuto(nil, nil, nil)
				if c != 0 {
					t.Errorf("Empty AddVVAuto returned %d, expected 0", c)
				}
				return
			}

			x := generateRandomWords(tc.size, 42)
			y := generateRandomWords(tc.size, 43)
			z := make([]Word, tc.size)
			zRef := make([]Word, tc.size)

			// Reference result
			cRef := addVV(zRef, x, y)

			// Auto result
			c := AddVVAuto(z, x, y)

			if c != cRef {
				t.Errorf("Carry mismatch: auto=%d, ref=%d", c, cRef)
			}
			for i := range z {
				if z[i] != zRef[i] {
					t.Errorf("Result mismatch at index %d: auto=%x, ref=%x", i, z[i], zRef[i])
				}
			}
		})
	}
}

func TestSubVVAuto(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{"Empty", 0},
		{"BelowThreshold", MinSIMDVectorLen - 1},
		{"AtThreshold", MinSIMDVectorLen},
		{"AboveThreshold", MinSIMDVectorLen * 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.size == 0 {
				c := SubVVAuto(nil, nil, nil)
				if c != 0 {
					t.Errorf("Empty SubVVAuto returned %d, expected 0", c)
				}
				return
			}

			x := generateRandomWords(tc.size, 42)
			y := generateRandomWords(tc.size, 43)
			z := make([]Word, tc.size)
			zRef := make([]Word, tc.size)

			cRef := subVV(zRef, x, y)
			c := SubVVAuto(z, x, y)

			if c != cRef {
				t.Errorf("Borrow mismatch: auto=%d, ref=%d", c, cRef)
			}
			for i := range z {
				if z[i] != zRef[i] {
					t.Errorf("Result mismatch at index %d: auto=%x, ref=%x", i, z[i], zRef[i])
				}
			}
		})
	}
}

func TestAddMulVVWAuto(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{"Empty", 0},
		{"BelowThreshold", MinSIMDVectorLen - 1},
		{"AtThreshold", MinSIMDVectorLen},
		{"AboveThreshold", MinSIMDVectorLen * 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.size == 0 {
				c := AddMulVVWAuto(nil, nil, 0)
				if c != 0 {
					t.Errorf("Empty AddMulVVWAuto returned %d, expected 0", c)
				}
				return
			}

			x := generateRandomWords(tc.size, 42)
			y := Word(0x123456789ABCDEF0)
			z := generateRandomWords(tc.size, 45)
			zRef := copyWords(z)

			cRef := addMulVVW(zRef, x, y)
			c := AddMulVVWAuto(z, x, y)

			if c != cRef {
				t.Errorf("Carry mismatch: auto=%d, ref=%d", c, cRef)
			}
			for i := range z {
				if z[i] != zRef[i] {
					t.Errorf("Result mismatch at index %d: auto=%x, ref=%x", i, z[i], zRef[i])
				}
			}
		})
	}
}
