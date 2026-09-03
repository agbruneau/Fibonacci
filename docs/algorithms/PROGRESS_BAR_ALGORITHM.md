# Progress Bar Algorithm for O(log n) Algorithms

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

> **Note (A-10, L-01)** — This geometric series is the conceptual model only.
> The reported progress ratio is **not** computed as `done / TotalWork`: for
> `numBits >= 512` the quantity `4^numBits` overflows `float64` to `+Inf`,
> collapsing the ratio to `0`/`NaN` and freezing the bar. A-10 replaced the
> division with a **closed form** over the bit index that never materializes
> `4^numBits`.
>
> The 2026-09 audit (L-01) then removed the machinery A-10 had left behind:
> `CalcTotalWork`, `PrecomputePowers4` and the `powersOf4` table are **gone**,
> along with `ReportStepProgress`'s `totalWork` / `workDone` / `powers`
> parameters and its return value. None of them influenced the reported ratio
> any more — `totalWork` only guarded a test that was always true, and the
> running work total was assigned back by both callers without ever being read.
> Sections 1 and 2 below are kept as a record of the model, not of the code.

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

### 1. Total Work Calculation (REMOVED from the code — model only)

**Former function**: `CalcTotalWork(numBits int) float64`, deleted by audit
L-01. The listing below documents the model the progress curve still follows;
no such function exists in `internal/progress` any more.

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

### 2. Precomputation of Powers of 4 (REMOVED from the code)

**Former function**: `PrecomputePowers4(numBits int) []float64`, deleted by
audit L-01 together with the `powersOf4` table and its `init`. It only ever fed
the cumulative total that nothing read.

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

**Function**: `ReportStepProgress(...)`

**Signature**:
```go
func ReportStepProgress(
    progressReporter ProgressCallback,
    lastReported *float64,
    i int,           // Current bit index (numBits-1 down to 0)
    numBits int,
)
```

The reported fraction is computed in closed form by `stepProgress(i, numBits)`:

```
progress(i) = (4^(-i) - 4^(-numBits)) / (1 - 4^(-numBits))
```

Both `4^(-i)` and `4^(-numBits)` lie in `(0, 1]` and underflow gracefully
toward 0, so the expression is finite for any `numBits`. It equals exactly 1.0
at `i == 0` and is strictly increasing as `i` decreases — mathematically
identical to `done / TotalWork` on the domain where that was representable.

**Report Threshold**: `ProgressReportThreshold = 0.01` (1%)
- Avoids excessive updates
- Always reports at the start (i == numBits-1) and end (i == 0)

