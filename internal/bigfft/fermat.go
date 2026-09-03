package bigfft

import (
	"fmt"
	"math/big"
	"math/bits"
)

// ─────────────────────────────────────────────────────────────────────────────
// Error-returning wrappers (audit R3.8)
//
// The Mul, Sqr, Add, Sub and Shift methods below panic on size mismatches.
// Historically those were treated as programmer-error invariants because the
// internal FFT code always sizes its operands correctly. However, the public
// bigfft.Mul / Sqr / MulTo / SqrTo entry points (fft.go) recover those panics
// and report them as opaque errors, which makes it impossible to distinguish a
// genuine logic bug from external corruption.
//
// The *Safe wrappers below return a structured error on size mismatch instead
// of panicking, while preserving the panic-based fast path used everywhere
// inside the package. They are intended for callers who want to validate
// untrusted layouts (e.g. data crossing a serialization boundary) without
// relying on recover().
//
// Internal post-condition panics (e.g. "len(z) > 2n+1", "unexpected carry
// after normalization") are NOT converted: they signal a bug inside Mul/Sqr
// itself, not bad input from the caller. Surfacing them as errors would mask
// regressions.
// ─────────────────────────────────────────────────────────────────────────────

// smallMulThreshold is the number of words below which basicMul is used
// instead of big.Int.Mul for fermat multiplication. It trades big.Int's
// setup overhead against basicMul's O(n²) inner loop; 30 words is a setting,
// with no benchmark in the repo pinning it against a nearby value.
const smallMulThreshold = 30

// Arithmetic modulo 2^n+1.

// fermat represents a number modulo 2^(w*_W) + 1 as a slice of length w+1.
// The last word is zero or one. A number has at most two representatives
// satisfying the 0-1 last word constraint.
type fermat nat

func (z fermat) String() string { return nat(z).String() }

func (z fermat) norm() {
	n := len(z) - 1
	c := z[n]
	if c == 0 {
		return
	}
	if z[0] >= c {
		z[n] = 0
		z[0] -= c
		return
	}
	// z[0] < z[n].
	subVW(z, z, c) // Subtract c
	if c > 1 {
		z[n] -= c - 1
	}
	// Add back c.
	if z[n] == 1 {
		z[n] = 0
		return
	}
	addVW(z, z, 1)
}

// Shift computes (x << k) mod (2^n+1).
func (z fermat) Shift(x fermat, k int) {
	if len(z) != len(x) {
		panic("len(z) != len(x) in Shift")
	}
	n := len(x) - 1
	// Shift by n*_W is taking the opposite.
	k %= 2 * n * _W
	if k < 0 {
		k += 2 * n * _W
	}
	neg := false
	if k >= n*_W {
		k -= n * _W
		neg = true
	}

	kw, kb := k/_W, k%_W

	z[n] = 1 // Add (-1)
	if !neg {
		for i := 0; i < kw; i++ {
			z[i] = 0
		}
		// Shift left by kw words.
		// x = a·2^(n-k) + b
		// x<<k = (b<<k) - a
		copy(z[kw:], x[:n-kw])
		b := subVV(z[:kw+1], z[:kw+1], x[n-kw:])
		if z[kw+1] > 0 {
			z[kw+1] -= b
		} else {
			subVW(z[kw+1:], z[kw+1:], b)
		}
	} else {
		for i := kw + 1; i < n; i++ {
			z[i] = 0
		}
		// Shift left and negate, by kw words: the low part of z receives the
		// high part of x, then the high part of z has the low part of x
		// subtracted from it.
		copy(z[:kw+1], x[n-kw:n+1])
		b := subVV(z[kw:n], z[kw:n], x[:n-kw])
		z[n] -= b
	}
	// Add back 1.
	switch {
	case z[n] > 0:
		z[n]--
	case z[0] < ^big.Word(0):
		z[0]++
	default:
		addVW(z, z, 1)
	}
	// Shift left by kb bits.
	shlVU(z, z, uint(kb)) // #nosec G115 -- kb = k%_W is in [0, _W-1] by construction, so it always fits positively in uint
	z.norm()
}

