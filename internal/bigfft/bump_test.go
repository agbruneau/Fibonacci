package bigfft

import (
	"fmt"
	"math/big"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// Unit Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestBumpAllocatorAlloc(t *testing.T) {
	t.Parallel()
	ba := AcquireBumpAllocator(1000)
	defer ReleaseBumpAllocator(ba)

	// Test basic allocation
	slice1 := ba.Alloc(100)
	if len(slice1) != 100 {
		t.Errorf("Expected slice of length 100, got %d", len(slice1))
	}

	// Verify zeroed
	for i, v := range slice1 {
		if v != 0 {
			t.Errorf("Expected zero at index %d, got %d", i, v)
		}
	}

	// Test second allocation
	slice2 := ba.Alloc(200)
	if len(slice2) != 200 {
		t.Errorf("Expected slice of length 200, got %d", len(slice2))
	}

	// Verify they don't overlap
	slice1[0] = 42
	if slice2[0] == 42 {
		t.Error("Allocations should not overlap")
	}

	// Test used/remaining
	if ba.Used() != 300 {
		t.Errorf("Expected used = 300, got %d", ba.Used())
	}
	if ba.Remaining() != 700 {
		t.Errorf("Expected remaining = 700, got %d", ba.Remaining())
	}
}

func TestBumpAllocatorFallback(t *testing.T) {
	t.Parallel()
	ba := AcquireBumpAllocator(100)
	defer ReleaseBumpAllocator(ba)

	// Allocate more than capacity - should fall back to make()
	slice := ba.Alloc(200)
	if len(slice) != 200 {
		t.Errorf("Fallback allocation failed, expected 200, got %d", len(slice))
	}
}

func TestBumpAllocatorReset(t *testing.T) {
	t.Parallel()
	ba := AcquireBumpAllocator(500)
	defer ReleaseBumpAllocator(ba)

	ba.Alloc(100)
	ba.Alloc(100)

	if ba.Used() != 200 {
		t.Errorf("Expected used = 200, got %d", ba.Used())
	}

	ba.Reset()

	if ba.Used() != 0 {
		t.Errorf("Expected used = 0 after reset, got %d", ba.Used())
	}
	if ba.Remaining() != 500 {
		t.Errorf("Expected remaining = 500 after reset, got %d", ba.Remaining())
	}
}

func TestBumpAllocatorAllocFermat(t *testing.T) {
	t.Parallel()
	ba := AcquireBumpAllocator(1000)
	defer ReleaseBumpAllocator(ba)

	f := ba.AllocFermat(99) // Should allocate 100 words
	if len(f) != 100 {
		t.Errorf("Expected fermat of length 100, got %d", len(f))
	}

	// Verify zeroed
	for i, v := range f {
		if v != 0 {
			t.Errorf("Expected zero at index %d, got %d", i, v)
		}
	}
}

func TestBumpAllocatorAllocFermatSlice(t *testing.T) {
	t.Parallel()
	ba := AcquireBumpAllocator(10000)
	defer ReleaseBumpAllocator(ba)

	K := 8
	n := 15 // fermat size = 16

	fermats, bits := ba.AllocFermatSlice(K, n)

	if len(fermats) != K {
		t.Errorf("Expected %d fermats, got %d", K, len(fermats))
	}

	if len(bits) != K*(n+1) {
		t.Errorf("Expected bits length %d, got %d", K*(n+1), len(bits))
	}

	// Verify each fermat has correct length
	for i, f := range fermats {
		if len(f) != n+1 {
			t.Errorf("Fermat %d has length %d, expected %d", i, len(f), n+1)
		}
	}

	// Verify no overlap
	fermats[0][0] = 123
	if fermats[1][0] == 123 {
		t.Error("Fermat slices should not overlap")
	}
}

