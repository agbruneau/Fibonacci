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
// Core Karatsuba Implementation using big.Int
// ─────────────────────────────────────────────────────────────────────────────

// karatsubaMulBigInt multiplies x and y using Karatsuba algorithm.
// x and y must be non-negative.
func karatsubaMulBigInt(z, x, y *big.Int, depth int) {
	n := len(x.Bits())
	m := len(y.Bits())

	// Ensure x is the larger operand
	if n < m {
		x, y = y, x
		n, m = m, n
	}

	// Base case: use standard multiplication for small inputs
	if n <= karatsubaThreshold {
		z.Mul(x, y)
		return
	}

	// If y is very small, use standard multiplication
	if m <= karatsubaThreshold/2 {
		z.Mul(x, y)
		return
	}

	// Split point: use half of the smaller operand for balanced splits
	k := m / 2
	if k == 0 {
		k = 1
	}
	kBits := uint(k * _W) // k words = k * WordBits bits

	// Split x = x1*B^k + x0 where B = 2^(k*WordBits)
	x0 := acquireBigInt()
	x1 := acquireBigInt()
	x0.And(x, new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), kBits), big.NewInt(1)))
	x1.Rsh(x, kBits)
	defer releaseBigInt(x0)
	defer releaseBigInt(x1)

	// Split y = y1*B^k + y0
	y0 := acquireBigInt()
	y1 := acquireBigInt()
	y0.And(y, new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), kBits), big.NewInt(1)))
	y1.Rsh(y, kBits)
	defer releaseBigInt(y0)
	defer releaseBigInt(y1)

	// z0 = x0 * y0
	z0 := acquireBigInt()
	// z2 = x1 * y1
	z2 := acquireBigInt()
	// z1 = (x0 + x1) * (y0 + y1) - z0 - z2
	z1 := acquireBigInt()
	defer releaseBigInt(z0)
	defer releaseBigInt(z2)
	defer releaseBigInt(z1)

	// Compute x0 + x1 and y0 + y1
	sumX := acquireBigInt()
	sumY := acquireBigInt()
	sumX.Add(x0, x1)
	sumY.Add(y0, y1)
	defer releaseBigInt(sumX)
	defer releaseBigInt(sumY)

	// Decide whether to parallelize
	shouldParallelize := depth < MaxKaratsubaParallelDepth &&
		n >= karatsubaParallelThreshold

	if shouldParallelize {
		select {
		case getKaratsubaSemaphore() <- struct{}{}:
			// Got token, run z2 in parallel
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { <-getKaratsubaSemaphore() }()
				karatsubaMulBigInt(z2, x1, y1, depth+1)
			}()

			// Run z0 and z1 in current goroutine
			karatsubaMulBigInt(z0, x0, y0, depth+1)
			karatsubaMulBigInt(z1, sumX, sumY, depth+1)

			wg.Wait()
		default:
			// No token available, run sequentially
			karatsubaMulBigInt(z0, x0, y0, depth+1)
			karatsubaMulBigInt(z2, x1, y1, depth+1)
			karatsubaMulBigInt(z1, sumX, sumY, depth+1)
		}
	} else {
		// Sequential execution
		karatsubaMulBigInt(z0, x0, y0, depth+1)
		karatsubaMulBigInt(z2, x1, y1, depth+1)
		karatsubaMulBigInt(z1, sumX, sumY, depth+1)
	}

	// z1 = z1 - z0 - z2
	z1.Sub(z1, z0)
	z1.Sub(z1, z2)

	// z = z0 + z1*B^k + z2*B^(2k)
	// z = z0 + (z1 << kBits) + (z2 << 2*kBits)
	z.Set(z0)
	tmp := acquireBigInt()
	defer releaseBigInt(tmp)

	tmp.Lsh(z1, kBits)
	z.Add(z, tmp)

	tmp.Lsh(z2, 2*kBits)
	z.Add(z, tmp)
}

// karatsubaSqrBigInt computes x² using Karatsuba.
func karatsubaSqrBigInt(z, x *big.Int, depth int) {
	n := len(x.Bits())

	if n <= karatsubaThreshold {
		z.Mul(x, x)
		return
	}

	k := n / 2
	if k == 0 {
		k = 1
	}
	kBits := uint(k * _W)

	// Split x = x1*B^k + x0
	x0 := acquireBigInt()
	x1 := acquireBigInt()
	x0.And(x, new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), kBits), big.NewInt(1)))
	x1.Rsh(x, kBits)
	defer releaseBigInt(x0)
	defer releaseBigInt(x1)

	// z0 = x0²
	// z2 = x1²
	// z1 = (x0 + x1)² - z0 - z2 = 2*x0*x1
	z0 := acquireBigInt()
	z2 := acquireBigInt()
	z1 := acquireBigInt()
	defer releaseBigInt(z0)
	defer releaseBigInt(z2)
	defer releaseBigInt(z1)

	sumX := acquireBigInt()
	sumX.Add(x0, x1)
	defer releaseBigInt(sumX)

	karatsubaSqrBigInt(z0, x0, depth+1)
	karatsubaSqrBigInt(z2, x1, depth+1)
	karatsubaSqrBigInt(z1, sumX, depth+1)

	z1.Sub(z1, z0)
	z1.Sub(z1, z2)

	// z = z0 + z1*B^k + z2*B^(2k)
	z.Set(z0)
	tmp := acquireBigInt()
	defer releaseBigInt(tmp)

	tmp.Lsh(z1, kBits)
	z.Add(z, tmp)

	tmp.Lsh(z2, 2*kBits)
	z.Add(z, tmp)
}
