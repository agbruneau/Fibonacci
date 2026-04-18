package fibonacci

import (
	"fmt"
	"math/big"
	"math/bits"
)

// FastDoublingMod computes F(n) mod m using the fast doubling algorithm.
// Memory usage is O(log(m)) regardless of n, making it suitable for
// computing the last K digits of F(n) for arbitrarily large n.
//
// Uses the identities:
//
//	F(2k)   = F(k) * (2*F(k+1) - F(k))  mod m
//	F(2k+1) = F(k+1)² + F(k)²            mod m
func FastDoublingMod(n uint64, m *big.Int) (*big.Int, error) {
	if m == nil || m.Sign() <= 0 {
		return nil, fmt.Errorf("modulus must be positive")
	}

	if n == 0 {
		return big.NewInt(0), nil
	}

	fk := big.NewInt(0)  // F(k)
	fk1 := big.NewInt(1) // F(k+1)
	t1 := new(big.Int)   // temporary
	t2 := new(big.Int)   // temporary

	numBits := bits.Len64(n)

	for i := numBits - 1; i >= 0; i-- {
		// F(2k) = F(k) * (2*F(k+1) - F(k)) mod m
		t1.Lsh(fk1, 1)
		t1.Sub(t1, fk)
		t1.Mod(t1, m)
		// Handle negative mod result
		if t1.Sign() < 0 {
			t1.Add(t1, m)
		}
		t1.Mul(t1, fk)
		t1.Mod(t1, m)

		// F(2k+1) = F(k+1)² + F(k)² mod m
		t2.Mul(fk1, fk1)
		fk.Mul(fk, fk)
		t2.Add(t2, fk)
		t2.Mod(t2, m)

		// Assign
		fk.Set(t1)
		fk1.Set(t2)

		// If bit is set: shift to F(2k+1), F(2k+2)
		// i is a bit index in [0, 63], uint conversion safe. #nosec G115
		if (n>>uint(i))&1 == 1 {
			t1.Add(fk, fk1)
			t1.Mod(t1, m)
			fk.Set(fk1)
			fk1.Set(t1)
		}
	}

	return fk, nil
}
