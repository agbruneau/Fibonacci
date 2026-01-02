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
