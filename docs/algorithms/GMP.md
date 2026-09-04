# GMP-Based Calculator

## Overview

The GMP-based calculator utilizes the [GNU Multiple Precision Arithmetic Library (GMP)](https://gmplib.org/) to perform Fibonacci calculations, delegating every arithmetic operation to GMP's C/assembly routines instead of Go's `math/big`. Whether that wins, and above which N, is **not measured in this repo** — see [Performance](#performance) below. The type comment in `calculator_gmp.go` asserts an advantage above N = 100,000,000 and a CGO-overhead penalty below it; no artifact here backs either number.

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

func RegisterGMPCalculator(f *DefaultFactory) {
    f.Register("gmp", func() CoreCalculator { return &GMPCalculator{} })
}

func init() {
    RegisterGMPCalculator(globalFactory)
}
```

The `init()` only targets the package-private `globalFactory` kept for gmp builds. A factory you build yourself with `fibonacci.NewDefaultFactory()` pre-registers `"fast"`, `"matrix"` and `"fft"` only; add the `"gmp"` algorithm to it explicitly with `fibonacci.RegisterGMPCalculator(factory)`.

## Usage

### Go API

```go
// RegisterGMPCalculator only exists when built with -tags=gmp
factory := fibonacci.NewDefaultFactory()
fibonacci.RegisterGMPCalculator(factory)
calc, err := factory.Get("gmp")
if err != nil {
    // "gmp" not registered in this factory
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

> **The two benchmarks share no N, so that last command does not compare
> anything.** `BenchmarkGMPCalculator` (`calculator_gmp_test.go`) runs
> n = 100 / 1,000 / 10,000 through `CalculateCore` directly;
> `BenchmarkFibonacci` (`fibonacci_test.go`) runs n = 1,000,000 / 10,000,000
> through the `Calculator` wrapper. A real head-to-head requires editing one of
> the two size lists so they overlap.

## Performance

**This repo contains no GMP measurement.** Every benchmark artifact in
[`docs/audits/`](../audits/) covers `FastDoubling`, `MatrixExp` and `FFTBased`
only — `BenchmarkGMPCalculator` is behind the `gmp` build tag, so no artifact
here could contain it, and none does. Earlier revisions of this page carried
figures for CGO call overhead, a crossover around N = 1,000,000 and a net GMP
advantage above N = 100,000,000; none of them was ever backed by a run in this
repo, so they were removed on 2026-08-07 rather than restated more cautiously.

What is structurally true and checkable in the source: every arithmetic operation
crosses the CGO boundary (`internal/fibonacci/calculator_gmp.go`), so a per-call
cost exists — its size, and where it stops mattering, are unmeasured here.

To produce a real number, install the headers (`sudo apt-get install libgmp-dev`),
make the two benchmarks meet at a common N (see the note under
[Running Tests with GMP](#running-tests-with-gmp) — as committed they do not),
run

```bash
go test -tags=gmp -bench='Benchmark(Fibonacci|GMPCalculator)' -benchmem -run='^$' ./internal/fibonacci/
```

and archive the output under `docs/audits/` before quoting anything from it.

> **This page was verified by reading `calculator_gmp.go`, not by running it.**
> The file is behind `//go:build gmp` and needs CGO plus libgmp; neither is
> available on the Windows host used for the 2026-08-09 pass. Every statement
> here about the GMP path is a source claim, not an execution result.

## Implementation Details

- **Algorithm**: Fast Doubling (iterative, MSB-to-LSB), factored identity `F(2k) = F(k)·(2·F(k+1) − F(k))` — `gmpDoublingStep`
- **Arithmetic**: Uses `github.com/ncw/gmp` bindings to call `libgmp` (`go.mod`: `github.com/ncw/gmp v1.0.5`)
- **Memory Management**: four `gmp.Int` (`a`, `b`, `t1`, `t2`) allocated per `CalculateCore` call and reused across every iteration of the loop; no `sync.Pool`, no arena
- **Concurrency**: none — the three multiplications of a step run sequentially; the `Options` thresholds (`ParallelThreshold`, `FFTThreshold`, `StrassenThreshold`) are ignored on this path
- **Result conversion**: `gmpToStdBigInt` copies through `g.Bytes()` into a fresh `big.Int` — one full serialize/parse of the result per call
- **File**: `internal/fibonacci/calculator_gmp.go`
- **Name()**: Returns `"GMP (Fast Doubling)"`
- **Registration**: `"gmp"` key, but only in a factory you register it into yourself. `init()` registers into the package-private `globalFactory` (`calculator_gmp.go`, its `globalFactory` var and `init`), which no caller reads; `app.New` builds its own via `NewDefaultFactory()` (`internal/app/app.go:New`, `registry.go:NewDefaultFactory` — `fast`/`matrix`/`fft`). `fibcalc -algo gmp` is therefore rejected even in a `-tags gmp` build.

## Research backends beyond GMP

FibCalc may cite **FLINT**, other LGPL/GPL C libraries, or experimental arbitrary-precision stacks as *research comparisons*. The current design constraints:

- **No additional mandatory C/C++ backend:** only `math/big` and optional GMP are integrated. Adding another requires a reproducible build matrix, license review, and golden-style equivalence tests on a bounded set of indices — see **ADR-010** in [ARCH.md](../ARCH.md).
- **Extension point:** new calculators register through `Register` on a factory built with `fibonacci.NewDefaultFactory()` (same pattern as `RegisterGMPCalculator` under the `gmp` build tag). Prototypes should live on a dedicated branch or fork until quality and legal criteria are met.
- **Equivalence:** when evaluating a candidate backend, compare against `"fast"` / `"gmp"` on shared `N` values and existing `testdata` where applicable.
