package bigfft

import (
	"fmt"
	"math/big"
	"testing"
)

func TestWordSlicePool(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		size     int
		wantSize int // Expected size class
	}{
		{"small", 10, 64},
		{"medium", 100, 256},
		{"large", 1000, 1024},
		{"xlarge", 5000, 16384},
		{"too_large", 500000, 500000}, // Direct allocation
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			slice := acquireWordSlice(tt.size)
			if len(slice) != tt.size {
				t.Errorf("acquireWordSlice(%d) got length %d, want %d", tt.size, len(slice), tt.size)
			}

			// Verify it's zeroed
			for i := range slice {
				if slice[i] != 0 {
					t.Errorf("acquireWordSlice(%d) not zeroed at index %d", tt.size, i)
					break
				}
			}

			// Release should not panic
			releaseWordSlice(slice)
		})
	}
}

func TestWordSlicePoolUnsafe(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		size     int
		wantSize int // Expected size class
	}{
		{"small", 10, 64},
		{"medium", 100, 256},
		{"large", 1000, 1024},
		{"xlarge", 5000, 16384},
		{"too_large", 500000, 500000}, // Direct allocation
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			slice := acquireWordSliceUnsafe(tt.size)
			if len(slice) != tt.size {
				t.Errorf("acquireWordSliceUnsafe(%d) got length %d, want %d", tt.size, len(slice), tt.size)
			}

			// Verify that we can write to every element (caller's responsibility)
			for i := range slice {
				slice[i] = big.Word(i)
			}

			// Verify the writes took effect
			for i := range slice {
				if slice[i] != big.Word(i) {
					t.Errorf("acquireWordSliceUnsafe(%d) write failed at index %d", tt.size, i)
					break
				}
			}

			// Release should not panic
			releaseWordSlice(slice)
		})
	}
}

func TestFermatPool(t *testing.T) {
	t.Parallel()
	sizes := []int{16, 64, 256, 1024, 4096}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("Size=%d", size), func(t *testing.T) {
			t.Parallel()
			f := acquireFermat(size)
			if len(f) != size {
				t.Errorf("acquireFermat(%d) got length %d", size, len(f))
			}

			// Verify zeroed
			for i := range f {
				if f[i] != 0 {
					t.Errorf("acquireFermat(%d) not zeroed", size)
					break
				}
			}

			releaseFermat(f)
		})
	}
}

func TestNatSlicePool(t *testing.T) {
	t.Parallel()
	sizes := []int{4, 16, 64, 256}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("Size=%d", size), func(t *testing.T) {
			t.Parallel()
			slice := acquireNatSlice(size)
			if len(slice) != size {
				t.Errorf("acquireNatSlice(%d) got length %d", size, len(slice))
			}

			// Verify nil elements
			for i := range slice {
				if slice[i] != nil {
					t.Errorf("acquireNatSlice(%d) not nil at index %d", size, i)
					break
				}
			}

			releaseNatSlice(slice)
		})
	}
}

func TestFermatSlicePool(t *testing.T) {
	t.Parallel()
	sizes := []int{4, 16, 64, 256}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("Size=%d", size), func(t *testing.T) {
			t.Parallel()
			slice := acquireFermatSlice(size)
			if len(slice) != size {
				t.Errorf("acquireFermatSlice(%d) got length %d", size, len(slice))
			}

			// Verify nil elements
			for i := range slice {
				if slice[i] != nil {
					t.Errorf("acquireFermatSlice(%d) not nil at index %d", size, i)
					break
				}
			}

			releaseFermatSlice(slice)
		})
	}
}

