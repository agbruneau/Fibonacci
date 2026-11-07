package fibmath

import (
	"math"
	"math/big"
	"testing"
)

// GrowthFactor must be log2(phi) to the precision the sizing code relies on.
// It is not a magic number: every buffer sized from n in this repository starts
// from it, and a wrong value silently under- or over-allocates everywhere at
// once.
func TestGrowthFactorIsLog2Phi(t *testing.T) {
	t.Parallel()

	phi := (1 + math.Sqrt(5)) / 2
	want := math.Log2(phi)

	// Exact, not within 1e-5: that tolerance let the truncation 0.69424 pass
	// although it under-sized F(n) from n = 10^8 on (audit 2026-11-07).
	if GrowthFactor != want {
		t.Errorf("GrowthFactor = %v, want log2(phi) = %v exactly (diff %v)", GrowthFactor, want, GrowthFactor-want)
	}
}

// TestGrowthFactorBoundsTheBitLength checks the property the sizing code needs
// at the n it actually runs, far beyond what the naive oracle below can reach:
// int(BitsFor(n))/64+1 words, the pre-sizing formula, must hold F(n), whose bit
// length is floor(n*log2(phi) - log2(sqrt(5))) + 1 by Binet's formula. With
// the old truncated constant it failed from n = 10^8 (2 words short).
func TestGrowthFactorBoundsTheBitLength(t *testing.T) {
	t.Parallel()

	log2Phi, log2Sqrt5 := math.Log2(math.Phi), math.Log2(math.Sqrt(5))
	for _, n := range []uint64{1_000_000, 100_000_000, 1_000_000_000, 1_000_000_000_000} {
		bitLen := math.Floor(float64(n)*log2Phi-log2Sqrt5) + 1
		need := int(math.Ceil(bitLen / 64))
		if got := int(BitsFor(n))/64 + 1; got < need {
			t.Errorf("n=%d: pre-sized %d words, F(n) needs %d", n, got, need)
		}
	}
}

// MaxUint64Index must be the largest n with F(n) <= MaxUint64. Computing both
// F(93) and F(94) here rather than asserting on the literal 93 means the
// constant is checked against the definition, not against itself.
func TestMaxUint64IndexIsTheLastFittingIndex(t *testing.T) {
	t.Parallel()

	maxUint64 := new(big.Int).SetUint64(math.MaxUint64)

	if fib(MaxUint64Index).Cmp(maxUint64) > 0 {
		t.Errorf("F(%d) does not fit in a uint64", MaxUint64Index)
	}
	if fib(MaxUint64Index+1).Cmp(maxUint64) <= 0 {
		t.Errorf("F(%d) also fits in a uint64, so %d is not the largest index",
			MaxUint64Index+1, MaxUint64Index)
	}
}

// BitsFor must approximate the real bit length of F(n). The tolerance is
// generous because the closed form drops the -log2(sqrt(5)) term, which costs
// about 1.16 bits at every n; what must not drift is the slope.
func TestBitsForApproximatesRealBitLength(t *testing.T) {
	t.Parallel()

	for _, n := range []uint64{100, 1_000, 10_000, 100_000} {
		got := BitsFor(n)
		actual := float64(fib(n).BitLen())

		if diff := math.Abs(got - actual); diff > 3 {
			t.Errorf("BitsFor(%d) = %.2f, actual bit length %.0f (diff %.2f)", n, got, actual, diff)
		}
		// The estimate must never fall below the truth by more than the
		// constant term: callers size buffers from it.
		if got < actual-3 {
			t.Errorf("BitsFor(%d) under-estimates: %.2f < %.0f", n, got, actual)
		}
	}
}

func TestBitsForZero(t *testing.T) {
	t.Parallel()
	if got := BitsFor(0); got != 0 {
		t.Errorf("BitsFor(0) = %v, want 0", got)
	}
}

// fib is a deliberately naive iterative oracle: this package must not depend on
// the calculators it serves.
func fib(n uint64) *big.Int {
	a, b := big.NewInt(0), big.NewInt(1)
	for i := uint64(0); i < n; i++ {
		a.Add(a, b)
		a, b = b, a
	}
	return a
}
