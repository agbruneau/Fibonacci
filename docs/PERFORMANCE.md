# Performance Guide

## Overview

This document describes the optimization techniques used in the Fibonacci Calculator and provides advice on achieving the best performance on your hardware.

## Reference Benchmarks

### The only throughput baseline this repo carries

`docs/audits/bench-baseline.txt` is the **only whole-calculator throughput
baseline** tracked in the repository — the file `benchstat` compares against.
Two other files time whole calculators without being baselines: the
[scale curve](#scale-curve-four-sizes-one-host) (`bench-scale-2026-09.txt`) and
the GMP reference (`bench-gmp-2026-09.txt`, read in
[algorithms/GMP.md § Performance](algorithms/GMP.md#performance)). The rest of
[`docs/audits/`](audits/) holds targeted A/B runs (the now-removed dynamic
thresholds and their removal, FFT cache, pool memclr, micro-benchmark stability)
or a resident-memory reading, each cited where it is used.
Medians of the 5 samples per row, computed from
that file — linux/amd64, 24 threads, `-count=5 -benchtime=1x`, header stamp
`baseline-2026-07-07`:

| N | Fast Doubling | Matrix Exp. | FFT-Based |
|---|---|---|---|
| 1,000,000 | **3.15 ms** / 1.32 MB per op | 6.03 ms / 6.33 MB | 5.13 ms / 5.38 MB |
| 10,000,000 | **23.87 ms** / 17.38 MB per op | 30.84 ms / 92.25 MB | 29.08 ms / 30.88 MB |

`-benchtime=1x` means one iteration per sample, so each row includes first-call
warm-up. That warm-up shows up as a high first sample in four of the six groups —
FastDoubling/1M, FastDoubling/10M, MatrixExp/10M and FFTBased/10M, where sample 1
is the slowest of the five — but **not** in the other two: in MatrixExp/1M the
first sample (5,781,043 ns) is the second *lowest* of its five, and in FFTBased/1M
the first sample (5,134,173 ns) is exactly the median. Counted from
[`docs/audits/bench-baseline.txt`](audits/bench-baseline.txt). These
are the numbers `benchstat` compares against.

### What the removed historical tables left behind

Earlier revisions carried two throughput tables (a Ryzen 9 5900X « historical »
run and an Intel Core Ultra 9 275HX « snapshot ») for which no output was ever
archived. Re-checked on 2026-09-04 against the CPU the second one names, their
magnitudes were off by 2× to 88×, so both were removed (EVAL-05). The one thing
that check confirmed is the **ordering**: Fast Doubling ≤ FFT-Based < Matrix
Exponentiation at N = 1M, 10M and 100M, on that host. The baseline table above
agrees at 1M and 10M. The scale curve below, on the same CPU, agrees at 1M and
100M; at 10M it separates only Fast Doubling from Matrix, and FFT-Based from
neither.

Fast Doubling is also the most **memory**-efficient: on the same baseline it
allocates **~4.8×** fewer bytes per op than Matrix at F(1M) (1.32 MB vs 6.33 MB)
and **~5.3×** fewer at F(10M) (17.38 MB vs 92.25 MB). Those are `-benchmem` B/op
medians — total bytes allocated, not peak RSS.

### Scale curve: four sizes, one host

[`docs/audits/bench-scale-2026-09.txt`](audits/bench-scale-2026-09.txt) (EVAL-09)
times `BenchmarkFibonacci` at N = 100K, 1M, 10M and 100M for the three
calculators, 10 samples each — 120 result lines. It is a curve, **not** a
regression baseline: host, OS and Go version all differ from
`bench-baseline.txt`, and the 5 % rule below still compares against the baseline
only. `benchstat` itself will not line the two files up (it keys results on
`goos` and `cpu`, which differ), so a gap between them — FastDoubling/10M is
32.28 ms here, 23.87 ms there — mixes host, OS, Go version and code, and is not a
regression signal.

From the artifact header (lines 2–6): `go1.27.0 windows/amd64`, Intel Core Ultra 9
275HX, `GOMAXPROCS=24` (default), Windows 11, commit `751c1cc`, 2026-09-23, run
with

```bash
go test -bench='BenchmarkFibonacci/(FastDoubling|MatrixExp|FFTBased)' \
    -benchmem -run='^$' -count=10 -benchtime=1x ./internal/fibonacci/
```

Every figure below is printed by

```bash
go run golang.org/x/perf/cmd/benchstat@v0.0.0-20260825160852-19be9d8e6c70 docs/audits/bench-scale-2026-09.txt
```

(prefix `MSYS_NO_PATHCONV=1` under Git Bash). Median, 95 % confidence interval,
and median B/op in benchstat's binary units:

| N | Fast Doubling | Matrix Exp. | FFT-Based |
|---|---|---|---|
| 100,000 | **143.2 µs ± 46 %** / 54.12 KiB | 342.8 µs ± 24 % / 382.7 KiB | 876.7 µs ± 25 % / 523.7 KiB |
| 1,000,000 | **3.972 ms ± 27 %** / 1.270 MiB | 8.341 ms ± 8 % / 6.150 MiB | 5.853 ms ± 17 % / 5.157 MiB |
| 10,000,000 | **32.28 ms ± 13 %** / 16.63 MiB | 35.23 ms ± 5 % / 84.95 MiB | 36.53 ms ± 13 % / 29.52 MiB |
| 100,000,000 | **193.2 ms ± 11 %** / 561.5 MiB | 351.4 ms ± 19 % / 1.059 GiB | 272.0 ms ± 3 % / 347.1 MiB |

**What is timed.** One `Calculate` call returning a `*big.Int` — no decimal
conversion, no I/O — as wall time on 24 hardware threads. Fast Doubling runs its
three products concurrently above `ParallelThreshold` (4,096 bits) except while
the FFT is in play, where it waits for operands above `ParallelFFTThreshold`
(5,000,000 bits); `internal/bigfft` recurses in parallel up to depth 3
([FFT Parallelism](#fft-parallelism-bigfft-package)). No CPU time is recorded, so
no figure here measures work; each measures elapsed time on this machine.

**What `-benchtime=1x` does to it.** Each sample is one call. With `1x` the
`testing` package keeps the result of its initial `N = 1` run and runs nothing
else (`src/testing/benchmark.go`, `launch`: "If -benchtime=1x was requested, use
that result"), so no warm-up iteration is discarded. It shows: the first sample
is the slowest of its ten in 6 of the 12 groups (FastDoubling/100K, FFTBased/100K,
FastDoubling/1M, MatrixExp/1M, MatrixExp/10M, MatrixExp/100M), and the largest on
B/op in 5 — FastDoubling/100K's first sample allocates 836,168 B (line 11) against
a 54.12 KiB median, MatrixExp/100M's 2,098,162,616 B (line 111) against 1.059 GiB.
A median of ten is insensitive to one such sample, and so is benchstat's
interval, which here runs from the 2nd to the 9th of the ten sorted values
(checked against both FastDoubling/100K intervals). The ± 46 % on FastDoubling/100K
time therefore comes from the spread of the other samples (118.0 to 209.4 µs), and
the ± 151 % on its B/op from two more samples near 137 KB (lines 16–17): the 100K
row is noisy beyond warm-up. Read it as an order of magnitude.

**Ordering.** Pairwise, each group renamed so benchstat compares two files
(Mann–Whitney U, 10 against 10):

```bash
f=docs/audits/bench-scale-2026-09.txt
grep 'FastDoubling/10M-' $f | sed 's#FastDoubling/##' > a.txt
grep 'MatrixExp/10M-'    $f | sed 's#MatrixExp/##'    > b.txt
benchstat a.txt b.txt      # +9.13 % (p=0.035 n=10)
```

| N | Matrix vs Fast | FFT vs Fast | FFT vs Matrix |
|---|---|---|---|
| 100K | +139.43 % (p = 0.002) | +512.43 % (p = 0.000) | +155.78 % (p = 0.000) |
| 1M | +110.01 % (p = 0.000) | +47.36 % (p = 0.000) | −29.83 % (p = 0.000) |
| 10M | +9.13 % (p = 0.035) | ~ (p = 0.063) | ~ (p = 0.579) |
| 100M | +81.91 % (p = 0.000) | +40.82 % (p = 0.000) | −22.59 % (p = 0.000) |

Twelve tests are read together, so each p-value is held to the Bonferroni level
0.05 / 12 ≈ 0.004. Fast Doubling has the lowest median at all four sizes, and
that ordering is significant at that level at 100K, 1M and 100M; at 10M the three
calculators are not separated (Matrix vs Fast p = 0.035 falls short of 0.004,
the two FFT comparisons are not significant at all). FFT-Based beats Matrix at 1M and 100M, loses at 100K and ties at
10M; it never beats Fast Doubling, so the hypothesis that forcing the FFT pays
off at large N is still unconfirmed at 100M. On allocated bytes the order is not
the time order at 100M: FFT-Based allocates 347.1 MiB per op against Fast
Doubling's 561.5 MiB, so "most memory-efficient" above holds at the baseline
sizes, not at 100M. The p-values assume independent
samples, and here the ten samples of a group run back to back (file order), so a
drift of the machine during the run would read as a difference between groups;
nothing in the test guards against it.

**Slope between sizes, against the documented bound.** Ratio of medians per decade
of N, with its base-10 logarithm — the local exponent — in parentheses:

| Decade | Fast Doubling | Matrix Exp. | FFT-Based |
|---|---|---|---|
| 100K → 1M | ×27.7 (1.44) | ×24.3 (1.39) | ×6.7 (0.82) |
| 1M → 10M | ×8.1 (0.91) | ×4.2 (0.63) | ×6.2 (0.80) |
| 10M → 100M | ×6.0 (0.78) | ×10.0 (1.00) | ×7.4 (0.87) |

[FFT.md § Complexity Analysis](algorithms/FFT.md#complexity-analysis) puts every
pure-Go calculator at Θ(n^log2 3): ×38.5 per decade. That exponent is asymptotic,
and FFT.md derives it from the `k = 16` cap, which the `"fast"` and `"fft"` paths
reach only from n ≈ 9.06·10^8 — beyond this curve. Below the cap `k` grows with N
and FFT.md states no law for time. Nor can three wall-clock ratios per calculator
estimate an exponent. What the table does show:

- **Six of the nine ratios are below ×10**, and Matrix 10M → 100M sits at ×10.0.
  Work cannot grow slower than the result, which has ≈ 0.694·n bits, so a ratio
  under ×10 means the elapsed time at the smaller size is not proportional to
  work: fixed costs weigh there, or parallel efficiency changes between the two
  sizes, or the multiplication path does (Matrix switches to the FFT between 1M
  and 10M, at N = 1,768,788). For Fast Doubling between 10M and 100M (×6.0), one
  candidate, read from
  `shouldParallelizeMultiplicationCached` (`internal/fibonacci/fastdoubling.go`)
  and not measured: at N = 10M the largest operand, F(5M), has ≈ 3.47M bits, below
  `ParallelFFTThreshold`, so Fast Doubling runs its three FFT products one after
  another (each still recursing in parallel inside `bigfft`); at N = 100M the last steps' operands (≈ 34.7M bits for F(50M)) are above
  it and the three run concurrently.
- **One ratio stays inside a single multiplication regime.** At 100K and 1M Fast
  Doubling never reaches the FFT (its crossover is N = 1,440,422,
  [FFT.md § Crossover Point](algorithms/FFT.md#crossover-point)), so every product
  goes to `math/big` — Karatsuba above 40 words, schoolbook below. ×27.7 is under the ×38.5 of the pure power law, as
  lower-order terms and fixed per-call costs at 100K would make it; with ± 46 % on
  the 100K median that is consistent with the Karatsuba regime, not a measurement
  of its exponent.

So the curve neither confirms nor refutes Θ(n^1.585). What it establishes, on this
host only: the elapsed time to F(100M) — 0.19 s to 0.35 s depending on the
calculator — and the ordering above. No second host has reproduced it; the
Core Ultra 9 275HX mixes performance and efficiency cores, and which ones a run
lands on is not controlled.

### Running Benchmarks

```bash
# Run all benchmarks
go test -bench=. -benchmem ./internal/fibonacci/

# Benchmark specific algorithm (sub-benchmarks of BenchmarkFibonacci)
go test -bench='BenchmarkFibonacci/FastDoubling' -benchmem -run='^$' ./internal/fibonacci/

# Benchmark with specific iteration count
go test -bench='BenchmarkFibonacci/FastDoubling' -benchtime=5x -run='^$' ./internal/fibonacci/
```

### Regression baseline (>= 5 %, local discipline)

Any commit touching `internal/fibonacci/` or `internal/bigfft/` should be
verified locally against `docs/audits/bench-baseline.txt` using
`benchstat`. The convention is: **no sub-benchmark regression > 5 %** in
`BenchmarkFibonacci/(FastDoubling|MatrixExp|FFTBased)`.

```bash
go test -bench='BenchmarkFibonacci/(FastDoubling|MatrixExp|FFTBased)' \
    -benchmem -run='^$' -count=5 -benchtime=1x ./internal/fibonacci/ > /tmp/new.txt
benchstat docs/audits/bench-baseline.txt /tmp/new.txt
```

The flags must match the baseline exactly (these are the flags `make
bench-baseline` uses to write the file); `make benchmark` (`-bench=.`,
no `-count`) is **not** benchstat-comparable to the baseline.

Refresh the baseline on a quiet machine when an intentional perf change
lands :

```bash
make bench-baseline                            # writes docs/audits/bench-baseline.txt
git add docs/audits/bench-baseline.txt && git commit -m 'perf(bench): refresh baseline'
```

The dynamic threshold manager was removed on 2026-09-23 (ADR-0013 D1,
EVAL-10): measured neutral at `-count=8`, with +17.9 % allocs/op at F(1M).
The measurement that decided it stays archived in
[`docs/audits/bench-dtm-2026-09.txt`](audits/bench-dtm-2026-09.txt); the
benchmark that produced it left with the code and is in git history.

### Versioned benchmark snapshots (regression tracking)

To compare performance across Git revisions on the **same machine**, use a fixed command and record the environment:

1. **Record a snapshot** (writes `build/bench/snapshot-*.txt` with `go version`, full Git SHA, and benchmark output):

   ```bash
   make bench-versioned
   ```

   This runs `go test` with fixed flags: `-bench='BenchmarkFibonacci/(FastDoubling|MatrixExp|FFTBased)' -benchmem -count=3 -benchtime=2s ./internal/fibonacci/`.

2. **Annotate the result**: note the Git tag or commit (`git rev-parse HEAD`) in your changelog or ticket when you archive a snapshot. Single-run numbers are noisy; compare trends only on an idle machine, same flags, same `GOMAXPROCS` if you tune it.

3. **CHANGELOG** (optional): add a one-line entry such as
   `Perf: benchmark snapshot @ <SHA> — FastDoubling ns/op ±X% vs previous main` when you publish a measured change.

The baseline medians table in [Reference Benchmarks](#reference-benchmarks) comes from `docs/audits/bench-baseline.txt` (linux/amd64, 24 threads, stamped `baseline-2026-07-07`) and carries no Go-version stamp. It documents *someone else's* runner — your snapshots document *yours*.

## Hardware heuristic defaults

When CLI thresholds are left at **0** (auto) and no valid calibration profile is loaded, FibCalc applies `internal/config.ApplyAdaptiveThresholds`, which calls `EstimateOptimalParallelThreshold`, `EstimateOptimalFFTThreshold`, and `EstimateOptimalStrassenThreshold`. These functions use:

- **`runtime.NumCPU()`** for parallelism tiers (unchanged broad behavior).
- **`HardwareHeuristic`** (`internal/config/hardware.go`): on **amd64** and **386**, **AVX2** and **AVX-512 (AVX512F)** are read from `golang.org/x/sys/cpu` to nudge defaults (e.g. slightly lower FFT crossover on SIMD-rich CPUs, adjusted Strassen threshold when at least four cores are available).

Diagnostics and the in-package unit tests (`internal/config/hardware_test.go`) exercise the unexported `estimateParallelThresholdForHeuristic`, `estimateFFTThresholdForHeuristic`, and `estimateStrassenThresholdForHeuristic` with a synthetic `HardwareHeuristic`. Cached calibration profiles store `cpu_heuristic_key` (profile format v4 since audit M-01) so a change in SIMD class invalidates stale JSON — see [CALIBRATION.md](CALIBRATION.md).

## Implemented Optimizations

### 1. Zero-Allocation Strategy

#### Problem
Fibonacci calculations for large N create millions of temporary `big.Int` objects, causing excessive garbage collector pressure.

#### Solution
Using `sync.Pool` to recycle calculation states:

```go
var statePool = sync.Pool{
    New: func() any {
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
- Fewer allocations per calculation. The one figure the repo can show for the
  combined pooling work is `docs/audits/bench-baseline.txt`: 1.32 MB per op at
  F(1M) and 17.38 MB at F(10M) for Fast Doubling, against 6.33 MB / 92.25 MB for
  Matrix Exponentiation. No before/after measurement of pooling in isolation
  exists here; the "95 %+ / 20-30 %" figures this section used to carry had no
  backing artifact and were removed on 2026-08-07.
- Reduced GC pause times

**Calculation Arena (state-bound)**: For N > 1,000 a contiguous `CalculationArena` pre-allocates all 5 `big.Int` backing arrays from a single `[]big.Word` block, reducing GC tracking overhead and memory fragmentation. The arena is owned by `CalculationState` and travels through the same `sync.Pool`: `AcquireStateForN(n)` reuses the existing arena (`Reset()` only) when the previous tenancy was large enough, otherwise it reallocates. `ReleaseStateWithResult(s, src)` deep-copies the result out of the arena before resetting it and detaches every state slot before pool return, so a subsequent acquisition cannot alias another caller's result. The arena falls back to heap allocation when exhausted, and is dropped (not pooled) past `maxArenaPoolWords` (~50M words ≈ 400 MB) to bound resident memory.

**GC-immune state cache and per-state FFT scratch (2026-06-10)**: `sync.Pool` alone cannot retain the arena across calls — the GC-disable pattern of large calculations (`GCController`) re-enables GC right after every call, and that collection flushes the pool, so each repeated call paid a full arena reallocation (~46 % of all allocations at F(10M)). Since commit `fa13bfd`, each `FastDoublingCalculator` keeps a single-slot, **GC-immune** cache of its last released state (`cachedState`, an `atomic.Pointer[CalculationState]`), capped at `maxCachedArenaWords` (4M words ≈ 32 MB — which covers **n up to ≈ 36.9M**, since `arenaTotalWords(n) = (⌊n × 0.69424 / 64⌋ + 1) × 10` after the ADR-0009 R4 ×15 → ×10 change; the "roughly 20M" figure that used to appear here was computed against the old ×15 factor); larger arenas keep the historical pool-only behavior. Since commit `7999c39` (F-012), the FFT forward-transform `BumpAllocator` is also carried by the `CalculationState`: it is acquired once per calculation at final-operand size and only `Reset()` between doubling steps, instead of being re-acquired and re-grown at every step. Cumulative effect measured 2026-06-10: FastDoubling/10M 33.30 ms → 28.20 ms, B/op at 10M ~−70 %, geomean sec/op −12.0 % ([`CHANGELOG.md`](../CHANGELOG.md)).

### 2. 2-Tier Adaptive Multiplication

> **Which calculators actually reach this switch is decided elsewhere.** The two tiers below
> describe `smartMultiply` itself; they are the real routing only on the `"matrix"` path. On
> `"fast"` the threshold is read one level up and tier 1 here is unreachable, and on `"fft"` it
> is not read at all. Canonical description, per path and per threshold value:
> [`algorithms/FFT.md` § FFT Routing](algorithms/FFT.md#fft-routing).

The `smartMultiply` function selects the optimal multiplication algorithm based on operand bit size:

```go
func smartMultiply(z, x, y *big.Int, fftThreshold int) (*big.Int, error) {
    bx := x.BitLen()
    by := y.BitLen()

    // Tier 1: FFT multiplication (one-level Schönhage-Strassen, see FFT.md)
    if fftThreshold > 0 && bx > fftThreshold && by > fftThreshold {
        return bigfft.MulTo(z, x, y)
    }

    // Tier 2: Standard math/big (uses Karatsuba internally for large operands)
    return z.Mul(x, y), nil
}
```

| Tier | Algorithm | Complexity | Activation Threshold (default) |
|------|-----------|------------|-------------------------------|
| 1 | FFT (Schönhage-Strassen, one level) | Θ(n^1.585), smaller constant ([FFT.md](algorithms/FFT.md#complexity-analysis)) | > 500,000 bits |
| 2 | Standard `math/big` | O(n^2) / O(n^1.585) | Below FFT threshold |

> **Note — sub-threshold cost is library-bound.** Below `DefaultFFTThreshold` (500,000 bits), wall time is dominated by `math/big`'s Karatsuba multiplication and `sync.Pool` P-pinning, not by project code. This is the expected behavior under the FFT threshold and is not a regression of the calculator itself.

### 3. Multi-core Parallelism

The three main multiplications in the Fast Doubling algorithm can be parallelized via the `DoublingStepExecutor.ExecuteStep` method. The caller decides: `DoublingFramework.ExecuteDoublingLoop` evaluates `shouldParallelizeMultiplicationCached` per iteration and passes the verdict as `ExecuteStep`'s `inParallel` argument.

#### Considerations

- **Activation threshold**: `ParallelThreshold` (default: 4096 bits)
- **Disabled with FFT**: Parallelism is disabled when FFT is used as FFT already saturates the CPU
- **Parallel FFT threshold**: Re-enabled above 5,000,000 bits (`ParallelFFTThreshold`)

### 4. Strassen Algorithm

For matrix exponentiation, the Strassen algorithm reduces the number of multiplications from 8 to 7:

```
Classic 2x2 multiplication: 8 multiplications
Strassen-Winograd 2x2 (implemented): 7 multiplications + 15 additions/subtractions
  (the classical Strassen formulation needs 18)
```

Enabled via `StrassenThreshold` (default: 3,072 bits via config; internal default: 256 bits) when matrix elements are large enough for the multiplication savings to compensate for additional additions. The per-calculation threshold is set via `Options.StrassenThreshold`; the internal default is reachable through `fibonacci.SetDefaultStrassenThreshold()` but is a test-only fallback — `normalizeOptions` fills a zero `StrassenThreshold` with 3,072 before any matrix multiply, so production never reads it.

### 5. Symmetric Matrix Squaring

Specific optimization for squaring symmetric matrices (where b = c), reducing multiplications from 8 to 4.

### 6. GC Controller

For large calculations (N ≥ 1M), the `GCController` suppresses *heap-growth-triggered* GC during computation (`debug.SetGCPercent(-1)`). The motivation is `GOGC=100`'s default behaviour — the live heap is allowed to double before a cycle triggers — but the repo carries no peak-RSS measurement with and without the controller.

**The guarded region is not GC-free.** The same `Begin()` installs a soft memory limit of `3 × MemStats.Sys` (`DefaultMemoryLimitMultiplier`, `internal/fibonacci/memory/gc_control.go`), and a `GOMEMLIMIT` is honoured by the runtime *even with `GOGC=off`* — that is precisely its stated purpose here: the constant's own doc comment says it "acts as an OOM safety net: if the calculation runs away, the Go runtime will trigger emergency GC instead of letting the process consume unbounded memory". So the accurate claim is: no *GOGC-driven* cycle runs inside the region, and a memory-limit-driven cycle can. The repo carries no measurement of how often the limit is actually reached.

| Mode | Activation | Behavior |
|------|-----------|----------|
| `auto` (default) | N ≥ 1,000,000 | Disable GC during calculation |
| `aggressive` | Always | Disable GC regardless of N |
| `disabled` | Never | Standard GC behavior |

The mode defaults to `auto` (selected on calculation size) and is user-overridable through the `--gc-control` flag or the `FIBCALC_GC_CONTROL` environment variable.

> **Note — concurrent comparison (`--algo all`).** When several calculators run in parallel (each with its own `GCController`), GC disable/restore is serialized by a package-level refcount (`gcGlobalMu`/`gcActiveDepth`/`gcSavedPercent`): GC stays off while *any* sibling runs and the real `GOGC` is restored exactly once when the *last* one finishes. See [`docs/adr/0005-gc-control-concurrent.md`](adr/0005-gc-control-concurrent.md).

### 7. Memory Budget Estimation

Pre-calculate estimated memory usage before starting with `--memory-limit`:

`memory.EstimateMemoryUsage(n)` (`internal/fibonacci/memory/budget.go`) is the sole
source of these numbers. It computes `bytesPerFib = (⌊n × 0.69424 / 64⌋ + 1) × 8` and
totals **180 × bytesPerFib**, plus a flat **10 MiB** floor once the FFT machinery is
engaged at all (`n > 93`). The four terms are a descriptive split of one calibrated
total — state 36× + FFT buffers 39× + transform cache 48× + overhead 57× — so moving
weight between them (as the M-08 cache bound did, 4× → 48×) must keep the sum constant.

| N | Estimated Peak Memory |
|---|---|
| 1M | ~24.9 MB |
| 10M | ~159 MB |
| 100M | ~1.5 GB |
| 1B | ~14.6 GB |
| 5B | ~72.7 GB |

Measured on the binary, 2026-09-03 and re-verified 2026-09-04 (same five totals):
`fibcalc -n <N> -memory-limit 1K -algo fast` prints the estimate in its refusal
message, e.g. at N=10M `State: 29.8 MB, FFT: 32.3 MB, Cache: 39.7 MB,
Overhead: 57.2 MB, Total: 159.0 MB exceeds limit 1K.`

> **Re-modelled by audit H-03 (2026-09).** The previous model totalled 15 × bytesPerFib
> and put F(10M) at ~12 MB against **141 MB** actually observed — it counted neither the
> ×10 arena over-sizing, nor `sync.Pool` pre-warming (the largest single term, whose
> power-of-four size classes make the true cost a step function), nor the fact that
> `--algo all` runs three calculators at once. It under-estimated by 5× to 12× at every
> measured point, which emptied `--memory-limit` of its meaning. The estimate is now a
> deliberate **safety bound**, between 1.0× and 2.5× the measured figure across the
> range and never under. Raw measurements: [`docs/audits/mem-baseline-2026-09.txt`](audits/mem-baseline-2026-09.txt).
> **A limit tuned against the old figures will now be more constraining.**

If the estimate exceeds the limit, the tool exits with an error and suggests `--last-digits K` as an alternative. Note that this is the estimator's figure, not a measured RSS: pick `--memory-limit` against this model, not against observed process memory. A malformed `--memory-limit` (or an out-of-range `--last-digits`) is now rejected at flag-parsing time, on every mode, instead of only on the paths that reached the check (audit M-02).

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

> The programmatic entry point is `calibration.RunCalibration(ctx, out, calculatorRegistry, profilePath, progressDisplay, colorProvider) int`; see [CALIBRATION.md](CALIBRATION.md) for the full API and the 3-tier fallback.

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
| `FFTCacheMaxEntries` | see below | Maximum number of cached FFT transforms |
| `FFTCacheEnabled` | `true` | Enable/disable FFT transform caching |
| (no option) `MaxBytes` | `48 × size(F(n))` | Byte ceiling on the whole cache, installed by `configureFFTCache` from `n` (audit M-08); not settable through `Options` |

> **The entry cap is not a memory bound (M-08).** An entry holds `K × (n+1)` words —
> roughly twice its operand — so a fixed entry budget lets the cache grow linearly with
> the Fibonacci index, and nothing frees it between calculations in a long-lived process
> (TUI restart, calibration sweep). Since the 2026-09 audit the cache is also capped in
> bytes at `FFTCacheMaxBytesFactor = 48` times the size of F(n) (`internal/fibonacci/constants.go`),
> sized to hold one calculation's transforms. A tighter bound was measured and **rejected**:
> 4× (2 entries at F(10M)) cost MatrixExp/10M **+22 % sec/op**, +76 % B/op and +137 %
> allocs/op — see [`docs/audits/bench-fftcache-2026-09.txt`](audits/bench-fftcache-2026-09.txt)
> and [ADR-0010 R1](adr/0010-audit-2026-09-decisions.md).

> **`FFTCacheMaxEntries` has no fixed default of 256.** `bigfft.DefaultTransformCacheConfig()` does return `MaxEntries: 256` (`internal/bigfft/fft_cache.go:DefaultTransformCacheConfig`), but `configureFFTCache` overrides it whenever the option is left at 0 and `n > 0`, computing `clamp(2 × bits.Len64(n), 64, 4096)` (`internal/fibonacci/options.go:configureFFTCache`). For n = 10M that is `2 × 24 = 48`, clamped up to **64**. The dynamic value can never reach 256: `bits.Len64` maxes out at 64, so the expression tops out at 128. The 256 constant only reaches a caller who bypasses `configureFFTCache`, or who leaves `n = 0`. The doc comment on `Options.FFTCacheMaxEntries` (`internal/fibonacci/options.go:Options.FFTCacheMaxEntries`) states this rule directly: it says the field is sized from n as `clamp(2*bits.Len64(n), 64, 4096)`, "which is 64..128 in practice since bits.Len64 caps at 64; the package default of 256 only applies when n is 0."

**Cache reach is path-specific — the default algorithm never touches it.** The
FFT transform cache (`internal/bigfft/fft_cache.go`) is consulted **only** by
`TransformCached*` / `MulCachedWithBump` / `SqrCachedWithBump`, which are reached
exclusively from `bigfft.Mul` / `Sqr` / `MulTo` / `SqrTo`
(`fft_core.go:fftmulTo`, `fftsqrTo`). Which production callers get there:

```mermaid
flowchart LR
    FD["FastDoublingCalculator<br/>AdaptiveStrategy.ExecuteStep"] --> EDS
    FO["FFTOnlyStrategy.ExecuteStep<br/>--algo fft loop"] --> EDS["executeDoublingStepFFT<br/>internal/fibonacci/fft.go"]
    EDS --> TWB["Poly.TransformWithBump<br/>NO cache lookup"]
    MX["MatrixExponentiationCalculator<br/>matrix_ops.go"] --> SM["smartMultiply / smartSquare<br/>internal/fibonacci/fft.go"]
    FOM["FFTOnlyStrategy.Multiply / Square"] --> BM
    CAL["internal/calibration<br/>microbench.go"] --> BM
    SM --> BM["bigfft.Mul / Sqr / MulTo / SqrTo"]
    BM --> CACHE[("global transform cache<br/>MulCachedWithBump / SqrCachedWithBump")]
```

So the inter-iteration cache speedup does not apply to `--algo fast` (the
default), nor to the `--algo fft` **loop**, which runs the same
`executeDoublingStepFFT`: hits and misses are both zero on those paths. That
conclusion follows from the call graph alone and needs no benchmark. Any
cache-speedup claim would concern the matrix calculator, `FFTOnlyStrategy`'s
`Multiply`/`Square` helpers and direct `bigfft` calls — and none is measured
here.

> The invariant `putByKey` allocates a fresh backing buffer on every insert (no
> eviction-time recycling) still holds; its rationale lives in the `putByKey`
> doc comment (`internal/bigfft/fft_cache.go`), which records it as an Audit-PRD
> E1-R4 / [ADR-0002](adr/0002-recover-strategy.md) follow-up.
>
> A `BenchmarkCacheImpact` figure (22.95 ms vs 21.18 ms) used to be quoted here
> and attributed to [`CHANGELOG.md`](../CHANGELOG.md); that attribution was
> **false** — `grep -n "22.95\|BenchmarkCacheImpact" CHANGELOG.md` returns
> nothing (re-run 2026-09-04, exit 1), and no run of that benchmark is archived
> anywhere in the repo. The numbers were removed on 2026-08-07. The benchmark
> could not have measured the cache in any case: it drives a
> `FastDoublingCalculator` (`internal/fibonacci/cache_bench_test.go`), i.e. the
> uncached branch of the diagram above. Reworking the default step to use the
> cache stays a won't-fix without a supporting benchmark.

#### FFT Parallelism (bigfft package)

| Variable (unexported atomic) | Accessor | Default | Description |
|----------|----------|---------|-------------|
| `parallelFFTRecursionThreshold` | `bigfft.GetParallelFFTRecursionThreshold()` | 4 | Minimum FFT size (log2) for parallel recursion |
| `maxParallelFFTDepth` | `bigfft.GetMaxParallelFFTDepth()` | 3 | Maximum depth of parallel FFT recursion |

Both are `atomic.Uint64` package variables seeded in `init()`; they are runtime-configurable via `bigfft.SetFFTParallelismConfig()`.

All threshold parameters are configured via the `fibonacci.Options` struct:

```go
opts := fibonacci.Options{
    ParallelThreshold:  4096,
    FFTThreshold:       500_000,
    StrassenThreshold:  3072,
    FFTCacheEnabled:    boolPtr(true),
    FFTCacheMaxEntries: 256,
    FFTCacheMinBitLen:  100_000,
}
```

## Performance Monitoring

### Go Profiling

```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench='BenchmarkFibonacci/FastDoubling' -run='^$' ./internal/fibonacci/
go tool pprof cpu.prof

# Memory profiling
go test -memprofile=mem.prof -bench='BenchmarkFibonacci/FastDoubling' -run='^$' ./internal/fibonacci/
go tool pprof mem.prof

# Trace
go test -trace=trace.out -bench='BenchmarkFibonacci/FastDoubling' -run='^$' ./internal/fibonacci/
go tool trace trace.out
```

### Profiling the binary itself

Since 2026-09-07 (audit OBS-02) the binary writes its own pprof profiles, so a
real run — GC controller, arena and progress channel in place — can be profiled
without going through a benchmark:

```bash
fibcalc -n 50000000 -algo fast -cpuprofile cpu.prof -memprofile mem.prof
go tool pprof -top cpu.prof
go tool pprof -sample_index=alloc_space -top mem.prof
```

`-cpuprofile` covers the whole run. `-memprofile` is a heap profile written when
the run ends, after a forced `runtime.GC()`, so it shows what survives the
calculation rather than the peak.

### Escape analysis

```bash
go build -gcflags=-m ./internal/fibonacci/ 2>&1 | grep -E 'moved to heap|escapes to heap'
```

Lists every value the compiler moves to the heap. This is the check audit
MEM-03 asked to be written down: a new `escapes to heap` line inside a hot loop
is a regression `benchstat` will show as B/op before it shows as sec/op.

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
- 4 multiplications per loop iteration when the exponent bit is clear, 11-12 when
  it is set (one symmetric squaring, plus one full matrix multiply on a set bit),
  against a flat 3 for Fast Doubling — see
  [algorithms/MATRIX.md](algorithms/MATRIX.md#comparison-with-fast-doubling)
- Slower than Fast Doubling at both sizes the baseline covers and at all four of
  the [scale curve](#scale-curve-four-sizes-one-host)

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

1. **Memory**: `EstimateMemoryUsage` puts F(1 billion) at ~14.6 GB (safety bound, re-modelled by audit H-03). Use `--memory-limit` to validate before starting.
2. **Time**: no N above 100M is timed in this repo; the [scale curve](#scale-curve-four-sizes-one-host) stops there, at 0.19 s to 0.35 s on one host
3. **FFT Contention**: The FFT algorithm saturates cores, limiting external parallelism
4. **Workaround**: Use `--last-digits K` for O(K) memory usage with arbitrarily large N.
