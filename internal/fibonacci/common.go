package fibonacci

import (
	"context"
	"fmt"
	"math/big"
	"runtime"
	"sync"

	"golang.org/x/sync/errgroup"
)

// ─────────────────────────────────────────────────────────────────────────────
// Task Concurrency Limiter (P2-02)
// ─────────────────────────────────────────────────────────────────────────────

// globalSem is the single Fibonacci-level task semaphore. Every parallel
// helper in this package (executeParallel3, executeTasks, executeMixedTasks)
// acquires a token from this channel before spawning real work, ensuring
// that across all three entry points we never oversubscribe the CPU.
//
// Before P2-02 each helper referenced the semaphore via getTaskSemaphore()
// separately; the indirection obscured that they shared a single pool and
// left a refactor footgun — anyone adding a new parallel helper could
// accidentally create a second semaphore. The rename and the direct
// variable make the invariant obvious.
//
// Sizing: runtime.GOMAXPROCS(0). The previous value was NumCPU*2, which
// allowed oversubscription when big.Int arithmetic is already
// CPU-saturating (e.g. FFT paths). GOMAXPROCS(0) tracks the actual parallel
// capacity the runtime is allowed to use and matches the bigfft FFT-level
// semaphore at the same tier. The two semaphores remain separate because
// collapsing them across packages would require internal/bigfft to import
// internal/fibonacci — a layering inversion.
//
// globalSem is lazily initialized on first use so tests that adjust
// GOMAXPROCS before importing this package still get the correct sizing.
var (
	globalSem     chan struct{}
	globalSemOnce sync.Once
)

// getTaskSemaphore returns globalSem, initializing it on first call.
// Kept as a function (rather than a package-level var) so we pick up the
// runtime.GOMAXPROCS(0) value after test harnesses adjust it.
func getTaskSemaphore() chan struct{} {
	globalSemOnce.Do(func() {
		globalSem = make(chan struct{}, runtime.GOMAXPROCS(0))
	})
	return globalSem
}

// MaxPooledBitLen is the maximum size, in BITS, of a big.Int accepted into
// the sync.Pool. Larger objects are left for GC. The value allows pooling of
// intermediate results for large Fibonacci calculations (e.g. F(10^8)),
// avoiding repeated allocation of multi-megabyte big.Int values.
//
// A-16: this is a *bit*-length cap and is distinct from
// fastdoubling.maxArenaPoolWords, which is a *word*-count cap on the retained
// arena. The two are numerically equal (50_000_000) but expressed in different
// units (bits here, machine words there); they are independent knobs and must
// not be assumed interchangeable when tuning. 50_000_000 bits ≈ 6.25 MB.
const MaxPooledBitLen = 50_000_000

// checkLimit checks if a big.Int exceeds the maximum pooled bit length.
// This is used to prevent the pool from holding onto excessively large objects.
func checkLimit(z *big.Int) bool {
	return z != nil && z.BitLen() > MaxPooledBitLen
}

// ─────────────────────────────────────────────────────────────────────────────
// Parallel Execution Helper
// ─────────────────────────────────────────────────────────────────────────────

// parallel3Result bundles the WaitGroup and three per-operation error slots
// used by executeParallel3 so the whole call needs exactly one heap
// allocation. Three separate escaping locals (a *sync.WaitGroup plus three
// *error) would each be allocated independently by the compiler; grouping
// them into one struct and taking its address once collapses that to one.
// executeParallel3 runs once per non-FFT parallel doubling step on the
// FastDoubling hot path — benchstat caught this on FastDoubling/1M.
type parallel3Result struct {
	wg               sync.WaitGroup
	err1, err2, err3 error
}

// runParallel3Op is the per-goroutine worker body for executeParallel3. It
// acquires a semaphore token, checks for context cancellation, then runs op,
// storing the outcome in *errOut (a field of the caller's parallel3Result).
func runParallel3Op(ctx context.Context, sem chan struct{}, r *parallel3Result, errOut *error, op func() error) { //nolint:gocritic // *error is deliberate: errOut is written from a different goroutine than the caller that reads it, a plain error return can't do that.
	defer r.wg.Done()
	sem <- struct{}{}
	defer func() { <-sem }()
	if err := ctx.Err(); err != nil {
		*errOut = fmt.Errorf("canceled before parallel operation: %w", err)
		return
	}
	*errOut = op()
}

// executeParallel3 runs three operations concurrently, returning the first
// error encountered. Each goroutine acquires a semaphore token to respect
// the global concurrency limit, then checks for context cancellation before
// starting its operation. The caller is responsible for ensuring that the
// three operations write to disjoint memory (no shared mutable state).
//
// Parameters:
//   - ctx: The context for cancellation checking before each operation.
//   - op1, op2, op3: The operations to execute concurrently.
//
// Returns:
//   - error: The first error from any operation, or a context error.
func executeParallel3(ctx context.Context, op1, op2, op3 func() error) error {
	sem := getTaskSemaphore()
	r := &parallel3Result{}
	r.wg.Add(3)

	go runParallel3Op(ctx, sem, r, &r.err1, op1)
	go runParallel3Op(ctx, sem, r, &r.err2, op2)
	go runParallel3Op(ctx, sem, r, &r.err3, op3)

	r.wg.Wait()
	if r.err1 != nil {
		return r.err1
	}
	if r.err2 != nil {
		return r.err2
	}
	return r.err3
}

