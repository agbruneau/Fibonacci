package memory

import (
	"math/big"
	"testing"
)

// TestPreSizeFromArena_HeapFallback covers the arena-exhaustion branch of
// PreSizeFromArena -> preSizeBigInt (arena.go), which the existing test never
// reaches because it always sizes the arena with room to spare (audit F-008).
// The fallback must give z a backing array of at least `words` capacity while
// leaving z numerically zero, and (being a fresh heap buffer) must not alias
// the arena.
func TestPreSizeFromArena_HeapFallback(t *testing.T) {
	t.Parallel()

	t.Run("nil_arena_buffer", func(t *testing.T) {
		t.Parallel()
		// n < 1000 yields an arena with a nil backing buffer.
		a := NewCalculationArena(100)
		if a.CapacityWords() != 0 {
			t.Fatalf("precondition: expected empty arena, got capacity %d", a.CapacityWords())
		}
		z := new(big.Int)
		const words = 512
		a.PreSizeFromArena(z, words)
		if got := cap(z.Bits()); got < words {
			t.Errorf("cap(z.Bits()) = %d, want >= %d", got, words)
		}
		if z.Sign() != 0 {
			t.Errorf("z should be zero after pre-sizing, got %s", z.String())
		}
	})

	t.Run("exhausted_arena", func(t *testing.T) {
		t.Parallel()
		a := NewCalculationArena(2000) // small but non-nil backing buffer
		capWords := a.CapacityWords()
		if capWords == 0 {
			t.Skip("arena unexpectedly empty for n=2000")
		}
		z := new(big.Int)
		// Request more words than the arena can satisfy -> heap fallback.
		words := capWords + 1024
		a.PreSizeFromArena(z, words)
		if got := cap(z.Bits()); got < words {
			t.Errorf("cap(z.Bits()) = %d, want >= %d", got, words)
		}
		if z.Sign() != 0 {
			t.Errorf("z should be zero after pre-sizing, got %s", z.String())
		}
	})
}
