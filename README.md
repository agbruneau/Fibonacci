# High-Performance Fibonacci Sequence Calculator

![Go version](https://img.shields.io/badge/Go-1.18+-blue.svg)
![License](https://img.shields.io/badge/License-MIT-green.svg)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)

## 1. Summary

This project is not just a simple calculator for the Fibonacci sequence; it is a **case study** and a reference implementation demonstrating advanced software engineering techniques in the Go language. The main objective is to explore and implement calculation algorithms for very large integers, while applying low-level optimizations and high-level design patterns to achieve maximum performance.

The code is fully commented in English, with an academic perspective, to serve as a pedagogical support.

## 2. Main Features

*   **Calculation on Very Large Numbers**: Use of `math/big` for arbitrary arithmetic precision.
*   **Multiple Algorithms**: Implementation of several logarithmic complexity algorithms:
    *   **Fast Doubling** (`fast`)
    *   **Matrix Exponentiation** (`matrix`)
    *   **FFT-Based Doubling** (`fft`)
*   **Advanced Performance Optimizations**:
    *   **"Zero-Allocation" Strategy**: Intensive use of `sync.Pool` for reusing objects (`big.Int`, calculation states), minimizing pressure on the Garbage Collector.
    *   **Task Parallelism**: Exploitation of multi-core processors to parallelize integer multiplications beyond a configurable threshold.
    *   **FFT Multiplication**: Adaptive use of the Fast Fourier Transform for multiplying numbers exceeding a threshold of several tens of thousands of bits.
    *   **Lookup Table (LUT)**: O(1) resolution for small Fibonacci numbers via a pre-calculated table.
*   **Modular and Robust Architecture**:
    *   **Separation of Concerns (SoC)**: Strict decoupling between business logic, user interface, and orchestration.
    *   **Lifecycle Management**: Advanced use of `context` for clean management of timeouts and interruption signals (graceful shutdown).
    *   **Structured Concurrency**: Orchestration of parallel calculations with `golang.org/x/sync/errgroup`.
*   **Rich Command-Line Interface (CLI)**:
    *   Dynamic and non-blocking progress bar.
    *   Comparison, calibration, and detailed display modes.
    *   Robust configuration validation.

## 3. Design Principles and Patterns

This project serves as a practical demonstration for several fundamental design principles and patterns:

*   **SOLID**:
    *   **Single Responsibility Principle**: Each module (`cmd/fibcalc`, `internal/fibonacci`, `internal/cli`) has a unique and well-defined responsibility.
    *   **Open/Closed Principle**: The `calculatorRegistry` allows adding new algorithms without modifying the existing orchestration code.
    *   **Dependency Inversion Principle**: High-level modules depend on abstractions (`Calculator`) rather than concrete implementations.
    *   **Interface Segregation Principle**: The separation between `Calculator` (public interface) and `coreCalculator` (internal interface) avoids overloading implementations with unnecessary dependencies.
*   **Decorator Pattern**: The `FibCalculator` structure encapsulates a `coreCalculator` to transparently add cross-cutting features (like LUT optimization).
*   **Adapter Pattern**: `FibCalculator` also adapts the channel-based communication interface (`chan`) into a simpler callback interface (`ProgressReporter`) for the algorithms.
*   **Producer/Consumer Pattern**: The algorithms (Producers) generate progress updates that are processed asynchronously by the UI (Consumer) via Go channels.
*   **Registry Pattern**: The `calculatorRegistry` centralizes the available algorithm implementations, promoting loose coupling.
*   **Object Pooling**: The use of `sync.Pool` to manage calculation states (`calculationState`, `matrixState`) is a crucial memory optimization to achieve "zero-allocation".

## 4. Software Architecture

The project is structured into three main modules:

*   `cmd/fibcalc`: **The Composition Root**. Entry point of the application, responsible for parsing arguments, configuration, dependency injection, and lifecycle orchestration.
*   `internal/fibonacci`: **The Business Domain**. Contains all the calculation logic, algorithm implementations, and low-level optimizations.
*   `internal/cli`: **The Presentation Layer**. Manages all interactions with the user (progress bar, result display).

## 5. Installation and Compilation

The project uses standard Go modules. To compile the executable:

```bash
go build -o fibcalc ./cmd/fibcalc
```

A binary named `fibcalc` (or `fibcalc.exe` on Windows) will be created in the current directory.

## 6. User Guide and Performance Optimization

The executable is used as follows:

```bash
./fibcalc [options]
```

### Command-Line Options

| Flag             | Alias       | Description                                                              | Default      |
| ---------------- | ----------- | ------------------------------------------------------------------------ | ----------- |
| `-n`             |             | The index 'n' of the Fibonacci sequence to calculate.                        | `100000000` |
| `-algo`          |             | Algorithm: `fast`, `matrix`, `fft`, or `all` to compare.            | `all`       |
| `-timeout`       |             | Maximum execution time (e.g., `10s`, `1m30s`).                          | `5m0s`      |
| `-threshold`     |             | Threshold (in bits) to parallelize multiplications.                   | `4096`      |
| `-fft-threshold` |             | Threshold (in bits) to use FFT multiplication (0=disabled).        | `20000`     |
| `-d`             | `--details` | Display performance details and result metadata.      | `false`     |
| `-v`             | `--verbose` | Display the full value of the result (can be extremely long).    | `false`     |
| `--calibrate`    |             | Run calibration to find the optimal parallelism threshold.     | `false`     |

### Performance Optimization

To achieve the best possible performance, a methodical approach is recommended:

#### Step 1: Calibration of the Parallelism Threshold

The performance of calculations on very large numbers strongly depends on your processor's architecture. This project includes a calibration mode to empirically determine the best parallelism threshold (`--threshold`) for your machine.

Run the following command:
```bash
./fibcalc --calibrate
```
The program will test several threshold values and provide a recommendation, for example: `✅ Recommendation for this machine: --threshold 4096`.

#### Step 2: Using Optimal Parameters

Once the optimal threshold is determined, use it for your calculations.

*   `--threshold`: The parallelism threshold. A well-adjusted value (via `--calibrate`) is crucial for calculations on multi-core machines.
*   `--fft-threshold`: This threshold enables FFT multiplication, which is faster for numbers exceeding several tens of thousands of bits. The default value of `20000` is a good starting point for most modern architectures.

#### Step 3: Algorithm Comparison

The program provides two state-of-the-art algorithms. Their performance may vary. Use the comparison mode to identify the fastest for your use case.

```bash
./fibcalc -n <a_large_number> -algo all --threshold <calibrated_value>
```
The program will run both algorithms in parallel and display a comparative table. It also performs a **cross-validation**: if the calculations succeed, it verifies that their results are identical, ensuring accuracy.

### Usage Examples

**1. Find the optimal performance parameter for your machine:**
```bash
./fibcalc --calibrate
```

**2. Compare algorithms for F(10,000,000) using a calibrated parallelism threshold of 4096:**
```bash
./fibcalc -n 10000000 -algo all --threshold 4096 -d
```

**3. Calculate F(250,000,000) with the fastest algorithm, detailed display, and a 10-minute timeout:**
```bash
# After determining that 'fast' is the fastest in Step 2
./fibcalc -n 250000000 -algo fast --threshold 4096 -d --timeout 10m
```

## 7. Validation and Tests

The project has a comprehensive test suite to ensure its accuracy and robustness.

*   **Unit and Integration Tests**:
    ```bash
    go test ./... -v
    ```
    This command runs all project tests, including validation of parsing logic, algorithm edge cases, and UI behavior.

*   **Performance Tests (Benchmarks)**:
    ```bash
    go test -bench . ./...
    ```
    This command runs benchmarks to measure the latency and memory allocations of the algorithms.

*   **Property-Based Testing**:
    The project uses property-based tests (with the `gopter` library) to validate mathematical invariants, such as **Cassini's Identity**, providing a higher level of confidence in the accuracy of the algorithms.

## 8. License

This project is distributed under the MIT license. See the `LICENSE` file for more details.