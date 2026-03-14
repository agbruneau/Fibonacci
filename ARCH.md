# FibGo / FibCalc Architecture

## 1) Project Overview

**FibGo** (module/library name: **FibCalc**) is a high-performance Fibonacci computation system implemented in Go.

- **Go module path:** `github.com/agbru/fibcalc`
- **Primary binary:** `cmd/fibcalc`
- **Purpose:** compute very large Fibonacci values efficiently, compare multiple algorithms, and expose both CLI and TUI execution modes.
- **Core strengths:**
  - Multiple `O(log n)` Fibonacci algorithms
  - Adaptive multiplication strategy (`math/big` vs FFT)
  - Optional GMP backend via build tag
  - Runtime calibration/adaptive thresholds
  - Concurrency-aware orchestration and progress reporting

At runtime, FibCalc can execute one or many calculators in parallel, aggregate progress, validate result consistency across algorithms, and present results through CLI or TUI presentation layers.

---

## 2) High-Level Architecture (Clean Architecture)

```text
+-----------------------------------------------------------------------+
|                              Interfaces                               |
|                                                                       |
|  cmd/fibcalc  cmd/generate-golden  internal/cli  internal/tui  ui    |
+----------------------------------+------------------------------------+
                                   |
                                   v
+-----------------------------------------------------------------------+
|                           Application Layer                           |
|                                                                       |
|    internal/app          internal/config         internal/calibration |
|  (lifecycle/modes)       (flags/env/validation) (profile + tuning)   |
+----------------------------------+------------------------------------+
                                   |
                                   v
+-----------------------------------------------------------------------+
|                             Use-Case Layer                            |
|                                                                       |
|                   internal/orchestration                              |
|     (calculator selection, parallel execution, aggregation, compare)  |
+----------------------------------+------------------------------------+
                                   |
                                   v
+-----------------------------------------------------------------------+
|                             Domain Layer                              |
|                                                                       |
| internal/fibonacci  internal/progress  internal/bigfft  internal/parallel |
| (algorithms)        (observer model)   (FFT arithmetic) (concurrency errs) |
+----------------------------------+------------------------------------+
                                   |
                                   v
+-----------------------------------------------------------------------+
|                          Infrastructure Helpers                       |
|                                                                       |
| internal/metrics  internal/sysmon  internal/format  test/e2e, docs    |
+-----------------------------------------------------------------------+
```

---

## 3) Directory Structure

### Top-level tree (annotated)

```text
.
├── cmd/
│   ├── fibcalc/                 # Main application entrypoint
│   └── generate-golden/         # Golden-data generator for tests
├── internal/                    # Application and domain internals
├── test/
│   └── e2e/                     # End-to-end CLI tests
├── docs/                        # Architecture, algorithm, build, test, perf docs
├── .env.example                 # Supported FIBCALC_* env variables
├── go.mod                       # Module + direct dependencies
├── Makefile                     # Build/test/lint/PGO/cross-compile workflows
├── README.md                    # Product and usage overview
└── ARCH.md                      # This architecture document
```

### `internal/` package map

```text
internal/
├── app/                         # Lifecycle, mode dispatch, version
├── bigfft/                      # FFT multiplication engine for big.Int
├── calibration/                 # Threshold benchmarking + profile persistence
├── cli/                         # CLI output/presenter/spinner/completion
├── config/                      # Flag parsing, env override, adaptive thresholds
├── errors/                      # Typed app errors + exit code handling
├── fibonacci/                   # Core Fibonacci algorithms + framework/strategy/factory
│   ├── memory/                  # Arena allocator, GC control, memory budget
│   └── threshold/               # Dynamic threshold manager
├── format/                      # Duration/number/progress ETA formatting
├── metrics/                     # Runtime performance/memory indicators
├── orchestration/               # Concurrent execution and result analysis
├── parallel/                    # Thread-safe first-error collector
├── progress/                    # Observer pattern (subject/observers/update model)
├── sysmon/                      # System monitoring hooks (CPU/memory)
├── testutil/                    # Shared test helpers
├── tui/                         # Bubble Tea dashboard mode
└── ui/                          # Themes/colors/NO_COLOR behavior
```

