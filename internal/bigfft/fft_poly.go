// Package bigfft implements multiplication of big.Int using FFT.
package bigfft

import (
	"math/big"
)

// Poly represents an integer via a polynomial in Z[x]/(x^K+1)
// where K is the FFT length and b^m is the computation basis 1<<(m*_W).
// If P = a[0] + a[1] x + ... a[n] x^(K-1), the associated natural number
// is P(b^m).
type Poly struct {
	K uint  // K is such that 1<<K is the FFT length.
	M int   // the M such that P(b^M) is the original number.
	A []nat // a slice of at most 1<<K M-word coefficients.
}

// polyFromNat slices the number x into a Polynomial
// with 1<<k coefficients made of m words.
func polyFromNat(x nat, k uint, m int) Poly {
	p := Poly{K: k, M: m}
	// Calculate exact length needed to avoid over-allocation
	// We need ceil(len(x) / m) coefficients
	length := (len(x) + m - 1) / m
	if length == 0 {
		length = 1 // At least one coefficient for zero
	}
	p.A = make([]nat, length)
	for i := range p.A {
		if len(x) < m {
			p.A[i] = make(nat, m)
			copy(p.A[i], x)
			break
		}
		p.A[i] = x[:m]
		x = x[m:]
	}
	return p
}

// IntToBigInt converts the Poly back to a *big.Int, reusing its buffer if possible.
func (p *Poly) IntToBigInt(z *big.Int) *big.Int {
	zb := p.IntTo(z.Bits())
	z.SetBits(zb)
	return z
}

// Int evaluates back a Poly to its integer value.
func (p *Poly) Int() nat {
	return p.IntTo(nil)
}

// IntTo evaluates back a Poly to its integer value, reusing the provided
// destination buffer if it has sufficient capacity. If dst is nil or too
// small, a new slice is allocated.
//
// This optimization reduces memory allocations when the caller already has
// a buffer that can be reused, which is common in iterative multiplication
// scenarios like Fibonacci calculations.
func (p *Poly) IntTo(dst nat) nat {
	length := len(p.A)*p.M + 1
	if na := len(p.A); na > 0 {
		length += len(p.A[na-1])
	}

	// Reuse dst if it has sufficient capacity, otherwise allocate new
	var n nat
	if cap(dst) >= length {
		n = dst[:length]
		// Clear the buffer before use
		for i := range n {
			n[i] = 0
		}
	} else {
		n = make(nat, length)
	}

	m := p.M
	np := n
	for i := range p.A {
		l := len(p.A[i])
		c := addVV(np[:l], np[:l], p.A[i])
		if np[l] < ^big.Word(0) {
			np[l] += c
		} else {
			addVW(np[l:], np[l:], c)
		}
		np = np[m:]
	}
	n = trim(n)
	return n
}

func trim(n nat) nat {
	for i := range n {
		if n[len(n)-1-i] != 0 {
			return n[:len(n)-i]
		}
	}
	return nil
}

// Mul multiplies p and q modulo X^K-1, where K = 1<<p.K.
// The product is done via a Fourier transform.
func (p *Poly) Mul(q *Poly) (Poly, error) {
	return p.mul(q, GetPoolAllocator())
}

// MulWithBump multiplies p and q using a bump allocator for temporary allocations.
// This provides better cache locality and reduces GC pressure.
func (p *Poly) MulWithBump(q *Poly, ba *BumpAllocator) (Poly, error) {
	return p.mul(q, NewBumpAllocatorAdapter(ba))
}

func (p *Poly) mul(q *Poly, alloc TempAllocator) (Poly, error) {
	// extra=2 because:
	// * some power of 2 is a K-th root of unity when n is a multiple of K/2
	// * 2 itself is a square (see fermat.ShiftHalf)
	n := valueSize(p.K, p.M, 2)

	pv, err := p.transform(n, alloc)
	if err != nil {
		return Poly{}, err
	}
	qv, err := q.transform(n, alloc)
	if err != nil {
		return Poly{}, err
	}
	rv, err := pv.mul(&qv, alloc)
	if err != nil {
		return Poly{}, err
	}
	r, err := rv.invTransform(alloc)
	if err != nil {
		return Poly{}, err
	}
	r.M = p.M
	return r, nil
}

// A PolValues represents the value of a Poly at the powers of a
// K-th root of unity θ=2^(l/2) in Z/(b^n+1)Z, where b^n = 2^(K/4*l).
type PolValues struct {
	K      uint     // K is such that 1<<K is the FFT length.
	N      int      // the length of coefficients, n*_W a multiple of K/4.
	Values []fermat // a slice of 1<<K (n+1)-word values
}

