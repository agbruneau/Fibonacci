# Specification: FibCalc Rust Port (Rust Transmutation)

## 1. Objectives

The primary objective is to re-engineer the high-performance Go-based Fibonacci Calculator (`FibCalc`) into idiomatic Rust. This migration aims to leverage Rust's memory safety and zero-cost abstractions to **strictly exceed** the performance of the current Go implementation while maintaining 1:1 feature parity. This project serves as a high-performance reference implementation for computing large Fibonacci numbers using pure Rust libraries.

## 2. User Stories

### Core Calculation & Performance

- **As a performance engineer**, I want to calculate Fibonacci numbers (e.g., N=250,000,000) faster than the current Go implementation, so that I can demonstrate the superiority of the Rust port.
- **As a user**, I want to use the same algorithms (Fast Doubling, Matrix Exponentiation, FFT) as the Go version, so that I have consistent calculation methods available.
- **As a user**, I want the system to automatically calibrate parallelism thresholds based on my hardware, so that I get optimal performance without manual tuning.

### Interface & Usability

- **As a CLI user**, I want a command-line interface that mirrors the Go version (flags, subcommands, interactive REPL, spinners, colored output), so that my transition to the Rust version is seamless.
- **As an API consumer**, I want a REST API server with identical endpoints (`/calculate`, `/health`, `/metrics`, etc.), so that I can swap the backend without changing my client code.

### Platform & Deployment

- **As a Windows 11 user**, I want the application to run natively without requiring complex external dependencies (like GMP/MinGW), so that installation is straightforward.
- **As a Linux user**, I want a highly optimized binary that utilizes my system resources efficiently.

## 3. Acceptance Criteria

### Functional Parity (1:1)

- [ ] **CLI**: Implements all current flags (`-n`, `--algo`, `--json`, `--hex`, etc.) and the Interactive REPL mode.
- [ ] **Server**: Implements a web server (using `Axum` or `Actix`) with endpoints `/calculate`, `/health`, `/metrics` (Prometheus compatible), and `/algorithms`.
- [ ] **Algorithms**: Correctly implements Fast Doubling, Matrix Exponentiation (with Strassen), and FFT-based multiplication logic.
- [ ] **Features**: Includes Auto-calibration, Rate Limiting, and Graceful Shutdown.

### Performance

- [ ] **Speed**: The Rust implementation consistently outperforms the Go binary:
  - [ ] For $N = 100,000,000$: Rust execution time ≤ 95% of Go (5%+ faster)
  - [ ] For $N = 250,000,000$: Rust maintains the same 5%+ speedup
- [ ] **Memory**: Peak memory usage ≤ 90% of the Go version for equivalent calculations.
- [ ] **Latency (API)**: For $N = 1,000,000$ via REST API:
  - [ ] P50 latency ≤ 200ms
  - [ ] P99 latency ≤ 500ms

### Additional Quality Criteria

- [ ] **Startup Time**: CLI binary cold start < 50ms.
- [ ] **Binary Size**: Release binary (stripped) < 10MB.
- [ ] **Memory Safety**: Zero `unsafe` blocks in application code (only allowed in dependencies).

### Technical Implementation

- [ ] **Language**: Written in safe, idiomatic Rust (using Workspaces for project structure).
- [ ] **Dependencies**: Uses `num-bigint` for arbitrary-precision arithmetic (Pure Rust); does **not** rely on `rug` or system GMP libraries.
- [ ] **Platform Support**: Verified to build and pass all tests on **Linux** and **Windows 11**.

## 4. Non-Goals

- **MacOS / BSD Support**: Explicit support or testing for macOS, BSD, or other UNIX-like systems is out of scope for this iteration.
- **GMP Integration**: We will not implement a wrapper for the C GMP library (e.g., via `rug`) in this phase to ensure a pure Rust dependency chain and easier Windows support.
- **Kubernetes Manifests**: While the app will be container-ready, recreating specific K8s deployment manifests is not a priority for the initial code port.
- **GUI**: No graphical user interface will be developed.
