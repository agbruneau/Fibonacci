# Progress Bar Algorithm for O(log n) Algorithms

> Interactive architecture map: **[agbruneau.github.io/FibGo/dashboard/](https://agbruneau.github.io/FibGo/dashboard/)** (knowledge graph, 1128 nodes / 4782 edges / 9 layers / 12-step tour)

## Description

This algorithm implements a precise progress tracking system for O(log n) time complexity algorithms, specifically designed for algorithms that iterate over the bits of a number (such as Fast Doubling, Matrix Exponentiation). It models the work performed as a geometric series where each step requires approximately 4 times more work than the previous one.

## Context

- **Target algorithms**: O(log n) algorithms that iterate over the bits of a number
- **Examples**: Fast Doubling for Fibonacci, Matrix Exponentiation
- **Key characteristic**: The work performed increases exponentially as the algorithm progresses toward the least significant bits

## Mathematical Model

### Geometric Series of Work

The algorithm models the total work as a geometric series:

```
TotalWork = 4^0 + 4^1 + 4^2 + ... + 4^(n-1) = (4^n - 1) / 3
```

Where `n` is the number of bits of the input number.

> **Note (A-10)** — This geometric series is the conceptual model only. The
> reported progress ratio is **not** computed as `done / TotalWork`: for
> `numBits >= 512` the quantity `4^numBits` overflows `float64` to `+Inf`,
> collapsing the ratio to `0`/`NaN` and freezing the bar. The implementation
> therefore evaluates progress in **closed form** from the bit index (never
> materializing `4^numBits`) and clamps `CalcTotalWork` above the overflow
> boundary — see [Step Progress Reporting](#3-step-progress-reporting) below.

### Justification

O(log n) algorithms for computing F(n):
- Start from the most significant bits (MSB) where values are small
- Progress toward the least significant bits (LSB) where values become very large
- The multiplication/calculation work approximately quadruples at each step

**Example**: For a number with 20 bits (e.g., n = 1,000,000):
- Bit 19 (MSB): work ~ 4^0 = 1 unit
- Bit 10: work ~ 4^9 = 262,144 units
- Bit 0 (LSB): work ~ 4^19 = 274,877,906,944 units

## Algorithm Components

### 1. Total Work Calculation

**Function**: `CalcTotalWork(numBits int) float64`

The raw geometric sum overflows `float64` for large inputs: `4^512 == 2^1024`
already exceeds `math.MaxFloat64`, so `math.Pow(4, numBits)` returns `+Inf` for
`numBits >= 512`. To stay finite, `CalcTotalWork` is an overflow-safe function
with three branches — below the boundary it returns the exact geometric sum;
above it (`numBits > 511`) it clamps to the largest representable sum:

```go
func CalcTotalWork(numBits int) float64 {
    if numBits <= 0 {
        return 0
    }
    // Below the float64 overflow boundary the exact geometric sum is safe.
    const safeNumBits = 511 // 4^511 < MaxFloat64 < 4^512
    if numBits <= safeNumBits {
        // Geometric sum: 4^0 + 4^1 + ... + 4^(n-1) = (4^n - 1) / 3
        return (math.Pow(4, float64(numBits)) - 1) / 3
    }
    // Past the boundary the true sum is unrepresentable. Clamp to the
    // safe-domain maximum so the result stays finite and strictly positive.
    return (math.Pow(4, float64(safeNumBits)) - 1) / 3
}
```

**Parameters**:
- `numBits`: Number of bits in the input number

**Returns**:
- A finite, positive estimate of the total work in units

**Notes**:
- Returns 0 if `numBits <= 0`.
- The clamp above `safeNumBits` keeps the historical `totalWork > 0` guard (and
  its callers/tests) working. The reported progress ratio no longer divides by
  this value (see step progress below, A-10), so clamping it does not affect the
  progress bar — it only keeps the guard finite.

### 2. Precomputation of Powers of 4

**Function**: `PrecomputePowers4(numBits int) []float64`

The implementation uses a global precomputed lookup table to avoid allocations:

```go
// Global lookup table for powers of 4 (max 64 entries for uint64 inputs)
var powersOf4 [64]float64

func init() {
    powersOf4[0] = 1.0
    for i := 1; i < 64; i++ {
        powersOf4[i] = powersOf4[i-1] * 4.0
    }
}

func PrecomputePowers4(numBits int) []float64 {
    if numBits <= 0 {
        return nil
    }
    if numBits > 64 {
        // Fall back to allocation for unusually large inputs
        powers := make([]float64, numBits)
        copy(powers, powersOf4[:])
        for i := 64; i < numBits; i++ {
            powers[i] = powers[i-1] * 4.0
        }
        return powers
    }
    return powersOf4[:numBits]  // Zero allocation — slice of global array
}
```

**Optimization**: For the common case (numBits <= 64), this returns a slice of the global array with zero allocation. Avoids repeated calls to `math.Pow(4, x)` during the calculation loop, providing O(1) lookup.

### 3. Step Progress Reporting

**Function**: `ReportStepProgress(...) float64`

**Signature**:
```go
func ReportStepProgress(
    progressReporter ProgressCallback,
    lastReported *float64,
    totalWork float64,
    workDone float64,
    i int,           // Current bit index (numBits-1 down to 0)
    numBits int,
    powers []float64,
) float64
```

> `totalWork` and `powers` are retained for signature/back-compat and to thread
> the cumulative work value across calls (the returned `float64`). Since the
> A-10 fix the reported progress ratio no longer divides by either — it is
> derived in closed form from `i`/`numBits` (step 4 below). `totalWork > 0` is
> still checked as a guard before reporting.

**Logic**:

1. **Step index calculation**:
   ```go
   stepIndex = numBits - 1 - i
   ```
   - For `i = numBits - 1` (first bit, MSB) -> `stepIndex = 0` (minimal work)
   - For `i = 0` (last bit, LSB) -> `stepIndex = numBits - 1` (maximum work)

2. **Step work calculation**:
   ```go
   workOfStep = powers[stepIndex]  // O(1) lookup
   ```

3. **Cumulative work calculation** (threaded for back-compat, no longer used as the ratio numerator):
   ```go
   currentTotalDone = workDone + workOfStep
   ```

4. **Progress calculation (closed form — A-10 fix)**:

   The progress ratio is **not** computed as `currentTotalDone / totalWork`. The
   geometric quantities overflow `float64` to `+Inf` for `numBits >= 512`
   (`4^512 == 2^1024 > MaxFloat64`), which drove the ratio to `0`/`NaN` and froze
   the bar on exactly the large calculations this project targets (A-10).
   Progress is instead derived in closed form from the bit index via
   `stepProgress(i, numBits)`, which never materializes `4^numBits`:

   ```go
   // progress(i) = (4^(numBits-i) - 1) / (4^numBits - 1)
   // Dividing numerator and denominator by 4^numBits gives the
   // numerically stable, overflow-free equivalent:
   //   progress(i) = (4^(-i) - 4^(-numBits)) / (1 - 4^(-numBits))
   currentProgress = stepProgress(i, numBits)
   ```

   Both `4^(-i)` and `4^(-numBits)` lie in `(0, 1]` and underflow gracefully
   toward `0` for large exponents, so the expression never overflows. It equals
   `1.0` at `i == 0` and is strictly increasing as `i` decreases — identical to
   the old `WorkDone / TotalWork` form bit-for-bit on the safe domain
   (`numBits < 512`), while remaining finite, monotone and bounded to `[0, 1]`
   for any `numBits`.

5. **Conditional reporting**:
   ```go
   if currentProgress - *lastReported >= ProgressReportThreshold ||
      i == 0 || i == numBits - 1 {
       progressReporter(currentProgress)
       *lastReported = currentProgress
   }
   ```

**Report Threshold**: `ProgressReportThreshold = 0.01` (1%)
- Avoids excessive updates
- Always reports at the start (i == numBits-1) and end (i == 0)

**Returns**: The updated cumulative work done

### 4. Callback Type

```go
type ProgressCallback func(progress float64)
```

- `progress`: Normalized value from 0.0 to 1.0

## Integration into the Calculation Loop

### Usage Example

```go
func ExecuteCalculation(ctx context.Context, reporter ProgressCallback, n uint64) (*big.Int, error) {
    numBits := bits.Len64(n)

    // Initialization
    totalWork := CalcTotalWork(numBits)
    powers := PrecomputePowers4(numBits)
    workDone := 0.0
    lastReportedProgress := -1.0  // -1 to force the first report

    // Main loop: iterate over bits from numBits-1 down to 0
    for i := numBits - 1; i >= 0; i-- {
        // Cancellation check
        if err := ctx.Err(); err != nil {
            return nil, err
        }

        // ... Perform the step calculation (doubling, addition, etc.) ...

        // Progress reporting
        workDone = ReportStepProgress(
            reporter,
            &lastReportedProgress,
            totalWork,
            workDone,
            i,
            numBits,
            powers,
        )
    }

    // ... Return the result ...
}
```

## Guaranteed Properties

1. **Monotonicity**: Progress is always increasing (or stable), never decreasing
2. **Valid range**: Progress values are always in [0.0, 1.0]
3. **Finalization**: Final progress is always close to 1.0 (>= 0.99)
4. **Performance**: No exponentiation calculations in the loop (precomputed)

## Progression Behavior

### Characteristics

- **Slow progress at the start**: The first steps (most significant bits) represent little work
- **Acceleration toward the end**: The last steps (least significant bits) represent the majority of work
- **Distribution**: For 20 bits, approximately 50% of the work is done in the last 2-3 steps

### Numerical Example

For `numBits = 11` (e.g., n ~ 2,000):
- TotalWork = (4^11 - 1) / 3 = 1,398,101 units
- First step (i=10): 4^0 = 1 unit -> ~0.00007% of total
- Middle step (i=5): 4^5 = 1,024 units -> ~0.073% of total
- Last step (i=0): 4^10 = 1,048,576 units -> ~75% of total

## Edge Cases and Validation

### Cases to Handle

1. **numBits = 0**:
   - `CalcTotalWork(0)` -> 0
   - `PrecomputePowers4(0)` -> nil

2. **totalWork = 0**:
   - `ReportStepProgress` must avoid division by zero
   - Do not report progress if `totalWork <= 0`

3. **First and last iteration**:
   - Always report, even if the change is below the threshold

### Recommended Tests

```bash
# Run progress-related tests
go test -v -run TestProgress ./internal/fibonacci/
go test -v -run 'TestCalcTotalWork|TestReportStepProgress' ./internal/progress/
```

```go
// Test 1: Total work increases with number of bits
func TestCalcTotalWorkMonotonic(t *testing.T) {
    prev := CalcTotalWork(1)
    for bits := 2; bits <= 20; bits++ {
        current := CalcTotalWork(bits)
        assert.True(current > prev)
        prev = current
    }
}

// Test 2: Monotonic progress
func TestProgressMonotonic(t *testing.T) {
    numBits := 20
    totalWork := CalcTotalWork(numBits)
    powers := PrecomputePowers4(numBits)

    var lastReported float64
    var prevProgress float64

    reporter := func(progress float64) {
        assert.True(progress >= prevProgress)
        prevProgress = progress
    }

    workDone := 0.0
    for i := numBits - 1; i >= 0; i-- {
        workDone = ReportStepProgress(reporter, &lastReported, totalWork, workDone, i, numBits, powers)
    }

    assert.True(prevProgress >= 0.99)
}
```

## Optimizations

### Performance

1. **Precomputed powers**: Global `[64]float64` array avoids `math.Pow(4, x)` in the loop (O(1) vs O(log x))
2. **Zero-allocation lookup**: `PrecomputePowers4` returns a slice of the global array for numBits <= 64
3. **Report threshold**: Reduces the number of callbacks (less I/O overhead)

### Complexity

- **Time**: O(1) per iteration (lookup from precomputed array)
- **Space**: O(1) — global array, no per-call allocation for typical inputs

## Adaptation for Other Algorithms

### Possible Modifications

1. **Growth factor**: If work triples per step instead of quadrupling, use 3 instead of 4
2. **Alternative formula**: For algorithms with different growth, adapt the geometric formula
3. **Weighting**: If certain steps take more/less time, adjust `workOfStep`

### Example: Factor of 3

```go
func CalcTotalWork3(numBits int) float64 {
    if numBits == 0 {
        return 0
    }
    // Geometric sum: 3^0 + 3^1 + ... + 3^(n-1) = (3^n - 1) / 2
    return (math.Pow(3, float64(numBits)) - 1) / 2
}
```

## Progress Callback Interface

### Definition

```go
// Callback type for progress reporting
type ProgressCallback func(progress float64)
```

### Usage in Calculation

```go
// Option 1: Simple callback
reporter := func(progress float64) {
    fmt.Printf("Progress: %.2f%%\n", progress*100)
}

// Option 2: Via observer pattern (used by FibCalculator)
subject := NewProgressSubject()
subject.Register(NewChannelObserver(progressChan))
reporter := subject.Freeze(calcIndex)  // Lock-free snapshot

// Option 3: Send on a channel (for asynchronous UI)
progressChan := make(chan ProgressUpdate, 10)
reporter := func(progress float64) {
    select {
    case progressChan <- ProgressUpdate{Value: progress}:
    default:
        // Channel full, skip to avoid blocking
    }
}
```

## Key Constants

```go
const (
    // Minimum progress change threshold before reporting (1%)
    ProgressReportThreshold = 0.01
)
```

## Summary of Key Equations

1. **Total work** (overflow-safe, clamped above `numBits = 511`): `TotalWork = (4^numBits - 1) / 3`
2. **Work per step**: `WorkOfStep(i) = 4^(numBits - 1 - i)`
3. **Progress** (closed form, A-10 — not `WorkDone / TotalWork`):
   `Progress(i) = (4^(-i) - 4^(-numBits)) / (1 - 4^(-numBits))`
   (algebraically equal to `(4^(numBits-i) - 1) / (4^numBits - 1)`, evaluated
   without materializing `4^numBits`)
4. **Report condition**: `currentProgress - lastReported >= 0.01 || i == 0 || i == numBits-1`

## Implementation Notes

- Use `float64` for calculation precision
- Initialize `lastReported` to `-1.0` to force the first report
- Validate that `totalWork > 0` before division
- Clamp progress values to [0.0, 1.0] if necessary
- Handle cases where `numBits == 0` or very small

## Reference Implementation

See `internal/progress/progress.go` and `internal/fibonacci/doubling_framework.go` for the complete reference implementation.
