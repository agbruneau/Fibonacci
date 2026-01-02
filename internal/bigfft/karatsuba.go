// Package bigfft implements multiplication of big.Int using FFT.
// This file provides a Karatsuba multiplication implementation with
// memory pooling for reduced GC pressure.
package bigfft

import (
	"math/big"
	"sync"
)

// ─────────────────────────────────────────────────────────────────────────────
// Configuration Constants
// ─────────────────────────────────────────────────────────────────────────────

// DefaultKaratsubaThreshold is the size in words below which the naive O(n²)
// schoolbook multiplication is faster than Karatsuba. This value has been
// chosen based on typical CPU cache characteristics.
const DefaultKaratsubaThreshold = 32

// DefaultParallelKaratsubaThreshold is the minimum size in words for which
// Karatsuba should parallelize its recursive calls.
const DefaultParallelKaratsubaThreshold = 4096

// MaxKaratsubaParallelDepth limits the maximum depth of parallel recursion
// to avoid excessive goroutine creation.
const MaxKaratsubaParallelDepth = 3

// karatsubaThreshold is the current threshold (can be modified for tuning).
var karatsubaThreshold = DefaultKaratsubaThreshold

// karatsubaParallelThreshold is the current parallel threshold.
var karatsubaParallelThreshold = DefaultParallelKaratsubaThreshold

// ─────────────────────────────────────────────────────────────────────────────
// Memory Pools for big.Int
// ─────────────────────────────────────────────────────────────────────────────

// bigIntPool pools *big.Int objects for reuse
var bigIntPool = sync.Pool{
	New: func() any {
		return new(big.Int)
	},
}

// acquireBigInt gets a big.Int from the pool.
func acquireBigInt() *big.Int {
	return bigIntPool.Get().(*big.Int)
}

// releaseBigInt returns a big.Int to the pool.
func releaseBigInt(x *big.Int) {
	x.SetInt64(0)
	bigIntPool.Put(x)
}

// ─────────────────────────────────────────────────────────────────────────────
// Karatsuba Semaphore for Parallelism
// ─────────────────────────────────────────────────────────────────────────────

var karatsubaSemaphore chan struct{}
var karatsubaSemaphoreOnce sync.Once

// getKaratsubaSemaphore returns the semaphore for limiting parallel goroutines.
func getKaratsubaSemaphore() chan struct{} {
	karatsubaSemaphoreOnce.Do(func() {
		// Use same concurrency limit as FFT
		karatsubaSemaphore = getSemaphore()
	})
	return karatsubaSemaphore
}

// ─────────────────────────────────────────────────────────────────────────────
// Public API
// ─────────────────────────────────────────────────────────────────────────────

// KaratsubaMultiply computes x * y using the Karatsuba algorithm.
// It returns a new *big.Int containing the result.
//
// This implementation uses memory pooling to reduce GC pressure,
// making it suitable for hot loops like Fibonacci's Fast Doubling.
func KaratsubaMultiply(x, y *big.Int) *big.Int {
	return KaratsubaMultiplyTo(new(big.Int), x, y)
}

// KaratsubaMultiplyTo computes x * y and stores the result in z.
// This allows reusing the allocated memory of z.
//
// The function handles signs correctly: the result sign is
// positive if x and y have the same sign, negative otherwise.
// KaratsubaMultiplyTo computes x * y and stores the result in z.
func KaratsubaMultiplyTo(z, x, y *big.Int) *big.Int {
	// Handle zero cases
	if x.Sign() == 0 || y.Sign() == 0 {
		return z.SetInt64(0)
	}

	// Get absolute values
	xAbs := acquireBigInt()
	yAbs := acquireBigInt()
	xAbs.Abs(x)
	yAbs.Abs(y)
	defer releaseBigInt(xAbs)
	defer releaseBigInt(yAbs)

	// Determine result sign
	negative := x.Sign() != y.Sign()

	// Perform multiplication using internal Karatsuba
	karatsubaMulBigInt(z, xAbs, yAbs, 0)

	if negative {
		z.Neg(z)
	}

	return z
}

// KaratsubaSqr computes x² using the Karatsuba algorithm.
// This is slightly more efficient than KaratsubaMultiply(x, x)
// because we can reuse some intermediate computations.
func KaratsubaSqr(x *big.Int) *big.Int {
	return KaratsubaSqrTo(new(big.Int), x)
}

// KaratsubaSqrTo computes x² and stores the result in z.
func KaratsubaSqrTo(z, x *big.Int) *big.Int {
	if x.Sign() == 0 {
		return z.SetInt64(0)
	}

	xAbs := acquireBigInt()
	xAbs.Abs(x)
	defer releaseBigInt(xAbs)

	karatsubaSqrBigInt(z, xAbs, 0)

	// x² is always non-negative
	return z
}

// SetKaratsubaThreshold sets the threshold for Karatsuba vs naive multiplication.
// This is useful for benchmarking and tuning.
func SetKaratsubaThreshold(threshold int) {
	if threshold < 1 {
		threshold = 1
	}
	karatsubaThreshold = threshold
}

// GetKaratsubaThreshold returns the current Karatsuba threshold.
func GetKaratsubaThreshold() int {
	return karatsubaThreshold
}

// ─────────────────────────────────────────────────────────────────────────────
// Core Karatsuba Implementation using low-level word slices (nat)
// ─────────────────────────────────────────────────────────────────────────────