// Transform evaluates p at θ^i for i = 0...K-1, where
// θ is a K-th primitive root of unity in Z/(b^n+1)Z.
func (p *Poly) Transform(n int) (PolValues, error) {
	return p.transform(n, GetPoolAllocator())
}

// TransformWithBump evaluates p at θ^i for i = 0...K-1, using a bump allocator
// for temporary allocations. This provides better cache locality and reduces
// GC pressure compared to Transform().
func (p *Poly) TransformWithBump(n int, ba *BumpAllocator) (PolValues, error) {
	return p.transform(n, NewBumpAllocatorAdapter(ba))
}

func (p *Poly) transform(n int, alloc TempAllocator) (PolValues, error) {
	k := p.K
	K := 1 << k
	wordCount := (n + 1) * K

	// Use allocator for temporary input buffers
	input, _, cleanup := alloc.AllocFermatSlice(K, n)
	defer cleanup()

	// Use regular allocation for output buffers (they are returned and cannot be pooled)
	valbits := make([]big.Word, wordCount)
	values := make([]fermat, K)

	for i := 0; i < K; i++ {
		if i < len(p.A) {
			copy(input[i], p.A[i])
		}
		values[i] = fermat(valbits[i*(n+1) : (i+1)*(n+1)])
	}

	// Determine if we are using bump allocator to pass down
	var ba *BumpAllocator
	if adapter, ok := alloc.(*BumpAllocatorAdapter); ok {
		ba = adapter.ba
	}

	if ba != nil {
		if err := fourierWithBump(values, input, false, n, k, ba); err != nil {
			return PolValues{}, err
		}
	} else {
		if err := fourier(values, input, false, n, k); err != nil {
			return PolValues{}, err
		}
	}

	return PolValues{k, n, values}, nil
}

// InvTransform reconstructs p (modulo X^K - 1) from its
// values at θ^i for i = 0..K-1.
func (v *PolValues) InvTransform() (Poly, error) {
	return v.invTransform(GetPoolAllocator())
}

// InvTransformWithBump reconstructs p (modulo X^K - 1) from its values,
// using a bump allocator for temporary allocations.
func (v *PolValues) InvTransformWithBump(ba *BumpAllocator) (Poly, error) {
	return v.invTransform(NewBumpAllocatorAdapter(ba))
}

func (v *PolValues) invTransform(alloc TempAllocator) (Poly, error) {
	k, n := v.K, v.N
	K := 1 << k
	wordCount := (n + 1) * K

	// Perform an inverse Fourier transform to recover p.
	// Use regular allocation since pbits data is returned via a[i]
	pbits := make([]big.Word, wordCount)
	p := make([]fermat, K)
	for i := 0; i < K; i++ {
		p[i] = fermat(pbits[i*(n+1) : (i+1)*(n+1)])
	}

	// Determine if we are using bump allocator to pass down
	var ba *BumpAllocator
	if adapter, ok := alloc.(*BumpAllocatorAdapter); ok {
		ba = adapter.ba
	}

	if ba != nil {
		if err := fourierWithBump(p, v.Values, true, n, k, ba); err != nil {
			return Poly{}, err
		}
	} else {
		if err := fourier(p, v.Values, true, n, k); err != nil {
			return Poly{}, err
		}
	}

	// Divide by K, and untwist q to recover p.
	// Use allocator for temporary u
	u, cleanup := alloc.AllocFermatTemp(n)
	defer cleanup()

	// Use regular allocation for a since it's returned
	a := make([]nat, K)
	for i := 0; i < K; i++ {
		u.Shift(p[i], -int(k))
		copy(p[i], u)
		a[i] = nat(p[i])
	}

	return Poly{K: k, M: 0, A: a}, nil
}

// NTransform evaluates p at θω^i for i = 0...K-1, where
// θ is a (2K)-th primitive root of unity in Z/(b^n+1)Z
// and ω = θ².
func (p *Poly) NTransform(n int) PolValues {
	k := p.K
	if len(p.A) >= 1<<k {
		panic("Transform: len(p.A) >= 1<<k")
	}
	// θ is represented as a shift.
	θshift := (n * _W) >> k
	// p(x) = a_0 + a_1 x + ... + a_{K-1} x^(K-1)
	// p(θx) = q(x) where
	// q(x) = a_0 + θa_1 x + ... + θ^(K-1) a_{K-1} x^(K-1)
	//
	// Twist p by θ to obtain q.
	tbits := make([]big.Word, (n+1)<<k)
	twisted := make([]fermat, 1<<k)
	src := make(fermat, n+1)
	for i := range twisted {
		twisted[i] = fermat(tbits[i*(n+1) : (i+1)*(n+1)])
		if i < len(p.A) {
			for i := range src {
				src[i] = 0
			}
			copy(src, p.A[i])
			twisted[i].Shift(src, θshift*i)
		}
	}

	// Now computed q(ω^i) for i = 0 ... K-1
	valbits := make([]big.Word, (n+1)<<k)
	values := make([]fermat, 1<<k)
	for i := range values {
		values[i] = fermat(valbits[i*(n+1) : (i+1)*(n+1)])
	}
	fourier(values, twisted, false, n, k)
	return PolValues{k, n, values}
}

