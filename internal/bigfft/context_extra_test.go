package bigfft

import (
	"math/big"
	"strings"
	"testing"
)

// TestMulToWithContextBranches covers the dispatch branches of the
// context-scoped MulTo variant: nil-context fallback, below-threshold
// math/big path, FFT path, and sign handling on the FFT path.
func TestMulToWithContextBranches(t *testing.T) {
	t.Parallel()

	small := big.NewInt(123456789)
	smallY := big.NewInt(987654321)
	bigX := makeBigInt(21, 2000)
	bigY := makeBigInt(22, 2000)
	negX := new(big.Int).Neg(makeBigInt(23, 2000))

	tests := []struct {
		name string
		ctx  *FFTContext
		x, y *big.Int
	}{
		{"nil context falls back to MulTo", nil, small, smallY},
		{"below threshold uses math/big", NewFFTContext(FFTContextOptions{}), small, smallY},
		{"FFT path", NewFFTContext(FFTContextOptions{}), bigX, bigY},
		{"FFT path negative operand", NewFFTContext(FFTContextOptions{}), negX, bigY},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			expected := new(big.Int).Mul(tc.x, tc.y)
			z := new(big.Int)
			got, err := MulToWithContext(tc.ctx, z, tc.x, tc.y)
			if err != nil {
				t.Fatalf("MulToWithContext failed: %v", err)
			}
			if got != z {
				t.Fatal("MulToWithContext must return its destination argument")
			}
			if got.Cmp(expected) != 0 {
				t.Fatal("MulToWithContext result differs from math/big reference")
			}
			if expected.Sign() != got.Sign() {
				t.Fatalf("sign mismatch: got %d, want %d", got.Sign(), expected.Sign())
			}
		})
	}
}

// TestWithContextSmallAndNilFallbacks covers the nil-context and
// below-threshold branches of the remaining *WithContext entry points.
func TestWithContextSmallAndNilFallbacks(t *testing.T) {
	t.Parallel()
	ctx := NewFFTContext(FFTContextOptions{})
	x := big.NewInt(0x1234567890ABCDEF)
	y := big.NewInt(0x0FEDCBA987654321)
	expectedSqr := new(big.Int).Mul(x, x)
	expectedMul := new(big.Int).Mul(x, y)

	if got, err := SqrWithContext(nil, x); err != nil || got.Cmp(expectedSqr) != 0 {
		t.Fatalf("SqrWithContext(nil): got %v, err %v", got, err)
	}
	if got, err := SqrWithContext(ctx, x); err != nil || got.Cmp(expectedSqr) != 0 {
		t.Fatalf("SqrWithContext small operand: got %v, err %v", got, err)
	}
	if got, err := SqrToWithContext(nil, new(big.Int), x); err != nil || got.Cmp(expectedSqr) != 0 {
		t.Fatalf("SqrToWithContext(nil): got %v, err %v", got, err)
	}
	z := new(big.Int)
	if got, err := SqrToWithContext(ctx, z, x); err != nil || got.Cmp(expectedSqr) != 0 || got != z {
		t.Fatalf("SqrToWithContext small operand: got %v, err %v", got, err)
	}
	if got, err := MulWithContext(ctx, x, y); err != nil || got.Cmp(expectedMul) != 0 {
		t.Fatalf("MulWithContext small operands: got %v, err %v", got, err)
	}
}

// TestFFTContextResolved verifies the lazy-singleton resolution contract: an
// empty context must pick up the package-level cache and semaphore, while a
// fully-populated context must keep its own instances.
func TestFFTContextResolved(t *testing.T) {
	t.Parallel()

	empty := &FFTContext{}
	r := empty.resolved()
	if r.Cache != GetTransformCache() {
		t.Error("resolved() must fill Cache from the global singleton")
	}
	if r.Semaphore != getSemaphore() {
		t.Error("resolved() must fill Semaphore from the global singleton")
	}

	own := NewFFTContext(FFTContextOptions{SemaphoreSize: 2})
	cache, sem := own.Cache, own.Semaphore
	r2 := own.resolved()
	if r2.Cache != cache || r2.Semaphore != sem {
		t.Error("resolved() must not replace explicitly-set Cache/Semaphore")
	}
}

