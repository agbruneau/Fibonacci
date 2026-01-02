# AGENTS.md - Fibonacci Calculator Development Guide

## Build & Test Commands

```bash
make build              # Build binary to ./build/fibcalc
make test               # Run all tests with race detector
make test <PKG>         # Run tests for specific package (e.g., make test ./internal/fibonacci)
go test -v -run <TEST> ./internal/fibonacci/  # Run single test by name
make coverage           # Generate coverage report (coverage.html)
make benchmark          # Run benchmarks for fibonacci algorithms
make lint               # Run golangci-lint
make check              # Run format + lint + test
make clean              # Remove build artifacts
```

## Architecture & Codebase Structure

**Go Module**: github.com/agbru/fibcalc (Go 1.25+)

**Core Packages**:

- `cmd/fibcalc` - CLI entry point (main application)
- `internal/fibonacci` - Core algorithms (Fast Doubling, Matrix, FFT); Calculator interface
- `internal/bigfft` - FFT multiplication for large integers
- `internal/server` - HTTP server with REST API endpoints (/calculate, /health, /metrics)
- `internal/cli` - REPL, UI, spinner, progress
- `internal/orchestration` - Parallel algorithm execution & strategy selection
- `internal/calibration` - System benchmarking for optimal thresholds
- `internal/service` - Business logic layer
- `internal/config` - Configuration management
- `internal/parallel` - Concurrency utilities
- `internal/errors` - Custom error types
- `internal/testutil` - Test helpers
- `internal/app` - Application coordination
- `internal/logging` - Unified logging interface and adapters

**Key Dependencies**: prometheus/client_golang, zerolog, go.opentelemetry.io, golang.org/x/sync, gmp

## Code Style & Conventions

**Imports**: Group as (1) std, (2) third-party, (3) internal; use gofmt & goimports

**Naming**: Follow Go conventions (CamelCase public, lowercase private); package comments required

**Functions**: Exported functions must have doc comments; use functional options pattern for configuration

**Error Handling**: Use `internal/errors` package; always return errors, prefer wrapping

**Concurrency**: Use `sync.Pool` for object recycling; goroutines with context cancellation

**Testing**: Table-driven tests; subtests for organization; >75% coverage target; use mockgen for mocks

**Formatting**: `make format` (gofmt + gofmt -s); golangci-lint enabled (cyclomatic complexity max 15)

**Linting Config**: .golangci.yml enforces gofmt, govet, errcheck, staticcheck, revive, gosec

**Key Patterns**: Strategy pattern for algorithms; Calculator interface abstraction; sync.Pool for memory optimization
