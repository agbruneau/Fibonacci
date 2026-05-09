package fibonacci

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/agbru/fibcalc/internal/bigfft"
	apperrors "github.com/agbru/fibcalc/internal/errors"
	"github.com/agbru/fibcalc/internal/fibonacci/memory"
	"github.com/agbru/fibcalc/internal/progress"
	"github.com/rs/zerolog/log"
)

// MaxFibUint64 = 93 because F(93) is the largest Fibonacci number that fits in a uint64,
// as F(94) exceeds 2^64. This value is derived from the very rapid growth of the sequence.
const (
	MaxFibUint64 = 93 // Justified above
)

// Calculator defines the public interface for a Fibonacci calculator.
// It is the primary abstraction used by the application's orchestration layer to
// interact with different Fibonacci calculation algorithms.
type Calculator interface {
	// Calculate executes the calculation of the n-th Fibonacci number. It is
	// designed for safe concurrent execution and supports cancellation through the
	// provided context. Progress updates are sent asynchronously to the
	// progressChan.
	//
	// Parameters:
	//   - ctx: The context for managing cancellation and deadlines.
	//   - progressChan: The channel for sending progress updates.
	//   - calcIndex: A unique index for the calculator instance.
	//   - n: The index of the Fibonacci number to calculate.
	//   - opts: Configuration options for the calculation.
	//
	// Returns:
	//   - *big.Int: The calculated Fibonacci number.
	//   - error: An error if one occurred (e.g., context cancellation).
	Calculate(ctx context.Context, progressChan chan<- progress.ProgressUpdate, calcIndex int, n uint64, opts Options) (*big.Int, error)

	// Name returns the display name of the calculation algorithm (e.g., "Fast Doubling").
	//
	// Returns:
	//   - string: The name of the algorithm.
	Name() string
}

// CoreCalculator defines the interface for a pure calculation algorithm.
// It is the extension point for adding new Fibonacci calculation strategies.
// Implementations only need to provide the core algorithm logic; cross-cutting
// concerns (small-N fast path, GC control, FFT cache, progress adaptation)
// are handled by FibCalculator which wraps CoreCalculator via the Decorator pattern.
//
// To register a custom algorithm:
//
//	type MyAlgorithm struct{}
//	func (a *MyAlgorithm) CalculateCore(ctx context.Context, reporter progress.ProgressCallback, n uint64, opts fibonacci.Options) (*big.Int, error) { ... }
//	func (a *MyAlgorithm) Name() string { return "My Algorithm" }
//
//	factory := fibonacci.NewDefaultFactory()
//	factory.Register("myalgo", func() fibonacci.CoreCalculator { return &MyAlgorithm{} })
type CoreCalculator interface {
	CalculateCore(ctx context.Context, reporter progress.ProgressCallback, n uint64, opts Options) (*big.Int, error)
	Name() string
}

// FibCalculator is an implementation of the Calculator interface that uses the
// Decorator design pattern.
// It wraps a CoreCalculator to add cross-cutting concerns, such as the lookup
// table optimization for small `n` and the adaptation of the progress reporting
// mechanism.
type FibCalculator struct {
	core CoreCalculator
}

// NewCalculator is a factory function that constructs and returns a new
// FibCalculator.
// It takes a CoreCalculator as input, which represents the specific Fibonacci
// algorithm to be used. This function panics if the core calculator is nil,
// ensuring system integrity.
//
// Parameters:
//   - core: The core calculator to be wrapped.
//
// Returns:
//   - Calculator: A new FibCalculator instance implementing the Calculator interface.
//   - error: An error if the core implementation is nil.
func NewCalculator(core CoreCalculator) (Calculator, error) {
	if core == nil {
		return nil, errors.New("fibonacci: the `CoreCalculator` implementation cannot be nil")
	}
	return &FibCalculator{core: core}, nil
}

// MustNewCalculator is a helper that wraps a call to NewCalculator and panics
// if the returned error is non-nil. It is intended for use in variable initializations
// and `init()` functions where propagating an error is impossible.
//
// Parameters:
//   - core: The core calculator to be wrapped.
//
// Returns:
//   - Calculator: A new FibCalculator instance implementing the Calculator interface.
func MustNewCalculator(core CoreCalculator) Calculator {
	calc, err := NewCalculator(core)
	if err != nil {
		panic(err)
	}
	return calc
}

// Name returns the name of the encapsulated CoreCalculator, fulfilling the
// Calculator interface by delegating the call.
//
// Returns:
//   - string: The name of the algorithm.
func (c *FibCalculator) Name() string {
	return c.core.Name()
}

// Calculate orchestrates the calculation process.
// It first checks for small values of `n` (≤93) which can be computed
// efficiently using iterative addition without the overhead of the full
// algorithm. For larger values, it adapts the progressChan into a
// ProgressCallback callback and delegates the core calculation to the wrapped
// CoreCalculator. This method ensures that progress is reported completely upon
// successful calculation.
//
// This method provides backward compatibility with channel-based progress reporting.
// For more flexible observer-based progress reporting, use CalculateWithObservers.
//
// Parameters:
//   - ctx: The context for managing cancellation and deadlines.
//   - progressChan: The channel for sending progress updates.
//   - calcIndex: A unique index for the calculator instance.
//   - n: The index of the Fibonacci number to calculate.
//   - opts: Configuration options for the calculation.
//
// Returns:
//   - *big.Int: The calculated Fibonacci number.
//   - error: An error if one occurred.
func (c *FibCalculator) Calculate(ctx context.Context, progressChan chan<- progress.ProgressUpdate, calcIndex int, n uint64, opts Options) (result *big.Int, err error) {
	// Create a subject with a channel observer for backward compatibility
	subject := progress.NewProgressSubject()
	if progressChan != nil {
		subject.Register(progress.NewChannelObserver(progressChan))
	}
	return c.CalculateWithObservers(ctx, subject, calcIndex, n, opts)
}

