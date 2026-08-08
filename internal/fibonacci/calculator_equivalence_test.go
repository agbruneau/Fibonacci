package fibonacci

import (
	"math/big"
	"testing"
)

// A-20 — Oracle equivalence lock.
//
// internal/fibonacci.calculateSmall (calculator.go) intentionally duplicates
// cmd/generate-golden.fibBig. The duplication is load-bearing: the golden-data
// oracle must not import this library, otherwise the library would generate the
// ground truth for its own tests and every golden assertion would degrade into
// a tautology (P2-04). Today that contract is protected only by a prose comment
// on calculateSmall — nothing fails if the two implementations silently drift.
//
// This test is the executable lock. It compares calculateSmall against an
// INDEPENDENT iterative big.Int Fibonacci computed here, written in a
// deliberately different shape (explicit temp, no pointer-swap aliasing) so the
// comparison genuinely cross-checks the recurrence rather than re-deriving
// calculateSmall's exact statement sequence. It does NOT import
// cmd/generate-golden and does NOT read testdata/fibonacci_golden.json: this is
// an oracle-vs-oracle check, by construction not a golden test.
//
// Domain: k in [0, MaxFibUint64] (0..93) — exactly the small-N fast path that
// calculateSmall serves. The test PASSES against the current code; a failure
// means calculateSmall and the reference recurrence have diverged and the
// generate-golden duplicate almost certainly needs the lock-step update its
// comment describes.

// independentFib computes F(k) with a plain two-accumulator iteration using a
// dedicated scratch big.Int. It shares no code with calculateSmall and uses a
// structurally distinct update (no a, b = b, a pointer swap), so agreement is
// meaningful evidence both oracles encode the same sequence.
func independentFib(k uint64) *big.Int {
	if k == 0 {
		return big.NewInt(0)
	}
	if k == 1 {
		return big.NewInt(1)
	}
	prev := big.NewInt(0) // F(0)
	curr := big.NewInt(1) // F(1)
	next := new(big.Int)
	for i := uint64(2); i <= k; i++ {
		next.Add(prev, curr) // next: F(i-2) + F(i-1)
		prev.Set(curr)
		curr.Set(next)
	}
	return new(big.Int).Set(curr)
}

// TestCalculateSmall_OracleEquivalence asserts calculateSmall(k) == an
// independent big.Int Fibonacci for every k in the small-N domain [0, 93].
func TestCalculateSmall_OracleEquivalence(t *testing.T) {
	t.Parallel()

	for k := uint64(0); k <= MaxFibUint64; k++ {
		got := calculateSmall(k)
		want := independentFib(k)
		if got == nil {
			t.Fatalf("calculateSmall(%d) returned nil", k)
		}
		if got.Cmp(want) != 0 {
			t.Fatalf("oracle drift at k=%d: calculateSmall=%s independent=%s "+
				"(calculateSmall and cmd/generate-golden.fibBig must be updated in lock-step)",
				k, got.String(), want.String())
		}
	}
}

// TestCalculateSmall_KnownAnchors pins a few hand-verified values so a
// simultaneous identical mutation of BOTH the function under test and the
// in-test reference (which would slip past the equivalence comparison) is still
// caught. F(10)=55, F(50)=12586269025, F(93)=12200160415121876738 (the largest
// Fibonacci number fitting in uint64, matching MaxFibUint64).
func TestCalculateSmall_KnownAnchors(t *testing.T) {
	t.Parallel()

	anchors := map[uint64]string{
		0:  "0",
		1:  "1",
		2:  "1",
		10: "55",
		50: "12586269025",
		93: "12200160415121876738",
	}
	for k, want := range anchors {
		got := calculateSmall(k).String()
		if got != want {
			t.Fatalf("calculateSmall(%d) = %s, want %s", k, got, want)
		}
	}
}
