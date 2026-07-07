package memory

import (
	"math"
	"math/big"
	"testing"
	"unsafe"
)

func TestCalculationArena_AllocBigInt(t *testing.T) {
	t.Parallel()

	arena := NewCalculationArena(1_000_000) // F(1M)

	z := arena.AllocBigInt(1000)
	if z == nil {
		t.Fatal("AllocBigInt returned nil")
	}
	if cap(z.Bits()) < 1000 {
		t.Errorf("cap(z.Bits()) = %d, want >= 1000", cap(z.Bits()))
	}

	// z should be usable as a normal big.Int
	z.SetInt64(42)
	if z.Int64() != 42 {
		t.Errorf("z = %d, want 42", z.Int64())
	}
}

func TestCalculationArena_MultipleAllocs_NoAliasing(t *testing.T) {
	t.Parallel()

	arena := NewCalculationArena(100_000)

	allocs := make([]*big.Int, 5)
	for i := range allocs {
		allocs[i] = arena.AllocBigInt(100)
		allocs[i].SetInt64(int64(i + 1))
	}

	// All values should be independent (no aliasing)
	for i, z := range allocs {
		if z.Int64() != int64(i+1) {
			t.Errorf("allocs[%d] = %d, want %d", i, z.Int64(), i+1)
		}
	}

	// Verify backing arrays don't overlap
	for i := 0; i < len(allocs); i++ {
		for j := i + 1; j < len(allocs); j++ {
			bi := allocs[i].Bits()
			bj := allocs[j].Bits()
			if cap(bi) > 0 && cap(bj) > 0 {
				pi := uintptr(unsafe.Pointer(&bi[:cap(bi)][0]))
				pj := uintptr(unsafe.Pointer(&bj[:cap(bj)][0]))
				endI := pi + uintptr(cap(bi))*unsafe.Sizeof(big.Word(0))
				endJ := pj + uintptr(cap(bj))*unsafe.Sizeof(big.Word(0))
				if (pi >= pj && pi < endJ) || (pj >= pi && pj < endI) {
					t.Errorf("allocs[%d] and allocs[%d] have overlapping backing arrays", i, j)
				}
			}
		}
	}
}

func TestCalculationArena_Reset(t *testing.T) {
	t.Parallel()

	arena := NewCalculationArena(100_000)

	_ = arena.AllocBigInt(500)
	_ = arena.AllocBigInt(500)

	used := arena.UsedWords()
	if used < 1000 {
		t.Errorf("UsedWords() = %d, want >= 1000", used)
	}

	arena.Reset()

	if arena.UsedWords() != 0 {
		t.Errorf("UsedWords() after Reset = %d, want 0", arena.UsedWords())
	}
}

func TestCalculationArena_Fallback(t *testing.T) {
	t.Parallel()

	// Tiny arena that forces fallback
	arena := NewCalculationArena(10)

	// Request more than arena can hold
	z := arena.AllocBigInt(1_000_000)
	if z == nil {
		t.Fatal("AllocBigInt should fallback to heap, not return nil")
	}
}

func TestCalculationArena_PreSizeFromArena(t *testing.T) {
	t.Parallel()

	arena := NewCalculationArena(100_000)

	z := new(big.Int)
	arena.PreSizeFromArena(z, 500)

	if cap(z.Bits()) < 500 {
		t.Errorf("cap after PreSizeFromArena = %d, want >= 500", cap(z.Bits()))
	}

	// Should be a no-op if already large enough
	arena.PreSizeFromArena(z, 100)
}

// TestArenaTotalWords_ClampNoUB checks the sizing helper: bit-identical to the
// naive formula for realistic n, and clamped (no float->int UB, no int
// overflow) for a physically uncomputable n. It does NOT allocate the clamped
// size — only the integer sizing is exercised.
func TestArenaTotalWords_ClampNoUB(t *testing.T) {
	t.Parallel()

	naive := func(n uint64) int {
		return (int(float64(n)*fibonacciGrowthFactor/64) + 1) * 10
	}

	for _, n := range []uint64{1001, 100_000, 1_000_000, 1_000_000_000, 1_000_000_000_000} {
		if got, want := arenaTotalWords(n), naive(n); got != want {
			t.Errorf("arenaTotalWords(%d) = %d, want %d (must match naive formula)", n, got, want)
		}
	}

	if got := arenaTotalWords(math.MaxUint64); got != maxReasonableWords {
		t.Errorf("arenaTotalWords(MaxUint64) = %d, want clamp %d", got, maxReasonableWords)
	}
}