---

## 4) Core Packages (Responsibilities, Key Types, Interfaces)

## `internal/app`
- **Responsibility:** startup + runtime mode orchestration (completion, calibration, TUI, normal calculation).
- **Key types/functions:** `Application`, `New`, `Run`, `runCalculate`, `runTUI`, `runCalibration`.

## `internal/config`
- **Responsibility:** parse CLI flags, validate configuration, apply `FIBCALC_` env overrides, apply adaptive thresholds.
- **Key types:** `AppConfig`.
- **Key functions:** `ParseConfig`, `ApplyAdaptiveThresholds`, `EstimateOptimal*Threshold`.

## `internal/calibration`
- **Responsibility:** full/quick calibration, adaptive threshold candidate generation, profile file persistence.
- **Key types:** `CalibrationProfile`, `CalibrationOptions`.
- **Key functions:** `RunCalibration`, `AutoCalibrate`, `LoadOrCreateProfile`, `SaveProfile`.

## `internal/orchestration`
- **Responsibility:** execute calculators concurrently, collect durations/errors/results, compare consistency, present summary.
- **Key types:** `CalculationResult`, `PresentationOptions`, `ProgressAggregator`.
- **Key interfaces:**
  - `ProgressReporter`
  - `ResultPresenter`
  - `ErrorHandler`

## `internal/fibonacci`
- **Responsibility:** domain algorithms, strategy selection, factory/registry, pooled state, framework loops.
- **Key interfaces:**
  - `Calculator` (public)
  - `coreCalculator` (internal)
  - `CalculatorFactory`
  - `Multiplier`
  - `DoublingStepExecutor`
- **Key types:**
  - `FibCalculator` (decorator)
  - `OptimizedFastDoubling`
  - `MatrixExponentiation`
  - `FFTBasedCalculator`
  - `Options`
  - `CalculationState`
  - `DefaultFactory`

### `internal/fibonacci/memory`
- **Responsibility:** memory management during large computations.
- **Key types/functions:**
  - `CalculationArena` (contiguous bump-style arena)
  - `GCController` (`auto`/`aggressive`/`disabled`)
  - `EstimateMemoryUsage`, `ParseMemoryLimit`

### `internal/fibonacci/threshold`
- **Responsibility:** dynamic runtime threshold adjustment based on observed iteration performance.
- **Key types:** `DynamicThresholdManager`, `DynamicThresholdConfig`, `IterationMetric`, `ThresholdStats`.

## `internal/progress`
- **Responsibility:** Observer pattern for progress updates.
- **Key types/interfaces:** `ProgressObserver`, `ProgressSubject`, `ProgressUpdate`, `ProgressCallback`.

## `internal/bigfft`
- **Responsibility:** high-performance FFT-based multiplication/squaring for `big.Int`.
- **Key APIs:** `Mul`, `MulTo`, `Sqr`, `SqrTo`.
- **Subsystems:** FFT recursion, transform cache, object pools, bump allocator, Fermat arithmetic, CPU feature probing.

## `internal/cli`
- **Responsibility:** terminal UX for non-TUI mode (progress, table/result output, shell completion).
- **Key components:** `CLIProgressReporter`, `CLIResultPresenter`, output formatters/writers.

## `internal/tui`
- **Responsibility:** Bubble Tea Elm-style dashboard (`Model-Update-View`) for interactive execution.
- **Integration:** provides orchestration-compatible progress/result bridge.

## `internal/errors`
- **Responsibility:** typed errors, wrappers, exit code mapping, standardized calculation-error handling.
- **Key types:** `ConfigError`, `CalculationError`, `TimeoutError`, `ValidationError`, `MemoryError`.

## `internal/parallel`
- **Responsibility:** concurrency utility for safe first-error capture.
- **Key type:** `ErrorCollector`.

## `internal/metrics`, `internal/format`, `internal/ui`, `internal/sysmon`, `internal/testutil`
- **Responsibility:** telemetry formatting, memory/performance indicators, theming/color controls, host metrics access, test helpers.

---

## 5) Design Patterns

