package bigfft

import (
	"fmt"
	"runtime"
	"sync"
)

// concurrencySemaphore is a buffered channel used to limit the number of
// concurrent goroutines in the FFT recursion.
var concurrencySemaphore chan struct{}
var concurrencyOnce sync.Once

// getSemaphore returns the global concurrency semaphore for FFT recursion,
// initializing it to runtime.NumCPU() on the first call. This is independent
// from the Fibonacci-level task semaphore (fibonacci/common.go, NumCPU*2).
// See getTaskSemaphore documentation for interaction details.
func getSemaphore() chan struct{} {
	concurrencyOnce.Do(func() {
		concurrencySemaphore = make(chan struct{}, runtime.NumCPU())
	})
	return concurrencySemaphore
}

// ParallelFFTRecursionThreshold is the minimum size (in bits of k, where K=2^k)
// for which FFT recursion should be parallelized. Below this threshold, the
// overhead of goroutine creation exceeds the benefits of parallelism.
var ParallelFFTRecursionThreshold uint = 4

// MaxParallelFFTDepth limits the maximum depth of parallel recursion to avoid
// excessive goroutine creation. Once this depth is reached, recursion continues
// sequentially.
var MaxParallelFFTDepth uint = 3

// FFTParallelismConfig holds the configurable FFT parallelism thresholds.
type FFTParallelismConfig struct {
	// RecursionThreshold is the minimum FFT size (as log2) for parallel recursion.
	RecursionThreshold uint
	// MaxDepth is the maximum depth of parallel recursion.
	MaxDepth uint
}

// SetFFTParallelismConfig updates the FFT parallelism thresholds.
// This allows runtime calibration of parallelism behavior.
func SetFFTParallelismConfig(config FFTParallelismConfig) {
	if config.RecursionThreshold > 0 {
		ParallelFFTRecursionThreshold = config.RecursionThreshold
	}
	if config.MaxDepth > 0 {
		MaxParallelFFTDepth = config.MaxDepth
	}
}

// GetFFTParallelismConfig returns the current FFT parallelism configuration.
func GetFFTParallelismConfig() FFTParallelismConfig {
	return FFTParallelismConfig{
		RecursionThreshold: ParallelFFTRecursionThreshold,
		MaxDepth:           MaxParallelFFTDepth,
	}
}

// fourierRecursiveUnified is the unified recursive FFT function that works with
// any TempAllocator implementation. This eliminates code duplication between
// pool-based and bump-allocator-based variants.
//
// Parameters:
//   - dst: destination slice for FFT results
//   - src: source slice of fermat numbers
//   - backward: true for inverse transform
//   - n: coefficient length
//   - k: log2 of FFT size
//   - size: current recursion size
//   - depth: current recursion depth
//   - tmp, tmp2: temporary buffers for this goroutine
//   - alloc: allocator for creating new temp buffers in parallel goroutines
func fourierRecursiveUnified(dst, src []fermat, backward bool, n int, k, size, depth uint, tmp, tmp2 fermat, alloc TempAllocator) error {
	idxShift := k - size
	ω2shift := (4 * n * _W) >> size
	if backward {
		ω2shift = -ω2shift
	}

	// Validation
	if len(src[0]) != n+1 || len(dst[0]) != n+1 {
		return fmt.Errorf("FFT recursion validation failed: len(src[0])=%d, len(dst[0])=%d, expected n+1=%d", len(src[0]), len(dst[0]), n+1)
	}

	// Base cases
	switch size {
	case 0:
		copy(dst[0], src[0])
		return nil
	case 1:
		dst[0].Add(src[0], src[1<<idxShift])
		dst[1].Sub(src[0], src[1<<idxShift])
		return nil
	}

	// Split destination vectors in halves
	dst1 := dst[:1<<(size-1)]
	dst2 := dst[1<<(size-1):]

	// Try to acquire token for parallelism
	// We only try to parallelize if the size is large enough to justify overhead
	// and we haven't exceeded the maximum parallelism depth
	if size >= ParallelFFTRecursionThreshold && depth < MaxParallelFFTDepth {
		select {
		case getSemaphore() <- struct{}{}:
			// Got token, run second half in parallel
			var wg sync.WaitGroup
			wg.Add(1)
			var errAsync error
			go func() {
				defer wg.Done()
				defer func() { <-getSemaphore() }()

				// Allocate new temps for this branch using the allocator
				// For parallel goroutines, we always use pool to avoid race conditions
				// on non-thread-safe bump allocators
				t1, cleanup1 := GetPoolAllocator().AllocFermatTemp(n)
				t2, cleanup2 := GetPoolAllocator().AllocFermatTemp(n)
				defer cleanup1()
				defer cleanup2()

				errAsync = fourierRecursiveUnified(dst2, src[1<<idxShift:], backward, n, k, size-1, depth+1, t1, t2, alloc)
			}()

			// Run first half in current thread with current temps
			errSync := fourierRecursiveUnified(dst1, src, backward, n, k, size-1, depth+1, tmp, tmp2, alloc)

			wg.Wait()
			if errAsync != nil {
				return errAsync
			}
			if errSync != nil {
				return errSync
			}
			return executeReconstruction(dst1, dst2, ω2shift, tmp, tmp2)
		default:
			// Fallthrough to sequential
		}
	}

	// Recursive calls (Sequential)
	if err := fourierRecursiveUnified(dst1, src, backward, n, k, size-1, depth+1, tmp, tmp2, alloc); err != nil {
		return err
	}
	if err := fourierRecursiveUnified(dst2, src[1<<idxShift:], backward, n, k, size-1, depth+1, tmp, tmp2, alloc); err != nil {
		return err
	}
	return executeReconstruction(dst1, dst2, ω2shift, tmp, tmp2)
}

// executeReconstruction applies the butterfly reconstruction step, combining
// the two halves of the FFT transform using the twiddle factor shift.
func executeReconstruction(dst1, dst2 []fermat, ω2shift int, tmp, tmp2 fermat) error {
	for i := range dst1 {
		tmp.ShiftHalf(dst2[i], i*ω2shift, tmp2)
		dst2[i].Sub(dst1[i], tmp)
		dst1[i].Add(dst1[i], tmp)
	}
	return nil
}

// fourierRecursive is a convenience wrapper that uses pool allocation.
// Kept for backward compatibility.
func fourierRecursive(dst, src []fermat, backward bool, n int, k, size, depth uint, tmp, tmp2 fermat) error {
	return fourierRecursiveUnified(dst, src, backward, n, k, size, depth, tmp, tmp2, GetPoolAllocator())
}
