package bigfft

// fourier performs an unnormalized Fourier transform
// of src, a length 1<<k vector of numbers modulo b^n+1
// where b = 1<<_W.
func fourier(dst []fermat, src []fermat, backward bool, n int, k uint) error {
	tmp := acquireFermat(n + 1)
	tmp2 := acquireFermat(n + 1)
	defer releaseFermat(tmp)
	defer releaseFermat(tmp2)

	return fourierRecursive(dst, src, backward, n, k, k, 0, tmp, tmp2)
}

// fourierWithBump performs the Fourier transform using a bump allocator for
// temporary buffers, giving better cache locality than the pooled-buffer path
// in fourier() (which acquires tmp/tmp2 via acquireFermat/releaseFermat).
func fourierWithBump(dst []fermat, src []fermat, backward bool, n int, k uint, ba *BumpAllocator) error {
	tmp := ba.AllocFermat(n)
	tmp2 := ba.AllocFermat(n)

	// *BumpAllocator implements TempAllocator directly (see bump.go).
	return fourierRecursiveUnified(dst, src, backward, n, k, k, 0, tmp, tmp2, ba)
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
// Transform caching: When the global TransformCache is enabled, FFT transforms
// are cached and reused for repeated multiplications of the same values,
// providing 15-30% speedup in iterative algorithms like Fibonacci.
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
// Transform caching: When the global TransformCache is enabled, FFT transforms
// are cached and reused for repeated squaring of the same values,
// providing significant speedup in iterative algorithms like Fibonacci.
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
