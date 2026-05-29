// This file introduces FFTContext, an opt-in container that encapsulates the
// per-package globals (transform cache, concurrency semaphore, default
// allocator hint) so callers can run isolated FFT pipelines side by side.
//
// The package-level Mul/Sqr/MulTo/SqrTo entry points continue to use the
// historical globals (the package-level transform cache and concurrency
// semaphore) for full backward compatibility. Code that needs isolation
// (e.g. parallel benchmarks, multi-tenant scenarios) can build its own
// *FFTContext and call MulWithContext / SqrWithContext / MulToWithContext /
// SqrToWithContext.
//
// Audit reference: R3.7 — minimal scope (Partial). Full removal of all
// globals would touch >500 LOC across the hot path; the minimal scope keeps
// the hot path untouched and exposes context-aware variants only.

package bigfft

import (
	"fmt"
	"math/big"
	"runtime"
	"runtime/debug"
)

// FFTContextOptions configures a new FFTContext.
//
// All fields are optional; zero values fall back to sensible defaults:
//   - CacheConfig:   DefaultTransformCacheConfig().
//   - SemaphoreSize: runtime.NumCPU().
//   - DisableCache:  false (cache enabled).
type FFTContextOptions struct {
	// CacheConfig overrides the transform-cache configuration for this
	// context. When zero, DefaultTransformCacheConfig() is used.
	CacheConfig TransformCacheConfig
	// SemaphoreSize is the maximum number of concurrent goroutines spawned
	// by the FFT recursion in this context. When <= 0, runtime.NumCPU().
	SemaphoreSize int
	// DisableCache forces the context to skip its transform cache entirely.
	// Useful for benchmarks that want pure compute timings.
	DisableCache bool
}

// FFTContext encapsulates the per-context state previously held in
// package-level globals: a transform cache, a concurrency semaphore, and an
// allocator factory. Two FFTContexts created with NewFFTContext have fully
// independent caches and semaphores; they cannot cross-contaminate each other.
//
// FFTContext is safe for concurrent use by multiple goroutines once
// constructed: the transform cache uses internal locking and the semaphore is
// a buffered channel.
//
// Use the *WithContext entry points (MulWithContext, SqrWithContext, etc.)
// to drive an isolated pipeline. The package-level Mul/Sqr/MulTo/SqrTo
// continue to use the global default state.
type FFTContext struct {
	Cache     *TransformCache
	Semaphore chan struct{}
	// allocFactory returns a fresh *BumpAllocator sized for the given word
	// length. The default factory uses the package-level bump-allocator
	// pool, exactly like the global path. A field is kept (rather than a
	// hard-coded call) so future contexts can pin private pools.
	allocFactory func(wordLen int) *BumpAllocator
	// allocRelease returns the allocator to whatever pool produced it.
	allocRelease func(*BumpAllocator)
}

// NewFFTContext constructs a fresh FFTContext from the given options.
//
// The returned context owns its own *TransformCache and concurrency
// semaphore — operations driven through it (MulWithContext, SqrWithContext,
// MulToWithContext, SqrToWithContext) will not share cache state nor
// goroutine slots with the package-level default state.
func NewFFTContext(opts FFTContextOptions) *FFTContext {
	cacheCfg := opts.CacheConfig
	// Detect zero-value config (all fields zero) and substitute defaults.
	if cacheCfg.MaxEntries == 0 && cacheCfg.MinBitLen == 0 && !cacheCfg.Enabled {
		cacheCfg = DefaultTransformCacheConfig()
	}
	if opts.DisableCache {
		cacheCfg.Enabled = false
	}

	semSize := opts.SemaphoreSize
	if semSize <= 0 {
		semSize = runtime.NumCPU()
	}

	return &FFTContext{
		Cache:        NewTransformCache(cacheCfg),
		Semaphore:    make(chan struct{}, semSize),
		allocFactory: func(wordLen int) *BumpAllocator { return AcquireBumpAllocator(EstimateBumpCapacity(wordLen)) },
		allocRelease: ReleaseBumpAllocator,
	}
}

// defaultContext is the lazily-initialized package-level context that backs
// the historical Mul/Sqr/MulTo/SqrTo entry points. It mirrors the global
// transform cache and global concurrency semaphore so behavior is unchanged.
//
// We don't use sync.Once here: the underlying cache and semaphore have their
// own lazy init (transformCacheOnce, concurrencyOnce); defaultContext just
// captures references to those existing singletons. Construction is cheap,
// idempotent, and free of side effects on the hot path.
//
//nolint:unused // reserve migration FFTContext exclusif (ADR-0004 B1, backlog won't-fix)
var defaultContext = &FFTContext{
	// Cache and Semaphore are lazily resolved on first use to avoid
	// triggering the package-level sync.Once initializers at import time.
	allocFactory: func(wordLen int) *BumpAllocator { return AcquireBumpAllocator(EstimateBumpCapacity(wordLen)) },
	allocRelease: ReleaseBumpAllocator,
}

