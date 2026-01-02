//go:build amd64

// Package bigfft implements multiplication of big.Int using FFT.
// This file provides the dispatcher for SIMD-optimized arithmetic operations on amd64.
package bigfft

import (
	"math/big"
)

// ─────────────────────────────────────────────────────────────────────────────
// Function Pointers for Dynamic Dispatch
// ─────────────────────────────────────────────────────────────────────────────

// These function variables allow switching between implementations at runtime.
// They are initialized to use go:linkname functions and can be switched to
// AVX2 or AVX-512 implementations based on CPU capabilities.

var (
	// addVVFunc is the active implementation of vector-vector addition
	addVVFunc func(z, x, y []Word) Word

	// subVVFunc is the active implementation of vector-vector subtraction
	subVVFunc func(z, x, y []Word) Word

	// addMulVVWFunc is the active implementation of multiply-accumulate
	addMulVVWFunc func(z, x []Word, y Word) Word
)

func init() {
	// Initialize with default (go:linkname) implementations
	addVVFunc = addVV
	subVVFunc = subVV
	addMulVVWFunc = addMulVVW

	// Select best available implementation based on CPU features
	selectImplementation()
}

// ─────────────────────────────────────────────────────────────────────────────
// Dispatched Functions
// ─────────────────────────────────────────────────────────────────────────────

// AddVV computes z = x + y for vectors, returning the carry.
// This function dispatches to the best available implementation.
func AddVV(z, x, y []big.Word) big.Word {
	if len(z) == 0 {
		return 0
	}
	return addVVFunc(z, x, y)
}

// SubVV computes z = x - y for vectors, returning the borrow.
// This function dispatches to the best available implementation.
func SubVV(z, x, y []big.Word) big.Word {
	if len(z) == 0 {
		return 0
	}
	return subVVFunc(z, x, y)
}

// AddMulVVW computes z += x * y where y is a single word.
// This function dispatches to the best available implementation.
func AddMulVVW(z, x []big.Word, y big.Word) big.Word {
	if len(z) == 0 {
		return 0
	}
	return addMulVVWFunc(z, x, y)
}

// ─────────────────────────────────────────────────────────────────────────────
// AVX2 Implementation Declarations
// ─────────────────────────────────────────────────────────────────────────────

// These are implemented in arith_amd64.s

//go:noescape
func addVVAvx2(z, x, y []Word) (c Word)

//go:noescape
func subVVAvx2(z, x, y []Word) (c Word)

//go:noescape
func addMulVVWAvx2(z, x []Word, y Word) (c Word)

// ─────────────────────────────────────────────────────────────────────────────
// Implementation Selection
// ─────────────────────────────────────────────────────────────────────────────

// UseAVX2 switches to AVX2 implementations if available.
// Returns true if AVX2 was enabled.
func UseAVX2() bool {
	if !hasAVX2 {
		return false
	}
	addVVFunc = addVVAvx2
	subVVFunc = subVVAvx2
	addMulVVWFunc = addMulVVWAvx2
	implLevel = SIMDAVX2
	return true
}

// UseDefault switches back to the default (go:linkname) implementations.
func UseDefault() {
	addVVFunc = addVV
	subVVFunc = subVV
	addMulVVWFunc = addMulVVW
	implLevel = SIMDNone
}

// ─────────────────────────────────────────────────────────────────────────────
// Threshold-based Auto Selection
// ─────────────────────────────────────────────────────────────────────────────

// MinSIMDVectorLen is the minimum vector length where SIMD provides benefit.
// Below this threshold, scalar operations may be faster due to overhead.
const MinSIMDVectorLen = 8

// AddVVAuto automatically selects the best implementation based on vector length.
// For short vectors, uses scalar. For long vectors, uses SIMD if available.
func AddVVAuto(z, x, y []big.Word) big.Word {
	n := len(z)
	if n == 0 {
		return 0
	}
	if n < MinSIMDVectorLen || !hasAVX2 {
		return addVV(z, x, y)
	}
	return addVVAvx2(z, x, y)
}

// SubVVAuto automatically selects the best implementation based on vector length.
func SubVVAuto(z, x, y []big.Word) big.Word {
	n := len(z)
	if n == 0 {
		return 0
	}
	if n < MinSIMDVectorLen || !hasAVX2 {
		return subVV(z, x, y)
	}
	return subVVAvx2(z, x, y)
}

// AddMulVVWAuto automatically selects the best implementation based on vector length.
func AddMulVVWAuto(z, x []big.Word, y big.Word) big.Word {
	n := len(z)
	if n == 0 {
		return 0
	}
	if n < MinSIMDVectorLen || !hasAVX2 {
		return addMulVVW(z, x, y)
	}
	return addMulVVWAvx2(z, x, y)
}
