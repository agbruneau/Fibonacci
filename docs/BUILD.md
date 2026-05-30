# Build Configuration

> Interactive architecture map: **[agbruneau.github.io/FibGo/dashboard/](https://agbruneau.github.io/FibGo/dashboard/)** (knowledge graph, 747 nodes / 8 layers / 13-step tour). Build steps for that dashboard live below in [Dashboard statique (GitHub Pages)](#dashboard-statique-github-pages).

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

The GMP calculator auto-registers via `init()` in `calculator_gmp.go`:

```go
func init() {
    RegisterGMPCalculator(globalFactory)
}

func RegisterGMPCalculator(f *DefaultFactory) {
    f.Register("gmp", func() CoreCalculator { return &GMPCalculator{} })
}
```

#### Platform Requirements

| Platform | Package |
|----------|---------|
| Ubuntu/Debian | `sudo apt-get install libgmp-dev` |
| macOS (Homebrew) | `brew install gmp` |
| Windows | MinGW with GMP, or build under WSL |

### Profile-Guided Optimization (PGO)

PGO uses a CPU profile from a representative workload to guide the compiler toward better optimization decisions. Expected improvement is approximately 5-10% for compute-heavy paths (indicative — measure on your own workload).

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

This runs `build-linux`, `build-linux-arm64`, `build-windows`, `build-windows-arm64`, and `build-darwin` in sequence.

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
| `build-linux-arm64` | linux | arm64 | `arith_generic.go` fallback |
| `build-windows` | windows | amd64 | Full SIMD support |
| `build-windows-arm64` | windows | arm64 | `arith_generic.go` fallback |
| `build-darwin` | darwin | amd64 + arm64 | SIMD on amd64 only |

Assembly-optimized routines are amd64-only. All other architectures use the `arith_generic.go` fallback automatically. Run `make build-all` locally to exercise `linux/arm64`, `darwin/arm64`, and `darwin/amd64` so a latent amd64-only import surfaces immediately. Full matrix and fallback contract: [`docs/PORTABILITY.md`](PORTABILITY.md).

## Reproducible Build (Docker / devcontainer)

Two artifacts ship for environment isolation :

### Multi-stage `Dockerfile`

```bash
docker build -t fibcalc:local .
docker run --rm fibcalc:local --help
```

- Stage 1 (`golang:1.26-bookworm` builder) — installs `libgmp-dev` and the
  full CGO toolchain. Consumes `cmd/fibcalc/default.pgo` if present.
- Stage 2 (`gcr.io/distroless/base-debian12` runtime) — ships only the
  linked binary as `nonroot`. Image size < 50 MB (indicative — measure on
  your own image, e.g. `docker images fibcalc:local`).

To build with the GMP backend enabled : edit the Dockerfile's build line
to add `-tags=gmp`.

### `.devcontainer/devcontainer.json` (VS Code)

Opening the repo in a VS Code Dev Container loads
`mcr.microsoft.com/devcontainers/go:1.26-bookworm` with `libgmp-dev`,
`build-essential`, `staticcheck`, and `benchstat` pre-installed via
`postCreateCommand`. `CGO_ENABLED=1` is set in the container env so
`go test -race` works out of the box.

See also [`docs/PORTABILITY.md`](PORTABILITY.md) §4 for per-target build
commands and [`docs/PERFORMANCE.md`](PERFORMANCE.md) for the benchmark
baseline that documents the > 5 % regression budget.

## Version Injection

Version metadata is injected at build time via `-ldflags`:

```bash
go build -ldflags "\
  -X github.com/agbruneau/FibGo/internal/app.Version=$(git describe --tags --always --dirty) \
  -X github.com/agbruneau/FibGo/internal/app.Commit=$(git rev-parse --short HEAD) \
  -X github.com/agbruneau/FibGo/internal/app.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
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
| `build-all` | Build for Linux, Windows, and macOS (amd64 + arm64) |
| `build-linux` | Build for Linux amd64 |
| `build-linux-arm64` | Build for Linux arm64 |
| `build-windows` | Build for Windows amd64 |
| `build-windows-arm64` | Build for Windows arm64 |
| `build-darwin` | Build for macOS amd64 and arm64 |
| `build-pgo` | Build with profile-guided optimization |
| `build-pgo-all` | Build all platforms with PGO |
| `install` | Install to `$GOPATH/bin` |
| `clean` | Remove build artifacts |

### Test and Quality Targets

| Target | Description |
|--------|-------------|
| `test` | `go test -v -race -cover ./...` (race requires CGO) |
| `test-win` | `go test -v -cover ./...` (no `-race`; Windows / no-CGO hosts) |
| `test-short` | `go test -v -short ./...` |
| `coverage` | Generate `coverage.html` |
| `coverage-check` | Fail if module total coverage drops below 80% |
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
| `install-tools` | Install golangci-lint and gosec |

### Utility Targets

| Target | Description |
|--------|-------------|
| `version` | Display version info |
| `help` | Display all available targets |

## Linting

The project uses `golangci-lint` with 24 linters configured in `.golangci.yml`.

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

## Local Pre-Commit Checks

There is no remote CI; validation is a deliberately local-only responsibility. Two
equivalent gate scripts run the same sequence (build, vet, test, lint, 80% coverage
floor):

```bash
# CGO / Linux / macOS hosts (tests run WITH the race detector)
./scripts/check.sh

# Windows / no-CGO hosts (tests run WITHOUT -race; the race detector needs a C toolchain)
pwsh ./scripts/check.ps1
```

The hard gate is build/vet/test/coverage; `golangci-lint` is run as an advisory step
(reported but non-blocking). The Makefile exposes the same building blocks via
`make test` / `make test-win` and `make coverage-check`.

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

The implementation lives in the `internal/cli/completion/` package (shared `registry.go` plus one generator per shell: `bash.go`, `zsh.go`, `fish.go`, `powershell.go`), dispatched via `internal/cli/completion_dispatch.go`.

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
| `FIBCALC_TUI_THEME` | TUI palette; `high-contrast` for the accessible variant, empty for the dark default | (dark) |
| `FIBCALC_MACHINE_OUTPUT` | Emit machine-readable output (same as `--machine`) | `false` |
| `FIBCALC_MEMORY_LIMIT` | Memory budget ceiling (e.g. `4GB`); pre-flight estimator aborts if exceeded | (unbounded) |
| `NO_COLOR` | Disable ANSI color output (standard) | (unset) |

### Calibration

| Variable | Description | Default |
|----------|-------------|---------|
| `FIBCALC_CALIBRATE` | Run full calibration mode | `false` |
| `FIBCALC_AUTO_CALIBRATE` | Run quick startup calibration | `false` |
| `FIBCALC_CALIBRATION_PROFILE` | Path to calibration profile file | (none) |

See `.env.example` for a complete reference.

## Dashboard statique (GitHub Pages)

The interactive knowledge-graph dashboard at <https://agbruneau.github.io/FibGo/dashboard/> is a static build of the `@understand-anything/dashboard` package, pre-bundled with the project's knowledge graph and served from [`docs/dashboard/`](dashboard/).

### Regenerate the graph

```bash
# Inside Claude Code (or any Anthropic agent with the plugin installed):
/understand
```

This runs the `understand-anything` plugin and rewrites:
- `.understand-anything/knowledge-graph.json` (graph itself)
- `.understand-anything/meta.json` (commit hash + timestamp)
- `.understand-anything/fingerprints.json` (incremental-update baseline)

### Rebuild the dashboard bundle

Requires the `understand-anything` plugin checkout (Node 22+, pnpm 10+):

```bash
PLUGIN=~/.claude/plugins/cache/understand-anything/understand-anything/2.7.3
cd "$PLUGIN" && pnpm install --ignore-scripts && pnpm --filter @understand-anything/core build

# Bake env vars into the bundle pointing at the colocated JSON files
cd "$PLUGIN/packages/dashboard"
MSYS_NO_PATHCONV=1 \
VITE_GRAPH_URL="./knowledge-graph.json" \
VITE_META_URL="./meta.json" \
VITE_CONFIG_URL="./config.json" \
npx vite build --config vite.config.demo.ts --base=/FibGo/dashboard/

# Copy the build + graph files into docs/dashboard/
TARGET=<FibGo>/docs/dashboard
rm -rf "$TARGET"/*
cp -r dist/* "$TARGET/"
cp <FibGo>/.understand-anything/knowledge-graph.json "$TARGET/"
cp <FibGo>/.understand-anything/meta.json "$TARGET/"
touch "$TARGET/.nojekyll"
```

### Why `MSYS_NO_PATHCONV=1`?

Git Bash on Windows otherwise rewrites `/FibGo/dashboard/` into `C:\Program Files\Git\FibGo\dashboard\`, which Vite bakes into `index.html` and breaks asset loading.

### Demo mode flags

The dashboard ships with `vite.config.demo.ts` which sets `VITE_DEMO_MODE=true`. In that mode:

- The access-token gate is bypassed (`__demo__` placeholder)
- `/knowledge-graph.json`, `/meta.json`, `/config.json` URLs come from the `VITE_*_URL` env vars baked at build time
- The `/file-content.json` source-preview endpoint is unavailable (it's a dev-server middleware, not portable to static hosting)
- Search, layers, tour, graph layouts, all interactive views remain fully functional

### GitHub Pages activation

Repository **Settings → Pages**:

- **Source:** *Deploy from a branch*
- **Branch:** `main` · **Folder:** `/docs`
- Save

GitHub Pages picks up `docs/dashboard/.nojekyll` and skips Jekyll processing for the bundle. URL becomes available after ~1-2 minutes.

## Related Documentation

- [PERFORMANCE.md](PERFORMANCE.md) -- Optimization techniques and benchmark results
- [CALIBRATION.md](CALIBRATION.md) -- Automatic threshold calibration system
- [TESTING.md](TESTING.md) -- Test strategy and execution