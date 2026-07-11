package bigfft

import (
	"math/big"
	"strings"
	"sync/atomic"
	"testing"
)

// pinFFTParallelismConfig pins the package-level FFT parallelism knobs to
// values that guarantee the parallel branch is taken (RecursionThreshold <=
// 4, MaxDepth > 0), restoring the prior config via t.Cleanup. Guardian tests
// for ADR-0002 panic re-propagation must exercise the parallel branch
// unconditionally: a t.Skip guarded only by reading the current knobs would
// silently stop testing the invariant if a future tuning ever raised
// RecursionThreshold above 4 (TEST-04).
func pinFFTParallelismConfig(t *testing.T) {
	t.Helper()
	orig := GetFFTParallelismConfig()
	t.Cleanup(func() { SetFFTParallelismConfig(orig) })
	SetFFTParallelismConfig(FFTParallelismConfig{RecursionThreshold: 4, MaxDepth: 3})
}

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

	pinFFTParallelismConfig(t)

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

// TestFourierRecursiveAsyncPanicPropagates plants a malformed dst element at
// an odd index in the second (async) half. That index is the dst[1] of a
// size-1 leaf, which the recursion's top-of-call validation never checks (it
// only validates dst[0]/src[0]), so fermat.Sub panics deep inside the spawned
// goroutine. The panic must surface to the caller (ADR-0002 re-propagation)
// instead of crashing the process from a bare goroutine — the recursion-split
// counterpart of TestExecuteReconstructionPanicPropagates.
//
// Not parallel: relies on free tokens on the package-level FFT semaphore so
// the async branch is taken.
func TestFourierRecursiveAsyncPanicPropagates(t *testing.T) {
	const (
		k = 4
		n = 511
	)
	pinFFTParallelismConfig(t)
	K := 1 << k
	src := newFermatVec(K, n)
	dst := newFermatVec(K, n)
	dst[K/2+1] = make(fermat, 5) // odd index in the async half: dst[1] of a leaf, Sub panics

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic from the async recursion half to propagate to the caller")
		}
	}()
	_ = fourierRecursive(dst, src, false, n, k, k, 0, make(fermat, n+1), make(fermat, n+1))
	t.Fatal("fourierRecursive returned normally despite a malformed async-half element")
}

// TestFourierRecursiveSizeZeroIsIdentity pins the size-0 base case of the
// recursion: a 1-point FFT is the identity copy.
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

// assertSemaphoreFullyReleased drains sem immediately (no retry, no sleep)
// and fails if any token is still outstanding. With wg.Wait() unconditional
// before the re-panic (the fix), the worker's token-release defer runs
// before Wait() returns, which happens-before the panic reaches the caller
// via a real synchronization edge — so every token must already be back
// the instant this is called. With wg.Wait() skipped (the FFT-02 bug), the
// still-running worker has had no such happens-before edge with the caller,
// so this reliably (not just occasionally) observes a missing token.
func assertSemaphoreFullyReleased(t *testing.T, sem chan struct{}) {
	t.Helper()
	capacity := cap(sem)
	filled := 0
	for filled < capacity {
		select {
		case sem <- struct{}{}:
			filled++
		default:
			goto drain
		}
	}
drain:
	for i := 0; i < filled; i++ {
		<-sem
	}
	if filled != capacity {
		t.Fatalf("semaphore short %d/%d tokens right after the panic recovered: wg.Wait() was skipped while a worker still held a token", capacity-filled, capacity)
	}
}

// tokenReleasedBeforePanic runs call, which must panic, then immediately
// asserts every token of sem is available.
func tokenReleasedBeforePanic(t *testing.T, sem chan struct{}, call func()) {
	t.Helper()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected the sync half to panic")
			}
		}()
		call()
	}()

	assertSemaphoreFullyReleased(t, sem)
}

// TestFourierRecursiveUnifiedSyncPanicWaitsForWorker reproduces audit finding
// FFT-02 at the fourierRecursiveUnified site: the sync half (dst1) panics
// while the async half (dst2), given deliberately heavy real work via a large
// n, is still running. The fix must wg.Wait() unconditionally before
// re-panicking, so the worker's semaphore token is always released by the
// time the caller observes the panic.
func TestFourierRecursiveUnifiedSyncPanicWaitsForWorker(t *testing.T) {
	pinFFTParallelismConfig(t)
	const (
		k = 4
		n = 4095 // heavy enough that the async half outlives an early sync panic
	)
	K := 1 << k
	src := newFermatVec(K, n)
	dst := newFermatVec(K, n)
	dst[1] = make(fermat, 5) // dst[1] of the first leaf in the sync half (dst1): panics almost immediately

	tokenReleasedBeforePanic(t, getSemaphore(), func() {
		_ = fourierRecursive(dst, src, false, n, k, k, 0, make(fermat, n+1), make(fermat, n+1))
	})
}

// TestExecuteReconstructionChunkZeroPanicWaitsForWorker reproduces audit
// finding FFT-02 at the executeReconstruction site: chunk 0 (the span run
// synchronously on the calling goroutine) panics while the other chunks,
// dispatched as workers over a deliberately large n, are still running.
func TestExecuteReconstructionChunkZeroPanicWaitsForWorker(t *testing.T) {
	const (
		K = 128
		n = 8191 // heavy enough that worker chunks outlive an early chunk-0 panic
	)
	dst1 := newFermatVec(K, n)
	dst2 := newFermatVec(K, n)
	dst1[0] = make(fermat, 5) // chunk 0 element: Sub panics almost immediately

	tmp := make(fermat, n+1)
	tmp2 := make(fermat, n+1)

	tokenReleasedBeforePanic(t, getSemaphore(), func() {
		_ = executeReconstruction(dst1, dst2, 511, tmp, tmp2)
	})
}

// TestRunPointwiseChunkZeroPanicWaitsForWorker reproduces audit finding
// FFT-02 at the runPointwise site: chunk 0 panics while other chunks,
// dispatched as workers over a deliberately large n, are still running.
func TestRunPointwiseChunkZeroPanicWaitsForWorker(t *testing.T) {
	const k, n = 8, 8191 // heavy enough that worker chunks outlive an early chunk-0 panic
	p := makePointwiseOperand(t, k, n, 5)
	q := makePointwiseOperand(t, k, n, 6)
	q.Values[0] = q.Values[0][:n] // chunk 0 element: fermat.Mul panics almost immediately

	tokenReleasedBeforePanic(t, getSemaphore(), func() {
		_, _ = p.Mul(&q)
	})
}
