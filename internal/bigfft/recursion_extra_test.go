package bigfft

import (
	"math/big"
	"strings"
	"sync/atomic"
	"testing"
)

// newFermatVec builds count zeroed fermat elements of n+1 words each.
func newFermatVec(count, n int) []fermat {
	v := make([]fermat, count)
	for i := range v {
		v[i] = make(fermat, n+1)
	}
	return v
}

// fillFermatVec writes deterministic pseudo-random words into every element,
// leaving the top word zero so each element is a valid residue mod 2^(n*_W)+1.
func fillFermatVec(v []fermat, seed uint64) {
	for i := range v {
		for j := 0; j < len(v[i])-1; j++ {
			v[i][j] = big.Word(seed + uint64(i+1)*0x9E3779B97F4A7C15 + uint64(j)*0xBF58476D1CE4E5B9)
		}
	}
}

// cloneFermatVec deep-copies a fermat vector.
func cloneFermatVec(v []fermat) []fermat {
	out := make([]fermat, len(v))
	for i := range v {
		out[i] = append(fermat(nil), v[i]...)
	}
	return out
}

// runReconstructionCase compares executeReconstruction against a sequential
// reference computed with the same primitives. A regression in the chunk math
// (e.g. relative instead of absolute indices in the twiddle shift) or a race
// on the scratch buffers would diverge from the reference.
func runReconstructionCase(t *testing.T, count, n int, exhaustSemaphore bool) {
	t.Helper()
	if count*(n+1) < reconstructionMinParallelWords {
		t.Fatalf("test setup must reach the parallel gate: %d < %d", count*(n+1), reconstructionMinParallelWords)
	}
	if exhaustSemaphore {
		sem := getSemaphore()
		filled := 0
		for filling := true; filling; {
			select {
			case sem <- struct{}{}:
				filled++
			default:
				filling = false
			}
		}
		defer func() {
			for i := 0; i < filled; i++ {
				<-sem
			}
		}()
	}
	// Odd shift so ShiftHalf exercises its half-bit path as in production.
	const shift = 511

	dst1 := newFermatVec(count, n)
	dst2 := newFermatVec(count, n)
	fillFermatVec(dst1, 1)
	fillFermatVec(dst2, 2)
	ref1 := cloneFermatVec(dst1)
	ref2 := cloneFermatVec(dst2)

	// Sequential reference using the same fermat primitives as the body.
	t1 := make(fermat, n+1)
	t2 := make(fermat, n+1)
	for i := 0; i < count; i++ {
		t1.ShiftHalf(ref2[i], i*shift, t2)
		ref2[i].Sub(ref1[i], t1)
		ref1[i].Add(ref1[i], t1)
	}

	tmp := make(fermat, n+1)
	tmp2 := make(fermat, n+1)
	if err := executeReconstruction(dst1, dst2, shift, tmp, tmp2); err != nil {
		t.Fatalf("executeReconstruction failed: %v", err)
	}

	for i := 0; i < count; i++ {
		for j := range dst1[i] {
			if dst1[i][j] != ref1[i][j] {
				t.Fatalf("dst1[%d][%d] = %#x, want %#x", i, j, dst1[i][j], ref1[i][j])
			}
			if dst2[i][j] != ref2[i][j] {
				t.Fatalf("dst2[%d][%d] = %#x, want %#x", i, j, dst2[i][j], ref2[i][j])
			}
		}
	}
}

// TestExecuteReconstructionMatchesSequentialReference drives the chunked
// butterfly across its dispatch variants: wide span (worker dispatch), narrow
// span (worker count clamps to len(dst1)), and exhausted semaphore (every
// chunk falls back to the calling goroutine).
//
// Not parallel: the parallel path draws tokens from the package-level FFT
// semaphore; running serially keeps token availability deterministic.
func TestExecuteReconstructionMatchesSequentialReference(t *testing.T) {
	t.Run("parallel span", func(t *testing.T) {
		runReconstructionCase(t, 128, 511, false)
	})
	t.Run("narrow span clamps workers", func(t *testing.T) {
		runReconstructionCase(t, 8, 8192, false)
	})
	t.Run("semaphore exhausted falls back to caller", func(t *testing.T) {
		runReconstructionCase(t, 128, 511, true)
	})
}

