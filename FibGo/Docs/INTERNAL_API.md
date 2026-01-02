# Internal API Documentation

> **Version**: 1.0.0
> **Last Updated**: December 2025

## Overview

This document describes the internal interfaces and types of the Fibonacci Calculator application. It provides a comprehensive reference for developers wishing to extend or integrate the system.

The architecture follows **Clean Architecture** principles with clear separation of concerns and low coupling between modules. The main interfaces are designed to allow dependency injection and facilitate testing.

## Table of Contents

1. [Main Interfaces](#main-interfaces)
2. [Design Patterns](#design-patterns)
3. [Types and Structures](#types-and-structures)
4. [Usage Examples](#usage-examples)
5. [Component Relationships](#component-relationships)
6. [GoDoc Documentation Generation](#godoc-documentation-generation)

---

## Main Interfaces

### `Calculator`

The main interface for calculating Fibonacci numbers. It abstracts the different algorithms (Fast Doubling, Matrix Exponentiation, FFT-based) and allows their interchangeable use.

**Location**: `internal/fibonacci/calculator.go`

```go
type Calculator interface {
    // Calculate executes the calculation of the nth Fibonacci number
    // Designed for concurrent execution and supports cancellation via context
    Calculate(ctx context.Context, progressChan chan<- ProgressUpdate, 
              calcIndex int, n uint64, opts Options) (*big.Int, error)
    
    // Name returns the display name of the algorithm (e.g., "Fast Doubling")
    Name() string
}
```

**Implementations**:
- `FibCalculator`: Generic wrapper that encapsulates a `coreCalculator`
- Concrete algorithms implement `coreCalculator` (internal interface)

**Advanced Methods**:
- `CalculateWithObservers()`: Version with Observer pattern support for progress tracking

**Usage Example**:
```go
factory := fibonacci.NewDefaultFactory()
calc, _ := factory.Get("fast")
result, err := calc.Calculate(ctx, progressChan, 0, 1000000, opts)
```

---

### `coreCalculator`

Internal interface for pure calculation algorithms. Concrete implementations include:

- `OptimizedFastDoubling`: Optimized Fast Doubling algorithm (O(log n), parallel, zero-allocation)
- `MatrixExponentiation`: Matrix exponentiation with Strassen (O(log n))
- `FFTBasedCalculator`: FFT-based calculation (O(log n), FFT multiplication)

```go
type coreCalculator interface {
    CalculateCore(ctx context.Context, reporter ProgressReporter, 
                  n uint64, opts Options) (*big.Int, error)
    Name() string
}
```

---

### `ProgressReporter`

Functional type for progress reporting. Allows calculation algorithms to report their progress without being coupled to the communication mechanism.

**Location**: `internal/fibonacci/progress.go`

```go
type ProgressReporter func(progress float64)
```

**Parameters**:
- `progress`: Normalized progress value (0.0 to 1.0)

**Usage**:
```go
reporter := func(progress float64) {
    fmt.Printf("Progress: %.2f%%\n", progress*100)
}
```

---

### `ProgressObserver`

Interface for observing progress events. Implements the Observer pattern to allow decoupled processing of progress updates.

**Location**: `internal/fibonacci/observer.go`

```go
type ProgressObserver interface {
    // Update is called when progress changes
    Update(calcIndex int, progress float64)
}
```

**Provided Implementations**:
- `ChannelObserver`: Adapts the Observer pattern to channels (backward compatibility)
- `LoggingObserver`: Logs updates with zerolog
- `MetricsObserver`: Exports metrics to Prometheus
- `NoOpObserver`: Null Object pattern for tests

**Example**:
```go
subject := fibonacci.NewProgressSubject()
subject.Register(fibonacci.NewChannelObserver(progressChan))
subject.Register(fibonacci.NewLoggingObserver(logger, 0.1))
```

---

### `ProgressSubject`

Manages the registration and notification of progress observers. Implements the Subject part of the Observer pattern.

**Location**: `internal/fibonacci/observer.go`

```go
type ProgressSubject struct {
    observers []ProgressObserver
    mu        sync.RWMutex  // Thread-safe
}
```

**Main Methods**:
- `Register(observer ProgressObserver)`: Adds an observer
- `Unregister(observer ProgressObserver)`: Removes an observer
- `Notify(calcIndex int, progress float64)`: Notifies all observers
- `AsProgressReporter(calcIndex int) ProgressReporter`: Converts to ProgressReporter for compatibility

---

### `CalculatorFactory`

Interface for creating and managing `Calculator` instances. Allows dependency injection and facilitates testing.

**Location**: `internal/fibonacci/registry.go`

```go
type CalculatorFactory interface {
    // Create creates a new Calculator instance by name
    Create(name string) (Calculator, error)
    
    // Get returns an existing Calculator instance (with cache)
    Get(name string) (Calculator, error)
    
    // List returns a sorted list of registered calculator names
    List() []string
    
    // Register adds a new calculator type to the factory
    Register(name string, creator func() coreCalculator) error
    
    // GetAll returns a map of all registered calculators
    GetAll() map[string]Calculator
}
```

**Implementation**: `DefaultFactory`

**Pre-registered Calculators**:
- `"fast"`: OptimizedFastDoubling
- `"matrix"`: MatrixExponentiation
- `"fft"`: FFTBasedCalculator

**Example**:
```go
factory := fibonacci.NewDefaultFactory()
calc, err := factory.Get("fast")
if err != nil {
    log.Fatal(err)
}
```

---

### `MultiplicationStrategy`

Interface for multiplication and squaring operations used in Fibonacci calculations. Allows choosing between Karatsuba, FFT, or other algorithms.

**Location**: `internal/fibonacci/strategy.go`

```go
type MultiplicationStrategy interface {
    // Multiply calculates x * y and stores the result in z (which can be reused)
    Multiply(z, x, y *big.Int, opts Options) (*big.Int, error)
    
    // Square calculates x * x (optimized compared to general multiplication)
    Square(z, x *big.Int, opts Options) (*big.Int, error)
    
    // Name returns a descriptive name for the strategy
    Name() string
    
    // ExecuteStep performs a complete doubling step
    ExecuteStep(s *CalculationState, opts Options, inParallel bool) error
}
```

**Implementations**:
- `AdaptiveStrategy`: Adaptively chooses between Karatsuba and FFT depending on operand size
- `FFTOnlyStrategy`: Forces FFT multiplication for all operations
- `KaratsubaStrategy`: Forces Karatsuba multiplication (via math/big)

---

### `Service`

Interface for Fibonacci calculation services. High-level abstraction used by the HTTP server layer.

**Location**: `internal/service/calculator_service.go`

```go
type Service interface {
    // Calculate performs the Fibonacci calculation for the given algorithm and index
    Calculate(ctx context.Context, algoName string, n uint64) (*big.Int, error)
}
```

**Implementation**: `CalculatorService`

**Features**:
- Input validation (maxN limit)
- Algorithm retrieval via factory
- Centralized application of configuration options

---

## Design Patterns

### Observer Pattern

The Observer pattern is used for progress reporting, allowing decoupling between calculators and progress consumers.

**Components**:
- `ProgressSubject`: Observable subject
- `ProgressObserver`: Observer interface
- `ChannelObserver`, `LoggingObserver`, `MetricsObserver`: Concrete implementations

**Flow**:
```
Calculator → ProgressReporter → ProgressSubject → Observers
```

**Benefits**:
- Decoupling: Calculators do not know their observers
- Extensibility: Easy to add new types of observers
- Testability: Easy to mock observers

---

### Factory Pattern

The Factory pattern is used to create and manage calculator instances.

**Components**:
- `CalculatorFactory`: Factory interface
- `DefaultFactory`: Implementation with cache and thread-safety

**Benefits**:
- Dependency injection
- Instance reuse (cache)
- Dynamic registration of new calculators

---

### Strategy Pattern

The Strategy pattern is used for multiplication operations, allowing dynamic algorithm selection (Karatsuba, FFT, etc.).

**Components**:
- `MultiplicationStrategy`: Strategy interface
- `AdaptiveStrategy`, `FFTOnlyStrategy`, `KaratsubaStrategy`: Implementations

**Benefits**:
- Flexibility: Runtime algorithm switching
- Testability: Easy to test different algorithms
- Performance: Optimal choice based on operand size

---

### Decorator Pattern

The Decorator pattern is used to add cross-cutting concerns to calculators.

**Components**:
- `FibCalculator`: Decorator wrapping a `coreCalculator`
- Added features:
  - Optimization for small n (lookup table)
  - Progress mechanism adaptation
  - Prometheus metrics
  - OpenTelemetry tracing

---

## Types and Structures

### `Options`

Configuration structure for Fibonacci calculations.

**Location**: `internal/fibonacci/options.go`

```go
type Options struct {
    ParallelThreshold      int  // Threshold (bits) to parallelize multiplications
    FFTThreshold          int  // Threshold (bits) to use FFT multiplication
    KaratsubaThreshold    int  // Threshold (bits) for optimized Karatsuba
    StrassenThreshold     int  // Threshold (bits) for Strassen algorithm
    FFTCacheMinBitLen     int  // Minimum length (bits) to cache FFT transforms
    FFTCacheMaxEntries    int  // Maximum number of entries in FFT cache
    FFTCacheEnabled       *bool // Enables/disables FFT cache
    EnableDynamicThresholds bool // Enables dynamic threshold adjustment
    DynamicAdjustmentInterval int // Interval between threshold checks
}
```

**Default Values**:
- `ParallelThreshold`: 4096 bits
- `FFTThreshold`: 500000 bits
- `StrassenThreshold`: 3072 bits

---

### `ProgressUpdate`

DTO (Data Transfer Object) encapsulating calculation progress state.

**Location**: `internal/fibonacci/progress.go`

```go
type ProgressUpdate struct {
    CalculatorIndex int     // Unique calculator identifier
    Value           float64 // Normalized progress (0.0 to 1.0)
}
```

---

### `CalculationState`

Aggregates temporary variables for the "Fast Doubling" algorithm, enabling efficient management via an object pool.

**Location**: `internal/fibonacci/fastdoubling.go`

```go
type CalculationState struct {
    FK, FK1, T1, T2, T3, T4 *big.Int
}
```

**Methods**:
- `Reset()`: Prepares state for a new calculation

**Object Pool**:
- `AcquireState()`: Obtains a state from the pool
- `ReleaseState(s *CalculationState)`: Releases a state to the pool

---

### `CalculationResult`

Encapsulates the result of a Fibonacci calculation, facilitating comparison and reporting.

**Location**: `internal/orchestration/orchestrator.go`

```go
type CalculationResult struct {
    Name     string        // Algorithm identifier
    Result   *big.Int      // Calculated Fibonacci number (nil if error)
    Duration time.Duration // Execution time
    Err      error         // Potential error
}
```

---

## Usage Examples

### Example 1: Simple Calculation with an Algorithm

```go
package main

import (
    "context"
    "fmt"
    "github.com/agbru/fibcalc/internal/fibonacci"
)

func main() {
    // Create a factory
    factory := fibonacci.NewDefaultFactory()
    
    // Get a calculator
    calc, err := factory.Get("fast")
    if err != nil {
        panic(err)
    }
    
    // Configure options
    opts := fibonacci.Options{
        ParallelThreshold: 4096,
        FFTThreshold:     500000,
    }
    
    // Calculate F(1000000)
    ctx := context.Background()
    result, err := calc.Calculate(ctx, nil, 0, 1000000, opts)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("F(1000000) = %s\n", result.String())
}
```

---

### Example 2: Calculation with Progress Tracking

```go
package main

import (
    "context"
    "fmt"
    "github.com/agbru/fibcalc/internal/fibonacci"
)

func main() {
    factory := fibonacci.NewDefaultFactory()
    calc, _ := factory.Get("fast")
    
    // Create a progress subject
    subject := fibonacci.NewProgressSubject()
    
    // Register a channel observer
    progressChan := make(chan fibonacci.ProgressUpdate, 10)
    subject.Register(fibonacci.NewChannelObserver(progressChan))
    
    // Read progress updates
    go func() {
        for update := range progressChan {
            fmt.Printf("Calculator %d: %.2f%%\n",
                       update.CalculatorIndex, update.Value*100)
        }
    }()
    
    // Calculate with observers
    ctx := context.Background()
    opts := fibonacci.Options{}
    result, err := calc.CalculateWithObservers(ctx, subject, 0, 1000000, opts)
    
    close(progressChan)
    
    if err != nil {
        panic(err)
    }
    fmt.Printf("Result: %s\n", result.String())
}
```

---

### Example 3: Comparison of Multiple Algorithms

```go
package main

import (
    "context"
    "fmt"
    "github.com/agbru/fibcalc/internal/fibonacci"
    "github.com/agbru/fibcalc/internal/orchestration"
    "github.com/agbru/fibcalc/internal/config"
    "os"
)

func main() {
    factory := fibonacci.NewDefaultFactory()
    cfg := config.AppConfig{
        N: 1000000,
    }
    
    // Get all calculators
    calculators := []fibonacci.Calculator{}
    for _, name := range factory.List() {
        calc, _ := factory.Get(name)
        calculators = append(calculators, calc)
    }
    
    // Execute calculations in parallel
    ctx := context.Background()
    results := orchestration.ExecuteCalculations(ctx, calculators, cfg, os.Stdout)
    
    // Analyze results
    exitCode := orchestration.AnalyzeComparisonResults(results, cfg, os.Stdout)
    os.Exit(exitCode)
}
```

---

### Example 4: Registering a Custom Calculator

```go
package main

import (
    "context"
    "fmt"
    "math/big"
    "github.com/agbru/fibcalc/internal/fibonacci"
)

// Custom Implementation
type CustomCalculator struct{}

func (c *CustomCalculator) Name() string {
    return "Custom Algorithm"
}

func (c *CustomCalculator) CalculateCore(ctx context.Context, 
                                        reporter fibonacci.ProgressReporter,
                                        n uint64, 
                                        opts fibonacci.Options) (*big.Int, error) {
    // Custom implementation
    reporter(0.5) // 50% progress
    result := big.NewInt(int64(n)) // Simplified example
    reporter(1.0) // 100% progress
    return result, nil
}

func main() {
    factory := fibonacci.NewDefaultFactory()
    
    // Register custom calculator
    factory.Register("custom", func() fibonacci.coreCalculator {
        return &CustomCalculator{}
    })
    
    // Use custom calculator
    calc, _ := factory.Get("custom")
    result, _ := calc.Calculate(context.Background(), nil, 0, 100, fibonacci.Options{})
    fmt.Println(result)
}
```

---

### Example 5: Using the Service

```go
package main

import (
    "context"
    "fmt"
    "github.com/agbru/fibcalc/internal/fibonacci"
    "github.com/agbru/fibcalc/internal/service"
    "github.com/agbru/fibcalc/internal/config"
)

func main() {
    factory := fibonacci.NewDefaultFactory()
    cfg := config.AppConfig{}
    
    // Create service
    svc := service.NewCalculatorService(factory, cfg, 0) // 0 = no limit
    
    // Use service
    ctx := context.Background()
    result, err := svc.Calculate(ctx, "fast", 1000000)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Result: %s\n", result.String())
}
```

---

## Component Relationships

### Dependency Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    Entry Points                             │
│  (CLI, Server, REPL)                                        │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│              Orchestration Layer                            │
│  • ExecuteCalculations()                                    │
│  • AnalyzeComparisonResults()                               │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│                  Service Layer                              │
│  • CalculatorService                                        │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│              Fibonacci Package                              │
│                                                             │
│  ┌──────────────┐    ┌──────────────┐    ┌─────────────┐ │
│  │  Calculator  │───▶│   Factory    │───▶│  Algorithms │ │
│  │  Interface   │    │              │    │             │ │
│  └──────────────┘    └──────────────┘    └─────────────┘ │
│         │                    │                   │         │
│         │                    │                   │         │
│         ▼                    ▼                   ▼         │
│  ┌──────────────┐    ┌──────────────┐    ┌─────────────┐ │
│  │   Progress   │    │   Strategy    │    │   Options   │ │
│  │   Observer   │    │   Pattern    │    │             │ │
│  └──────────────┘    └──────────────┘    └─────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Data Flow

1. **Initialization**:
   ```
   Factory → Register Calculators → Cache Instances
   ```

2. **Calculation**:
   ```
   Calculator.Calculate() 
   → ProgressReporter 
   → ProgressSubject 
   → Observers (Channel, Logging, Metrics)
   ```

3. **Orchestration**:
   ```
   ExecuteCalculations() 
   → Multiple Calculators (concurrent)
   → Collect Results
   → AnalyzeComparisonResults()
   ```

---

## GoDoc Documentation Generation

### Access via pkg.go.dev

GoDoc documentation is automatically generated and hosted on [pkg.go.dev](https://pkg.go.dev) for public packages. For internal packages, you can generate documentation locally.

### Local Generation

```bash
# Generate HTML documentation for all packages
godoc -http=:6060

# Access documentation via browser
# http://localhost:6060/pkg/github.com/agbru/fibcalc/internal/fibonacci/
```

### Documentation Structure

GoDoc comments follow standard conventions:

- **Package comment**: Description of the package (first line of the file)
- **Type comments**: Description of types and interfaces
- **Method comments**: Description of methods with parameters and returns
- **Example functions**: Usage examples (`Example*` functions)

### GoDoc Comment Example

```go
// Calculator defines the public interface for a Fibonacci calculator.
// It is the main abstraction used by the orchestration layer
// to interact with different calculation algorithms.
type Calculator interface {
    // Calculate executes the calculation of the nth Fibonacci number.
    // Designed for safe concurrent execution and supports cancellation
    // via the provided context.
    //
    // Parameters:
    //   - ctx: The context to handle cancellation and timeouts.
    //   - progressChan: The channel to send progress updates.
    //   - calcIndex: A unique index for the calculator instance.
    //   - n: The index of the Fibonacci number to calculate.
    //   - opts: Configuration options for the calculation.
    //
    // Returns:
    //   - *big.Int: The calculated Fibonacci number.
    //   - error: An error if one occurred.
    Calculate(ctx context.Context, progressChan chan<- ProgressUpdate,
              calcIndex int, n uint64, opts Options) (*big.Int, error)
}
```

---

## Best Practices

### 1. Use of Interfaces

- Prefer interfaces over concrete types for function parameters
- Use dependency injection via factories
- Avoid creating circular dependencies

### 2. Progress Management

- Use `ProgressSubject` to register multiple observers
- Use `CalculateWithObservers()` for fine-grained control
- Use `Calculate()` for channel compatibility

### 3. Error Handling

- Always check returned errors
- Use `context.Context` for cancellation
- Respect configured timeouts

### 4. Performance

- Reuse calculator instances via `Factory.Get()`
- Configure thresholds according to your use case
- Use the object pool for repeated calculations

### 5. Tests

- Use generated mocks (`mockgen`) for tests
- Test interfaces, not implementations
- Use `NoOpObserver` for tests without progress

---

## Additional Resources

- [General Architecture](./ARCHITECTURE.md)
- [REST API Documentation](./api/API.md)
- [Performance Guide](./PERFORMANCE.md)
- [Algorithm Documentation](./algorithms/)

---

## Contribution

To contribute to this documentation:

1. Update this file with new interfaces/types
2. Add usage examples for new features
3. Maintain consistency with existing GoDoc comments
4. Test that examples work correctly

---

**Note**: This documentation is manually maintained. For automatically generated documentation from GoDoc comments, refer to `godoc` output or pkg.go.dev.
