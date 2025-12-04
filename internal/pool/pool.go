// Package pool provides a global memory pool for big.Int reuse.
// This helps reduce GC pressure by recycling big.Int objects across
// different parts of the application.
package pool

import (
	"math/big"
	"sync"
)

// bigIntPool is a sync.Pool for *big.Int objects.
var bigIntPool = sync.Pool{
	New: func() interface{} {
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
func ReleaseBigInt(z *big.Int) {
	if z != nil {
		bigIntPool.Put(z)
	}
}
