package fibonacci

import (
	"context"
	"fmt"
	"math/big"

	"github.com/agbru/fibcalc/internal/bigfft"
)

// FFTSafetyMarginWords is the safety margin added to FFT word count to avoid overflow.
const FFTSafetyMarginWords = 2

// mulFFT performs the multiplication of two *big.Int instances, x and y.
// It uses an algorithm based on the Fast Fourier Transform (FFT), and returns
// the result as a new *big.Int. This method is particularly efficient for
// multiplying very large numbers, typically offering a time complexity of
// O(N log N), where N is the number of bits in the operands. It serves as a
// high-performance alternative to the standard big.Int.Mul method for numbers
// exceeding a certain size threshold.
//
// Parameters:
//   - x: The first operand.
//   - y: The second operand.
//
// Returns:
//   - *big.Int: The product of x and y.
//   - error: An error if the calculation failed.
func mulFFT(x, y *big.Int) (*big.Int, error) {
	return bigfft.Mul(x, y)
}

// sqrFFT performs optimized squaring of a *big.Int using FFT.
// Squaring is more efficient than general multiplication because
// we only need to transform x once, saving approximately 33% of
// the FFT computation time for large numbers.
//
// Parameters:
//   - x: The operand to square.
//
// Returns:
//   - *big.Int: The result of x * x.
//   - error: An error if the calculation failed.
func sqrFFT(x *big.Int) (*big.Int, error) {
	return bigfft.Sqr(x)
}

func smartMultiply(z, x, y *big.Int, fftThreshold int) (*big.Int, error) {
	if z == nil {
		z = new(big.Int)
	}

	bx := x.BitLen()
	by := y.BitLen()

	// Tier 1: FFT Multiplication for very large operands
	if fftThreshold > 0 && bx > fftThreshold && by > fftThreshold {
		return bigfft.MulTo(z, x, y)
	}

	// Tier 2: math/big Multiplication (uses optimized algorithms internally)
	return z.Mul(x, y), nil
}

// smartSquare performs optimized squaring, choosing between math/big.Mul and
// FFT (internal/bigfft) based on the operand size.
func smartSquare(z, x *big.Int, fftThreshold int) (*big.Int, error) {
	if z == nil {
		z = new(big.Int)
	}

	bx := x.BitLen()

	// Tier 1: FFT Squaring for very large operands
	if fftThreshold > 0 && bx > fftThreshold {
		return bigfft.SqrTo(z, x)
	}

	// Tier 2: math/big Squaring (uses optimized algorithms internally)
	return z.Mul(x, x), nil
}

// executeDoublingStepFFT performs the three multiplications of a doubling step
// while minimizing redundant FFT transforms.
// It transforms F_k and F_k1 only once and then performs the calculations.
//
// The `opts` parameter is currently unused inside the body (thresholds are
// only consulted by the caller to decide whether to route here), but the
// signature is kept symmetrical with executeDoublingStepMultiplications so
// both can be invoked uniformly from the strategy dispatcher — see
// strategy.go AdaptiveStrategy.ExecuteStep / FFTOnlyStrategy.ExecuteStep.
// Extra tuning knobs (e.g. an FFT-specific threshold override) will plug
// in through this parameter without a signature break.
func executeDoublingStepFFT(ctx context.Context, s *CalculationState, opts Options, inParallel bool) error { //nolint:unparam // opts kept for dispatcher signature parity, documented above
	// FK1 = F(k) * (2*F(k+1) - F(k))
	// F2k1 = F(k+1)^2 + F(k)^2

	// Determine result size bit length (approx 2 * bitlen(F_k))
	// FK1 is roughly N iterations * 2.
	// For GetFFTParams, we need words.
	fk1Words := len(s.FK1.Bits())
	targetWords := 2*fk1Words + FFTSafetyMarginWords
	k, m := bigfft.GetFFTParams(targetWords)

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("canceled before FFT transforms: %w", err)
	}

	// Transform operands once
	// Use ValueSize to get the correct coefficient length n in words
	nWords := bigfft.ValueSize(k, m, 2)
	n := nWords

	// P1-01: allocate a bump allocator for the two forward transforms.
	// The bump allocator supplies the transient fermat input buffers, keeping
	// them contiguous and avoiding per-call sync.Pool churn for the initial
	// transforms. The returned PolValues' output storage is still pooled
	// (see internal/bigfft/fft_poly.go transform()) and gets released via
	// fkPoly.Release()/fk1Poly.Release() in the parallel/sequential helpers.
	//
	// Sizing: two forward transforms each need ~K*(n+1) words of temp plus
	// some FFT working memory — EstimateBumpCapacity sizes for a full
	// mul/sqr, which is a safe over-estimate for two transforms.
	baCap := bigfft.EstimateBumpCapacity(2 * fk1Words)
	ba := bigfft.AcquireBumpAllocator(baCap)
	defer bigfft.ReleaseBumpAllocator(ba)

	pFk := bigfft.PolyFromInt(s.FK, k, m)
	fkPoly, err := pFk.TransformWithBump(n, ba)
	if err != nil {
		return fmt.Errorf("FFT transform FK failed: %w", err)
	}
	defer fkPoly.Release()

	pFk1 := bigfft.PolyFromInt(s.FK1, k, m)
	fk1Poly, err := pFk1.TransformWithBump(n, ba)
	if err != nil {
		return fmt.Errorf("FFT transform FK1 failed: %w", err)
	}
	defer fk1Poly.Release()

	if inParallel {
		return executeFFTTransformsParallel(ctx, &fkPoly, &fk1Poly, s, m)
	}
	return executeFFTTransformsSequential(ctx, &fkPoly, &fk1Poly, s, m)
}