// CanCalculate reports whether F(n) is expected to fit within the supplied
// memory budget. It returns the underlying [memory.MemoryEstimate] so callers
// can present the predicted breakdown regardless of the verdict.
//
// Semantics:
//   - memLimitBytes == 0 disables the check; CanCalculate always returns true.
//   - Otherwise, the verdict is (estimate.TotalBytes <= memLimitBytes).
//
// CanCalculate is a pure function: it performs no allocation beyond the
// returned estimate and does not consult runtime.MemStats. It is the
// programmatic entry point shared by [FibCalculator.CalculateWithObservers]
// (calculator-level guard) and external callers that want to gate work
// upstream of the calculator (e.g. dispatch layer, tests).
//
// This complements (does not replace) config.ValidateMemoryBudget which
// performs the same check at config-parsing time; CanCalculate intervenes
// lower in the stack so the protection holds even when the validator is
// bypassed.
func CanCalculate(n uint64, memLimitBytes uint64) (bool, memory.MemoryEstimate) {
	est := memory.EstimateMemoryUsage(n)
	if memLimitBytes == 0 {
		return true, est
	}
	return est.TotalBytes <= memLimitBytes, est
}

// CalculateWithObservers executes the calculation with observer-based progress reporting.
// This method allows for dynamic registration of multiple progress observers,
// enabling decoupled handling of progress events for UI, logging, metrics, etc.
//
// Use this method when you need to register multiple observers or when you want
// to use custom observer implementations. For simple channel-based reporting,
// use the Calculate method instead.
//
// Parameters:
//   - ctx: The context for managing cancellation and deadlines.
//   - subject: The progress subject with registered observers. If nil, progress is ignored.
//   - calcIndex: A unique index for the calculator instance.
//   - n: The index of the Fibonacci number to calculate.
//   - opts: Configuration options for the calculation.
//
// Returns:
//   - *big.Int: The calculated Fibonacci number.
//   - error: An error if one occurred.
func (c *FibCalculator) CalculateWithObservers(ctx context.Context, subject *progress.ProgressSubject, calcIndex int, n uint64, opts Options) (result *big.Int, err error) {
	// Defence-in-depth memory budget check. Runs before any heavy allocation
	// or pool warm-up so an over-budget request fails fast. Mirrors the
	// contract of config.ValidateMemoryBudget but at the calculator boundary,
	// so the guard holds even when the config-level validator is bypassed
	// (programmatic embedding, tests, future entry points).
	if opts.MemoryLimitBytes > 0 {
		if ok, est := CanCalculate(n, opts.MemoryLimitBytes); !ok {
			return nil, apperrors.MemoryError{
				Requested: est.TotalBytes,
				Available: opts.MemoryLimitBytes,
				Limit:     opts.MemoryLimitBytes,
			}
		}
	}

	// GC control for large calculations
	gcMode := opts.GCMode
	if gcMode == "" {
		gcMode = "auto"
	}
	gcCtrl := memory.NewGCController(gcMode, n)

	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		status := "success"
		if err != nil {
			status = "error"
		}
		log.Trace().
			Str("algo", c.core.Name()).
			Uint64("n", n).
			Float64("duration", duration).
			Str("status", status).
			Msg("calculation completed")
	}()

	// Create a reporter that notifies all observers.
	// Fast path: if no observers are registered, use a no-op reporter
	// to avoid lock acquisition and iteration overhead on every progress update.
	// Use Freeze() to create a lock-free snapshot for the calculation loop.
	var reporter progress.ProgressCallback
	if subject != nil && subject.ObserverCount() > 0 {
		reporter = subject.Freeze(calcIndex)
	} else {
		reporter = func(float64) {} // No-op reporter
	}

	if n <= MaxFibUint64 {
		reporter(1.0)
		return calculateSmall(n), nil
	}

	// Configure FFT cache based on options for optimal performance
	configureFFTCache(opts, n)

	// Pre-warm pools once for large calculations (one-time initialization)
	bigfft.EnsurePoolsWarmed(n)

	// Run the core calculation under panic-safe GC control. WithGC guarantees
	// debug.SetGCPercent restoration even if CalculateCore panics, preventing
	// process-wide GC leaks.
	err = gcCtrl.WithGC(func() error {
		var coreErr error
		result, coreErr = c.core.CalculateCore(ctx, reporter, n, opts)
		return coreErr
	})
	if err == nil && result != nil {
		reporter(1.0)
	}
	return result, err
}

// calculateSmall returns the n-th Fibonacci number for small n using
// iterative addition. This replaces the old LUT approach.
//
// P2-04: intentionally duplicates cmd/generate-golden.fibBig. The
// golden-data oracle must not import this package (using the library
// to generate the library's own ground truth would turn every golden
// test into a tautology). If this function changes, update
// cmd/generate-golden/main.go#fibBig in lock-step, regenerate the
// golden JSON, and verify the Fibonacci golden suite still passes —
// otherwise drift silently invalidates every downstream test.
func calculateSmall(n uint64) *big.Int {
	if n == 0 {
		return big.NewInt(0)
	}
	if n == 1 {
		return big.NewInt(1)
	}
	a := big.NewInt(0)
	b := big.NewInt(1)
	for i := uint64(2); i <= n; i++ {
		a.Add(a, b)
		a, b = b, a
	}
	return b
}
