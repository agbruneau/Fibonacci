// Package bigfft implements multiplication of big.Int using FFT.
//
// The implementation is based on the Schönhage-Strassen method
// using integer FFT modulo 2^n+1.
package bigfft

import (
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"unsafe"
)

// concurrencySemaphore is a buffered channel used to limit the number of
// concurrent goroutines in the FFT recursion.
var concurrencySemaphore chan struct{}
var concurrencyOnce sync.Once

// getSemaphore returns the global concurrency semaphore, initializing it
// to runtime.NumCPU() on the first call.
func getSemaphore() chan struct{} {
	concurrencyOnce.Do(func() {
		concurrencySemaphore = make(chan struct{}, runtime.NumCPU())
	})
	return concurrencySemaphore
}

const _W = int(unsafe.Sizeof(big.Word(0)) * 8)

// ParallelFFTRecursionThreshold is the minimum size (in bits of k, where K=2^k)
// for which FFT recursion should be parallelized. Below this threshold, the
// overhead of goroutine creation exceeds the benefits of parallelism.
const ParallelFFTRecursionThreshold uint = 4

// MaxParallelFFTDepth limits the maximum depth of parallel recursion to avoid
// excessive goroutine creation. Once this depth is reached, recursion continues
// sequentially.
const MaxParallelFFTDepth uint = 3

type nat []big.Word

func (n nat) String() string {
	v := new(big.Int)
	v.SetBits(n)
	return v.String()
}

// defaultFFTThresholdWords is the default size (in words) above which FFT is
// used over Karatsuba from math/big.
//
// TestCalibrate seems to indicate a threshold of 60kbits on 32-bit
// arches and 110kbits on 64-bit arches. The value 1800 words corresponds to
// approximately 115kbits on 64-bit systems (1800 * 64 = 115200 bits).
const defaultFFTThresholdWords = 1800

// fftThreshold is the size (in words) above which FFT is used over
// Karatsuba from math/big. This can be modified for tuning purposes.
var fftThreshold = defaultFFTThresholdWords

// Mul computes the product x*y and returns z.
// It can be used instead of the Mul method of
// *big.Int from math/big package.
func Mul(x, y *big.Int) (*big.Int, error) {
	xwords := len(x.Bits())
	ywords := len(y.Bits())
	if xwords > fftThreshold && ywords > fftThreshold {
		return mulFFT(x, y)
	}
	return new(big.Int).Mul(x, y), nil
}

// MulTo computes the product x*y and stores the result in z.
// This allows reusing the allocated memory of z, which is more
// efficient than Mul when z is already allocated and large enough.
//
// Optimization: When FFT multiplication is used, the existing buffer of z
// is passed through to the final IntTo() call, potentially avoiding a
// large allocation if z already has sufficient capacity.
// MulTo computes the product x*y and stores the result in z.
// This allows reusing the allocated memory of z, which is more
// efficient than Mul when z is already allocated and large enough.
//
// Optimization: When FFT multiplication is used, the existing buffer of z
// is passed through to the final IntTo() call, potentially avoiding a
// large allocation if z already has sufficient capacity.
func MulTo(z, x, y *big.Int) (*big.Int, error) {
	xwords := len(x.Bits())
	ywords := len(y.Bits())
	if xwords > fftThreshold && ywords > fftThreshold {
		var xb, yb nat = x.Bits(), y.Bits()
		// Reuse z's existing buffer if available
		zb, err := fftmulTo(z.Bits(), xb, yb)
		if err != nil {
			return nil, err
		}
		z.SetBits(zb)
		if x.Sign()*y.Sign() < 0 {
			z.Neg(z)
		}
		return z, nil
	}
	return z.Mul(x, y), nil
}

// Sqr computes x*x and returns the result as a new *big.Int.
// Squaring is optimized because we only need to transform x once,
// which saves approximately 33% of the FFT computation compared to Mul.
func Sqr(x *big.Int) (*big.Int, error) {
	xwords := len(x.Bits())
	if xwords > fftThreshold {
		return sqrFFT(x)
	}
	return new(big.Int).Mul(x, x), nil
}

