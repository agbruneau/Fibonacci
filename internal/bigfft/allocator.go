// This file provides the tempAllocator interface for unified temporary buffer management.

package bigfft

import "math/big"

// tempAllocator abstracts temporary fermat buffer allocation.
// This interface allows the FFT algorithm to work with different allocation
// strategies (sync.Pool, bump allocator) without code duplication.
//
// It is deliberately unexported: its whole method set is unexported, so no
// type outside bigfft can implement it anyway.
//
// Every method of this interface names its results, here and in each
// implementation: gocritic's unnamedResult requires it for the
// (fermat, func()) shape, and the rest of the set follows the same rule so
// the cleanup contract reads identically at every declaration.
type tempAllocator interface {
	// allocFermatTemp allocates a temporary fermat buffer of size n+1.
	// Returns the buffer and a cleanup function that should be called
	// when the buffer is no longer needed.
	//
	// Parameters:
	//   - n: The n parameter for fermat (resulting slice has n+1 elements).
	//
	// Returns:
	//   - fermat: A zeroed fermat slice.
	//   - func(): Cleanup function (may be no-op for some allocators).
	allocFermatTemp(n int) (buf fermat, cleanup func())

	// allocFermatSlice allocates count fermat numbers, each of size n+1.
	// Returns both the slice of fermat references and the backing word buffer.
	//
	// Parameters:
	//   - count: Number of fermat slices to allocate.
	//   - n: The n parameter for each fermat (each slice has n+1 elements).
	//
	// Returns:
	//   - []fermat: Slice of count fermat references.
	//   - []big.Word: The backing buffer (for potential release tracking).
	//   - func(): Cleanup function.
	allocFermatSlice(count, n int) (fermats []fermat, bits []big.Word, cleanup func())
}

// poolAllocator implements tempAllocator using sync.Pool.
// This is the default allocator when no bump allocator is available.
type poolAllocator struct{}

// allocFermatTemp allocates a fermat buffer from the pool.
// The cleanup function returns the buffer to the pool.
//
// The cleanup function should be called with defer immediately after allocation:
//
//	f, cleanup := allocator.allocFermatTemp(n)
//	defer cleanup()
//
// This ensures the buffer is returned to the pool even if an error occurs.
func (p *poolAllocator) allocFermatTemp(n int) (buf fermat, cleanup func()) {
	f := acquireFermat(n + 1)
	return f, func() { releaseFermat(f) }
}

// allocFermatSlice allocates count fermat numbers using pooled buffers.
//
// The cleanup function should be called with defer immediately after allocation:
//
//	fermats, bits, cleanup := allocator.allocFermatSlice(count, n)
//	defer cleanup()
//
// This ensures all buffers are returned to the pool even if an error occurs.
// The results are named for the interface contract only; the buffers live in
// locals so the cleanup closure captures them by value. Assigning them to the
// named results instead makes each captured slice header escape on its own
// (3 -> 5 allocations per call, caught by TestPoolAllocatorSliceAllocsSteadyState).
func (p *poolAllocator) allocFermatSlice(count, n int) (fermats []fermat, bits []big.Word, cleanup func()) {
	wordCount := count * (n + 1)
	words := acquireWordSlice(wordCount)
	bufs := acquireFermatSlice(count)

	// Initialize bufs to point to words slices
	for i := 0; i < count; i++ {
		bufs[i] = fermat(words[i*(n+1) : (i+1)*(n+1)])
	}

	return bufs, words, func() {
		releaseWordSlice(words)
		releaseFermatSlice(bufs)
	}
}

// Note: *BumpAllocator implements tempAllocator directly (see bump.go).
// The previous BumpAllocatorAdapter wrapper has been removed; pass a
// *BumpAllocator wherever a tempAllocator is expected.

// defaultPoolAllocator is the shared poolAllocator instance, used wherever no
// bump allocator is available.
var defaultPoolAllocator = &poolAllocator{}
