# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Interactive TUI mode**: btop-style dashboard built with Bubble Tea (Elm architecture), featuring real-time progress charts, algorithm comparison, and keyboard navigation
- Portable arithmetic fallback for non-amd64 architectures (`arith_generic.go`)
- Godoc example functions for `Calculator`, `DefaultFactory`, and `CalculateWithObservers`
- `doc.go` for every internal package; enriched `internal/bigfft`, `internal/calibration`, `internal/app`, `internal/parallel`, `internal/testutil` with role/invariants/example comments (audit P2-13, P2-18)
- Cross-compilation targets for Linux and Windows **arm64** in the Makefile
- `.env.example` coverage for `FIBCALC_TUI_THEME`, `FIBCALC_MACHINE_OUTPUT`, `FIBCALC_MEMORY_LIMIT` (audit P1-13, P1-14)
- `docs/architecture/patterns/design-patterns.md` inventory of concrete design patterns in use (audit P2-21)
- **Arena pooling**: `AcquireStateForN(n)` / `ReleaseStateWithResult(s, src)` API in `internal/fibonacci/` — `CalculationState` now owns its `CalculationArena`, the two pools share a single lifecycle, and all big.Int slot aliases are detached before `sync.Pool.Put` to keep the arena race-free for reuse (audit P1-04)
- New regression tests: `TestArenaPoolingNoAliasing`, `TestArenaStateConcurrent` (16 goroutines × 8 iters × 3 sizes), `TestStateReuseAcrossSizes` in `internal/fibonacci/state_pool_arena_test.go`
- `docs/audits/2026-04/INTERVENTION_PLAN.md` and `docs/audits/2026-04/bench/perf-results/P1-04-arena-pool/` (before/after benchmarks + validation summary) tracking the post-PR-#17 finalization

### Changed

- **Go toolchain**: bumped `go.mod` to Go 1.25 (toolchain go1.26.2) — audit P0-02
- **Dependencies**: minor/patch upgrades for `golang.org/x/sync`, `x/sys`, `x/term`, `x/text`, `github.com/rs/zerolog`, and `gopsutil` (audit P1-24)
- **Dependencies (major)**: bumped `github.com/charmbracelet/bubbles` from `v0.21.1` to `v1.0.0`. The bubbles v1.0 release preserved the `key` and `viewport` sub-package surfaces actually used by the TUI; zero source changes were required (audit P0-03)
- **Package restructuring**: Extracted `internal/progress/` package from `internal/fibonacci/` (observer pattern, progress types); backward-compatible type aliases in `progress_aliases.go`
- **Package restructuring**: Extracted `internal/fibonacci/memory/` sub-package (arena, GC control, memory budget)
- **Package restructuring**: Extracted `internal/fibonacci/threshold/` sub-package (dynamic threshold manager)
- Extracted `internal/app/calculate.go` — calculation dispatch logic from `app.go`
- Extracted `internal/config/thresholds.go` — adaptive threshold estimation (canonical implementation)
- Added `internal/orchestration/progress.go` — `ProgressAggregator` for multi-calculator progress
- Dependency injection: `app.New()` accepts `WithFactory()` option for custom `CalculatorFactory`
- Removed `MultiplicationStrategy` deprecated type alias from `strategy.go`
- Removed server, REPL, and observability layers to simplify the codebase
- Documentation restructure: badge coverage updated to 87.5%; README condensed (~38 KB → ≤ 22 KB) with deep-links to `docs/PERFORMANCE.md`, `docs/BUILD.md`, `docs/TUI_GUIDE.md`
- Benchmark reporting unified on AMD Ryzen 9 5900X reference; Intel Core Ultra 9 numbers retained as comparison annex (audit P2-19)
- Dependency graph (`docs/architecture/dependency-graph.mermaid`) now includes `progress`, `memory`, `threshold` nodes (audit P2-22)
- Makefile hygiene: POSIX-only targets, `go mod tidy`, removed dead cross-compile targets (audit P2-23 to P2-26)

### Fixed