// SqrTo computes x*x and stores the result in z.
// This allows reusing the allocated memory of z, which is more
// efficient than Sqr when z is already allocated and large enough.
//
// Optimization: Squaring only requires one FFT transform instead of two,
// saving approximately 33% of the computation time for large numbers.
// SqrTo computes x*x and stores the result in z.
// This allows reusing the allocated memory of z, which is more
// efficient than Sqr when z is already allocated and large enough.
//
// Optimization: Squaring only requires one FFT transform instead of two,
// saving approximately 33% of the computation time for large numbers.
func SqrTo(z, x *big.Int) (*big.Int, error) {
	xwords := len(x.Bits())
	if xwords > fftThreshold {
		var xb nat = x.Bits()
		zb, err := fftsqrTo(z.Bits(), xb)
		if err != nil {
			return nil, err
		}
		z.SetBits(zb)
		// x*x is always non-negative, no sign handling needed
		return z, nil
	}
	return z.Mul(x, x), nil
}

func sqrFFT(x *big.Int) (*big.Int, error) {
	var xb nat = x.Bits()
	zb, err := fftsqr(xb)
	if err != nil {
		return nil, err
	}
	z := new(big.Int)
	z.SetBits(zb)
	// x*x is always non-negative
	return z, nil
}

func fftsqr(x nat) (nat, error) {
	return fftsqrTo(nil, x)
}

// fftsqrTo performs FFT squaring of x, reusing dst as the destination buffer
// if it has sufficient capacity. This is optimized compared to fftmulTo
// because we only need to transform x once.
//
// Uses a bump allocator for temporary allocations to minimize GC pressure
// and improve cache locality during the FFT computation.
func fftsqrTo(dst, x nat) (nat, error) {
	k, m := fftSizeSqr(x)

	// Estimate and acquire bump allocator for temporary allocations
	wordLen := 2 * len(x)
	ba := AcquireBumpAllocator(EstimateBumpCapacity(wordLen))
	defer ReleaseBumpAllocator(ba)

	xp := polyFromNat(x, k, m)
	n := valueSize(k, m, 2)
	xv, err := xp.TransformWithBump(n, ba)
	if err != nil {
		return nil, err
	}
	rv, err := xv.SqrWithBump(ba) // Pointwise squaring - no need for second transform
	if err != nil {
		return nil, err
	}
	r, err := rv.InvTransformWithBump(ba)
	if err != nil {
		return nil, err
	}
	r.m = m
	return r.IntTo(dst), nil
}

// fftSizeSqr returns the FFT parameters for squaring x.
// For squaring, the result size is 2*len(x) words.
func fftSizeSqr(x nat) (k uint, m int) {
	words := 2 * len(x) // x*x has at most 2*len(x) words
	bits := int64(words) * int64(_W)
	k = uint(len(fftSizeThreshold))
	for i := range fftSizeThreshold {
		if fftSizeThreshold[i] > bits {
			k = uint(i)
			break
		}
	}
	m = words>>k + 1
	return
}

func mulFFT(x, y *big.Int) (*big.Int, error) {
	var xb, yb nat = x.Bits(), y.Bits()
	zb, err := fftmul(xb, yb)
	if err != nil {
		return nil, err
	}
	z := new(big.Int)
	z.SetBits(zb)
	if x.Sign()*y.Sign() < 0 {
		z.Neg(z)
	}
	return z, nil
}

// A FFT size of K=1<<k is adequate when K is about 2*sqrt(N) where
// N = x.Bitlen() + y.Bitlen().

func fftmul(x, y nat) (nat, error) {
	return fftmulTo(nil, x, y)
}

// fftmulTo performs FFT multiplication of x and y, reusing dst as the
// destination buffer if it has sufficient capacity. This reduces allocations
// in iterative multiplication scenarios.
//
// Uses a bump allocator for temporary allocations to minimize GC pressure
// and improve cache locality during the FFT computation.
func fftmulTo(dst, x, y nat) (nat, error) {
	k, m := fftSize(x, y)

	// Estimate and acquire bump allocator for temporary allocations
	wordLen := len(x) + len(y)
	ba := AcquireBumpAllocator(EstimateBumpCapacity(wordLen))
	defer ReleaseBumpAllocator(ba)

	xp := polyFromNat(x, k, m)
	yp := polyFromNat(y, k, m)
	rp, err := xp.MulWithBump(&yp, ba)
	if err != nil {
		return nil, err
	}
	return rp.IntTo(dst), nil
}

// fftSizeThreshold[i] is the maximal size (in bits) where we should use
// fft size i.
var fftSizeThreshold = [...]int64{0, 0, 0,
	4 << 10, 8 << 10, 16 << 10, // 5
	32 << 10, 64 << 10, 1 << 18, 1 << 20, 3 << 20, // 10
	8 << 20, 30 << 20, 100 << 20, 300 << 20, 600 << 20,
}

