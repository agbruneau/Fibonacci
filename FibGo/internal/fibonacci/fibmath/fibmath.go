// Package fibmath holds the closed-form facts about the Fibonacci sequence that
// several packages need to size buffers and validate inputs.
//
// It exists because those facts were written down more than once (audit
// TYP-04). log2(phi) appeared as the literal 0.69424 in three places —
// fibonacci/constants.go, fibonacci/memory/budget.go and bigfft/memory_est.go —
// each with its own comment explaining that it could not import the others, and
// 93 appeared twice, as fibonacci.MaxFibUint64 and as memory's baselineMinN,
// the second carrying a note that it "mirrors" the first. The book calls that
// the duplication worth removing: not repeated syntax, but one piece of
// knowledge written down in several places (ch. 6, p. 144).
//
// This package imports nothing, so any package under internal/fibonacci can
// depend on it without creating a cycle — which is what internal/fibonacci and
// internal/fibonacci/memory could not do with each other.
//
// internal/bigfft deliberately keeps its own literal: it is the kernel and
// imports no other package of this repository, a property ARCH.md states and
// the architecture gate would not otherwise protect. Its copy carries a pointer
// here.
package fibmath

// GrowthFactor is log2(phi), where phi = (1+sqrt(5))/2 is the golden ratio.
//
// F(n) is asymptotically phi^n / sqrt(5), so its bit length is n*log2(phi)
// to within a constant. Every buffer sized from n in this repository starts
// from this number.
const GrowthFactor = 0.69424

// MaxUint64Index is the largest n for which F(n) fits in a uint64: F(93) is
// 12200160415121876738, and F(94) exceeds 2^64.
//
// It marks the boundary between the trivial path (a uint64 loop, no pool, no
// arena) and the big-integer machinery, which is why both the calculator and
// the memory estimator need it.
const MaxUint64Index = 93

// BitsFor returns the approximate bit length of F(n).
//
// The result is a float64 because it feeds size computations that divide by the
// word size before converting; converting here would lose the fraction and
// under-size by up to one word.
func BitsFor(n uint64) float64 {
	return float64(n) * GrowthFactor
}
