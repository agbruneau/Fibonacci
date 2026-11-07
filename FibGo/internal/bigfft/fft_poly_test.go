package bigfft

import (
	"math/big"
	"testing"
)

func TestPoly_Mul(t *testing.T) {
	t.Parallel()

	t.Run("Multiply two polynomials", func(t *testing.T) {
		t.Parallel()
		k := uint(4)
		m := 1
		p := Poly{K: k, M: m}
		p.A = make([]nat, 1<<k)
		for i := range p.A {
			p.A[i] = nat{big.Word(i + 1)}
		}

		q := Poly{K: k, M: m}
		q.A = make([]nat, 1<<k)
		for i := range q.A {
			q.A[i] = nat{big.Word(i + 2)}
		}

		result, err := p.Mul(&q)
		if err != nil {
			t.Fatalf("Mul failed: %v", err)
		}

		if result.K != k {
			t.Errorf("Expected K=%d, got %d", k, result.K)
		}
		if len(result.A) == 0 {
			t.Error("Result should have coefficients")
		}
	})

	t.Run("Multiply with different sizes", func(t *testing.T) {
		t.Parallel()
		k := uint(5)
		m := 2
		p := Poly{K: k, M: m}
		p.A = make([]nat, 1<<k)
		for i := range p.A {
			p.A[i] = nat{big.Word(i + 1)}
		}

		q := Poly{K: k, M: m}
		q.A = make([]nat, 1<<k)
		for i := range q.A {
			q.A[i] = nat{big.Word(i + 2)}
		}

		result, err := p.Mul(&q)
		if err != nil {
			t.Fatalf("Mul failed: %v", err)
		}

		if result.K != k {
			t.Errorf("Expected K=%d, got %d", k, result.K)
		}
	})
}

func TestPoly_MulWithBump(t *testing.T) {
	t.Parallel()

	t.Run("Multiply with bump allocator", func(t *testing.T) {
		t.Parallel()
		ba := AcquireBumpAllocator(10000)
		defer ReleaseBumpAllocator(ba)

		k := uint(4)
		m := 1
		p := Poly{K: k, M: m}
		p.A = make([]nat, 1<<k)
		for i := range p.A {
			p.A[i] = nat{big.Word(i + 1)}
		}

		q := Poly{K: k, M: m}
		q.A = make([]nat, 1<<k)
		for i := range q.A {
			q.A[i] = nat{big.Word(i + 2)}
		}

		result, err := p.MulWithBump(&q, ba)
		if err != nil {
			t.Fatalf("MulWithBump failed: %v", err)
		}

		if result.K != k {
			t.Errorf("Expected K=%d, got %d", k, result.K)
		}
		if len(result.A) == 0 {
			t.Error("Result should have coefficients")
		}
	})
}

// TestNTransform_DirtyPoolBucket reproduces audit finding FFT-01: NTransform
// used acquireWordSliceUnsafe (uninitialized pool memory) for the twist
// buffer, so any coefficient index >= len(p.A) kept whatever garbage the
// pool bucket last held instead of being zero. Dirtying the exact bucket
// backing wordCount = (n+1)<<k before calling NTransform must not change
// the NTransform/InvNTransform round-trip result relative to a clean pool.
func TestNTransform_DirtyPoolBucket(t *testing.T) {
	k := uint(4)
	m := 1
	n := valueSize(k, m, 2)

	newPoly := func() Poly {
		p := Poly{K: k, M: m}
		p.A = make([]nat, (1<<k)-1)
		for i := range p.A {
			p.A[i] = nat{big.Word(i + 1)}
		}
		return p
	}

	roundTrip := func() []nat {
		p := newPoly()
		vals, err := p.NTransform(n)
		if err != nil {
			t.Fatalf("NTransform failed: %v", err)
		}
		pRes, err := vals.InvNTransform()
		if err != nil {
			t.Fatalf("InvNTransform failed: %v", err)
		}
		return pRes.A
	}

	// Reference: NTransform/InvNTransform round-trip against a clean pool.
	want := roundTrip()

	// Dirty the exact word-slice bucket that NTransform's twist buffer
	// (wordCount = (n+1)<<k) will be served from.
	wordCount := (n + 1) << k
	dirty := acquireWordSlice(wordCount)
	for i := range dirty {
		dirty[i] = ^big.Word(0)
	}
	releaseWordSlice(dirty)

	got := roundTrip()

	if len(got) != len(want) {
		t.Fatalf("coefficient count mismatch: got %d, want %d", len(got), len(want))
	}
	for i := range want {
		gi := new(big.Int).SetBits(got[i])
		wi := new(big.Int).SetBits(want[i])
		if gi.Cmp(wi) != 0 {
			t.Fatalf("coefficient %d differs after dirtying pool bucket: got %s, want %s", i, gi, wi)
		}
	}
}

func TestPoly_mul(t *testing.T) {
	t.Parallel()

	t.Run("Internal mul function", func(t *testing.T) {
		t.Parallel()
		// mul is called by Mul and MulWithBump, so we test it indirectly
		// But we can also test it more directly by using a custom allocator
		k := uint(4)
		m := 1
		p := Poly{K: k, M: m}
		p.A = make([]nat, 1<<k)
		for i := range p.A {
			p.A[i] = nat{big.Word(i + 1)}
		}

		q := Poly{K: k, M: m}
		q.A = make([]nat, 1<<k)
		for i := range q.A {
			q.A[i] = nat{big.Word(i + 2)}
		}

		// Test via Mul which calls mul internally
		result, err := p.Mul(&q)
		if err != nil {
			t.Fatalf("Mul (which calls mul) failed: %v", err)
		}

		if result.K != k {
			t.Errorf("Expected K=%d, got %d", k, result.K)
		}
	})
}
