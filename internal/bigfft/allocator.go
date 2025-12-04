// Package bigfft implements multiplication of big.Int using FFT.
// This file provides the TempAllocator interface for unified temporary buffer management.
package bigfft

// TempAllocator abstracts temporary fermat buffer allocation.
// This interface allows the FFT algorithm to work with different allocation
// strategies (sync.Pool, bump allocator) without code duplication.
type TempAllocator interface {
	// AllocFermatTemp allocates a temporary fermat buffer of size n+1.
	// Returns the buffer and a cleanup function that should be called
	// when the buffer is no longer needed.
	//
	// Parameters:
	//   - n: The n parameter for fermat (resulting slice has n+1 elements).
	//
	// Returns:
	//   - fermat: A zeroed fermat slice.
	//   - func(): Cleanup function (may be no-op for some allocators).
	AllocFermatTemp(n int) (fermat, func())
}

// PoolAllocator implements TempAllocator using sync.Pool.
// This is the default allocator when no bump allocator is available.
type PoolAllocator struct{}

// AllocFermatTemp allocates a fermat buffer from the pool.
// The cleanup function returns the buffer to the pool.
func (p *PoolAllocator) AllocFermatTemp(n int) (fermat, func()) {
	f := acquireFermat(n + 1)
	return f, func() { releaseFermat(f) }
}

// BumpAllocatorAdapter adapts BumpAllocator to the TempAllocator interface.
// Since bump allocator releases all memory at once, cleanup is a no-op.
type BumpAllocatorAdapter struct {
	ba *BumpAllocator
}

// NewBumpAllocatorAdapter creates a new adapter wrapping the given BumpAllocator.
func NewBumpAllocatorAdapter(ba *BumpAllocator) *BumpAllocatorAdapter {
	return &BumpAllocatorAdapter{ba: ba}
}

// AllocFermatTemp allocates a fermat buffer from the bump allocator.
// The cleanup function is a no-op since bump allocator releases all at once.
func (b *BumpAllocatorAdapter) AllocFermatTemp(n int) (fermat, func()) {
	return b.ba.AllocFermat(n), func() {} // no-op cleanup
}

// defaultPoolAllocator is a shared instance of PoolAllocator.
var defaultPoolAllocator = &PoolAllocator{}

// GetPoolAllocator returns the shared PoolAllocator instance.
func GetPoolAllocator() TempAllocator {
	return defaultPoolAllocator
}

