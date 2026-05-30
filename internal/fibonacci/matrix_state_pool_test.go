package fibonacci

import (
	"math/big"
	"testing"
)

// TestReleaseMatrixState_Oversize guards the anti-pinning branch of
// releaseMatrixState / matrixStateOversized (matrix_types.go), which the
// existing pool tests never reach because they only release freshly-Reset
// states (audit F-006). It mirrors TestReleaseState_OverLimit_AliasesCleared on
// the matrix side: a naive regression (inverting the oversize check or dropping
// a group helper) would otherwise pass the whole suite. matrixState carries no
// alias invariant (results are deep-copied via matrix.Set), so this is a pure
// coverage guard with no memory hazard.
func TestReleaseMatrixState_Oversize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		element func(s *matrixState) *big.Int
	}{
		{"strassen_product_p1", func(s *matrixState) *big.Int { return s.p1 }},
		{"strassen_sum_s1", func(s *matrixState) *big.Int { return s.s1 }},
		{"scratch_t1", func(s *matrixState) *big.Int { return s.t1 }},
		{"matrix_element_res_a", func(s *matrixState) *big.Int { return s.res.a }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := acquireMatrixState()
			z := tc.element(s)
			z.SetBit(z, MaxPooledBitLen+1, 1)
			if !checkLimit(z) {
				t.Fatalf("precondition: inflated %s (BitLen=%d) not over MaxPooledBitLen=%d",
					tc.name, z.BitLen(), MaxPooledBitLen)
			}
			if !matrixStateOversized(s) {
				t.Fatalf("expected matrixStateOversized=true after inflating %s", tc.name)
			}
			// Exercise the early-return (no Put) branch; must not panic.
			releaseMatrixState(s)
		})
	}

	t.Run("nominal_not_oversized", func(t *testing.T) {
		t.Parallel()
		s := acquireMatrixState()
		if matrixStateOversized(s) {
			t.Fatalf("freshly reset state should not be reported oversized")
		}
		releaseMatrixState(s)
	})
}
