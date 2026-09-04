# Build Configuration

## Overview

This document covers the build system, compilation options, cross-compilation, and environment configuration for the Fibonacci Calculator. The project uses standard Go tooling with a Makefile for common workflows.

## Quick Start

```bash
# Build the CLI binary
go build -o fibcalc ./cmd/fibcalc

# Build and run with arguments
go run ./cmd/fibcalc -n 1000 -algo fast
```

Do **not** insert `--` before the program's own flags. `--` terminates Go's
flag parsing *and* `fibcalc`'s: everything after it becomes a positional
argument that the program ignores. Verified 2026-08-07 — `fibcalc -- -n 100
-algo bogus` **exits 0** having computed the default `-n 100000000` with
`-algo all`, while `fibcalc -n 100 -algo bogus` prints the usage and
**exits 4** (`internal/config/config.go:AppConfig.Validate`;
`ExitErrorConfig = 4`, `internal/errors/errors.go:ExitErrorConfig`).

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

**PGO is already on by default — there is nothing to opt into.** The profile is
committed to the repository (`.gitignore` ignores `*.pgo` but re-includes this
one explicitly, lines 33-36), and Go's `-pgo=auto` default picks up a
`default.pgo` sitting next to the main package. So the Quick Start command
above already builds an optimized binary. Verified 2026-09-04:
`go build -n ./cmd/fibcalc` embeds
`build -pgo=C:\…\cmd\fibcalc\default.pgo` in the module info. To measure
*without* PGO, pass `-pgo=off`.

The commands below therefore matter only when you want to **refresh** the
profile on your own hardware:

#### PGO Workflow

```bash
# Step 1: Generate CPU profile (runs the BenchmarkFibonacci sub-benchmarks, 5s benchtime, 3 count)
make pgo-profile

# Step 2: Build with PGO
make build-pgo
# or explicitly (equivalent to the implicit -pgo=auto above):
go build -pgo=cmd/fibcalc/default.pgo ./cmd/fibcalc

# Build with PGO disabled, for an A/B measurement
go build -pgo=off -o fibcalc-nopgo ./cmd/fibcalc

# Full workflow (profile + build in one step)
make pgo-rebuild
```

#### PGO Makefile Targets

| Target | Description |
|--------|-------------|
| `pgo-profile` | Generate CPU profile from benchmarks |
| `build-pgo` | Build with PGO optimization |
| `build-pgo-all` | Build linux/amd64, windows/amd64, and macOS (amd64 + arm64) with PGO — unlike `build-all`, no linux/arm64 or windows/arm64 |
| `pgo-rebuild` | Full PGO workflow (profile + build) |
| `pgo-check` | Verify PGO profile exists |
| `pgo-clean` | Clean PGO artifacts |

## Vector Arithmetic

The `internal/bigfft` package uses `go:linkname` to access `math/big` internal vector arithmetic functions (`addVV`, `subVV`, `addMulVVW`, etc.) for performance. These are declared in `arith_decl.go` and wrapped by a single portable file (the former `arith_amd64.go`/`arith_generic.go` build-tag split was merged by audit FFT-06):

| File | Responsibility |
|------|---------------|
| `internal/bigfft/arith_decl.go` | `go:linkname` declarations to `math/big` internals (all platforms): `addVV`, `subVV`, `addVW`, `subVW`, `shlVU`, `addMulVVW` |
| `internal/bigfft/arith.go` | Exported wrappers `AddVV` / `SubVV` / `AddMulVVW`, portable — no build tags |

The exported wrappers are **not** the hot path: they exist as test oracles for
`arith_test.go`, and each carries a `len(z) == 0` guard. Production code in
`bigfft` calls the linkname'd `addVV` / `subVV` / `addMulVVW` directly (see the
doc comment on each wrapper in `arith.go`).

