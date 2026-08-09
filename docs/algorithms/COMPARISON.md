# Algorithm Comparison

## Overview

This document compares the three Fibonacci calculation algorithms implemented in FibCalc.

## Available Algorithms

| Algorithm | Registry Name | Name() Output |
|-----------|--------------|---------------|
| Fast Doubling | `"fast"` | "Fast Doubling (O(log n), Parallel, Zero-Alloc)" |
| Matrix Exponentiation | `"matrix"` | "Matrix Exponentiation (O(log n), Parallel, Zero-Alloc)" |
| FFT-Based | `"fft"` | "FFT-Based Doubling" |
| Modular Fast Doubling | `--last-digits` mode | n/a — free function, not a registered calculator |

An optional GMP-based calculator (`"gmp"`) is compiled in with `-tags=gmp`. The tag alone does **not** make it reachable from the CLI: its `init()` registers it only into a package-private factory, while `app.New` builds a fresh `fibonacci.NewDefaultFactory()` that pre-registers `"fast"`, `"matrix"` and `"fft"` only. A caller must add it explicitly with `fibonacci.RegisterGMPCalculator(factory)` — see [`GMP.md`](GMP.md).

> **Note on Modular Fast Doubling**: unlike the three rows above, this is **not** a registered `CoreCalculator` and has no `Name()` method. It is the free function `FastDoublingMod(ctx context.Context, n uint64, m *big.Int) (*big.Int, error)` in `internal/fibonacci/modular.go`, reached only through the `--last-digits` CLI mode (which computes F(n) mod 10^K).

## Theoretical Comparison

### Complexity

All algorithms have the same asymptotic complexity:

```
O(log n * M(n))
```

Where M(n) is the cost of multiplying numbers of n bits.

### Detailed Operation Count

The unit differs by row — a loop iteration for Fast Doubling, a single matrix
operation for the three Matrix Exp. rows; the note under the table spells out
which is which.

| Algorithm | BigInt Mults | BigInt Adds | BigInt Subs | BigInt Shifts | Total Ops |
|-----------|--------------|-------------|-------------|---------------|-----------|
| **Fast Doubling** | **3** | 1 | 1 | 1 | **6** |
| Matrix Exp. (Classic) | 8 | 4 | 0 | 0 | 12 |
| Matrix Exp. (Symmetric) | 4 | 3 | 0 | 0 | 7 |
| Matrix Exp. (Strassen-Winograd) | 7 | 7 | 8 | 0 | 22 |

> **Note**: Fast Doubling's row is a per-loop-iteration count
> (`ExecuteDoublingLoop`: `T3.Lsh`, `T3.Sub`, `T1.Add`, plus the three
> multiplications inside `ExecuteStep`), and excludes the one extra `Add` that a
> set exponent bit triggers. The three Matrix Exp. rows count **one matrix
> operation**, not one loop iteration: each iteration of `ExecuteMatrixLoop`
> performs one symmetric squaring (the row with 4 mults) and, when the exponent
> bit is set, one full multiplication (Classic or Strassen-Winograd) as well.

> **Note**: The implemented Strassen-Winograd variant (`internal/fibonacci/matrix_ops.go`) keeps 7 multiplications and uses 15 additions/subtractions total (7 adds + 8 subs) — fewer than the textbook Strassen (18 add/sub). It still trades more add/sub for fewer multiplications, so it only pays off for extremely large numbers where M(n) >> A(n).

> **Note**: The three multiplications in the current implementation are `FK×FK1`, `FK²`, and `FK1²` (using the reformulated `F(2k) = 2·FK·FK1 - FK²` identity).

### Asymptotic Constants Analysis

Let T(n) be the time to compute F(n):

```
T(n) ~ k * log2(n) * M(n)
```

The constant k represents the "multiplicative density" of the algorithm.

1. **Fast Doubling (k = 3)**:
   - Exactly 3 multiplications per loop iteration, for every iteration: `FK·FK1`, `FK²`, `FK1²`
   - F(2k) = F(k) * (2*F(k+1) - F(k)), evaluated as `2·FK·FK1 - FK²`
   - F(2k+1) = F(k+1)^2 + F(k)^2
   - It is the smallest k among the algorithms implemented here. Whether 3 is the
     information-theoretic minimum for a doubling step is not established
     anywhere in this repo, so no such claim is made.

2. **Matrix Exponentiation (k = 4 when the exponent bit is clear, 11-12 when set)**:
   - Symmetric squaring runs every iteration but the last: 4 mults (`squareSymmetricMatrix`)
   - A set exponent bit adds one matrix multiplication: 7 with Strassen-Winograd, 8 classic
   - So the per-iteration k alternates between 4 and 11/12, averaging ~7.5-8 over a
     random exponent, against Fast Doubling's flat 3