// task defines a common interface for executable tasks.
// This allows using generics to eliminate code duplication between
// multiplication and squaring task execution.
type task interface {
	execute() error
}

// multiplicationTask represents a single multiplication operation
// to be executed either sequentially or in parallel.
type multiplicationTask struct {
	dest         **big.Int
	a, b         *big.Int
	fftThreshold int
}

// execute performs the multiplication task.
func (t *multiplicationTask) execute() error {
	var err error
	*t.dest, err = smartMultiply(*t.dest, t.a, t.b, t.fftThreshold)
	return err
}

// squaringTask represents a single squaring operation (x * x)
// to be executed either sequentially or in parallel.
// Squaring is optimized compared to general multiplication because
// it exploits the symmetry of the computation.
type squaringTask struct {
	dest         **big.Int
	x            *big.Int
	fftThreshold int
}

// execute performs the squaring task.
func (t *squaringTask) execute() error {
	var err error
	*t.dest, err = smartSquare(*t.dest, t.x, t.fftThreshold)
	return err
}

// executeTasks executes a batch of tasks (multiplication or squaring) either
// sequentially or in parallel based on the inParallel flag.
// This generic function eliminates code duplication between different task types
// by using Go 1.18+ generics with a pointer constraint pattern.
//
// Type Parameters:
//   - T: The value type of the task (e.g., multiplicationTask, squaringTask).
//   - PT: A pointer type to T that implements the task interface.
//
// Parameters:
//   - tasks: The slice of tasks to execute (values, not pointers).
//   - inParallel: Whether to execute tasks in parallel.
//
// Returns:
//   - error: An error if any task failed.
func executeTasks[T any, PT interface {
	*T
	task
}](ctx context.Context, tasks []T, inParallel bool) error {
	if inParallel {
		sem := getTaskSemaphore()
		var g errgroup.Group
		for i := range tasks {
			t := PT(&tasks[i])
			g.Go(func() error {
				// Acquire semaphore token to limit concurrency.
				sem <- struct{}{}
				defer func() { <-sem }()
				// Check for context cancellation after acquiring the semaphore:
				// the token may have been held for a while during contention,
				// and the context could have been canceled in the meantime.
				// Skipping this check would execute expensive multiplications
				// after the caller has already abandoned the computation.
				if err := ctx.Err(); err != nil {
					return err
				}
				return t.execute()
			})
		}
		return g.Wait()
	}
	for i := range tasks {
		if err := PT(&tasks[i]).execute(); err != nil {
			return err
		}
	}
	return nil
}

// executeMixedTasks executes a mix of squaring and multiplication tasks together,
// either sequentially or in parallel. This eliminates code duplication when
// both types of operations need to be executed together.
//
// Parameters:
//   - sqrTasks: The squaring tasks to execute.
//   - mulTasks: The multiplication tasks to execute.
//   - inParallel: Whether to execute tasks in parallel.
//
// Returns:
//   - error: An error if any task failed.
func executeMixedTasks(ctx context.Context, sqrTasks []squaringTask, mulTasks []multiplicationTask, inParallel bool) error {
	if len(sqrTasks)+len(mulTasks) == 0 {
		return nil
	}

	if inParallel {
		sem := getTaskSemaphore()
		var g errgroup.Group

		// Execute squaring tasks in parallel
		for i := range sqrTasks {
			t := &sqrTasks[i]
			g.Go(func() error {
				// Acquire semaphore token to limit concurrency.
				sem <- struct{}{}
				defer func() { <-sem }()
				// Check for context cancellation after acquiring the semaphore
				// (the token may have been held for a while during contention).
				if err := ctx.Err(); err != nil {
					return err
				}
				return t.execute()
			})
		}

		// Execute multiplication tasks in parallel
		for i := range mulTasks {
			t := &mulTasks[i]
			g.Go(func() error {
				// Acquire semaphore token to limit concurrency.
				sem <- struct{}{}
				defer func() { <-sem }()
				// Check for context cancellation after acquiring the semaphore
				// (the token may have been held for a while during contention).
				if err := ctx.Err(); err != nil {
					return err
				}
				return t.execute()
			})
		}

		return g.Wait()
	}

	// Sequential execution
	for i := range sqrTasks {
		if err := sqrTasks[i].execute(); err != nil {
			return err
		}
	}
	for i := range mulTasks {
		if err := mulTasks[i].execute(); err != nil {
			return err
		}
	}
	return nil
}
