# Governance & Standards - Rust Transmutation

This document defines the strict technical and stylistic standards for the `Rust_Transmutation` project. It is formatted to serve as the `.cursorrules` configuration for AI assistants to ensure consistency and high quality.

---

```markdown
# .cursorrules

# Project Context
You are working on "Rust Transmutation", a high-performance port of a Go Fibonacci Calculator to Rust.
The goal is to strictly exceed the performance of the Go version while maintaining 1:1 feature parity.
Target Platforms: Linux, Windows 11.

# Codebase Standards & Golden Rules

## 1. Technical Stack & Dependencies
- **Language**: Rust (Latest Stable).
- **Big Int Arithmetic**: MUST use `num-bigint` (Pure Rust). DO NOT use `rug` or `gmp-mpfr-sys` (to ensure easy Windows support).
- **Async Runtime**: `tokio` with `full` features.
- **Web Server**: `axum`.
- **CLI**: `clap` with `derive` feature.
- **Logging/Tracing**: `tracing` and `tracing-subscriber`.
- **Serialization**: `serde` and `serde_json`.
- **Error Handling**: `thiserror` for libraries/modules, `anyhow` for application/CLI entry points.

## 2. Architectural Principles
- **Clean Architecture**: Strictly separate `core` (domain logic, algorithms) from `adapters` (CLI, Server).
- **Modularity**: Use a Cargo Workspace structure.
  - `crates/fib-core`: Core algorithms and calculator logic.
  - `crates/fib-cli`: Command-line interface logic.
  - `crates/fib-server`: REST API logic.
  - `apps/fibcalc`: Main binary composition root.
- **Zero-Cost Abstractions**: Prefer Generics and Traits over `Box<dyn Trait>` in hot paths unless dynamic dispatch is strictly required.
- **Composition over Inheritance**: Use Traits and Struct composition.

## 3. Coding Style & Conventions
- **Formatting**: Strictly follow `rustfmt`.
- **Linting**: Enforce `clippy` pedantic rules.
  - `#![warn(clippy::pedantic)]`
  - `#![deny(clippy::unwrap_used)]` (No `unwrap()` or `expect()` in core logic; handle errors gracefully).
- **Naming**:
  - Types/Traits: `UpperCamelCase`
  - Functions/Variables/Modules: `snake_case`
  - Constants: `SCREAMING_SNAKE_CASE`
- **Visibility**: Default to private. Use `pub(crate)` where possible. Only `pub` what is part of the public API.

## 4. Documentation
- **Rustdoc**: Every public struct, enum, function, and trait MUST have a documentation comment (`///`).
- **Examples**: Include usage examples in doc comments where non-trivial.
- **Readme**: Keep the project README updated with setup and usage instructions.

## 5. Testing Strategy
- **Unit Tests**: Co-located in the same file as the code, in a `mod tests` block.
- **Integration Tests**: In the `tests/` directory of the crate.
- **Property-Based Testing**: Use `proptest` to verify mathematical properties (e.g., Cassini's Identity).
- **Benchmarking**: Use `criterion` for performance verification against the Go baseline.

## 6. Error Handling
- **Typed Errors**: Define custom error enums using `thiserror` for domain logic.
- **Context**: Attach context to errors using `anyhow::Context` in application layers.
- **Panic Policy**: Do NOT panic in library code. Return `Result`.

## 7. Performance Guidelines
- **Allocations**: Minimize heap allocations in the hot loop. Reuse buffers/BigInts where possible.
- **Parallelism**: Use `rayon` for data parallelism (e.g., parallel matrix multiplication).
- **Optimization**: Profile before optimizing. Focus on the critical path (Fibonacci algorithms).

## 8. Specific "Transmutation" Rules (Go to Rust)
- **WaitGroups**: Replace `sync.WaitGroup` with `tokio::task::JoinSet` or `rayon` scopes.
- **Channels**: Use `tokio::sync::mpsc` for async or `crossbeam` for sync channels.
- **Context**: Use `tokio_util::sync::CancellationToken` for cancellation propagation instead of Go's `context.Context`.
- **Structs**: Map Go structs to Rust structs. Implement `Default` trait where applicable.
```