// resolved returns the context with Cache/Semaphore lazily filled in from the
// package-level singletons when the context was the default.
func (ctx *FFTContext) resolved() *FFTContext {
	if ctx.Cache == nil {
		ctx.Cache = GetTransformCache()
	}
	if ctx.Semaphore == nil {
		ctx.Semaphore = getSemaphore()
	}
	return ctx
}

// ─────────────────────────────────────────────────────────────────────────────
// Context-aware entry points
// ─────────────────────────────────────────────────────────────────────────────

// MulWithContext is a context-scoped variant of Mul. It uses ctx's transform
// cache and concurrency semaphore. The result is allocated as a new *big.Int.
//
// If ctx is nil, the package default context is used (i.e. behavior
// equivalent to Mul).
func MulWithContext(ctx *FFTContext, x, y *big.Int) (res *big.Int, err error) {
	if ctx == nil {
		return Mul(x, y)
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in bigfft.MulWithContext: %v\nStack: %s", r, debug.Stack())
		}
	}()
	xwords := len(x.Bits())
	ywords := len(y.Bits())
	if t := getFFTThreshold(); xwords > t && ywords > t {
		return mulFFTCtx(ctx.resolved(), x, y)
	}
	return new(big.Int).Mul(x, y), nil
}

// MulToWithContext is a context-scoped variant of MulTo.
//
// If ctx is nil, the package default context is used.
func MulToWithContext(ctx *FFTContext, z, x, y *big.Int) (res *big.Int, err error) {
	if ctx == nil {
		return MulTo(z, x, y)
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in bigfft.MulToWithContext: %v\nStack: %s", r, debug.Stack())
		}
	}()
	xwords := len(x.Bits())
	ywords := len(y.Bits())
	if t := getFFTThreshold(); xwords > t && ywords > t {
		var xb, yb nat = x.Bits(), y.Bits()
		zb, err := fftmulToCtx(ctx.resolved(), z.Bits(), xb, yb)
		if err != nil {
			return nil, err
		}
		z.SetBits(zb)
		if x.Sign()*y.Sign() < 0 {
			z.Neg(z)
		}
		return z, nil
	}
	return z.Mul(x, y), nil
}

// SqrWithContext is a context-scoped variant of Sqr.
//
// If ctx is nil, the package default context is used.
func SqrWithContext(ctx *FFTContext, x *big.Int) (res *big.Int, err error) {
	if ctx == nil {
		return Sqr(x)
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in bigfft.SqrWithContext: %v\nStack: %s", r, debug.Stack())
		}
	}()
	xwords := len(x.Bits())
	if xwords > getFFTThreshold() {
		return sqrFFTCtx(ctx.resolved(), x)
	}
	return new(big.Int).Mul(x, x), nil
}

// SqrToWithContext is a context-scoped variant of SqrTo.
//
// If ctx is nil, the package default context is used.
func SqrToWithContext(ctx *FFTContext, z, x *big.Int) (res *big.Int, err error) {
	if ctx == nil {
		return SqrTo(z, x)
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in bigfft.SqrToWithContext: %v\nStack: %s", r, debug.Stack())
		}
	}()
	xwords := len(x.Bits())
	if xwords > getFFTThreshold() {
		var xb nat = x.Bits()
		zb, err := fftsqrToCtx(ctx.resolved(), z.Bits(), xb)
		if err != nil {
			return nil, err
		}
		z.SetBits(zb)
		return z, nil
	}
	return z.Mul(x, x), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal context-aware FFT pipeline
//
// These mirror mulFFT/sqrFFT/fftmulTo/fftsqrTo from fft.go and fft_core.go,
// but consult ctx.Cache instead of the package-level cache and route the
// fourier recursion through the context's semaphore.
// ─────────────────────────────────────────────────────────────────────────────

func mulFFTCtx(ctx *FFTContext, x, y *big.Int) (*big.Int, error) {
	var xb, yb nat = x.Bits(), y.Bits()
	zb, err := fftmulToCtx(ctx, nil, xb, yb)
	if err != nil {
		return nil, err
	}
	z := new(big.Int)
	z.SetBits(zb)
	if x.Sign()*y.Sign() < 0 {
		z.Neg(z)
	}
	return z, nil
}

func sqrFFTCtx(ctx *FFTContext, x *big.Int) (*big.Int, error) {
	var xb nat = x.Bits()
	zb, err := fftsqrToCtx(ctx, nil, xb)
	if err != nil {
		return nil, err
	}
	z := new(big.Int)
	z.SetBits(zb)
	return z, nil
}

func fftmulToCtx(ctx *FFTContext, dst, x, y nat) (nat, error) {
	k, m := fftSize(x, y)

	wordLen := len(x) + len(y)
	ba := ctx.allocFactory(wordLen)
	defer ctx.allocRelease(ba)

	xp := polyFromNat(x, k, m)
	yp := polyFromNat(y, k, m)

	rp, err := mulCachedWithBumpCtx(ctx, &xp, &yp, ba)
	if err != nil {
		return nil, err
	}
	result := rp.IntTo(dst)
	rp.Release()
	return result, nil
}

