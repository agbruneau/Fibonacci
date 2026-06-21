package bigfft

import (
	"math/big"
	"testing"
)

// TestReleaseWordSliceResizedReturnsToBucket verifies that a slice whose
// length was reduced after acquisition (but whose capacity still matches a
// bucket size exactly) is correctly returned to its pool, and not silently
// dropped.
//
// Regression test for R1.4: releaseWordSlice must route on cap(slice), not
// on len(slice), and must restore the full bucket capacity before Put so
// the next acquisition gets a usable buffer.
func TestReleaseWordSliceResizedReturnsToBucket(t *testing.T) {
	// No t.Parallel(): we inspect the global miss-counter state.

	// Acquire a slice from the smallest bucket (size 64).
	s := acquireWordSliceUnsafe(64)
	if cap(s) != 64 {
		t.Fatalf("precondition: want cap=64 from bucket 0, got cap=%d", cap(s))
	}
	if len(s) != 64 {
		t.Fatalf("precondition: want len=64 from acquireWordSliceUnsafe(64), got len=%d", len(s))
	}

	// Simulate a caller that reshapes the slice down (e.g. via a temporary
	// view): cap stays 64, only len shrinks.
	s = s[:30]
	if cap(s) != 64 {
		t.Fatalf("precondition: cap must stay 64 after s[:30], got cap=%d", cap(s))
	}

	// The reliable contract: a slice whose cap matches a bucket exactly must
	// be routed back to that bucket (Put), NOT counted as a miss — even when
	// len has been reduced. A zero miss-counter delta proves the slice was
	// pooled rather than dropped.
	//
	// We deliberately do NOT assert sync.Pool *identity* (that the same
	// backing array reappears on a later Get): the stdlib makes no such
	// guarantee, and the per-P victim cache can drop any entry on GC, so a
	// reappearance check is inherently flaky — notably under the -race build's
	// different GC timing. The exact-bucket routing is further covered by
	// TestReleaseWordSliceAllExactBuckets.
	missBefore := WordSlicePoolMissCount()
	releaseWordSlice(s)
	missAfter := WordSlicePoolMissCount()
	if missAfter != missBefore {
		t.Fatalf("releaseWordSlice(cap=64, len=30) was treated as a miss: "+
			"counter delta=%d (want 0)", missAfter-missBefore)
	}
}

// TestReleaseWordSliceMissCountOnNonBucketCap verifies that a slice whose
// capacity does not match any bucket exactly is NOT pushed back into a
// pool (which would corrupt the bucket size invariant) and that the miss
// is recorded for observability.
func TestReleaseWordSliceMissCountOnNonBucketCap(t *testing.T) {
	// No t.Parallel(): the miss counter is global. Other parallel tests
	// may concurrently increment it, so we assert with >= rather than ==.

	// cap=42 falls between buckets (size 64 is the smallest) and is not a
	// bucket size itself — must be a miss.
	s := make([]big.Word, 42)

	missBefore := WordSlicePoolMissCount()
	releaseWordSlice(s)
	missAfter := WordSlicePoolMissCount()

	if missAfter < missBefore+1 {
		t.Fatalf("releaseWordSlice(cap=42) did not increment miss counter: "+
			"before=%d after=%d (want delta >= 1)", missBefore, missAfter)
	}
}

// TestReleaseWordSliceMissCountOnThreeIndexSlice verifies the realistic
// failure mode: a pooled slice that was three-index-sliced (s[a:b:c])
// has its cap shrunk away from the bucket size and must be dropped to GC,
// not silently re-pooled into the wrong bucket.
func TestReleaseWordSliceMissCountOnThreeIndexSlice(t *testing.T) {
	// No t.Parallel(): inspecting the global miss counter.

	// Acquire from bucket 1 (size 256) and shrink cap via three-index slice.
	s := acquireWordSliceUnsafe(256)
	if cap(s) != 256 {
		t.Fatalf("precondition: want cap=256 from bucket 1, got cap=%d", cap(s))
	}
	// s[:100:100] has cap=100, which matches no bucket exactly.
	shrunk := s[:100:100]
	if cap(shrunk) != 100 {
		t.Fatalf("precondition: three-index slice should give cap=100, got %d", cap(shrunk))
	}

	missBefore := WordSlicePoolMissCount()
	releaseWordSlice(shrunk)
	missAfter := WordSlicePoolMissCount()

	if missAfter < missBefore+1 {
		t.Fatalf("releaseWordSlice on three-index-shrunk slice (cap=100) "+
			"did not increment miss counter: before=%d after=%d", missBefore, missAfter)
	}

	// We still hold the original full-cap reference s. Releasing it must
	// succeed cleanly (cap=256 matches bucket 1).
	missBefore2 := WordSlicePoolMissCount()
	releaseWordSlice(s)
	missAfter2 := WordSlicePoolMissCount()
	if missAfter2 != missBefore2 {
		t.Fatalf("releaseWordSlice on full-cap slice (cap=256, bucket 1) "+
			"was wrongly counted as a miss: delta=%d", missAfter2-missBefore2)
	}
}

// TestReleaseWordSliceAllExactBuckets verifies that every documented bucket
// size is accepted as an exact match by releaseWordSlice (no miss
// increment) — this guards against any future drift between
// wordSliceSizes and getWordSlicePoolIndex.
func TestReleaseWordSliceAllExactBuckets(t *testing.T) {
	// No t.Parallel(): global miss counter.

	for _, size := range wordSliceSizes {
		// Skip the largest classes to keep the test memory-light: exercising
		// the first few buckets is enough to validate the routing logic;
		// the boundary table is already covered by TestPoolIndexBoundaryValues.
		if size > 16384 {
			continue
		}
		s := acquireWordSliceUnsafe(size)
		if cap(s) != size {
			t.Fatalf("precondition: acquireWordSliceUnsafe(%d) gave cap=%d", size, cap(s))
		}
		// Reduce length to force the "restore full capacity" path.
		s = s[:1]

		missBefore := WordSlicePoolMissCount()
		releaseWordSlice(s)
		missAfter := WordSlicePoolMissCount()
		if missAfter != missBefore {
			t.Errorf("releaseWordSlice(cap=%d) wrongly counted as miss: delta=%d",
				size, missAfter-missBefore)
		}
	}
}
