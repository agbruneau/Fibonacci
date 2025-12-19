# GMP-Based Calculator

## Overview

The GMP-based calculator utilizes the [GNU Multiple Precision Arithmetic Library (GMP)](https://gmplib.org/) to perform Fibonacci calculations. GMP is widely regarded as the fastest library for arbitrary-precision arithmetic, often outperforming Go's standard `math/big` library for extremely large numbers (e.g., >100 million bits).

This implementation uses the **Fast Doubling** algorithm, identical to the standard `fast` strategy, but delegates all arithmetic operations (addition, subtraction, multiplication, squaring) to GMP's highly optimized C assembly routines.

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

## Compilation

Because this implementation relies on CGO and an external C library, it is hidden behind a build tag (`gmp`) to prevent build failures on systems without GMP.

To build `fibcalc` with GMP support:

```bash
go build -tags gmp -o fibcalc ./cmd/fibcalc
```

## Usage

Once compiled with the `gmp` tag, a new algorithm option `gmp` becomes available.

```bash
./fibcalc -algo gmp -n 1000000
```

## Performance

GMP excels at extremely high precision. For inputs $N < 1,000,000$, Go's native `math/big` (and especially the optimized `bigfft` implementation used in the `fast` strategy) is often competitive or even faster due to CGO overhead. However, for $N > 100,000,000$, GMP's hand-tuned assembly loops typically provide a significant speed advantage.

## Implementation Details

*   **Algorithm:** Fast Doubling (iterative, MSB-to-LSB).
*   **Arithmetic:** Uses `github.com/ncw/gmp` bindings to call `libgmp`.
*   **Memory Management:** Reuses `gmp.Int` instances to minimize allocation overhead, similar to the `math/big` implementation.
