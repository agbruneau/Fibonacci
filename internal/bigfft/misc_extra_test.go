package bigfft

import (
	"math/big"
	"strings"
	"testing"
)

// assertFermatEqual fails when the two fermat values differ in shape or words.
func assertFermatEqual(t *testing.T, got, want fermat) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("word %d: got %#x, want %#x", i, got[i], want[i])
		}
	}
}

// TestFermatSafeWrappersSuccess verifies that each *Safe wrapper delegates to
// its panicking counterpart on well-formed operands (the error tests cover
// rejection; these cover the pass-through results).
func TestFermatSafeWrappersSuccess(t *testing.T) {
	t.Parallel()
	// n = 4: 5-word operands with a zero top word (valid residues).
	x := fermat{1, 2, 3, 4, 0}
	y := fermat{5, 6, 7, 8, 0}

	t.Run("AddSafe matches Add", func(t *testing.T) {
		t.Parallel()
		got, err := make(fermat, 5).AddSafe(x, y)
		if err != nil {
			t.Fatalf("AddSafe: %v", err)
		}
		assertFermatEqual(t, got, make(fermat, 5).Add(x, y))
	})
	t.Run("SubSafe matches Sub", func(t *testing.T) {
		t.Parallel()
		got, err := make(fermat, 5).SubSafe(x, y)
		if err != nil {
			t.Fatalf("SubSafe: %v", err)
		}
		assertFermatEqual(t, got, make(fermat, 5).Sub(x, y))
	})
	t.Run("ShiftSafe matches Shift", func(t *testing.T) {
		t.Parallel()
		got := make(fermat, 5)
		if err := got.ShiftSafe(x, 7); err != nil {
			t.Fatalf("ShiftSafe: %v", err)
		}
		want := make(fermat, 5)
		want.Shift(x, 7)
		assertFermatEqual(t, got, want)
	})
	t.Run("SqrSafe matches Sqr", func(t *testing.T) {
		t.Parallel()
		got, err := make(fermat, 8*4).SqrSafe(x)
		if err != nil {
			t.Fatalf("SqrSafe: %v", err)
		}
		assertFermatEqual(t, got, make(fermat, 8*4).Sqr(x))
	})
	t.Run("MulSafe matches Mul", func(t *testing.T) {
		t.Parallel()
		got, err := make(fermat, 8*4).MulSafe(x, y)
		if err != nil {
			t.Fatalf("MulSafe: %v", err)
		}
		assertFermatEqual(t, got, make(fermat, 8*4).Mul(x, y))
	})
}

// TestFermatSafeWrappersSecondOperandMismatch covers the validation arms not
// reached by the existing rejection tests (AddSafe on y, SubSafe on x).
func TestFermatSafeWrappersSecondOperandMismatch(t *testing.T) {
	t.Parallel()
	z := make(fermat, 5)
	x := make(fermat, 5)
	bad := make(fermat, 7)

	if _, err := z.AddSafe(x, bad); err == nil || !strings.Contains(err.Error(), "len(y)") {
		t.Errorf("AddSafe must reject mismatched y, got %v", err)
	}
	if _, err := z.SubSafe(bad, x); err == nil || !strings.Contains(err.Error(), "len(x)") {
		t.Errorf("SubSafe must reject mismatched x, got %v", err)
	}
}

// TestScannerChunkSizePanicsAtThreshold pins the documented precondition of
// chunkSize: sizes at or below quadraticScanThreshold are caller bugs.
func TestScannerChunkSizePanicsAtThreshold(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("chunkSize must panic for size <= quadraticScanThreshold")
		}
	}()
	var s scanner
	s.chunkSize(quadraticScanThreshold)
}

// TestFromDecimalStringMalformedHalves drives the divide-and-conquer scanner
// above the quadratic threshold with an invalid character in each half: both
// recursive branches must surface the validation error (audit F-016).
func TestFromDecimalStringMalformedHalves(t *testing.T) {
	t.Parallel()
	longDigits := strings.Repeat("7", 2*quadraticScanThreshold)

	tests := []struct {
		name, input string
	}{
		{"left half malformed", "x" + longDigits},
		{"right half malformed", longDigits + "x"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			z, err := FromDecimalString(tc.input)
			if err == nil {
				t.Fatalf("expected error, got value with bitlen %d", z.BitLen())
			}
			if !strings.Contains(err.Error(), "invalid decimal input") {
				t.Fatalf("error must surface the invalid-input cause, got: %v", err)
			}
		})
	}
}

