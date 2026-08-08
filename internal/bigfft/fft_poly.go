package bigfft

import (
	"fmt"
	"math/big"
	"runtime"
	"sync"
)

// Poly represents an integer via a polynomial in Z[x]/(x^K+1)
// where K is the FFT length and b^m is the computation basis 1<<(m*_W).
// If P = a[0] + a[1] x + ... a[n] x^(K-1), the associated natural number
// is P(b^m).
type Poly struct {
	K uint  // K is such that 1<<K is the FFT length.
	M int   // the M such that P(b^M) is the original number.
	A []nat // a slice of at most 1<<K M-word coefficients.

	// pooledBacking, if non-nil, is the big.Word backing that should be
	// returned to the word-slice pool when Release() is called. This is set
	// by invTransform() which acquires the backing via acquireWordSliceUnsafe.
	// Left nil when Poly is constructed directly (e.g. polyFromNat) or copied
	// from non-pooled memory.
	pooledBacking []big.Word
	// pooledA indicates whether A itself was obtained from the []nat pool and
	// should be released on Release().
	pooledA bool
}

// Release returns any pooled backing buffers of p to their sync.Pool shards.
// Safe to call multiple times and on a zero-value Poly. After Release() the
// Poly must not be used again.
//
// This addresses audit finding P0-01: callers of Transform/InvTransform/Mul/Sqr
// previously leaked pool buffers because Poly/PolValues had no release API.
func (p *Poly) Release() {
	if p == nil {
		return
	}
	if p.pooledBacking != nil {
		releaseWordSlice(p.pooledBacking)
		p.pooledBacking = nil
	}
	if p.pooledA && p.A != nil {
		releaseNatSlice(p.A)
	}
	p.A = nil
	p.pooledA = false
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

// Int evaluates back a Poly to its integer value, as a big.Word magnitude
// slice in the layout expected by big.Int.SetBits.
func (p *Poly) Int() []big.Word {
	return p.IntTo(nil)
}

// IntTo evaluates back a Poly to its integer value, reusing the provided
// destination buffer if it has sufficient capacity. If dst is nil or too
// small, a new slice is allocated.
//
// This optimization reduces memory allocations when the caller already has
// a buffer that can be reused, which is common in iterative multiplication
// scenarios like Fibonacci calculations.
func (p *Poly) IntTo(dst []big.Word) []big.Word {
	length := len(p.A)*p.M + 1
	if na := len(p.A); na > 0 {
		length += len(p.A[na-1])
	}

	// Reuse dst if it has sufficient capacity, otherwise allocate new
	var n nat
	if cap(dst) >= length {
		n = dst[:length]
		clear(n)
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
//
// Test oracle: no production caller; cross-validated by fuzz/precision tests
// against MulWithBump and other multiplication paths (audit OVR-10). Kept.
func (p *Poly) Mul(q *Poly) (Poly, error) {
	return p.mul(q, defaultPoolAllocator)
}

// MulWithBump multiplies p and q using a bump allocator for temporary allocations.
// This provides better cache locality and reduces GC pressure.
func (p *Poly) MulWithBump(q *Poly, ba *BumpAllocator) (Poly, error) {
	return p.mul(q, ba)
}

func (p *Poly) mul(q *Poly, alloc tempAllocator) (Poly, error) {
	// extra=2 because:
	// * some power of 2 is a K-th root of unity when n is a multiple of K/2
	// * 2 itself is a square (see fermat.ShiftHalf)
	n := valueSize(p.K, p.M, 2)

	pv, err := p.transform(n, alloc)
	if err != nil {
		return Poly{}, err
	}
	// pv is not returned — release its pooled backing once mul() finishes.
	// invTransform below no longer needs pv after producing rv.
	defer pv.Release()

	qv, err := q.transform(n, alloc)
	if err != nil {
		return Poly{}, err
	}
	defer qv.Release()

	rv, err := pv.mul(&qv, alloc)
	if err != nil {
		return Poly{}, err
	}
	defer rv.Release()

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

	// pooledBacking, if non-nil, is the big.Word backing that should be
	// returned to the word-slice pool when Release() is called. This is set
	// only when PolValues was built from sync.Pool buffers (see transform(),
	// PolValues.mul, PolValues.sqr).
	//
	// It is deliberately left nil on cache hits (see TransformCache.getByKey)
	// so that callers that Release() a cache-shared PolValues do not poison
	// the pool.
	pooledBacking []big.Word
	// pooledValues indicates Values was obtained from the []fermat pool.
	pooledValues bool
}

// Release returns any pooled backing buffers of v to their sync.Pool shards.
// Safe to call on a zero-value PolValues and on cache-shared values
// (which have no pooled backing — Release() becomes a no-op).
// After Release() the PolValues must not be used again.
//
// This addresses audit finding P0-01/P0-09: Transform/InvTransform/Mul/Sqr
// previously leaked pool buffers because there was no release API.
func (v *PolValues) Release() {
	if v == nil {
		return
	}
	if v.pooledBacking != nil {
		releaseWordSlice(v.pooledBacking)
		v.pooledBacking = nil
	}
	if v.pooledValues && v.Values != nil {
		releaseFermatSlice(v.Values)
	}
	v.Values = nil
	v.pooledValues = false
}

// Transform evaluates p at θ^i for i = 0...K-1, where
// θ is a K-th primitive root of unity in Z/(b^n+1)Z.
//
// Test oracle: production code uses TransformWithBump/TransformCached*; this
// pool-backed variant is retained as the reference for cross-validation
// (audit OVR-10).
func (p *Poly) Transform(n int) (PolValues, error) {
	return p.transform(n, defaultPoolAllocator)
}

// TransformWithBump evaluates p at θ^i for i = 0...K-1, using a bump allocator
// for temporary allocations. This provides better cache locality and reduces
// GC pressure compared to Transform().
func (p *Poly) TransformWithBump(n int, ba *BumpAllocator) (PolValues, error) {
	return p.transform(n, ba)
}

func (p *Poly) transform(n int, alloc tempAllocator) (PolValues, error) {
	k := p.K
	K := 1 << k
	wordCount := (n + 1) * K

	// Use allocator for temporary input buffers
	input, _, cleanup := alloc.allocFermatSlice(K, n)
	defer cleanup()

	// Use pooled allocation for output buffers (contiguous backing array)
	valbits := acquireWordSliceUnsafe(wordCount)
	values := acquireFermatSlice(K)

	for i := 0; i < K; i++ {
		if i < len(p.A) {
			copy(input[i], p.A[i])
		}
		values[i] = fermat(valbits[i*(n+1) : (i+1)*(n+1)])
	}

	// Determine if we are using bump allocator to pass down
	if ba, ok := alloc.(*BumpAllocator); ok {
		if err := fourierWithBump(values, input, false, n, k, ba); err != nil {
			releaseWordSlice(valbits)
			releaseFermatSlice(values)
			return PolValues{}, err
		}
	} else {
		if err := fourier(values, input, false, n, k); err != nil {
			releaseWordSlice(valbits)
			releaseFermatSlice(values)
			return PolValues{}, err
		}
	}

	return PolValues{
		K:             k,
		N:             n,
		Values:        values,
		pooledBacking: valbits,
		pooledValues:  true,
	}, nil
}

// InvTransform reconstructs p (modulo X^K - 1) from its
// values at θ^i for i = 0..K-1.
func (v *PolValues) InvTransform() (Poly, error) {
	return v.invTransform(defaultPoolAllocator)
}

// InvTransformWithBump reconstructs p (modulo X^K - 1) from its values,
// using a bump allocator for temporary allocations.
func (v *PolValues) InvTransformWithBump(ba *BumpAllocator) (Poly, error) {
	return v.invTransform(ba)
}

func (v *PolValues) invTransform(alloc tempAllocator) (Poly, error) {
	k, n := v.K, v.N
	K := 1 << k
	wordCount := (n + 1) * K

	// Perform an inverse Fourier transform to recover p.
	// Use pooled allocation for output buffers (contiguous backing array).
	// pbits ends up shared with the returned Poly.A via a[i] = nat(p[i]).
	// p (the []fermat slice container) is only needed locally and can be
	// released before we return — its elements continue to reference pbits
	// through the nat slices in a.
	pbits := acquireWordSliceUnsafe(wordCount)
	p := acquireFermatSlice(K)
	defer releaseFermatSlice(p)
	for i := 0; i < K; i++ {
		p[i] = fermat(pbits[i*(n+1) : (i+1)*(n+1)])
	}

	// Determine if we are using bump allocator to pass down
	if ba, ok := alloc.(*BumpAllocator); ok {
		if err := fourierWithBump(p, v.Values, true, n, k, ba); err != nil {
			releaseWordSlice(pbits)
			return Poly{}, err
		}
	} else {
		if err := fourier(p, v.Values, true, n, k); err != nil {
			releaseWordSlice(pbits)
			return Poly{}, err
		}
	}

	// Divide by K, and untwist q to recover p.
	// Use allocator for temporary u
	u, cleanup := alloc.allocFermatTemp(n)
	defer cleanup()

	// Use pooled allocation for a
	a := acquireNatSlice(K)
	for i := 0; i < K; i++ {
		u.Shift(p[i], -int(k)) // #nosec G115 -- k is an FFT level (< 64 by construction), so int(k) cannot overflow
		copy(p[i], u)
		a[i] = nat(p[i])
	}

	return Poly{
		K:             k,
		M:             0,
		A:             a,
		pooledBacking: pbits,
		pooledA:       true,
	}, nil
}

// NTransform evaluates p at θω^i for i = 0...K-1, where
// θ is a (2K)-th primitive root of unity in Z/(b^n+1)Z
// and ω = θ².
//
// Returns an error if the underlying Fourier transform fails validation
// (e.g. malformed operand sizes). Callers previously silently discarded
// this error — see audit P2-12.
//
// Test oracle: no production caller (Mul/MulTo/Sqr/SqrTo route through
// TransformWithBump via executeDoublingStepFFT); retained as a
// cross-validation reference (audit OVR-10).
func (p *Poly) NTransform(n int) (PolValues, error) {
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
	wordCount := (n + 1) << k
	tbits := acquireWordSlice(wordCount)
	defer releaseWordSlice(tbits)

	K := 1 << k
	twisted := acquireFermatSlice(K)
	defer releaseFermatSlice(twisted)

	src := fermat(acquireWordSliceUnsafe(n + 1))
	defer releaseWordSlice([]big.Word(src))

	for i := range twisted {
		twisted[i] = fermat(tbits[i*(n+1) : (i+1)*(n+1)])
		if i < len(p.A) {
			clear(src)
			copy(src, p.A[i])
			twisted[i].Shift(src, θshift*i)
		}
	}

	// Now computed q(ω^i) for i = 0 ... K-1
	valbits := acquireWordSliceUnsafe(wordCount)
	values := acquireFermatSlice(K)
	for i := range values {
		values[i] = fermat(valbits[i*(n+1) : (i+1)*(n+1)])
	}
	if err := fourier(values, twisted, false, n, k); err != nil {
		releaseWordSlice(valbits)
		releaseFermatSlice(values)
		return PolValues{}, fmt.Errorf("NTransform: forward fourier failed: %w", err)
	}
	return PolValues{K: k, N: n, Values: values, pooledBacking: valbits, pooledValues: true}, nil
}

// InvNTransform reconstructs a polynomial from its values at
// roots of x^K+1. The M field of the returned polynomial
// is unspecified.
//
// Returns an error if the underlying inverse Fourier transform fails
// validation. Callers previously silently discarded this error — see
// audit P2-12.
//
// Test oracle: no production caller; pairs with NTransform as a
// round-trip cross-validation reference (audit OVR-10).
func (v *PolValues) InvNTransform() (Poly, error) {
	k := v.K
	n := v.N
	θshift := (n * _W) >> k

	// Perform an inverse Fourier transform to recover q.
	qbits := make([]big.Word, (n+1)<<k)
	q := make([]fermat, 1<<k)
	for i := range q {
		q[i] = fermat(qbits[i*(n+1) : (i+1)*(n+1)])
	}
	if err := fourier(q, v.Values, true, n, k); err != nil {
		return Poly{}, fmt.Errorf("InvNTransform: inverse fourier failed: %w", err)
	}

	// Divide by K, and untwist q to recover p.
	u := make(fermat, n+1)
	a := make([]nat, 1<<k)
	for i := range q {
		u.Shift(q[i], -int(k)-i*θshift) // #nosec G115 -- k is an FFT level (< 64 by construction), so int(k) cannot overflow
		copy(q[i], u)
		a[i] = nat(q[i])
	}
	return Poly{K: k, M: 0, A: a}, nil
}

// pointwiseMinParallelWords gates the parallel pointwise path: below this
// total output size (K*(n+1) words) goroutine dispatch and pool traffic
// cost more than the coefficient multiplications they spread. Selected by
// paired end-to-end benchmarks on a 24-thread host (2026-06): F(10M)-class
// inputs (~245k words) engage it, matrix-path coefficients stay far below.
const pointwiseMinParallelWords = 1 << 16

// runPointwise executes body(i, buf) for every i in [0, count). Each invocation
// must write only to destination index i (disjoint writes); buf is an
// 8*n-word fermat scratch reserved for the invocation's exclusive use.
//
// When the total output (count*(n+1) words) reaches pointwiseMinParallelWords,
// chunks of the index space run on extra goroutines bounded by the global
// FFT semaphore with a non-blocking acquire — the same contract as
// fourierRecursiveUnified: when no token is available the chunk simply runs
// on the calling goroutine, so this can never deadlock against the
// recursion's token usage. Parallel workers draw their scratch from the
// pool allocator because bump allocators are not thread-safe (same rule as
// the parallel recursion). Worker panics are captured and re-panicked in
// the calling goroutine so the public entry points' recover policy
// (ADR-0002 sentinel re-propagation) keeps applying unchanged.
func runPointwise(count, n int, alloc tempAllocator, body func(i int, buf fermat)) {
	if count*(n+1) < pointwiseMinParallelWords || runtime.NumCPU() == 1 {
		buf, cleanup := alloc.allocFermatTemp(8 * n)
		defer cleanup()
		for i := 0; i < count; i++ {
			body(i, buf)
		}
		return
	}

	workers := runtime.NumCPU()
	if workers > count {
		workers = count
	}
	chunk := (count + workers - 1) / workers

	sem := getSemaphore()
	var wg sync.WaitGroup
	panicCh := make(chan any, 1)

	runChunk := func(lo, hi int) {
		buf, cleanup := defaultPoolAllocator.allocFermatTemp(8 * n)
		defer cleanup()
		for i := lo; i < hi; i++ {
			body(i, buf)
		}
	}

	// Chunk 0 is reserved for the calling goroutine (processed below with
	// the caller-provided allocator); the rest spawn when a token is free.
	for lo := chunk; lo < count; lo += chunk {
		hi := lo + chunk
		if hi > count {
			hi = count
		}
		select {
		case sem <- struct{}{}:
			wg.Add(1)
			go func(lo, hi int) {
				defer wg.Done()
				defer func() { <-sem }()
				defer func() {
					if r := recover(); r != nil {
						select {
						case panicCh <- r:
						default:
						}
					}
				}()
				runChunk(lo, hi)
			}(lo, hi)
		default:
			runChunk(lo, hi)
		}
	}

	// Chunk 0 runs wrapped so a panic here still lets wg.Wait() run below
	// instead of unwinding straight past it — otherwise a spawned worker could
	// still be reading/writing shared buffers when the caller returns (FFT-02).
	var rSync any
	func() {
		defer func() { rSync = recover() }()
		buf, cleanup := alloc.allocFermatTemp(8 * n)
		defer cleanup()
		hi := chunk
		if hi > count {
			hi = count
		}
		for i := 0; i < hi; i++ {
			body(i, buf)
		}
	}()

	wg.Wait()
	if rSync != nil {
		panic(rSync)
	}
	select {
	case r := <-panicCh:
		panic(r)
	default:
	}
}

// Mul returns the pointwise product of v and q.
func (v *PolValues) Mul(q *PolValues) (PolValues, error) {
	return v.mul(q, defaultPoolAllocator)
}

// MulWithBump returns the pointwise product of v and q, using a bump allocator
// for temporary buffers.
func (v *PolValues) MulWithBump(q *PolValues, ba *BumpAllocator) (PolValues, error) {
	return v.mul(q, ba)
}

func (v *PolValues) mul(q *PolValues, alloc tempAllocator) (PolValues, error) {
	n := v.N
	K := len(v.Values)
	var r PolValues
	r.K, r.N = v.K, v.N

	// Use pooled allocation for returned data (contiguous backing array)
	r.Values = acquireFermatSlice(K)
	wordCount := K * (n + 1)
	bits := acquireWordSliceUnsafe(wordCount)
	for i := 0; i < K; i++ {
		r.Values[i] = bits[i*(n+1) : (i+1)*(n+1)]
	}

	// The K coefficient products are independent (disjoint destinations);
	// runPointwise spreads them across cores for large transforms. The
	// scratch buffer needs 8*n words, consistent with the historical code.
	runPointwise(K, n, alloc, func(i int, buf fermat) {
		z := buf.Mul(v.Values[i], q.Values[i])
		copy(r.Values[i], z)
	})

	r.pooledBacking = bits
	r.pooledValues = true
	return r, nil
}

// Sqr returns the pointwise square of v (v[i] * v[i] for each i).
// This is optimized for squaring as we don't need a second set of values.
func (v *PolValues) Sqr() (PolValues, error) {
	return v.sqr(defaultPoolAllocator)
}

// SqrWithBump returns the pointwise square of v, using a bump allocator
// for temporary buffers.
func (v *PolValues) SqrWithBump(ba *BumpAllocator) (PolValues, error) {
	return v.sqr(ba)
}

func (v *PolValues) sqr(alloc tempAllocator) (PolValues, error) {
	n := v.N
	K := len(v.Values)
	var r PolValues
	r.K, r.N = v.K, v.N

	// Use pooled allocation for returned data (contiguous backing array)
	r.Values = acquireFermatSlice(K)
	wordCount := K * (n + 1)
	bits := acquireWordSliceUnsafe(wordCount)
	for i := 0; i < K; i++ {
		r.Values[i] = bits[i*(n+1) : (i+1)*(n+1)]
	}

	// Same parallel dispatch as mul; Sqr is the specialized squaring.
	runPointwise(K, n, alloc, func(i int, buf fermat) {
		z := buf.Sqr(v.Values[i])
		copy(r.Values[i], z)
	})

	r.pooledBacking = bits
	r.pooledValues = true
	return r, nil
}

// Clone creates a deep copy of PolValues to allow safe concurrent usage.
// This is essential when the same transformed polynomial needs to be used
// in multiple goroutines simultaneously (e.g., for both Mul and Sqr operations).
//
// Test oracle: no production caller today; retained for tests that need an
// independent, aliasing-free copy to diff against a mutated original
// (audit OVR-10).
func (v *PolValues) Clone() PolValues {
	K := len(v.Values)
	n := v.N
	wordCount := K * (n + 1)

	// Allocate new backing array and values slice
	bits := make([]big.Word, wordCount)
	values := make([]fermat, K)

	for i := 0; i < K; i++ {
		values[i] = fermat(bits[i*(n+1) : (i+1)*(n+1)])
		copy(values[i], v.Values[i])
	}

	return PolValues{
		K:      v.K,
		N:      v.N,
		Values: values,
	}
}