// returns the FFT length k, m the number of words per chunk
// such that m << k is larger than the number of words
// in x*y.
func fftSize(x, y nat) (k uint, m int) {
	words := len(x) + len(y)
	bits := int64(words) * int64(_W)
	k = uint(len(fftSizeThreshold))
	for i := range fftSizeThreshold {
		if fftSizeThreshold[i] > bits {
			k = uint(i)
			break
		}
	}
	// The 1<<k chunks of m words must have N bits so that
	// 2^N-1 is larger than x*y. That is, m<<k > words
	m = words>>k + 1
	return
}

// valueSize returns the length (in words) to use for polynomial
// coefficients, to compute a correct product of polynomials P*Q
// where deg(P*Q) < K (== 1<<k) and where coefficients of P and Q are
// less than b^m (== 1 << (m*_W)).
// The chosen length (in bits) must be a multiple of 1 << (k-extra).
func valueSize(k uint, m int, extra uint) int {
	// The coefficients of P*Q are less than b^(2m)*K
	// so we need W * valueSize >= 2*m*W+K
	n := 2*m*_W + int(k) // necessary bits
	K := 1 << (k - extra)
	if K < _W {
		K = _W
	}
	n = ((n / K) + 1) * K // round to a multiple of K
	return n / _W
}

// poly represents an integer via a polynomial in Z[x]/(x^K+1)
// where K is the FFT length and b^m is the computation basis 1<<(m*_W).
// If P = a[0] + a[1] x + ... a[n] x^(K-1), the associated natural number
// is P(b^m).
type poly struct {
	k uint  // k is such that K = 1<<k.
	m int   // the m such that P(b^m) is the original number.
	a []nat // a slice of at most K m-word coefficients.
}

// polyFromNat slices the number x into a polynomial
// with 1<<k coefficients made of m words.
func polyFromNat(x nat, k uint, m int) poly {
	p := poly{k: k, m: m}
	// Calculate exact length needed to avoid over-allocation
	// We need ceil(len(x) / m) coefficients
	length := (len(x) + m - 1) / m
	if length == 0 {
		length = 1 // At least one coefficient for zero
	}
	p.a = make([]nat, length)
	for i := range p.a {
		if len(x) < m {
			p.a[i] = make(nat, m)
			copy(p.a[i], x)
			break
		}
		p.a[i] = x[:m]
		x = x[m:]
	}
	return p
}

// Int evaluates back a poly to its integer value.
func (p *poly) Int() nat {
	return p.IntTo(nil)
}

