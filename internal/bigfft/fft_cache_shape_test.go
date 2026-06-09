package bigfft

import (
	"math/big"
	"testing"
)

// TestPutByKeyRejectsMalformedShape is the [A-05] regression test.
//
// putByKey copies each pv.Values[i] into a backing sub-slice of length
// n+1. If a coefficient has len != n+1, `copy` silently truncates (len <
// n+1) or drops trailing data (len > n+1), producing a corrupt cached
// transform that later yields wrong FFT results — with no error surfaced.
//
// The fix rejects any PolValues whose coefficients are not all exactly
// n+1 words: putByKey returns early (drop, never partial-store). A dropped
// Put leaves the cache unchanged, so a subsequent Get is a clean miss —
// callers fall back to recompute, never to corrupt data.
func TestPutByKeyRejectsMalformedShape(t *testing.T) {
	t.Parallel()

	cache := NewTransformCache(TransformCacheConfig{
		MaxEntries: 16,
		MinBitLen:  64,
		Enabled:    true,
	})

	data := make(nat, 64)
	for i := range data {
		data[i] = big.Word(i + 1)
	}

	// N=100 => each coefficient must be len 101. These are len 3: malformed.
	malformed := PolValues{
		K:      4,
		N:      100,
		Values: []fermat{{1, 2, 3}, {4, 5, 6}},
	}
	cache.Put(data, malformed)

	if _, found := cache.Get(data, 4, 100); found {
		t.Fatal("[A-05] malformed-shape PolValues was cached; putByKey must " +
			"drop it so Get is a clean miss (recompute), never corrupt")
	}
	if cache.Stats().Size != 0 {
		t.Fatalf("[A-05] expected empty cache after malformed Put, size=%d",
			cache.Stats().Size)
	}

	// A well-formed entry of the same N must still be accepted.
	good := PolValues{K: 4, N: 100, Values: make([]fermat, 2)}
	for i := range good.Values {
		good.Values[i] = make(fermat, 101) // exactly N+1
		good.Values[i][0] = big.Word(0xABCD + i)
	}
	cache.Put(data, good)
	got, found := cache.Get(data, 4, 100)
	if !found {
		t.Fatal("[A-05] well-formed entry must still be cached")
	}
	if got.N != 100 || len(got.Values) != 2 {
		t.Fatalf("[A-05] well-formed entry corrupted: N=%d len=%d",
			got.N, len(got.Values))
	}
	got.Release()
}
