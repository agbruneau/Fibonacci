# Algorithm Comparison

> Interactive architecture map: **[agbruneau.github.io/FibGo/dashboard/](https://agbruneau.github.io/FibGo/dashboard/)** (knowledge graph, 957 nodes / 8 layers / 13-step tour)

## Overview

This document compares the three Fibonacci calculation algorithms implemented in FibCalc.

## Available Algorithms

| Algorithm | Registry Name | Name() Output |
|-----------|--------------|---------------|
| Fast Doubling | `"fast"` | "Fast Doubling (O(log n), Parallel, Zero-Alloc)" |
| Matrix Exponentiation | `"matrix"` | "Matrix Exponentiation (O(log n), Parallel, Zero-Alloc)" |
| FFT-Based | `"fft"` | "FFT-Based Doubling (O(log n), FFT Mul)" |
| Modular Fast Doubling | `--last-digits` mode | "Modular Fast Doubling (O(log n), O(K) memory)" |

An optional GMP-based calculator (`"gmp"`) is available when built with `-tags=gmp`.

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
| Matrix Exp. (Strassen) | 7 | 18 | 18 | 43 |

> **Note**: While Strassen reduces multiplications (the most expensive operation), it significantly increases additions and subtractions. This explains why it is only beneficial for extremely large numbers where M(n) >> A(n).

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
| Fast Doubling | 5 big.Int | CalculationState |
| Matrix Exp. | 3 matrices + ~22 big.Int | matrixState |

## Benchmarks

### Test Configuration

```
CPU: AMD Ryzen 9 5900X (12 cores)
RAM: 32 GB DDR4-3600
Go: 1.25.0
OS: Linux 6.1
```

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
factory := fibonacci.GlobalFactory()
calc, _ := factory.Get("fast")
result, _ := calc.Calculate(ctx, progressChan, 0, 10_000_000, fibonacci.Options{
    ParallelThreshold: 4096,
    FFTThreshold:      500_000,
})
```

### Matrix Exponentiation (`"matrix"`)

**Recommended for**: Educational understanding, cross-verification of results, testing Strassen algorithm.

```go
factory := fibonacci.GlobalFactory()
calc, _ := factory.Get("matrix")
result, _ := calc.Calculate(ctx, progressChan, 0, 10_000_000, fibonacci.Options{
    StrassenThreshold: 3072,
})
```

### FFT-Based (`"fft"`)

**Recommended for**: FFT multiplication benchmarking, very large number tests (N > 100M), FFT vs standard math/big performance comparison.

```go
factory := fibonacci.GlobalFactory()
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
go test -bench='Benchmark(FastDoubling|Matrix|FFT)' -benchmem ./internal/fibonacci/
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