// TestBumpAllocatorEdgeCases covers the defensive paths of the bump
// allocator: nil release, over-capacity fallback, and estimation extremes.
func TestBumpAllocatorEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("ReleaseBumpAllocator nil is a no-op", func(t *testing.T) {
		t.Parallel()
		ReleaseBumpAllocator(nil)
	})

	t.Run("AllocUnsafe falls back beyond capacity", func(t *testing.T) {
		t.Parallel()
		ba := AcquireBumpAllocator(8)
		defer ReleaseBumpAllocator(ba)
		used := ba.Used()
		s := ba.AllocUnsafe(64)
		if len(s) != 64 {
			t.Fatalf("fallback slice has len %d, want 64", len(s))
		}
		if ba.Used() != used {
			t.Fatal("fallback allocation must not consume bump capacity")
		}
	})

	t.Run("EstimateBumpCapacity extremes", func(t *testing.T) {
		t.Parallel()
		if got := EstimateBumpCapacity(0); got != 0 {
			t.Fatalf("EstimateBumpCapacity(0) = %d, want 0", got)
		}
		// Above the largest fftSizeThreshold entry the FFT level falls back
		// to the last table slot; the estimate must remain positive.
		huge := int(fftSizeThreshold[len(fftSizeThreshold)-1]/int64(_W)) + 1
		if got := EstimateBumpCapacity(huge); got <= 0 {
			t.Fatalf("EstimateBumpCapacity(%d) = %d, want > 0", huge, got)
		}
	})
}

// TestAcquireWordSliceUnsafeBeyondPoolMax covers the direct-allocation route
// for sizes that exceed the largest pool bucket.
func TestAcquireWordSliceUnsafeBeyondPoolMax(t *testing.T) {
	t.Parallel()
	size := wordSliceSizes[len(wordSliceSizes)-1] + 1
	s := acquireWordSliceUnsafe(size)
	if len(s) != size {
		t.Fatalf("len = %d, want %d", len(s), size)
	}
}

// TestPreWarmPoolsLargeTiers exercises the 1M and 10M buffer-count tiers and
// asserts the size estimates still target real pool buckets: if
// EstimateMemoryNeeds ever outgrew the bucket tables, warming would silently
// become a no-op.
func TestPreWarmPoolsLargeTiers(t *testing.T) {
	t.Parallel()
	for _, n := range []uint64{1_000_000, 10_000_000} {
		est := EstimateMemoryNeeds(n)
		if getWordSlicePoolIndex(est.MaxWordSliceSize) < 0 {
			t.Fatalf("n=%d: MaxWordSliceSize %d exceeds poolable buckets", n, est.MaxWordSliceSize)
		}
		if getFermatPoolIndex(est.MaxFermatSize) < 0 {
			t.Fatalf("n=%d: MaxFermatSize %d exceeds poolable buckets", n, est.MaxFermatSize)
		}
		PreWarmPools(n)
	}
}

// TestPolValuesClone verifies the deep-copy contract: same shape and words,
// independent backing, and no inherited pool ownership.
func TestPolValuesClone(t *testing.T) {
	t.Parallel()
	const (
		K = 4
		n = 3
	)
	backing := make([]big.Word, K*(n+1))
	values := make([]fermat, K)
	for i := range values {
		values[i] = fermat(backing[i*(n+1) : (i+1)*(n+1)])
		for j := range values[i] {
			values[i][j] = big.Word(i*10 + j)
		}
	}
	orig := PolValues{K: 2, N: n, Values: values, pooledBacking: backing, pooledValues: true}

	clone := orig.Clone()
	if clone.K != orig.K || clone.N != orig.N {
		t.Fatal("Clone must copy K and N")
	}
	if clone.pooledBacking != nil || clone.pooledValues {
		t.Fatal("Clone must not inherit pool ownership markers")
	}
	for i := range orig.Values {
		assertFermatEqual(t, clone.Values[i], orig.Values[i])
	}
	clone.Values[0][0] = 12345
	if orig.Values[0][0] == 12345 {
		t.Fatal("mutating the clone must not affect the original (shared backing)")
	}
}

// TestInvTransformsRejectMalformedValues: a PolValues whose coefficients are
// not N+1 words must surface a validation error through every
// inverse-transform entry point, not a panic and not a corrupt result.
func TestInvTransformsRejectMalformedValues(t *testing.T) {
	t.Parallel()
	malformed := func() PolValues {
		return PolValues{K: 2, N: 8, Values: []fermat{
			make(fermat, 3), make(fermat, 3), make(fermat, 3), make(fermat, 3),
		}}
	}
	assertValidationErr := func(t *testing.T, err error) {
		t.Helper()
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
		if !strings.Contains(err.Error(), "validation failed") {
			t.Fatalf("unexpected error cause: %v", err)
		}
	}

	t.Run("InvTransform", func(t *testing.T) {
		t.Parallel()
		v := malformed()
		_, err := v.InvTransform()
		assertValidationErr(t, err)
	})
	t.Run("InvTransformWithBump", func(t *testing.T) {
		t.Parallel()
		v := malformed()
		ba := AcquireBumpAllocator(1024)
		defer ReleaseBumpAllocator(ba)
		_, err := v.InvTransformWithBump(ba)
		assertValidationErr(t, err)
	})
	t.Run("InvNTransform", func(t *testing.T) {
		t.Parallel()
		v := malformed()
		_, err := v.InvNTransform()
		assertValidationErr(t, err)
	})
}