// InvNTransform reconstructs a polynomial from its values at
// roots of x^K+1. The M field of the returned polynomial
// is unspecified.
func (v *PolValues) InvNTransform() Poly {
	k := v.K
	n := v.N
	θshift := (n * _W) >> k

	// Perform an inverse Fourier transform to recover q.
	qbits := make([]big.Word, (n+1)<<k)
	q := make([]fermat, 1<<k)
	for i := range q {
		q[i] = fermat(qbits[i*(n+1) : (i+1)*(n+1)])
	}
	fourier(q, v.Values, true, n, k)

	// Divide by K, and untwist q to recover p.
	u := make(fermat, n+1)
	a := make([]nat, 1<<k)
	for i := range q {
		u.Shift(q[i], -int(k)-i*θshift)
		copy(q[i], u)
		a[i] = nat(q[i])
	}
	return Poly{K: k, M: 0, A: a}
}

// Mul returns the pointwise product of p and q.
func (p *PolValues) Mul(q *PolValues) (PolValues, error) {
	return p.mul(q, GetPoolAllocator())
}

// MulWithBump returns the pointwise product of p and q, using a bump allocator
// for temporary buffers.
func (p *PolValues) MulWithBump(q *PolValues, ba *BumpAllocator) (PolValues, error) {
	return p.mul(q, NewBumpAllocatorAdapter(ba))
}

func (p *PolValues) mul(q *PolValues, alloc TempAllocator) (PolValues, error) {
	n := p.N
	K := len(p.Values)
	var r PolValues
	r.K, r.N = p.K, p.N

	// Use regular allocation for returned data
	r.Values = make([]fermat, K)
	wordCount := K * (n + 1)
	bits := make([]big.Word, wordCount)

	// Use allocator for temporary multiplication result
	// The temporary buffer needs to be 8*n (or 8*n - 1 if optimized)
	// We use 8*n to be safe and consistent with previous code
	buf, cleanup := alloc.AllocFermatTemp(8 * n)
	defer cleanup()

	for i := 0; i < K; i++ {
		r.Values[i] = bits[i*(n+1) : (i+1)*(n+1)]
		z := buf.Mul(p.Values[i], q.Values[i])
		copy(r.Values[i], z)
	}

	return r, nil
}

// Sqr returns the pointwise square of p (p[i] * p[i] for each i).
// This is optimized for squaring as we don't need a second set of values.
func (p *PolValues) Sqr() (PolValues, error) {
	return p.sqr(GetPoolAllocator())
}

// SqrWithBump returns the pointwise square of p, using a bump allocator
// for temporary buffers.
func (p *PolValues) SqrWithBump(ba *BumpAllocator) (PolValues, error) {
	return p.sqr(NewBumpAllocatorAdapter(ba))
}

func (p *PolValues) sqr(alloc TempAllocator) (PolValues, error) {
	n := p.N
	K := len(p.Values)
	var r PolValues
	r.K, r.N = p.K, p.N

	// Use regular allocation for returned data
	r.Values = make([]fermat, K)
	wordCount := K * (n + 1)
	bits := make([]big.Word, wordCount)

	// Use allocator for temporary multiplication result
	buf, cleanup := alloc.AllocFermatTemp(8 * n)
	defer cleanup()

	for i := 0; i < K; i++ {
		r.Values[i] = bits[i*(n+1) : (i+1)*(n+1)]
		// Square: multiply p.Values[i] by itself
		z := buf.Mul(p.Values[i], p.Values[i])
		copy(r.Values[i], z)
	}

	return r, nil
}

// Clone creates a deep copy of PolValues to allow safe concurrent usage.
// This is essential when the same transformed polynomial needs to be used
// in multiple goroutines simultaneously (e.g., for both Mul and Sqr operations).
func (p *PolValues) Clone() PolValues {
	K := len(p.Values)
	n := p.N
	wordCount := K * (n + 1)

	// Allocate new backing array and values slice
	bits := make([]big.Word, wordCount)
	values := make([]fermat, K)

	for i := 0; i < K; i++ {
		values[i] = fermat(bits[i*(n+1) : (i+1)*(n+1)])
		copy(values[i], p.Values[i])
	}

	return PolValues{
		K:      p.K,
		N:      p.N,
		Values: values,
	}
}