Go's `math/big` package already includes platform-optimized assembly for these operations, so the `go:linkname` approach provides the best available performance on all architectures without maintaining separate assembly code. Runtime CPU feature detection (`golang.org/x/sys/cpu`) lives separately in `internal/config/hardware.go`, used for adaptive threshold heuristics — not by the `bigfft` vector arithmetic above.

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
| `build-linux` | linux | amd64 | `math/big` assembly (amd64) |
| `build-linux-arm64` | linux | arm64 | `math/big` assembly (arm64) |
| `build-windows` | windows | amd64 | `math/big` assembly (amd64) |
| `build-windows-arm64` | windows | arm64 | `math/big` assembly (arm64) |
| `build-darwin` | darwin | amd64 + arm64 | `math/big` assembly per arch |

The wrappers in `arith.go` are portable (no build tags): every architecture delegates to `math/big`'s own platform-optimized assembly via `go:linkname`. Run `make build-all` locally to exercise `linux/arm64`, `darwin/arm64`, and `darwin/amd64` so a latent platform-specific import surfaces immediately. Full matrix and portability contract: [`docs/PORTABILITY.md`](PORTABILITY.md).

Verified 2026-09-04 from a Windows host, running the same six `go build`
invocations `build-all` issues (`GOOS=<os> GOARCH=<arch> go build -trimpath ./cmd/fibcalc`):
`linux/amd64`, `linux/arm64`, `windows/amd64`, `windows/arm64`, `darwin/amd64`
and `darwin/arm64` all exit 0.

## Reproducible Build (Docker / devcontainer)

Two artifacts ship for environment isolation :

### Multi-stage `Dockerfile`

```bash
docker build -t fibcalc:local .
docker run --rm fibcalc:local --help
```

- Stage 1 (`golang:1.26-bookworm` builder) — `CGO_ENABLED=0`, no `apt`
  packages installed; builds the static default binary (no `gmp` tag).
  Consumes `cmd/fibcalc/default.pgo` if present.
- Stage 2 (`gcr.io/distroless/base-debian12` runtime) — ships only the
  linked binary as `nonroot`. Image size < 50 MB (indicative — measure on
  your own image, e.g. `docker images fibcalc:local`).