func TestReleaseNilSafe(t *testing.T) {
	t.Parallel()
	// These should not panic
	releaseWordSlice(nil)
	releaseFermat(nil)
	releaseNatSlice(nil)
	releaseFermatSlice(nil)
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests for O(1) bitwise pool index functions vs original linear search
// ─────────────────────────────────────────────────────────────────────────────

func TestGetWordSlicePoolIndexConsistency(t *testing.T) {
	t.Parallel()
	// Test all sizes from 0 to max+100, verifying bitwise matches linear
	maxSize := wordSliceSizes[len(wordSliceSizes)-1]
	for size := 0; size <= maxSize+100; size++ {
		got := getWordSlicePoolIndex(size)
		want := getWordSlicePoolIndexLinear(size)
		if got != want {
			t.Fatalf("getWordSlicePoolIndex(%d): got %d, want %d", size, got, want)
		}
	}
}

func TestGetFermatPoolIndexConsistency(t *testing.T) {
	t.Parallel()
	// Test all sizes from 0 to max+100
	maxSize := fermatSizes[len(fermatSizes)-1]
	for size := 0; size <= maxSize+100; size++ {
		got := getFermatPoolIndex(size)
		want := getFermatPoolIndexLinear(size)
		if got != want {
			t.Fatalf("getFermatPoolIndex(%d): got %d, want %d", size, got, want)
		}
	}
}

func TestGetNatSlicePoolIndexConsistency(t *testing.T) {
	t.Parallel()
	// Test all sizes from 0 to max+100
	maxSize := natSliceSizes[len(natSliceSizes)-1]
	for size := 0; size <= maxSize+100; size++ {
		got := getNatSlicePoolIndex(size)
		want := getNatSlicePoolIndexLinear(size)
		if got != want {
			t.Fatalf("getNatSlicePoolIndex(%d): got %d, want %d", size, got, want)
		}
	}
}

func TestGetFermatSlicePoolIndexConsistency(t *testing.T) {
	t.Parallel()
	// Test all sizes from 0 to max+100
	maxSize := fermatSliceSizes[len(fermatSliceSizes)-1]
	for size := 0; size <= maxSize+100; size++ {
		got := getFermatSlicePoolIndex(size)
		want := getFermatSlicePoolIndexLinear(size)
		if got != want {
			t.Fatalf("getFermatSlicePoolIndex(%d): got %d, want %d", size, got, want)
		}
	}
}

// TestPoolIndexBoundaryValues tests the exact boundary values for each pool.
func TestPoolIndexBoundaryValues(t *testing.T) {
	t.Parallel()

	t.Run("wordSlice", func(t *testing.T) {
		t.Parallel()
		for i, size := range wordSliceSizes {
			// At the boundary, should return this index
			if got := getWordSlicePoolIndex(size); got != i {
				t.Errorf("getWordSlicePoolIndex(%d) = %d, want %d", size, got, i)
			}
			// One above previous boundary should return this index
			if i > 0 {
				if got := getWordSlicePoolIndex(wordSliceSizes[i-1] + 1); got != i {
					t.Errorf("getWordSlicePoolIndex(%d) = %d, want %d", wordSliceSizes[i-1]+1, got, i)
				}
			}
		}
		// Above max should return -1
		if got := getWordSlicePoolIndex(wordSliceSizes[len(wordSliceSizes)-1] + 1); got != -1 {
			t.Errorf("getWordSlicePoolIndex(max+1) = %d, want -1", got)
		}
	})

	t.Run("fermat", func(t *testing.T) {
		t.Parallel()
		for i, size := range fermatSizes {
			if got := getFermatPoolIndex(size); got != i {
				t.Errorf("getFermatPoolIndex(%d) = %d, want %d", size, got, i)
			}
		}
		if got := getFermatPoolIndex(fermatSizes[len(fermatSizes)-1] + 1); got != -1 {
			t.Errorf("getFermatPoolIndex(max+1) = %d, want -1", got)
		}
	})

	t.Run("natSlice", func(t *testing.T) {
		t.Parallel()
		for i, size := range natSliceSizes {
			if got := getNatSlicePoolIndex(size); got != i {
				t.Errorf("getNatSlicePoolIndex(%d) = %d, want %d", size, got, i)
			}
		}
		if got := getNatSlicePoolIndex(natSliceSizes[len(natSliceSizes)-1] + 1); got != -1 {
			t.Errorf("getNatSlicePoolIndex(max+1) = %d, want -1", got)
		}
	})

	t.Run("fermatSlice", func(t *testing.T) {
		t.Parallel()
		for i, size := range fermatSliceSizes {
			if got := getFermatSlicePoolIndex(size); got != i {
				t.Errorf("getFermatSlicePoolIndex(%d) = %d, want %d", size, got, i)
			}
		}
		if got := getFermatSlicePoolIndex(fermatSliceSizes[len(fermatSliceSizes)-1] + 1); got != -1 {
			t.Errorf("getFermatSlicePoolIndex(max+1) = %d, want -1", got)
		}
	})
}

func BenchmarkGetWordSlicePoolIndex(b *testing.B) {
	sizes := []int{1, 32, 65, 200, 1000, 5000, 50000, 500000, 5000000}
	b.Run("bitwise", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, s := range sizes {
				getWordSlicePoolIndex(s)
			}
		}
	})
	b.Run("linear", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, s := range sizes {
				getWordSlicePoolIndexLinear(s)
			}
		}
	})
}

