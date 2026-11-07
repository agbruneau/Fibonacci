package memory

import (
	"math/big"
	"math/bits"

	"github.com/agbruneau/FibGo/internal/fibonacci/fibmath"
)

// MaxReasonableWords caps arena sizing well above any computable F(n). It
// exists only to keep the float->int conversion and the ×10 multiplication in
// the sizing helpers from invoking impl-defined behavior or integer overflow on
// a physically impossible n — defense-in-depth.
//
// Word-size relative, not a literal 1<<60 (audit TYP-01). The constant was
// written as `1 << 60` in TWO places, here and in fibonacci/fastdoubling.go,
// and 1<<60 does not fit in a 32-bit int: `GOOS=linux GOARCH=386 go build ./...`
// failed to compile this file, which is how a documented "64-bit only" limit
// turned out to be a constant-overflow error rather than a deliberate guard.
// bits.UintSize-4 keeps the ×10 headroom (10 < 16) on both widths and leaves
// the 64-bit value unchanged at 1<<60 (~1.15e18 words, ~9 EB).
//
// Exported so fastdoubling.go shares this definition instead of repeating it.
const MaxReasonableWords = 1 << (bits.UintSize - 4)

// arenaTotalWords computes the arena size (in big.Word) for F(n): one int of
// ~n*fibmath.GrowthFactor bits, times 10 temporaries (over-sizing factor swept
// from 15 to 10 in ADR-0009 R4, 2026-07-07). The clamp only changes the result
// for n far beyond the computable range; for realistic n it is bit-identical to
// the naive estimatedBits/64+1 then ×10 computation.
func arenaTotalWords(n uint64) int {
	estimatedBits := fibmath.BitsFor(n)
	// A float64 outside the int range yields an impl-defined value on
	// conversion; clamp before converting.
	if estimatedBits/64 >= float64(MaxReasonableWords) {
		return MaxReasonableWords
	}
	wordsPerInt := int(estimatedBits/64) + 1
	if wordsPerInt > MaxReasonableWords/10 {
		return MaxReasonableWords
	}
	return wordsPerInt * 10
}

// CalculationArena pre-allocates a contiguous block of big.Word memory
// for all big.Int temporaries in a Fibonacci calculation. This eliminates
// per-buffer GC tracking and enables O(1) bulk release via Reset().
//
// The arena uses a bump-pointer allocation strategy: each AllocBigInt
// call advances the offset pointer. When capacity is exhausted, it falls
// back to standard heap allocation.
type CalculationArena struct {
	buf    []big.Word
	offset int
}

// NewCalculationArena creates an arena sized for F(n).
// It estimates the total memory needed for 10 big.Int temporaries
// of size ~ n * fibmath.GrowthFactor bits.
func NewCalculationArena(n uint64) *CalculationArena {
	if n < 1000 {
		return &CalculationArena{}
	}
	// x10 over-sizing = growth headroom for the big.Int temporaries; reduced
	// from x15 by the ADR-0009 R4 sweep (2026-07-07): x10 is CPU-optimal on the
	// reference machine (-2.9% geomean, -17% FFT B/op, flat allocs).
	totalWords := arenaTotalWords(n)
	return &CalculationArena{
		buf: make([]big.Word, totalWords),
	}
}

// AllocBigInt returns a new big.Int whose backing array is allocated from
// the arena. If the arena is exhausted, falls back to heap allocation.
//
// Test oracle (audit L-02): production pre-sizes through PreSizeFromArena and
// never calls this. It is kept, not deleted, because it and UsedWords are the
// only observable surface for the bump invariants PreSizeFromArena relies on —
// offset advance, three-index slicing, exhaustion fallback, Reset. Removing
// them would take arena_test.go's coverage of those invariants with them.
// Same reasoning as the bigfft oracle cluster, ADR-0009 R3.
func (a *CalculationArena) AllocBigInt(words int) *big.Int {
	if words <= 0 {
		return new(big.Int)
	}
	z := new(big.Int)
	if a.buf == nil || a.offset+words > len(a.buf) {
		// Fallback: allocate from heap
		buf := make([]big.Word, 0, words)
		z.SetBits(buf)
		return z
	}
	slice := a.buf[a.offset : a.offset+words : a.offset+words]
	a.offset += words
	z.SetBits(slice[:0]) // length 0, capacity words — z is 0
	return z
}

// PreSizeFromArena sets z's backing array to a slice from the arena.
// If the arena is exhausted, falls back to heap pre-sizing.
func (a *CalculationArena) PreSizeFromArena(z *big.Int, words int) {
	if z == nil || words <= 0 {
		return
	}
	if cap(z.Bits()) >= words {
		return // already large enough
	}
	if a.buf != nil && a.offset+words <= len(a.buf) {
		slice := a.buf[a.offset : a.offset+words : a.offset+words]
		a.offset += words
		z.SetBits(slice[:0])
	} else {
		preSizeBigInt(z, words)
	}
}

// preSizeBigInt ensures a big.Int has at least the specified word capacity.
// This avoids repeated reallocation during the doubling loop as values grow.
// Uses SetBits with a length-0 capacity-N slice to pre-allocate without
// changing the numeric value.
func preSizeBigInt(z *big.Int, words int) {
	if z == nil || words <= 0 {
		return
	}
	// Only pre-size if current capacity is smaller
	if cap(z.Bits()) >= words {
		return
	}
	// SetBits([]big.Word{}) with length 0 sets z to 0.
	// We use a slice with length=0, cap=words to give z the backing array.
	buf := make([]big.Word, 0, words)
	z.SetBits(buf)
}

// Reset resets the arena for reuse without freeing the backing block.
// All previously allocated big.Int values become invalid after Reset.
func (a *CalculationArena) Reset() {
	a.offset = 0
}

// UsedWords returns the number of words currently allocated from the arena.
//
// Test oracle; see AllocBigInt.
func (a *CalculationArena) UsedWords() int {
	return a.offset
}

// CapacityWords returns the total capacity of the arena in words.
func (a *CalculationArena) CapacityWords() int {
	return len(a.buf)
}
