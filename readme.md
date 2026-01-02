# Fibonacci

Welcome to the **Fibonacci** monorepo! This repository houses two high-performance implementations of Fibonacci number calculators, showcasing advanced algorithmic optimizations and modern software engineering practices in **Go** and **Rust**.

Both projects are designed to compute massive Fibonacci numbers (millions of digits) with extreme speed, utilizing techniques like Fast Doubling, Matrix Exponentiation with Strassen's algorithm, and FFT-based multiplication.

## 📂 Project Structure

This repository is split into two independent projects:

### 1. [FibGo](./FibGo) (Go Implementation)

A state-of-the-art Go toolkit featuring a CLI, an interactive REPL, and a high-performance REST API.

- **Status**: Production-Ready
- **Tech Stack**: Go 1.25+, Fiber (Server), Zerolog, Docker.
- **Key Features**:
  - **Zero-Allocation**: Heavy use of `sync.Pool` to minimize GC pressure.
  - **Algorithms**: Fast Doubling (default), Matrix+Strassen, FFT.
  - **Connectivity**: Robust REST API with metrics and rate limiting.
  - **Interactive**: Built-in CLI REPL for experimentation.

### 2. [FibRust](./FibRust) (Rust Implementation)

A highly parallelized Rust workspace leveraging the safety and speed of the Rust ecosystem.

- **Status**: Stable / Active Development
- **Tech Stack**: Rust 1.75+, Axum (Server), Clap (CLI), Rayon, RustFFT, `ibig`.
- **Key Features**:
  - **Adaptive Execution**: Automatically transforms strategies (Sequential -> Parallel -> FFT) based on input size.
  - **Parallelism**: Rayon-powered parallel Fast Doubling for multi-core scaling.
  - **Correctness**: Uses `ibig` for arbitrary precision integers ensuring no overflows.
  - **Modular**: Clean workspace structure (`core`, `cli`, `server`).

## 🚀 Quick Comparison

| Feature           | FibGo                              | FibRust                           |
| :---------------- | :--------------------------------- | :-------------------------------- |
| **Primary Focus** | Web Services / Tooling Consistency | Raw Parallel Computation Power    |
| **Default Algo**  | Fast Doubling                      | Adaptive (Fast Doubling -> FFT)   |
| **Parallelism**   | Goroutines (Manual orchestration)  | Rayon (Work-stealing thread pool) |
| **Big Int Lib**   | `math/big` (GMP optional)          | `ibig` (pure Rust)                |
| **Server**        | Fiber                              | Axum                              |

## 🏁 Getting Started

Choose your preferred language implementation to get started:

### Go

```bash
cd FibGo
go run ./cmd/fibcalc -n 1000000
```

### Rust

```bash
cd FibRust
# Run via Cargo Workspace
cargo run -p fibrust-cli --release -- 1000000
```

## 📄 License

This project hosts code under open-source licenses. Please refer to the individual [FibGo LICENSE](./FibGo/LICENSE) and [FibRust LICENSE](./FibRust/Cargo.toml) (MIT) for details.
