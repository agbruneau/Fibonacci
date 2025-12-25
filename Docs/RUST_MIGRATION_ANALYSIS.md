# Analysis: Migrating FibCalc to Rust

## 1. Executive Summary

This document provides a comprehensive analysis for migrating the high-performance Fibonacci calculator (`fibcalc`) from Go to Rust. The proposed strategy is a **"from scratch" rewrite** rather than a direct port, leveraging Rust's zero-cost abstractions, ownership model, and advanced type system to achieve equal or superior performance and safety.

The primary motivation for this migration is typically to exploit Rust's lack of Garbage Collection (GC) for more predictable latency in high-performance arithmetic and to utilize its robust type safety for correctness.

## 2. Architectural Strategy

The new project should be structured as a **Cargo Workspace**, promoting modularity and separation of concerns.

### Workspace Structure
```
fibcalc/
├── Cargo.toml              # Workspace definition
├── crates/
│   ├── fibcalc-core/       # Core domain logic (Algorithms, Math)
│   ├── fibcalc-cli/        # Command-line interface
│   └── fibcalc-server/     # HTTP API server
└── tests/                  # Integration tests
```

- **fibcalc-core**: A pure library crate with minimal dependencies. It contains the Fibonacci algorithms (Fast Doubling, Matrix), the `Calculator` trait, and math abstractions.
- **fibcalc-cli**: The binary crate handling user input, flags, and output formatting.
- **fibcalc-server**: The binary crate for the REST API, handling metrics and rate limiting.

## 3. Core Algorithm Implementation

### 3.1. Big Integer Arithmetic

The current Go implementation relies on `math/big` and a custom assembly-optimized FFT implementation (`internal/bigfft`).

**Recommendation: `rug` (GMP bindings)**
- **Pros:** `rug` provides bindings to GMP (GNU Multiple Precision Arithmetic Library), which is the gold standard for performance. It will likely outperform the Go implementation immediately because Go's `math/big` is slower than GMP.
- **Cons:** Requires system dependency (`libgmp`).
- **Strategy:** Use `rug::Integer` for the core type. This simplifies the implementation significantly as GMP internally handles the transition to FFT multiplication for large numbers, potentially removing the need for a custom FFT implementation.

**Alternative: Pure Rust (`ibig` or `num-bigint`)**
- **Pros:** No external C dependencies; easier cross-compilation.
- **Cons:** `num-bigint` is generally slower. `ibig` is faster but may still lag behind GMP.
- **Decision:** Given the "high-performance" requirement, **`rug`** is the recommended starting point. A feature flag `pure-rust` could be added later using `ibig`.

### 3.2. Concurrency Model

- **Go:** Uses goroutines and `sync.WaitGroup` for parallel multiplication.
- **Rust:** Use **`rayon`** for data parallelism.
    - Instead of manually spawning goroutines, `rayon` allows defining parallel iterators or using `join` to execute the recursive steps of Fast Doubling (FK and FK1) in parallel.
    - Rust's ownership model guarantees thread safety at compile time, eliminating data races.

### 3.3. Memory Management

- **Go:** Uses `sync.Pool` to reuse `big.Int` allocations and reduce GC pressure.
- **Rust:**
    - **No GC:** Memory is deterministically freed when it goes out of scope.
    - **Optimization:** While `sync.Pool` is less critical in Rust, re-using buffers (e.g., `rug::Integer` capacity) is still beneficial for tight loops. We can implement a similar "Arena" or "Scratchpad" pattern if profiling shows allocator overhead.

## 4. Ecosystem Mapping

| Feature | Go Component | Recommended Rust Crate |
| :--- | :--- | :--- |
| **CLI Argument Parsing** | `flag` / Custom | **`clap`** (derive mode) |
| **HTTP Server** | `net/http` | **`axum`** or `actix-web` |
| **JSON Serialization** | `encoding/json` | **`serde`** + `serde_json` |
| **Big Int Math** | `math/big` | **`rug`** (GMP) |
| **Parallelism** | `go` / `sync` | **`rayon`** |
| **Async Runtime** | N/A (runtime built-in) | **`tokio`** (for Server) |
| **Testing** | `testing` | `cargo test` |
| **Property Testing** | `gopter` | **`proptest`** |
| **Benchmarking** | `testing` | **`criterion`** |
| **UI/Spinner** | `go-spinner` | **`indicatif`** |
| **Logging** | `zerolog` | **`tracing`** + `tracing-subscriber` |

## 5. Implementation Roadmap

### Phase 1: The Core (`fibcalc-core`)
1.  Define the `Calculator` trait.
2.  Implement `FastDoubling` using `rug`.
3.  Implement `MatrixExponentiation` using `rug`.
4.  Port the recursion logic.
5.  Add `criterion` benchmarks to compare against the Go version (compiled binary).

### Phase 2: The CLI (`fibcalc-cli`)
1.  Set up `clap` for argument parsing (replicating flags `-n`, `-algo`, etc.).
2.  Implement the UI/Spinner using `indicatif`.
3.  Wire up the core algorithms.

### Phase 3: The Server (`fibcalc-server`)
1.  Set up `axum` with `tokio`.
2.  Implement `/calculate` endpoint.
3.  Add metrics middleware (`metrics-rs`).

### Phase 4: Optimization & Refinement
1.  Profile with `flamegraph`.
2.  Implement "Adaptive Strategy" (switching algorithms based on input size).
3.  Add parallel computation support using `rayon`.

## 6. Risk Assessment

1.  **Complexity of FFT:** If GMP (`rug`) is deemed insufficient and a custom FFT is required in Rust, the complexity increases significantly. There is no direct "drop-in" high-performance pure-Rust FFT library comparable to the hand-tuned Go assembly in `internal/bigfft`.
    *   *Mitigation:* Stick to GMP initially. It is industry standard.
2.  **Learning Curve:** Rust's borrow checker can be challenging for recursive algorithms involving shared state (though Fibonacci algorithms are mostly pure functions, reducing this risk).
3.  **Build Times:** Rust compile times are longer than Go.
    *   *Mitigation:* Use `cargo-nextest` and a good linker (`mold`).

## 7. Conclusion

Migrating to Rust offers a significant opportunity to modernize the codebase and potentially increase raw computational performance through GMP integration and zero-cost abstractions. The strict type system will also improve long-term maintainability and correctness. The "From Scratch" approach is the correct one, as it allows adopting idiomatic Rust patterns (Traits, RAII, Rayon) rather than fighting the language to mimic Go's structural typing and goroutines.
