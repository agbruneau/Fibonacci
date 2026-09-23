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

An optional GMP-based calculator (`"gmp"`) is compiled in with `-tags=gmp`, and every factory `NewDefaultFactory()` builds then offers it: `fibcalc -algo gmp` works and `-algo all` compares four calculators (since 2026-09-23, EVAL-23; before that the tag registered it into a private factory nothing read) — see [`GMP.md`](GMP.md).

> **Note on Modular Fast Doubling**: unlike the three rows above, this is **not** a registered `CoreCalculator` and has no `Name()` method. It is the free function `FastDoublingMod(ctx context.Context, n uint64, m *big.Int) (*big.Int, error)` in `internal/fibonacci/modular.go`, reached only through the `--last-digits` CLI mode (which computes F(n) mod 10^K).

## Theoretical Comparison

### Complexity

All algorithms have the same asymptotic complexity:

```
Θ(M(n))
```

where M(n) is the cost of multiplying two n-bit numbers. The familiar O(log n · M(n)) — O(log n)
steps times one multiplication each — is an upper bound, not the order: the operands double
from one step to the next, so for any M with M(x)/x non-decreasing the per-step costs form a
geometric series dominated by the last step
([FAST_DOUBLING.md § Total Complexity](FAST_DOUBLING.md#total-complexity),
[MATRIX.md § Total Complexity](MATRIX.md#total-complexity)).

With the multiplication routines in this repository — `math/big` Karatsuba
[[3]](../REFERENCES.md#ref-3), [[13]](../REFERENCES.md#ref-13), and `internal/bigfft`, a
one-level Schönhage–Strassen FFT over the same Karatsuba [[1]](../REFERENCES.md#ref-1),
[[12]](../REFERENCES.md#ref-12) — M(n) = Θ(n^log2 3), so every pure-Go calculator is
Θ(n^log2 3) ≈ Θ(n^1.585) in the index n. Neither the O(n log n log log n) of recursive
Schönhage–Strassen [[1]](../REFERENCES.md#ref-1) nor the O(n log n) of Harvey–van der Hoeven
[[2]](../REFERENCES.md#ref-2) applies to them
([FFT.md § Complexity Analysis](FFT.md#complexity-analysis)). Under `-tags gmp` the products
are GMP's [[11]](../REFERENCES.md#ref-11). Numbers in brackets refer to
[`docs/REFERENCES.md`](../REFERENCES.md).

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

Let T(n) be the time to compute F(n), with k multiplications per step on operands that halve
going backwards from about γn/2 bits (γ = log2 φ ≈ 0.694):

```
T(n) ≈ k · Σ_j M(γn / 2^j)  ≤  k · M(γn)            (M(x)/x non-decreasing)
     = k · M(γn/2) / (1 − 2^-α)                      (M(x) = c·x^α; α = log2 3 gives k/2 · M(γn))
```

There is no log2(n) factor. The constant k represents the "multiplicative density" of the
algorithm, and it is what the comparison below turns on.

1. **Fast Doubling (k = 3)**:
   - Exactly 3 multiplications per loop iteration, for every iteration: `FK·FK1`, `FK²`, `FK1²`
   - F(2k) = F(k) * (2*F(k+1) - F(k)), evaluated as `2·FK·FK1 - FK²`
   - F(2k+1) = F(k+1)^2 + F(k)^2
   - It is the smallest k among the algorithms implemented here, but **not the smallest
     known**: GMP's `mpz_fib_ui` [[11]](../REFERENCES.md#ref-11) and Takahashi's algorithm
     [[7]](../REFERENCES.md#ref-7) double with two squarings per bit, using the (−1)^k terms
     of Cassini's identity [[8]](../REFERENCES.md#ref-8). Neither is implemented here; the
     cost model that puts this loop at about twice their multiplication work is in
     [FAST_DOUBLING.md § Against the two-squaring formulations](FAST_DOUBLING.md#against-the-two-squaring-formulations).

2. **Matrix Exponentiation (k = 4 when the exponent bit is clear, 11-12 when set)**:
   - Symmetric squaring runs every iteration but the last: 4 mults (`squareSymmetricMatrix`)
   - A set exponent bit adds one matrix multiplication: 7 with Strassen-Winograd, 8 classic
   - So the per-iteration k alternates between 4 and 11/12, averaging ~7.5-8 over a
     random exponent, against Fast Doubling's flat 3

**Conclusion**: Fast Doubling's constant factor k is strictly smaller (a flat 3 vs 4 or 11-12). The one measurement artifact the repo carries, `docs/audits/bench-baseline.txt` (linux/amd64, 24 threads, `-count=5 -benchtime=1x`, 2026-07-07), agrees at the two sizes it covers — medians 3.15 ms vs 6.03 ms (Matrix) and 5.13 ms (FFT) at N=1M; 23.87 ms vs 30.84 ms and 29.08 ms at N=10M. The scale curve (`docs/audits/bench-scale-2026-09.txt`, one Windows host, 10 samples) extends this to N = 100K and 100M: Fast Doubling has the lowest median at all four sizes, significantly except against FFT-Based at 10M ([Ordering](#ordering-as-far-as-it-is-established)). A smaller k lowers the constant; it does not decide the ordering by itself, since the calculators also differ in parallelism and in which multiplication path they take.

### Memory

| Algorithm | Temporary variables | Pool objects |
|-----------|---------------------|--------------|
| Fast Doubling | 5 big.Int | CalculationState (sync.Pool + per-instance GC-immune cache slot, ~32 MB cap) |
| Matrix Exp. | 3 matrices (res, p, tempMatrix) + 20 big.Int | matrixState |

> **Note (2026-06, commit `fa13bfd`)**: `FastDoublingCalculator` additionally retains the last released `CalculationState` in a per-instance, GC-immune cache slot (arena capped at 4M words ≈ 32 MB), preferred over the shared `sync.Pool` for repeated calls. Matrix Exp. and FFT-Based are unaffected (FFT-Based acquires its state through the shared `AcquireStateForN` pool path). Measured impact (2026-06-10, cumulative with the F-012 bump fix): `BenchmarkFibonacci/FastDoubling/10M` 33.30 ms -> 28.20 ms sec/op, ~-70 % B/op — see [`CHANGELOG.md`](../../CHANGELOG.md).

#### Resident memory, measured

[`docs/audits/mem-baseline-2026-09.txt`](../audits/mem-baseline-2026-09.txt)
records the `runtime.MemStats.Sys` delta across one calculation, **one process
per data point** — `Sys` never shrinks, so several points in one process would
all report the same high-water mark. `all` runs the three calculators
concurrently, as `--algo all` does by default. Blank cells were not run.

| n | `fast` | `fft` | `matrix` | `all` |
|---|---|---|---|---|
| 1,000 | 0 MB | | | |
| 100,000 | 0 MB | | | 6 MB |
| 1,000,000 | 9 MB | 18 MB | 13 MB | 23 MB |
| 5,000,000 | 34 MB | | | |
| 10,000,000 | 62 MB | 67 MB | 141 MB | 101 MB |
| 50,000,000 | 536 MB | | | |
| 100,000,000 | 617 MB | 460 MB | | |

Two things this table says that the `B/op` table above does not. First, the
resident gap between Fast Doubling and Matrix at F(10M) is **2.3x**, where the
allocated-bytes ratio is 5.3x — most of Matrix's extra allocation is churn
through pooled buffers, not retained footprint. Second, at F(1M) the two
metrics disagree on the *order*: Matrix allocates more than FFT-Based
(6.33 MB vs 5.38 MB `B/op`) but holds less (13 MB vs 18 MB `Sys`). At N = 100M
the only other figures are the scale curve's timings and `B/op`
([PERFORMANCE.md § Scale curve](../PERFORMANCE.md#scale-curve-four-sizes-one-host));
this table stays the only resident-memory reading there.

The same artifact is why `--memory-limit`'s estimator was rewritten (audit
H-03 / M-08): the old model was under the real figure at every measured point,
by 5x to 12x, so it validated runs that then consumed several times the
declared budget. The replacement is never under and at most 2.47x over, the
overshoot peaking where `n` sits just past one of `bigfft`'s power-of-four pool
size classes.

## Benchmarks

### What this repo has actually measured

Nine measurement artifacts live in [`docs/audits/`](../audits/). One is the
regression reference, two extend it — a scale curve and an external reference —
and the rest exist to justify a specific change:

| Artifact | Kind | What it measures | Host |
|---|---|---|---|
| [`bench-baseline.txt`](../audits/bench-baseline.txt) | reference | `BenchmarkFibonacci`, the three calculators at N = 1M / 10M — **the table below** | linux/amd64, 24 threads, 2026-07-07 |
| [`bench-scale-2026-09.txt`](../audits/bench-scale-2026-09.txt) | scale curve, 10 samples | `BenchmarkFibonacci`, the three calculators at N = 100K / 1M / 10M / 100M (EVAL-09) — read in [PERFORMANCE.md § Scale curve](../PERFORMANCE.md#scale-curve-four-sizes-one-host) | Core Ultra 9 275HX, windows/amd64, go1.27.0 |
| [`bench-gmp-2026-09.txt`](../audits/bench-gmp-2026-09.txt) | external reference, 5 samples | `FastDoubling` against `GMPCalculator` at N = 1M / 10M (EVAL-07) — read in [GMP.md § Performance](GMP.md#performance) | Core Ultra 9 275HX, WSL2 linux/amd64, go1.26.1 |
| [`bench-dtm-removal-2026-09.txt`](../audits/bench-dtm-removal-2026-09.txt) | benchstat A/B | `main` against the branch without the dynamic threshold manager (EVAL-10) | Core Ultra 9 275HX, windows/amd64, go1.27.0 |
| [`bench-fftcache-2026-09.txt`](../audits/bench-fftcache-2026-09.txt) | benchstat A/B | the same six cases, for the FFT-cache byte bound (M-08) | Core Ultra 9 275HX, windows/amd64, go1.27.0 |
| [`bench-poolclear-2026-09.txt`](../audits/bench-poolclear-2026-09.txt) | benchstat A/B, **both orders** | the same six cases, for the pool memclr narrowing (M-05) | idem |
| [`bench-dtm-2026-09.txt`](../audits/bench-dtm-2026-09.txt) | benchstat A/B | dynamic thresholds off vs on, N = 1M / 10M (M-04) | idem |
| [`mem-baseline-2026-09.txt`](../audits/mem-baseline-2026-09.txt) | one-shot probe | resident memory (`MemStats.Sys` delta), N = 1,000 … 100M (H-03 / M-08) | idem |
| [`microbench-stability-2026-09.txt`](../audits/microbench-stability-2026-09.txt) | repeatability | ten `QuickCalibrate()` runs, before/after M-01 | idem |

The four benchstat A/B files are **not** a cross-algorithm ranking: their two
columns differ by one code change, not by algorithm, and they were taken on a
different host and OS from the reference. On **timing**, only the scale curve
goes beyond 1M and 10M, to 100K and 100M, on one host; the wider N range in
`mem-baseline-2026-09.txt` is memory only, with no clock attached.

### The cross-algorithm reference

`docs/audits/bench-baseline.txt` records `BenchmarkFibonacci`
(`internal/fibonacci/fibonacci_test.go`) at two sizes only — N = 1,000,000 and
N = 10,000,000 — with `-count=5 -benchtime=1x`, on `linux/amd64`, 24 threads,
dated 2026-07-07. Medians of the five samples:

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
> were deleted rather than restated more cautiously. The only timing beyond 1M
> and 10M is now the scale curve, which stops at 100M.

### Ordering, as far as it is established

At the two baseline sizes, Fast Doubling is fastest and smallest, in that order:
Fast Doubling < FFT-Based < Matrix Exp. on time, Fast Doubling < FFT-Based <
Matrix Exp. on allocated bytes.

The scale curve (`bench-scale-2026-09.txt`, medians of 10, pairwise benchstat;
the full table and its caveats are in
[PERFORMANCE.md § Scale curve](../PERFORMANCE.md#scale-curve-four-sizes-one-host))
refines the time ordering on its one host:

| N | Fast / Matrix / FFT (median) | Established order |
|---|---|---|
| 100K | 143.2 µs / 342.8 µs / 876.7 µs | Fast < Matrix < FFT |
| 1M | 3.972 ms / 8.341 ms / 5.853 ms | Fast < FFT < Matrix |
| 10M | 32.28 ms / 35.23 ms / 36.53 ms | not separated at the Bonferroni level 0.004 (Fast vs Matrix p = 0.035; FFT p = 0.063, 0.579) |
| 100M | 193.2 ms / 351.4 ms / 272.0 ms | Fast < FFT < Matrix |

So FFT-Based sits between the other two only at 1M and 100M. An earlier revision
of PERFORMANCE.md warned, without an artifact, that the Fast Doubling / Matrix
ordering can invert at N ≥ 10M on some CPUs; on this CPU it does not, at 10M or
100M. No second CPU has been measured, so the warning is neither confirmed nor
ruled out elsewhere.

On memory, one qualification from the resident-memory table above: the `B/op`
order and the `Sys` order agree at F(10M) but **not** at F(1M), where Matrix
allocates more than FFT-Based yet holds less. The two artifacts come from
different hosts and operating systems, so the disagreement cannot be attributed
to the metric alone.

## Related work (not measured)

Two established implementations compute F(n) directly. Neither is measured in
this repo; they are cited so that a reader knows what a fair external comparison
would be, and how far the one archived measurement is from it.

- **GMP `mpz_fib_ui`** [[11]](../REFERENCES.md#ref-11) — the GNU MP library's own
  Fibonacci function, which doubles with two squarings per bit using the
  (−1)^k term of Cassini's identity, against this repo's three products. The cost
  model in [FAST_DOUBLING.md § Against the two-squaring formulations](FAST_DOUBLING.md#against-the-two-squaring-formulations)
  puts this repo's loop at about twice its multiplication work; that is a model,
  not a timing. What [`bench-gmp-2026-09.txt`](../audits/bench-gmp-2026-09.txt)
  measures instead is `GMPCalculator`, this repo's three-product loop on GMP
  `mpz` arithmetic ([GMP.md § Performance](GMP.md#performance)): GMP's
  multiplication under our algorithm, not GMP's algorithm. The binding the repo
  uses, `github.com/ncw/gmp` v1.0.5, has no Fibonacci entry point (no `fib` in its
  sources), so measuring `mpz_fib_ui` would take a CGO call of our own.
- **PARI/GP `fibonacci(x)`** — documented as "x-th Fibonacci number", library
  syntax `GEN fibo(long x)`
  (<https://pari.math.u-bordeaux.fr/dochtml/html/Combinatorics.html>, read on
  2026-09-23; the page states no version). Its documentation says nothing about
  the algorithm, and this repo has not read the PARI sources, so no claim is made
  here about how it compares.

## When to Use Each Algorithm

### Fast Doubling (`"fast"`)

**Recommended for**: general usage. Fastest and smallest of the three at both baseline sizes, and lowest median time at the four sizes of the scale curve (N = 100K to 100M, one host) — though not the fewest bytes at 100M, where FFT-Based allocates 347.1 MiB per op against its 561.5 MiB; behavior above N = 100M is untested here. Note the CLI does **not** default to it: `DefaultAlgo = "all"` (`internal/config/config.go`), which runs every registered calculator (`GetCalculatorsToRun`, `internal/orchestration/calculator_selection.go`). Pass `-algo fast` to run this one alone.

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

**Recommended for**: exercising the FFT path in isolation — FFT multiplication benchmarking, regression testing, FFT vs standard `math/big` comparison. It is **not** the faster calculator at any size this repo measures (see the tables above), and the type comment on `FFTBasedCalculator` (`internal/fibonacci/fft_based.go`) says so explicitly. The scale curve has it 40.82 % slower than Fast Doubling at 100M (272.0 ms vs 193.2 ms, p = 0.000), so a crossover where forcing FFT at every size pays off, if any, lies beyond F(100M) and is unmeasured here.

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
- **Bound**: `--last-digits K` is rejected above `config.MaxLastDigits = 10_000_000`, at parse time in `config.Validate` and again as defense in depth in `internal/app/calculate.go` (audit M-02)

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

Setting `FFTThreshold` is not the same as changing when the FFT runs, and how much
it changes depends on the calculator. On `"fast"` the value is compared against an
intermediate operand, so `500_000` keeps the FFT out of the picture entirely below
`N = 1 440 422` — the "medium" block above is a no-op on that path for most of its
range. On `"matrix"` it is compared against the matrix entries instead — a different
rule, and one that never sees a larger operand than `"fast"` does at the same `N`, so
the same 500 000 keeps the FFT out below `N = 1 768 788`. On `"fft"` the field is not
read at all. Before tuning it, read
[FFT.md § FFT Routing](FFT.md#fft-routing), which is the canonical description of
what this option actually gates.

## Conclusion

**Fast Doubling** is the recommended algorithm for all general use cases: it requires only 3 multiplications per iteration — the fewest of the three implementations here — and allocates the least up to N = 10M. `docs/audits/bench-baseline.txt` — the repo's only cross-algorithm baseline — shows it fastest and smallest at the two sizes it covers: medians of 3.15 ms / 1.32 MB per op at N=1M (vs Matrix 6.03 ms / 6.33 MB and FFT 5.13 ms / 5.38 MB) and 23.87 ms / 17.38 MB at N=10M (vs Matrix 30.84 ms / 92.25 MB and FFT 29.08 ms / 30.88 MB). `mem-baseline-2026-09.txt` puts it lowest on resident memory too at both sizes (9 / 62 MB). The scale curve (`bench-scale-2026-09.txt`, one Windows host) keeps it fastest at N = 100K and 100M as well (143.2 µs and 193.2 ms medians), but at 100M FFT-Based allocates less (347.1 MiB vs 561.5 MiB per op), as `mem-baseline-2026-09.txt` also has it on resident memory (460 MB vs 617 MB).

**Matrix Exponentiation** is valuable for educational purposes and result verification. Its elegant mathematical foundation (Q-matrix) makes it ideal for understanding the theory, and the Strassen optimization demonstrates practical algorithm design. In `docs/audits/bench-baseline.txt` it is slower than Fast Doubling by **+91 %** at N=1M and **+29 %** at N=10M — the gap narrows with N and is not a stable 30–50 % band. The scale curve says the narrowing does not continue: +139 % at 100K, +110 % at 1M, +9 % at 10M, +82 % at 100M, all significant on its one host.

**FFT-Based** is a specialized variant that forces FFT multiplication for all operations. At the two baseline sizes it is slower than Fast Doubling (5.13 ms vs 3.15 ms at N=1M; 29.08 ms vs 23.87 ms at N=10M) and allocates ~4x more at 1M. Whether forcing FFT at every size ever pays off — the usual argument being that FFT multiplication's lower growth rate eventually outweighs its overhead — is not borne out up to N = 100M: the scale curve has it 47.36 % slower than Fast Doubling at 1M, tied at 10M (p = 0.063) and 40.82 % slower at 100M, so the gap does not close steadily with N. Nothing here measures beyond 100M. For this repository's FFT the argument is weaker than usual: `internal/bigfft` runs one transform level over Karatsuba and keeps Karatsuba's exponent, so what it can win is a constant factor ([FFT.md § Complexity Analysis](FFT.md#complexity-analysis)). Its established use is exercising the FFT subsystem in isolation.
