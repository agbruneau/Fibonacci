# Algorithm Comparison

> Interactive architecture map: **[agbruneau.github.io/FibGo/dashboard/](https://agbruneau.github.io/FibGo/dashboard/)** (knowledge graph, 1128 nodes / 4782 edges / 9 layers / 12-step tour)

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

### Detailed Operation Count (Per Iteration)

| Algorithm | BigInt Mults | BigInt Adds | BigInt Subs | Total Ops |
|-----------|--------------|-------------|-------------|-----------|
| **Fast Doubling** | **3** | 1 | 1 | **5** |
| Matrix Exp. (Classic) | 8 | 4 | 0 | 12 |
| Matrix Exp. (Symmetric) | 4 | 3 | 0 | 7 |
| Matrix Exp. (Strassen-Winograd) | 7 | 7 | 8 | 22 |

> **Note**: The implemented Strassen-Winograd variant (`internal/fibonacci/matrix_ops.go`) keeps 7 multiplications and uses 15 additions/subtractions total (7 adds + 8 subs) — fewer than the textbook Strassen (18 add/sub). It still trades more add/sub for fewer multiplications, so it only pays off for extremely large numbers where M(n) >> A(n).

> **Note**: The three multiplications in the current implementation are `FK×FK1`, `FK²`, and `FK1²` (using the reformulated `F(2k) = 2·FK·FK1 - FK²` identity).

### Asymptotic Constants Analysis

Let T(n) be the time to compute F(n):

```
T(n) ~ k * log2(n) * M(n)
```

The constant k represents the "multiplicative density" of the algorithm.

1. **Fast Doubling (k ~ 3)**:
   - Requires 3 multiplications per bit
   - F(2k) = F(k) * (2*F(k+1) - F(k))
   - F(2k+1) = F(k+1)^2 + F(k)^2
   - This is effectively the lower bound for any doubling-based method

2. **Matrix Exponentiation (k ~ 4-8)**:
   - Naive matrix multiplication requires 8 mults (k=8)
   - Symmetric optimization (B=C) reduces this to 4 mults (k=4)
   - Even with optimization, it performs slightly more auxiliary work than Fast Doubling

**Conclusion**: Fast Doubling's constant factor k is strictly smaller (3 vs 4+). The one measurement artifact the repo carries, `docs/audits/bench-baseline.txt` (linux/amd64, 24 threads, `-count=5 -benchtime=1x`, 2026-07-07), agrees at the two sizes it covers — medians 3.15 ms vs 6.03 ms (Matrix) and 5.13 ms (FFT) at N=1M; 23.87 ms vs 30.84 ms and 29.08 ms at N=10M. Nothing in the repo measures any other N.

### Memory

| Algorithm | Temporary variables | Pool objects |
|-----------|---------------------|--------------|
| Fast Doubling | 5 big.Int | CalculationState (sync.Pool + per-instance GC-immune cache slot, ~32 MB cap) |
| Matrix Exp. | 3 matrices (res, p, tempMatrix) + 20 big.Int | matrixState |

> **Note (2026-06, commit `fa13bfd`)**: `FastDoublingCalculator` additionally retains the last released `CalculationState` in a per-instance, GC-immune cache slot (arena capped at 4M words ≈ 32 MB), preferred over the shared `sync.Pool` for repeated calls. Matrix Exp. and FFT-Based are unaffected (FFT-Based acquires its state through the shared `AcquireStateForN` pool path). Measured impact (2026-06-10, cumulative with the F-012 bump fix): `BenchmarkFibonacci/FastDoubling/10M` 33.30 ms -> 28.20 ms sec/op, ~-70 % B/op — see [`CHANGELOG.md`](../../CHANGELOG.md).

## Benchmarks

### Test Configuration

```
CPU: AMD Ryzen 9 5900X (12 cores)
RAM: 32 GB DDR4-3600
Go: 1.25.0
OS: Linux 6.1
```

> **Provenance**: the timings below are historical measurements (env. Go 1.25.0, Ryzen reference machine) kept unchanged as a relative-ordering reference; their raw benchmark output was not archived (no backing file in `docs/audits/`), so they are indicative only. The project now targets Go 1.26.0+; the current dated non-regression baseline is regenerated via `make bench-baseline`; the 2026-06-10 figures are recorded in [`CHANGELOG.md`](../../CHANGELOG.md) (they report `BenchmarkFibonacci` op times with warm per-process pools, not one-shot wall times like the tables below). For up-to-date numbers on your hardware, run `make benchmark`.

### Results (average times over 10 runs)

#### Small N (N <= 10,000)

| N | Fast Doubling | Matrix Exp. | FFT-Based |
|---|---------------|-------------|-----------|
| 100 | 1.2us | 1.5us | 8.5us |
| 1,000 | 15us | 18us | 45us |
| 10,000 | 180us | 220us | 350us |

**Winner**: Fast Doubling (3-4x faster than FFT-Based)

#### Medium N (10,000 < N <= 1,000,000)

| N | Fast Doubling | Matrix Exp. | FFT-Based |
|---|---------------|-------------|-----------|
| 100,000 | 3.2ms | 4.1ms | 5.8ms |
| 500,000 | 35ms | 48ms | 42ms |
| 1,000,000 | 85ms | 110ms | 95ms |

**Winner**: Fast Doubling, but gap narrows with FFT-Based

#### Large N (N > 1,000,000)

| N | Fast Doubling | Matrix Exp. | FFT-Based |
|---|---------------|-------------|-----------|
| 5,000,000 | 850ms | 1.15s | 920ms |
| 10,000,000 | 2.1s | 2.8s | 2.3s |
| 50,000,000 | 18s | 25s | 19s |
| 100,000,000 | 45s | 62s | 48s |

**Winner**: Fast Doubling narrowly (FFT-Based very close)

#### Very Large N (N > 100,000,000)

| N | Fast Doubling | Matrix Exp. | FFT-Based |
|---|---------------|-------------|-----------|
| 250,000,000 | 3m12s | 4m25s | 3m28s |
| 500,000,000 | 8m45s | 12m10s | 9m15s |

**Winner**: Fast Doubling (still ~10% faster)

## Performance Graph

```
Time (log)
    |
  1h+                                    /
    |                                   / <- Matrix
    |                                  /
 10m+                              /  /
    |                             / /
    |                            //  <- FFT-Based
  1m+                         ///
    |                       ///
    |                     /// <- Fast Doubling
 10s+                  ///
    |               ///
    |            ///
  1s+         ///
    |      ///
    |   ///
100ms+///
    +-----+-----+-----+-----+-----+-----
        10K   100K    1M   10M  100M    N
```

## When to Use Each Algorithm

### Fast Doubling (`"fast"`)

**Recommended for**: General usage (default), maximum performance, all orders of magnitude of N.

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

**Recommended for**: FFT multiplication benchmarking, very large number tests (N > 100M), FFT vs standard math/big performance comparison.

```go
factory := fibonacci.NewDefaultFactory()
calc, _ := factory.Get("fft")
result, _ := calc.Calculate(ctx, progressChan, 0, 100_000_000, fibonacci.Options{
    FFTThreshold: 500_000,
})
```

### Modular Fast Doubling (`--last-digits`)

**Recommended for**: Computing the last K digits of F(N) for arbitrarily large N without storing the full result.

- **Complexity**: O(log N) time, O(K) memory where K is the number of digits
- **Use case**: N > 1 billion where full computation exceeds available RAM

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

**Fast Doubling** is the recommended algorithm for all general use cases: it requires only 3 multiplications per iteration (the theoretical minimum for doubling-based methods) and allocates the least. `docs/audits/bench-baseline.txt` — the repo's only measurement artifact — shows it fastest and smallest at the two sizes it covers: medians of 3.15 ms / 1.32 MB per op at N=1M (vs Matrix 6.03 ms / 6.33 MB and FFT 5.13 ms / 5.38 MB) and 23.87 ms / 17.38 MB at N=10M (vs Matrix 30.84 ms / 92.25 MB and FFT 29.08 ms / 30.88 MB). It is not measured anywhere else in the repo.

**Matrix Exponentiation** is valuable for educational purposes and result verification. Its elegant mathematical foundation (Q-matrix) makes it ideal for understanding the theory, and the Strassen optimization demonstrates practical algorithm design. In `docs/audits/bench-baseline.txt` it is slower than Fast Doubling by **+91 %** at N=1M and **+29 %** at N=10M — the gap narrows with N and is not a stable 30–50 % band. [`../PERFORMANCE.md`](../PERFORMANCE.md) additionally warns that the Fast Doubling / Matrix ordering can invert at N ≥ 10M on some CPUs.

**FFT-Based** is a specialized variant that forces FFT multiplication for all operations. It approaches Fast Doubling's performance for very large N (> 100M) where FFT's O(n log n) multiplication dominates, but carries unnecessary overhead for smaller inputs. Its primary use is benchmarking the FFT subsystem.
