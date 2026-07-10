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
// standard math/big multiplication. It is held in an atomic.Int64 so
// concurrent readers on the hot path observe a coherent value without
// taking a lock; writers go through SetFFTThreshold.
var fftThreshold atomic.Int64

func init() {
	fftThreshold.Store(int64(defaultFFTThresholdWords))
}

// getFFTThreshold returns the current FFT activation threshold (in words).
// Hot-path readers must use this accessor, not direct loads.
func getFFTThreshold() int { return int(fftThreshold.Load()) }

// SetFFTThreshold updates the FFT activation threshold (in words) atomically.
// Test-only in practice: no production caller exists (internal/calibration's
// only touchpoint with this package is bigfft.Mul, which reads the threshold
// via getFFTThreshold but never calls this setter); production code should
// prefer FFTContext-based configuration.
func SetFFTThreshold(v int) { fftThreshold.Store(int64(v)) }

// Mul computes the product x*y and returns z.
// It can be used instead of the Mul method of
// *big.Int from math/big package.
//
// Panic policy (ADR-0002): pre-condition panics (operand-size mismatches)
// are converted to error so callers retain the API contract. Post-condition
// panics (internal algorithmic invariants such as "unexpected carry" or
// "len(z) > 2n+1") are RE-PROPAGATED so genuine bugs in the modular
// reduction surface as panics in tests/CI rather than masquerading as
// opaque errors. See isFermatPostConditionPanic for the sentinel list.
func Mul(x, y *big.Int) (res *big.Int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fermatPanicToError(r, "Mul")
		}
	}()
	xwords := len(x.Bits())
	ywords := len(y.Bits())
	if t := getFFTThreshold(); xwords > t && ywords > t {
		return mulFFT(x, y)
	}
	return new(big.Int).Mul(x, y), nil
}

// MulTo computes the product x*y and stores the result in z.
// It can be used instead of the Mul method of *big.Int from math/big package.
//
// Panic policy: see Mul.
func MulTo(z, x, y *big.Int) (res *big.Int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fermatPanicToError(r, "MulTo")
		}
	}()
	xwords := len(x.Bits())
	ywords := len(y.Bits())
	if t := getFFTThreshold(); xwords > t && ywords > t {
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
//
// Panic policy: see Mul.
func Sqr(x *big.Int) (res *big.Int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fermatPanicToError(r, "Sqr")
		}
	}()
	xwords := len(x.Bits())
	if xwords > getFFTThreshold() {
		return sqrFFT(x)
	}
	return new(big.Int).Mul(x, x), nil
}

// SqrTo computes x*x and stores the result in z.
//
// Panic policy: see Mul.
func SqrTo(z, x *big.Int) (res *big.Int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fermatPanicToError(r, "SqrTo")
		}
	}()
	xwords := len(x.Bits())
	if xwords > getFFTThreshold() {
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

// fermatPostConditionPanics enumerates the panic messages emitted from
// internal Fermat-arithmetic post-conditions. They are NOT caller errors;
// they signal a bug in the modular reduction routine. The four entry
// points (Mul/MulTo/Sqr/SqrTo) re-propagate these panics so they cannot
// be silently transformed into an opaque error value. Pre-condition
// panics (operand-size mismatches) remain converted to errors, preserving
// the existing public contract.
var fermatPostConditionPanics = map[string]struct{}{
	panicMsgOversizedProduct: {},
	panicMsgMulCarry:         {},
	panicMsgSqrCarry:         {},
}

// isFermatPostConditionPanic reports whether r is a recover()'d value that
// originated from one of the internal Fermat post-condition panics.
func isFermatPostConditionPanic(r any) bool {
	msg, ok := r.(string)
	if !ok {
		return false
	}
	_, found := fermatPostConditionPanics[msg]
	return found
}

// fermatPanicToError is the single implementation of the panic policy
// (ADR-0002) shared by every public entry point — Mul/MulTo/Sqr/SqrTo and
// their *WithContext variants. Post-condition sentinels from fermat.go are
// re-propagated so genuine bugs in the modular reduction surface as panics;
// every other panic is converted to an error carrying the stack. r must be
// the non-nil value obtained from recover().
func fermatPanicToError(r any, funcName string) error {
	if isFermatPostConditionPanic(r) {
		panic(r) // post-condition violation: do not mask
	}
	return fmt.Errorf("panic in bigfft.%s: %v\nStack: %s", funcName, r, debug.Stack())
}