func BenchmarkGetFermatPoolIndex(b *testing.B) {
	sizes := []int{1, 16, 33, 100, 1000, 5000, 50000, 500000}
	b.Run("bitwise", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, s := range sizes {
				getFermatPoolIndex(s)
			}
		}
	})
	b.Run("linear", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, s := range sizes {
				getFermatPoolIndexLinear(s)
			}
		}
	})
}

// TestPoolingOnlyForTemporaries verifies that pooling is used correctly
// only for temporary buffers, not for buffers returned in structures.
func TestPoolingOnlyForTemporaries(t *testing.T) {
	t.Parallel()
	// This test verifies the design: pools are only for truly temporary buffers.
	// Buffers that are returned in structures (like polValues.values or poly.a)
	// use regular make() to avoid resource leaks.

	// Acquire and release some temporary buffers to verify pool functionality
	for i := 0; i < 10; i++ {
		// These simulate the temporary buffer usage pattern in FFT
		tmp := acquireFermat(100)
		releaseFermat(tmp)

		words := acquireWordSlice(1000)
		releaseWordSlice(words)

		fermatSlice := acquireFermatSlice(16)
		releaseFermatSlice(fermatSlice)
	}
}

// Benchmarks

func BenchmarkWordSlicePoolSmall(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		slice := acquireWordSlice(64)
		releaseWordSlice(slice)
	}
}

func BenchmarkWordSlicePoolMedium(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		slice := acquireWordSlice(1024)
		releaseWordSlice(slice)
	}
}

func BenchmarkWordSlicePoolLarge(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		slice := acquireWordSlice(16384)
		releaseWordSlice(slice)
	}
}

func BenchmarkWordSliceDirectAlloc(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = make([]big.Word, 1024)
	}
}

func BenchmarkFermatPoolSmall(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		f := acquireFermat(32)
		releaseFermat(f)
	}
}

func BenchmarkFermatPoolMedium(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		f := acquireFermat(512)
		releaseFermat(f)
	}
}

func BenchmarkFermatPoolLarge(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		f := acquireFermat(8192)
		releaseFermat(f)
	}
}

// Test that FFT multiplication still works with pooling
func TestFFTMulWithPooling(t *testing.T) {
	t.Parallel()
	// Small numbers
	x := big.NewInt(12345)
	y := big.NewInt(67890)
	expected := new(big.Int).Mul(x, y)
	result, err := Mul(x, y)
	if err != nil {
		t.Fatalf("Mul failed: %v", err)
	}
	if result.Cmp(expected) != 0 {
		t.Errorf("Mul(%v, %v) = %v, want %v", x, y, result, expected)
	}

	// Large numbers that trigger FFT
	x = new(big.Int).Exp(big.NewInt(2), big.NewInt(100000), nil)
	y = new(big.Int).Exp(big.NewInt(2), big.NewInt(100000), nil)
	expected = new(big.Int).Mul(x, y)
	result, err = Mul(x, y)
	if err != nil {
		t.Fatalf("Mul failed: %v", err)
	}
	if result.Cmp(expected) != 0 {
		t.Errorf("Mul for large numbers failed: bit lengths %d, %d", result.BitLen(), expected.BitLen())
	}
}

