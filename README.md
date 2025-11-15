# High-Performance Fibonacci Sequence Calculator

![Go version](https://img.shields.io/badge/Go-1.25+-blue.svg)
![License](https://img.shields.io/badge/License-MIT-green.svg)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)
![Coverage Status](https://img.shields.io/badge/coverage-69.7%25-brightgreen)

## 1. Summary

This project is not just a simple Fibonacci calculator; it is a **case study** and a reference implementation demonstrating advanced software engineering techniques in Go. The main objective is to explore and implement calculation algorithms for very large integers, while applying low-level optimizations and high-level design patterns to achieve maximum performance.

## 2. Key Features

*   **Very Large Integer Calculation**: `math/big` for arbitrary precision.
*   **Multiple Algorithms** (logarithmic complexity):
    *   **Fast Doubling** (`fast`)
    *   **Matrix Exponentiation** (`matrix`)
    *   **FFT-based Fast Doubling** (`fft`)
*   **Advanced Performance Optimizations**:
    *   **"Zero Allocation" Strategy**: intensive use of `sync.Pool` (object reuse), minimized GC pressure.
    *   **Task Parallelism**: multi-core exploitation beyond a configurable threshold.
    *   **FFT Multiplication**: adaptive activation (configurable bit threshold).
*   **Modular and Robust Architecture**:
    *   **Separation of Concerns (SoC)**: strict decoupling of logic/presentation/orchestration.
    *   **Lifecycle Management**: `context` for deadlines and signals (clean shutdown).
    *   **Structured Concurrency**: orchestration via `golang.org/x/sync/errgroup`.
*   **Rich CLI Interface**:
    *   Animation (spinner) and progress bar.
    *   Comparison, calibration, detailed display modes.
    *   Robust configuration validation.

## 3. Design Principles and Patterns

This project concretely illustrates several design principles and patterns:

*   **SOLID**:
    *   **Single Responsibility**: each module (`cmd/fibcalc`, `internal/fibonacci`, `internal/config`, `internal/cli`) has a clear role.
    *   **Open/Closed**: `calculatorRegistry` allows adding algorithms without modifying the orchestration.
    *   **Dependency Inversion**: dependency on the `Calculator` interface.
    *   **Interface Segregation**: `Calculator` (public) vs `coreCalculator` (internal).
*   **Decorator**: `FibCalculator` wraps a `coreCalculator` to add cross-cutting concerns (LUT).
*   **Adapter**: adaptation of the UI channel into a `ProgressReporter` callback for the algorithms.
*   **Producer/Consumer**: asynchronous sending of progress via channels.
*   **Registry**: centralization of available implementations.
*   **Object Pooling**: `sync.Pool` for calculation states to approach "zero allocation".

## 4. Software Architecture

The project is structured into four main modules:

*   `cmd/fibcalc`: **composition root**. Entry point: argument parsing, dependency injection, orchestration.
*   `internal/config`: **configuration layer**. CLI flags and validation.
*   `internal/fibonacci`: **business domain**. Algorithms and optimizations.
*   `internal/cli`: **presentation layer**. Display, progress, result rendering.

## 5. Installation and Compilation

The project uses Go modules. To compile the executable:

```bash
go build -o fibcalc ./cmd/fibcalc
```

A `fibcalc` binary (or `fibcalc.exe` on Windows) will be created in the current directory.

## 6. Usage Guide and Performance Optimization

Using the executable:

```bash
./fibcalc [options]
```

### Command-Line Options

| Flag             | Alias       | Description                                                              | Default      |
| ---------------- | ----------- | ------------------------------------------------------------------------ | ----------- |
| `-n`             |             | Index `n` of the Fibonacci number to calculate.                         | `250000000` |
| `-algo`          |             | Algorithm: `fast`, `matrix`, `fft`, or `all` to compare.        | `all`       |
| `-timeout`       |             | Maximum execution time (e.g., `10s`, `1m30s`).                      | `5m0s`      |
| `-threshold`     |             | Bit threshold for parallelizing multiplications.               | `4096`      |
| `-fft-threshold` |             | Bit threshold to enable FFT multiplication (0 to disable). | `20000`     |
| `--strassen-threshold` |      | Bit threshold to switch to Strassen in matrix multiplication. | `256`       |
| `-d`             | `--details` | Display performance details and result metadata.   | `false`     |
| `-v`             |             | Display the full value of the result (very long).                  | `false`     |
| `--calibrate`    |             | Run calibration of the optimal parallelism threshold. | `false`     |
| `--auto-calibrate` |           | Quick calibration at startup to fine-tune `threshold` and `fft-threshold`. | `false`      |
| `--lang`         |             | i18n language code (e.g., `fr`, `en`).                                   | `en`        |
| `--i18n-dir`     |             | Directory containing `<lang>.json` to override messages.     | `""`       |

### Performance Optimization

To achieve the best performance, follow a methodical approach:

#### Step 1: Calibrate the Parallelism Threshold

Performance on very large numbers heavily depends on the processor architecture. The project includes a calibration mode to empirically determine the best parallelism threshold (`--threshold`) for your machine.

Run the following command:
```bash
./fibcalc --calibrate
```
The program tests a predefined list of threshold values to find the one that yields the best performance, for example: `✅ Recommendation for this machine: --threshold 4096`.

#### Step 2: Use Optimal Parameters

Once the optimal threshold is determined, use it in your calculations.

*   `--threshold`: parallelism threshold (calibrated), crucial on multi-core machines.
*   `--fft-threshold`: activation threshold for FFT multiplication, effective for huge numbers (millions of bits).
*   `--strassen-threshold`: threshold for switching to Strassen in the matrix algorithm (default 256, configurable).
*   `--auto-calibrate`: can be enabled for an opportunistic adjustment at startup, though it is disabled by default.

#### Step 3: Compare Algorithms

The program offers three cutting-edge algorithms. Their performance may vary. Use the comparison mode to identify the fastest for your use case.

```bash
./fibcalc -n <a_large_number> -algo all --threshold <calibrated_value>
```
The program runs all algorithms in parallel and displays a comparative table. It also performs a **cross-validation**: in case of success, it checks the equality of the results to ensure accuracy.

### Usage Examples

**1. Find the optimal performance parameter for your machine:**
```bash
./fibcalc --calibrate
```

**2. Compare algorithms for F(10,000,000) with a parallelism threshold calibrated to 4096:**
```bash
./fibcalc -n 10000000 -algo all --threshold 4096 -d
```

**3. Calculate F(250,000,000) with the fastest algorithm, detailed display, and a 10-minute timeout:**
```bash
# After determining in step 2 that `fast` is the fastest
./fibcalc -n 250000000 -algo fast --threshold 4096 -d --timeout 10m
```

**4. Adjust the Strassen threshold for the matrix algorithm:**
```bash
./fibcalc -n 10000000 -algo matrix --strassen-threshold 384 -d
```

**5. Enable dynamic i18n (JSON loading):**
```bash
./fibcalc --i18n-dir ./locales --lang en
# expects a ./locales/en.json file in the format { "Key": "Text" }
```

## 7. Validation and Testing

The project has a comprehensive test suite to ensure accuracy and robustness.

*   **Unit and Integration Tests**:
    ```bash
    go test ./... -v
    ```
    This command runs all project tests: configuration parsing, algorithm edge cases, UI behavior, etc.

*   **Performance Tests (Benchmarks)**:
    ```bash
    go test -bench . ./...
    ```
    This command runs benchmarks to measure latency and memory allocations of the algorithms.

*   **Property-Based Testing**:
    The project uses property-based tests (library `gopter`) to validate mathematical invariants, such as **Cassini's identity**, providing a high level of confidence.

## 8. License

This project is distributed under the MIT license. See the `LICENSE` file for more details.
