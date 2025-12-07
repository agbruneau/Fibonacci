// Package pool provides a global memory pool for big.Int reuse.
// This helps reduce GC pressure by recycling big.Int objects across
// different parts of the application.
package pool

import (
	"math/big"
	"sync"
)

// MaxPooledBitLen est la taille maximale (en bits) d'un big.Int
// accepté dans le pool. Au-delà, on laisse le GC le ramasser.
// ~512 Ko de données.
const MaxPooledBitLen = 4_000_000

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
// Les objets dépassant MaxPooledBitLen sont ignorés pour éviter
// de bloquer la mémoire avec des slices géants inutilisés.
func ReleaseBigInt(z *big.Int) {
	if z == nil {
		return
	}
	// Éviter de garder des objets trop gros en mémoire
	if z.BitLen() > MaxPooledBitLen {
		return // Laisse le GC le ramasser
	}
	bigIntPool.Put(z)
}
