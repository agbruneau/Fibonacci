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
> [Algorithm Components §1](#1-what-was-removed-and-why-it-does-not-affect-the-curve)
> lists what went; the model in this section still describes the curve.

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

### 1. What was removed, and why it does not affect the curve

Two functions and one table implemented the model literally, and all three are
gone. They are recorded here because the progress *curve* is unchanged and the
model above is still the right way to read it — but do not go looking for this
code in `internal/progress`:

| Removed | By | Why it was there | Why it went |
|---|---|---|---|
| `CalcTotalWork(numBits) float64` | L-01 | returned the geometric sum `(4^numBits - 1)/3`, clamped above `numBits = 511` because `4^512 == 2^1024 > math.MaxFloat64` | after A-10 nothing divided by it; its only remaining use was a `totalWork > 0` guard that could never be false for a real call |
| `PrecomputePowers4(numBits) []float64` + the `powersOf4 [64]float64` table and its `init` | L-01 | O(1) lookup of `4^i`, zero-allocation for `numBits <= 64`, avoiding `math.Pow` in the loop | fed a running work total that both callers assigned back without ever reading |
| `ReportStepProgress`'s `totalWork` / `workDone` / `powers` parameters and its return value | L-01 | carried that running total between iterations | same: written, never read |

The pivot was **A-10**, not L-01. A-10 replaced `workDone / totalWork` with the
closed form of §2, which is where the overflow was actually fixed; L-01 only
swept up the machinery A-10 had orphaned. The two are worth keeping distinct:
A-10 changed a reported value, L-01 changed no observable behaviour at all.

### 2. Step Progress Reporting

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

**Returns**: nothing (see §1).

### 3. Callback Type

```go
type ProgressCallback func(progress float64)
```

`progress` is normalized to [0.0, 1.0]. Three ways to supply one are shown under
[Progress Callback Interface](#progress-callback-interface).

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

### The call sites do not agree on loop direction

`ReportStepProgress` assumes `i` **counts down** from `numBits-1` (little work)
to `0` (most work). There are exactly three production callers
(`grep -rn "ReportStepProgress(" --include=*.go internal/ cmd/ | grep -v _test`);
two match that shape directly, and the matrix loop inverts the index at the call:

| Caller | Loop | Call |
|---|---|---|
| `DoublingFramework.ExecuteDoublingLoop` (`doubling_framework.go:162,225`) — serves `"fast"` and `"fft"` | `for i := numBits-1; i >= 0; i--` | `ReportStepProgress(reporter, &last, i, numBits)` |
| `GMPCalculator.CalculateCore` (`calculator_gmp.go:127,144`, `//go:build gmp`) | idem | idem |
| `MatrixFramework.ExecuteMatrixLoop` (`matrix_framework.go:63,92`) | `for i := 0; i < numBits; i++` — LSB to MSB | `ReportStepProgress(reporter, &last, numBits-1-i, numBits)` |

A fourth path reports nothing: `FastDoublingMod` (`modular.go`, the
`--last-digits` mode) runs the same MSB→LSB loop but takes no reporter at all.
Its operands are bounded by the modulus rather than growing, so the 4^i work
model would be wrong there anyway — every iteration costs about the same.

The matrix loop walks the exponent from its least significant bit upward, so
its *work* still grows with `i` — the operands double every squaring. Passing
`numBits-1-i` maps its ascending index onto the descending index
`stepProgress` expects, which is why both calculators produce the same
accelerating curve from opposite loop directions. Anyone adding a fourth caller
has to make the same choice explicitly: the parameter is a *work-remaining*
index, not a loop counter.

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

### What pins these properties

```bash
go test -run 'TestReportStepProgress|TestProgress_' ./internal/progress/
go test -run TestProgress ./internal/fibonacci/
```

| Test | File | Pins |
|---|---|---|
| `TestProgress_MonotonicLargeN` | `internal/progress/progress_test.go` | the A-10 guard: for `numBits ∈ {64, 512, 2000, 100000}` every reported value is finite, inside [0,1] and non-decreasing, and the last is ≥ 0.99. 512 and above is exactly where the old geometric formula returned `+Inf` |
| `TestReportStepProgressMonotonic` | idem | monotonicity at `numBits = 20` |
| `TestReportStepProgress` | idem | the threshold/first/last reporting rule |
| `TestProgressCalculationLogic`, `TestProgressCallback` | `internal/fibonacci/fibonacci_test.go` | that a real calculation drives the reporter end to end |

There is no test asserting that progress is *proportional to elapsed time* —
the 4^i model is an assumption about the work curve, never validated against a
clock in this repo. A bar that visibly stalls near the end is consistent with
everything the tests check.

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

There is only one place to change — the base inside `stepProgress`. The closed
form is base-agnostic: for a growth factor `g`, `progress(i) = (g^-i - g^-numBits) / (1 - g^-numBits)`,
and the overflow argument holds for any `g > 1` because both powers stay in
`(0, 1]`.

```go
// Same shape as stepProgress, with 4 replaced by 3 throughout.
func stepProgress3(i, numBits int) float64 {
    if numBits <= 0 || i <= 0 {
        return 1
    }
    if i >= numBits {
        i = numBits
    }
    negI := math.Pow(3, -float64(i))
    negN := math.Pow(3, -float64(numBits))
    denom := 1 - negN
    if denom <= 0 {
        return negI
    }
    return math.Max(0, math.Min(1, (negI-negN)/denom))
}
```

Do **not** reintroduce a `CalcTotalWork`-style geometric sum to divide by: that
is exactly the shape A-10 removed, and it overflows `float64` at
`numBits >= 512` for base 4 (`>= 647` for base 3) — a different base only moves
the cliff, it does not remove it.

## Progress Callback Interface

Three ways to build the `ProgressCallback` of §3. Option 2 is what
`FibCalculator` uses; the observer types live in `internal/progress/observer.go`
and `observers.go`.

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

Only equation 3 is executed. Equations 1 and 2 describe the model the curve
follows; no code computes them.

1. **Total work** (model only): `TotalWork = (4^numBits - 1) / 3`
2. **Work per step** (model only): `WorkOfStep(i) = 4^(numBits - 1 - i)`
3. **Progress** (closed form, A-10 — what `stepProgress` actually evaluates, and
   *not* `WorkDone / TotalWork`):
   `Progress(i) = (4^(-i) - 4^(-numBits)) / (1 - 4^(-numBits))`
   (algebraically equal to `(4^(numBits-i) - 1) / (4^numBits - 1)`, evaluated
   without materializing `4^numBits`)
4. **Report condition**: `currentProgress - lastReported >= 0.01 || i == 0 || i == numBits-1`

## Implementation Notes

- Use `float64` for calculation precision
- Initialize `lastReported` to `-1.0` to force the first report — though `i == numBits-1` forces the first report anyway, so this only affects a caller that enters the loop mid-way
- Guard `numBits <= 0` and return without reporting; there is no `totalWork > 0` test to make, because there is no division by a total (audit L-01)
- Clamp progress to [0.0, 1.0]; `stepProgress` already does, at both ends

## Reference Implementation

See `internal/progress/progress.go` and `internal/fibonacci/doubling_framework.go` for the complete reference implementation.
