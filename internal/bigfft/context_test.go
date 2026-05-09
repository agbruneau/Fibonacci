package bigfft

import (
	"math/big"
	"sync"
	"testing"
)

// makeBigInt builds a big.Int with the given word count and a deterministic
// pattern so we can exercise the FFT path (>fftThreshold words).
func makeBigInt(seed, numWords int) *big.Int {
	z := new(big.Int)
	words := make([]big.Word, numWords)
	for i := range words {
		words[i] = big.Word(uint64(seed)*0x9E3779B97F4A7C15 + uint64(i))
	}
	z.SetBits(words)
	return z
}

func TestFFTContext_BasicMul(t *testing.T) {
	t.Parallel()
	ctx := NewFFTContext(FFTContextOptions{})
	if ctx == nil || ctx.Cache == nil || ctx.Semaphore == nil {
		t.Fatal("NewFFTContext returned malformed context")
	}

	x := makeBigInt(1, 2000)
	y := makeBigInt(2, 2000)

	expected := new(big.Int).Mul(x, y)

	got, err := MulWithContext(ctx, x, y)
	if err != nil {
		t.Fatalf("MulWithContext failed: %v", err)
	}
	if got.Cmp(expected) != 0 {
		t.Fatalf("MulWithContext mismatch")
	}
}

func TestFFTContext_BasicSqr(t *testing.T) {
	t.Parallel()
	ctx := NewFFTContext(FFTContextOptions{})

	x := makeBigInt(7, 2000)
	expected := new(big.Int).Mul(x, x)

	got, err := SqrWithContext(ctx, x)
	if err != nil {
		t.Fatalf("SqrWithContext failed: %v", err)
	}
	if got.Cmp(expected) != 0 {
		t.Fatalf("SqrWithContext mismatch")
	}

	// SqrToWithContext
	z := new(big.Int)
	got, err = SqrToWithContext(ctx, z, x)
	if err != nil {
		t.Fatalf("SqrToWithContext failed: %v", err)
	}
	if got.Cmp(expected) != 0 {
		t.Fatalf("SqrToWithContext mismatch")
	}
	if got != z {
		t.Fatalf("SqrToWithContext did not reuse destination pointer")
	}
}

func TestFFTContext_NilFallsBackToDefault(t *testing.T) {
	t.Parallel()
	x := makeBigInt(11, 2000)
	y := makeBigInt(13, 2000)

	expected := new(big.Int).Mul(x, y)

	got, err := MulWithContext(nil, x, y)
	if err != nil {
		t.Fatalf("MulWithContext(nil) failed: %v", err)
	}
	if got.Cmp(expected) != 0 {
		t.Fatalf("MulWithContext(nil) mismatch with reference")
	}
}

// TestFFTContext_IsolationParallel is the critical test for R3.7. It runs two
// independent contexts in parallel, performing many Muls on each, and verifies
// that:
//   - Results are correct (no cross-contamination of internal state).
//   - Each context's cache observes only its own activity.
func TestFFTContext_IsolationParallel(t *testing.T) {
	t.Parallel()

	ctxA := NewFFTContext(FFTContextOptions{SemaphoreSize: 4})
	ctxB := NewFFTContext(FFTContextOptions{SemaphoreSize: 4})

	// Independent operand sets so the two contexts hash to different
	// cache keys; each context should see hits only from its own loop.
	xA := makeBigInt(101, 2000)
	yA := makeBigInt(202, 2000)
	expectedA := new(big.Int).Mul(xA, yA)

	xB := makeBigInt(909, 2000)
	yB := makeBigInt(808, 2000)
	expectedB := new(big.Int).Mul(xB, yB)

	const iters = 8

	var wg sync.WaitGroup
	wg.Add(2)

	errCh := make(chan error, 2)

	go func() {
		defer wg.Done()
		for i := 0; i < iters; i++ {
			res, err := MulWithContext(ctxA, xA, yA)
			if err != nil {
				errCh <- err
				return
			}
			if res.Cmp(expectedA) != 0 {
				errCh <- errMismatch("ctxA")
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < iters; i++ {
			res, err := MulWithContext(ctxB, xB, yB)
			if err != nil {
				errCh <- err
				return
			}
			if res.Cmp(expectedB) != 0 {
				errCh <- errMismatch("ctxB")
				return
			}
		}
	}()

	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("isolation test error: %v", err)
	}

	statsA := ctxA.Cache.Stats()
	statsB := ctxB.Cache.Stats()

	// Each context should have at least one hit (cache was actually used)
	// and the totals should reflect only that context's own operations.
	// A Mul of the SAME (xA, yA) repeated `iters` times should yield
	// (iters * 2 transforms) - 2 unique = (iters-1)*2 hits per context
	// (two transforms per Mul: one for xA and one for yA). We assert a
	// loose lower bound to stay robust against future caching tweaks.
	if statsA.Hits == 0 {
		t.Errorf("ctxA cache hits == 0 (expected > 0): %+v", statsA)
	}
	if statsB.Hits == 0 {
		t.Errorf("ctxB cache hits == 0 (expected > 0): %+v", statsB)
	}

	// Critical isolation invariant: the two contexts must not share state.
	// If they did, ctxA's cache would also see ctxB's misses (and vice
	// versa), inflating their totals. We assert that each context's total
	// access count is bounded by its own iteration count's transform load.
	// Each Mul does 2 forward transforms; cache lookups happen per transform.
	maxAccesses := uint64(iters * 2)
	if statsA.Hits+statsA.Misses > maxAccesses {
		t.Errorf("ctxA observed %d cache accesses, expected <= %d (cross-contamination?)",
			statsA.Hits+statsA.Misses, maxAccesses)
	}
	if statsB.Hits+statsB.Misses > maxAccesses {
		t.Errorf("ctxB observed %d cache accesses, expected <= %d (cross-contamination?)",
			statsB.Hits+statsB.Misses, maxAccesses)
	}
}

func TestFFTContext_DisableCache(t *testing.T) {
	t.Parallel()
	ctx := NewFFTContext(FFTContextOptions{DisableCache: true})

	x := makeBigInt(5, 2000)
	y := makeBigInt(6, 2000)
	expected := new(big.Int).Mul(x, y)

	for i := 0; i < 3; i++ {
		got, err := MulWithContext(ctx, x, y)
		if err != nil {
			t.Fatalf("MulWithContext failed: %v", err)
		}
		if got.Cmp(expected) != 0 {
			t.Fatalf("MulWithContext mismatch")
		}
	}

	stats := ctx.Cache.Stats()
	if stats.Hits != 0 || stats.Misses != 0 {
		t.Errorf("disabled cache should record no activity, got %+v", stats)
	}
}

type errString string

func (e errString) Error() string { return string(e) }

func errMismatch(tag string) error { return errString("result mismatch in " + tag) }