// executeFFTTransformsParallel performs the three FFT pointwise multiplications
// and inverse transforms concurrently using executeParallel3.
//
// Optimization: No clones needed. PolValues.Mul() and PolValues.Sqr() are
// read-only on their receiver — they read p.Values[i] as operands to
// fermat.Mul(buf, x, y) where buf is a separate temporary, so the source
// PolValues are never modified. Multiple concurrent readers with no writers
// is safe, eliminating two Clone() calls that previously allocated and
// copied K*(n+1) words each (e.g., ~hundreds of KB for F(10M)).
func executeFFTTransformsParallel(ctx context.Context, fkPoly, fk1Poly *bigfft.PolValues, s *CalculationState, m int) error {
	return executeParallel3(ctx,
		func() error {
			v, err := fkPoly.Mul(fk1Poly)
			if err != nil {
				return err
			}
			defer v.Release()
			p, err := v.InvTransform()
			if err != nil {
				return err
			}
			defer p.Release()
			p.M = m
			s.T3 = p.IntToBigInt(s.T3)
			return nil
		},
		func() error {
			v, err := fk1Poly.Sqr()
			if err != nil {
				return err
			}
			defer v.Release()
			p, err := v.InvTransform()
			if err != nil {
				return err
			}
			defer p.Release()
			p.M = m
			s.T1 = p.IntToBigInt(s.T1)
			return nil
		},
		func() error {
			v, err := fkPoly.Sqr()
			if err != nil {
				return err
			}
			defer v.Release()
			p, err := v.InvTransform()
			if err != nil {
				return err
			}
			defer p.Release()
			p.M = m
			s.T2 = p.IntToBigInt(s.T2)
			return nil
		},
	)
}

// executeFFTTransformsSequential performs the three FFT pointwise multiplications
// and inverse transforms sequentially with context cancellation checks between operations.
func executeFFTTransformsSequential(ctx context.Context, fkPoly, fk1Poly *bigfft.PolValues, s *CalculationState, m int) error {
	v1, err := fkPoly.Mul(fk1Poly)
	if err != nil {
		return err
	}
	defer v1.Release()
	p1, err := v1.InvTransform()
	if err != nil {
		return err
	}
	defer p1.Release()
	p1.M = m
	s.T3 = p1.IntToBigInt(s.T3)

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("canceled after FFT multiply: %w", err)
	}

	v2, err := fk1Poly.Sqr()
	if err != nil {
		return err
	}
	defer v2.Release()
	p2, err := v2.InvTransform()
	if err != nil {
		return err
	}
	defer p2.Release()
	p2.M = m
	s.T1 = p2.IntToBigInt(s.T1)

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("canceled after FFT square FK1: %w", err)
	}

	v3, err := fkPoly.Sqr()
	if err != nil {
		return err
	}
	defer v3.Release()
	p3, err := v3.InvTransform()
	if err != nil {
		return err
	}
	defer p3.Release()
	p3.M = m
	s.T2 = p3.IntToBigInt(s.T2)

	return nil
}