func TestCalculationArena_CapacityWords(t *testing.T) {
	t.Parallel()

	arena := NewCalculationArena(100_000)
	if arena.CapacityWords() == 0 {
		t.Error("CapacityWords() should be > 0 for n=100000")
	}

	small := NewCalculationArena(10)
	if small.CapacityWords() != 0 {
		t.Errorf("CapacityWords() should be 0 for small n, got %d", small.CapacityWords())
	}
}

// TestCalculationArena_AllocBigInt_NonPositiveWords covers the guard branch:
// a non-positive request must yield a usable zero big.Int and must not
// consume arena capacity.
func TestCalculationArena_AllocBigInt_NonPositiveWords(t *testing.T) {
	t.Parallel()

	arena := NewCalculationArena(100_000)
	for _, words := range []int{0, -7} {
		z := arena.AllocBigInt(words)
		if z == nil {
			t.Fatalf("AllocBigInt(%d) returned nil", words)
		}
		if z.Sign() != 0 {
			t.Errorf("AllocBigInt(%d) should be zero, got %s", words, z)
		}
		if got := arena.UsedWords(); got != 0 {
			t.Errorf("AllocBigInt(%d) consumed %d arena words, want 0", words, got)
		}
		// The returned int must behave like a regular big.Int.
		z.SetInt64(7)
		if z.Int64() != 7 {
			t.Errorf("AllocBigInt(%d) result unusable: got %d, want 7", words, z.Int64())
		}
	}
}

// TestCalculationArena_AllocBigInt_ExhaustedFallback covers the heap fallback
// on a non-nil but too-small buffer: the bump pointer must not advance on a
// failed arena allocation, and the arena must stay usable afterward.
func TestCalculationArena_AllocBigInt_ExhaustedFallback(t *testing.T) {
	t.Parallel()

	arena := NewCalculationArena(2000)
	capWords := arena.CapacityWords()
	if capWords == 0 {
		t.Fatal("precondition: arena must have a backing buffer for n=2000")
	}

	z := arena.AllocBigInt(capWords + 1)
	if z == nil {
		t.Fatal("AllocBigInt should fall back to heap, not return nil")
	}
	if got := cap(z.Bits()); got < capWords+1 {
		t.Errorf("fallback cap = %d, want >= %d", got, capWords+1)
	}
	if got := arena.UsedWords(); got != 0 {
		t.Errorf("fallback consumed %d arena words, want 0", got)
	}

	// A subsequent fitting request must still be served from the arena.
	w := arena.AllocBigInt(capWords)
	w.SetInt64(9)
	if w.Int64() != 9 {
		t.Errorf("post-fallback alloc unusable: got %d, want 9", w.Int64())
	}
	if got := arena.UsedWords(); got != capWords {
		t.Errorf("UsedWords() = %d, want %d", got, capWords)
	}
}

// TestCalculationArena_PreSizeFromArena_Guards covers the nil/non-positive
// guard: the call must be a no-op (no panic, value preserved, no arena
// consumption).
func TestCalculationArena_PreSizeFromArena_Guards(t *testing.T) {
	t.Parallel()

	arena := NewCalculationArena(100_000)
	arena.PreSizeFromArena(nil, 100) // must not panic

	z := big.NewInt(42)
	arena.PreSizeFromArena(z, 0)
	arena.PreSizeFromArena(z, -1)
	if z.Int64() != 42 {
		t.Errorf("guard must not clobber the value: got %d, want 42", z.Int64())
	}
	if got := arena.UsedWords(); got != 0 {
		t.Errorf("guard paths consumed %d arena words, want 0", got)
	}
}

// TestPreSizeBigInt_Guards exercises the heap pre-sizing helper's defensive
// guards directly (its callers pre-filter these cases, so they are otherwise
// unreachable). The growth path intentionally zeroes z, so the nil,
// non-positive and sufficient-capacity early returns are what protect a live
// value from being clobbered.
func TestPreSizeBigInt_Guards(t *testing.T) {
	t.Parallel()

	preSizeBigInt(nil, 64) // must not panic

	z := big.NewInt(42)
	preSizeBigInt(z, 0)
	preSizeBigInt(z, -5)
	if z.Int64() != 42 {
		t.Errorf("non-positive words must not modify z: got %d, want 42", z.Int64())
	}

	// cap(z.Bits()) >= 1 for the value 42: requesting 1 word must be a no-op.
	preSizeBigInt(z, 1)
	if z.Int64() != 42 {
		t.Errorf("sufficient capacity must preserve the value: got %d, want 42", z.Int64())
	}

	// Growth path zeroes the value by contract (SetBits with an empty slice).
	grown := big.NewInt(42)
	preSizeBigInt(grown, 128)
	if got := cap(grown.Bits()); got < 128 {
		t.Errorf("cap = %d, want >= 128", got)
	}
	if grown.Sign() != 0 {
		t.Errorf("grown value should be zero after pre-sizing, got %s", grown)
	}
}
