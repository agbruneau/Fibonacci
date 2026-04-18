# Build Configuration

## Overview

This document covers the build system, compilation options, cross-compilation, and environment configuration for the Fibonacci Calculator. The project uses standard Go tooling with a Makefile for common workflows.

## Quick Start

```bash
# Build the CLI binary
go build -o fibcalc ./cmd/fibcalc

# Build and run with arguments
go run ./cmd/fibcalc -- -n 1000 -algo fast
```

The default build produces a statically linked binary for the current platform. No external dependencies are required unless building with GMP support.

## Build Tags

### GMP

The GMP build tag enables the GNU Multiple Precision Arithmetic Library backend, which can outperform pure Go for very large Fibonacci indices.

- **Source file**: `internal/fibonacci/calculator_gmp.go`
- **Build tag**: `gmp`

```bash
go build -tags=gmp -o fibcalc ./cmd/fibcalc
```

The GMP calculator auto-registers via `init()`:

```go
RegisterCalculator("gmp", func() coreCalculator { return &GMPCalculator{} })
```

#### Platform Requirements

| Platform | Package |
|----------|---------|
| Ubuntu/Debian | `sudo apt-get install libgmp-dev` |
| macOS (Homebrew) | `brew install gmp` |
| Windows | MinGW with GMP, or build under WSL |

### Profile-Guided Optimization (PGO)

PGO uses a CPU profile from a representative workload to guide the compiler toward better optimization decisions. Expected improvement is approximately 5-10% for compute-heavy paths.

- **Profile location**: `cmd/fibcalc/default.pgo`

#### PGO Workflow

```bash
# Step 1: Generate CPU profile (runs BenchmarkFastDoubling with 5s benchtime, 3 count)
make pgo-profile

# Step 2: Build with PGO
make build-pgo
# or explicitly:
go build -pgo=cmd/fibcalc/default.pgo ./cmd/fibcalc

# Full workflow (profile + build in one step)
make pgo-rebuild
```

#### PGO Makefile Targets

| Target | Description |
|--------|-------------|
| `pgo-profile` | Generate CPU profile from benchmarks |
| `build-pgo` | Build with PGO optimization |
| `build-pgo-all` | Build all platforms with PGO |
| `pgo-rebuild` | Full PGO workflow (profile + build) |
| `pgo-check` | Verify PGO profile exists |
| `pgo-clean` | Clean PGO artifacts |

## Vector Arithmetic

The `internal/bigfft` package uses `go:linkname` to access `math/big` internal vector arithmetic functions (`addVV`, `subVV`, `addMulVVW`, etc.) for performance. These are declared in `arith_decl.go` and wrapped by platform-specific files:

| File | Responsibility |
|------|---------------|
| `internal/bigfft/arith_decl.go` | `go:linkname` declarations to `math/big` internals (all platforms) |
| `internal/bigfft/arith_amd64.go` | Exported wrappers for amd64 |
| `internal/bigfft/arith_generic.go` | Exported wrappers for non-amd64 platforms |
| `internal/bigfft/cpu_amd64.go` | Runtime CPU feature detection via `golang.org/x/sys/cpu` |

Go's `math/big` package already includes platform-optimized assembly for these operations, so the `go:linkname` approach provides the best available performance on all architectures without maintaining separate assembly code.

## Cross-Compilation

### Build All Platforms

```bash
make build-all
```

This runs `build-linux`, `build-windows`, and `build-darwin` in sequence.

### Platform-Specific Builds

```bash
# Linux amd64
GOOS=linux GOARCH=amd64 go build -o fibcalc-linux-amd64 ./cmd/fibcalc

# Windows amd64
GOOS=windows GOARCH=amd64 go build -o fibcalc-windows-amd64.exe ./cmd/fibcalc

# macOS amd64
GOOS=darwin GOARCH=amd64 go build -o fibcalc-darwin-amd64 ./cmd/fibcalc

# macOS arm64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o fibcalc-darwin-arm64 ./cmd/fibcalc
```

### Cross-Compilation Targets

| Target | GOOS | GOARCH | Notes |
|--------|------|--------|-------|
| `build-linux` | linux | amd64 | Full SIMD support |
| `build-windows` | windows | amd64 | Full SIMD support |
| `build-darwin` | darwin | amd64 + arm64 | SIMD on amd64 only |

Assembly-optimized routines are amd64-only. All other architectures use the `arith_generic.go` fallback automatically.

## Version Injection

Version metadata is injected at build time via `-ldflags`:

```bash
go build -ldflags "\
  -X github.com/agbru/fibcalc/internal/app.Version=$(git describe --tags --always --dirty) \
  -X github.com/agbru/fibcalc/internal/app.Commit=$(git rev-parse --short HEAD) \
  -X github.com/agbru/fibcalc/internal/app.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  ./cmd/fibcalc
```

The Makefile `build` target handles this automatically. The injected values are available at runtime via the `--version` flag.

| Variable | Source |
|----------|--------|
| `app.Version` | `git describe --tags --always --dirty` |
| `app.Commit` | `git rev-parse --short HEAD` |
| `app.BuildDate` | UTC date in ISO 8601 format |

## Makefile Reference