func fftsqrToCtx(ctx *FFTContext, dst, x nat) (nat, error) {
	k, m := fftSizeSqr(x)

	wordLen := 2 * len(x)
	ba := ctx.allocFactory(wordLen)
	defer ctx.allocRelease(ba)

	xp := polyFromNat(x, k, m)

	rp, err := sqrCachedWithBumpCtx(ctx, &xp, ba)
	if err != nil {
		return nil, err
	}
	rp.M = m
	result := rp.IntTo(dst)
	rp.Release()
	return result, nil
}

// transformCachedWithBumpCtx is the context-aware twin of
// (*Poly).TransformCachedWithBump. It consults ctx.Cache (not the global
// cache) and routes fourier through the context semaphore.
func transformCachedWithBumpCtx(ctx *FFTContext, p *Poly, n int, ba *BumpAllocator) (PolValues, error) {
	cache := ctx.Cache

	if cache == nil {
		return transformWithBumpCtx(ctx, p, n, ba)
	}
	enabled, minBitLen := cache.cacheGate()
	if !enabled || polyBitLen(p) < minBitLen {
		return transformWithBumpCtx(ctx, p, n, ba)
	}

	key := computePolyKey(p, p.K, n)

	if cached, found := cache.getByKey(key); found {
		return cached, nil
	}

	pv, err := transformWithBumpCtx(ctx, p, n, ba)
	if err != nil {
		return PolValues{}, err
	}
	cache.putByKey(key, pv)
	return pv, nil
}

// transformWithBumpCtx mirrors (*Poly).transform with the bump allocator
// path, but pushes fourier recursion through ctx.Semaphore so context
// isolation is preserved.
func transformWithBumpCtx(ctx *FFTContext, p *Poly, n int, ba *BumpAllocator) (PolValues, error) {
	k := p.K
	K := 1 << k
	wordCount := (n + 1) * K

	input, _, cleanup := ba.AllocFermatSlice(K, n)
	defer cleanup()

	valbits := acquireWordSliceUnsafe(wordCount)
	values := acquireFermatSlice(K)

	for i := 0; i < K; i++ {
		if i < len(p.A) {
			copy(input[i], p.A[i])
		}
		values[i] = fermat(valbits[i*(n+1) : (i+1)*(n+1)])
	}

	if err := fourierWithBumpCtx(ctx, values, input, false, n, k, ba); err != nil {
		return PolValues{}, err
	}

	return PolValues{
		K:             k,
		N:             n,
		Values:        values,
		pooledBacking: valbits,
		pooledValues:  true,
	}, nil
}

// invTransformWithBumpCtx mirrors PolValues.invTransform for the bump
// allocator path, routed through ctx.Semaphore.
func invTransformWithBumpCtx(ctx *FFTContext, v *PolValues, ba *BumpAllocator) (Poly, error) {
	k, n := v.K, v.N
	K := 1 << k
	wordCount := (n + 1) * K

	pbits := acquireWordSliceUnsafe(wordCount)
	p := acquireFermatSlice(K)
	defer releaseFermatSlice(p)
	for i := 0; i < K; i++ {
		p[i] = fermat(pbits[i*(n+1) : (i+1)*(n+1)])
	}

	if err := fourierWithBumpCtx(ctx, p, v.Values, true, n, k, ba); err != nil {
		return Poly{}, err
	}

	u, cleanup := ba.AllocFermatTemp(n)
	defer cleanup()

	a := acquireNatSlice(K)
	for i := 0; i < K; i++ {
		// k is the FFT level (small: typically < 32). #nosec G115
		u.Shift(p[i], -int(k))
		copy(p[i], u)
		a[i] = nat(p[i])
	}

	return Poly{
		K:             k,
		M:             0,
		A:             a,
		pooledBacking: pbits,
		pooledA:       true,
	}, nil
}

func mulCachedWithBumpCtx(ctx *FFTContext, p, q *Poly, ba *BumpAllocator) (Poly, error) {
	n := valueSize(p.K, p.M, 2)

	pv, err := transformCachedWithBumpCtx(ctx, p, n, ba)
	if err != nil {
		return Poly{}, err
	}
	defer pv.Release()

	qv, err := transformCachedWithBumpCtx(ctx, q, n, ba)
	if err != nil {
		return Poly{}, err
	}
	defer qv.Release()

	rv, err := pv.MulWithBump(&qv, ba)
	if err != nil {
		return Poly{}, err
	}
	defer rv.Release()

	r, err := invTransformWithBumpCtx(ctx, &rv, ba)
	if err != nil {
		return Poly{}, err
	}
	r.M = p.M
	return r, nil
}

func sqrCachedWithBumpCtx(ctx *FFTContext, p *Poly, ba *BumpAllocator) (Poly, error) {
	n := valueSize(p.K, p.M, 2)

	pv, err := transformCachedWithBumpCtx(ctx, p, n, ba)
	if err != nil {
		return Poly{}, err
	}
	defer pv.Release()

	rv, err := pv.SqrWithBump(ba)
	if err != nil {
		return Poly{}, err
	}
	defer rv.Release()

	r, err := invTransformWithBumpCtx(ctx, &rv, ba)
	if err != nil {
		return Poly{}, err
	}
	r.M = p.M
	return r, nil
}
