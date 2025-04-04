# Parallel Fibonacci Calculator in Go (v1.6)

[![Go Version](https://img.shields.io/badge/Go-1.18+-blue.svg)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Description

This Go program calculates Fibonacci numbers (F(n)) for a given list of potentially very large values of `n`. It is highly optimized for performance and memory usage, leveraging parallelism to compute multiple results concurrently.

It employs the efficient **matrix exponentiation** method (specifically, "fast doubling") with `math/big.Int` to handle arbitrarily large numbers. Further optimizations include aggressive object pooling (`sync.Pool`) and a configurable LRU cache to minimize memory allocations and redundant computations.

## Features

*   Calculates Fibonacci(n) for very large `n` using `math/big.Int`.
*   Processes a **list** of `n` values defined in the configuration.
*   **Parallel Calculation:** Uses a worker pool model to compute multiple F(n) values concurrently across available CPU cores (configurable number of workers).
*   **Efficient Algorithm:** Implements the matrix exponentiation (Fast Doubling) method, with a time complexity roughly O(log n) large integer operations.
*   **Memory Optimization:** Utilizes `sync.Pool` extensively to reuse `*big.Int` objects and internal matrix structures, significantly reducing garbage collector pressure.
*   **Caching:** Integrates a thread-safe LRU (Least Recently Used) cache (`github.com/hashicorp/golang-lru/v2`) to store and quickly retrieve previously computed results (configurable and enabled by default).
*   **Timeout & Cancellation:** Supports a global timeout for the entire batch calculation via `context.Context`. Workers check for cancellation periodically.
*   **Detailed Metrics:** Collects and displays aggregated performance metrics upon completion:
    *   Total execution time.
    *   Task completion status (successful, failed).
    *   Total matrix multiplication operations.
    *   Total cache hits.
    *   Estimated `*big.Int` allocations avoided via pooling.
    *   Current LRU cache size.
*   **Profiling Support:** Optional integration with Go's built-in `pprof` tool for CPU and memory heap profiling.
*   **Configurable:** Easily modify calculation parameters (list of N, timeout, workers, cache settings, profiling) within the `main.go` source file.
*   **Clear Output:** Displays results for each `n`, including the number of digits and handling very large numbers by showing the beginning and end digits, along with scientific notation.

## Requirements

*   Go 1.18 or later (due to the use of generics in the LRU cache dependency).

## Dependencies

This project relies on the following external Go module:

*   `github.com/hashicorp/golang-lru/v2`: For the thread-safe LRU cache implementation.

## Installation & Setup

1.  **Clone the repository:**
    ```bash
    git clone <repository-url>
    cd <repository-directory>
    ```
2.  **Download Dependencies:**
    ```bash
    go mod tidy
    # or explicitly: go get github.com/hashicorp/golang-lru/v2
    ```

## Usage

You can run the program directly using `go run`:

```bash
go run ./main.go

go build -o fibonacci_calculator .
./fibonacci_calculator```

The program will start the calculation based on the parameters defined in `DefaultConfig()` within `main.go` and print logs, metrics, and results to the console.

## Configuration

All configuration parameters are set within the `DefaultConfig()` function inside the `main.go` file. Modify this function to change the program's behavior:

*   `NsToCalculate []int`: The list of `n` values for which to calculate F(n).
*   `Timeout time.Duration`: The maximum total duration allowed for calculating the entire list.
*   `Precision int`: The number of significant digits for the scientific notation output.
*   `Workers int`: The number of worker goroutines to use for parallel calculation (defaults to the number of CPU cores). Also sets `GOMAXPROCS`.
*   `EnableCache bool`: Set to `true` to use the LRU cache, `false` to disable it.
*   `CacheSize int`: The maximum number of F(n) results to store in the LRU cache when enabled.
*   `EnableProfiling bool`: Set to `true` to enable CPU and Memory profiling, `false` to disable.

## Profiling

If `EnableProfiling` is set to `true` in the configuration:

1.  The program will generate `cpu.pprof` and `mem.pprof` files in the current directory upon completion (or potentially partial files on timeout/error).
2.  You can analyze these profiles using the Go tool `pprof`:
    ```bash
    # For CPU profiling (e.g., view top functions, generate graph)
    go tool pprof cpu.pprof
    # (pprof) top
    # (pprof) web

    # For Memory profiling (e.g., view heap usage)
    go tool pprof mem.pprof
    # (pprof) top
    # (pprof) web
    ```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details (assuming an MIT license).