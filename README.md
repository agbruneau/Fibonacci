# High-Performance Fibonacci Sequence Calculator

<div align="center">

![Go version](https://img.shields.io/badge/Go-1.25+-blue.svg?style=for-the-badge&logo=go)
![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=for-the-badge)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg?style=for-the-badge)
![Coverage Status](https://img.shields.io/badge/coverage-75.2%25-brightgreen?style=for-the-badge)

</div>

---

## 📋 Table of Contents

- [High-Performance Fibonacci Sequence Calculator](#high-performance-fibonacci-sequence-calculator)
  - [📋 Table of Contents](#-table-of-contents)
  - [⚡ Quick Start](#-quick-start)
  - [🎥 Demo](#-demo)
  - [🚀 Performance vs Baseline](#-performance-vs-baseline)
  - [1. Objective](#1-objective)
  - [2. Getting Started](#2-getting-started)
    - [Prerequisites](#prerequisites)
    - [Installation Methods](#installation-methods)
      - [Option A: Quick Install (Recommended for users)](#option-a-quick-install-recommended-for-users)
      - [Option B: Build from Source (Recommended for developers)](#option-b-build-from-source-recommended-for-developers)
      - [Option C: Docker](#option-c-docker)
    - [Verification](#verification)
  - [3. Features](#3-features)
  - [4. Usage](#4-usage)
    - [Essential Commands](#essential-commands)
    - [Complete CLI Options](#complete-cli-options)
    - [Configuration via Environment Variables](#configuration-via-environment-variables)
    - [Interactive Mode (REPL)](#interactive-mode-repl)
    - [API Server Mode](#api-server-mode)
    - [💡 Usage Examples (Snippets)](#-usage-examples-snippets)
  - [5. Software Architecture](#5-software-architecture)
  - [6. Algorithms](#6-algorithms)
  - [7. Performance Optimisations](#7-performance-optimisations)
  - [8. Tests](#8-tests)
  - [9. Development](#9-development)
  - [10. Deployment](#10-deployment)
  - [11. Documentation](#11-documentation)
  - [12. Licence](#12-licence)

---

## ⚡ Quick Start

Leverage the power of modern Go without complex installation.

```bash
# 🚀 Run immediately (Requires Go installed)
go run ./cmd/fibcalc -n 100000 -algo fast

# 🛠️ Or compile for maximum performance
make build
./build/fibcalc -n 1000000
```

> **No Go?** Use Docker:
> `docker run --rm fibcalc -n 1000`

---

## 🎥 Demo

See `fibcalc` in action, calculating the 1,000,000th Fibonacci number in under 100ms.

```console
$ ./build/fibcalc -n 1000000 --algo fast
🚀 Calculating Fibonacci for n=1000000...
Algorithm: Fast Doubling (O(log n), Parallel)

✅ F(1000000) calculated in 85ms
   Bits:      694,242
   Digits:    208,988
   Value:     19532821287077577316320149475962563... (truncated)
```

---

## 🚀 Performance vs Baseline

Execution time comparison on a standard processor (Ryzen 9 5900X).
The **Fast Doubling** algorithm significantly outperforms the standard matrix approach for large numbers.

| N (Index)   | Fast Doubling | Matrix Exp. | Speedup  |
| ----------- | ------------- | ----------- | -------- |
| 1,000       | **15µs**      | 18µs        | 1.2x     |
| 10,000      | **180µs**     | 220µs       | 1.2x     |
| 100,000     | **3.2ms**     | 4.1ms       | 1.3x     |
| 1,000,000   | **85ms**      | 110ms       | 1.3x     |
| 10,000,000  | **2.1s**      | 2.8s        | 1.35x    |
| 100,000,000 | **45s**       | 62s         | **1.4x** |
| 250,000,000 | **3m12s**     | 4m25s       | **1.4x** |

> **Note:** A naive iterative implementation (O(n)) would take **years** to calculate F(100,000,000). Our logarithmic algorithms (O(log n)) do it in under a minute.

---

## 1. Objective

This project is a high-performance Fibonacci calculator and a case study in advanced software engineering with Go. It is designed to explore and implement efficient algorithms for handling very large integers, applying low-level optimisations and high-level design patterns to maximise performance.

The main objectives are:

- **Technical Reference**: Serve as a reference implementation for complex mathematical algorithms (Fast Doubling, Strassen, FFT).
- **Clean Architecture**: Demonstrate a modular, testable, and decoupled architecture (Clean Architecture).
- **Extreme Performance**: Illustrate advanced optimisation techniques such as memory recycling (`sync.Pool`), fine-grained concurrency, and hardware-adapted arithmetic.
- **Production-Ready**: Offer a robust CLI, an interactive REPL mode, and a REST API with graceful shutdown, monitoring, and dynamic configuration.

## 2. Getting Started

Follow these steps to set up the Fibonacci calculator on your local machine.

### Prerequisites

- **Go**: Version 1.25 or later ([Download Go](https://go.dev/dl/))
- **Git**: To clone the repository
- **Make** (Optional): For using the Makefile

### Installation Methods

#### Option A: Quick Install (Recommended for users)

If you have Go installed, you can install the binary directly to your `$GOPATH/bin`:

```bash
go install ./cmd/fibcalc
```

Ensure your `$GOPATH/bin` is in your system's `PATH`. You can then run the tool simply by typing `fibcalc`.

#### Option B: Build from Source (Recommended for developers)

1. **Clone the repository:**

   ```bash
   git clone https://github.com/agbru/fibcalc.git
   cd fibcalc
   ```

2. **Build the binary:**

   Using Make:
   ```bash
   make build
   ```

   Using Go directly:
   ```bash
   go build -o build/fibcalc ./cmd/fibcalc
   ```

   The executable will be located in the `build/` directory (or `build/fibcalc.exe` on Windows).

#### Option C: Docker

You can run `fibcalc` without installing Go using Docker:

```bash
docker build -t fibcalc .
docker run --rm fibcalc -n 1000
```

### Verification

Once the project is installed, it is recommended to verify that everything works correctly by running the test suite:

```bash
make test
# or if Make is not available:
go test ./...
```

This step will validate that your environment is correctly configured and that the code is functional on your architecture.

## 3. Features

- **Large Number Support**: Uses `math/big` for arbitrary-precision arithmetic, capable of calculating Fibonacci numbers with millions of digits.
- **Multiple Algorithms**:
  - **Fast Doubling (`fast`)**: The default algorithm. Combines logarithmic complexity, parallelism, and hybrid multiplication (Karatsuba/FFT).
  - **Matrix Exponentiation (`matrix`)**: Uses binary decomposition of the exponent and the Strassen algorithm for large matrices.
  - **FFT-Based Doubling (`fft`)**: Forces the use of FFT multiplication for all calculations.
  - **GMP-Based (`gmp`)**: An optional custom mode utilizing the GNU Multiple Precision Arithmetic Library (GMP) for extreme calculations (>100M bits). Requires compilation with `-tags gmp`.
- **Multiple Execution Modes**:
  - **CLI**: One-off calculations via command line.
  - **Interactive Mode (REPL)**: Interactive session for multiple calculations.
  - **HTTP Server Mode**: High-performance REST API for on-demand calculations.
  - **Docker**: Production-ready containerised deployment.
- **Flexible Output**:
  - JSON format (`--json`) for integration into pipelines.
  - Export to file (`-o, --output`).
  - Hexadecimal display (`--hex`).
  - Quiet mode (`-q, --quiet`) for scripts.
  - Display calculated value (`-c, --calculate`).
- **Performance Optimisations**:

  - **Zero-Allocation Strategy**: Uses `sync.Pool` to recycle `big.Int` objects.
  - **Modular Architecture**: Reusable frameworks and interchangeable multiplication strategies.
  - **Multi-level Parallelism**: Parallelisation at both algorithm and internal FFT levels.
  - **Strassen-Winograd Algorithm**: Optimized matrix multiplication reducing additions/subtractions by 17%.
  - **Global Memory Pooling**: Unified pooling strategies for `big.Int` across algorithms to minimize GC pressure.

  - **Robust Error Handling**: Panic-free architecture with explicit error propagation.
  - **Automatic Calibration**: Detection of optimal thresholds for the hardware.

- **Security**: Rate limiting, input validation, HTTP security headers, DoS protection.

## 4. Usage

The calculator is controlled via command-line flags:

```bash
./build/fibcalc [options]
```

### Essential Commands

| Command              | Description            |
| -------------------- | ---------------------- |
| `make build`         | Compile the project    |
| `make test`          | Run all tests          |
| `make run-fast`      | Quick test (n=1000)    |
| `make run-server`    | Start the HTTP server  |
| `make run-calibrate` | Calibrate performance  |
| `make coverage`      | HTML coverage report   |
| `make benchmark`     | Run benchmarks         |
| `make docker-build`  | Build the Docker image |
| `make clean`         | Clean build artefacts  |
| `make help`          | Display all commands   |

### Complete CLI Options

| Flag                    | Alias         | Description                                                      | Default                       |
| ----------------------- | ------------- | ---------------------------------------------------------------- | ----------------------------- |
| `-n`                    |               | Index of the Fibonacci number to calculate.                      | `250000000`                   |
| `-algo`                 |               | Algorithm: `fast`, `matrix`, `fft`, or `all`.                    | `all`                         |
| `-timeout`              |               | Maximum execution time (e.g., `10s`, `1m30s`).                   | `5m`                          |
| `-threshold`            |               | Bit threshold to parallelise multiplications.                    | `4096`                        |
| `-fft-threshold`        |               | Bit threshold to enable FFT multiplication.                      | `500000`                      |
| `--strassen-threshold`  |               | Bit threshold for the Strassen algorithm.                        | `3072`                        |
| `-d`                    | `--details`   | Display performance details.                                     | `false`                       |
| `-v`                    |               | Display the full result (can be very long).                      | `false`                       |
| `-c`                    | `--calculate` | Display the calculated value (disabled by default).              | `false`                       |
| `--calibrate`           |               | Calibrate the optimal parallelism threshold.                     | `false`                       |
| `--auto-calibrate`      |               | Quick calibration at startup.                                    | `false`                       |
| `--calibration-profile` |               | Path to the calibration profile file.                            | `~/.fibcalc_calibration.json` |
| `--json`                |               | Output in JSON format.                                           | `false`                       |
| `--server`              |               | Start in HTTP server mode.                                       | `false`                       |
| `--port`                |               | Listening port for server mode.                                  | `8080`                        |
| `--interactive`         |               | Start in interactive mode (REPL).                                | `false`                       |
| `-o`                    | `--output`    | Save the result to a file.                                       | `""`                          |
| `-q`                    | `--quiet`     | Quiet mode (minimal output).                                     | `false`                       |
| `--hex`                 |               | Display the result in hexadecimal.                               | `false`                       |
| `--no-color`            |               | Disable colours (also respects `NO_COLOR`).                      | `false`                       |
| `--completion`          |               | Generate an autocompletion script (bash, zsh, fish, powershell). | `""`                          |
| `--version`             | `-V`          | Display the program version.                                     |                               |

### Configuration via Environment Variables

In addition to CLI flags, `fibcalc` can be configured via environment variables. This is particularly useful for Docker and Kubernetes deployments, following [12-Factor App](https://12factor.net/config) best practices.

**Configuration Priority:** CLI Flags > Environment Variables > Default Values

| Variable                      | Type     | Description                        | Default     |
| ----------------------------- | -------- | ---------------------------------- | ----------- |
| `FIBCALC_N`                   | uint64   | Fibonacci number index             | `250000000` |
| `FIBCALC_ALGO`                | string   | Algorithm (fast, matrix, fft, all) | `all`       |
| `FIBCALC_PORT`                | string   | HTTP server port                   | `8080`      |
| `FIBCALC_TIMEOUT`             | duration | Timeout (e.g., "5m", "30s")        | `5m`        |
| `FIBCALC_THRESHOLD`           | int      | Parallelism threshold (bits)       | `4096`      |
| `FIBCALC_FFT_THRESHOLD`       | int      | FFT threshold (bits)               | `500000`    |
| `FIBCALC_STRASSEN_THRESHOLD`  | int      | Strassen threshold (bits)          | `3072`      |
| `FIBCALC_SERVER`              | bool     | Server mode (true/false)           | `false`     |
| `FIBCALC_JSON`                | bool     | JSON output                        | `false`     |
| `FIBCALC_VERBOSE`             | bool     | Verbose mode                       | `false`     |
| `FIBCALC_DETAILS`             | bool     | Display performance details        | `false`     |
| `FIBCALC_QUIET`               | bool     | Quiet mode                         | `false`     |
| `FIBCALC_HEX`                 | bool     | Hexadecimal output                 | `false`     |
| `FIBCALC_INTERACTIVE`         | bool     | REPL mode                          | `false`     |
| `FIBCALC_NO_COLOR`            | bool     | Disable colours                    | `false`     |
| `FIBCALC_OUTPUT`              | string   | Output file                        | `""`        |
| `FIBCALC_CALIBRATE`           | bool     | Run calibration mode               | `false`     |
| `FIBCALC_AUTO_CALIBRATE`      | bool     | Run auto-calibration at startup    | `false`     |
| `FIBCALC_CALIBRATION_PROFILE` | string   | Calibration file                   | `""`        |
| `FIBCALC_CALCULATE`           | bool     | Display calculated value           | `false`     |

**Examples:**

```bash
# Simple calculation via environment variable
FIBCALC_N=1000 FIBCALC_ALGO=fast ./build/fibcalc

# Server with environment configuration
export FIBCALC_SERVER=true
export FIBCALC_PORT=9090
export FIBCALC_THRESHOLD=8192
./build/fibcalc

# CLI flags always take priority
FIBCALC_N=99999 ./build/fibcalc -n 100  # Will use n=100
```

**Docker Compose:**

```yaml
services:
  fibcalc:
    image: fibcalc:latest
    ports:
      - "8080:8080"
    environment:
      - FIBCALC_SERVER=true
      - FIBCALC_PORT=8080
      - FIBCALC_THRESHOLD=8192
      - FIBCALC_FFT_THRESHOLD=500000
      - FIBCALC_TIMEOUT=10m
```

### Interactive Mode (REPL)

The interactive mode allows you to perform multiple calculations in a session:

```bash
./build/fibcalc --interactive
```

**Commands available in the REPL:**

| Command                     | Description                               |
| --------------------------- | ----------------------------------------- |
| `calc <n>` or `c <n>`       | Calculate F(n) with the current algorithm |
| `algo <name>` or `a <name>` | Change the algorithm (fast, matrix, fft)  |
| `compare <n>` or `cmp <n>`  | Compare all algorithms for F(n)           |
| `list` or `ls`              | List available algorithms                 |
| `hex`                       | Toggle hexadecimal display                |
| `status` or `st`            | Display current configuration             |
| `help` or `h`               | Display help                              |
| `exit` or `quit`            | Exit interactive mode                     |

**Example REPL session:**

```
fib> calc 1000
Calculating F(1000) with Fast Doubling (O(log n), Parallel, Zero-Alloc)...

Result:
  Time: 15.2µs
  Bits:  693
  Digits: 209
  F(1000) = 43466...03811 (truncated)

fib> algo matrix
Algorithm changed to: Matrix Exponentiation (O(log n), Parallel, Zero-Alloc)

fib> compare 10000
Comparison for F(10000):
─────────────────────────────────────────────
  fast                : 180.5µs ✓
  matrix              : 220.3µs ✓
  fft                 : 350.1µs ✓
─────────────────────────────────────────────

fib> exit
Goodbye!
```

### API Server Mode

```bash
# Start the server
make run-server
# or
./build/fibcalc --server --port 8080
```

**Available endpoints:**

| Endpoint      | Method | Description                                 |
| ------------- | ------ | ------------------------------------------- |
| `/calculate`  | GET    | Calculate F(n) with the specified algorithm |
| `/health`     | GET    | Server health check                         |
| `/algorithms` | GET    | List available algorithms                   |
| `/metrics`    | GET    | Server performance metrics                  |

**Request examples:**

```bash
# Simple calculation
curl "http://localhost:8080/calculate?n=1000&algo=fast"

# Health check
curl "http://localhost:8080/health"

# List algorithms
curl "http://localhost:8080/algorithms"

# Metrics
curl "http://localhost:8080/metrics"
```

See [Docs/api/API.md](Docs/api/API.md) for the complete API documentation.

### 💡 Usage Examples (Snippets)

Here are common scenarios and how to execute them efficiently.

#### 1. Basic Calculation
Calculate the 100,000th Fibonacci number and display the truncated result:

```bash
./build/fibcalc -n 100000 -c
```

#### 2. Performance Comparison
Compare all available algorithms for a specific input and show detailed metrics:

```bash
./build/fibcalc -n 1000000 -algo all -d
```

#### 3. Scripting Integration (JSON)
Output the result in JSON format for parsing by other tools (e.g., `jq`):

```bash
./build/fibcalc -n 1000 --json | jq .result.value
```

#### 4. Saving to File
Calculate a very large number (e.g., 50 million) and save the result to a file:

```bash
./build/fibcalc -n 50000000 -o fib_50m.txt
```

#### 5. Server Mode
Start the REST API server on a custom port (9090):

```bash
./build/fibcalc --server --port 9090
```

Then query it from another terminal:
```bash
curl "http://localhost:9090/calculate?n=5000"
```

#### 6. Hexadecimal Output
View the underlying binary representation in Hexadecimal:

```bash
./build/fibcalc -n 1000 --hex
```

#### 7. Silent Mode (Benchmarks)
Run a calculation without outputting the result or progress bars (useful for timing):

```bash
time ./build/fibcalc -n 10000000 -q
```

## 5. Software Architecture

This project is structured according to Go software engineering best practices, with emphasis on **modularity** and **separation of concerns**.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           ENTRY POINTS                                  │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌──────────┐ │
│  │   CLI Mode  │    │ Server Mode │    │   Docker    │    │   REPL   │ │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘    └────┬─────┘ │
└─────────┼──────────────────┼──────────────────┼────────────────┼───────┘
          └──────────────────┼──────────────────┘                │
                             ▼                                   ▼
                     ┌───────────────┐                  ┌────────────────┐
                     │ cmd/fibcalc   │                  │ internal/cli   │
                     │   main.go     │                  │   repl.go      │
                     └───────┬───────┘                  └────────────────┘
                             │
┌────────────────────────────┼────────────────────────────────────────────┐
│                   ORCHESTRATION LAYER                                   │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │
│  │   config    │  │ calibration │  │   server    │  │orchestration│    │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘    │
└────────────────────────────┼────────────────────────────────────────────┐
                             │
┌────────────────────────────┼────────────────────────────────────────────┐
│                      BUSINESS LAYER                                     │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │                    internal/fibonacci                             │  │
│  │  ┌──────────────────┐  ┌──────────────────┐  ┌────────────────┐  │  │
│  │  │  Fast Doubling   │  │     Matrix       │  │    FFT-Based   │  │  │
│  │  │  O(log n)        │  │  Exponentiation  │  │    Doubling    │  │  │
│  │  └──────────────────┘  └──────────────────┘  └────────────────┘  │  │
│  │                            │                                      │  │
│  │                    ┌───────┴───────────────────────────────────┐  │  │
│  │                    │           internal/bigfft                 │  │  │
│  │                    │  FFT Multiplication for very large N      │  │  │
│  │                    └───────────────────────────────────────────┘  │  │
│  └──────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

**Main packages:**

- **`cmd/fibcalc`**: Entry point. Orchestrates initialisation and delegates execution.
- **`internal/app`**: Application lifecycle management, signal handling, and version information.
- **`internal/fibonacci`**: Core mathematical logic (Fast Doubling, Matrix, FFT).
  - `strategy.go`: Interface and multiplication strategy implementations.
  - `doubling_framework.go`: Reusable framework for Fast Doubling.
  - `matrix_framework.go`: Framework for matrix exponentiation.
- **`internal/calibration`**: Automatic and manual performance calibration.
- **`internal/orchestration`**: Concurrent calculation execution management.
- **`internal/server`**: HTTP REST API server with security and metrics.
- **`internal/cli`**: User interface (spinner, bars, themes, REPL).
- **`internal/bigfft`**: FFT multiplication for very large numbers.
  - `pool.go`: Pooling system with pre-warming.
  - `fft.go`: FFT implementation with internal parallelisation.
- **`internal/config`**: Configuration management and flag validation.
- **`internal/errors`**: Centralised error handling.

See [Docs/ARCHITECTURE.md](Docs/ARCHITECTURE.md) for complete details.

## 6. Algorithms

| Algorithm                 | Flag           | Complexity         | Description                                                                                      |
| ------------------------- | -------------- | ------------------ | ------------------------------------------------------------------------------------------------ |
| **Fast Doubling**         | `-algo fast`   | O(log n × M(n))    | Most performant. 3 multiplications per iteration. Uses DoublingFramework with adaptive strategy. |
| **Matrix Exponentiation** | `-algo matrix` | O(log n × M(n))    | Matrix approach with Strassen-Winograd optimisation. Uses MatrixFramework.                       |
| **FFT-Based**             | `-algo fft`    | O(log n × n log n) | Forces FFT multiplication for all calculations. Uses DoublingFramework with FFT-only strategy.   |
| **GMP-Based**             | `-algo gmp`    | O(log n × M(n))    | Uses GNU MP (libgmp) C library for arithmetic. Extreme performance for very large N.             |

**Note**: All algorithms now share common frameworks that eliminate code duplication and facilitate maintenance. Multiplication strategies can be dynamically interchanged.

### Fast Doubling Formula Derivation

The _Fast Doubling_ identities are derived from the matrix form:

```math
F(2k)   = F(k) × [2×F(k+1) - F(k)]
F(2k+1) = F(k+1)^2 + F(k)^2
```

## 7. Performance Optimisations

The project integrates several layers of advanced optimisations to maximise performance:

### Zero-Allocation Strategy

- **Object Pools (`sync.Pool`)**: Calculation states are recycled to minimise GC pressure.
- **Global Memory Pooling**: Unified pooling strategies provide recycled `big.Int` objects to all algorithms (`matrix`, `fastdoubling`), significantly reducing allocation overhead.

- **Symmetric Squaring**: Reduces the number of multiplications to 4 (compared to 8 with the naive method).
- **Strassen-Winograd**: An improved variant of Strassen's algorithm that reduces the number of additions/subtractions from 18 to 15, while maintaining 7 multiplications.

### PGO Optimisation (Profile-Guided Optimization)

The project supports profile-guided optimisation (PGO), available since Go 1.20.

- **Principle**: The compiler uses a real execution profile (`default.pgo`) to optimise critical code paths (inlining, devirtualisation).
- **Gain**: **~5-10%** performance improvement on large calculations.
- **Usage**: `make build-pgo` automatically uses the included profile.

### Modular Architecture with Strategies

- **MultiplicationStrategy**: Abstraction allowing dynamic choice between different multiplication methods (Adaptive, FFT-only, Karatsuba).
- **DoublingFramework**: Reusable framework that eliminates code duplication between Fast Doubling and FFT-Based implementations.
- **MatrixFramework**: Similar framework for matrix exponentiation, facilitating maintenance and extension.

### Multi-level Parallelism

- **Multi-core Parallelism**: Multiplications are executed in parallel at the algorithm level.
- **Internal FFT Parallelisation**: FFT recursion is parallelised for large transforms, effectively leveraging multiple CPU cores during FFT calculations.
- **Configurable Thresholds**:
  - `--threshold` (default `4096` bits): Enables parallelism at the algorithm level.
  - `--fft-threshold` (default `500000` bits): Enables FFT multiplication.
  - `--strassen-threshold` (default `3072` bits): Enables the Strassen algorithm.

### Advanced Memory Optimisations

- **Prior Memory Estimation**: The system estimates memory requirements before calculation based on F(n) size ≈ n × 0.694 bits.
- **Pool Pre-warming**: Memory pools are pre-warmed with optimal buffers according to estimated needs, reducing hot allocations.
- **Buffer Reuse**: Temporary buffers are efficiently reused via the pooling system.

### Calibration

```bash
# Full calibration (recommended)
./build/fibcalc --calibrate

# Quick calibration at startup
./build/fibcalc --auto-calibrate -n 100000000
```

### Expected Performance Gains

Recent optimisations provide the following improvements:

- **Allocation Reduction**: 10-20% reduction in GC pressure thanks to pool pre-warming and recycling.
- **Maintainability Improvement**: More modular and extensible code through frameworks and strategies.
- **FFT Parallelisation**: Significant gains for N > 100M where FFT dominates calculations.

See [Docs/PERFORMANCE.md](Docs/PERFORMANCE.md) for the complete tuning guide.

## 8. Tests

The project includes a robust test suite:

```bash
# Run all tests
make test

# Short unit tests
go test -v -short ./...

# Property tests (gopter) and benchmarks
go test -bench=. -benchmem ./internal/fibonacci/

# Coverage check
make coverage

# Fuzzing tests
go test -fuzz=FuzzFastDoublingConsistency ./internal/fibonacci/
```

**Types of tests included:**

- Unit tests
- Property tests (gopter)
- Fuzzing tests (Go 1.18+)
- Benchmarks
- HTTP integration tests
- Load/stress tests

## 9. Development

### Makefile

```bash
make help          # Display all commands
make build         # Compile the project
make build-all     # Compile for all platforms
make build-pgo     # Build with Profile-Guided Optimization
make test          # Run tests
make coverage      # Generate coverage report
make benchmark     # Run benchmarks
make lint          # Check code with golangci-lint
make format        # Format code
make check         # Run all checks
make tidy          # Clean go.mod and go.sum
make deps          # Download dependencies
make upgrade       # Update dependencies
```

### Project Structure

```
.
├── cmd/
│   └── fibcalc/                   # Application entry point
│       ├── main.go                # Main logic
│       ├── main_test.go           # Integration tests
│       └── default.pgo            # PGO profile
│
├── internal/                      # Internal packages
│   ├── app/                       # Application lifecycle
│   ├── bigfft/                    # FFT multiplication for big.Int
│   ├── calibration/               # Automatic calibration
│   ├── cli/                       # CLI interface (spinner, REPL, themes)
│   ├── config/                    # Configuration and flags
│   ├── errors/                    # Centralised error handling
│   ├── fibonacci/                 # Calculation algorithms
│   ├── orchestration/             # Calculation orchestration

│   ├── server/                    # HTTP REST server
│   └── testutil/                  # Test utilities
│
├── Docs/                          # Detailed documentation
│   ├── algorithms/                # Algorithm documentation
│   │   ├── COMPARISON.md
│   │   ├── FAST_DOUBLING.md
│   │   ├── FFT.md
│   │   └── MATRIX.md
│   ├── api/                       # API documentation
│   │   ├── API.md                 # REST API documentation
│   │   ├── openapi.yaml
│   │   └── postman_collection.json
│   ├── deployment/                # Deployment guides
│   │   ├── DOCKER.md
│   │   └── KUBERNETES.md
│   ├── ARCHITECTURE.md            # Project architecture
│   ├── PERFORMANCE.md             # Performance guide
│   └── SECURITY.md                # Security policy
├── CONTRIBUTING.md                # 🤝 Contribution guide
├── Dockerfile                     # 🐳 Docker configuration
├── go.mod                         # Go dependencies
├── go.sum                         # Dependency checksums
├── LICENSE                        # Apache 2.0 licence
├── Makefile                       # 🔧 Development commands
└── README.md                      # 📚 Main documentation
```

## 10. Deployment

### Docker

```bash
# Build the image
docker build -t fibcalc:latest .

# Run in CLI mode
docker run --rm fibcalc:latest -n 1000 -algo fast -d

# Run in server mode
docker run -d -p 8080:8080 fibcalc:latest --server --port 8080
```

### Docker Compose

```yaml
version: "3.8"

services:
  fibcalc:
    build: .
    ports:
      - "8080:8080"
    command: ["--server", "--port", "8080", "--auto-calibrate"]
    deploy:
      resources:
        limits:
          cpus: "4"
          memory: 2G
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    restart: unless-stopped
```

### Kubernetes

See [Docs/deployment/KUBERNETES.md](Docs/deployment/KUBERNETES.md) for complete Kubernetes manifests.

### Resource Recommendations

| Usage            | CPU      | RAM    |
| ---------------- | -------- | ------ |
| Small (N < 100K) | 1 core   | 512 MB |
| Medium (N < 10M) | 2 cores  | 1 GB   |
| Large (N > 10M)  | 4+ cores | 2+ GB  |

## 11. Documentation

| Document                                     | Description             |
| -------------------------------------------- | ----------------------- |
| [README.md](README.md)                       | Main documentation      |
| [Docs/api/API.md](Docs/api/API.md)           | REST API documentation  |
| [CONTRIBUTING.md](CONTRIBUTING.md)           | Contribution guide      |
| [Docs/ARCHITECTURE.md](Docs/ARCHITECTURE.md) | Project architecture    |
| [Docs/PERFORMANCE.md](Docs/PERFORMANCE.md)   | Performance guide       |
| [Docs/SECURITY.md](Docs/SECURITY.md)         | Security policy         |
| [Docs/algorithms/](Docs/algorithms/)         | Algorithm documentation |
| [Docs/deployment/](Docs/deployment/)         | Deployment guides       |

## 12. Licence

This project is licensed under the Apache 2.0 licence. See the [LICENSE](LICENSE) file for details.

---

_Developed with ❤️ in Go - December 2024_