// ShiftHalf shifts x by k/2 bits the left. Shifting by 1/2 bit
// is multiplication by sqrt(2) mod 2^n+1 which is 2^(3n/4) - 2^(n/4).
// A temporary buffer must be provided in tmp.
func (z fermat) ShiftHalf(x fermat, k int, tmp fermat) {
	n := len(z) - 1
	if k%2 == 0 {
		z.Shift(x, k/2)
		return
	}
	u := (k - 1) / 2
	a := u + (3*_W/4)*n
	b := u + (_W/4)*n
	z.Shift(x, a)
	tmp.Shift(x, b)
	z.Sub(z, tmp)
}

// Add computes addition mod 2^n+1.
func (z fermat) Add(x, y fermat) fermat {
	if len(z) != len(x) {
		panic("Add: len(z) != len(x)")
	}
	addVV(z, x, y) // there cannot be a carry here.
	z.norm()
	return z
}

// Sub computes subtraction mod 2^n+1.
func (z fermat) Sub(x, y fermat) fermat {
	if len(z) != len(x) {
		panic("fermat.Sub: len(z) != len(x)")
	}
	n := len(y) - 1
	b := subVV(z[:n], x[:n], y[:n])
	b += y[n]
	// If b > 0, we need to subtract b<<n, which is the same as adding b.
	z[n] = x[n]
	if z[0] <= ^big.Word(0)-b {
		z[0] += b
	} else {
		addVW(z, z, b)
	}
	z.norm()
	return z
}

func (z fermat) Mul(x, y fermat) fermat {
	if len(x) != len(y) {
		panic("Mul: len(x) != len(y)")
	}
	n := len(x) - 1
	if n < smallMulThreshold {
		z = z[:2*n+2]
		basicMul(z, x, y)
		z = z[:2*n+1]
	} else {
		var xi, yi, zi big.Int
		xi.SetBits(x)
		yi.SetBits(y)
		zi.SetBits(z)
		zb := zi.Mul(&xi, &yi).Bits()
		if len(zb) <= n {
			// Short product.
			copy(z, zb)
			for i := len(zb); i < len(z); i++ {
				z[i] = 0
			}
			return z
		}
		z = zb
	}
	return z.reduce(n, "Mul")
}

// reduce folds the raw product z (at most 2n+1 words) back modulo
// 2^(n*_W)+1. We have z = z[:n] + 1<<(n*W) * z[n:2n+1], which normalizes to
// z = z[:n] - z[n:2n] + z[2n]; the carries are then restored (z[n] -= c2 is
// the same as z[0] += c2) and the result normalized. Shared tail of Mul and
// Sqr; who names the caller in the post-condition panic, whose exact text is
// a sentinel of fermatPostConditionPanics (fft.go) and must not change.
func (z fermat) reduce(n int, who string) fermat {
	if len(z) > 2*n+1 {
		panic("len(z) > 2n+1")
	}
	c1 := big.Word(0)
	if len(z) > 2*n {
		c1 = addVW(z[:n], z[:n], z[2*n])
	}
	c2 := big.Word(0)
	if len(z) >= 2*n {
		c2 = subVV(z[:n], z[:n], z[n:2*n])
	} else {
		m := len(z) - n
		c2 = subVV(z[:m], z[:m], z[n:])
		c2 = subVW(z[m:n], z[m:n], c2)
	}
	z = z[:n+1]
	z[n] = c1
	if c := addVW(z, z, c2); c != 0 {
		panic("fermat." + who + ": unexpected carry after normalization")
	}
	z.norm()
	return z
}

// Sqr computes x*x mod 2^(n*_W)+1. It is optimized compared to Mul(x, x)
// because:
//   - For n < smallMulThreshold: basicSqr exploits the symmetry of x*x to
//     skip roughly half of the partial products.
//   - For n >= smallMulThreshold: uses the same big.Int pointer for both
//     operands so that Go's math/big can detect squaring internally.
func (z fermat) Sqr(x fermat) fermat {
	n := len(x) - 1
	if n < smallMulThreshold {
		z = z[:2*n+2]
		basicSqr(z, x)
		z = z[:2*n+1]
	} else {
		var xi, zi big.Int
		xi.SetBits(x)
		zi.SetBits(z)
		// Pass &xi for both operands so big.Int.Mul detects squaring
		zb := zi.Mul(&xi, &xi).Bits()
		if len(zb) <= n {
			// Short product.
			copy(z, zb)
			for i := len(zb); i < len(z); i++ {
				z[i] = 0
			}
			return z
		}
		z = zb
	}
	return z.reduce(n, "Sqr")
}

