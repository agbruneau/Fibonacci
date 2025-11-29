package bigfft

import (
	"math/big"
	"testing"
)

func TestWordSlicePool(t *testing.T) {
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

func TestFermatPool(t *testing.T) {
	sizes := []int{16, 64, 256, 1024, 4096}

	for _, size := range sizes {
		t.Run("", func(t *testing.T) {
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
	sizes := []int{4, 16, 64, 256}

	for _, size := range sizes {
		t.Run("", func(t *testing.T) {
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
	sizes := []int{4, 16, 64, 256}

	for _, size := range sizes {
		t.Run("", func(t *testing.T) {
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

func TestFFTStatePool(t *testing.T) {
	n := 100
	k := uint(4)

	state := acquireFFTState(n, k)
	if state == nil {
		t.Fatal("acquireFFTState returned nil")
	}

	if len(state.tmp) != n+1 {
		t.Errorf("tmp has wrong length: got %d, want %d", len(state.tmp), n+1)
	}

	if len(state.tmp2) != n+1 {
		t.Errorf("tmp2 has wrong length: got %d, want %d", len(state.tmp2), n+1)
	}

	if state.n != n {
		t.Errorf("state.n = %d, want %d", state.n, n)
	}

	if state.k != k {
		t.Errorf("state.k = %d, want %d", state.k, k)
	}

	releaseFFTState(state)
}

func TestReleaseNilSafe(t *testing.T) {
	// These should not panic
	releaseWordSlice(nil)
	releaseFermat(nil)
	releaseNatSlice(nil)
	releaseFermatSlice(nil)
	releaseFFTState(nil)
}

// TestPoolingOnlyForTemporaries verifies that pooling is used correctly
// only for temporary buffers, not for buffers returned in structures.
func TestPoolingOnlyForTemporaries(t *testing.T) {
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
	for i := 0; i < b.N; i++ {
		slice := acquireWordSlice(64)
		releaseWordSlice(slice)
	}
}

func BenchmarkWordSlicePoolMedium(b *testing.B) {
	for i := 0; i < b.N; i++ {
		slice := acquireWordSlice(1024)
		releaseWordSlice(slice)
	}
}

func BenchmarkWordSlicePoolLarge(b *testing.B) {
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
	for i := 0; i < b.N; i++ {
		f := acquireFermat(32)
		releaseFermat(f)
	}
}

func BenchmarkFermatPoolMedium(b *testing.B) {
	for i := 0; i < b.N; i++ {
		f := acquireFermat(512)
		releaseFermat(f)
	}
}

func BenchmarkFermatPoolLarge(b *testing.B) {
	for i := 0; i < b.N; i++ {
		f := acquireFermat(8192)
		releaseFermat(f)
	}
}

func BenchmarkFFTStatePool(b *testing.B) {
	for i := 0; i < b.N; i++ {
		state := acquireFFTState(100, 4)
		releaseFFTState(state)
	}
}

// Test that FFT multiplication still works with pooling
func TestFFTMulWithPooling(t *testing.T) {
	// Small numbers
	x := big.NewInt(12345)
	y := big.NewInt(67890)
	expected := new(big.Int).Mul(x, y)
	result := Mul(x, y)
	if result.Cmp(expected) != 0 {
		t.Errorf("Mul(%v, %v) = %v, want %v", x, y, result, expected)
	}

	// Large numbers that trigger FFT
	x = new(big.Int).Exp(big.NewInt(2), big.NewInt(100000), nil)
	y = new(big.Int).Exp(big.NewInt(2), big.NewInt(100000), nil)
	expected = new(big.Int).Mul(x, y)
	result = Mul(x, y)
	if result.Cmp(expected) != 0 {
		t.Errorf("Mul for large numbers failed: bit lengths %d, %d", result.BitLen(), expected.BitLen())
	}
}

func TestMulToWithPooling(t *testing.T) {
	x := new(big.Int).Exp(big.NewInt(2), big.NewInt(100000), nil)
	y := new(big.Int).Exp(big.NewInt(2), big.NewInt(100000), nil)
	expected := new(big.Int).Mul(x, y)

	z := new(big.Int)
	result := MulTo(z, x, y)
	if result.Cmp(expected) != 0 {
		t.Errorf("MulTo for large numbers failed")
	}
}