**Conclusion**: Fast Doubling's constant factor k is strictly smaller (a flat 3 vs 4 or 11-12). The one measurement artifact the repo carries, `docs/audits/bench-baseline.txt` (linux/amd64, 24 threads, `-count=5 -benchtime=1x`, 2026-07-07), agrees at the two sizes it covers — medians 3.15 ms vs 6.03 ms (Matrix) and 5.13 ms (FFT) at N=1M; 23.87 ms vs 30.84 ms and 29.08 ms at N=10M. Nothing in the repo measures any other N.

### Memory

| Algorithm | Temporary variables | Pool objects |
|-----------|---------------------|--------------|
| Fast Doubling | 5 big.Int | CalculationState (sync.Pool + per-instance GC-immune cache slot, ~32 MB cap) |
| Matrix Exp. | 3 matrices (res, p, tempMatrix) + 20 big.Int | matrixState |

> **Note (2026-06, commit `fa13bfd`)**: `FastDoublingCalculator` additionally retains the last released `CalculationState` in a per-instance, GC-immune cache slot (arena capped at 4M words ≈ 32 MB), preferred over the shared `sync.Pool` for repeated calls. Matrix Exp. and FFT-Based are unaffected (FFT-Based acquires its state through the shared `AcquireStateForN` pool path). Measured impact (2026-06-10, cumulative with the F-012 bump fix): `BenchmarkFibonacci/FastDoubling/10M` 33.30 ms -> 28.20 ms sec/op, ~-70 % B/op — see [`CHANGELOG.md`](../../CHANGELOG.md).

## Benchmarks

### The only measurement in this repo

`docs/audits/bench-baseline.txt` is the sole benchmark artifact tracked here. It
records `BenchmarkFibonacci` (`internal/fibonacci/fibonacci_test.go`) at two
sizes only — N = 1,000,000 and N = 10,000,000 — with `-count=5 -benchtime=1x`,
on `linux/amd64`, 24 threads, dated 2026-07-07. Medians of the five samples:

| N | Metric | Fast Doubling | Matrix Exp. | FFT-Based |
|---|--------|---------------|-------------|-----------|
| 1M | ns/op | **3.15 ms** | 6.03 ms | 5.13 ms |
| 1M | B/op | **1.32 MB** | 6.33 MB | 5.38 MB |
| 10M | ns/op | **23.87 ms** | 30.84 ms | 29.08 ms |
| 10M | B/op | **17.38 MB** | 92.25 MB | 30.88 MB |

These are op times with per-process pools already warm, not one-shot wall times
for a cold `fibcalc` run. Reproduce with `make bench-baseline` (which rewrites
the file) or measure your own host with `make benchmark`.

> **Removed on 2026-08-09.** This section previously carried four tables
> (N = 100 … 500,000,000), a "Test Configuration" block naming a Ryzen 9 5900X,
> and an ASCII performance graph, all flagged as "historical, indicative only"
> because no raw output was ever archived. They were not merely unbacked, they
> contradicted the file above by more than an order of magnitude — the table
> claimed 85 ms for Fast Doubling at N = 1M against the baseline's 3.15 ms — and
> the N ≥ 50M rows described runs nothing in this repo has ever performed. They
> were deleted rather than restated more cautiously. Nothing here measures any N
> other than 1M and 10M.

### Ordering, as far as it is established

At the two measured sizes, Fast Doubling is fastest and smallest, in that order:
Fast Doubling < FFT-Based < Matrix Exp. on time, Fast Doubling < FFT-Based <
Matrix Exp. on allocation. [`../PERFORMANCE.md`](../PERFORMANCE.md) warns that
the Fast Doubling / Matrix ordering can invert at N ≥ 10M on some CPUs depending
on L3 size and memory latency, and that only the memory ordering is
hardware-independent. Beyond N = 10M nothing is measured, so no ranking is
claimed.

## When to Use Each Algorithm

### Fast Doubling (`"fast"`)

**Recommended for**: general usage. Fastest and smallest of the three at both measured sizes; behavior above N = 10M is untested here. Note the CLI does **not** default to it: `DefaultAlgo = "all"` (`internal/config/config.go`), which runs every registered calculator (`GetCalculatorsToRun`, `internal/orchestration/calculator_selection.go`). Pass `-algo fast` to run this one alone.

```go
factory := fibonacci.NewDefaultFactory()
calc, _ := factory.Get("fast")
result, _ := calc.Calculate(ctx, progressChan, 0, 10_000_000, fibonacci.Options{
    ParallelThreshold: 4096,
    FFTThreshold:      500_000,
})
```

### Matrix Exponentiation (`"matrix"`)

**Recommended for**: Educational understanding, cross-verification of results, testing Strassen algorithm.