// TestMulNegativeOperandsFFTPath checks sign handling on the FFT route of the
// package-level entry points (operands above the default threshold).
func TestMulNegativeOperandsFFTPath(t *testing.T) {
	t.Parallel()
	x := new(big.Int).Neg(makeBigInt(81, 2000))
	y := makeBigInt(82, 2000)
	expected := new(big.Int).Mul(x, y)

	got, err := Mul(x, y)
	if err != nil {
		t.Fatalf("Mul: %v", err)
	}
	if got.Cmp(expected) != 0 {
		t.Fatal("Mul negative result differs from math/big reference")
	}
	if got.Sign() >= 0 {
		t.Fatal("product of negative and positive operands must be negative")
	}

	z := new(big.Int)
	got2, err := MulTo(z, x, y)
	if err != nil {
		t.Fatalf("MulTo: %v", err)
	}
	if got2.Cmp(expected) != 0 {
		t.Fatal("MulTo negative result differs from math/big reference")
	}
}

// TestTransformCacheConfigAccessor pins the Config() snapshot contract on a
// local cache instance.
func TestTransformCacheConfigAccessor(t *testing.T) {
	t.Parallel()
	cfg := TransformCacheConfig{MaxEntries: 7, MinBitLen: 4242, Enabled: true}
	tc := NewTransformCache(cfg)
	if got := tc.Config(); got != cfg {
		t.Fatalf("Config() = %+v, want %+v", got, cfg)
	}
}

// TestReleaseNilReceivers pins the documented nil-safety of the release APIs.
func TestReleaseNilReceivers(t *testing.T) {
	t.Parallel()
	var p *Poly
	p.Release()
	var v *PolValues
	v.Release()
}

// TestPolyFromNatEmptyInput: an empty operand must still produce one zeroed
// coefficient so downstream transforms see a well-formed polynomial.
func TestPolyFromNatEmptyInput(t *testing.T) {
	t.Parallel()
	p := polyFromNat(nil, 2, 3)
	if len(p.A) != 1 {
		t.Fatalf("empty input must yield exactly one coefficient, got %d", len(p.A))
	}
	if len(p.A[0]) != 3 {
		t.Fatalf("coefficient must have m words, got %d", len(p.A[0]))
	}
	for i, w := range p.A[0] {
		if w != 0 {
			t.Fatalf("coefficient word %d = %#x, want 0", i, w)
		}
	}
	if got := p.Int(); got != nil {
		t.Fatalf("zero polynomial must evaluate to zero (nil nat), got %v", got)
	}
}

// TestIntToCarrySaturatedWord rebuilds an integer whose coefficient sums
// overflow into an all-ones word: P(b) with A = [{0,b-1,b-1},{1}], M=1 equals
// exactly b^3. The saturated middle word forces the long-carry branch of
// IntTo; a carry-propagation bug would break the equality.
func TestIntToCarrySaturatedWord(t *testing.T) {
	t.Parallel()
	maxw := ^big.Word(0)
	p := Poly{K: 1, M: 1, A: []nat{{0, maxw, maxw}, {1}}}

	got := new(big.Int)
	got.SetBits(p.Int())

	expected := new(big.Int).Lsh(big.NewInt(1), uint(3*_W))
	if got.Cmp(expected) != 0 {
		t.Fatalf("IntTo carry propagation broken: got %s, want b^3", got.String())
	}
}

// TestNTransformPanicsWhenPolyOverflowsK pins the documented precondition:
// a polynomial with 1<<K or more coefficients is a caller bug.
func TestNTransformPanicsWhenPolyOverflowsK(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("NTransform must panic when len(p.A) >= 1<<k")
		}
	}()
	p := Poly{K: 1, M: 1, A: []nat{{1}, {2}}} // len(A) == 1<<K
	_, _ = p.NTransform(4)
}

// TestPoolNewClosuresMatchBucketSizes executes every pool's New constructor
// directly and checks it against the size table: a mismatch would corrupt
// the by-capacity routing invariant of releaseWordSlice and friends.
func TestPoolNewClosuresMatchBucketSizes(t *testing.T) {
	t.Parallel()
	for i := range wordSlicePools {
		if got := len(wordSlicePools[i].New().([]big.Word)); got != wordSliceSizes[i] {
			t.Errorf("wordSlicePools[%d].New len = %d, want %d", i, got, wordSliceSizes[i])
		}
	}
	for i := range fermatPools {
		if got := len(fermatPools[i].New().(fermat)); got != fermatSizes[i] {
			t.Errorf("fermatPools[%d].New len = %d, want %d", i, got, fermatSizes[i])
		}
	}
	for i := range natSlicePools {
		if got := len(natSlicePools[i].New().([]nat)); got != natSliceSizes[i] {
			t.Errorf("natSlicePools[%d].New len = %d, want %d", i, got, natSliceSizes[i])
		}
	}
	for i := range fermatSlicePools {
		if got := len(fermatSlicePools[i].New().([]fermat)); got != fermatSliceSizes[i] {
			t.Errorf("fermatSlicePools[%d].New len = %d, want %d", i, got, fermatSliceSizes[i])
		}
	}
}