The Makefile provides targets for building, testing, linting, and maintenance. Requires `make` (not available on all systems).

### Build Targets

| Target | Description |
|--------|-------------|
| `all` | Clean, build, and test |
| `build` | Build for current platform (auto-PGO if profile exists) |
| `build-all` | Build for Linux, Windows, and macOS |
| `build-linux` | Build for Linux amd64 |
| `build-windows` | Build for Windows amd64 |
| `build-darwin` | Build for macOS amd64 and arm64 |
| `build-pgo` | Build with profile-guided optimization |
| `build-pgo-all` | Build all platforms with PGO |
| `install` | Install to `$GOPATH/bin` |
| `clean` | Remove build artifacts |

### Test and Quality Targets

| Target | Description |
|--------|-------------|
| `test` | `go test -v -race -cover ./...` |
| `test-short` | `go test -v -short ./...` |
| `coverage` | Generate `coverage.html` |
| `benchmark` | Run benchmarks |
| `lint` | `golangci-lint run ./...` |
| `security` | `gosec ./...` |
| `format` | `go fmt` + `gofmt` |
| `check` | Format, lint, and test |

### Run Targets

| Target | Description |
|--------|-------------|
| `run` | Build and run |
| `run-fast` | Quick run with `n=1000` |
| `run-calibrate` | Run calibration mode |

### Dependency and Code Generation Targets

| Target | Description |
|--------|-------------|
| `tidy` | `go mod tidy` + verify |
| `deps` | `go mod download` |
| `upgrade` | `go get -u` + tidy |
| `generate-mocks` | `go generate ./...` |
| `install-mockgen` | Install mockgen tool |
| `install-tools` | Install golangci-lint and gosec |

### Utility Targets

| Target | Description |
|--------|-------------|
| `version` | Display version info |
| `help` | Display all available targets |

## Linting

The project uses `golangci-lint` with 22 linters configured in `.golangci.yml`.

```bash
# Run linter
make lint
# or
golangci-lint run ./...
```

### Key Limits

| Rule | Limit |
|------|-------|
| Cyclomatic complexity | 15 |
| Cognitive complexity | 30 |
| Function length | 100 lines |
| Function statements | 50 |

These limits are relaxed in `_test.go` files to accommodate table-driven test patterns.

## Shell Completion

Shell completion scripts can be generated for popular shells:

```bash
# Bash
fibcalc -completion bash > /etc/bash_completion.d/fibcalc

# Zsh
fibcalc -completion zsh > ~/.zsh/completions/_fibcalc

# Fish
fibcalc -completion fish > ~/.config/fish/completions/fibcalc.fish

# PowerShell
fibcalc -completion powershell >> $PROFILE
```

The implementation is in `internal/cli/completion.go`.

## Environment Variables

All environment variables use the `FIBCALC_` prefix. Configuration priority is: CLI flags > Environment variables > Adaptive hardware estimation > Static defaults.

### Calculation Parameters

| Variable | Description | Default |
|----------|-------------|---------|
| `FIBCALC_N` | Fibonacci index to compute | `100000000` |
| `FIBCALC_ALGO` | Algorithm selection (`fast`, `matrix`, `fft`, `all`) | `all` |
| `FIBCALC_TIMEOUT` | Calculation timeout | `5m` |

### Threshold Tuning

| Variable | Description | Default |
|----------|-------------|---------|
| `FIBCALC_THRESHOLD` | Parallelism activation threshold (bits) | `0` (auto: hardware-adaptive) |
| `FIBCALC_FFT_THRESHOLD` | FFT multiplication threshold (bits) | `0` (auto: hardware-adaptive) |
| `FIBCALC_STRASSEN_THRESHOLD` | Strassen algorithm threshold (bits) | `0` (auto: hardware-adaptive) |

When set to `0` (the default), thresholds are resolved via: calibration profile > hardware-adaptive estimation > static defaults (parallelism=4,096, FFT=500,000, Strassen=3,072).

### Output Control

| Variable | Description | Default |
|----------|-------------|---------|
| `FIBCALC_VERBOSE` | Enable verbose output | `false` |
| `FIBCALC_DETAILS` | Show performance details | `false` |
| `FIBCALC_QUIET` | Suppress all non-essential output | `false` |
| `FIBCALC_CALCULATE` | Display the computed Fibonacci value | `false` |
| `FIBCALC_OUTPUT` | Write result to file path | (none) |
| `FIBCALC_TUI` | Launch interactive TUI dashboard | `false` |
| `NO_COLOR` | Disable ANSI color output (standard) | (unset) |

### Calibration

| Variable | Description | Default |
|----------|-------------|---------|
| `FIBCALC_CALIBRATE` | Run full calibration mode | `false` |
| `FIBCALC_AUTO_CALIBRATE` | Run quick startup calibration | `false` |
| `FIBCALC_CALIBRATION_PROFILE` | Path to calibration profile file | (none) |

See `.env.example` for a complete reference.

## Related Documentation

- [PERFORMANCE.md](PERFORMANCE.md) -- Optimization techniques and benchmark results
- [CALIBRATION.md](CALIBRATION.md) -- Automatic threshold calibration system
- [TESTING.md](TESTING.md) -- Test strategy and execution