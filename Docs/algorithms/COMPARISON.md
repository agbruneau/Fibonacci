# Algorithm Comparison

> **Last Updated**: November 2025

## Overview

This document compares the three Fibonacci calculation algorithms implemented in FibCalc.

## Available Algorithms

| Algorithm | Flag | Description |
|-----------|------|-------------|
| Fast Doubling | `-algo fast` | Main algorithm, most performant |
| Matrix Exponentiation | `-algo matrix` | Matrix approach with Strassen |
| FFT-Based | `-algo fft` | Forces FFT multiplication |

## Theoretical Comparison

### Complexity

All algorithms have the same asymptotic complexity:

```
O(log n × M(n))
```

Where M(n) is the cost of multiplying numbers of n bits.

### Detailed Operation Count (Per Iteration)

| Algorithm | BigInt Mults | BigInt Adds | BigInt Subs | Total Ops |
|-----------|--------------|-------------|-------------|-----------|
| **Fast Doubling** | **3** | 1 | 1 | **5** |
| Matrix Exp. (Classic) | 8 | 4 | 0 | 12 |
| Matrix Exp. (Symmetric) | 4 | 4 | 0 | 8 |
| Matrix Exp. (Strassen) | 7 | 18 | 18 | 43 |

> **Note**: While Strassen reduces multiplications (the most expensive operation), it significantly increases additions and subtractions. This explains why it is only beneficial for extremely large numbers where $M(n) \gg A(n)$.

### Asymptotic Constants Analysis

Let $T(n)$ be the time to compute $F_n$.
$$ T(n) \approx k \cdot \log_2(n) \cdot M(n) $$

The constant $k$ represents the "multiplicative density" of the algorithm.

1.  **Fast Doubling ($k \approx 3$)**:
    - Requires 3 multiplications per bit.
    - $F_{2k} = F_k(2F_{k+1} - F_k)$
    - $F_{2k+1} = F_{k+1}^2 + F_k^2$
    - This is effectively the lower bound for any doubling-based method.

2.  **Matrix Exponentiation ($k \approx 4-8$)**:
    - Naive matrix multiplication requires 8 mults ($k=8$).
    - Symmetric optimization ($B=C$) reduces this to 4 mults ($k=4$).
    - Even with optimization, it performs slightly more auxiliary work (additions/memory moves) than Fast Doubling.

**Conclusion**: Fast Doubling is consistently faster because its constant factor $k$ is strictly smaller (3 vs 4+).

### Memory

| Algorithm | Temporary variables | Pool objects |
|-----------|---------------------|--------------|
| Fast Doubling | 6 big.Int | calculationState |
| Matrix Exp. | 3 matrices + ~22 big.Int | matrixState |

## Benchmarks

### Test Configuration

```
CPU: AMD Ryzen 9 5900X (12 cores)
RAM: 32 GB DDR4-3600
Go: 1.25
OS: Linux 6.1
```

### Results (average times over 10 runs)

#### Small N (N ≤ 10,000)

| N | Fast Doubling | Matrix Exp. | FFT-Based |
|---|---------------|-------------|-----------|
| 100 | 1.2µs | 1.5µs | 8.5µs |
| 1,000 | 15µs | 18µs | 45µs |
| 10,000 | 180µs | 220µs | 350µs |

**Winner**: Fast Doubling (3-4× faster than FFT-Based)

#### Medium N (10,000 < N ≤ 1,000,000)

| N | Fast Doubling | Matrix Exp. | FFT-Based |
|---|---------------|-------------|-----------|
| 100,000 | 3.2ms | 4.1ms | 5.8ms |
| 500,000 | 35ms | 48ms | 42ms |
| 1,000,000 | 85ms | 110ms | 95ms |

**Winner**: Fast Doubling, but gap narrowed with FFT-Based

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
    │
  1h├                                    /
    │                                   / ← Matrix
    │                                  /
 10m├                              /  /
    │                             / /
    │                            /╱  ← FFT-Based
  1m├                         /╱╱
    │                       ╱╱╱
    │                     ╱╱╱ ← Fast Doubling
 10s├                  ╱╱╱
    │               ╱╱╱
    │            ╱╱╱
  1s├         ╱╱╱
    │      ╱╱╱
    │   ╱╱╱
100ms├╱╱╱
    └─────┬─────┬─────┬─────┬─────┬─────
        10K   100K    1M   10M  100M    N
```

## When to Use Each Algorithm

### Fast Doubling (`-algo fast`)

✅ **Recommended for**:
- General usage (default)
- Maximum performance
- All orders of magnitude of N

```bash
./fibcalc -n 10000000 -algo fast
```

### Matrix Exponentiation (`-algo matrix`)

✅ **Recommended for**:
- Educational understanding
- Cross-verification of results
- When you want to test the Strassen algorithm

```bash
./fibcalc -n 10000000 -algo matrix --strassen-threshold 2048
```

### FFT-Based (`-algo fft`)

✅ **Recommended for**:
- FFT multiplication benchmarking
- Very large number tests (N > 100M)
- FFT vs Karatsuba performance comparison

```bash
./fibcalc -n 100000000 -algo fft
```

## Complete Comparison

```bash
# Compare all algorithms on the same N
./fibcalc -n 10000000 -algo all -d
```

Typical output:

```
=== Execution Configuration ===
Calculating F(10000000) with a timeout of 5m0s.
Environment: 24 logical processors, Go go1.25.
Optimization thresholds: Parallelism=4096 bits, FFT=1000000 bits.
Execution mode: Parallel comparison of all algorithms.

=== Comparison Summary ===
Algorithm                                    Duration    Status
Fast Doubling (O(log n), Parallel, Zero-Alloc)   2.1s       ✅ Success
FFT-Based Doubling (O(log n), FFT Mul)           2.3s       ✅ Success
Matrix Exponentiation (O(log n), Parallel, Zero-Alloc)   2.8s       ✅ Success

=== All algorithms succeeded ===
Result binary size: 6,942,420 bits.
```

## Configuration Recommendations

### For Small Calculations (N < 100,000)

```bash
./fibcalc -n 50000 -algo fast --threshold 0 --fft-threshold 0
```

- Disable parallelism (overhead > gains)
- Disable FFT (too small)

### For Medium Calculations (100,000 < N < 10,000,000)

```bash
./fibcalc -n 5000000 -algo fast --threshold 4096
```

- Parallelism enabled
- FFT for the largest operations

### For Large Calculations (N > 10,000,000)

```bash
./fibcalc -n 100000000 -algo fast --auto-calibrate --timeout 30m
```

- Auto-calibration for optimal thresholds
- Extended timeout

## Conclusion

| Criterion | Fast Doubling | Matrix Exp. | FFT-Based |
|-----------|---------------|-------------|-----------|
| **Performance** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ |
| **Memory** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ |
| **Simplicity** | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ |
| **Educational** | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐ |

**General recommendation**: Use **Fast Doubling** (`-algo fast`) for all use cases, except for specific testing or comparison needs.
