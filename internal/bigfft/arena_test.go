package bigfft

import (
	"testing"
)

func TestEstimateMemoryNeeds(t *testing.T) {
	tests := []struct {
		name     string
		n        uint64
		validate func(t *testing.T, est MemoryEstimate)
	}{
		{
			name: "small N",
			n:    1000,
			validate: func(t *testing.T, est MemoryEstimate) {
				if est.MaxWordSliceSize <= 0 {
					t.Error("MaxWordSliceSize should be positive")
				}
				if est.MaxFermatSize <= 0 {
					t.Error("MaxFermatSize should be positive")
				}
			},
		},
		{
			name: "medium N",
			n:    1_000_000,
			validate: func(t *testing.T, est MemoryEstimate) {
				if est.MaxWordSliceSize < est.MaxFermatSize {
					t.Error("MaxWordSliceSize should be >= MaxFermatSize for FFT buffers")
				}
			},
		},
		{
			name: "large N",
			n:    100_000_000,
			validate: func(t *testing.T, est MemoryEstimate) {
				// Large N should have larger estimates
				if est.MaxWordSliceSize < 1000 {
					t.Error("Large N should have substantial buffer estimates")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			est := EstimateMemoryNeeds(tt.n)
			tt.validate(t, est)
		})
	}
}

func TestCalculationArena(t *testing.T) {
	arena := NewCalculationArena(10_000_000)
	if arena == nil {
		t.Fatal("NewCalculationArena returned nil")
	}

	// Test allocation methods (they should work even if using global pools)
	wordSlice := arena.AllocWordSlice(100)
	if len(wordSlice) < 100 {
		t.Error("AllocWordSlice should return slice of at least requested size")
	}

	fermatSlice := arena.AllocFermat(50)
	if len(fermatSlice) < 50 {
		t.Error("AllocFermat should return slice of at least requested size")
	}

	natSlice := arena.AllocNatSlice(20)
	if len(natSlice) < 20 {
		t.Error("AllocNatSlice should return slice of at least requested size")
	}

	fermatSliceSlice := arena.AllocFermatSlice(10)
	if len(fermatSliceSlice) < 10 {
		t.Error("AllocFermatSlice should return slice of at least requested size")
	}

	// Release should not panic
	arena.Release()
}

func TestPreWarmPools(t *testing.T) {
	// Pre-warm pools for a medium-sized calculation
	PreWarmPools(1_000_000)

	// Verify that pools have been warmed by checking that we can acquire
	// buffers without allocation (though we can't directly verify this,
	// we can at least ensure the function doesn't panic)
	wordSlice := acquireWordSlice(1024)
	if len(wordSlice) < 1024 {
		t.Error("acquireWordSlice should work after PreWarmPools")
	}
	releaseWordSlice(wordSlice)
}

func BenchmarkEstimateMemoryNeeds(b *testing.B) {
	ns := []uint64{1_000, 1_000_000, 10_000_000, 100_000_000}
	for _, n := range ns {
		b.Run("", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = EstimateMemoryNeeds(n)
			}
		})
	}
}

func BenchmarkCalculationArenaAlloc(b *testing.B) {
	arena := NewCalculationArena(10_000_000)
	b.ResetTimer()
	
	b.Run("AllocWordSlice", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			slice := arena.AllocWordSlice(1024)
			_ = slice
		}
	})
	
	b.Run("AllocFermat", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			slice := arena.AllocFermat(512)
			_ = slice
		}
	})
}

