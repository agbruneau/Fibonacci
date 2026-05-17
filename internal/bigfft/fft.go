// The implementation is based on the Schönhage-Strassen method
// using integer FFT modulo 2^n+1.

package bigfft

import (
	"fmt"
	"math/big"
	"runtime/debug"
	"sync/atomic"
	"unsafe"
)

const _W = int(unsafe.Sizeof(big.Word(0)) * 8)

// nat is an alias for a slice of big.Word, used as the internal
// representation of large integers in FFT-based arithmetic.
type nat []big.Word

func (n nat) String() string {
	v := new(big.Int)
	v.SetBits(n)
	return v.String()
}

// defaultFFTThresholdWords is the default size (in words) above which FFT is
// used over standard math/big multiplication.
//
// TestCalibrate seems to indicate a threshold of 60kbits on 32-bit
// arches and 110kbits on 64-bit arches. The value 1800 words corresponds to
// approximately 115kbits on 64-bit systems (1800 * 64 = 115200 bits).
const defaultFFTThresholdWords = 1800

// fftThreshold is the size (in words) above which FFT is used over
// standard math/big multiplication. This can be modified for tuning
// purposes. It is read on every Mul/Sqr/MulTo/SqrTo entry by N concurrent
// goroutines, so it is an atomic to make any (current or future) tuning
// write race-free (audit A-03). Same global, type changed only
// (int -> atomic.Int64); no new global introduced. Read via fftThresholdValue().
var fftThreshold atomic.Int64

func init() { fftThreshold.Store(defaultFFTThresholdWords) }

// fftThresholdValue returns the current FFT threshold in words.
func fftThresholdValue() int { return int(fftThreshold.Load()) }

// Mul computes the product x*y and returns z.
// It can be used instead of the Mul method of
// *big.Int from math/big package.
func Mul(x, y *big.Int) (res *big.Int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in bigfft.Mul: %v\nStack: %s", r, debug.Stack())
		}
	}()
	xwords := len(x.Bits())
	ywords := len(y.Bits())
	thr := fftThresholdValue()
	if xwords > thr && ywords > thr {
		return mulFFT(x, y)
	}
	return new(big.Int).Mul(x, y), nil
}

// MulTo computes the product x*y and stores the result in z.
// It can be used instead of the Mul method of *big.Int from math/big package.
func MulTo(z, x, y *big.Int) (res *big.Int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in bigfft.MulTo: %v\nStack: %s", r, debug.Stack())
		}
	}()
	xwords := len(x.Bits())
	ywords := len(y.Bits())
	thr := fftThresholdValue()
	if xwords > thr && ywords > thr {
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
func Sqr(x *big.Int) (res *big.Int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in bigfft.Sqr: %v\nStack: %s", r, debug.Stack())
		}
	}()
	xwords := len(x.Bits())
	if xwords > fftThresholdValue() {
		return sqrFFT(x)
	}
	return new(big.Int).Mul(x, x), nil
}

func SqrTo(z, x *big.Int) (res *big.Int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in bigfft.SqrTo: %v\nStack: %s", r, debug.Stack())
		}
	}()
	xwords := len(x.Bits())
	if xwords > fftThresholdValue() {
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

// PolyFromInt converts a *big.Int to a Poly representation.
// The parameters k and m must be appropriate for the intended operations.
func PolyFromInt(x *big.Int, k uint, m int) Poly {
	return polyFromNat(x.Bits(), k, m)
}

// GetFFTParams returns the FFT parameters k and m suitable for a result
// of a given number of words.
func GetFFTParams(words int) (k uint, m int) {
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

// ValueSize returns the length (in words) to use for polynomial
// coefficients. The chosen length must be a multiple of 1 << (k-extra).
func ValueSize(k uint, m int, extra uint) int {
	return valueSize(k, m, extra)
}

func valueSize(k uint, m int, extra uint) int {
	// The coefficients of P*Q are less than b^(2m)*K
	// so we need W * valueSize >= 2*m*W+K.
	// k is the FFT level (small: typically < 32), so int(k) is safe. #nosec G115
	n := 2*m*_W + int(k) // necessary bits
	K := 1 << (k - extra)
	if K < _W {
		K = _W
	}
	n = ((n / K) + 1) * K // round to a multiple of K
	return n / _W
}
