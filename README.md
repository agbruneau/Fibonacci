# High-Performance Fibonacci Sequence Calculator

![Go version](https://img.shields.io/badge/Go-1.25+-blue.svg)
![License](https://img.shields.io/badge/License-MIT-green.svg)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)
![Coverage Status](https://img.shields.io/badge/coverage-69.7%25-brightgreen)

## 1. Purpose

This project is a high-performance Fibonacci calculator and a case study in advanced software engineering with Go. It's designed to explore and implement efficient algorithms for handling very large integers, applying low-level optimizations and high-level design patterns to maximize performance.

The primary goals are:
- To serve as a reference for implementing sophisticated algorithms in Go.
- To demonstrate best practices in software architecture, including modularity and testability.
- To provide a hands-on example of performance optimization techniques.

## 2. Getting Started

Follow these steps to get the Fibonacci calculator up and running on your local machine.

### Prerequisites

- Go 1.25 or later

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/your-username/fibcalc.git
   cd fibcalc
   ```

2. Build the executable:
   ```bash
   go build -o fibcalc ./cmd/fibcalc
   ```
   This will create a `fibcalc` binary (or `fibcalc.exe` on Windows) in the project root.

### Quick Start

- **Calibrate for your machine:**
  ```bash
  ./fibcalc --calibrate
  ```
  This will determine the optimal performance settings for your system.

- **Run a comparison of all algorithms:**
  ```bash
  ./fibcalc -n 10000000 -algo all
  ```

## 3. Features

*   **Large Number Support**: Utilizes `math/big` for arbitrary-precision arithmetic.
*   **Multiple Algorithms**: Implements several O(log n) algorithms:
    *   **Fast Doubling** (`fast`)
    *   **Matrix Exponentiation** (`matrix`)
    *   **FFT-based Fast Doubling** (`fft`)
*   **Performance Optimizations**:
    *   **Zero-Allocation Strategy**: Employs `sync.Pool` to minimize garbage collector overhead.
    *   **Parallelism**: Leverages multiple cores for enhanced performance.
    *   **Adaptive FFT Multiplication**: Switches to FFT-based multiplication for very large numbers.
*   **Modular Architecture**:
    *   **Separation of Concerns**: Decouples logic, presentation, and orchestration.
    *   **Clean Shutdown**: Manages application lifecycle with `context`.
    *   **Structured Concurrency**: Uses `golang.org/x/sync/errgroup` for orchestration.
*   **User-Friendly CLI**:
    *   Spinner and progress bar for visual feedback.
    *   Modes for comparison, calibration, and detailed results.
    *   Robust configuration and validation.

## 4. Usage

The calculator is controlled via command-line flags:

```bash
./fibcalc [options]
```

### Options

| Flag                 | Alias       | Description                                                 | Default      |
| -------------------- | ----------- | ----------------------------------------------------------- | ------------ |
| `-n`                 |             | Index of the Fibonacci number to calculate.                 | `250000000`  |
| `-algo`              |             | Algorithm to use: `fast`, `matrix`, `fft`, or `all`.        | `all`        |
| `-timeout`           |             | Maximum execution time (e.g., `10s`, `1m30s`).              | `5m`         |
| `-threshold`         |             | Bit threshold for parallelizing multiplications.            | `4096`       |
| `-fft-threshold`     |             | Bit threshold for enabling FFT multiplication.              | `20000`      |
| `--strassen-threshold` |           | Bit threshold for switching to Strassen's algorithm.        | `256`        |
| `-d`                 | `--details` | Display performance details and metadata.                   | `false`      |
| `-v`                 |             | Display the full result (can be very long).                 | `false`      |
| `--calibrate`        |             | Calibrate the optimal parallelism threshold.                | `false`      |
| `--auto-calibrate`   |             | Run a quick calibration at startup.                         | `false`      |
| `--lang`             |             | Language for i18n (e.g., `fr`, `en`).                       | `en`         |
| `--i18n-dir`         |             | Directory for translation files (e.g., `./locales`).        | `""`         |

### Examples

- **Calibrate for your machine:**
  ```bash
  ./fibcalc --calibrate
  ```

- **Compare algorithms for F(10,000,000):**
  ```bash
  ./fibcalc -n 10000000 -algo all -d
  ```

- **Calculate F(250,000,000) with a 10-minute timeout:**
  ```bash
  ./fibcalc -n 250000000 -algo fast -d --timeout 10m
  ```

## 5. Software Architecture

The project follows a modular design with a clear separation of concerns:

*   `cmd/fibcalc`: The application's entry point, responsible for parsing arguments and orchestrating the calculation.
*   `internal/config`: Handles CLI flags and configuration validation.
*   `internal/fibonacci`: Contains the core Fibonacci algorithms and optimizations.
*   `internal/cli`: Manages the presentation layer, including the spinner, progress bar, and result display.

## 6. Testing

The project includes a comprehensive test suite to ensure correctness and stability.

- **Run all tests:**
  ```bash
  go test ./... -v
  ```

- **Run benchmarks:**
  ```bash
  go test -bench . ./...
  ```

## 7. License

This project is licensed under the MIT License. See the `LICENSE` file for details.
