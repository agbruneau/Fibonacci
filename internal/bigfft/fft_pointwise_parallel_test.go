package bigfft

import (
	"math/big"
	"math/rand"
	"testing"
)

// makePointwiseOperand builds a PolValues with K random coefficients of
// n+1 words each (top word zero, as produced by a real transform), large
// enough that K*(n+1) reaches pointwiseMinParallelWords and the parallel
// path engages.
func makePointwiseOperand(t *testing.T, k uint, n int, seed int64) PolValues {
	t.Helper()
	K := 1 << k
	if K*(n+1) < pointwiseMinParallelWords {
		t.Fatalf("test operand too small to engage the parallel path: K*(n+1)=%d < %d",
			K*(n+1), pointwiseMinParallelWords)
	}
	rnd := rand.New(rand.NewSource(seed))
	backing := make([]big.Word, K*(n+1))
	values := make([]fermat, K)
	for i := 0; i < K; i++ {
		values[i] = fermat(backing[i*(n+1) : (i+1)*(n+1)])
		for j := 0; j < n; j++ { // top word stays zero
			values[i][j] = big.Word(rnd.Uint64())
		}
	}
	return PolValues{K: k, N: n, Values: values}
}

// TestPointwiseParallelMatchesSequential pins the parallel pointwise
// dispatch (runPointwise) to the sequential per-index reference: same
// inputs, bit-identical products and squares. Guards the disjoint-write
// contract and the per-worker scratch isolation.
func TestPointwiseParallelMatchesSequential(t *testing.T) {
	t.Parallel()

	const k, n = 8, 255 // K=256, K*(n+1) = 65536 = pointwiseMinParallelWords
	p := makePointwiseOperand(t, k, n, 1)
	q := makePointwiseOperand(t, k, n, 2)

	got, err := p.Mul(&q)
	if err != nil {
		t.Fatalf("Mul failed: %v", err)
	}
	defer got.Release()

	gotSqr, err := p.Sqr()
	if err != nil {
		t.Fatalf("Sqr failed: %v", err)
	}
	defer gotSqr.Release()

	buf := make(fermat, 8*n)
	for i := range p.Values {
		want := make(fermat, n+1)
		copy(want, buf.Mul(p.Values[i], q.Values[i]))
		for j := range want {
			if got.Values[i][j] != want[j] {
				t.Fatalf("Mul mismatch at coefficient %d word %d", i, j)
			}
		}
		copy(want, buf.Sqr(p.Values[i]))
		for j := range want {
			if gotSqr.Values[i][j] != want[j] {
				t.Fatalf("Sqr mismatch at coefficient %d word %d", i, j)
			}
		}
	}
}

// TestPointwiseWorkerPanicPropagates verifies that a panic raised inside a
// parallel pointwise worker is re-panicked on the CALLING goroutine, so the
// public entry points' recover policy (ADR-0002) applies exactly as in the
// sequential path instead of crashing the process from a bare goroutine.
func TestPointwiseWorkerPanicPropagates(t *testing.T) {
	t.Parallel()

	const k, n = 8, 255
	p := makePointwiseOperand(t, k, n, 3)
	q := makePointwiseOperand(t, k, n, 4)
	// Malform a coefficient in the LAST chunk so a spawned worker (not the
	// caller's chunk 0) hits the fermat.Mul size-mismatch panic.
	last := len(q.Values) - 1
	q.Values[last] = q.Values[last][:n] // len n instead of n+1

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected the worker panic to propagate to the caller")
		}
		if r != "Mul: len(x) != len(y)" {
			t.Fatalf("unexpected panic value: %v", r)
		}
	}()
	_, _ = p.Mul(&q)
	t.Fatal("unreachable: Mul must panic on malformed operand")
}