// MulSafe is the error-returning counterpart of (fermat).Mul. It validates
// that len(x) == len(y) before delegating to Mul, returning a descriptive
// error on mismatch instead of panicking. Internal post-condition violations
// inside Mul still panic (they signal a bug, not bad input).
func (z fermat) MulSafe(x, y fermat) (fermat, error) {
	if len(x) != len(y) {
		return nil, fmt.Errorf("fermat.MulSafe: operand size mismatch: len(x)=%d, len(y)=%d", len(x), len(y))
	}
	return z.Mul(x, y), nil
}

// SqrSafe is the error-returning counterpart of (fermat).Sqr. Sqr itself has
// no externally-validatable size precondition (it only requires len(x) >= 1),
// but we expose this wrapper for symmetry with MulSafe and to centralize the
// minimal validation. Internal post-condition violations still panic.
func (z fermat) SqrSafe(x fermat) (fermat, error) {
	if len(x) < 1 {
		return nil, fmt.Errorf("fermat.SqrSafe: empty operand: len(x)=%d", len(x))
	}
	return z.Sqr(x), nil
}

// ShiftSafe is the error-returning counterpart of (fermat).Shift. It
// validates that len(z) == len(x) before delegating to Shift.
func (z fermat) ShiftSafe(x fermat, k int) error {
	if len(z) != len(x) {
		return fmt.Errorf("fermat.ShiftSafe: operand size mismatch: len(z)=%d, len(x)=%d", len(z), len(x))
	}
	z.Shift(x, k)
	return nil
}

// AddSafe is the error-returning counterpart of (fermat).Add.
func (z fermat) AddSafe(x, y fermat) (fermat, error) {
	if len(z) != len(x) {
		return nil, fmt.Errorf("fermat.AddSafe: operand size mismatch: len(z)=%d, len(x)=%d", len(z), len(x))
	}
	if len(z) != len(y) {
		return nil, fmt.Errorf("fermat.AddSafe: operand size mismatch: len(z)=%d, len(y)=%d", len(z), len(y))
	}
	return z.Add(x, y), nil
}

// SubSafe is the error-returning counterpart of (fermat).Sub.
func (z fermat) SubSafe(x, y fermat) (fermat, error) {
	if len(z) != len(x) {
		return nil, fmt.Errorf("fermat.SubSafe: operand size mismatch: len(z)=%d, len(x)=%d", len(z), len(x))
	}
	if len(z) != len(y) {
		return nil, fmt.Errorf("fermat.SubSafe: operand size mismatch: len(z)=%d, len(y)=%d", len(z), len(y))
	}
	return z.Sub(x, y), nil
}

// copied from math/big
//
// basicMul multiplies x and y and leaves the result in z.
// The (non-normalized) result is placed in z[0 : len(x) + len(y)].
func basicMul(z, x, y fermat) {
	clear(z)
	for i, d := range y {
		if d != 0 {
			z[len(x)+i] = addMulVVW(z[i:i+len(x)], x, d)
		}
	}
}

// basicSqr computes x*x and places the result in z.
// It exploits the symmetry of squaring: for i != j, the cross-terms
// x[i]*x[j] appear twice, so we compute them once and double (shift left by 1).
// Then we add the diagonal terms x[i]*x[i].
// This saves roughly half the multiplications compared to basicMul(z, x, x).
func basicSqr(z, x fermat) {
	n := len(x)
	clear(z[:2*n])

	// Off-diagonal terms: for each i, accumulate x[i] * x[j] for j > i.
	// Each iteration computes x[i] * x[i+1..n-1] and places the partial
	// products into z[2*i+1..i+n-1] with carry into z[i+n].
	for i := 0; i < n-1; i++ {
		if x[i] != 0 {
			// x[i+1:n] has length n-1-i, z[2*i+1:i+n] also has length n-1-i
			z[i+n] = addMulVVW(z[2*i+1:i+n], x[i+1:n], x[i])
		}
	}

	// Double the off-diagonal terms (shift left by 1 bit)
	shlVU(z[:2*n], z[:2*n], 1)

	// Add diagonal terms: x[i]*x[i]
	for i := 0; i < n; i++ {
		if x[i] != 0 {
			// x[i]*x[i] produces a 2-word result
			hi, lo := bits.Mul(uint(x[i]), uint(x[i]))
			c := addVW(z[2*i:2*i+1], z[2*i:2*i+1], big.Word(lo))
			c = addVW(z[2*i+1:2*n], z[2*i+1:2*n], big.Word(hi)+c)
			_ = c
		}
	}
}
