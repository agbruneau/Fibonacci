# Performance Guide

> Interactive architecture map: **[agbruneau.github.io/FibGo/dashboard/](https://agbruneau.github.io/FibGo/dashboard/)** (knowledge graph, 906 nodes / 8 layers / 11-step tour)

## Overview

This document describes the optimization techniques used in the Fibonacci Calculator and provides advice on achieving the best performance on your hardware.

## Reference Benchmarks

### Test Configuration

- **CPU**: AMD Ryzen 9 5900X (12 cores, 24 threads)
- **RAM**: 32 GB DDR4-3600
- **OS**: Linux 6.1
- **Go**: 1.25.0

### Results

| N | Fast Doubling | Matrix Exp. | FFT-Based | Result (digits) |
|---|---------------|-------------|-----------|-----------------|
| 1,000 | 15us | 18us | 45us | 209 |
| 10,000 | 180us | 220us | 350us | 2,090 |
| 100,000 | 3.2ms | 4.1ms | 5.8ms | 20,899 |
| 1,000,000 | 85ms | 110ms | 95ms | 208,988 |
| 10,000,000 | 2.1s | 2.8s | 2.3s | 2,089,877 |
| 100,000,000 | 45s | 62s | 48s | 20,898,764 |
| 250,000,000 | 3m12s | 4m25s | 3m28s | 52,246,909 |

### Comparison snapshot — Intel Core Ultra 9 275HX (24 cores)

Informative only; kept for cross-architecture comparison. Ryzen remains the canonical reference.

| N | Fast Doubling | Matrix Exp. | FFT-Based | Result (digits) |
|---|---------------|-------------|-----------|-----------------|
| 10,000       | 120us  | 180us  | 280us  | 2,090      |
| 1,000,000    | ~3ms   | 55ms   | 45ms   | 208,988    |
| 10,000,000   | ~60ms  | 750ms  | 600ms  | 2,089,877  |
| 100,000,000  | 30s    | 42s    | 33s    | 20,898,764 |
| 250,000,000  | 2m10s  | 3m05s  | 2m25s  | 52,246,909 |

Intel figures reflect a mobile workstation profile with higher single-thread performance; absolute ordering of algorithms (Fast Doubling < FFT < Matrix for medium N) is consistent across both platforms.

### Running Benchmarks

```bash
# Run all benchmarks
go test -bench=. -benchmem ./internal/fibonacci/

# Benchmark specific algorithm
go test -bench=BenchmarkFastDoubling -benchmem ./internal/fibonacci/

# Benchmark with specific iteration count
go test -bench=BenchmarkFastDoubling -benchtime=5x ./internal/fibonacci/
```

### Regression gate (>= 5 %, CI-enforced)

The `bench` job in `.github/workflows/ci.yml` is **blocking** — a PR
introducing any sub-benchmark regression > 5 % in
`BenchmarkFibonacci/(FastDoubling|MatrixExp|FFTBased)` fails the build.

The gate compares each PR against `docs/audits/bench-baseline.txt` using
`benchstat` ; the percentage threshold is enforced by
`.github/scripts/bench_gate.py`. Refresh the baseline on a quiet machine :

```bash
make bench-baseline                            # writes docs/audits/bench-baseline.txt
git add docs/audits/bench-baseline.txt && git commit -m 'perf(bench): refresh baseline'
```

For a one-shot DTM (Dynamic Threshold Manager) comparison, see
`docs/audits/bench-dtm-{on,off}.txt` and ADR-0001.

### Versioned benchmark snapshots (regression tracking)

To compare performance across Git revisions on the **same machine**, use a fixed command and record the environment:

1. **Record a snapshot** (writes `build/bench/snapshot-*.txt` with `go version`, full Git SHA, and benchmark output):

   ```bash
   make bench-versioned
   ```

   This runs `go test` with fixed flags: `-bench=BenchmarkFastDoubling -benchmem -count=3 -benchtime=2s ./internal/fibonacci/`.

2. **Annotate the result**: note the Git tag or commit (`git rev-parse HEAD`) in your changelog or ticket when you archive a snapshot. Single-run numbers are noisy; compare trends only on an idle machine, same flags, same `GOMAXPROCS` if you tune it.

3. **CHANGELOG** (optional): add a one-line entry such as
   `Perf: benchmark snapshot @ <SHA> — FastDoubling ns/op ±X% vs previous main` when you publish a measured change.

The reference table in [Reference Benchmarks](#reference-benchmarks) above is tied to a specific hardware and Go 1.25.0 configuration; your snapshots document *your* runner.

## Hardware heuristic defaults

When CLI thresholds are left at **0** (auto) and no valid calibration profile is loaded, FibCalc applies `internal/config.ApplyAdaptiveThresholds`, which calls `EstimateOptimalParallelThreshold`, `EstimateOptimalFFTThreshold`, and `EstimateOptimalStrassenThreshold`. These functions use:

- **`runtime.NumCPU()`** for parallelism tiers (unchanged broad behavior).
- **`HardwareHeuristic`** (`internal/config/hardware.go`): on **amd64** and **386**, **AVX2** and **AVX-512 (AVX512F)** are read from `golang.org/x/sys/cpu` to nudge defaults (e.g. slightly lower FFT crossover on SIMD-rich CPUs, adjusted Strassen threshold when at least four cores are available).

Diagnostics and unit tests use the exported `EstimateParallelThresholdForHeuristic`, `EstimateFFTThresholdForHeuristic`, and `EstimateStrassenThresholdForHeuristic` with a synthetic `HardwareHeuristic`. Cached calibration profiles store `cpu_heuristic_key` (profile format v3) so a change in SIMD class invalidates stale JSON — see [CALIBRATION.md](CALIBRATION.md).

## Implemented Optimizations

### 1. Zero-Allocation Strategy

#### Problem
Fibonacci calculations for large N create millions of temporary `big.Int` objects, causing excessive garbage collector pressure.

#### Solution
Using `sync.Pool` to recycle calculation states:

```go
var statePool = sync.Pool{
    New: func() interface{} {
        return &CalculationState{
            FK:  new(big.Int),
            FK1: new(big.Int),
            T1:  new(big.Int),
            T2:  new(big.Int),
            T3:  new(big.Int),
        }
    },
}
```

#### Impact
- 95%+ reduction in allocations
- 20-30% performance improvement
- Reduced GC pause times

**Calculation Arena (state-bound)**: For N > 1,000 a contiguous `CalculationArena` pre-allocates all 5 `big.Int` backing arrays from a single `[]big.Word` block, reducing GC tracking overhead and memory fragmentation. The arena is owned by `CalculationState` and travels through the same `sync.Pool`: `AcquireStateForN(n)` reuses the existing arena (`Reset()` only) when the previous tenancy was large enough, otherwise it reallocates. `ReleaseStateWithResult(s, src)` deep-copies the result out of the arena before resetting it and detaches every state slot before pool return, so a subsequent acquisition cannot alias another caller's result. The arena falls back to heap allocation when exhausted, and is dropped (not pooled) past `maxArenaPoolWords` (~50M words ≈ 400 MB) to bound resident memory.

### 2. 2-Tier Adaptive Multiplication

The `smartMultiply` function selects the optimal multiplication algorithm based on operand bit size:

```go
func smartMultiply(z, x, y *big.Int, fftThreshold int) (*big.Int, error) {
    bx := x.BitLen()
    by := y.BitLen()

    // Tier 1: FFT Multiplication — O(n log n)
    if fftThreshold > 0 && bx > fftThreshold && by > fftThreshold {
        return bigfft.MulTo(z, x, y)
    }

    // Tier 2: Standard math/big (uses Karatsuba internally for large operands)
    return z.Mul(x, y), nil
}
```

| Tier | Algorithm | Complexity | Activation Threshold (default) |
|------|-----------|------------|-------------------------------|
| 1 | FFT (Schonhage-Strassen) | O(n log n) | > 500,000 bits |
| 2 | Standard `math/big` | O(n^2) / O(n^1.585) | Below FFT threshold |

### 3. Multi-core Parallelism

The three main multiplications in the Fast Doubling algorithm can be parallelized via the `DoublingStepExecutor.ExecuteStep` method. The strategy dispatches multiplication work across goroutines when the operand size exceeds the parallel threshold.

#### Considerations

- **Activation threshold**: `ParallelThreshold` (default: 4096 bits)
- **Disabled with FFT**: Parallelism is disabled when FFT is used as FFT already saturates the CPU
- **Parallel FFT threshold**: Re-enabled above 5,000,000 bits (`ParallelFFTThreshold`)

### 4. Strassen Algorithm

For matrix exponentiation, the Strassen algorithm reduces the number of multiplications from 8 to 7:

```
Classic 2x2 multiplication: 8 multiplications
Strassen 2x2: 7 multiplications + 18 additions
```

Enabled via `StrassenThreshold` (default: 3,072 bits via config; internal default: 256 bits) when matrix elements are large enough for the multiplication savings to compensate for additional additions. The internal default can be adjusted at runtime via `fibonacci.SetDefaultStrassenThreshold()`, and the per-calculation threshold is set via `Options.StrassenThreshold`.

### 5. Symmetric Matrix Squaring

Specific optimization for squaring symmetric matrices (where b = c), reducing multiplications from 8 to 4.

### 6. GC Controller

For large calculations (N ≥ 1M), the `GCController` disables Go's garbage collector during computation, eliminating GC pauses and reducing the ~2× memory overhead from GC scanning. A soft memory limit (3× current Sys) acts as an OOM safety net.

| Mode | Activation | Behavior |
|------|-----------|----------|
| `auto` (default) | N ≥ 1,000,000 | Disable GC during calculation |
| `aggressive` | Always | Disable GC regardless of N |
| `disabled` | Never | Standard GC behavior |

GC control is handled automatically; no user-facing flag or environment variable is exposed. The mode is selected internally based on the calculation size.

### 7. Memory Budget Estimation

Pre-calculate estimated memory usage before starting with `--memory-limit`:

| N | Estimated Peak Memory |
|---|---|
| 10M | ~120 MB |
| 100M | ~1.2 GB |
| 1B | ~12 GB |
| 5B | ~58 GB |

If the estimate exceeds the limit, the tool exits with an error and suggests `--last-digits K` as an alternative.

### 8. Partial Computation (Last Digits)

The `--last-digits K` mode computes F(N) mod 10^K using modular arithmetic in O(log N) time and O(K) memory, enabling computation for arbitrarily large N:

```bash
fibcalc -n 10000000000 --last-digits 100
```

## Tuning Guide

### Automatic Calibration

The calibration system (`internal/calibration`) tests different thresholds and determines optimal values for your hardware:

```bash
# Full threshold sweep (saves ~/.fibcalc_calibration.json)
fibcalc --calibrate

# Quick startup calibration with cached-profile fallback
fibcalc --auto-calibrate
```

> The programmatic entry point is `calibration.RunCalibration(ctx, out, calculatorRegistry, progressDisplay, colorProvider) int`; see [CALIBRATION.md](CALIBRATION.md) for the full API and the 3-tier fallback.

### Configuration Parameters

#### Algorithm Thresholds

| Parameter | Default | Description | Adjustment |
|-----------|---------|-------------|------------|
| `ParallelThreshold` | 4,096 bits | Parallelism activation threshold | Increase on slow CPU, decrease on many-core |
| `FFTThreshold` | 500,000 bits | FFT multiplication threshold | Decrease on CPU with large L3 cache |
| `StrassenThreshold` | 3,072 bits | Strassen algorithm threshold | Increase if addition overhead is visible |

#### FFT Cache Settings

| Parameter | Default | Description |
|-----------|---------|-------------|
| `FFTCacheMinBitLen` | 100,000 bits | Minimum operand bit length to cache FFT transforms |
| `FFTCacheMaxEntries` | 128 entries | Maximum number of cached FFT transforms |
| `FFTCacheEnabled` | `true` | Enable/disable FFT transform caching |

#### Dynamic Threshold Adjustment

| Parameter | Default | Description |
|-----------|---------|-------------|
| `EnableDynamicThresholds` | `false` | Enable real-time threshold adjustment based on per-iteration timing |
| `DynamicAdjustmentInterval` | 5 iterations | Iterations between threshold checks (when enabled) |

#### FFT Parallelism (bigfft package)

| Variable | Default | Description |
|----------|---------|-------------|
| `ParallelFFTRecursionThreshold` | 4 | Minimum FFT size (log2) for parallel recursion |
| `MaxParallelFFTDepth` | 3 | Maximum depth of parallel FFT recursion |

These FFT parallelism settings are runtime-configurable via `bigfft.SetFFTParallelismConfig()`.

All threshold parameters are configured via the `fibonacci.Options` struct:

```go
opts := fibonacci.Options{
    ParallelThreshold:         4096,
    FFTThreshold:              500_000,
    StrassenThreshold:         3072,
    FFTCacheEnabled:           boolPtr(true),
    FFTCacheMaxEntries:        128,
    FFTCacheMinBitLen:         100_000,
    EnableDynamicThresholds:   false,
    DynamicAdjustmentInterval: 5,
}
```

## Performance Monitoring

### Go Profiling

```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=BenchmarkFastDoubling ./internal/fibonacci/
go tool pprof cpu.prof

# Memory profiling
go test -memprofile=mem.prof -bench=BenchmarkFastDoubling ./internal/fibonacci/
go tool pprof mem.prof

# Trace
go test -trace=trace.out -bench=BenchmarkFastDoubling ./internal/fibonacci/
go tool trace trace.out
```

## Algorithm Comparison

### Fast Doubling

**Advantages**:
- Fastest for the majority of cases
- Efficient parallelization
- Fewer multiplications than Matrix (3 per iteration)

**Disadvantages**:
- More complex code

### Matrix Exponentiation

**Advantages**:
- Elegant and mathematically clear implementation
- Efficient Strassen optimization for large numbers

**Disadvantages**:
- 4-8 multiplications per iteration vs 3 for Fast Doubling
- Slower in practice

### FFT-Based

**Advantages**:
- Forces FFT use for all multiplications
- Useful for FFT benchmarking

**Disadvantages**:
- Significant overhead for small numbers
- Primarily used for testing and benchmarking

## Advanced Optimization Tips

### 1. CPU Affinity (Linux)

```bash
# Force use of specific cores
taskset -c 0-7 <your-binary> [args]
```

### 2. Disable Frequency Scaling

```bash
# Performance mode
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor
```

### 3. GOMAXPROCS

```bash
# Limit number of Go threads
GOMAXPROCS=8 go test -bench=. ./internal/fibonacci/
```

### 4. Optimized Compilation

```bash
# Build with aggressive optimizations
go build -ldflags="-s -w" -gcflags="-B" ./cmd/fibcalc
```

## Known Limitations

1. **Memory**: F(1 billion) requires ~12 GB of RAM. Use `--memory-limit` to validate before starting.
2. **Time**: Calculations for N > 500M can take hours
3. **FFT Contention**: The FFT algorithm saturates cores, limiting external parallelism
4. **Workaround**: Use `--last-digits K` for O(K) memory usage with arbitrarily large N.