| Pattern | Where | Why it exists |
|---|---|---|
| **Decorator** | `fibonacci.FibCalculator` wrapping `coreCalculator` | Adds cross-cutting behavior (small-N fast path, observer adaptation, GC control hooks) without changing algorithm cores |
| **Strategy** | `Multiplier` / `DoublingStepExecutor` with `AdaptiveStrategy`, `FFTOnlyStrategy`, `KaratsubaStrategy` | Enables swapping multiplication policy by workload/benchmark intent |
| **Observer** | `progress.ProgressSubject` + `ProgressObserver` implementations | Decouples progress production from UI/log consumers |
| **Factory + Registry** | `DefaultFactory` implementing `CalculatorFactory` | Centralized calculator registration/lookup/caching (`fast`, `matrix`, `fft`, optional `gmp`) |
| **Framework (Template-like loop ownership)** | `DoublingFramework`, `MatrixFramework` | Keeps algorithm loops stable while plugging in operation strategy/threshold behavior |
| **Object Pool** | `sync.Pool` in Fibonacci state and `bigfft` pools | Cuts allocations and GC pressure in hot paths |
| **Arena Allocator** | `memory.CalculationArena` | Pre-sizes contiguous backing storage for big.Int state to reduce fragmentation/GC overhead |

Additional notable engineering patterns include runtime-configurable threshold heuristics, channel-based aggregation, and lock-free observer snapshots (`Freeze`).

---

## 6) Data Flow (CLI input to final result)

1. **Process entry**
   - `cmd/fibcalc/main.go` calls `app.New(args, stderr)` then `Run(ctx, stdout)`.
2. **Config resolution**
   - `config.ParseConfig` parses flags.
   - Env overrides apply for unset flags (`FIBCALC_*`).
   - Validation checks semantic constraints.
   - Calibration profile may be loaded; otherwise adaptive threshold estimation is applied.
3. **Mode dispatch**
   - Completion mode (`-completion`) OR
   - Calibration mode (`-calibrate`) OR
   - TUI mode (`-tui`) OR
   - Standard CLI calculation mode.
4. **Context lifecycle**
   - `context.WithTimeout(config.Timeout)` and signal cancel (`SIGINT`, `SIGTERM`).
5. **Calculator selection**
   - `orchestration.GetCalculatorsToRun(algo, factory)` chooses one or many calculators.
6. **Concurrent execution model**
   - `orchestration.ExecuteCalculations` starts progress display goroutine.
   - Single calculator: direct call path.
   - Multiple calculators: `errgroup` fan-out; each algorithm runs in its own goroutine.
   - Inside calculators, further parallelism may occur for multiplication steps (threshold-dependent) with semaphore limiting (`NumCPU*2`).
7. **Progress propagation**
   - Core algorithm emits normalized progress.
   - `ProgressSubject` notifies observers (channel/log/no-op).
   - Reporter (CLI/TUI) aggregates updates + ETA.
8. **Result analysis**
   - `AnalyzeComparisonResults` sorts by success/duration, checks mismatches, emits status.
9. **Output and exit**
   - Presenter prints comparison table and selected result.
   - Optional file output write.
   - Error handler maps failures to exit codes.

### Concurrency summary
- **Level 1:** algorithm-level parallel execution (multi-calculator comparison).
- **Level 2:** intra-algorithm operation parallelism (doubling/matrix tasks).
- **Level 3:** FFT internals may parallelize recursion; logic avoids over-parallelization when FFT already saturates cores.

---

## 7) Algorithm Layer

## A. Fast Doubling (`OptimizedFastDoubling`)
- Core identities:
  - `F(2k)   = F(k) * (2F(k+1) - F(k))`
  - `F(2k+1) = F(k+1)^2 + F(k)^2`
- Uses `DoublingFramework` + `AdaptiveStrategy`.
- Employs pooled `CalculationState`, memory arena pre-sizing, and optional dynamic threshold updates.

## B. Matrix Exponentiation (`MatrixExponentiation`)
- Uses binary exponentiation of Fibonacci Q-matrix.
- `MatrixFramework` drives loop.
- Matrix ops switch between naive 2x2 multiply and Strassen based on thresholds.
- Includes symmetric squaring optimization.