The GMP backend needs CGO + `libgmp-dev` and is intentionally out of
scope for this image — the default binary is the intended artifact here,
kept `CGO_ENABLED=0` for a smaller, statically-linked image; build with
GMP locally instead — see [Build Tags § GMP](#gmp) above.

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

The Makefile provides targets for building, testing, linting, and maintenance.
`make` alone is not enough: the recipes are **POSIX/WSL only** (they use `[ -f ]`,
`mkdir -p`, `rm -rf`, and `find`/`xargs`/`awk`), as the `Makefile` header states.
On a bare Windows host run them through WSL (`wsl make ...`); the native gate
there is `scripts/check.ps1`.

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
| `build-pgo-all` | Build linux/amd64, windows/amd64, and macOS (amd64 + arm64) with PGO — unlike `build-all`, no linux/arm64 or windows/arm64 |
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
| `check` | Run the canonical pre-commit gate (`bash scripts/check.sh`): build, vet, `go test -race` + coverage profile, `-tags gmp` step, lint report, 80% coverage floor. No formatting step — run `make format` separately |

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
| `stats` | Print package and LOC counts (canonical source for the counts quoted in `docs/ARCH.md`) |
| `bench-baseline` | Refresh `docs/audits/bench-baseline.txt` (regression baseline; fixed flags, benchstat-comparable) |
| `bench-versioned` | Record a benchmark snapshot with Go version and Git revision to `build/bench/` |
| `version` | Display version info |
| `help` | Display all available targets |

## Linting

The project uses `golangci-lint` **v2** with 21 linters plus the `gofmt`
formatter, configured in `.golangci.yml` (schema `version: "2"`).

```bash
# Install (v2 — a v1 binary cannot analyze this module under go1.27+)
make install-tools

# Run linter
make lint
# or
golangci-lint run ./...
```

The expected state is **zero findings** — the gate treats any non-zero exit as
a failure, findings and toolchain breakage alike. Verified 2026-09-04 with
`golangci-lint v2.13.2 built with go1.27.0`: `golangci-lint run ./...` prints
`0 issues.` and exits 0.

### Key Limits

| Rule | Limit |
|------|-------|
| Cyclomatic complexity | 15 |
| Cognitive complexity | 30 |
| Function length | 100 lines |
| Function statements | 50 |

These limits are relaxed in `_test.go` files to accommodate table-driven test
patterns, along with `gosec`, `unparam` and `noctx` (the e2e suites spawn
`go build` and the built binary on purpose).

## Local Pre-Commit Checks

There is no remote CI; validation is a deliberately local-only responsibility. Two
gate scripts run the same core sequence (build, vet, test, lint, 80% coverage
floor). One difference remains: `check.sh` has a step 3b that builds, vets
and tests under `-tags gmp` when the libgmp headers are present
(`scripts/check.sh`, the block headed `step "gmp build tag (-tags gmp)"`);
`check.ps1` has no such step (its header comment lists five steps, not six).

The race detector is **no longer** one of the differences (audit D4, 2026-09-03):
`check.ps1` probes for `CGO_ENABLED` and a C compiler and adds `-race` when both
are present — verified green on all 21 packages of a Windows host. Without a C
toolchain it runs the same suite without `-race`. Re-run 2026-09-04 on this
host (`go1.27.0 windows/amd64`, `CGO_ENABLED=1`, MinGW-W64 gcc 16.1.0):
`go test -race -count=1 ./...` exits 0 with `ok` on all 21 packages.

```bash
# CGO / Linux / macOS hosts (tests run WITH the race detector)
./scripts/check.sh

# Windows hosts (with -race when CGO and a C compiler are detected, without otherwise)
pwsh ./scripts/check.ps1
```

Every step is a hard gate: build, vet, test, **lint** and the 80% coverage floor.
Lint became blocking in audit GATE-01 (2026-09-03); it had been advisory, which
silently hid the fact that the pinned v1 linter could not run at all under a
go1.27 toolchain. A golangci-lint that is absent, or that exits with a code
other than 1, now fails the script rather than printing `Overall: PASS`. The
Makefile exposes the same building blocks via `make test` / `make test-win` and
`make coverage-check`.

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

The implementation lives in the `internal/cli/completion/` package: three shared
files — `registry.go` (the `flagRegistry` every generator reads), `completion.go`
(the `Generate` shell switch) and `escape.go` (shell escaping) — plus one
generator per shell (`bash.go`, `zsh.go`, `fish.go`, `powershell.go`).
`internal/app/app.go:runCompletion` calls `completion.Generate` directly; the
`cli.GenerateCompletion` wrapper and its `internal/cli/completion_dispatch.go`
file were deleted as a dead 1:1 pass-through (commit `d2aa36a`,
[ADR-0011](adr/0011-audit-2026-09-ponytail.md)).

`Generate` also answers to `"ps"`, but **the CLI does not**: `config.Validate`
accepts only the four names above and rejects the alias before `Generate` is
reached. Verified 2026-09-04 — `fibcalc -completion powershell` emits the
script (exit 0), while `fibcalc -completion ps` exits **4** with:

```
Configuration error: unrecognized completion shell: 'ps'. Valid shells are: bash, zsh, fish, powershell
```

The alias is reachable only by a Go caller using the package API directly
(`internal/cli/completion/completion.go`, the `case "powershell", "ps"` arm).

## Environment Variables

All environment variables use the `FIBCALC_` prefix (except the standard `NO_COLOR`). A `FIBCALC_*` variable is read only when the matching flag is absent from the command line, so the priority is **CLI flags > environment variables > static defaults** — uniformly, the three thresholds included since audit M-03 (see [Threshold Tuning](#threshold-tuning) below).

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

A **value you supply wins over a cached calibration profile** (audit M-03, 2026-09).
`ParseConfig` marks each threshold explicit when it came from the flag *or* from the
matching `FIBCALC_*` variable (`internal/config/config.go:markExplicitThresholds`);
`app.New` then runs `calibration.LoadCachedCalibration` after `ParseConfig`
(`internal/app/app.go:New`), and `applyProfileThresholds` fills only the thresholds
that were left to the tool (`internal/calibration/calibration.go`). Deleting the
profile is no longer necessary to make an explicit threshold stick.

When no valid profile loads, `ApplyAdaptiveThresholds` fills in just the thresholds
still left at `0`: hardware-adaptive estimation, then the static defaults
(parallelism=4,096, FFT=500,000, Strassen=3,072).

`-1` means **disabled** for `FIBCALC_THRESHOLD` / `FIBCALC_FFT_THRESHOLD` and their
flags (audit H-02) — it is the value calibration stores on hosts where sequential
wins, and validation used to reject it, which silently discarded the calibrated
profile at every start-up. `FIBCALC_STRASSEN_THRESHOLD` does **not** accept `-1`:
its consumer compares `size <= threshold`, so a negative value would force Strassen
on permanently instead of turning it off.

A fresh `--calibrate` / `--auto-calibrate` pass stays outside this rule: you asked
for a measurement, so the measurement is what gets stored and applied.

### Output Control

| Variable | Description | Default |
|----------|-------------|---------|
| `FIBCALC_VERBOSE` | Print the full result value (backs `-v`/`-verbose`; not a log-verbosity switch) | `false` |
| `FIBCALC_DETAILS` | Show performance details | `false` |
| `FIBCALC_QUIET` | Suppress all non-essential output | `false` |
| `FIBCALC_CALCULATE` | Display the computed Fibonacci value | `false` |
| `FIBCALC_OUTPUT` | Write result to file path | (none) |
| `FIBCALC_LAST_DIGITS` | Compute only the last K decimal digits (O(K) memory); `0` computes the full value | `0` |
| `FIBCALC_TUI` | Launch interactive TUI dashboard | `false` |
| `FIBCALC_TUI_THEME` | TUI palette; `high-contrast` for the accessible variant, empty for the dark default | (dark) |
| `FIBCALC_MACHINE_OUTPUT` | Emit machine-readable output (same as `--machine`) | `false` |
| `FIBCALC_MEMORY_LIMIT` | Memory budget ceiling; pre-flight estimator aborts if exceeded. Suffix is a **single letter** `K`/`M`/`G` (case-insensitive), e.g. `4G`, `512M` — `4GB` is rejected | (unbounded) |
| `FIBCALC_GC_CONTROL` | GC control during calculation: `auto`, `aggressive`, `disabled` | `auto` |
| `FIBCALC_DYNAMIC_THRESHOLDS` | Adjust the FFT/parallelism thresholds mid-computation (backs `--dynamic-thresholds`; measured neutral, [ADR-0001](adr/0001-dtm-decision.md)) | `false` |
| `NO_COLOR` | Disable ANSI color output (standard; no `FIBCALC_` prefix) | (unset) |

### Calibration

| Variable | Description | Default |
|----------|-------------|---------|
| `FIBCALC_CALIBRATE` | Run full calibration mode | `false` |
| `FIBCALC_AUTO_CALIBRATE` | Run quick startup calibration | `false` |
| `FIBCALC_CALIBRATION_PROFILE` | Path to calibration profile file | (none) |
| `FIBCALC_PROFILE_MAX_AGE` | Freshness window for a cached profile; beyond it, re-calibration runs (`calibration.ProfileMaxAgeEnv`) | `168h` (7 d) |

See `.env.example` for a complete reference. The `FIBCALC_*` table above mirrors
`envOverrides` in `internal/config/env.go`, plus `FIBCALC_TUI_THEME` (read by
`internal/ui`) and `FIBCALC_PROFILE_MAX_AGE` (read by `internal/calibration`).

## Related Documentation

- [PERFORMANCE.md](PERFORMANCE.md) -- Optimization techniques and benchmark results
- [CALIBRATION.md](CALIBRATION.md) -- Automatic threshold calibration system
- [TESTING.md](TESTING.md) -- Test strategy and execution