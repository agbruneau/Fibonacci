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

An optional GMP-based calculator (`"gmp"`) is available when built with `-tags=gmp`.

> **Note on Modular Fast Doubling**: unlike the three rows above, this is **not** a registered `CoreCalculator` and has no `Name()` method. It is the free function `FastDoublingMod(n uint64, m *big.Int) (*big.Int, error)` in `internal/fibonacci/modular.go`, reached only through the `--last-digits` CLI mode (which computes F(n) mod 10^K).

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
| Matrix Exp. (Symmetric) | 4 | 4 | 0 | 8 |
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

**Conclusion**: Fast Doubling is consistently faster because its constant factor k is strictly smaller (3 vs 4+).

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
// Small calculations (N < 100,000): disable parallelism and FFT
opts := fibonacci.Options{
    ParallelThreshold: 0,  // disable parallelism (overhead > gains)
    FFTThreshold:      0,  // disable FFT (too small)
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

**Fast Doubling** is the recommended algorithm for all general use cases. It has the best performance across all input sizes due to requiring only 3 multiplications per iteration (the theoretical minimum for doubling-based methods). It also has the lowest memory footprint.

**Matrix Exponentiation** is valuable for educational purposes and result verification. Its elegant mathematical foundation (Q-matrix) makes it ideal for understanding the theory, and the Strassen optimization demonstrates practical algorithm design. However, it is consistently 30-50% slower than Fast Doubling.

**FFT-Based** is a specialized variant that forces FFT multiplication for all operations. It approaches Fast Doubling's performance for very large N (> 100M) where FFT's O(n log n) multiplication dominates, but carries unnecessary overhead for smaller inputs. Its primary use is benchmarking the FFT subsystem.
