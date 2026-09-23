// Copyright 2010 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Derived from github.com/remyoudompheng/bigfft (BSD 3-Clause; see LICENSE in
// this directory), file arith_decl.go at upstream commit 24d4a6f8daec; the
// three-line notice above is upstream's, and the LICENSE file it names is
// internal/bigfft/LICENSE.
//
// Taken from upstream: the Word alias and the go:linkname declarations of
// addVV, subVV, addVW, subVW, shlVU and addMulVVW.
//
// Modifications Copyright 2026 André-Guy Bruneau, licensed under Apache-2.0
// (see /LICENSE and /NOTICE): the mulAddVWW declaration was dropped, and the
// warning below and the per-declaration comments were added.

// WARNING: This file uses //go:linkname to access unexported functions from
// math/big for performance reasons. This technique is fragile and carries
// several risks:
//
//  1. These internal functions are not part of Go's public API and may change
//     or be removed in future Go versions without notice.
//  2. The function signatures must match exactly; any mismatch can cause
//     runtime panics or memory corruption.
//  3. This approach may break with different Go compilers or build modes.
//
// If this package fails to compile or behaves unexpectedly after a Go upgrade,
// the linkname declarations below should be reviewed against the current
// math/big implementation.

package bigfft

import (
	"math/big"
	_ "unsafe" // Required for go:linkname
)

// Word is an alias for big.Word, representing a single digit in arbitrary-precision arithmetic.
type Word = big.Word

// The following functions are linked to internal math/big functions for performance.
// They provide low-level vector arithmetic operations used in FFT-based multiplication.

// addVV computes z = x + y element-wise and returns the carry.
//
//go:linkname addVV math/big.addVV
func addVV(z, x, y []Word) (c Word)

// subVV computes z = x - y element-wise and returns the borrow.
//
//go:linkname subVV math/big.subVV
func subVV(z, x, y []Word) (c Word)

// addVW computes z = x + y where y is a single word, and returns the carry.
//
//go:linkname addVW math/big.addVW
func addVW(z, x []Word, y Word) (c Word)

// subVW computes z = x - y where y is a single word, and returns the borrow.
//
//go:linkname subVW math/big.subVW
func subVW(z, x []Word, y Word) (c Word)

// shlVU computes z = x << s and returns the shifted-out high bits.
//
//go:linkname shlVU math/big.shlVU
func shlVU(z, x []Word, s uint) (c Word)

// addMulVVW computes z += x*y element-wise and returns the carry.
//
//go:linkname addMulVVW math/big.addMulVVW
func addMulVVW(z, x []Word, y Word) (c Word)
