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
// Task Concurrency Limiter
// ─────────────────────────────────────────────────────────────────────────────

// taskSemaphore limits the number of concurrent goroutines for multiplication
// and squaring tasks. This prevents excessive goroutine creation which can
// lead to contention and increased memory pressure.
var taskSemaphore chan struct{}
var taskSemaphoreOnce sync.Once

// getTaskSemaphore returns a semaphore limiting Fibonacci-level parallelism
// to NumCPU*2 goroutines. This is independent from the FFT-level semaphore
// (bigfft/fft_recursion.go, NumCPU goroutines). When both are active, up to
// NumCPU*3 goroutines may be active simultaneously. This is mitigated by
// ShouldParallelizeMultiplication() which disables Fibonacci-level parallelism
// when FFT is active (except for operands > ParallelFFTThreshold = 5M bits).
func getTaskSemaphore() chan struct{} {
	taskSemaphoreOnce.Do(func() {
		taskSemaphore = make(chan struct{}, runtime.NumCPU()*2)
	})
	return taskSemaphore
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
// error encountered. Each goroutine checks for context cancellation before
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
	var wg sync.WaitGroup
	var ec parallel.ErrorCollector
	wg.Add(3)

	for _, op := range [3]func() error{op1, op2, op3} {
		go func(fn func() error) {
			defer wg.Done()
			if err := ctx.Err(); err != nil {
				ec.SetError(fmt.Errorf("canceled before parallel operation: %w", err))
				return
			}
			ec.SetError(fn())
		}(op)
	}

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
}](tasks []T, inParallel bool) error {
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
				// Acquire semaphore token to limit concurrency
				sem <- struct{}{}
				defer func() { <-sem }()
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
func executeMixedTasks(sqrTasks []squaringTask, mulTasks []multiplicationTask, inParallel bool) error {
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
				// Acquire semaphore token to limit concurrency
				sem <- struct{}{}
				defer func() { <-sem }()
				ec.SetError(t.execute())
			}(&sqrTasks[i])
		}

		// Execute multiplication tasks in parallel
		for i := range mulTasks {
			go func(t *multiplicationTask) {
				defer wg.Done()
				// Acquire semaphore token to limit concurrency
				sem <- struct{}{}
				defer func() { <-sem }()
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
