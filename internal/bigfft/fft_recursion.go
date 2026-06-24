package bigfft

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
)

// concurrencySemaphore is a buffered channel used to limit the number of
// concurrent goroutines in the FFT recursion.
var concurrencySemaphore chan struct{}
var concurrencyOnce sync.Once

// getSemaphore returns the global concurrency semaphore for FFT recursion,
// initializing it to runtime.NumCPU() on the first call. This is independent
// from the Fibonacci-level task semaphore (fibonacci/common.go, NumCPU).
// See getTaskSemaphore documentation for interaction details.
func getSemaphore() chan struct{} {
	concurrencyOnce.Do(func() {
		concurrencySemaphore = make(chan struct{}, runtime.NumCPU())
	})
	return concurrencySemaphore
}

// parallelFFTRecursionThreshold is the minimum size (in bits of k, where K=2^k)
// for which FFT recursion should be parallelized. Below this threshold, the
// overhead of goroutine creation exceeds the benefits of parallelism.
//
// Held in an atomic.Uint64 to ensure thread-safe runtime updates via
// SetFFTParallelismConfig while hot-path readers stay lock-free.
var parallelFFTRecursionThreshold atomic.Uint64

// maxParallelFFTDepth limits the maximum depth of parallel recursion to avoid
// excessive goroutine creation. Once this depth is reached, recursion continues
// sequentially. See parallelFFTRecursionThreshold for the concurrency contract.
var maxParallelFFTDepth atomic.Uint64

func init() {
	parallelFFTRecursionThreshold.Store(4)
	maxParallelFFTDepth.Store(3)
}

// GetParallelFFTRecursionThreshold returns the current parallel-recursion
// activation threshold. Hot-path readers within bigfft must call this accessor.
func GetParallelFFTRecursionThreshold() uint { return uint(parallelFFTRecursionThreshold.Load()) }

// GetMaxParallelFFTDepth returns the current maximum parallel recursion depth.
func GetMaxParallelFFTDepth() uint { return uint(maxParallelFFTDepth.Load()) }

// FFTParallelismConfig holds the configurable FFT parallelism thresholds.
type FFTParallelismConfig struct {
	// RecursionThreshold is the minimum FFT size (as log2) for parallel recursion.
	RecursionThreshold uint
	// MaxDepth is the maximum depth of parallel recursion.
	MaxDepth uint
}

// SetFFTParallelismConfig updates the FFT parallelism thresholds atomically.
// This allows runtime calibration of parallelism behavior; concurrent readers
// observe either the pre- or post-update value, never a torn write.
func SetFFTParallelismConfig(config FFTParallelismConfig) {
	if config.RecursionThreshold > 0 {
		parallelFFTRecursionThreshold.Store(uint64(config.RecursionThreshold))
	}
	if config.MaxDepth > 0 {
		maxParallelFFTDepth.Store(uint64(config.MaxDepth))
	}
}

// GetFFTParallelismConfig returns the current FFT parallelism configuration.
func GetFFTParallelismConfig() FFTParallelismConfig {
	return FFTParallelismConfig{
		RecursionThreshold: GetParallelFFTRecursionThreshold(),
		MaxDepth:           GetMaxParallelFFTDepth(),
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
//
//nolint:gocognit // dispatch récursion FFT : branche séquentielle + branche parallèle (token non bloquant) avec capture/re-propagation des panics worker (ADR-0002) ; la scinder masquerait le hot path, comportement épinglé par TestFourierRecursive*/golden.
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
	if size >= GetParallelFFTRecursionThreshold() && depth < GetMaxParallelFFTDepth() {
		select {
		case getSemaphore() <- struct{}{}:
			// Got token, run second half in parallel
			var wg sync.WaitGroup
			wg.Add(1)
			var errAsync error
			panicCh := make(chan any, 1)
			go func() {
				defer wg.Done()
				defer func() { <-getSemaphore() }()
				defer func() {
					if r := recover(); r != nil {
						select {
						case panicCh <- r:
						default:
						}
					}
				}()

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
			select {
			case r := <-panicCh:
				panic(r)
			default:
			}
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

// reconstructionMinParallelWords gates the parallel butterfly path: below
// this span (len(dst1) * coefficient words) goroutine dispatch costs more
// than the linear Shift/Sub/Add work it spreads. The butterflies of the
// top one or two recursion levels of a large transform sit above it; every
// level of a small transform stays sequential. Tuned by paired end-to-end
// benchmarks on a 24-thread host (2026-06).
const reconstructionMinParallelWords = 1 << 16

// executeReconstruction applies the butterfly reconstruction step, combining
// the two halves of the FFT transform using the twiddle factor shift.
//
// Each index i touches only dst1[i]/dst2[i], so for large spans the loop is
// chunked across goroutines bounded by the global FFT semaphore with a
// non-blocking acquire (same contract as fourierRecursiveUnified and
// runPointwise — no token means the chunk runs on the calling goroutine,
// so no deadlock against other token holders is possible). Parallel
// workers draw their tmp/tmp2 scratch from the pool allocator; the caller's
// tmp/tmp2 stay reserved for the chunks that run on the calling goroutine.
// Worker panics are re-panicked on the calling goroutine (ADR-0002).
func executeReconstruction(dst1, dst2 []fermat, ω2shift int, tmp, tmp2 fermat) error {
	body := func(lo, hi int, t1, t2 fermat) {
		for i := lo; i < hi; i++ {
			t1.ShiftHalf(dst2[i], i*ω2shift, t2)
			dst2[i].Sub(dst1[i], t1)
			dst1[i].Add(dst1[i], t1)
		}
	}

	if len(dst1)*len(tmp) < reconstructionMinParallelWords || runtime.NumCPU() == 1 {
		body(0, len(dst1), tmp, tmp2)
		return nil
	}

	workers := runtime.NumCPU()
	if workers > len(dst1) {
		workers = len(dst1)
	}
	chunk := (len(dst1) + workers - 1) / workers
	n := len(tmp) - 1

	sem := getSemaphore()
	var wg sync.WaitGroup
	panicCh := make(chan any, 1)

	// Chunk 0 is reserved for the calling goroutine (run below with the
	// caller's temps); the rest spawn when a semaphore token is free.
	for lo := chunk; lo < len(dst1); lo += chunk {
		hi := lo + chunk
		if hi > len(dst1) {
			hi = len(dst1)
		}
		select {
		case sem <- struct{}{}:
			wg.Add(1)
			go func(lo, hi int) {
				defer wg.Done()
				defer func() { <-sem }()
				defer func() {
					if r := recover(); r != nil {
						select {
						case panicCh <- r:
						default:
						}
					}
				}()
				t1, cleanup1 := GetPoolAllocator().AllocFermatTemp(n)
				t2, cleanup2 := GetPoolAllocator().AllocFermatTemp(n)
				defer cleanup1()
				defer cleanup2()
				body(lo, hi, t1, t2)
			}(lo, hi)
		default:
			body(lo, hi, tmp, tmp2)
		}
	}

	hi := chunk
	if hi > len(dst1) {
		hi = len(dst1)
	}
	body(0, hi, tmp, tmp2)

	wg.Wait()
	select {
	case r := <-panicCh:
		panic(r)
	default:
	}
	return nil
}

// fourierRecursive is a convenience wrapper that uses pool allocation.
// Kept for backward compatibility.
func fourierRecursive(dst, src []fermat, backward bool, n int, k, size, depth uint, tmp, tmp2 fermat) error {
	return fourierRecursiveUnified(dst, src, backward, n, k, size, depth, tmp, tmp2, GetPoolAllocator())
}
