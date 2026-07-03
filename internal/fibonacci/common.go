package fibonacci

import (
	"context"
	"fmt"
	"math/big"
	"runtime"
	"sync"

	"github.com/agbruneau/FibGo/internal/parallel"
	"github.com/rs/zerolog"
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
// Logging
// ─────────────────────────────────────────────────────────────────────────────

// taskLogger is the package-level logger for parallel task distribution.
// Defaults to zerolog.Nop() (no output) to avoid performance impact. It is not
// currently wired to a configurable sink (no SetTaskLogger): the Debug calls
// below are kept as ready instrumentation that logs to Nop until a future
// app-level wiring opts in.
var taskLogger = zerolog.Nop()

// ─────────────────────────────────────────────────────────────────────────────
// Parallel Execution Helper
// ─────────────────────────────────────────────────────────────────────────────

// runParallel3Op is the per-goroutine worker body for executeParallel3.
// Taking wg/ec/sem/ctx/op explicitly as parameters (rather than closing
// over them in a function literal) keeps the goroutine's closure empty
// and eliminates the func-literal heap allocation per call.
//
// wg and ec are still heap-allocated in executeParallel3 itself because
// they are shared across goroutines — Go's escape analysis cannot prove
// they outlive the spawned goroutines — but the per-call closure
// allocation disappears.
//
// See P2-03.
func runParallel3Op(ctx context.Context, sem chan struct{}, wg *sync.WaitGroup, ec *parallel.ErrorCollector, op func() error) {
	sem <- struct{}{}
	defer func() {
		<-sem
		wg.Done()
	}()
	if err := ctx.Err(); err != nil {
		ec.SetError(fmt.Errorf("canceled before parallel operation: %w", err))
		return
	}
	ec.SetError(op())
}

// executeParallel3 runs three operations concurrently, returning the first
// error encountered. Each goroutine acquires a semaphore token to respect
// the global concurrency limit, then checks for context cancellation before
// starting its operation. The caller is responsible for ensuring that the
// three operations write to disjoint memory (no shared mutable state).
//
// P2-03: per-goroutine body is extracted to runParallel3Op so the
// `go` statements do not need function literals capturing wg/ec/sem.
// wg and ec still heap-allocate (they are shared across goroutines
// that may outlive the stack frame under Go's current escape analysis)
// but the three per-call closures no longer allocate. Verified with
// `go build -gcflags="-m=2"`.
//
// Parameters:
//   - ctx: The context for cancellation checking before each operation.
//   - op1, op2, op3: The operations to execute concurrently.
//
// Returns:
//   - error: The first error from any operation, or a context error.
func executeParallel3(ctx context.Context, op1, op2, op3 func() error) error {
	sem := getTaskSemaphore()
	wg := &sync.WaitGroup{}
	ec := &parallel.ErrorCollector{}
	wg.Add(3)

	go runParallel3Op(ctx, sem, wg, ec, op1)
	go runParallel3Op(ctx, sem, wg, ec, op2)
	go runParallel3Op(ctx, sem, wg, ec, op3)

	wg.Wait()
	return ec.Err()
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
	taskLogger.Debug().
		Int("task_count", len(tasks)).
		Bool("parallel", inParallel).
		Msg("executing tasks")
	if inParallel {
		sem := getTaskSemaphore()
		var wg sync.WaitGroup
		var ec parallel.ErrorCollector
		wg.Add(len(tasks))
		for i := range tasks {
			go func(t PT) {
				defer wg.Done()
				// Acquire semaphore token to limit concurrency.
				sem <- struct{}{}
				defer func() { <-sem }()
				// Check for context cancellation after acquiring the semaphore:
				// the token may have been held for a while during contention,
				// and the context could have been canceled in the meantime.
				// Skipping this check would execute expensive multiplications
				// after the caller has already abandoned the computation.
				if err := ctx.Err(); err != nil {
					ec.SetError(err)
					return
				}
				ec.SetError(t.execute())
			}(PT(&tasks[i]))
		}
		wg.Wait()
		return ec.Err()
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
	totalTasks := len(sqrTasks) + len(mulTasks)
	if totalTasks == 0 {
		return nil
	}

	taskLogger.Debug().
		Int("sqr_tasks", len(sqrTasks)).
		Int("mul_tasks", len(mulTasks)).
		Int("total_tasks", totalTasks).
		Bool("parallel", inParallel).
		Msg("executing mixed tasks")
	if inParallel {
		sem := getTaskSemaphore()
		var wg sync.WaitGroup
		var ec parallel.ErrorCollector
		wg.Add(totalTasks)

		// Execute squaring tasks in parallel
		for i := range sqrTasks {
			go func(t *squaringTask) {
				defer wg.Done()
				// Acquire semaphore token to limit concurrency.
				sem <- struct{}{}
				defer func() { <-sem }()
				// Check for context cancellation after acquiring the semaphore
				// (the token may have been held for a while during contention).
				if err := ctx.Err(); err != nil {
					ec.SetError(err)
					return
				}
				ec.SetError(t.execute())
			}(&sqrTasks[i])
		}

		// Execute multiplication tasks in parallel
		for i := range mulTasks {
			go func(t *multiplicationTask) {
				defer wg.Done()
				// Acquire semaphore token to limit concurrency.
				sem <- struct{}{}
				defer func() { <-sem }()
				// Check for context cancellation after acquiring the semaphore
				// (the token may have been held for a while during contention).
				if err := ctx.Err(); err != nil {
					ec.SetError(err)
					return
				}
				ec.SetError(t.execute())
			}(&mulTasks[i])
		}

		wg.Wait()
		return ec.Err()
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