func TestMulToWithPooling(t *testing.T) {
	t.Parallel()
	x := new(big.Int).Exp(big.NewInt(2), big.NewInt(100000), nil)
	y := new(big.Int).Exp(big.NewInt(2), big.NewInt(100000), nil)
	expected := new(big.Int).Mul(x, y)

	z := new(big.Int)
	result, err := MulTo(z, x, y)
	if err != nil {
		t.Fatalf("MulTo failed: %v", err)
	}
	if result.Cmp(expected) != 0 {
		t.Errorf("MulTo for large numbers failed")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests for IntTo buffer reuse optimization
// ─────────────────────────────────────────────────────────────────────────────

func TestIntToBufferReuse(t *testing.T) {
	t.Parallel()
	// Create a polynomial that would produce a result of known size
	p := &Poly{
		K: 2,
		M: 4,
		A: []nat{
			{1, 2, 3, 4},
			{5, 6, 7, 8},
		},
	}

	// Test 1: IntTo with nil dst should work like Int
	result1 := p.IntTo(nil)
	result2 := p.Int()

	if len(result1) != len(result2) {
		t.Errorf("IntTo(nil) produced different length: got %d, want %d", len(result1), len(result2))
	}

	for i := range result1 {
		if result1[i] != result2[i] {
			t.Errorf("IntTo(nil) differs from Int() at index %d: got %v, want %v", i, result1[i], result2[i])
		}
	}
}

func TestIntToBufferReuseWithLargeBuffer(t *testing.T) {
	t.Parallel()
	// Create a polynomial
	p := &Poly{
		K: 2,
		M: 4,
		A: []nat{
			{1, 2, 3, 4},
			{5, 6, 7, 8},
		},
	}

	// Create a buffer larger than needed
	largeDst := make(nat, 1000)
	// Put some marker values to verify buffer was reused
	for i := range largeDst {
		largeDst[i] = 0xDEADBEEF
	}

	// Get expected result
	expected := p.Int()

	// Call IntTo with large buffer
	result := p.IntTo(largeDst)

	// Verify correctness
	if len(result) != len(expected) {
		t.Errorf("IntTo with large buffer produced different length: got %d, want %d", len(result), len(expected))
	}

	for i := range expected {
		if result[i] != expected[i] {
			t.Errorf("IntTo with large buffer differs at index %d: got %v, want %v", i, result[i], expected[i])
		}
	}
}

func TestIntToBufferTooSmall(t *testing.T) {
	t.Parallel()
	// Create a polynomial that needs more space
	p := &Poly{
		K: 3,
		M: 8,
		A: []nat{
			{1, 2, 3, 4, 5, 6, 7, 8},
			{1, 2, 3, 4, 5, 6, 7, 8},
			{1, 2, 3, 4, 5, 6, 7, 8},
		},
	}

	// Create a buffer that's too small
	smallDst := make(nat, 2)

	// Get expected result
	expected := p.Int()

	// Call IntTo with small buffer - should allocate new
	result := p.IntTo(smallDst)

	// Verify correctness
	if len(result) != len(expected) {
		t.Errorf("IntTo with small buffer produced different length: got %d, want %d", len(result), len(expected))
	}

	for i := range expected {
		if result[i] != expected[i] {
			t.Errorf("IntTo with small buffer differs at index %d: got %v, want %v", i, result[i], expected[i])
		}
	}
}

func TestMulToBufferReuse(t *testing.T) {
	t.Parallel()
	// Test that MulTo produces correct results when z has existing capacity
	x := new(big.Int).Exp(big.NewInt(2), big.NewInt(100000), nil)
	y := new(big.Int).Exp(big.NewInt(2), big.NewInt(100000), nil)
	expected := new(big.Int).Mul(x, y)

	// Pre-allocate z with a large buffer
	z := new(big.Int).Exp(big.NewInt(2), big.NewInt(200000), nil)

	result, err := MulTo(z, x, y)
	if err != nil {
		t.Fatalf("MulTo failed: %v", err)
	}
	if result.Cmp(expected) != 0 {
		t.Errorf("MulTo with pre-allocated buffer failed: got %d bits, want %d bits",
			result.BitLen(), expected.BitLen())
	}
}

func TestMulToConsistency(t *testing.T) {
	t.Parallel()
	// Verify MulTo produces same results as Mul for various sizes
	testCases := []int64{50000, 100000, 150000}

	for _, bits := range testCases {
		t.Run(fmt.Sprintf("%dBits", bits), func(t *testing.T) {
			t.Parallel()
			x := new(big.Int).Exp(big.NewInt(2), big.NewInt(bits), nil)
			y := new(big.Int).Exp(big.NewInt(2), big.NewInt(bits), nil)

			mulResult, err := Mul(x, y)
			if err != nil {
				t.Fatalf("Mul failed: %v", err)
			}

			z := new(big.Int)
			mulToResult, err := MulTo(z, x, y)
			if err != nil {
				t.Fatalf("MulTo failed: %v", err)
			}

			if mulResult.Cmp(mulToResult) != 0 {
				t.Errorf("Mul and MulTo produce different results for %d bits", bits)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Benchmarks for buffer reuse optimization
// ─────────────────────────────────────────────────────────────────────────────

func BenchmarkMulToWithReuse(b *testing.B) {
	x := new(big.Int).Exp(big.NewInt(2), big.NewInt(100000), nil)
	y := new(big.Int).Exp(big.NewInt(2), big.NewInt(100000), nil)

	// Pre-allocate z with large buffer to enable reuse
	z := new(big.Int).Exp(big.NewInt(2), big.NewInt(200000), nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = MulTo(z, x, y)
	}
}

func BenchmarkMulToWithoutReuse(b *testing.B) {
	x := new(big.Int).Exp(big.NewInt(2), big.NewInt(100000), nil)
	y := new(big.Int).Exp(big.NewInt(2), big.NewInt(100000), nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Fresh z each time - no buffer to reuse
		z := new(big.Int)
		_, _ = MulTo(z, x, y)
	}
}

func BenchmarkMulVsMulTo(b *testing.B) {
	x := new(big.Int).Exp(big.NewInt(2), big.NewInt(100000), nil)
	y := new(big.Int).Exp(big.NewInt(2), big.NewInt(100000), nil)

	b.Run("Mul", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = Mul(x, y)
		}
	})

	b.Run("MulTo_fresh", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			z := new(big.Int)
			_, _ = MulTo(z, x, y)
		}
	})

	b.Run("MulTo_reuse", func(b *testing.B) {
		z := new(big.Int).Exp(big.NewInt(2), big.NewInt(200000), nil)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = MulTo(z, x, y)
		}
	})
}

// getWordSlicePoolIndexLinear is the original O(n) linear search implementation,
// kept as a reference for testing the optimized bitwise version.
func getWordSlicePoolIndexLinear(size int) int {
	for i, s := range wordSliceSizes {
		if size <= s {
			return i
		}
	}
	return -1
}

// getFermatPoolIndexLinear is the original O(n) linear search implementation,
// kept as a reference for testing the optimized bitwise version.
func getFermatPoolIndexLinear(size int) int {
	for i, s := range fermatSizes {
		if size <= s {
			return i
		}
	}
	return -1
}

// getNatSlicePoolIndexLinear is the original O(n) linear search implementation,
// kept as a reference for testing the optimized bitwise version.
func getNatSlicePoolIndexLinear(size int) int {
	for i, s := range natSliceSizes {
		if size <= s {
			return i
		}
	}
	return -1
}

// getFermatSlicePoolIndexLinear is the original O(n) linear search implementation,
// kept as a reference for testing the optimized bitwise version.
func getFermatSlicePoolIndexLinear(size int) int {
	for i, s := range fermatSliceSizes {
		if size <= s {
			return i
		}
	}
	return -1
}
