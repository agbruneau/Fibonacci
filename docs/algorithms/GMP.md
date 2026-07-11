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

> **Status (2026-06-10)**: the figures in this section (CGO overhead per call, crossover around N = 1,000,000, net advantage for N > 100,000,000) are unverified approximations to date — no dated GMP benchmark exists in `docs/audits/`. A measurement was attempted on 2026-06-10 under WSL but could not run: the `libgmp-dev` headers are not installed there (`gmp.h: No such file or directory`). Once installed (`sudo apt-get install libgmp-dev`), run `go test -tags=gmp -bench='Benchmark(Fibonacci|GMPCalculator)' -benchmem -run='^$' ./internal/fibonacci/` and archive the output in `docs/audits/` to replace these estimates.

GMP excels at extremely high precision. For inputs N < 1,000,000, Go's native `math/big` (and especially the optimized `bigfft` implementation used in the `"fast"` strategy) is often competitive or even faster due to CGO overhead. However, for N > 100,000,000, GMP's hand-tuned assembly loops typically provide a significant speed advantage.

### CGO Overhead

Each call to a GMP function incurs CGO overhead (typically 50-100ns per call). For small numbers, this overhead dominates the actual computation time, making native Go faster. The crossover point where GMP becomes faster depends on the specific hardware and operation, but is generally around N = 1,000,000.

## Implementation Details

- **Algorithm**: Fast Doubling (iterative, MSB-to-LSB)
- **Arithmetic**: Uses `github.com/ncw/gmp` bindings to call `libgmp`
- **Memory Management**: Reuses `gmp.Int` instances to minimize allocation overhead
- **File**: `internal/fibonacci/calculator_gmp.go`
- **Name()**: Returns `"GMP (Fast Doubling)"`
- **Registration**: `"gmp"` key in the calculator factory

## Research backends beyond GMP

FibCalc may cite **FLINT**, other LGPL/GPL C libraries, or experimental arbitrary-precision stacks as *research comparisons*. The current design constraints:

- **No additional mandatory C/C++ backend:** only `math/big` and optional GMP are integrated. Adding another requires a reproducible build matrix, license review, and golden-style equivalence tests on a bounded set of indices — see **ADR-010** in [ARCH.md](../ARCH.md).
- **Extension point:** new calculators register through `Register` on a factory built with `fibonacci.NewDefaultFactory()` (same pattern as `RegisterGMPCalculator` under the `gmp` build tag). Prototypes should live on a dedicated branch or fork until quality and legal criteria are met.
- **Equivalence:** when evaluating a candidate backend, compare against `"fast"` / `"gmp"` on shared `N` values and existing `testdata` where applicable.