```go
factory := fibonacci.NewDefaultFactory()
calc, _ := factory.Get("matrix")
result, _ := calc.Calculate(ctx, progressChan, 0, 10_000_000, fibonacci.Options{
    StrassenThreshold: 3072,
})
```

### FFT-Based (`"fft"`)

**Recommended for**: exercising the FFT path in isolation — FFT multiplication benchmarking, regression testing, FFT vs standard `math/big` comparison. It is **not** the faster calculator at any size this repo measures (see the table above), and the type comment on `FFTBasedCalculator` (`internal/fibonacci/fft_based.go`) says so explicitly: a crossover where forcing FFT at every size pays off would lie beyond F(10M) and is unmeasured here.

```go
factory := fibonacci.NewDefaultFactory()
calc, _ := factory.Get("fft")
result, _ := calc.Calculate(ctx, progressChan, 0, 100_000_000, fibonacci.Options{
    FFTThreshold: 500_000,
})
```

### Modular Fast Doubling (`--last-digits`)

**Recommended for**: Computing the last K digits of F(N) for arbitrarily large N without storing the full result.

- **Complexity**: `bits.Len64(N)` = ⌊log2 N⌋ + 1 iterations, each doing 3 multiplications and 3 `Mod` reductions (4 when the exponent bit is set) on operands bounded by the modulus — so O(log N · M(K)) time, not O(log N). Memory is O(K): `FastDoublingMod` holds four `big.Int` (`fk`, `fk1`, `t1`, `t2`), each ≤ 2·log2(10^K) bits, and never materializes F(N).
- **Use case**: N > 1 billion where full computation exceeds available RAM
- **Bound**: `--last-digits K` is rejected above `maxLastDigits = 10_000_000` (`internal/app/calculate.go`)

## Running a Complete Comparison

```bash
# Compare all algorithm benchmarks
go test -bench='BenchmarkFibonacci/(FastDoubling|MatrixExp|FFTBased)' -benchmem -run='^$' ./internal/fibonacci/
```

## Configuration Recommendations

All thresholds are configured via the `fibonacci.Options` struct:

```go
// Small calculations (N < 100,000): keep parallelism and FFT out of the way.
// Note: 0 is NOT "off" — normalizeOptions() rewrites 0 to the package default
// (4096 / 500,000). Use -1 for the genuine sequential path, or set the FFT
// threshold above any operand you expect.
opts := fibonacci.Options{
    ParallelThreshold: -1,        // sequential (the real "off" sentinel)
    FFTThreshold:      1 << 30,   // effectively never reached at this size
}

// Medium calculations (100,000 < N < 10,000,000)
opts := fibonacci.Options{
    ParallelThreshold: 4096,
    FFTThreshold:      500_000,
}

// Large calculations (N > 10,000,000): use calibration or defaults
opts := fibonacci.Options{
    ParallelThreshold: 4096,
    FFTThreshold:      500_000,
    StrassenThreshold: 3072,
}
```

## Conclusion

**Fast Doubling** is the recommended algorithm for all general use cases: it requires only 3 multiplications per iteration — the fewest of the three implementations here — and allocates the least. `docs/audits/bench-baseline.txt` — the repo's only measurement artifact — shows it fastest and smallest at the two sizes it covers: medians of 3.15 ms / 1.32 MB per op at N=1M (vs Matrix 6.03 ms / 6.33 MB and FFT 5.13 ms / 5.38 MB) and 23.87 ms / 17.38 MB at N=10M (vs Matrix 30.84 ms / 92.25 MB and FFT 29.08 ms / 30.88 MB). It is not measured anywhere else in the repo.

**Matrix Exponentiation** is valuable for educational purposes and result verification. Its elegant mathematical foundation (Q-matrix) makes it ideal for understanding the theory, and the Strassen optimization demonstrates practical algorithm design. In `docs/audits/bench-baseline.txt` it is slower than Fast Doubling by **+91 %** at N=1M and **+29 %** at N=10M — the gap narrows with N and is not a stable 30–50 % band. [`../PERFORMANCE.md`](../PERFORMANCE.md) additionally warns that the Fast Doubling / Matrix ordering can invert at N ≥ 10M on some CPUs.

**FFT-Based** is a specialized variant that forces FFT multiplication for all operations. At the two measured sizes it is slower than Fast Doubling (5.13 ms vs 3.15 ms at N=1M; 29.08 ms vs 23.87 ms at N=10M) and allocates ~4x more at 1M. Whether forcing FFT at every size ever pays off — the usual argument being that O(n log n) multiplication eventually dominates the constant-factor overhead — is a hypothesis this repo does not test: nothing here measures beyond N=10M. Its established use is exercising the FFT subsystem in isolation.
