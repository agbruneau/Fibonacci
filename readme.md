# Concurrent Fibonacci Calculator in Go

This project is a command-line tool written in Go to calculate the n-th Fibonacci number. Its key feature is the implementation of several distinct algorithms, executing them in parallel to compare their performance, memory consumption, and validate their results.

The application is designed to be both a practical tool and a demonstration of several advanced Go concepts, such as concurrency, context management, memory optimization, and performance testing (benchmarking).

✨ Features

*   **Very Large Number Calculation**: Uses the `math/big` package to calculate Fibonacci numbers far beyond the limits of standard integer types.
*   **Concurrent Execution**: Launches multiple algorithms in parallel using goroutines for direct performance comparison.
*   **Multi-Algorithm**: Implements four different calculation methods with distinct performance characteristics:
    *   Fast Doubling
    *   Matrix Exponentiation
    *   Binet's Formula
    *   Iterative Method
*   **Progress Display**: Shows real-time progress of each algorithm on a single, updating line.
*   **Timeout Management**: Uses `context.WithTimeout` to ensure the program terminates cleanly if calculations take too long.
*   **Memory Optimization**: Employs a `sync.Pool` to recycle `*big.Int` objects, reducing pressure on the Garbage Collector.
*   **Comprehensive Test Suite**: Includes unit tests to validate algorithm correctness and benchmarks to formally measure their performance.
*   **Selective Algorithm Execution**: Allows users to specify which algorithms to run via a command-line flag.

🛠️ Prerequisites

*   Go (version 1.18+ recommended, as the project uses `go fmt ./...` and some newer practices. Core logic might work on 1.16+ but 1.18+ is advised for full compatibility with development environment tooling.)
*   Git (to clone the project)

🚀 Installation

