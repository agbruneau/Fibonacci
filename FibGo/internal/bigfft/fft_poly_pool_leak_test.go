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

// badFermatSliceAllocator is a tempAllocator test double whose
// allocFermatSlice returns a slice with element 0 short by one word,
// deterministically tripping the "FFT recursion validation failed" error
// inside fourier/fourierRecursive without touching any package globals.
type badFermatSliceAllocator struct{}

func (badFermatSliceAllocator) allocFermatTemp(n int) (buf fermat, cleanup func()) {
	f := acquireFermat(n + 1)
	return f, func() { releaseFermat(f) }
}

// Like poolAllocator.allocFermatSlice, the buffers stay in locals so the
// cleanup closure captures them by value: the allocation counts asserted
// below are measured on that shape.
func (badFermatSliceAllocator) allocFermatSlice(count, n int) (fermats []fermat, bits []big.Word, cleanup func()) {
	wordCount := count * (n + 1)
	words := acquireWordSlice(wordCount)
	bufs := acquireFermatSlice(count)
	for i := 0; i < count; i++ {
		bufs[i] = fermat(words[i*(n+1) : (i+1)*(n+1)])
	}
	// Corrupt element 0 so len(input[0]) != n+1, tripping the recursion's
	// validation check.
	bufs[0] = bufs[0][:n]
	return bufs, words, func() {
		releaseWordSlice(words)
		releaseFermatSlice(bufs)
	}
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
	if _, err := v.invTransform(defaultPoolAllocator); err == nil {
		t.Fatal("expected validation error from corrupted Values[0]")
	}

	avg := testing.AllocsPerRun(20, func() {
		v := makeCorruptValues()
		if _, err := v.invTransform(defaultPoolAllocator); err == nil {
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

// TestPoolAllocatorSliceAllocsSteadyState pins the steady-state allocation
// count of the PRODUCTION allocator, poolAllocator.allocFermatSlice — the two
// tests above measure a test double and invTransform respectively, so neither
// constrains it.
//
// What it guards: allocFermatSlice keeps its buffers in locals and returns
// them, so the cleanup closure captures the two slice headers by value and
// only the closure itself escapes. Assigning to the named results instead
// makes each captured header escape on its own, adding 2 allocs per call.
// The ceiling below sits between the two, with one alloc of headroom so
// unrelated pool-internal jitter does not turn this into a flaky gate.
//
// Not marked t.Parallel(): testing.AllocsPerRun panics in parallel tests.
func TestPoolAllocatorSliceAllocsSteadyState(t *testing.T) {
	const (
		count = 8
		n     = 4
	)
	p := &poolAllocator{}

	// Warm the pools so the measured window sees steady state, not the
	// one-time New() calls.
	_, _, cleanup := p.allocFermatSlice(count, n)
	cleanup()

	avg := testing.AllocsPerRun(20, func() {
		_, _, cleanup := p.allocFermatSlice(count, n)
		cleanup()
	})

	// Measured on these exact inputs: 3 allocs/run as written, 5 when the
	// buffers are assigned to the named results. The ceiling carries one
	// allocation of slack for escape-analysis differences across toolchains
	// and architectures, so it pins the +2 regression above and would miss
	// a +1 one.
	const maxAllocsPerRun = 4
	if avg > maxAllocsPerRun {
		t.Fatalf("poolAllocator.allocFermatSlice allocates more than its steady state: "+
			"avg allocs/run = %.1f (want <= %d); the buffers must stay in locals "+
			"so the cleanup closure captures them by value", avg, maxAllocsPerRun)
	}
}
