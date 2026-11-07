package fibonacci

import (
	"context"
	"math/big"

	"github.com/agbruneau/FibGo/internal/progress"
)

// FFTBasedCalculator is a specialized Fibonacci calculator that uses the Fast
// Doubling algorithm, but with a significant modification: it exclusively relies
// on FFT-based multiplication for all big.Int operations.
//
// Unlike the FastDoublingCalculator calculator, which adaptively switches
// between standard and FFT-based multiplication, this implementation uses
// the FFT doubling step for every multiplication, regardless of the numbers'
// size. This makes it a tool for benchmarking the FFT path in isolation.
//
// It is NOT the faster calculator at the sizes the repo measures: in
// docs/audits/bench-baseline.txt the medians are FFTBased 5.13 ms vs
// FastDoubling 3.15 ms at F(1M), and 29.1 ms vs 23.9 ms at F(10M). Any
// crossover where forcing FFT at every size pays off lies beyond F(10M) and
// is unmeasured here — so "use this for very large n" is a hypothesis, not a
// result.
type FFTBasedCalculator struct{}

// Name returns the name of the algorithm, indicating its reliance on FFT.
//
// Returns:
//   - string: The name of the algorithm.
func (c *FFTBasedCalculator) Name() string {
	return "FFT-Based Doubling"
}

// CalculateCore computes F(n) using the Fast Doubling algorithm, with all
// multiplications performed via FFT.
//
// This implementation uses the DoublingFramework with FFTOnlyStrategy to
// consistently use FFT-based multiplication for all operations, regardless
// of operand size. That makes it a way to exercise the FFT path in isolation;
// it is not the faster calculator at any size the repo measures (see the type
// doc above).
//
// Parameters:
//   - ctx: The context for managing cancellation and deadlines.
//   - reporter: The function used for reporting progress.
//   - n: The index of the Fibonacci number to calculate.
//   - opts: Configuration options for the calculation (thresholds ignored).
//
// Returns:
//   - *big.Int: The calculated Fibonacci number.
//   - error: An error if one occurred (e.g., context cancellation).
func (c *FFTBasedCalculator) CalculateCore(ctx context.Context, reporter progress.ProgressCallback, n uint64, opts Options) (*big.Int, error) {
	// AcquireStateForN: state-bound arena (see fastdoubling.go for the
	// invariant). On success we hand off via ReleaseStateWithResult so the
	// returned big.Int does not alias pooled arena memory.
	s := AcquireStateForN(n)

	// Use framework with FFT-only strategy
	strategy := &FFTOnlyStrategy{}
	framework := NewDoublingFramework(strategy)

	// Execute the doubling loop (no parallelization for FFT-based)
	raw, err := framework.ExecuteDoublingLoop(ctx, reporter, n, opts, s, false)
	if err != nil {
		ReleaseState(s)
		return nil, err
	}
	return ReleaseStateWithResult(s, raw), nil
}
