package bigfft

// fourier performs an unnormalized Fourier transform
// of src, a length 1<<k vector of numbers modulo b^n+1
// where b = 1<<_W.
func fourier(dst, src []fermat, backward bool, n int, k uint) error {
	tmp := acquireFermat(n + 1)
	tmp2 := acquireFermat(n + 1)
	defer releaseFermat(tmp)
	defer releaseFermat(tmp2)

	return fourierRecursive(dst, src, backward, n, k, k, tmp, tmp2)
}

// fourierWithBump performs the Fourier transform with its two top-level
// scratch buffers taken from a bump allocator, giving better cache locality
// than the pooled-buffer path in fourier() (which acquires tmp/tmp2 via
// acquireFermat/releaseFermat).
func fourierWithBump(dst, src []fermat, backward bool, n int, k uint, ba *BumpAllocator) error {
	// The bump allocator supplies the two top-level scratch buffers. It does
	// NOT reach further down: the recursion's parallel branch must use the
	// pool (BumpAllocator is not thread-safe) and its sequential branch reuses
	// these same two buffers, which is why the recursion no longer takes an
	// allocator at all (audit L-06).
	tmp := ba.allocFermat(n)
	tmp2 := ba.allocFermat(n)

	return fourierRecursiveUnified(dst, src, backward, n, k, k, 0, tmp, tmp2)
}

func fftmul(x, y nat) (nat, error) {
	return fftmulTo(nil, x, y)
}

// fftmulTo performs FFT multiplication of x and y, reusing dst as the
// destination buffer if it has sufficient capacity. This reduces allocations
// in iterative multiplication scenarios.
//
// Uses a bump allocator for temporary allocations to minimize GC pressure
// and improve cache locality during the FFT computation.
//
// Transform caching: this is one of the two places the global TransformCache is
// consulted (via MulCachedWithBump; the other is fftsqrTo). A hit requires the
// cache to be Enabled, the operand's polyBitLen to reach MinBitLen, and the
// exact coefficient values to have been transformed before under the same
// (K, n) shape — see TransformCachedWithBump in fft_cache.go.
//
// It is NOT on the Fast Doubling FFT path: executeDoublingStepFFT
// (internal/fibonacci/fft.go) transforms its operands with TransformWithBump
// and never consults the cache. Production reaches here through
// bigfft.Mul/MulTo, i.e. the matrix-exponentiation calculator (which reaches
// smartMultiply, defined in fibonacci/fft.go, from matrix_ops.go) and
// calibration/microbench.go.
//
// The repo records no measurement of the cache's effect, and carries no
// benchmark that produces one: BenchmarkCacheImpact drives a
// FastDoublingCalculator, which by the paragraph above never reaches this
// code, so it reports a 0% hit rate whatever the cache is configured to do.
func fftmulTo(dst, x, y nat) (nat, error) {
	k, m := fftSize(x, y)

	// Estimate and acquire bump allocator for temporary allocations
	wordLen := len(x) + len(y)
	ba := AcquireBumpAllocator(EstimateBumpCapacity(wordLen))
	defer ReleaseBumpAllocator(ba)

	xp := polyFromNat(x, k, m)
	yp := polyFromNat(y, k, m)

	// Use cached multiplication when cache is enabled
	rp, err := xp.MulCachedWithBump(&yp, ba)
	if err != nil {
		return nil, err
	}
	result := rp.IntTo(dst)
	// rp owns pooled buffers — release them now that result has been copied.
	rp.Release()
	return result, nil
}

func fftsqr(x nat) (nat, error) {
	return fftsqrTo(nil, x)
}

// fftsqrTo performs FFT squaring of x, reusing dst as the destination buffer
// if it has sufficient capacity. This is optimized compared to fftmulTo
// because we only need to transform x once.
//
// Uses a bump allocator for temporary allocations to minimize GC pressure
// and improve cache locality during the FFT computation.
//
// Transform caching: same gating and same reachability caveat as fftmulTo —
// the cache is consulted here (via SqrCachedWithBump) but not on the Fast
// Doubling FFT path, and the repo records no measurement of its effect.
func fftsqrTo(dst, x nat) (nat, error) {
	k, m := fftSizeSqr(x)

	// Estimate and acquire bump allocator for temporary allocations
	wordLen := 2 * len(x)
	ba := AcquireBumpAllocator(EstimateBumpCapacity(wordLen))
	defer ReleaseBumpAllocator(ba)

	xp := polyFromNat(x, k, m)

	// Use cached squaring when cache is enabled
	rp, err := xp.SqrCachedWithBump(ba)
	if err != nil {
		return nil, err
	}
	rp.M = m
	result := rp.IntTo(dst)
	rp.Release()
	return result, nil
}
