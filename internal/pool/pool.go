// Package pool provides a global memory pool for big.Int reuse.
// This helps reduce GC pressure by recycling big.Int objects across
// different parts of the application.
package pool

import (
	"math/big"
	"sync"
)

// MaxPooledBitLen is the maximum size (in bits) of a big.Int
// accepted into the pool. Larger objects are left for GC collection.
// Approximately 512 KB of data.
const MaxPooledBitLen = 4_000_000

// bigIntPool is a sync.Pool for *big.Int objects.
var bigIntPool = sync.Pool{
	New: func() any {
		return new(big.Int)
	},
}

// AcquireBigInt returns a *big.Int from the pool.
// The returned big.Int is reset to 0 but retains its capacity.
func AcquireBigInt() *big.Int {
	z := bigIntPool.Get().(*big.Int)
	z.SetInt64(0)
	return z
}

// ReleaseBigInt returns a *big.Int to the pool.
// Objects exceeding MaxPooledBitLen are ignored to avoid
// holding memory with oversized unused slices.
func ReleaseBigInt(z *big.Int) {
	if z == nil {
		return
	}
	// Avoid keeping oversized objects in memory
	if z.BitLen() > MaxPooledBitLen {
		return // Let GC collect it
	}
	bigIntPool.Put(z)
}
