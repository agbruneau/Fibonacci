// The fibonacci package provides implementations for calculating Fibonacci
// numbers. It exposes a `Calculator` interface that abstracts the
// underlying calculation algorithm, allowing different strategies (e.g., Fast
// Doubling, Matrix Exponentiation) to be used interchangeably. The package also
// integrates optimizations such as a lookup table (LUT) for small values and
// memory management via object pools to minimize pressure on the garbage
// collector (GC).
package fibonacci

import (
	"context"
	"math"
	"math/big"

	"github.com/agbru/fibcalc/internal/bigfft"
)

// MaxFibUint64 = 93 because F(93) is the largest Fibonacci number that fits in a uint64,
// as F(94) exceeds 2^64. This value is derived from the very rapid growth of the sequence.
const (
	MaxFibUint64 = 93 // Justified above
)

// ProgressUpdate is a data transfer object (DTO) that encapsulates the
// progress state of a calculation. It is sent over a channel from the
// calculator to the user interface to provide asynchronous progress updates.
type ProgressUpdate struct {
	// CalculatorIndex is a unique identifier for the calculator instance, allowing
	// the UI to distinguish between multiple concurrent calculations.
	CalculatorIndex int
	// Value represents the normalized progress of the calculation, ranging from 0.0 to 1.0.
	Value float64
}

// ProgressReporter defines the functional type for a progress reporting
// callback. This simplified interface is used by core calculation algorithms to
// report their progress without being coupled to the channel-based communication
// mechanism of the broader application.
//
// Parameters:
//   - progress: The normalized progress value (0.0 to 1.0).
type ProgressReporter func(progress float64)

// CalcTotalWork calculates the total work expected for O(log n) algorithms.
// The number of weighted steps is modeled as a geometric series.
// Since the algorithms iterate over bits, the work involved is roughly
// proportional to the bit index.
//
// Parameters:
//   - numBits: The number of bits in the input number n.
//
// Returns:
//   - float64: A value representing the estimated total work units.
func CalcTotalWork(numBits int) float64 {
	if numBits == 0 {
		return 0
	}
	// Geometric sum: 4^0 + 4^1 + ... + 4^(n-1) = (4^n - 1) / 3
	// We use a simplified model where work roughly quadruples each bit.
	return (math.Pow(4, float64(numBits)) - 1) / 3
}

// PrecomputePowers4 pre-calculates powers of 4 from 0 to numBits-1.
// This optimization avoids repeated calls to math.Pow(4, x) during the
// progress reporting loop, providing O(1) lookup instead of expensive
// floating-point exponentiation at each iteration.
//
// Parameters:
//   - numBits: The number of powers to compute (0 to numBits-1).
//
// Returns:
//   - []float64: A slice where powers[i] = 4^i.
func PrecomputePowers4(numBits int) []float64 {
	if numBits <= 0 {
		return nil
	}
	powers := make([]float64, numBits)
	powers[0] = 1.0
	for i := 1; i < numBits; i++ {
		powers[i] = powers[i-1] * 4.0
	}
	return powers
}

// ReportStepProgress handles harmonized progress reporting for the calculation algorithms.
// It calculates the cumulative work done based on the current bit iteration and
// reports progress via the provided callback if a significant change has occurred.
//
// Parameters:
//   - progressReporter: The callback function to report progress.
//   - lastReported: A pointer to the last reported progress value to avoid
//     redundant updates.
//   - totalWork: The total estimated work units for the calculation.
//   - workDone: The accumulated work units completed so far.
//   - i: The current bit index being processed.
//   - numBits: The total number of bits in n.
//   - powers: Pre-computed powers of 4 (from PrecomputePowers4) for O(1) lookup.
//
// Returns:
//   - float64: The updated cumulative work done.
func ReportStepProgress(progressReporter ProgressReporter, lastReported *float64, totalWork, workDone float64, i, numBits int, powers []float64) float64 {
	// Work for this step (bit i, counting down from numBits-1 to 0)
	// The step index in the geometric series is (numBits - 1 - i).
	// Fast doubling starts from MSB (small current value) and doubles up.
	// So at i=numBits-1, we have F(1). Small work.
	// At i=0, we have F(n). Huge work.
	// So the work is proportional to 4^(numBits - 1 - i).

	stepIndex := numBits - 1 - i
	workOfStep := powers[stepIndex] // O(1) lookup instead of math.Pow

	currentTotalDone := workDone + workOfStep

	// Only report if enough progress or boundaries
	// Use ProgressReportThreshold constant to avoid magic numbers
	if totalWork > 0 {
		currentProgress := currentTotalDone / totalWork
		if currentProgress-*lastReported >= ProgressReportThreshold || i == 0 || i == numBits-1 {
			progressReporter(currentProgress)
			*lastReported = currentProgress
		}
	}
	return currentTotalDone
}