## C. FFT-Based Doubling (`FFTBasedCalculator`)
- Same doubling loop model, but strategy is `FFTOnlyStrategy`.
- Forces FFT multiplication/squaring for benchmark and extremely large-input scenarios.

## Strategy system
- `Multiplier` is narrow (multiply/square only).
- `DoublingStepExecutor` extends it with `ExecuteStep` for full doubling-step optimization.
- Implementations:
  - `AdaptiveStrategy`: threshold-driven `math/big` vs FFT
  - `FFTOnlyStrategy`: always FFT
  - `KaratsubaStrategy`: always `math/big` path

## `internal/bigfft` role
- Provides efficient arithmetic primitives for huge operands:
  - transform and inverse-transform pipeline
  - transform caching
  - buffer pooling + pool warming
  - bump allocation for temporary blocks
  - architecture-aware arithmetic wrappers
- Public API used by Fibonacci layer via `Mul/MulTo/Sqr/SqrTo`.

---

## 8) Configuration and Environment

## Core CLI flags (selected)

| Flag | Meaning |
|---|---|
| `-n` | Fibonacci index |
| `-algo` | `all`, `fast`, `matrix`, `fft` (and `gmp` if built/tagged) |
| `-timeout` | Global execution timeout |
| `-threshold` | Parallelism threshold (bits), `0` = auto |
| `-fft-threshold` | FFT threshold (bits), `0` = auto |
| `-strassen-threshold` | Strassen threshold (bits), `0` = auto |
| `-calibrate` / `-auto-calibrate` | Full calibration / startup calibration |
| `-calibration-profile` | Profile path override |
| `-tui` | Launch TUI mode |
| `-calculate` (`-c`) | Print value |
| `-details` (`-d`) | Show metadata/perf details |
| `-verbose` (`-v`) | Full value output |
| `-quiet` (`-q`) | Minimal output |
| `-output` (`-o`) | Write result to file |
| `-completion` | Shell completion script |
| `--last-digits` | Modular computation mode |
| `--memory-limit` | Memory budget guard |
| `--gc-control` | `auto` / `aggressive` / `disabled` |

## Environment variable overrides (`FIBCALC_` prefix)

Implemented with precedence: **CLI flags > env vars > adaptive estimation > static defaults**.

Supported keys include:

- `FIBCALC_N`, `FIBCALC_ALGO`, `FIBCALC_TIMEOUT`
- `FIBCALC_THRESHOLD`, `FIBCALC_FFT_THRESHOLD`, `FIBCALC_STRASSEN_THRESHOLD`
- `FIBCALC_VERBOSE`, `FIBCALC_DETAILS`, `FIBCALC_QUIET`, `FIBCALC_CALCULATE`
- `FIBCALC_CALIBRATE`, `FIBCALC_AUTO_CALIBRATE`, `FIBCALC_CALIBRATION_PROFILE`
- `FIBCALC_OUTPUT`, `FIBCALC_MEMORY_LIMIT`, `FIBCALC_TUI`

Also honors standard `NO_COLOR` behavior.

## Calibration profiles
- File-backed JSON profile (`~/.fibcalc_calibration.json` by default).
- Stores hardware signature and tuned thresholds:
  - parallel
  - FFT
  - Strassen
- Validity checks include profile version, CPU count, arch, and word size.

---

## 9) Error Handling

## Typed errors

| Type | Purpose |
|---|---|
| `ConfigError` | Invalid configuration/flags/parameters |
| `TimeoutError` | Operation exceeded duration limit |
| `MemoryError` | Requested memory exceeds available/configured constraints |
| `ValidationError` | Structured field-level validation failures |
| `CalculationError` | Wraps underlying computation failure cause |

Additional helpers: `WrapError`, `IsContextError`.

## Exit codes

