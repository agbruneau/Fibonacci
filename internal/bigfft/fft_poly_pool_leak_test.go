//go:build !race

// These tests assert exact pooled-allocation counts via testing.AllocsPerRun to
// prove the error paths release their buffers (FFT-08). The race detector adds
// its own allocations, so the counts are only meaningful in a non-race build;
// the buffer-release fix itself is exercised by the whole suite regardless.
package bigfft

import (
	"math/big"
	"testing"
)

// badFermatSliceAllocator is a TempAllocator test double whose
// AllocFermatSlice returns a slice with element 0 short by one word,
// deterministically tripping the "FFT recursion validation failed" error
// inside fourier/fourierRecursive without touching any package globals.
type badFermatSliceAllocator struct{}

func (badFermatSliceAllocator) AllocFermatTemp(n int) (fermat, func()) {
	f := acquireFermat(n + 1)
	return f, func() { releaseFermat(f) }
}

func (badFermatSliceAllocator) AllocFermatSlice(K, n int) ([]fermat, []big.Word, func()) {
	wordCount := K * (n + 1)
	bits := acquireWordSlice(wordCount)
	fermats := acquireFermatSlice(K)
	for i := 0; i < K; i++ {
		fermats[i] = fermat(bits[i*(n+1) : (i+1)*(n+1)])
	}
	// Corrupt element 0 so len(input[0]) != n+1, tripping the recursion's
	// validation check.
	fermats[0] = fermats[0][:n]
	cleanup := func() {
		releaseWordSlice(bits)
		releaseFermatSlice(fermats)
	}
	return fermats, bits, cleanup
}

// These tests reproduce audit finding FFT-08 (pool buffers not released on
// transform/invTransform error-return paths) using the same steady-state
// testing.AllocsPerRun technique as TestTransformCacheEvictionRecyclesBacking:
// a leaked pool buffer is never returned to sync.Pool, so every erroring
// call forces a fresh make() from the pool's New() func instead of reusing
// a previously released one, making per-call allocations keep climbing
// instead of settling at a steady-state bound. This avoids asserting
// sync.Pool object identity/reappearance, which this package deliberately
// treats as flaky (see TestReleaseWordSliceResizedReturnsToBucket).
//
// Not marked t.Parallel(): testing.AllocsPerRun panics in parallel tests.

// TestTransformReleasesBuffersOnError covers Poly.transform's error path
// (fft_poly.go, forward transform).
func TestTransformReleasesBuffersOnError(t *testing.T) {
	k := uint(3)
	m := 1
	n := valueSize(k, m, 2)
	K := 1 << k

	p := Poly{K: k, M: m}
	p.A = make([]nat, K)
	for i := range p.A {
		p.A[i] = nat{big.Word(i + 1)}
	}

	// Warm up once outside the measured window so any one-time setup cost
	// (e.g. populating the pool's sync.Pool internals) doesn't skew the
	// first measured iteration.
	if _, err := p.transform(n, badFermatSliceAllocator{}); err == nil {
		t.Fatal("expected validation error from corrupted allocator")
	}

	avg := testing.AllocsPerRun(20, func() {
		if _, err := p.transform(n, badFermatSliceAllocator{}); err == nil {
			t.Fatal("expected validation error from corrupted allocator")
		}
	})

	// A leak means valbits/values are never returned to their pools, so
	// acquireWordSliceUnsafe/acquireFermatSlice fall through to their New()
	// funcs on every single call, adding 2 allocs/run (one per buffer) over
	// the steady-state released path. Measured on this test's exact inputs:
	// 9 allocs/run with both buffers released (fix), 11 when leaked. The
	// threshold sits strictly between the two, deterministic across GOMAXPROCS(1)
	// runs (testing.AllocsPerRun's own setting).
	const maxAllocsPerRun = 9
	if avg > maxAllocsPerRun {
		t.Fatalf("transform leaked its valbits/values buffers on the error path (FFT-08): "+
			"avg allocs/run = %.1f (want <= %d)", avg, maxAllocsPerRun)
	}
}

// TestInvTransformReleasesBuffersOnError covers PolValues.invTransform's
// error path (fft_poly.go).
func TestInvTransformReleasesBuffersOnError(t *testing.T) {
	k := uint(3)
	n := valueSize(k, 1, 2)
	K := 1 << k
	wordCount := (n + 1) * K

	makeCorruptValues := func() PolValues {
		values := acquireFermatSlice(K)
		bits := acquireWordSlice(wordCount)
		for i := 0; i < K; i++ {
			values[i] = fermat(bits[i*(n+1) : (i+1)*(n+1)])
		}
		// Corrupt element 0 so the recursion's len(src[0]) check fails.
		values[0] = values[0][:n]
		return PolValues{K: k, N: n, Values: values}
	}

	v := makeCorruptValues()
	if _, err := v.invTransform(GetPoolAllocator()); err == nil {
		t.Fatal("expected validation error from corrupted Values[0]")
	}

	avg := testing.AllocsPerRun(20, func() {
		v := makeCorruptValues()
		if _, err := v.invTransform(GetPoolAllocator()); err == nil {
			t.Fatal("expected validation error from corrupted Values[0]")
		}
	})

	// Measured on this test's exact inputs: 10 allocs/run with pbits
	// released (fix), 11 when leaked (pbits falls through to New() every
	// call instead of being reused).
	const maxAllocsPerRun = 10
	if avg > maxAllocsPerRun {
		t.Fatalf("invTransform leaked its pbits buffer on the error path (FFT-08): "+
			"avg allocs/run = %.1f (want <= %d)", avg, maxAllocsPerRun)
	}
}

// TestInvTransformWithBumpCtxReleasesBuffersOnError mirrors
// TestInvTransformReleasesBuffersOnError for the context-aware bump-allocator
// twin (invTransformWithBumpCtx), covering the ctx code path named in FFT-08.
func TestInvTransformWithBumpCtxReleasesBuffersOnError(t *testing.T) {
	k := uint(3)
	n := valueSize(k, 1, 2)
	K := 1 << k
	wordCount := (n + 1) * K

	ctx := NewFFTContext(FFTContextOptions{})
	ba := AcquireBumpAllocator(EstimateBumpCapacity(wordCount))
	defer ReleaseBumpAllocator(ba)

	makeCorruptValues := func() PolValues {
		values := acquireFermatSlice(K)
		bits := acquireWordSlice(wordCount)
		for i := 0; i < K; i++ {
			values[i] = fermat(bits[i*(n+1) : (i+1)*(n+1)])
		}
		values[0] = values[0][:n]
		return PolValues{K: k, N: n, Values: values}
	}

	v := makeCorruptValues()
	if _, err := invTransformWithBumpCtx(ctx, &v, ba); err == nil {
		t.Fatal("expected validation error from corrupted Values[0]")
	}

	avg := testing.AllocsPerRun(20, func() {
		v := makeCorruptValues()
		if _, err := invTransformWithBumpCtx(ctx, &v, ba); err == nil {
			t.Fatal("expected validation error from corrupted Values[0]")
		}
	})

	// Measured on this test's exact inputs: 8 allocs/run with pbits
	// released (fix), 9 when leaked.
	const maxAllocsPerRun = 8
	if avg > maxAllocsPerRun {
		t.Fatalf("invTransformWithBumpCtx leaked its pbits buffer on the error path (FFT-08): "+
			"avg allocs/run = %.1f (want <= %d)", avg, maxAllocsPerRun)
	}
}