**Returns**: nothing. It used to return the running work total; audit L-01
removed it once it was established that both callers assigned it back without
reading it.

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
    lastReportedProgress := -1.0  // -1 to force the first report

    // Main loop: iterate over bits from numBits-1 down to 0
    for i := numBits - 1; i >= 0; i-- {
        // Cancellation check
        if err := ctx.Err(); err != nil {
            return nil, err
        }

        // ... Perform the step calculation (doubling, addition, etc.) ...

        // Progress reporting
        ReportStepProgress(reporter, &lastReportedProgress, i, numBits)
    }

    // ... Return the result ...
}
```

## Guaranteed Properties

1. **Monotonicity**: Progress is always increasing (or stable), never decreasing — `stepProgress` is strictly increasing as `i` decreases
2. **Valid range**: Progress values are always in [0.0, 1.0] — `stepProgress` clamps both ends explicitly (`if p < 0 return 0`, `if p > 1 return 1`)
3. **Finalization**: the last loop iteration (`i == 0`) reports **exactly 1.0** — `stepProgress` returns a literal `1` for `i <= 0`, and `i == 0` forces a report regardless of the threshold. `FibCalculator.CalculateWithObservers` additionally calls `reporter(1.0)` on success (`internal/fibonacci/calculator.go`)
4. **Performance**: bounded, allocation-free work per iteration — but **not** exponentiation-free. `ReportStepProgress` calls `stepProgress` (`internal/progress/progress.go:ReportStepProgress`), which evaluates two `math.Pow` per iteration: `math.Pow(4, -i)` and `math.Pow(4, -numBits)` (`progress.go:stepProgress`). Since audit L-01 there is no precomputed `powers` array and no cumulative total: the two `math.Pow` calls are the whole per-iteration cost.

## Progression Behavior

### Characteristics

- **Slow progress at the start**: The first steps (most significant bits) represent little work
- **Acceleration toward the end**: The last steps (least significant bits) represent the majority of work
- **Distribution**: the geometric 4^i model puts the last step's share at `3·4^(numBits−1) / (4^numBits − 1)`. That is 100 % for `numBits = 1`, 80 % for 2, 76.19 % for 3, and converges to 75 % from above — it is **not** independent of `numBits`, only asymptotically constant. For any realistic `numBits` (≥ 10) the ~75 % / ~94 % figures for the last one and last two steps hold to two decimals.

### Numerical Example

For `numBits = 11` (e.g., n ~ 2,000):
- TotalWork = (4^11 - 1) / 3 = 1,398,101 units
- First step (i=10): 4^0 = 1 unit -> ~0.00007% of total
- Middle step (i=5): 4^5 = 1,024 units -> ~0.073% of total
- Last step (i=0): 4^10 = 1,048,576 units -> ~75% of total

## Edge Cases and Validation

### Cases to Handle

1. **numBits <= 0**:
   - `ReportStepProgress` returns without reporting anything: a zero-length
     loop has no progress to describe. This replaced the old `totalWork > 0`
     guard, which could never be false for a real call (audit L-01).

2. **First and last iteration**:
   - Always report, even if the change is below the threshold

### Recommended Tests

```bash
# Run progress-related tests
go test -v -run TestProgress ./internal/fibonacci/
go test -v -run 'TestReportStepProgress|TestProgress_' ./internal/progress/
```

```go
// Test 1: Monotonic progress, finite and inside [0,1]
func TestReportStepProgressMonotonic(t *testing.T) {
    numBits := 20

    var lastReported float64
    var prevProgress float64

    reporter := func(progress float64) {
        assert.True(progress >= prevProgress)
        prevProgress = progress
    }

    for i := numBits - 1; i >= 0; i-- {
        ReportStepProgress(reporter, &lastReported, i, numBits)
    }

    assert.True(prevProgress >= 0.99)
}

// Test 2: the closed form survives past the old overflow boundary (A-10)
func TestProgress_MonotonicLargeN(t *testing.T) {
    for _, numBits := range []int{64, 512, 2000, 100000} {
        var lastReported float64
        var last float64 = -1
        for i := numBits - 1; i >= 0; i-- {
            ReportStepProgress(func(p float64) {
                assert.False(math.IsNaN(p) || math.IsInf(p, 0))
                assert.True(p >= last)
                last = p
            }, &lastReported, i, numBits)
        }
        assert.True(last >= 0.99)
    }
}
```

## Optimizations

### Performance

1. **Report threshold**: Reduces the number of callbacks (less I/O overhead), while always reporting the first and last step
2. **No precomputation**: the `[64]float64` power table and `PrecomputePowers4` were removed by audit L-01. They fed a cumulative work total that stopped influencing the reported ratio at A-10 and that neither caller read afterwards, so the lookup they saved was a lookup nothing needed.
3. **Not** exponentiation-free: the reported ratio comes from `stepProgress`, which pays two `math.Pow` calls per iteration (`internal/progress/progress.go:stepProgress`). That is the deliberate price of the closed form, which stays finite for `numBits >= 512` where the raw geometric sum overflows to `+Inf` (A-10)

### Complexity

- **Time**: O(1) per iteration — two `math.Pow` calls, no loop over bits
- **Space**: O(1) — no per-call allocation

## Adaptation for Other Algorithms

### Possible Modifications

1. **Growth factor**: If work triples per step instead of quadrupling, use 3 instead of 4
2. **Alternative formula**: For algorithms with different growth, adapt the geometric formula
3. **Weighting**: If certain steps take more/less time, adjust the exponent inside `stepProgress` (there is no longer a per-step `workOfStep` variable to weight — audit L-01)

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