1.  Clone the repository to your local machine:
    ```sh
    git clone https://github.com/your-username/your-repo.git
    ```
    (Replace the URL with your repository's actual URL)

2.  Navigate to the project directory:
    ```sh
    cd your-repo
    ```

💻 Usage

The tool is run directly from the command line.

**Simple Execution**

To run the calculation with default values (n=100000, timeout=1m, all algorithms):
```sh
go run .
```

**Command-Line Options**

You can customize the execution with the following options:

*   `-n <number>`: Specifies the index `n` of the Fibonacci number to calculate (non-negative integer). Default: `100000`.
*   `-timeout <duration>`: Specifies the global timeout for the execution (e.g., `30s`, `2m`, `1h`). Default: `1m`.
*   `-algorithms <list>`: Comma-separated list of algorithms to run. Available: `Fast Doubling`, `Matrix 2x2`, `Binet`, `Iterative`. Use `all` to run all. Names are case-insensitive (e.g., "fast doubling", "matrix 2x2", "binet", "iterative"). Default: `all`.

**Examples**

Calculate F(500,000) with a 30-second timeout, running only Fast Doubling and Iterative algorithms:
```sh
go run . -n 500000 -timeout 30s -algorithms "fast doubling,iterative"
```

Calculate F(1,000,000) with a 5-minute timeout, running all algorithms:
```sh
go run . -n 1000000 -timeout 5m -algorithms all
```
Or simply (as `all` is default for algorithms):
```sh
go run . -n 1000000 -timeout 5m
```

**Example Output**
```
2023/10/27 10:30:00 Calculating F(200000) with a timeout of 1m...
2023/10/27 10:30:00 Algorithms to run: Fast Doubling, Matrix 2x2, Binet, Iterative
2023/10/27 10:30:00 Launching concurrent calculations...
Fast Doubling:   100.00%   Matrix 2x2:      100.00%   Binet:           100.00%   Iterative:       100.00%
2023/10/27 10:30:01 Calculations finished.

--------------------------- ORDERED RESULTS ---------------------------
Fast Doubling    : 8.8475ms     [OK              ] Result: 25974...03125
Iterative        : 12.5032ms    [OK              ] Result: 25974...03125
Matrix 2x2       : 18.0673ms    [OK              ] Result: 25974...03125
Binet            : 43.1258ms    [OK              ] Result: 25974...03125
------------------------------------------------------------------------

🏆 Fastest algorithm (that succeeded): Fast Doubling (8.848ms)
Number of digits in F(200000): 41798
Value (scientific notation) ≈ 2.59740692e+41797
✅ All valid results produced are identical.
2023/10/27 10:30:01 Program finished.
```

🧠 Implemented Algorithms

1.  **Fast Doubling**
    One of the fastest known algorithms for large integers. It uses the identities:
    *   `F(2k) = F(k) * [2*F(k+1) – F(k)]`
    *   `F(2k+1) = F(k)² + F(k+1)²`
    to significantly reduce the number of operations. Complexity: O(log n) arithmetic operations.

2.  **Matrix Exponentiation (2x2)**
    A classic approach based on the property that raising the matrix `Q = [[1,1],[1,0]]` to the power `k` yields:
    ```
    Q^k  =  | F(k+1)  F(k)   |
           | F(k)    F(k-1) |
    ```
    The calculation of Q^(n-1) (to get F(n) as the top-left element) is optimized using exponentiation by squaring. Complexity: O(log n) matrix multiplications.

3.  **Binet's Formula**
    An analytical solution using the golden ratio (φ):
    `F(n) = (φ^n - ψ^n) / √5`, where `φ = (1+√5)/2` and `ψ = (1-√5)/2`.
    It's calculated using high-precision floating-point numbers (`big.Float`). While elegant, it's generally less performant for direct computation and can suffer from precision errors for very large `n`.

4.  **Iterative Method**
    Calculates Fibonacci numbers by iterating from F(0)=0 and F(1)=1 up to F(n) using the fundamental definition `F(k) = F(k-1) + F(k-2)`.
    This method is simple to understand and very memory-efficient (especially when `sync.Pool` is used for `big.Int` objects). However, with O(n) arithmetic operations (each on potentially large numbers), it is significantly slower for large `n` compared to the logarithmic methods.

🏗️ Code Architecture

The codebase is organized into several Go files for better modularity:

*   `main.go`: Contains the main application logic, including command-line flag parsing, orchestration of concurrent algorithm execution via goroutines, and the final display of results.
*   `algorithms.go`: Houses the implementations of the different Fibonacci calculation algorithms. This includes the `fibFunc` type definition and its concrete implementations (e.g., `fibFastDoubling`, `fibMatrix`, `fibBinet`, `fibIterative`).
*   `utils.go`: Provides utility functions shared across the application. Key components are the `progressPrinter` for real-time progress display and the `newIntPool` helper for managing the `sync.Pool` of `*big.Int` objects.
*   `main_test.go`: Contains a comprehensive suite of unit tests to verify the correctness of each algorithm and benchmarks to measure their performance characteristics (execution time and memory allocations).

Concurrency is managed using a `sync.WaitGroup` to ensure all calculation goroutines complete before the program proceeds to aggregate results. Progress updates from each concurrent task are sent over a shared channel (`progressAggregatorCh`) to the `progressPrinter` goroutine, which consolidates and displays them on a single line in the console.

✅ Tests

The project includes a comprehensive suite of tests to ensure correctness and measure performance.

**Run Unit Tests**

To verify that all implemented algorithms produce correct Fibonacci numbers for a set of known values (including edge cases):
```sh
go test -v ./...
```
This command runs all tests in the current package and any sub-packages.

**Run Benchmarks**

To measure and compare the performance (execution time and memory allocations) of each algorithm:
```sh
go test -bench . ./...
```
This command runs all benchmarks in the current package and sub-packages. The `.` indicates all benchmarks.
To run benchmarks for a specific algorithm or a group, you can use the `-bench` flag with a regular expression. For example, to benchmark only the Iterative method:
```sh
go test -bench=BenchmarkFibIterative ./...
```
Or to benchmark all Fibonacci algorithms:
```sh
go test -bench=Fib ./...
```

📜 License

This project is distributed under the MIT License. (Typically, a `LICENSE` file would be included in the repository with the full text of the MIT License.)