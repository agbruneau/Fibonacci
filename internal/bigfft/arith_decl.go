// Copyright 2010 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
