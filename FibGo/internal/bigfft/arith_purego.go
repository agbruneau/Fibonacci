//go:build purego

// Pure-Go replacements for the six math/big internals that arith_decl.go
// reaches through go:linkname (EVAL-21). Build with -tags purego when a Go
// release renames or removes one of them, or on a toolchain that refuses the
// pull-style linkname: the package then no longer depends on math/big's
// unexported symbols. The default build keeps the linkname path, which uses
// math/big's assembly kernels; this file is the fallback, not a replacement.
//
// Each function follows math/big's generic (non-assembly) version of the same
// name — operate on len(z) words, return the carry or borrow out of the top
// word. Those versions are Copyright The Go Authors, BSD 3-Clause
// (https://go.dev/LICENSE); see /NOTICE.

package bigfft

import (
	"math/big"
	"math/bits"
)

// Word is an alias for big.Word, representing a single digit in arbitrary-precision arithmetic.
type Word = big.Word

// addVV computes z = x + y element-wise and returns the carry.
func addVV(z, x, y []Word) (c Word) {
	for i := range z {
		s, cc := bits.Add(uint(x[i]), uint(y[i]), uint(c))
		z[i], c = Word(s), Word(cc)
	}
	return c
}

// subVV computes z = x - y element-wise and returns the borrow.
func subVV(z, x, y []Word) (c Word) {
	for i := range z {
		d, bb := bits.Sub(uint(x[i]), uint(y[i]), uint(c))
		z[i], c = Word(d), Word(bb)
	}
	return c
}

// addVW computes z = x + y where y is a single word, and returns the carry.
func addVW(z, x []Word, y Word) (c Word) {
	c = y
	for i := range z {
		s, cc := bits.Add(uint(x[i]), uint(c), 0)
		z[i], c = Word(s), Word(cc)
	}
	return c
}

// subVW computes z = x - y where y is a single word, and returns the borrow.
func subVW(z, x []Word, y Word) (c Word) {
	c = y
	for i := range z {
		d, bb := bits.Sub(uint(x[i]), uint(c), 0)
		z[i], c = Word(d), Word(bb)
	}
	return c
}

// shlVU computes z = x << s and returns the shifted-out high bits.
// s must be less than the word size, as for math/big's shlVU.
func shlVU(z, x []Word, s uint) (c Word) {
	if s == 0 {
		copy(z, x)
		return 0
	}
	if len(z) == 0 {
		return 0
	}
	s &= bits.UintSize - 1
	r := bits.UintSize - s
	c = x[len(z)-1] >> r
	for i := len(z) - 1; i > 0; i-- {
		z[i] = x[i]<<s | x[i-1]>>r
	}
	z[0] = x[0] << s
	return c
}

// addMulVVW computes z += x*y element-wise and returns the carry.
func addMulVVW(z, x []Word, y Word) (c Word) {
	for i := range z {
		hi, lo := bits.Mul(uint(x[i]), uint(y))
		lo, cc := bits.Add(lo, uint(z[i]), 0)
		hi += cc
		lo, cc = bits.Add(lo, uint(c), 0)
		z[i], c = Word(lo), Word(hi+cc)
	}
	return c
}
