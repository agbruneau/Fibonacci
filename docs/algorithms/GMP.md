# GMP-Based Calculator

> Interactive architecture map: **[agbruneau.github.io/FibGo/dashboard/](https://agbruneau.github.io/FibGo/dashboard/)** (knowledge graph, 1128 nodes / 4782 edges / 9 layers / 12-step tour)

## Overview

The GMP-based calculator utilizes the [GNU Multiple Precision Arithmetic Library (GMP)](https://gmplib.org/) to perform Fibonacci calculations. GMP is widely regarded as the fastest library for arbitrary-precision arithmetic, often outperforming Go's standard `math/big` library for extremely large numbers (> 100 million bits).

This implementation uses the **Fast Doubling** algorithm, identical to the standard `"fast"` strategy, but delegates all arithmetic operations (addition, subtraction, multiplication, squaring) to GMP's highly optimized C assembly routines.

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

## Performance

**This repo contains no GMP measurement.** `docs/audits/bench-baseline.txt` is the
only benchmark artifact tracked, and it covers `FastDoubling`, `MatrixExp` and
`FFTBased` only — `BenchmarkGMPCalculator` is behind the `gmp` build tag and its
output has never been archived here. Earlier revisions of this page carried
figures for CGO call overhead, a crossover around N = 1,000,000 and a net GMP
advantage above N = 100,000,000; none of them was ever backed by a run in this
repo, so they were removed on 2026-08-07 rather than restated more cautiously.

What is structurally true and checkable in the source: every arithmetic operation
crosses the CGO boundary (`internal/fibonacci/calculator_gmp.go`), so a per-call
cost exists — its size, and where it stops mattering, are unmeasured here.

To produce a real number, install the headers (`sudo apt-get install libgmp-dev`),
run

```bash
go test -tags=gmp -bench='Benchmark(Fibonacci|GMPCalculator)' -benchmem -run='^$' ./internal/fibonacci/
```

and archive the output under `docs/audits/` before quoting anything from it.

## Implementation Details

- **Algorithm**: Fast Doubling (iterative, MSB-to-LSB)
- **Arithmetic**: Uses `github.com/ncw/gmp` bindings to call `libgmp`
- **Memory Management**: Reuses `gmp.Int` instances to minimize allocation overhead
- **File**: `internal/fibonacci/calculator_gmp.go`
- **Name()**: Returns `"GMP (Fast Doubling)"`
- **Registration**: `"gmp"` key, but only in a factory you register it into yourself. `init()` registers into the package-private `globalFactory` (`calculator_gmp.go`, its `globalFactory` var and `init`), which no caller reads; `app.New` builds its own via `NewDefaultFactory()` (`internal/app/app.go:New`, `registry.go:NewDefaultFactory` — `fast`/`matrix`/`fft`). `fibcalc -algo gmp` is therefore rejected even in a `-tags gmp` build.

## Research backends beyond GMP

FibCalc may cite **FLINT**, other LGPL/GPL C libraries, or experimental arbitrary-precision stacks as *research comparisons*. The current design constraints:

- **No additional mandatory C/C++ backend:** only `math/big` and optional GMP are integrated. Adding another requires a reproducible build matrix, license review, and golden-style equivalence tests on a bounded set of indices — see **ADR-010** in [ARCH.md](../ARCH.md).
- **Extension point:** new calculators register through `Register` on a factory built with `fibonacci.NewDefaultFactory()` (same pattern as `RegisterGMPCalculator` under the `gmp` build tag). Prototypes should live on a dedicated branch or fork until quality and legal criteria are met.
- **Equivalence:** when evaluating a candidate backend, compare against `"fast"` / `"gmp"` on shared `N` values and existing `testdata` where applicable.
