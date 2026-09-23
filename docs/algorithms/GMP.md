# GMP-Based Calculator

## Overview

The GMP-based calculator utilizes the [GNU Multiple Precision Arithmetic Library (GMP)](https://gmplib.org/) to perform Fibonacci calculations, delegating every arithmetic operation to GMP's C/assembly routines instead of Go's `math/big`. Whether that wins is measured at two sizes only, on one host — see [Performance](#performance) below. Until 2026-09-23 the type comment in `calculator_gmp.go` asserted an advantage above N = 100,000,000 and a CGO-overhead penalty below it; it now states only what the measurement shows. The measurement stops at N = 10,000,000, so it tests neither former claim directly, and at the sizes it covers it does not show the penalty.

This implementation uses the **Fast Doubling** algorithm, like the `"fast"` strategy, but it is a separate loop, not the shared one:

- It does **not** use `DoublingFramework`, `AdaptiveStrategy`, `CalculationState`, the arena, or any parallelism. `CalculateCore` is a self-contained MSB→LSB loop over `bits.Len64(n)` (`internal/fibonacci/calculator_gmp.go:CalculateCore`).
- It evaluates the **factored** identity `F(2k) = F(k)·(2·F(k+1) − F(k))` (`gmpDoublingStep`: `t1 = 2b − a`, then `t1 = a·t1`), whereas the shared loop evaluates the expanded `F(2k) = 2·F(k)·F(k+1) − F(k)²`. Both are algebraically the same and both cost three multiplications; the operand shapes differ.
- The `--last-digits` path (`FastDoublingMod`, `modular.go`) uses the factored form too.

## Requirements

To use this calculator, you must have the GMP library and its development headers installed on your system.

### Installation

**Ubuntu/Debian:**
```bash
sudo apt-get install libgmp-dev
```

**macOS (via Homebrew):**
```bash
brew install gmp
```

**Fedora/Red Hat:**
```bash
sudo dnf install gmp-devel
```

**Windows:**
Requires MinGW or WSL with libgmp installed.

## Compilation

Because this implementation relies on CGO and an external C library, it is hidden behind a build tag (`gmp`) to prevent build failures on systems without GMP.

```bash
# Build with GMP support
go build -tags gmp -o fibcalc ./cmd/fibcalc
```

## Auto-Registration

When built with `-tags=gmp`, the GMP calculator auto-registers itself via an `init()` function:

```go
//go:build gmp

func init() {
    taggedRegistrations = append(taggedRegistrations, RegisterGMPCalculator)
}
```

`NewDefaultFactory()` applies every entry of `taggedRegistrations` after the three built-ins, so in a `-tags gmp` build each factory — including the one `app.New` builds — offers `"gmp"`: `fibcalc -algo gmp` works and `-algo all` compares four calculators. Until 2026-09-23 the `init()` registered into a private factory nothing read, and the CLI refused `-algo gmp` even with the tag (EVAL-23).

## Usage

### Go API

```go
// "gmp" is registered only when built with -tags=gmp
factory := fibonacci.NewDefaultFactory()
calc, err := factory.Get("gmp")
if err != nil {
    // built without the gmp tag
}
// nil progress channel disables progress reporting; to receive updates,
// pass a chan<- progress.ProgressUpdate (package internal/progress)
result, err := calc.Calculate(ctx, nil, 0, 100_000_000, fibonacci.Options{})
```

### Running Tests with GMP

```bash
# Run all tests with GMP support
go test -tags=gmp -v ./internal/fibonacci/

# Run benchmarks with GMP
go test -tags=gmp -bench=BenchmarkGMP -benchmem ./internal/fibonacci/

# Compare GMP vs native algorithms (native paths are sub-benchmarks of BenchmarkFibonacci)
go test -tags=gmp -bench='Benchmark(Fibonacci|GMPCalculator)' -benchmem -run='^$' ./internal/fibonacci/
```

`BenchmarkGMPCalculator` (`calculator_gmp_test.go`) runs n = 1,000,000 and
10,000,000 through the same `Calculator` wrapper and harness (`runBenchmark`) as
`BenchmarkFibonacci`, so `GMPCalculator/1M` and `FastDoubling/1M` measure the
same work (aligned on 2026-09-23, EVAL-07; before that the two shared no N).

## Performance

One measurement exists: [`docs/audits/bench-gmp-2026-09.txt`](../audits/bench-gmp-2026-09.txt)
(EVAL-07). From its header (lines 2–6): `go1.26.1 linux/amd64` under WSL2 Ubuntu on
an Intel Core Ultra 9 275HX, `nproc` 24, libgmp `2:6.3.0+dfsg-2ubuntu6.1`, commit
`751c1cc`, 2026-09-23, five samples per row:

```bash
go test -tags gmp -bench='BenchmarkFibonacci/FastDoubling|BenchmarkGMPCalculator' \
    -benchmem -run='^$' -count=5 -benchtime=1x ./internal/fibonacci/
```

Medians from `go run golang.org/x/perf/cmd/benchstat@v0.0.0-20260825160852-19be9d8e6c70 docs/audits/bench-gmp-2026-09.txt`;
the verdict column compares the two rows of one N after renaming them to the same
benchmark name, as in [PERFORMANCE.md § Scale curve](../PERFORMANCE.md#scale-curve-four-sizes-one-host):

| N | `fast` (`math/big`) | `gmp` | fast ÷ gmp | benchstat, gmp → fast |
|---|---|---|---|---|
| 1,000,000 | 4.037 ms | 3.324 ms | 1.21 | ~ (p = 0.151, n = 5): **no significant difference** |
| 10,000,000 | 31.49 ms | 41.06 ms | 0.77 | −23.31 % (p = 0.008, n = 5): `fast` faster |

What the ratio does and does not measure:

- **Same loop, not GMP's Fibonacci.** `GMPCalculator` is this repo's doubling
  loop with `mpz` arithmetic — three products per bit, like `fast`. It is **not**
  `mpz_fib_ui`, GMP's own function, which doubles with two squarings per bit
  [[11]](../REFERENCES.md#ref-11). The table measures GMP's arithmetic under this
  loop; `mpz_fib_ui` is unmeasured here
  ([COMPARISON.md § Related work](COMPARISON.md#related-work-not-measured)).
- **Parallel against sequential.** `fast` may run its three products concurrently
  and `internal/bigfft` recurses in parallel; `GMPCalculator` runs on one thread.
  The ratio is elapsed time on a 24-thread machine, not efficiency per core — no
  CPU time is recorded. The 10M win for `fast` is therefore not a claim that
  `math/big` multiplies faster than GMP.
- **Not the same exit.** The `gmp` time includes `gmpToStdBigInt`, one
  serialize/parse of the result into a `big.Int`; `fast` returns its own.
- **`B/op` is not comparable.** `gmp` reports 176.2 KiB at 1M and 1.656 MiB at
  10M, about twice the byte size of F(n) (F(10M) ≈ 0.87 MB) — consistent with the
  `Bytes()` + `SetBytes` copy above. libgmp allocates its limbs with `malloc`,
  outside the Go heap, where `-benchmem` does not look. (Inferred from the
  arithmetic, not traced.)
- **Five samples.** benchstat prints `± ∞` ("need >= 6 samples"), so there is no
  interval. The 1M verdict turns on one sample: `GMPCalculator/1M`'s first run,
  4,815,476 ns (line 11), is the slowest of both rows and of the same `-benchtime=1x`
  kind described in PERFORMANCE.md. Without it the four remaining GMP samples are
  all below every `fast` sample — which is why the 1.21 is a median ratio, not a
  result.
- **Same machine, not the scale curve's environment.** WSL2 and `go1.26.1` differ
  from the Windows `go1.27.0` run of `bench-scale-2026-09.txt`; the `FastDoubling` rows of the two
  files are not interchangeable (`FastDoubling/100M`: 219.1 ms here, 193.2 ms
  there). Take the ratio within this file only.

Against the former type comment: at 1M no CGO penalty is visible (the two do not
differ significantly), and at 10M the sequential GMP loop is slower than the
parallel `math/big` one. Neither size reaches the N > 100,000,000 that comment was
about; `BenchmarkGMPCalculator` stops at 10M.

The CI `gmp` job (`.github/workflows/ci.yml`) runs the same command on every push
and uploads `bench-gmp.txt`, stamped with the runner, CPU model and run ID. The
archived file is not one of those runs: it was taken locally, and a CI runner has a
different CPU.

## Implementation Details

- **Algorithm**: Fast Doubling (iterative, MSB-to-LSB), factored identity `F(2k) = F(k)·(2·F(k+1) − F(k))` — `gmpDoublingStep`
- **Arithmetic**: Uses `github.com/ncw/gmp` bindings to call `libgmp` (`go.mod`: `github.com/ncw/gmp v1.0.5`)
- **Memory Management**: four `gmp.Int` (`a`, `b`, `t1`, `t2`) allocated per `CalculateCore` call and reused across every iteration of the loop; no `sync.Pool`, no arena
- **Concurrency**: none — the three multiplications of a step run sequentially; the `Options` thresholds (`ParallelThreshold`, `FFTThreshold`, `StrassenThreshold`) are ignored on this path, libgmp selecting its own multiplication algorithm. For what those thresholds gate on the other paths, see [FFT.md § FFT Routing](FFT.md#fft-routing)
- **Result conversion**: `gmpToStdBigInt` copies through `g.Bytes()` into a fresh `big.Int` — one full serialize/parse of the result per call
- **File**: `internal/fibonacci/calculator_gmp.go`
- **Name()**: Returns `"GMP (Fast Doubling)"`
- **Registration**: `"gmp"` key in every factory `NewDefaultFactory()` builds under `-tags gmp` — `init()` appends `RegisterGMPCalculator` to `taggedRegistrations` (`registry.go`), so `fibcalc -algo gmp` works; see [Auto-Registration](#auto-registration).

## Research backends beyond GMP

FibCalc may cite **FLINT**, other LGPL/GPL C libraries, or experimental arbitrary-precision stacks as *research comparisons*. The current design constraints:

- **No additional mandatory C/C++ backend:** only `math/big` and optional GMP are integrated. Adding another requires a reproducible build matrix, license review, and golden-style equivalence tests on a bounded set of indices — see **ADR-010** in [ARCH.md](../ARCH.md).
- **Extension point:** new calculators register through `Register` on a factory built with `fibonacci.NewDefaultFactory()` (same pattern as `RegisterGMPCalculator` under the `gmp` build tag). Prototypes should live on a dedicated branch or fork until quality and legal criteria are met.
- **Equivalence:** when evaluating a candidate backend, compare against `"fast"` / `"gmp"` on shared `N` values and existing `testdata` where applicable.