// Options configures the Fibonacci calculation.
type Options struct {
	// ParallelThreshold is the bit size threshold for parallelizing multiplications.
	// If 0, a default value may be used by the implementation.
	ParallelThreshold int
	// FFTThreshold is the bit size threshold for using FFT-based multiplication.
	// If 0, a default value may be used by the implementation.
	FFTThreshold int
	// StrassenThreshold is the bit size threshold for switching to Strassen's algorithm.
	// If 0, a default value may be used by the implementation.
	StrassenThreshold int
	// FFTCacheMinBitLen is the minimum operand bit length to cache FFT transforms.
	// Smaller values don't benefit from caching. If 0, uses the default (100,000 bits).
	FFTCacheMinBitLen int
	// FFTCacheMaxEntries is the maximum number of cached FFT transforms.
	// If 0, uses the default (128 entries). Larger values improve hit rates
	// but consume more memory.
	FFTCacheMaxEntries int
	// FFTCacheEnabled controls whether FFT transform caching is active.
	// Default is true. Set to false to disable caching (useful for memory-constrained scenarios).
	FFTCacheEnabled *bool
}

// normalizeOptions returns a copy of opts with default values filled in for zero values.
// This ensures consistent threshold handling across all calculator implementations.
//
// Parameters:
//   - opts: The options to normalize.
//
// Returns:
//   - Options: A normalized copy of opts with defaults applied.
func normalizeOptions(opts Options) Options {
	normalized := opts
	if normalized.ParallelThreshold == 0 {
		normalized.ParallelThreshold = DefaultParallelThreshold
	}
	if normalized.FFTThreshold == 0 {
		normalized.FFTThreshold = DefaultFFTThreshold
	}
	if normalized.StrassenThreshold == 0 {
		normalized.StrassenThreshold = DefaultStrassenThreshold
	}
	return normalized
}

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
// It first checks for small values of `n` to leverage the lookup table
// optimization. For larger values, it adapts the progressChan into a
// ProgressReporter callback and delegates the core calculation to the wrapped
// coreCalculator. This method ensures that progress is reported completely upon
// successful calculation.
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
func (c *FibCalculator) Calculate(ctx context.Context, progressChan chan<- ProgressUpdate, calcIndex int, n uint64, opts Options) (*big.Int, error) {
	reporter := func(progress float64) {
		if progressChan == nil {
			return
		}
		if progress > 1.0 {
			progress = 1.0
		}
		update := ProgressUpdate{CalculatorIndex: calcIndex, Value: progress}
		select {
		case progressChan <- update:
		default:
		}
	}

	if n <= MaxFibUint64 {
		reporter(1.0)
		return lookupSmall(n), nil
	}

	// Configure FFT cache based on options for optimal performance
	configureFFTCache(opts)

	// Pre-warm pools for large calculations
	bigfft.PreWarmPools(n)

	result, err := c.core.CalculateCore(ctx, reporter, n, opts)
	if err == nil && result != nil {
		reporter(1.0)
	}
	return result, err
}

var fibLookupTable [MaxFibUint64 + 1]*big.Int

func init() {
	fibLookupTable[0] = big.NewInt(0)
	if MaxFibUint64 > 0 {
		fibLookupTable[1] = big.NewInt(1)
		for i := uint64(2); i <= MaxFibUint64; i++ {
			fibLookupTable[i] = new(big.Int).Add(fibLookupTable[i-1], fibLookupTable[i-2])
		}
	}
}

// lookupSmall returns a copy of the n-th Fibonacci number from the lookup
// table, ensuring the immutability of the table.
func lookupSmall(n uint64) *big.Int {
	return new(big.Int).Set(fibLookupTable[n])
}

// configureFFTCache configures the FFT transform cache based on the provided options.
// This optimization allows reusing expensive FFT transforms across iterations,
// providing 15-30% speedup for large calculations where FFT is used.
func configureFFTCache(opts Options) {
	// Get default config to use as base
	defaultConfig := bigfft.DefaultTransformCacheConfig()
	config := bigfft.TransformCacheConfig{
		MaxEntries: defaultConfig.MaxEntries,
		MinBitLen:  defaultConfig.MinBitLen,
		Enabled:    defaultConfig.Enabled,
	}

	// Override with user-provided options if specified
	if opts.FFTCacheMaxEntries > 0 {
		config.MaxEntries = opts.FFTCacheMaxEntries
	}
	if opts.FFTCacheMinBitLen > 0 {
		config.MinBitLen = opts.FFTCacheMinBitLen
	}
	if opts.FFTCacheEnabled != nil {
		config.Enabled = *opts.FFTCacheEnabled
	}

	// Apply configuration to global cache
	bigfft.SetTransformCacheConfig(config)
}