func TestAllocUnsafe(t *testing.T) {
	t.Parallel()
	ba := AcquireBumpAllocator(100)
	defer ReleaseBumpAllocator(ba)

	slice := ba.AllocUnsafe(50)
	if len(slice) != 50 {
		t.Errorf("Expected length 50, got %d", len(slice))
	}
}

func TestEstimateBumpCapacity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		wordLen int
		minCap  int // Minimum expected capacity
	}{
		{0, 0},
		{100, 100},
		{10000, 10000},
		{100000, 100000},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("Words=%d", tc.wordLen), func(t *testing.T) {
			t.Parallel()
			cap := EstimateBumpCapacity(tc.wordLen)
			if cap < tc.minCap {
				t.Errorf("EstimateBumpCapacity(%d) = %d, expected at least %d", tc.wordLen, cap, tc.minCap)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Benchmarks
// ─────────────────────────────────────────────────────────────────────────────

// BenchmarkAllocVsMake compares bump allocation to regular make()
func BenchmarkAllocVsMake(b *testing.B) {
	sizes := []int{64, 256, 1024, 4096}

	for _, size := range sizes {
		b.Run("Make_"+formatSize(size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = make([]big.Word, size)
			}
		})

		b.Run("Bump_"+formatSize(size), func(b *testing.B) {
			b.ReportAllocs()
			ba := AcquireBumpAllocator(size * b.N)
			defer ReleaseBumpAllocator(ba)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = ba.Alloc(size)
			}
		})

		b.Run("Pool_"+formatSize(size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				s := acquireWordSlice(size)
				releaseWordSlice(s)
			}
		})
	}
}

// BenchmarkFFTMulWithBump compares FFT multiplication with and without bump allocator
func BenchmarkFFTMulWithBump(b *testing.B) {
	sizes := []int{10000, 100000, 1000000}

	for _, size := range sizes {
		// Create test numbers
		x := make(nat, size)
		y := make(nat, size)
		for i := range x {
			x[i] = big.Word(i + 1)
			y[i] = big.Word(i + 2)
		}

		b.Run("fftmulTo_"+formatSize(size)+"_words", func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := fftmulTo(nil, x, y)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkFFTSqrWithBump compares FFT squaring with and without bump allocator
func BenchmarkFFTSqrWithBump(b *testing.B) {
	sizes := []int{10000, 100000, 1000000}

	for _, size := range sizes {
		x := make(nat, size)
		for i := range x {
			x[i] = big.Word(i + 1)
		}

		b.Run("fftsqrTo_"+formatSize(size)+"_words", func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := fftsqrTo(nil, x)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func formatSize(size int) string {
	if size >= 1000000 {
		return fmt.Sprintf("%dM", size/1000000)
	}
	if size >= 1000 {
		return fmt.Sprintf("%dK", size/1000)
	}
	return fmt.Sprintf("%d", size)
}

// BenchmarkBumpAllocatorReuse tests the benefit of reusing bump allocators
func BenchmarkBumpAllocatorReuse(b *testing.B) {
	b.ReportAllocs()
	capacity := 100000

	b.Run("NewEachTime", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ba := &BumpAllocator{buffer: make([]big.Word, capacity)}
			_ = ba.Alloc(1000)
			_ = ba.Alloc(2000)
			_ = ba.Alloc(3000)
		}
	})

	b.Run("PooledReuse", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ba := AcquireBumpAllocator(capacity)
			_ = ba.Alloc(1000)
			_ = ba.Alloc(2000)
			_ = ba.Alloc(3000)
			ReleaseBumpAllocator(ba)
		}
	})
}

// BenchmarkExtendedPoolSizes tests the new larger pool sizes
func BenchmarkExtendedPoolSizes(b *testing.B) {
	b.ReportAllocs()
	sizes := []int{262144, 1048576, 4194304} // 256K, 1M, 4M words

	for _, size := range sizes {
		b.Run("Pool_"+formatSize(size)+"_words", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				s := acquireWordSlice(size)
				releaseWordSlice(s)
			}
		})
	}
}
