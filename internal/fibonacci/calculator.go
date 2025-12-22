// Package fibonacci provides implementations for calculating Fibonacci numbers.
// It exposes a `Calculator` interface that abstracts the underlying calculation
// algorithm, allowing different strategies (Fast Doubling, Matrix Exponentiation,
// FFT-based) to be used interchangeably. The package integrates optimizations such
// as memory pooling, parallel processing, and dynamic threshold adjustment.
package fibonacci

//go:generate mockgen -source=calculator.go -destination=mocks/mock_calculator.go -package=mocks

import (
	"context"
	"math/big"
	"time"

	"github.com/agbru/fibcalc/internal/bigfft"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
)

// MaxFibUint64 = 93 because F(93) is the largest Fibonacci number that fits in a uint64,
// as F(94) exceeds 2^64. This value is derived from the very rapid growth of the sequence.
const (
	MaxFibUint64 = 93 // Justified above
)

var (
	calculationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fibonacci_calculations_total",
			Help: "The total number of Fibonacci calculations processed",
		},
		[]string{"algorithm", "status"},
	)
	calculationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "fibonacci_calculation_duration_seconds",
			Help: "The duration of Fibonacci calculations in seconds",
		},
		[]string{"algorithm"},
	)
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
	Calculate(ctx context.Context, progressChan chan<- ProgressUpdate, calcIndex int, n uint64, opts Options) (*big.Int, error)

	// Name returns the display name of the calculation algorithm (e.g., "Fast Doubling").
	//
	// Returns:
	//   - string: The name of the algorithm.
	Name() string
}

// coreCalculator defines the internal interface for a pure calculation
// algorithm.
type coreCalculator interface {
	CalculateCore(ctx context.Context, reporter ProgressReporter, n uint64, opts Options) (*big.Int, error)
	Name() string
}

// FibCalculator is an implementation of the Calculator interface that uses the
// Decorator design pattern.
// It wraps a coreCalculator to add cross-cutting concerns, such as the lookup
// table optimization for small `n` and the adaptation of the progress reporting
// mechanism.
type FibCalculator struct {
	core coreCalculator
}

// NewCalculator is a factory function that constructs and returns a new
// FibCalculator.
// It takes a coreCalculator as input, which represents the specific Fibonacci
// algorithm to be used. This function panics if the core calculator is nil,
// ensuring system integrity.
//
// Parameters:
//   - core: The core calculator to be wrapped.
//
// Returns:
//   - Calculator: A new FibCalculator instance implementing the Calculator interface.
func NewCalculator(core coreCalculator) Calculator {
	if core == nil {
		panic("fibonacci: the `coreCalculator` implementation cannot be nil")
	}
	return &FibCalculator{core: core}
}

// Name returns the name of the encapsulated coreCalculator, fulfilling the
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
// ProgressReporter callback and delegates the core calculation to the wrapped
// coreCalculator. This method ensures that progress is reported completely upon
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
func (c *FibCalculator) Calculate(ctx context.Context, progressChan chan<- ProgressUpdate, calcIndex int, n uint64, opts Options) (result *big.Int, err error) {
	// Create a subject with a channel observer for backward compatibility
	subject := NewProgressSubject()
	if progressChan != nil {
		subject.Register(NewChannelObserver(progressChan))
	}
	return c.CalculateWithObservers(ctx, subject, calcIndex, n, opts)
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
func (c *FibCalculator) CalculateWithObservers(ctx context.Context, subject *ProgressSubject, calcIndex int, n uint64, opts Options) (result *big.Int, err error) {
	tracer := otel.Tracer("fibonacci")
	ctx, span := tracer.Start(ctx, "Calculate")
	defer span.End()

	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		status := "success"
		if err != nil {
			status = "error"
		}
		algoName := c.core.Name()
		calculationsTotal.WithLabelValues(algoName, status).Inc()
		calculationDuration.WithLabelValues(algoName).Observe(duration)

		log.Debug().
			Str("algo", algoName).
			Uint64("n", n).
			Float64("duration", duration).
			Str("status", status).
			Msg("calculation completed")
	}()

	// Create a reporter that notifies all observers
	var reporter ProgressReporter
	if subject != nil {
		reporter = subject.AsProgressReporter(calcIndex)
	} else {
		reporter = func(float64) {} // No-op reporter
	}

	if n <= MaxFibUint64 {
		reporter(1.0)
		return calculateSmall(n), nil
	}

	// Configure FFT cache based on options for optimal performance
	configureFFTCache(opts)

	// Pre-warm pools once for large calculations (one-time initialization)
	bigfft.EnsurePoolsWarmed(n)

	result, err = c.core.CalculateCore(ctx, reporter, n, opts)
	if err == nil && result != nil {
		reporter(1.0)
	}
	return result, err
}

// calculateSmall returns the n-th Fibonacci number for small n using
// iterative addition. This replaces the old LUT approach.
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
