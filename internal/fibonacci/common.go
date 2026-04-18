package fibonacci

import (
	"context"
	"fmt"
	"math/big"
	"runtime"
	"sync"

	"github.com/agbru/fibcalc/internal/parallel"
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
// Sizing: runtime.NumCPU(). The previous value was NumCPU*2, which
// allowed oversubscription when big.Int arithmetic is already
// CPU-saturating (e.g. FFT paths). NumCPU tracks the actual parallel
// capacity of the machine and matches the bigfft FFT-level semaphore at
// the same tier. The two semaphores remain separate because collapsing
// them across packages would require internal/bigfft to import
// internal/fibonacci — a layering inversion.
//
// globalSem is lazily initialised on first use so tests that mutate
// runtime.NumCPU (via GOMAXPROCS) before importing this package still
// get the correct sizing.
var (
	globalSem     chan struct{}
	globalSemOnce sync.Once
)

// getTaskSemaphore returns globalSem, initialising it on first call.
// Kept as a function (rather than a package-level var) so we pick up the
// runtime.NumCPU() value after test harnesses adjust GOMAXPROCS.
func getTaskSemaphore() chan struct{} {
	globalSemOnce.Do(func() {
		globalSem = make(chan struct{}, runtime.NumCPU())
	})
	return globalSem
}

// MaxPooledBitLen is the maximum size (in bits) of a big.Int
// accepted into the pool. Larger objects are left for GC collection.
// Increased to 100M bits (~12.5 MB) to allow pooling of intermediate results
// for large Fibonacci calculations (e.g., F(10^8)), avoiding repeated
// allocation of multi-megabyte big.Int values.
const MaxPooledBitLen = 50_000_000

// checkLimit checks if a big.Int exceeds the maximum pooled bit length.
// This is used to prevent the pool from holding onto excessively large objects.
func checkLimit(z *big.Int) bool {
	return z != nil && z.BitLen() > MaxPooledBitLen
}

// preSizeBigInt ensures a big.Int has at least the specified word capacity.
// This avoids repeated reallocation during the doubling loop as values grow.
// Uses SetBits with a length-0 capacity-N slice to pre-allocate without
// changing the numeric value.
func preSizeBigInt(z *big.Int, words int) {
	if z == nil || words <= 0 {
		return
	}
	// Only pre-size if current capacity is smaller
	if cap(z.Bits()) >= words {
		return
	}
	// SetBits([]big.Word{}) with length 0 sets z to 0.
	// We use a slice with length=0, cap=words to give z the backing array.
	buf := make([]big.Word, 0, words)
	z.SetBits(buf)
}

// ─────────────────────────────────────────────────────────────────────────────
// Logging
// ─────────────────────────────────────────────────────────────────────────────

// taskLogger is the package-level logger for parallel task distribution.
// Defaults to zerolog.Nop() (no output) to avoid performance impact.
var taskLogger = zerolog.Nop()

// SetTaskLogger configures the logger used for parallel task distribution decisions.
func SetTaskLogger(l zerolog.Logger) {
	taskLogger = l
}

// ─────────────────────────────────────────────────────────────────────────────
// Parallel Execution Helper
// ─────────────────────────────────────────────────────────────────────────────

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
	var wg sync.WaitGroup
	var ec parallel.ErrorCollector
	wg.Add(3)

	go func() {
		sem <- struct{}{}
		if err := ctx.Err(); err != nil {
			ec.SetError(fmt.Errorf("canceled before parallel operation: %w", err))
			<-sem
			wg.Done()
			return
		}
		ec.SetError(op1())
		<-sem
		wg.Done()
	}()

	go func() {
		sem <- struct{}{}
		if err := ctx.Err(); err != nil {
			ec.SetError(fmt.Errorf("canceled before parallel operation: %w", err))
			<-sem
			wg.Done()
			return
		}
		ec.SetError(op2())
		<-sem
		wg.Done()
	}()

	go func() {
		sem <- struct{}{}
		if err := ctx.Err(); err != nil {
			ec.SetError(fmt.Errorf("canceled before parallel operation: %w", err))
			<-sem
			wg.Done()
			return
		}
		ec.SetError(op3())
		<-sem
		wg.Done()
	}()

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