// TestWithContextPanicPolicy asserts the ADR-0002 contract on the context
// entry points: a non-sentinel panic raised inside the FFT pipeline must be
// converted to an error identifying the entry point, never escape as a
// panic. A handcrafted context with a nil allocFactory deterministically
// panics (nil function call) before any pool buffer is acquired.
func TestWithContextPanicPolicy(t *testing.T) {
	t.Parallel()

	x := makeBigInt(31, 2000)
	y := makeBigInt(32, 2000)

	tests := []struct {
		name string
		call func(ctx *FFTContext) error
	}{
		{"MulWithContext", func(ctx *FFTContext) error { _, err := MulWithContext(ctx, x, y); return err }},
		{"MulToWithContext", func(ctx *FFTContext) error { _, err := MulToWithContext(ctx, new(big.Int), x, y); return err }},
		{"SqrWithContext", func(ctx *FFTContext) error { _, err := SqrWithContext(ctx, x); return err }},
		{"SqrToWithContext", func(ctx *FFTContext) error { _, err := SqrToWithContext(ctx, new(big.Int), x); return err }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// resolved() lazily fills Cache/Semaphore but leaves allocFactory
			// nil, so the pipeline panics before touching any pooled buffer.
			err := tc.call(&FFTContext{})
			if err == nil {
				t.Fatal("expected internal panic to be converted to an error, got nil")
			}
			if !strings.Contains(err.Error(), "panic in bigfft."+tc.name) {
				t.Fatalf("error must identify the entry point %q, got: %v", tc.name, err)
			}
		})
	}
}

// TestMulFFTCtxNilCacheAndSign drives mulFFTCtx directly with a context whose
// Cache is nil: transformCachedWithBumpCtx must fall back to the uncached
// transform, and the product sign must be preserved.
func TestMulFFTCtxNilCacheAndSign(t *testing.T) {
	t.Parallel()
	ctx := &FFTContext{
		Cache:     nil, // exercise the cache==nil fallback
		Semaphore: make(chan struct{}, 2),
		allocFactory: func(wordLen int) *BumpAllocator {
			return AcquireBumpAllocator(EstimateBumpCapacity(wordLen))
		},
		allocRelease: ReleaseBumpAllocator,
	}
	x := new(big.Int).Neg(makeBigInt(41, 300))
	y := makeBigInt(42, 300)
	expected := new(big.Int).Mul(x, y)

	got, err := mulFFTCtx(ctx, x, y)
	if err != nil {
		t.Fatalf("mulFFTCtx failed: %v", err)
	}
	if got.Cmp(expected) != 0 {
		t.Fatal("mulFFTCtx(nil cache) result differs from math/big reference")
	}
	if got.Sign() >= 0 {
		t.Fatal("product of negative and positive operands must be negative")
	}
}

// TestFFTContextSemaphoreExhaustedSequential pre-fills the context semaphore
// so every non-blocking acquire in the recursion fails: all levels must fall
// back to the sequential path and still produce a correct product (the
// no-deadlock contract of fourierRecursiveCtx).
func TestFFTContextSemaphoreExhaustedSequential(t *testing.T) {
	t.Parallel()
	ctx := NewFFTContext(FFTContextOptions{SemaphoreSize: 1, DisableCache: true})
	ctx.Semaphore <- struct{}{} // exhaust: non-blocking acquires must all fail

	x := makeBigInt(51, 2000)
	y := makeBigInt(52, 2000)
	expected := new(big.Int).Mul(x, y)

	got, err := MulWithContext(ctx, x, y)
	if err != nil {
		t.Fatalf("MulWithContext with exhausted semaphore failed: %v", err)
	}
	if got.Cmp(expected) != 0 {
		t.Fatal("sequential fallback produced a wrong product")
	}
}
