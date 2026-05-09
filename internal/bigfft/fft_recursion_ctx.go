// Context-aware sister of fourierRecursiveUnified that routes goroutine
// admission through an FFTContext-owned semaphore instead of the
// package-global concurrencySemaphore. This is the only place where the
// minimal FFTContext changes the recursion path; the global path (used by
// the historical Mul/Sqr entry points) is left untouched.

package bigfft

import (
	"fmt"
	"sync"
)

// fourierWithBumpCtx is the context-aware twin of fourierWithBump.
// It performs the Fourier transform using the bump allocator for temporary
// buffers and the supplied FFTContext for goroutine admission control.
func fourierWithBumpCtx(ctx *FFTContext, dst []fermat, src []fermat, backward bool, n int, k uint, ba *BumpAllocator) error {
	tmp := ba.AllocFermat(n)
	tmp2 := ba.AllocFermat(n)
	return fourierRecursiveCtx(ctx, dst, src, backward, n, k, k, 0, tmp, tmp2, ba)
}

// fourierRecursiveCtx mirrors fourierRecursiveUnified but uses
// ctx.Semaphore for parallel admission. Sequential structure is identical.
func fourierRecursiveCtx(ctx *FFTContext, dst, src []fermat, backward bool, n int, k, size, depth uint, tmp, tmp2 fermat, alloc TempAllocator) error {
	idxShift := k - size
	ω2shift := (4 * n * _W) >> size
	if backward {
		ω2shift = -ω2shift
	}

	if len(src[0]) != n+1 || len(dst[0]) != n+1 {
		return fmt.Errorf("FFT recursion validation failed: len(src[0])=%d, len(dst[0])=%d, expected n+1=%d", len(src[0]), len(dst[0]), n+1)
	}

	switch size {
	case 0:
		copy(dst[0], src[0])
		return nil
	case 1:
		dst[0].Add(src[0], src[1<<idxShift])
		dst[1].Sub(src[0], src[1<<idxShift])
		return nil
	}

	dst1 := dst[:1<<(size-1)]
	dst2 := dst[1<<(size-1):]

	if size >= ParallelFFTRecursionThreshold && depth < MaxParallelFFTDepth {
		select {
		case ctx.Semaphore <- struct{}{}:
			var wg sync.WaitGroup
			wg.Add(1)
			var errAsync error
			go func() {
				defer wg.Done()
				defer func() { <-ctx.Semaphore }()

				// Parallel goroutines use the global pool allocator for
				// temps to avoid races on non-thread-safe BumpAllocator.
				t1, cleanup1 := GetPoolAllocator().AllocFermatTemp(n)
				t2, cleanup2 := GetPoolAllocator().AllocFermatTemp(n)
				defer cleanup1()
				defer cleanup2()

				errAsync = fourierRecursiveCtx(ctx, dst2, src[1<<idxShift:], backward, n, k, size-1, depth+1, t1, t2, alloc)
			}()

			errSync := fourierRecursiveCtx(ctx, dst1, src, backward, n, k, size-1, depth+1, tmp, tmp2, alloc)

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

	if err := fourierRecursiveCtx(ctx, dst1, src, backward, n, k, size-1, depth+1, tmp, tmp2, alloc); err != nil {
		return err
	}
	if err := fourierRecursiveCtx(ctx, dst2, src[1<<idxShift:], backward, n, k, size-1, depth+1, tmp, tmp2, alloc); err != nil {
		return err
	}
	return executeReconstruction(dst1, dst2, ω2shift, tmp, tmp2)
}