| Code | Constant | Meaning |
|---:|---|---|
| `0` | `ExitSuccess` | Success |
| `1` | `ExitErrorGeneric` | Generic/unexpected error |
| `2` | `ExitErrorTimeout` | Timeout |
| `3` | `ExitErrorMismatch` | Cross-algorithm result mismatch |
| `4` | `ExitErrorConfig` | Configuration error |
| `130` | `ExitErrorCanceled` | Canceled (signal/context) |

`HandleCalculationError` maps timeout/cancel/generic failures into standardized user-facing messaging + exit status.

---

## 10) Testing Strategy

FibCalc uses a layered approach:

- **Unit tests:** extensive table-driven tests across internal packages.
- **Golden file tests:** canonical expected Fibonacci outputs (`internal/fibonacci/testdata/fibonacci_golden.json`), plus CLI output goldens.
- **Fuzz testing:** Go fuzzing for cross-algorithm consistency, identities, monotonic progress, modular arithmetic.
- **Property-based tests:** `gopter` checks mathematical invariants (e.g., Cassini identity).
- **Benchmarks:** algorithm and subsystem benchmarks with alloc stats and profiling hooks.
- **Race detector:** standard test invocation includes `-race`.
- **E2E tests:** build and execute binary subprocesses in `test/e2e`.

Typical commands:

```bash
go test -v -race -cover ./...
go test -bench=. -benchmem ./internal/fibonacci/
go test -fuzz=FuzzFastDoublingConsistency ./internal/fibonacci/
```

---

## 11) Build System

The project uses standard Go tooling + Makefile workflows.

## Key Make targets

- Build: `build`, `build-all`, `build-linux`, `build-windows`, `build-darwin`
- Test/quality: `test`, `test-short`, `coverage`, `benchmark`, `lint`, `security`, `check`
- Dev hygiene: `format`, `tidy`, `deps`, `upgrade`
- PGO: `pgo-profile`, `pgo-check`, `build-pgo`, `build-pgo-all`, `pgo-rebuild`, `pgo-clean`

## PGO support
- Profile path: `cmd/fibcalc/default.pgo`
- `make pgo-profile` generates profile from benchmark workload.
- `make build-pgo` compiles with `-pgo=...`.

## Cross-compilation
- Supported in Makefile for Linux/Windows/macOS (`amd64`, plus `arm64` for Darwin).

## GMP build tag
- Optional calculator in `internal/fibonacci/calculator_gmp.go`.
- Build with:

```bash
go build -tags=gmp -o fibcalc ./cmd/fibcalc
```

- Auto-registers a `gmp` algorithm at init time when tag is enabled.

---

## 12) External Dependencies (direct)

From `go.mod`, direct dependencies are:

| Module | Purpose in FibCalc |
|---|---|
| `golang.org/x/sync` | `errgroup` for structured concurrent execution |
| `github.com/briandowns/spinner` | CLI spinner UX |
| `github.com/charmbracelet/bubbles` | Bubble Tea UI components |
| `github.com/charmbracelet/bubbletea` | TUI framework (Elm architecture runtime) |
| `github.com/charmbracelet/lipgloss` | Terminal styling/theme for TUI |
| `github.com/leanovate/gopter` | Property-based testing |
| `github.com/ncw/gmp` | Optional GMP big integer backend (`gmp` build tag) |
| `github.com/rs/zerolog` | Structured logging |
| `github.com/shirou/gopsutil/v4` | Host/system metrics collection |
| `golang.org/x/sys` | Low-level OS/CPU support (including CPU feature usage) |

---

## Appendix: Architectural Notes for New Engineers

- Start from `cmd/fibcalc/main.go` and trace into `internal/app`.
- For execution semantics, read `internal/orchestration` first.
- For algorithm internals, focus on:
  1. `internal/fibonacci/fastdoubling.go`
  2. `internal/fibonacci/matrix_framework.go` + `matrix_ops.go`
  3. `internal/fibonacci/fft.go` + `internal/bigfft`
- For user interaction, study `internal/cli` and `internal/tui` presenters.
- For operational tuning, use `docs/CALIBRATION.md`, `docs/PERFORMANCE.md`, and Makefile PGO targets.

This architecture intentionally emphasizes separation of concerns, algorithmic interchangeability, and performance-tuning hooks while keeping orchestration and presentation decoupled.