# Performance Guide

> **Version**: 1.0.0  
> **Last Updated**: November 2025

## Overview

This document describes the optimization techniques used in the Fibonacci Calculator and provides advice on achieving the best performance on your hardware.

## Reference Benchmarks

### Test Configuration

- **CPU**: AMD Ryzen 9 5900X (12 cores, 24 threads)
- **RAM**: 32 GB DDR4-3600
- **OS**: Linux 6.1
- **Go**: 1.23

### Results

| N | Fast Doubling | Matrix Exp. | FFT-Based | Result (digits) |
|---|---------------|-------------|-----------|-----------------|
| 1,000 | 15µs | 18µs | 45µs | 209 |
| 10,000 | 180µs | 220µs | 350µs | 2,090 |
| 100,000 | 3.2ms | 4.1ms | 5.8ms | 20,899 |
| 1,000,000 | 85ms | 110ms | 95ms | 208,988 |
| 10,000,000 | 2.1s | 2.8s | 2.3s | 2,089,877 |
| 100,000,000 | 45s | 62s | 48s | 20,898,764 |
| 250,000,000 | 3m12s | 4m25s | 3m28s | 52,246,909 |

> **Note**: Times vary depending on hardware. Use `--calibrate` for accurate measurements on your system.

## Implemented Optimizations

### 1. Zero-Allocation Strategy

#### Problem
Fibonacci calculations for large N create millions of temporary `big.Int` objects, causing excessive garbage collector pressure.

#### Solution
Using `sync.Pool` to recycle calculation states:

```go
var statePool = sync.Pool{
    New: func() interface{} {
        return &calculationState{
            f_k:  new(big.Int),
            f_k1: new(big.Int),
            t1:   new(big.Int),
            // ...
        }
    },
}

func acquireState() *calculationState {
    s := statePool.Get().(*calculationState)
    s.Reset()
    return s
}

func releaseState(s *calculationState) {
    statePool.Put(s)
}
```

#### Impact
- 95%+ reduction in allocations
- 20-30% performance improvement
- Reduced GC pause times

### 2. Adaptive Multiplication (Karatsuba vs FFT)

#### Comparative Complexity

| Algorithm | Complexity | Best for |
|-----------|------------|----------|
| Standard | O(n²) | Small numbers |
| Karatsuba | O(n^1.585) | Medium numbers |
| FFT | O(n log n) | Very large numbers |

#### Switching Threshold

The `--fft-threshold` parameter (default: 1,000,000 bits) controls when FFT multiplication is used:

```go
func smartMultiply(z, x, y *big.Int, threshold int) *big.Int {
    if threshold > 0 {
        bx := x.BitLen()
        by := y.BitLen()
        if bx > threshold && by > threshold {
            return bigfft.MulTo(z, x, y)
        }
    }
    return z.Mul(x, y)
}
```

### 3. Multi-core Parallelism

#### Strategy

The three main multiplications in the Fast Doubling algorithm are parallelized:

```go
func parallelMultiply3Optimized(s *calculationState, fftThreshold int) {
    var wg sync.WaitGroup
    wg.Add(2)
    go func() {
        defer wg.Done()
        s.t3 = smartMultiply(s.t3, s.f_k, s.t2, fftThreshold)
    }()
    go func() {
        defer wg.Done()
        s.t1 = smartMultiply(s.t1, s.f_k1, s.f_k1, fftThreshold)
    }()
    s.t4 = smartMultiply(s.t4, s.f_k, s.f_k, fftThreshold)
    wg.Wait()
}
```

#### Considerations

- **Activation threshold**: `--threshold` (default: 4096 bits)
- **Disabled with FFT**: Parallelism is disabled when FFT is used as FFT already saturates the CPU
- **Parallel FFT threshold**: Re-enabled above 10 million bits

### 4. Strassen Algorithm

For matrix exponentiation, the Strassen algorithm reduces the number of multiplications from 8 to 7:

```
Classic 2x2 multiplication: 8 multiplications
Strassen 2x2: 7 multiplications + 18 additions
```

Enabled via `--strassen-threshold` (default: 3072 bits) when matrix elements are large enough for the multiplication savings to compensate for additional additions.

### 5. Symmetric Matrix Squaring

Specific optimization for squaring symmetric matrices (where b = c):