// karatsubaMulBigInt multiplies x and y using Karatsuba algorithm.
func karatsubaMulBigInt(z, x, y *big.Int, depth int) {
	xb, yb := x.Bits(), y.Bits()
	zb := karatsuba(xb, yb, depth)
	z.SetBits(zb)
}

// karatsuba is the low-level Karatsuba implementation operating on word slices.
func karatsuba(x, y nat, depth int) nat {
	n := len(x)
	m := len(y)

	if n < m {
		x, y = y, x
		n, m = m, n
	}

	// Base cases
	if m == 0 {
		return nil
	}
	if n <= karatsubaThreshold {
		return multiplyNaive(x, y)
	}

	// For highly asymmetric operands, split the larger one
	if n > 2*m {
		return multiplyAsymmetric(x, y, depth)
	}

	k := n / 2
	x0, x1 := x[:k], x[k:]
	y0, y1 := y[:k], y[k:]
	if len(y) <= k {
		y0, y1 = y, nil
	}

	// z0 = x0 * y0
	// z2 = x1 * y1
	// z1 = (x0 + x1) * (y0 + y1) - z0 - z2

	var z0, z1, z2 nat
	shouldParallel := depth < MaxKaratsubaParallelDepth && n >= karatsubaParallelThreshold

	if shouldParallel {
		select {
		case getKaratsubaSemaphore() <- struct{}{}:
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { <-getKaratsubaSemaphore() }()
				z2 = karatsuba(x1, y1, depth+1)
			}()
			z0 = karatsuba(x0, y0, depth+1)
			wg.Wait()
		default:
			z0 = karatsuba(x0, y0, depth+1)
			z2 = karatsuba(x1, y1, depth+1)
		}
	} else {
		z0 = karatsuba(x0, y0, depth+1)
		z2 = karatsuba(x1, y1, depth+1)
	}

	// sumX = x0 + x1
	sumX := add(x0, x1)
	sumY := add(y0, y1)
	z1 = karatsuba(sumX, sumY, depth+1)

	// z1 = z1 - z0 - z2
	z1 = sub(z1, z0)
	z1 = sub(z1, z2)

	// Result = z0 + (z1 << k) + (z2 << 2k)
	return assemble(z0, z1, z2, k)
}

// multiplyNaive uses math/big's internal multiplication for small inputs.
func multiplyNaive(x, y nat) nat {
	xi := new(big.Int).SetBits(x)
	yi := new(big.Int).SetBits(y)
	return new(big.Int).Mul(xi, yi).Bits()
}

// multiplyAsymmetric handles cases where one operand is much larger than the other.
func multiplyAsymmetric(x, y nat, depth int) nat {
	m := len(y)
	result := make(nat, len(x)+m)
	for i := 0; i < len(x); i += m {
		end := i + m
		if end > len(x) {
			end = len(x)
		}
		part := karatsuba(x[i:end], y, depth+1)
		addAt(result, part, i)
	}
	return trim(result)
}

func add(x, y nat) nat {
	if len(x) < len(y) {
		x, y = y, x
	}
	if len(y) == 0 {
		return x
	}
	z := make(nat, len(x)+1)
	c := AddVV(z[:len(y)], x[:len(y)], y)
	if len(x) > len(y) {
		c = addVW(z[len(y):len(x)], x[len(y):], c)
	}
	z[len(x)] = c
	return trim(z)
}

func sub(x, y nat) nat {
	// Assumes x >= y
	z := make(nat, len(x))
	if len(y) == 0 {
		copy(z, x)
		return z
	}
	c := SubVV(z[:len(y)], x[:len(y)], y)
	if len(x) > len(y) {
		subVW(z[len(y):], x[len(y):], c)
	}
	return trim(z)
}

func addAt(z, x nat, shift int) {
	if len(x) == 0 {
		return
	}
	n := len(z)
	m := len(x)
	if shift+m > n {
		panic("addAt: out of bounds")
	}
	c := AddVV(z[shift:shift+m], z[shift:shift+m], x)
	if c != 0 && shift+m < n {
		addVW(z[shift+m:], z[shift+m:], c)
	}
}

func assemble(z0, z1, z2 nat, k int) nat {
	// Let B = 2^(k*W)
	// result = z0 + z1*B + z2*B^2
	// size = len(z2) + 2*k
	size := len(z2) + 2*k
	if s := len(z1) + k; s > size {
		size = s
	}
	if s := len(z0); s > size {
		size = s
	}
	res := make(nat, size+1) // +1 for extra carry
	copy(res, z0)
	addAt(res, z1, k)
	addAt(res, z2, 2*k)
	return trim(res)
}

func karatsubaSqrBigInt(z, x *big.Int, depth int) {
	xb := x.Bits()
	zb := karatsubaSqr(xb, depth)
	z.SetBits(zb)
}

func karatsubaSqr(x nat, depth int) nat {
	n := len(x)
	if n <= karatsubaThreshold {
		xi := new(big.Int).SetBits(x)
		return new(big.Int).Mul(xi, xi).Bits()
	}

	k := n / 2
	x0, x1 := x[:k], x[k:]

	// z0 = x0^2, z2 = x1^2, z1 = (x0+x1)^2 - z0 - z2
	z0 := karatsubaSqr(x0, depth+1)
	z2 := karatsubaSqr(x1, depth+1)

	sumX := add(x0, x1)
	z1 := karatsubaSqr(sumX, depth+1)
	z1 = sub(z1, z0)
	z1 = sub(z1, z2)

	return assemble(z0, z1, z2, k)
}