// TestExecuteReconstructionPanicPropagates plants one malformed element in
// the parallel span: the panic raised inside the butterfly body must surface
// to the caller (ADR-0002 re-propagation), wherever the chunk runs.
//
// Not parallel: relies on the package-level FFT semaphore for the parallel
// dispatch under test.
func TestExecuteReconstructionPanicPropagates(t *testing.T) {
	const (
		K = 128
		n = 511
	)
	dst1 := newFermatVec(K, n)
	dst2 := newFermatVec(K, n)
	dst2[K-1] = make(fermat, 5) // wrong length: Shift panics inside the body

	tmp := make(fermat, n+1)
	tmp2 := make(fermat, n+1)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic from malformed element to propagate to the caller")
		}
	}()
	_ = executeReconstruction(dst1, dst2, 511, tmp, tmp2)
	t.Fatal("executeReconstruction returned normally despite a malformed element")
}

// TestFourierRecursiveParallelErrorPropagation checks that a validation
// failure inside either half of the parallel recursion is reported to the
// caller. Element 1 belongs to the second (async) half, element 2 to the
// deeper second half of the sync branch; the error must propagate either way
// (and also through the sequential fallback if no token were available).
//
// Not parallel: needs free tokens on the package-level FFT semaphore so the
// parallel branch is taken deterministically.
func TestFourierRecursiveParallelErrorPropagation(t *testing.T) {
	const n = 8

	if GetParallelFFTRecursionThreshold() > 4 || GetMaxParallelFFTDepth() == 0 {
		t.Skip("parallelism knobs exclude the parallel branch at this size")
	}

	for _, tc := range []struct {
		name   string
		k      uint
		badIdx int
	}{
		{"async half reports error", 4, 1},
		{"sync half reports error", 4, 2},
		// k=3 stays below the parallel threshold: the same malformed element
		// must propagate through the purely sequential recursion too.
		{"sequential path reports error", 3, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			K := 1 << tc.k
			src := newFermatVec(K, n)
			dst := newFermatVec(K, n)
			src[tc.badIdx] = make(fermat, 3) // breaks the n+1 contract deeper in

			err := fourierRecursive(dst, src, false, n, tc.k, tc.k, 0, make(fermat, n+1), make(fermat, n+1))
			if err == nil {
				t.Fatal("expected validation error to propagate from the recursion")
			}
			if !strings.Contains(err.Error(), "validation failed") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// TestFourierRecursiveCtxParallelErrorPropagation mirrors the test above for
// the context-aware recursion; the context owns its semaphore, so the
// parallel branch is deterministic without touching package globals.
func TestFourierRecursiveCtxParallelErrorPropagation(t *testing.T) {
	t.Parallel()
	const n = 8

	if GetParallelFFTRecursionThreshold() > 4 || GetMaxParallelFFTDepth() == 0 {
		t.Skip("parallelism knobs exclude the parallel branch at this size")
	}

	for _, tc := range []struct {
		name   string
		k      uint
		badIdx int
	}{
		{"async half reports error", 4, 1},
		{"sync half reports error", 4, 2},
		{"sequential path reports error", 3, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := NewFFTContext(FFTContextOptions{SemaphoreSize: 2, DisableCache: true})
			K := 1 << tc.k
			src := newFermatVec(K, n)
			dst := newFermatVec(K, n)
			src[tc.badIdx] = make(fermat, 3)

			err := fourierRecursiveCtx(ctx, dst, src, false, n, tc.k, tc.k, 0,
				make(fermat, n+1), make(fermat, n+1), GetPoolAllocator())
			if err == nil {
				t.Fatal("expected validation error to propagate from the ctx recursion")
			}
			if !strings.Contains(err.Error(), "validation failed") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// TestFourierRecursiveSizeZeroIsIdentity pins the size-0 base case of both
// recursion variants: a 1-point FFT is the identity copy.
func TestFourierRecursiveSizeZeroIsIdentity(t *testing.T) {
	t.Parallel()
	const n = 4
	src := newFermatVec(1, n)
	fillFermatVec(src, 3)

	dst := newFermatVec(1, n)
	if err := fourierRecursive(dst, src, false, n, 0, 0, 0, make(fermat, n+1), make(fermat, n+1)); err != nil {
		t.Fatalf("fourierRecursive size 0 failed: %v", err)
	}
	assertFermatEqual(t, dst[0], src[0])

	ctx := NewFFTContext(FFTContextOptions{SemaphoreSize: 1, DisableCache: true})
	dstCtx := newFermatVec(1, n)
	if err := fourierRecursiveCtx(ctx, dstCtx, src, false, n, 0, 0, 0,
		make(fermat, n+1), make(fermat, n+1), GetPoolAllocator()); err != nil {
		t.Fatalf("fourierRecursiveCtx size 0 failed: %v", err)
	}
	assertFermatEqual(t, dstCtx[0], src[0])
}

// TestRunPointwiseVisitsEveryIndexExactlyOnce drives the parallel pointwise
// dispatcher with K far below NumCPU (worker clamp) at exactly the parallel
// gate: every index must be processed exactly once whatever the chunking, and
// every invocation must receive the documented 8n-word scratch.
func TestRunPointwiseVisitsEveryIndexExactlyOnce(t *testing.T) {
	t.Parallel()
	const (
		K = 4
		n = 16383 // K*(n+1) == pointwiseMinParallelWords
	)
	visits := make([]atomic.Int32, K)
	runPointwise(K, n, GetPoolAllocator(), func(i int, buf fermat) {
		if len(buf) != 8*n+1 {
			panic("pointwise scratch buffer has wrong length")
		}
		visits[i].Add(1)
	})
	for i := range visits {
		if got := visits[i].Load(); got != 1 {
			t.Fatalf("index %d visited %d times, want exactly 1", i, got)
		}
	}
}

// TestFourierWithStateMatchesPoolPath runs the same forward transform through
// the explicit-state variant and the pooled variant; outputs must be
// bit-identical (the pre-allocated state must not alter the transform).
func TestFourierWithStateMatchesPoolPath(t *testing.T) {
	t.Parallel()
	const n = 8
	k := uint(2)
	K := 1 << k

	src := newFermatVec(K, n)
	fillFermatVec(src, 7)
	dstState := newFermatVec(K, n)
	dstPool := newFermatVec(K, n)

	state := acquireFFTState(n, k)
	defer releaseFFTState(state)
	if err := fourierWithState(dstState, src, false, n, k, state); err != nil {
		t.Fatalf("fourierWithState failed: %v", err)
	}
	if err := fourier(dstPool, src, false, n, k); err != nil {
		t.Fatalf("fourier failed: %v", err)
	}
	for i := 0; i < K; i++ {
		for j := range dstState[i] {
			if dstState[i][j] != dstPool[i][j] {
				t.Fatalf("transforms diverge at [%d][%d]", i, j)
			}
		}
	}
}

// TestReleaseFFTStateAntiBloat verifies the A2-05 contract: tmp buffers whose
// capacity exceeds maxPooledFFTTmpCap are detached before the state returns
// to the pool, so one huge FFT cannot pin multi-MB buffers process-wide.
//
// Not parallel: inspects the state object after it re-enters the global pool;
// a concurrently running acquireFFTState could legitimately re-take it.
func TestReleaseFFTStateAntiBloat(t *testing.T) {
	releaseFFTState(nil) // nil must be a safe no-op

	state := &fftState{
		tmp:  make(fermat, maxPooledFFTTmpCap+1),
		tmp2: make(fermat, maxPooledFFTTmpCap+1),
		n:    maxPooledFFTTmpCap,
		k:    1,
	}
	releaseFFTState(state)
	if state.tmp != nil || state.tmp2 != nil {
		t.Fatal("oversized tmp buffers must be detached on release (A2-05)")
	}
}

// TestAcquireFFTStateReturnsZeroedBuffers dirties a state's buffers, releases
// it, and re-acquires a smaller state: whatever object the pool hands back,
// tmp/tmp2 must come out zeroed at the requested length (a reuse path that
// forgot to clear would expose stale words).
func TestAcquireFFTStateReturnsZeroedBuffers(t *testing.T) {
	t.Parallel()
	s1 := acquireFFTState(64, 3)
	for i := range s1.tmp {
		s1.tmp[i] = ^big.Word(0)
	}
	for i := range s1.tmp2 {
		s1.tmp2[i] = ^big.Word(0)
	}
	releaseFFTState(s1)

	s2 := acquireFFTState(32, 2)
	defer releaseFFTState(s2)
	if len(s2.tmp) != 33 || len(s2.tmp2) != 33 {
		t.Fatalf("acquired buffers have lengths %d/%d, want 33/33", len(s2.tmp), len(s2.tmp2))
	}
	for i := range s2.tmp {
		if s2.tmp[i] != 0 || s2.tmp2[i] != 0 {
			t.Fatalf("buffers not cleared at word %d: %#x/%#x", i, s2.tmp[i], s2.tmp2[i])
		}
	}
}