- **orchestration**: propagate `errgroup` error instead of silently ignoring (audit P0-08)
- **fibonacci**: check `ctx.Err()` after semaphore acquisition (audit P1-19)
- **bigfft / io**: handle `Flush()` and `fourier()` errors explicitly (audit P2-11, P2-12)
- Documentation links: fixed 20+ broken cross-references across `README.md`, `docs/ARCH.md`, `docs/TESTING.md`, architecture/ hub (audit P0-04 through P0-07, P1-16)
- `docs/TESTING.md` Test Organization table: paths updated for extracted sub-packages (audit P1-15)
- `docs/PERFORMANCE.md`: removed phantom `FIBCALC_GC_CONTROL` reference; GC control is automatic (audit P1-12)
- `CONTRIBUTING.md` vs `docs/TESTING.md` mockgen divergence: `TESTING.md` is now the single source of truth (audit P2-17)
- Formatting: applied `gofmt -s` and `goimports` across the tree (audit P1-09)

### Security

- **gosec G115**: explicit whitelist with justification in `.golangci.yml` (audit P1-23, P2-09)
- **gosec G304**: documented file-path inclusion exceptions (audit P2-10)

### Performance

- **FFT pool leak**: release `PolValues` / `Poly` buffers in `internal/bigfft` (audit P0-01, P0-09)
- **Arena reuse**: `CalculationArena` is now retained across calls inside the pooled `CalculationState` (sized once, `Reset()` between uses; rebuilt only when `n` outgrows the previous tenancy). `B/op` improves ~7 % on `BenchmarkFibonacci/FastDoubling/1M`; `ns/op` is unchanged within run-to-run variance. The "steal `s.FK`" zero-copy trick was removed in `ExecuteDoublingLoop` (incompatible with arena reuse) and replaced by a single deep-copy in `ReleaseStateWithResult` (~850 KB memcpy for F(10M), <0.01 % of runtime). Audit P1-04 — previously SKIPPED due to a race documented in `docs/audits/2026-04/bench/perf-results/P1-04-SKIPPED.md`, now resolved.

### Removed

- Phantom `FIBCALC_GC_CONTROL` environment variable reference from `docs/PERFORMANCE.md` (never implemented)
- Tracked development log artifacts and stale files (audit P1-20)

---

## [1.0.0] - 2025-12-22

### Added

#### Core Features

- **Fast Doubling Algorithm**: O(log n) Fibonacci calculation with parallel multiplication
- **Matrix Exponentiation**: O(log n) with Strassen's algorithm for large matrices
- **FFT-Based Calculator**: Optimized for extremely large numbers using FFT multiplication
- **GMP Support**: Optional GNU Multiple Precision library integration via build tag

#### Performance Optimizations

- Zero-allocation strategy using `sync.Pool` for 95%+ reduction in GC pressure
- Adaptive parallelism based on input size and hardware capabilities
- Smart multiplication switching (Karatsuba vs FFT) based on operand size
- Symmetric matrix squaring optimization (50% reduction in multiplications)
- Auto-calibration system for hardware-specific threshold optimization

#### User Interface

- Modern CLI with progress spinners, ETA calculation, and colour themes
- Shell autocompletion generation (bash, zsh, fish, PowerShell)
- JSON output format support
- Hexadecimal result display option

#### Documentation

- Comprehensive README with production deployment guide
- Architecture documentation with ADRs
- Performance tuning guide
- Security policy with vulnerability disclosure process
- Algorithm-specific documentation (Fast Doubling, Matrix, FFT, GMP)

#### Development

- Comprehensive test suite with 80%+ coverage
- Benchmark suite for performance validation
- Mock generation with mockgen
- golangci-lint configuration

### Security

- Input validation for all parameters
- Maximum N value limit (1 billion) to prevent resource exhaustion
- Configurable request timeouts
- Rate limiting protection against DoS

---

## [0.1.0] - 2025-11-01

### Added

- Initial project structure
- Basic Fast Doubling implementation
- Command-line interface
- Unit tests for core algorithms

---

[Unreleased]: https://github.com/agbru/fibcalc/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/agbru/fibcalc/compare/v0.1.0...v1.0.0
[0.1.0]: https://github.com/agbru/fibcalc/releases/tag/v0.1.0