// IntTo evaluates back a poly to its integer value, reusing the provided
// destination buffer if it has sufficient capacity. If dst is nil or too
// small, a new slice is allocated.
//
// This optimization reduces memory allocations when the caller already has
// a buffer that can be reused, which is common in iterative multiplication
// scenarios like Fibonacci calculations.
func (p *poly) IntTo(dst nat) nat {
	length := len(p.a)*p.m + 1
	if na := len(p.a); na > 0 {
		length += len(p.a[na-1])
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

	m := p.m
	np := n
	for i := range p.a {
		l := len(p.a[i])
		c := addVV(np[:l], np[:l], p.a[i])
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

// Mul multiplies p and q modulo X^K-1, where K = 1<<p.k.
// The product is done via a Fourier transform.
func (p *poly) Mul(q *poly) (poly, error) {
	// extra=2 because:
	// * some power of 2 is a K-th root of unity when n is a multiple of K/2
	// * 2 itself is a square (see fermat.ShiftHalf)
	n := valueSize(p.k, p.m, 2)

	pv, err := p.Transform(n)
	if err != nil {
		return poly{}, err
	}
	qv, err := q.Transform(n)
	if err != nil {
		return poly{}, err
	}
	rv, err := pv.Mul(&qv)
	if err != nil {
		return poly{}, err
	}
	r, err := rv.InvTransform()
	if err != nil {
		return poly{}, err
	}
	r.m = p.m
	return r, nil
}

// MulWithBump multiplies p and q using a bump allocator for temporary allocations.
// This provides better cache locality and reduces GC pressure.
func (p *poly) MulWithBump(q *poly, ba *BumpAllocator) (poly, error) {
	n := valueSize(p.k, p.m, 2)

	pv, err := p.TransformWithBump(n, ba)
	if err != nil {
		return poly{}, err
	}
	qv, err := q.TransformWithBump(n, ba)
	if err != nil {
		return poly{}, err
	}
	rv, err := pv.MulWithBump(&qv, ba)
	if err != nil {
		return poly{}, err
	}
	r, err := rv.InvTransformWithBump(ba)
	if err != nil {
		return poly{}, err
	}
	r.m = p.m
	return r, nil
}

// A polValues represents the value of a poly at the powers of a
// K-th root of unity θ=2^(l/2) in Z/(b^n+1)Z, where b^n = 2^(K/4*l).
type polValues struct {
	k      uint     // k is such that K = 1<<k.
	n      int      // the length of coefficients, n*_W a multiple of K/4.
	values []fermat // a slice of K (n+1)-word values
}

// Transform evaluates p at θ^i for i = 0...K-1, where
// θ is a K-th primitive root of unity in Z/(b^n+1)Z.
func (p *poly) Transform(n int) (polValues, error) {
	k := p.k
	K := 1 << k
	wordCount := (n + 1) * K

	// Use pooled slices for temporary input buffers
	inputbits := acquireWordSlice(wordCount)
	input := acquireFermatSlice(K)

	// Use regular allocation for output buffers (they are returned and cannot be pooled)
	valbits := make([]big.Word, wordCount)
	values := make([]fermat, K)

	for i := 0; i < K; i++ {
		input[i] = inputbits[i*(n+1) : (i+1)*(n+1)]
		if i < len(p.a) {
			copy(input[i], p.a[i])
		}
		values[i] = fermat(valbits[i*(n+1) : (i+1)*(n+1)])
	}
	if err := fourier(values, input, false, n, k); err != nil {
		return polValues{}, err
	}

	// Release temporary input buffers
	releaseWordSlice(inputbits)
	releaseFermatSlice(input)

	return polValues{k, n, values}, nil
}

// TransformWithBump evaluates p at θ^i for i = 0...K-1, using a bump allocator
// for temporary allocations. This provides better cache locality and reduces
// GC pressure compared to Transform().
func (p *poly) TransformWithBump(n int, ba *BumpAllocator) (polValues, error) {
	k := p.k
	K := 1 << k
	wordCount := (n + 1) * K

	// Use bump allocator for temporary input buffers
	input, _ := ba.AllocFermatSlice(K, n)

	// Use regular allocation for output buffers (they are returned and cannot be pooled)
	valbits := make([]big.Word, wordCount)
	values := make([]fermat, K)

	for i := 0; i < K; i++ {
		if i < len(p.a) {
			copy(input[i], p.a[i])
		}
		values[i] = fermat(valbits[i*(n+1) : (i+1)*(n+1)])
	}
	if err := fourierWithBump(values, input, false, n, k, ba); err != nil {
		return polValues{}, err
	}

	// No need to release - bump allocator handles all temp memory
	return polValues{k, n, values}, nil
}

// InvTransform reconstructs p (modulo X^K - 1) from its
// values at θ^i for i = 0..K-1.
func (v *polValues) InvTransform() (poly, error) {
	k, n := v.k, v.n
	K := 1 << k
	wordCount := (n + 1) * K

	// Perform an inverse Fourier transform to recover p.
	// Use regular allocation since pbits data is returned via a[i]
	pbits := make([]big.Word, wordCount)
	p := make([]fermat, K)
	for i := 0; i < K; i++ {
		p[i] = fermat(pbits[i*(n+1) : (i+1)*(n+1)])
	}
	if err := fourier(p, v.values, true, n, k); err != nil {
		return poly{}, err
	}

	// Divide by K, and untwist q to recover p.
	// Use pooled buffer for temporary u
	u := acquireFermat(n + 1)
	// Use regular allocation for a since it's returned
	a := make([]nat, K)
	for i := 0; i < K; i++ {
		u.Shift(p[i], -int(k))
		copy(p[i], u)
		a[i] = nat(p[i])
	}

	// Release temporary buffer
	releaseFermat(u)

	return poly{k: k, m: 0, a: a}, nil
}

// InvTransformWithBump reconstructs p (modulo X^K - 1) from its values,
// using a bump allocator for temporary allocations.
func (v *polValues) InvTransformWithBump(ba *BumpAllocator) (poly, error) {
	k, n := v.k, v.n
	K := 1 << k
	wordCount := (n + 1) * K

	// Perform an inverse Fourier transform to recover p.
	// Use regular allocation since pbits data is returned via a[i]
	pbits := make([]big.Word, wordCount)
	p := make([]fermat, K)
	for i := 0; i < K; i++ {
		p[i] = fermat(pbits[i*(n+1) : (i+1)*(n+1)])
	}
	if err := fourierWithBump(p, v.values, true, n, k, ba); err != nil {
		return poly{}, err
	}

	// Divide by K, and untwist q to recover p.
	// Use bump allocator for temporary u
	u := ba.AllocFermat(n)
	// Use regular allocation for a since it's returned
	a := make([]nat, K)
	for i := 0; i < K; i++ {
		u.Shift(p[i], -int(k))
		copy(p[i], u)
		a[i] = nat(p[i])
	}

	// No release needed - bump allocator handles cleanup
	return poly{k: k, m: 0, a: a}, nil
}

// NTransform evaluates p at θω^i for i = 0...K-1, where
// θ is a (2K)-th primitive root of unity in Z/(b^n+1)Z
// and ω = θ².
func (p *poly) NTransform(n int) polValues {
	k := p.k
	if len(p.a) >= 1<<k {
		panic("Transform: len(p.a) >= 1<<k")
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
		if i < len(p.a) {
			for i := range src {
				src[i] = 0
			}
			copy(src, p.a[i])
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
	return polValues{k, n, values}
}

// InvNTransform reconstructs a polynomial from its values at
// roots of x^K+1. The m field of the returned polynomial
// is unspecified.
func (v *polValues) InvNTransform() poly {
	k := v.k
	n := v.n
	θshift := (n * _W) >> k

	// Perform an inverse Fourier transform to recover q.
	qbits := make([]big.Word, (n+1)<<k)
	q := make([]fermat, 1<<k)
	for i := range q {
		q[i] = fermat(qbits[i*(n+1) : (i+1)*(n+1)])
	}
	fourier(q, v.values, true, n, k)

	// Divide by K, and untwist q to recover p.
	u := make(fermat, n+1)
	a := make([]nat, 1<<k)
	for i := range q {
		u.Shift(q[i], -int(k)-i*θshift)
		copy(q[i], u)
		a[i] = nat(q[i])
	}
	return poly{k: k, m: 0, a: a}
}

// fourier performs an unnormalized Fourier transform
// of src, a length 1<<k vector of numbers modulo b^n+1
// where b = 1<<_W.
func fourier(dst []fermat, src []fermat, backward bool, n int, k uint) error {
	return fourierWithState(dst, src, backward, n, k, nil)
}

// fourierRecursive is the extracted recursive FFT function used for parallel execution.
// It takes its own temporary buffers to avoid race conditions when called from goroutines.
// This function eliminates code duplication that previously existed in parallel goroutines.
//
// Parameters:
//   - dst: destination slice for FFT results
//   - src: source slice of fermat numbers
//   - backward: true for inverse transform
//   - n: coefficient length
//   - k: log2 of FFT size
//   - size: current recursion size
//   - depth: current recursion depth
//   - tmp, tmp2: temporary buffers for this goroutine
func fourierRecursive(dst, src []fermat, backward bool, n int, k, size, depth uint, tmp, tmp2 fermat) error {
	idxShift := k - size
	ω2shift := (4 * n * _W) >> size
	if backward {
		ω2shift = -ω2shift
	}

	// Validation
	if len(src[0]) != n+1 || len(dst[0]) != n+1 {
		return fmt.Errorf("len(src[0]) != n+1 || len(dst[0]) != n+1")
	}

	// Base cases
	switch size {
	case 0:
		copy(dst[0], src[0])
		return nil
	case 1:
		dst[0].Add(src[0], src[1<<idxShift])
		dst[1].Sub(src[0], src[1<<idxShift])
		return nil
	}

	// Split destination vectors in halves
	dst1 := dst[:1<<(size-1)]
	dst2 := dst[1<<(size-1):]

	// Try to acquire token for parallelism
	// We only try to parallelize if the size is large enough to justify overhead
	if size >= ParallelFFTRecursionThreshold {
		select {
		case getSemaphore() <- struct{}{}:
			// Got token, run second half in parallel
			var wg sync.WaitGroup
			wg.Add(1)
			var errAsync error
			go func() {
				defer wg.Done()
				defer func() { <-getSemaphore() }()

				// Allocate new temps for this branch to avoid race conditions
				t1 := acquireFermat(n + 1)
				t2 := acquireFermat(n + 1)
				defer releaseFermat(t1)
				defer releaseFermat(t2)

				errAsync = fourierRecursive(dst2, src[1<<idxShift:], backward, n, k, size-1, depth+1, t1, t2)
			}()

			// Run first half in current thread with current temps
			errSync := fourierRecursive(dst1, src, backward, n, k, size-1, depth+1, tmp, tmp2)

			wg.Wait()
			if errAsync != nil {
				return errAsync
			}
			if errSync != nil {
				return errSync
			}
			goto Reconstruct
		default:
			// Fallthrough to sequential
		}
	}

	// Recursive calls (Sequential)
	if err := fourierRecursive(dst1, src, backward, n, k, size-1, depth+1, tmp, tmp2); err != nil {
		return err
	}
	if err := fourierRecursive(dst2, src[1<<idxShift:], backward, n, k, size-1, depth+1, tmp, tmp2); err != nil {
		return err
	}

Reconstruct:
	// Reconstruct transform
	for i := range dst1 {
		tmp.ShiftHalf(dst2[i], i*ω2shift, tmp2)
		dst2[i].Sub(dst1[i], tmp)
		dst1[i].Add(dst1[i], tmp)
	}
	return nil
}

// fourierWithState performs the Fourier transform with optional pre-allocated state.
// If state is nil, temporary buffers are allocated from the pool.
func fourierWithState(dst []fermat, src []fermat, backward bool, n int, k uint, state *fftState) error {
	// Use pooled state if not provided
	var tmp, tmp2 fermat
	if state != nil {
		tmp = state.tmp
		tmp2 = state.tmp2
	} else {
		tmp = acquireFermat(n + 1)
		tmp2 = acquireFermat(n + 1)
		defer releaseFermat(tmp)
		defer releaseFermat(tmp2)
	}

	// Call the recursive FFT function
	return fourierRecursive(dst, src, backward, n, k, k, 0, tmp, tmp2)
}

// fourierWithBump performs the Fourier transform using a bump allocator for
// temporary buffers. This provides better cache locality than fourierWithState.
func fourierWithBump(dst []fermat, src []fermat, backward bool, n int, k uint, ba *BumpAllocator) error {
	tmp := ba.AllocFermat(n)
	tmp2 := ba.AllocFermat(n)

	// Call the recursive FFT function with bump-allocated temps
	return fourierRecursiveWithBump(dst, src, backward, n, k, k, 0, tmp, tmp2, ba)
}

// fourierRecursiveWithBump is the bump-allocator variant of fourierRecursive.
// It allocates new temp buffers from the bump allocator for parallel goroutines.
func fourierRecursiveWithBump(dst, src []fermat, backward bool, n int, k, size, depth uint, tmp, tmp2 fermat, ba *BumpAllocator) error {
	idxShift := k - size
	ω2shift := (4 * n * _W) >> size
	if backward {
		ω2shift = -ω2shift
	}

	// Validation
	if len(src[0]) != n+1 || len(dst[0]) != n+1 {
		return fmt.Errorf("len(src[0]) != n+1 || len(dst[0]) != n+1")
	}

	// Base cases
	switch size {
	case 0:
		copy(dst[0], src[0])
		return nil
	case 1:
		dst[0].Add(src[0], src[1<<idxShift])
		dst[1].Sub(src[0], src[1<<idxShift])
		return nil
	}

	// Split destination vectors in halves
	dst1 := dst[:1<<(size-1)]
	dst2 := dst[1<<(size-1):]

	// Try to acquire token for parallelism
	if size >= ParallelFFTRecursionThreshold {
		select {
		case getSemaphore() <- struct{}{}:
			// Got token, run second half in parallel
			var wg sync.WaitGroup
			wg.Add(1)
			var errAsync error
			go func() {
				defer wg.Done()
				defer func() { <-getSemaphore() }()

				// Allocate new temps for this branch from bump allocator
				// Note: Since bump allocator is not thread-safe, we fall back to pool here
				t1 := acquireFermat(n + 1)
				t2 := acquireFermat(n + 1)
				defer releaseFermat(t1)
				defer releaseFermat(t2)

				errAsync = fourierRecursive(dst2, src[1<<idxShift:], backward, n, k, size-1, depth+1, t1, t2)
			}()

			// Run first half in current thread with current temps
			errSync := fourierRecursiveWithBump(dst1, src, backward, n, k, size-1, depth+1, tmp, tmp2, ba)

			wg.Wait()
			if errAsync != nil {
				return errAsync
			}
			if errSync != nil {
				return errSync
			}
			goto Reconstruct
		default:
			// Fallthrough to sequential
		}
	}

	// Recursive calls (Sequential)
	if err := fourierRecursiveWithBump(dst1, src, backward, n, k, size-1, depth+1, tmp, tmp2, ba); err != nil {
		return err
	}
	if err := fourierRecursiveWithBump(dst2, src[1<<idxShift:], backward, n, k, size-1, depth+1, tmp, tmp2, ba); err != nil {
		return err
	}

Reconstruct:
	// Reconstruct transform
	for i := range dst1 {
		tmp.ShiftHalf(dst2[i], i*ω2shift, tmp2)
		dst2[i].Sub(dst1[i], tmp)
		dst1[i].Add(dst1[i], tmp)
	}
	return nil
}

// Mul returns the pointwise product of p and q.
func (p *polValues) Mul(q *polValues) (polValues, error) {
	n := p.n
	K := len(p.values)
	var r polValues
	r.k, r.n = p.k, p.n

	// Use regular allocation for returned data
	r.values = make([]fermat, K)
	wordCount := K * (n + 1)
	bits := make([]big.Word, wordCount)

	// Use pooled buffer for temporary multiplication result
	buf := acquireFermat(8 * n)

	for i := 0; i < K; i++ {
		r.values[i] = bits[i*(n+1) : (i+1)*(n+1)]
		z := buf.Mul(p.values[i], q.values[i])
		copy(r.values[i], z)
	}

	// Release temporary buffer
	releaseFermat(buf)

	return r, nil
}

// MulWithBump returns the pointwise product of p and q, using a bump allocator
// for temporary buffers.
func (p *polValues) MulWithBump(q *polValues, ba *BumpAllocator) (polValues, error) {
	n := p.n
	K := len(p.values)
	var r polValues
	r.k, r.n = p.k, p.n

	// Use regular allocation for returned data
	r.values = make([]fermat, K)
	wordCount := K * (n + 1)
	bits := make([]big.Word, wordCount)

	// Use bump allocator for temporary multiplication result
	buf := ba.AllocFermat(8*n - 1)

	for i := 0; i < K; i++ {
		r.values[i] = bits[i*(n+1) : (i+1)*(n+1)]
		z := buf.Mul(p.values[i], q.values[i])
		copy(r.values[i], z)
	}

	// No release needed - bump allocator handles cleanup
	return r, nil
}

// Sqr returns the pointwise square of p (p[i] * p[i] for each i).
// This is optimized for squaring as we don't need a second set of values.
func (p *polValues) Sqr() (polValues, error) {
	n := p.n
	K := len(p.values)
	var r polValues
	r.k, r.n = p.k, p.n

	// Use regular allocation for returned data
	r.values = make([]fermat, K)
	wordCount := K * (n + 1)
	bits := make([]big.Word, wordCount)

	// Use pooled buffer for temporary multiplication result
	buf := acquireFermat(8 * n)

	for i := 0; i < K; i++ {
		r.values[i] = bits[i*(n+1) : (i+1)*(n+1)]
		// Square: multiply p.values[i] by itself
		z := buf.Mul(p.values[i], p.values[i])
		copy(r.values[i], z)
	}

	// Release temporary buffer
	releaseFermat(buf)

	return r, nil
}

// SqrWithBump returns the pointwise square of p, using a bump allocator
// for temporary buffers.
func (p *polValues) SqrWithBump(ba *BumpAllocator) (polValues, error) {
	n := p.n
	K := len(p.values)
	var r polValues
	r.k, r.n = p.k, p.n

	// Use regular allocation for returned data
	r.values = make([]fermat, K)
	wordCount := K * (n + 1)
	bits := make([]big.Word, wordCount)

	// Use bump allocator for temporary multiplication result
	buf := ba.AllocFermat(8*n - 1)

	for i := 0; i < K; i++ {
		r.values[i] = bits[i*(n+1) : (i+1)*(n+1)]
		// Square: multiply p.values[i] by itself
		z := buf.Mul(p.values[i], p.values[i])
		copy(r.values[i], z)
	}

	// No release needed - bump allocator handles cleanup
	return r, nil
}