```go
// Classic square: 8 multiplications
// Symmetric square: 4 multiplications
func squareSymmetricMatrix(dest, mat *matrix, state *matrixState, inParallel bool, fftThreshold int) {
    a2 = smartMultiply(a2, mat.a, mat.a, fftThreshold)  // a²
    b2 = smartMultiply(b2, mat.b, mat.b, fftThreshold)  // b²
    d2 = smartMultiply(d2, mat.d, mat.d, fftThreshold)  // d²
    b_ad = smartMultiply(b_ad, mat.b, ad, fftThreshold) // b(a+d)
    
    dest.a.Add(a2, b2)    // a² + b²
    dest.b.Set(b_ad)      // b(a+d)
    dest.c.Set(b_ad)      // = dest.b (symmetry)
    dest.d.Add(b2, d2)    // b² + d²
}
```

## Tuning Guide

### Automatic Calibration

```bash
# Full calibration (recommended for production)
./fibcalc --calibrate

# Quick calibration at startup
./fibcalc --auto-calibrate -n 100000000
```

Calibration tests different thresholds and determines optimal values for your hardware.

### Configuration Parameters

| Parameter | Default | Description | Adjustment |
|-----------|---------|-------------|------------|
| `--threshold` | 4096 | Parallelism threshold (bits) | ↑ on slow CPU, ↓ on many-core |
| `--fft-threshold` | 1000000 | FFT threshold (bits) | ↓ on CPU with large L3 cache |
| `--strassen-threshold` | 3072 | Strassen threshold (bits) | ↑ if addition overhead visible |

### Recommendations by Workload Type

#### Small Calculations (N < 10,000)

```bash
./fibcalc -n 5000 --threshold 0  # Disable parallelism
```

#### Medium Calculations (10,000 < N < 1,000,000)

```bash
./fibcalc -n 500000 --threshold 2048
```

#### Large Calculations (N > 1,000,000)

```bash
./fibcalc -n 10000000 --auto-calibrate
```

#### Very Large Calculations (N > 100,000,000)

```bash
./fibcalc -n 250000000 --fft-threshold 500000 --timeout 30m
```

## Performance Monitoring

### Server Mode

The server exposes metrics on `/metrics`:

```bash
curl http://localhost:8080/metrics
```

Available metrics:
- `total_requests`: Total number of requests
- `total_calculations`: Number of calculations performed
- `calculation_duration_*`: Duration distribution per algorithm
- `errors_*`: Error counters

### Go Profiling

```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=BenchmarkFastDoubling ./internal/fibonacci/

# Memory profiling
go test -memprofile=mem.prof -bench=BenchmarkFastDoubling ./internal/fibonacci/

# Analysis
go tool pprof cpu.prof
```

## Algorithm Comparison

### Fast Doubling

✅ **Advantages**:
- Fastest for the majority of cases
- Efficient parallelization
- Fewer multiplications than Matrix

⚠️ **Disadvantages**:
- More complex code

### Matrix Exponentiation

✅ **Advantages**:
- Elegant and mathematically clear implementation
- Efficient Strassen optimization for large numbers

⚠️ **Disadvantages**:
- 8 multiplications per iteration vs 3 for Fast Doubling
- Slower in practice

### FFT-Based

✅ **Advantages**:
- Forces FFT use for all multiplications
- Useful for FFT benchmarking

⚠️ **Disadvantages**:
- Significant overhead for small numbers
- Primarily used for testing

## Advanced Optimization Tips

### 1. CPU Affinity (Linux)

```bash
# Force use of specific cores
taskset -c 0-7 ./fibcalc -n 100000000
```

### 2. Disable Frequency Scaling

```bash
# Performance mode
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor
```

### 3. GOMAXPROCS

```bash
# Limit number of Go threads
GOMAXPROCS=8 ./fibcalc -n 100000000
```

### 4. Optimized Compilation

```bash
# Build with aggressive optimizations
go build -ldflags="-s -w" -gcflags="-B" ./cmd/fibcalc
```

## Known Limitations

1. **Memory**: F(1 billion) requires ~25 GB of RAM for the result alone
2. **Time**: Calculations for N > 500M can take hours
3. **FFT Contention**: The FFT algorithm saturates cores, limiting external parallelism
