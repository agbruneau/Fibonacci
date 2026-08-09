# FibGo / FibCalc Architecture

> **Ce document est la vue d'ensemble rapide** de l'architecture de FibCalc. Pour la référence détaillée (diagrammes C4, flows Mermaid, index complet de la documentation), voir **[docs/architecture/README.md](architecture/README.md)**.

> **Vue interactive** — Un dashboard navigable du graphe de connaissances (1 128 nœuds, 4 782 arêtes, 9 couches, tour guidé 12 étapes — comptés dans l'artefact lui-même) est publié sur **[agbruneau.github.io/Fibonacci/dashboard/](https://agbruneau.github.io/Fibonacci/dashboard/)**. Complément visuel à ce document statique. **Artefact daté** : le graphe décrit l'arbre au commit `6e3ec29` (2026-07-04) et n'a pas été régénéré depuis le 2026-07-06 — soit 126 commits de retard sur le HEAD `707044d`, 204 fichiers modifiés entre-temps, 14 chemins qu'il référence disparus (dont les packages supprimés `internal/parallel`, `internal/metrics/system` et `internal/fibonacci/fibonaccitest`) et 37 fichiers suivis absents du graphe. À lire comme la carte de l'architecture au 2026-07-04, non de l'état courant. Source : [`docs/dashboard/knowledge-graph.json`](dashboard/knowledge-graph.json), build statique : [`docs/dashboard/`](dashboard/).

## 1) Project Overview

**FibGo** (module/library name: **FibCalc**) is a high-performance Fibonacci computation system implemented in Go.

- **Go module path:** `github.com/agbruneau/FibGo`
- **Go version:** 1.26.0+ (`go.mod` declares `go 1.26.0`, no `toolchain` directive)
- **Primary binary:** `cmd/fibcalc`
- **Codebase stats:** run `make stats` for the canonical Go-package and LOC counts (the totals drift on every refactor; encoding them statically here has historically caused divergence between this document and reality).
- **Purpose:** compute very large Fibonacci values efficiently, compare multiple algorithms, and expose both CLI and TUI execution modes.
- **Core strengths:**
  - Multiple `O(log n)` Fibonacci algorithms (Fast Doubling, Matrix Exponentiation, FFT-Based Doubling)
  - Adaptive multiplication strategy (`math/big` Karatsuba vs FFT) with configurable thresholds
  - Optional GMP backend via build tag
  - Runtime calibration/adaptive thresholds with micro-benchmark support
  - Concurrency-aware orchestration with multi-level parallelism and progress reporting
  - Memory management via arena allocators, object pools, and GC control
  - Modular arithmetic mode (`--last-digits`) for O(K) memory computations

At runtime, FibCalc can execute one or many calculators in parallel, aggregate progress, validate result consistency across algorithms, and present results through CLI or TUI presentation layers.

---

## 2) High-Level Architecture (Clean Architecture)

FibCalc follows **Clean Architecture** principles with strict unidirectional dependency flow: outer layers depend on inner layers, never the reverse. The orchestration layer defines interfaces (`ProgressReporter`, `ResultPresenter`) that presentation layers implement, ensuring the business logic never imports UI code.

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
| internal/fibonacci  internal/progress  internal/bigfft                    |
| (algorithms)        (observer model)   (FFT arithmetic)                    |
|   fibonacci/memory   fibonacci/threshold                                   |
|   (arena, GC ctrl)   (dynamic tuning)                                      |
+-----------------------------------------------------------------------+

Cross-cutting leaves — NOT a layer below the Domain. No Domain package
imports them; the arrows come sideways, from Interfaces and Application:

+-----------------------------------------------------------------------+
|                       Shared Utility Packages                         |
|                                                                       |
| internal/metrics ← internal/cli, internal/tui                         |
| internal/format  ← internal/cli, internal/tui, internal/calibration   |
+-----------------------------------------------------------------------+
```

Importers verified on HEAD with
`go list -f '{{join .Imports " "}}' ./internal/<pkg>` over every package of
the four layers: only `cli`, `tui` and `calibration` come back. `test/e2e` and
`docs/` are consumers of the binary and prose about it — neither is an import
edge and neither belongs in this diagram.

### Dependency Rules

- **Interfaces layer** → imports Application, Use-Case *and* the shared Domain
  leaves. `internal/cli` and `internal/tui` both import `internal/progress`
  directly (they consume `progress.ProgressUpdate` off the channel), but
  neither imports `internal/fibonacci`: domain types reach them through the
  `orchestration.Calculator`/`Options` aliases — enforced by
  `internal/arch_test.go` for `tui`.
- **Application layer** → imports Use-Case + Domain layers, **plus** the
  Interfaces layer. `internal/app` is the composition root: it imports
  `internal/cli`, `internal/tui` and `internal/ui` in order to wire them, and
  that downward-looking rule does not apply to it. No other package in this
  layer does (`internal/config` and `internal/calibration` import `ui` for
  colored output and nothing else from the Interfaces layer).
- **Use-Case layer** → `internal/orchestration` imports exactly
  `internal/errors`, `internal/fibonacci`, `internal/fibonacci/memory` and
  `internal/progress` — Domain plus the `errors` leaf, never a presentation
  package.
- **Domain layer** → no imports from outer layers (self-contained).
  `internal/fibonacci/threshold` does **not** import `internal/config` ; the
  application layer wires `threshold.Tuning` via `threshold.SetTuning`.
  `internal/bigfft` imports no internal package at all.
- **Infrastructure** → utility packages with no upward dependencies.
  `internal/errors` ships its own byte-formatter (`formatBytesLocal`) instead
  of depending on `internal/format`.

`internal/arch_test.go` fails `go test` if any of **six** upward arrows is
reintroduced — a **test** gate, not a compile gate: the package
`github.com/agbruneau/FibGo/internal` has `GoFiles = []` and
`XTestGoFiles = [arch_test.go]`, so `go build ./...` compiles none of it and
passes regardless. They are grouped into **five** rules (`architectureRules`, one
subtest each): `threshold → config`, `errors → format` and `tui → fibonacci`
(May-2026 hardening sprint), `orchestration → format` (July-2026, APP-10), and
— as the two targets of the fifth and last rule —
`config → fibonacci` / `config → bigfft` (audit Fable5, ARCH-02 — the two
tolerated lateral imports `config → fibonacci/memory` and `config → ui` stay
allowed). Its package doc comment (the `internal_test` package comment in `internal/arch_test.go`) states the same
chain as this section: `cmd → app → orchestration → fibonacci → bigfft`, with
`config` a sibling of `orchestration` rather than a layer beneath `fibonacci`.

---

## 3) Directory Structure

### Top-level tree (annotated)

```text
.
├── cmd/
│   ├── fibcalc/                 # Main application entrypoint
│   └── generate-golden/         # Golden-data generator for tests
├── internal/                    # top-level packages (run `make stats` for the authoritative count)
├── test/
│   └── e2e/                     # End-to-end CLI tests
├── docs/                        # Architecture, algorithm, build, test, perf docs
│   ├── architecture/            # C4 diagrams, flow diagrams, patterns, README
│   └── algorithms/              # Algorithm deep-dive documents
├── .env.example                 # Supported FIBCALC_* env variables
├── .golangci.yml                # Linter configuration
├── go.mod                       # Module + direct dependencies
├── Makefile                     # Build/test/lint/PGO/cross-compile workflows
├── README.md                    # Product and usage overview
├── CONTRIBUTING.md              # Development guidelines
└── CHANGELOG.md                 # Version history
```

### `internal/` package map

```text
internal/
├── app/                         # Lifecycle, mode dispatch, version, DI via WithFactory()
├── bigfft/                      # FFT multiplication engine for big.Int
│   ├── fft.go                   # Public API: Mul, MulTo, Sqr, SqrTo
│   ├── fft_core.go              # Core FFT algorithm
│   ├── fft_recursion.go         # Recursive FFT decomposition, parallelism config
│   ├── fft_poly.go              # Polynomial operations
│   ├── fft_cache.go             # FFT transform caching
│   ├── memory_est.go            # Memory estimates for transforms
│   ├── fermat.go                # Fermat ring arithmetic (Z/(2^k+1))
│   ├── pool.go, pool_warming.go # Size-class pools, adaptive pre-warming
│   ├── allocator.go, bump.go    # Memory allocators (bump allocator)
│   ├── arith_decl.go            # go:linkname declarations into math/big
│   └── arith.go                 # AddVV/SubVV/AddMulVVW wrappers (no build-tag split)
├── calibration/                 # Threshold benchmarking + profile persistence
├── cli/                         # CLI output/presenter/spinner
│   └── completion/              # Shell completion generators (bash/zsh/fish/powershell)
├── config/                      # Flag parsing, env override, adaptive thresholds
├── errors/                      # Typed app errors + exit code handling
├── fibonacci/                   # Core Fibonacci algorithms + framework/strategy/factory
│   ├── memory/                  # Arena allocator, GC control, memory budget
│   └── threshold/               # Dynamic threshold manager
├── format/                      # Duration/number/progress ETA formatting
├── metrics/                     # Runtime performance/memory indicators
├── orchestration/               # Concurrent execution and result analysis
├── progress/                    # Observer pattern (subject/observers/update model)
├── testutil/                    # Shared test helpers
├── tui/                         # Bubble Tea interactive dashboard
└── ui/                          # Themes/colors/NO_COLOR behavior
```

---

## 4) Core Packages (Responsibilities, Key Types, Interfaces)

## `internal/app`
- **Responsibility:** startup + runtime mode orchestration (completion, calibration, TUI, normal calculation).
- **Key types/functions:** `Application`, `New`, `Run`, `runCalculate`, `runTUI`, `runCalibration`, `runLastDigits`, `runAutoCalibrationIfEnabled`.
- **DI support:** `AppOption` functional options, `WithFactory()` for injecting custom `CalculatorFactory`.
- **Lifecycle flow:** `New()` → parse config → load calibration profile (or apply adaptive thresholds) → `Run()` → mode dispatch.

## `internal/config`
- **Responsibility:** parse CLI flags, validate configuration, apply `FIBCALC_` env overrides, apply adaptive thresholds.
- **Key types:** `AppConfig` (20 fields covering all runtime parameters), `HardwareHeuristic` / `SIMDKind` (CPU class for default thresholds).
- **Key functions:** `ParseConfig`, `ApplyAdaptiveThresholds`, `DetectHardwareHeuristic`, `EstimateOptimalParallelThreshold`, `EstimateOptimalFFTThreshold`, `EstimateOptimalStrassenThreshold`. The per-heuristic variants `estimateParallelThresholdForHeuristic` / `estimateFFTThresholdForHeuristic` / `estimateStrassenThresholdForHeuristic` (`internal/config/thresholds.go`, the three `estimate*ThresholdForHeuristic` functions) are **unexported** — reachable only from in-package tests, not from diagnostics outside `internal/config`.
- **Precedence chain:** CLI flags > env vars (`applyEnvOverrides` skips any flag explicitly set on the command line, `internal/config/env.go:applyEnvOverrides`) > static defaults — **for every setting except the three thresholds**. `--threshold`, `--fft-threshold` and `--strassen-threshold` are *overridden* by a valid cached calibration profile; see [§9 Configuration and Environment](#9-configuration-and-environment).

## `internal/calibration`
- **Responsibility:** full/quick calibration, adaptive threshold candidate generation, micro-benchmarks, profile file persistence.
- **Key types:** `CalibrationProfile`, `CalibrationOptions`, `MicroBenchmark`, `ThresholdResults` (per-pass rows use the unexported `calibrationResult`).
- **Key functions:** `RunCalibration`, `AutoCalibrate`, `AutoCalibrateWithProfile`, `LoadCachedCalibration`, `LoadOrCreateProfile`, `SaveProfile`, `QuickCalibrate`, `GenerateParallelThresholds`.
- **Three-tier calibration:** (1) cached profile → (2) quick micro-benchmarks (`FastStrategy`; the source states ~100 ms as its design target — `internal/calibration/microbench.go`, its file comment and the `QuickCalibrate` doc comment — no measurement artifact in the repo) → (3) full benchmark with adaptive threshold search (`CompleteStrategy`).

## `internal/orchestration`
- **Responsibility:** execute calculators concurrently, collect durations/errors/results, compare consistency, present summary.
- **Key types:** `CalculationResult`, `PresentationOptions`, `ProgressAggregator`.
- **Key interfaces:**
  - `ProgressReporter` — displays progress (implemented by `TUIProgressReporter`, `NullProgressReporter`, and the `ProgressReporterFunc` adapter wrapping `cli.DisplayProgress` for the CLI)
  - `ResultPresenter` — formats results (implemented by `CLIResultPresenter`, `TUIResultPresenter`)
  - `ErrorHandler` — maps errors to exit codes
- **Concurrency model:** single-calculator fast path (no errgroup overhead) vs multi-calculator errgroup fan-out.

## `internal/fibonacci`
- **Responsibility:** domain algorithms, strategy selection, factory/registry, pooled state, framework loops, modular arithmetic.
- **Key interfaces (layered by scope):**
  - `Calculator` (public) — full calculation with context, progress channel, options
  - `CoreCalculator` (exported extension point) — pure algorithm computation with callback-based progress
  - `CalculatorFactory` — Create/Get/Register/List/GetAll for calculator management
  - `Multiplier` (narrow ISP) — multiply/square only
  - `DoublingStepExecutor` (wide) — extends Multiplier with full doubling-step awareness
- **Key types:**
  - `FibCalculator` (decorator) — wraps CoreCalculator with GC control, FFT cache config, pool warming, small-N fast path, observer adaptation
  - `FastDoublingCalculator` — Fast Doubling O(log n) with parallel multiplication; holds a per-instance GC-immune `cachedState` slot (`atomic.Pointer`, arenas ≤ 4M words) consulted before the shared `sync.Pool`
  - `MatrixExponentiationCalculator` — Matrix exponentiation O(log n) with Strassen dispatch
  - `FFTBasedCalculator` — FFT-only multiplication for benchmark/large-N scenarios
  - `Options` — comprehensive configuration (thresholds, FFT cache, dynamic thresholds, GC mode)
  - `CalculationState` — pooled 5-variable state (FK, FK1, T1-T3) for doubling algorithms
  - `DefaultFactory` — thread-safe factory with lazy creation, double-check locking, caching

### `internal/fibonacci/memory`
- **Responsibility:** memory management during large computations.
- **Key types/functions:**
  - `CalculationArena` — contiguous bump-style arena with `PreSizeFromArena` for state big.Int
  - `GCController` (`auto`/`aggressive`/`disabled`) — disables GC for N ≥ 1M, uses `debug.SetMemoryLimit` as OOM safety net
  - `EstimateMemoryUsage`, `ParseMemoryLimit`, `FormatMemoryEstimate`

### `internal/fibonacci/threshold`
- **Responsibility:** dynamic runtime threshold adjustment based on observed iteration performance.
- **Key types:** `DynamicThresholdManager`, `DynamicThresholdConfig`, `IterationMetric`, `ThresholdAnalyzer`.
- **Mechanism:** records per-iteration timing data, detects if FFT/parallel thresholds should be adjusted, returns new thresholds mid-computation.

## `internal/progress`
- **Responsibility:** Observer pattern for progress updates, decoupled from fibonacci package.
- **Key types/interfaces:** `ProgressObserver` (interface), `ProgressSubject` (observable with `Register`, `Freeze`, `ObserverCount`), `ProgressUpdate` (DTO), `ProgressCallback` (functional type).
- **Implementations:** `ChannelObserver`, `LoggingObserver`, `NoOpObserver`.
- **Optimization:** `Freeze()` creates a lock-free snapshot to avoid lock acquisition in hot computation loops.

## `internal/bigfft`
- **Responsibility:** high-performance FFT-based multiplication/squaring for `big.Int`.
- **Key APIs:** `Mul`, `MulTo`, `Sqr`, `SqrTo`.
- **Subsystems:**
  - FFT recursion with configurable parallelism (`FFTParallelismConfig`)
  - Transform cache for reuse across operations
  - Size-class object pools with adaptive pre-warming
  - Bump allocator for batch temporary allocations
  - Fermat ring arithmetic (`Z/(2^k+1)`) with `smallMulThreshold` cutover
  - Architecture-neutral arithmetic via `go:linkname` to `math/big` internals
    (unconditional declarations in `arith_decl.go`; this package performs **no**
    CPU-feature probing — `golang.org/x/sys/cpu` is read only by
    `internal/config/hardware.go`, for threshold heuristics)

## `internal/cli`
- **Responsibility:** terminal UX for non-TUI mode (progress, table/result output, shell completion).
- **Key components:** `CLIResultPresenter`, `CLIColorProvider`, `DisplayProgress` (wrapped by `orchestration.ProgressReporterFunc`), `DisplayQuietResult`, `WriteResultToFile`, `GenerateCompletion`.

## `internal/tui`
- **Responsibility:** Bubble Tea Elm-style dashboard (`Model-Update-View`) for interactive execution.
- **Sub-models:** Header (title, version, elapsed), Chart (progress bar, ETA, sparklines), Metrics (memory, heap, GC, goroutines), Logs (scrollable viewport), Footer (keymap, status).
- **Integration:** `TUIProgressReporter` and `TUIResultPresenter` implement orchestration interfaces.
- **Theme:** Orange-dominant dark palette with lipgloss rounded borders.

## `internal/errors`
- **Responsibility:** typed errors, wrappers, exit code mapping, standardized calculation-error handling.
- **Key types:** `ConfigError`, `CalculationError`, `MemoryError` (timeout/cancellation are classified via `errors.Is` on context sentinels, not dedicated types — OVR-07).
- **Key helpers:** `NewConfigError`, `WrapCalculationError`, `HandleCalculationError`, `ColorProvider` interface.

## `internal/metrics`, `internal/format`, `internal/ui`, `internal/testutil`
- **Responsibility:** telemetry formatting, performance indicators (throughput, O(1) properties), theming/color controls (`NO_COLOR` support), test helpers. Host CPU/memory sampling is inlined in `internal/tui` (its only consumer — audit Fable5 DEAD-05).

---

## 5) Design Patterns (14 patterns)

| Pattern | Where | Why it exists |
|---|---|---|
| **Decorator** | `fibonacci.FibCalculator` wrapping `CoreCalculator` | Adds cross-cutting behavior (small-N fast path, observer adaptation, GC control, FFT cache config, pool warming) without changing algorithm cores |
| **Strategy** | `Multiplier` / `DoublingStepExecutor` with `AdaptiveStrategy`, `FFTOnlyStrategy` | Enables swapping multiplication policy by workload/benchmark intent |
| **Interface Segregation (ISP)** | `Multiplier` (narrow: Multiply/Square) vs `DoublingStepExecutor` (wide: +ExecuteStep) | Consumers needing only multiply/square depend on the narrow interface; framework-level consumers use the wide one |
| **Observer** | `progress.ProgressSubject` + `ProgressObserver` implementations | Decouples progress production from UI/log consumers; supports multiple simultaneous observers |
| **Factory + Registry** | `DefaultFactory` implementing `CalculatorFactory` | Centralized calculator registration/lookup/caching with lazy creation and double-check locking |
| **Framework (Template Method)** | `DoublingFramework`, `MatrixFramework` | Owns algorithm loop (bit iteration, progress reporting, context checks) while plugging in operation strategy/threshold behavior |
| **Object Pool** | `sync.Pool` in Fibonacci state and `bigfft` pools | Cuts allocations and GC pressure in hot paths; size-limited via `MaxPooledBitLen` (50M bits) |
| **Arena Allocator** | `memory.CalculationArena` | Pre-sizes contiguous backing storage for big.Int state to reduce fragmentation/GC overhead |
| **Bump Allocator** | `bigfft.BumpAllocator` | Batch temporary allocations with O(1) reset for FFT internals |
| **FFT Transform Cache** | `internal/bigfft/fft_cache.go` | Caches FFT transforms for reuse across multiply/square operations within an iteration |
| **Dynamic Threshold Adjustment** | `threshold.DynamicThresholdManager` | Records per-iteration metrics and adjusts FFT/parallel thresholds mid-computation |
| **Zero-Copy Result Return** | `MatrixFramework.ExecuteMatrixLoop` only | "Steals" `res.a` from the matrix state instead of copying. Deliberately NOT done in `DoublingFramework.ExecuteDoublingLoop` (P1-04): its state aliases the arena, so the success path deep-copies via `ReleaseStateWithResult` |
| **Generics with Pointer Constraints** | `executeTasks[T any, PT interface{*T; task}]` | Generic task execution eliminating code duplication between multiplication and squaring tasks |
| **GC Controller** | `memory.GCController` | Disables GC during large computations (N ≥ 1M), restores afterward; uses `debug.SetMemoryLimit` as safety net |

Additional notable engineering patterns include:
- **Runtime-configurable threshold heuristics** with adaptive estimation based on CPU count
- **Channel-based progress aggregation** with buffered channels (`ProgressBufferMultiplier = 5`)
- **Lock-free observer snapshots** via `ProgressSubject.Freeze()` for hot loop performance
- **Semaphore-based concurrency limiting** — two separate semaphores, sized differently: the Fibonacci task semaphore (`fibonacci/common.go` `getTaskSemaphore`) is `runtime.GOMAXPROCS(0)`, the FFT recursion semaphore (`bigfft/fft_recursion.go` `getSemaphore`) is `runtime.NumCPU()`
- **Functional options** pattern for `Application` construction (`AppOption`, `WithFactory`)
- **Function adapter** pattern for `ProgressReporterFunc`

---

## 6) Data Flow (CLI input to final result)

### Complete Execution Flow

```text
┌───────────────────────────────────────────────────────────────────┐
│ 1. ENTRY POINT                                                    │
│    cmd/fibcalc/main.go → run(args, stdout, stderr)               │
│    ├─ Version flag check → HasVersionFlag → PrintVersion → exit  │
│    └─ app.New(args, stderr) → Application instance               │
├───────────────────────────────────────────────────────────────────┤
│ 2. CONFIG RESOLUTION                                              │
│    config.ParseConfig(name, args, errWriter, availableAlgos)     │
│    ├─ Flag parsing (flag.NewFlagSet with ContinueOnError)        │
│    ├─ applyEnvOverrides() for FIBCALC_* environment variables    │
│    ├─ Algo normalization (strings.ToLower)                       │
│    └─ config.Validate(availableAlgos) → semantic checks          │
├───────────────────────────────────────────────────────────────────┤
│ 3. THRESHOLD RESOLUTION (overrides step 2 for the 3 thresholds)   │
│    calibration.LoadCachedCalibration(cfg, profilePath)           │
│    ├─ IF profile valid AND cfg still validates → OVERWRITE       │
│    │    Threshold/FFTThreshold/StrassenThreshold with the        │
│    │    profile's, discarding any CLI flag or FIBCALC_* value    │
│    └─ ELSE → config.ApplyAdaptiveThresholds(cfg)                 │
│         ├─ EstimateOptimalParallelThreshold() (CPU-based)        │
│         ├─ EstimateOptimalFFTThreshold() (CPU-based)             │
│         └─ EstimateOptimalStrassenThreshold() (CPU-based)        │
├───────────────────────────────────────────────────────────────────┤
│ 4. MODE DISPATCH (Application.Run)                                │
│    ├─ Completion mode → cli.GenerateCompletion → exit            │
│    ├─ Calibration mode → calibration.RunCalibration → exit       │
│    ├─ Auto-calibration → calibration.AutoCalibrate → update cfg  │
│    ├─ TUI mode → tui.Run(ctx, calculators, cfg, version, errOut) │
│    └─ CLI mode → runCalculate(ctx, out) [default]                │
├───────────────────────────────────────────────────────────────────┤
│ 5. LIFECYCLE SETUP (for CLI/TUI modes)                            │
│    ├─ Last-digits mode → runLastDigits (dedicated path)          │
│    ├─ Memory budget validation (if --memory-limit set)           │
│    ├─ context.WithTimeout(cfg.Timeout) → deadline context        │
│    └─ signal.NotifyContext(SIGINT, SIGTERM) → cancellation       │
├───────────────────────────────────────────────────────────────────┤
│ 6. CALCULATOR SELECTION                                           │
│    orchestration.GetCalculatorsToRun(algo, factory)              │
│    ├─ algo="all" → factory.List() then factory.Get(k) per key    │
│    └─ algo=specific → factory.Get(algo) → single calculator     │
├───────────────────────────────────────────────────────────────────┤
│ 7. CONCURRENT EXECUTION                                           │
│    orchestration.ExecuteCalculations(ctx, ExecutionConfig{...})  │
│    ├─ Progress channel: make(chan, numCalcs * 5)                  │
│    ├─ Progress goroutine: reporter.DisplayProgress(wg, ch, ...)  │
│    ├─ Single calculator: direct call (no errgroup overhead)      │
│    └─ Multiple calculators: errgroup fan-out                     │
│         └─ Per calculator:                                        │
│             ├─ Calculator.Calculate(ctx, progCh, idx, n, opts)   │
│             ├─ → ProgressSubject + ChannelObserver registration   │
│             ├─ → CalculateWithObservers (source order)            │
│             │    ├─ subject.Freeze(calcIndex) → lock-free reporter│
│             │    ├─ Small-N fast path (n ≤ 93 → iterative add)   │
│             │    ├─ configureFFTCache(opts, n)                    │
│             │    ├─ bigfft.EnsurePoolsWarmed(n)                   │
│             │    └─ gcCtrl.WithGC(fn) — panic-safe GC control     │
│             │         (auto: GC off for N≥1M, restored after)     │
│             │         wrapping core.CalculateCore(ctx, ...)       │
│             └─ CalculationResult{Name, Result, Duration, Err}    │
├───────────────────────────────────────────────────────────────────┤
│ 8. ALGORITHM CORE (inside CalculateCore)                          │
│    Fast Doubling:                                                 │
│    ├─ fd.acquireStateForN(n) → CalculationState                   │
│    │    (GC-immune cachedState slot first, sync.Pool fallback;    │
│    │     state-bound arena reused/grown + PreSizeFromArena)       │
│    ├─ Create DoublingFramework(AdaptiveStrategy)                  │
│    │   (optional: with DynamicThresholdManager)                   │
│    └─ ExecuteDoublingLoop(ctx, reporter, n, opts, state, parallel)│
│         ├─ Bit iteration: MSB → LSB                              │
│         ├─ shouldParallelizeMultiplicationCached() decision,      │
│         │   computed HERE and passed to ExecuteStep as inParallel │
│         ├─ Per bit: ExecuteStep (3 multiplications)              │
│         │   ├─ Parallel: executeParallel3 (3 goroutines)         │
│         │   └─ Sequential: with ctx.Err() checks between ops    │
│         ├─ Post-multiply: F(2k) = 2·T3 - T2, F(2k+1) = T1 + T2│
│         ├─ Pointer rotation (zero-copy)                          │
│         ├─ Addition step: if bit=1, F(k) ← F(k+1), F(k+1) ← sum│
│         ├─ Dynamic threshold adjustment (if enabled)             │
│         └─ ReportStepProgress (geometric work model)              │
├───────────────────────────────────────────────────────────────────┤
│ 9. RESULT ANALYSIS                                                │
│    orchestration.AnalyzeComparisonResults(results, presOpts, ...) │
│    ├─ Sort by: success first, then by duration ascending         │
│    ├─ PresentComparisonTable(results, out) → formatted table     │
│    ├─ Consistency check: compare all Result values (big.Int.Cmp) │
│    ├─ Mismatch → ExitErrorMismatch (code 3)                      │
│    └─ Success → PresentResult(best, n, verbose, details, ...)    │
├───────────────────────────────────────────────────────────────────┤
│ 10. OUTPUT & EXIT                                                 │
│     ├─ Optional file output: WriteResultToFile (if -o set)       │
│     ├─ Quiet mode: DisplayQuietResult (minimal output)           │
│     └─ Error mapping → exit codes (0, 1, 2, 3, 4, 130)          │
└───────────────────────────────────────────────────────────────────┘
```

### Concurrency Model (3 levels)

```text
Level 1: Algorithm-level parallelism
   └─ errgroup fan-out: each calculator runs in its own goroutine
      (single calculator: direct call, no errgroup overhead)

Level 2: Intra-algorithm operation parallelism
   └─ executeParallel3(): 3 goroutines for doubling step multiplications
      (the 3 goroutines are SPAWNED first; each acquires its token inside
       runParallel3Op, so the semaphore throttles work, not spawning)
      └─ Controlled by shouldParallelizeMultiplicationCached(), computed in
         ExecuteDoublingLoop BEFORE ExecuteStep and passed down as inParallel:
         ├─ Enabled when: operand > ParallelThreshold (default: 4096 bits)
         ├─ Suppressed when: FFT active (FFT saturates CPU cores)
         └─ Re-enabled when: operand > ParallelFFTThreshold (5M bits)
      └─ Semaphore: runtime.GOMAXPROCS(0) concurrent goroutines max
         (shared with executeTasks / executeMixedTasks on the matrix path)

Level 3: FFT internal parallelism
   └─ bigfft recursive decomposition + pointwise chunking: configurable limit
      └─ Semaphore: runtime.NumCPU() concurrent goroutines max, acquired
         NON-BLOCKING — a chunk with no free token runs on the caller
      └─ Total system: up to GOMAXPROCS(0) + NumCPU simultaneous goroutines
         (equal by default; mitigated by Level 2 suppression during FFT)
```

### Progress Propagation Flow

```text
FibCalculator.CalculateWithObservers
   └─ reporter := subject.Freeze(calcIndex)   [copies the observer slice once,
       │                                       returns a lock-free closure of
       │                                       type progress.ProgressCallback;
       │                                       there is no FrozenProgressSubject
       │                                       type — the closure IS the snapshot]
       └─ handed to CoreCalculator.CalculateCore as `reporter`
           └─ Core Algorithm calls reporter(float64)
               [per-iteration, throttled by ReportStepProgress to ≥1% change,
                with the first and last bit always reported]
               └─ ChannelObserver.Update → progressChan (buffered, numCalcs*5)
                   └─ ProgressReporter goroutine (started BEFORE the calculators)
                       ├─ CLI: spinner + ETA + progress percentage
                       └─ TUI: TUIProgressReporter → programRef.Send(ProgressMsg)
```

---

## 7) Algorithm Layer

### A. Fast Doubling (`FastDoublingCalculator`)
- **Complexity:** O(log n) arithmetic operations; total: O(log n × M(n)) where M(n) is multiplication cost
- Core identities (derived from Q-matrix squaring):
  - `F(2k)   = F(k) * (2F(k+1) - F(k))`
  - `F(2k+1) = F(k+1)² + F(k)²`
- Uses `DoublingFramework` + `AdaptiveStrategy`.
- Employs pooled `CalculationState` (5 big.Int + bound `CalculationArena`), memory arena pre-sizing, and optional dynamic threshold updates.
- **Result detachment:** `ReleaseStateWithResult` deep-copies the result out of the arena (~850 KB for F(10M): ⌈10e6 × 0.69424⌉ bits ÷ 8; the repo carries no measurement of that copy's share of runtime) so the arena can safely be reset and reused on the next acquisition. The previous "steal `s.FK`" zero-copy trick was dropped because it left the result aliasing pooled memory the next tenant would overwrite.

### B. Matrix Exponentiation (`MatrixExponentiationCalculator`)
- Uses binary exponentiation of Fibonacci Q-matrix: `[[1,1],[1,0]]^(n-1)`.
- `MatrixFramework` drives loop (LSB → MSB iteration over the bits of `n-1`).
- The `result × base` multiply switches between naive 2×2 multiply and Strassen
  based on `StrassenThreshold`. The **squaring** path does not: it goes straight
  from `squareSymmetricMatrix` to `smartSquare`/`smartMultiply` and never
  consults `StrassenThreshold`.
- Symmetric squaring is applied **unconditionally** — `MatrixFramework.SquareFunc`
  is wired to `squareSymmetricMatrix` at construction and called on every
  iteration but the last. There is no symmetry test and no "standard squaring"
  alternative in the code; `[a,b; b,d]` symmetry is an invariant of the Q-matrix
  powers, not a runtime condition. Cost: 3 squarings + 1 multiply instead of 4
  multiplies.
- The squaring is skipped on the final bit (`if i < numBits-1`).
- **Zero-copy result return:** steals `res.a` from matrix state. Matrix exponentiation does not use the state-bound arena, so the steal trick is still safe here.

### C. FFT-Based Doubling (`FFTBasedCalculator`)
- Same doubling loop model (via `DoublingFramework`), but strategy is `FFTOnlyStrategy`.
- Every doubling step routes to `executeDoublingStepFFT` with no threshold test,
  regardless of operand size. It also passes `useParallel = false` to
  `ExecuteDoublingLoop`, so its three per-step operations always run sequentially.
- Useful for benchmarking FFT performance and extremely large-input scenarios.

### D. Modular Fast Doubling (`FastDoublingMod`)
- Dedicated `--last-digits` mode: computes F(N) mod 10^K.
- Uses O(K) memory regardless of N.
- Same fast doubling identities applied modularly.

### Strategy System

```text
Multiplier (narrow interface)
   ├─ Multiply(z, x, y, opts) → (*big.Int, error)
   ├─ Square(z, x, opts) → (*big.Int, error)
   └─ Name() → string

DoublingStepExecutor (wide interface, extends Multiplier)
   └─ ExecuteStep(ctx, state, opts, inParallel) → error

Implementations:
   ├─ AdaptiveStrategy:  threshold-driven math/big vs FFT selection
   │     └─ ExecuteStep: if FFT-sized → executeDoublingStepFFT (transform reuse)
   │                     else → executeDoublingStepMultiplications (standard)
   └─ FFTOnlyStrategy:   ExecuteStep always routes to executeDoublingStepFFT,
                         with no threshold test at all. (Its Multiply/Square
                         call bigfft.MulTo/SqrTo — or mulFFT/sqrFFT when the
                         destination is nil — but ExecuteStep never invokes
                         them, so they are unreachable from the doubling loop.)
```

### `internal/bigfft` Role
- Provides efficient arithmetic primitives for huge operands (hundreds of millions of bits):
  - **Fermat ring arithmetic:** operations in Z/(2^k+1) for FFT kernel
  - **Recursive FFT decomposition** with configurable parallelism
  - **Transform caching:** consulted only by the `Mul`/`MulTo`/`Sqr`/`SqrTo`
    entry points (via `TransformCachedWithBump`), which the matrix path reaches
    through `smartMultiply`/`smartSquare`. No doubling loop touches it:
    `executeDoublingStepFFT` calls `TransformWithBump` directly.
  - **Size-class pools** with adaptive pre-warming based on estimated operand sizes
  - **Bump allocator** for batch temporary allocations with O(1) reset
  - **Architecture-neutral:** `go:linkname` to `math/big` internal word
    operations, declared unconditionally in `arith_decl.go` — no build-tag
    split, **no CPU-feature detection**, and no pure-Go fallback in this repo.
    The SIMD assembly exploited is `math/big`'s own, on every architecture.
- Public API used by Fibonacci layer via `Mul/MulTo/Sqr/SqrTo`.

---

## 8) Integration Patterns

### Factory and Registration Pattern

```text
NewDefaultFactory()
   ├─ Register("fast", → FastDoublingCalculator)
   ├─ Register("matrix", → MatrixExponentiationCalculator)
   └─ Register("fft", → FFTBasedCalculator)

init() [in calculator_gmp.go, build tag: gmp]
   └─ RegisterGMPCalculator(globalFactory) → Register("gmp", → GMPCalculator)

Get(name) → lazy creation + double-check locking cache
GetAll() → lazily initializes all, returns copy
```

### Configuration Cascade

Two different cascades, applied in this order.

**Everything except the three thresholds** — resolved entirely inside `ParseConfig`:

```text
CLI flags (highest priority)
   ↓ applyEnvOverrides(config, flagSet)  — skips any flag the user set explicitly
FIBCALC_* environment variables
   ↓
Static flag defaults (registerFlags)
```

**The three thresholds** (`Threshold`, `FFTThreshold`, `StrassenThreshold`) — `app.New`
runs a second stage *after* `ParseConfig`, and that stage wins:

```text
Valid cached calibration profile (~/.fibcalc_calibration.json, or
--calibration-profile / FIBCALC_CALIBRATION_PROFILE)     ← HIGHEST
   │  LoadCachedCalibration overwrites the three fields unconditionally;
   │  it reads neither the flag set nor the environment.
   │  Kept only if the resulting AppConfig still passes Validate().
   ↓ no valid profile → config.ApplyAdaptiveThresholds()
CLI flag / FIBCALC_* value, when non-zero (Apply* only fills zeros)
   ↓ still 0
CPU-adaptive estimation (runtime.NumCPU, GOARCH, x86 SIMD tier)
   ↓
Static defaults (in constants.go)
```

Verified 2026-08-07 on the binary: with a profile carrying
`optimal_parallel_threshold: 777777` / `optimal_fft_threshold: 888888` and a matching
`cpu_heuristic_key`, `fibcalc -n 100 -algo fast -d --threshold 4242 --fft-threshold 4243
--calibration-profile <p>` prints `Parallelism=777777 bits, FFT=888888 bits`. Change the
profile's `cpu_heuristic_key` so `IsValid()` fails and the same command prints
`Parallelism=4242 bits, FFT=4243 bits`. Sources: `internal/calibration/calibration.go:LoadCachedCalibration`,
`internal/app/app.go:New`.

### Presentation Layer Integration

```text
internal/orchestration (defines interfaces)
   ├─ ProgressReporter interface
   │    ├─ internal/cli/display.go → ProgressReporterFunc(DisplayProgress)
   │    ├─ internal/tui/bridge.go → TUIProgressReporter
   │    └─ NullProgressReporter (quiet mode, testing)
   └─ ResultPresenter interface
        ├─ internal/cli/presenter.go → CLIResultPresenter
        └─ internal/tui/bridge.go → TUIResultPresenter
```

### Calibration System Integration

```text
Three-tier calibration approach:

1. CACHED PROFILE (one file read, no benchmark)
   LoadOrCreateProfile(path) → check IsValid() → apply thresholds

2. QUICK MICRO-BENCHMARKS (design target ~100 ms — microbench.go file comment + QuickCalibrate doc;
   the repo has no measurement of the actual wall time)
   QuickCalibrate(ctx) → parallel/FFT threshold tests
   → escalates to tier 3 when confidence < EscalationConfidenceThreshold
     (= 0.5, strategy.go:EscalationConfidenceThreshold; used in calibration.go:tryFastThenEscalate)

3. FULL CALIBRATION (seconds to minutes)
   RunCalibration(ctx, out, registry, profilePath, progressDisplay, colorProvider)
   ├─ registry["fast"] is the only calculator used  (calibration.go:configureHardwareDetection)
   ├─ GenerateParallelThresholds() → CPU-adaptive candidates  (idem)
   ├─ For each threshold: Calculate(ctx, progressChan, 0, CalibrationN, opts)
   ├─ Find best parallel threshold by duration — THE ONLY DIMENSION SWEPT
   ├─ FFT / Strassen come from the static heuristics, NOT from a sweep:
   │    config.EstimateOptimalFFTThreshold() / …StrassenThreshold()
   │    (calibration.go:persistCalibrationProfile)
   └─ SaveProfile(path) → persist for future runs
```

Only the `--auto-calibrate` escalation tier actually sweeps FFT and Strassen:
`CompleteStrategy.Calibrate` calls `findBestFFTThreshold` over
`GenerateFFTThresholds()` (`[-1]` — the sequential no-FFT baseline, prepended
before the loop — then 200K→1M bits, step 50K: 18 candidates) with the
`"fast"` calculator,
and `findBestStrassenThreshold` with the `"matrix"` calculator when it is registered
(`internal/calibration/strategy_complete.go:CompleteStrategy.Calibrate`,
`adaptive.go:GenerateFFTThresholds`, `runner.go:findBestFFTThreshold` /
`runner.go:findBestStrassenThreshold`).

---

## 9) Configuration and Environment

### Core CLI flags (selected)

| Flag | Meaning |
|---|---|
| `-n` | Fibonacci index (default: 100,000,000) |
| `-algo` | `all`, `fast`, `matrix`, `fft`. **Not `gmp`**: `app.New` builds its own factory with `fibonacci.NewDefaultFactory()` (`internal/app/app.go:New`), which registers `fast`/`matrix`/`fft` only (`internal/fibonacci/registry.go:NewDefaultFactory`). The `-tags gmp` `init()` registers into the package-private `globalFactory`, which nothing reads (`internal/fibonacci/calculator_gmp.go`, its `globalFactory` var and `init`). To use it, call `fibonacci.RegisterGMPCalculator` on your own factory — see [`docs/algorithms/GMP.md`](algorithms/GMP.md). |
| `-timeout` | Global execution timeout (default: 5m) |
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
| `-completion` | Shell completion script (bash, zsh, fish, powershell) |
| `--last-digits` | Modular computation mode (O(K) memory) |
| `--memory-limit` | Memory budget guard (e.g., "8G", "512M") |
| `--gc-control` | `auto` / `aggressive` / `disabled` |

### Environment variable overrides (`FIBCALC_` prefix)

A `FIBCALC_*` variable is read only when the matching flag is absent from the command line, so for every setting other than the three thresholds the order is **CLI flags > env vars > static defaults** (`internal/config/env.go:applyEnvOverrides`). For `FIBCALC_THRESHOLD` / `FIBCALC_FFT_THRESHOLD` / `FIBCALC_STRASSEN_THRESHOLD` — and for the flags they mirror — a valid cached calibration profile takes precedence over both; see [Configuration Cascade](#configuration-cascade) above.

Supported keys include:

- `FIBCALC_N`, `FIBCALC_ALGO`, `FIBCALC_TIMEOUT`
- `FIBCALC_THRESHOLD`, `FIBCALC_FFT_THRESHOLD`, `FIBCALC_STRASSEN_THRESHOLD`
- `FIBCALC_VERBOSE`, `FIBCALC_DETAILS`, `FIBCALC_QUIET`, `FIBCALC_CALCULATE`
- `FIBCALC_CALIBRATE`, `FIBCALC_AUTO_CALIBRATE`, `FIBCALC_CALIBRATION_PROFILE`
- `FIBCALC_OUTPUT`, `FIBCALC_MEMORY_LIMIT`, `FIBCALC_GC_CONTROL`, `FIBCALC_LAST_DIGITS`
- `FIBCALC_MACHINE_OUTPUT`, `FIBCALC_TUI`, `FIBCALC_TUI_THEME`

The list above is `envOverrides` (`internal/config/env.go`) plus `FIBCALC_TUI_THEME`
(read by `internal/ui`). `FIBCALC_PROFILE_MAX_AGE` is read separately by
`internal/calibration` (`ProfileMaxAgeEnv`), outside this override table.

Also honors standard `NO_COLOR` behavior.

### Calibration profiles
- File-backed JSON profile (`~/.fibcalc_calibration.json` by default).
- Stores hardware signature and tuned thresholds:
  - parallel
  - FFT
  - Strassen
- Validity checks include profile version, CPU count, arch, and word size.

### Performance-Tuning Constants

| Constant | Value | Purpose |
|---|---|---|
| `DefaultParallelThreshold` | 4,096 bits | Minimum operand size for parallel multiplication |
| `DefaultFFTThreshold` | 500,000 bits | Crossover point: math/big → FFT multiplication |
| `DefaultStrassenThreshold` | 3,072 bits | Crossover: naive matrix → Strassen multiplication |
| `ParallelFFTThreshold` | 5,000,000 bits | Re-enable parallelism when FFT is active |
| `CalibrationN` | 10,000,000 | Default N for calibration benchmarks |
| `MaxPooledBitLen` | 50,000,000 bits | Maximum big.Int size kept in pool (~6.25 MB) |
| `ProgressReportThreshold` | 0.01 (1%) | Minimum progress delta before UI update |
| `ProgressBufferMultiplier` | 5 | Progress channel buffer = numCalcs × 5 |
| `MaxFibUint64` | 93 | F(93) is the largest Fibonacci fitting in uint64 |

---

## 10) Error Handling

### Typed errors

| Type | Purpose |
|---|---|
| `ConfigError` | Invalid configuration/flags/parameters |
| `MemoryError` | Requested memory exceeds available/configured constraints |
| `CalculationError` | Wraps underlying computation failure cause (with `Unwrap()`) |

Timeouts and cancellations carry no dedicated type: they are classified with
`errors.Is` against `context.DeadlineExceeded`/`context.Canceled` (OVR-07
removed the former `TimeoutError`/`ValidationError`).

Additional helpers: `WrapCalculationError` (contextual wrapping with `%w` around a `CalculationContext`). Context classification is done inline in `HandleCalculationError` via `errors.Is` against `context.DeadlineExceeded`/`context.Canceled` — there is no `IsContextError` helper.

### Exit codes

| Code | Constant | Meaning |
|---:|---|---|
| `0` | `ExitSuccess` | Success |
| `1` | `ExitErrorGeneric` | Generic/unexpected error |
| `2` | `ExitErrorTimeout` | Timeout |
| `3` | `ExitErrorMismatch` | Cross-algorithm result mismatch |
| `4` | `ExitErrorConfig` | Configuration error |
| `130` | `ExitErrorCanceled` | Canceled (signal/context) |

`HandleCalculationError` maps timeout/cancel/generic failures into standardized user-facing messaging + exit status. The `ColorProvider` interface allows color-agnostic error formatting.

---

## 11) Testing Strategy

FibCalc uses a layered testing approach with 100+ `*_test.go` files:

- **Unit tests:** extensive table-driven tests across internal packages.
- **Golden file tests:** canonical expected Fibonacci outputs (`internal/fibonacci/testdata/fibonacci_golden.json`), plus CLI output goldens.
- **Fuzz testing:** Go fuzzing for cross-algorithm consistency, identities, monotonic progress, modular arithmetic.
- **Property-based tests:** `gopter` checks mathematical invariants (e.g., Cassini identity).
- **Benchmarks:** algorithm and subsystem benchmarks with alloc stats and profiling hooks.
- **Race detector:** standard test invocation includes `-race`.
- **E2E tests:** build and execute binary subprocesses in `test/e2e`.
- **Spy/mock patterns:** orchestration spy tests; for the algorithm core, implement [`fibonacci.CoreCalculator`](../internal/fibonacci/calculator.go) (exported interface) — a minimal stub fits in ~30 lines (see `coreStub` in `internal/orchestration/contract_test.go`).

Typical commands:

```bash
go test -v -race -cover ./...
go test -bench=. -benchmem ./internal/fibonacci/
go test -fuzz=FuzzFastDoublingConsistency ./internal/fibonacci/
```

---

## 12) Build System

The project uses standard Go tooling + Makefile workflows.

### Key Make targets

- **Build:** `build`, `build-all`, `build-linux`, `build-linux-arm64`, `build-windows`, `build-windows-arm64`, `build-darwin`
- **Test/quality:** `test`, `test-short`, `coverage`, `benchmark`, `lint`, `security`, `check`
- **Dev hygiene:** `format`, `tidy`, `deps`, `upgrade`
- **PGO:** `pgo-profile`, `pgo-check`, `build-pgo`, `build-pgo-all`, `pgo-rebuild`, `pgo-clean`
- **Tools:** `install-tools` (golangci-lint, gosec)

### Version injection

Build-time version injection via linker flags:
```bash
-X github.com/agbruneau/FibGo/internal/app.Version=$(VERSION)
-X github.com/agbruneau/FibGo/internal/app.Commit=$(COMMIT)
-X github.com/agbruneau/FibGo/internal/app.BuildDate=$(BUILD_DATE)
```

### PGO support
- Profile path: `cmd/fibcalc/default.pgo`
- `make pgo-profile` generates profile from benchmark workload (`BenchmarkFibonacci/(FastDoubling|MatrixExp|FFTBased)` with `-benchtime=5s -count=3`).
- `make build-pgo` compiles with `-pgo=...`.
- Build target auto-detects PGO profile and uses it if present.

### Cross-compilation
- `make build-all` covers Linux, Windows and macOS in both `amd64` and `arm64`. The PGO variant (`build-pgo-all`) covers linux/amd64, windows/amd64 and macOS amd64+arm64 only.

### GMP build tag
- Optional calculator in `internal/fibonacci/calculator_gmp.go`.
- Build with:

```bash
go build -tags=gmp -o fibcalc ./cmd/fibcalc
```

- The `init()` in that file registers a `gmp` algorithm at load time — but into
  the package-private `globalFactory`, which no production code reads. `-algo
  gmp` therefore stays unavailable even with the tag on; see the `-algo` row in
  [§9](#core-cli-flags-selected).

### Linting and security
- `.golangci.yml` configures comprehensive linting rules.
- `gosec` for security audits.

---

## 13) External Dependencies (direct)

From `go.mod`, direct dependencies are:

| Module | Purpose in FibCalc |
|---|---|
| `golang.org/x/sync` | `errgroup` for structured concurrent execution |
| `github.com/briandowns/spinner` | CLI spinner UX |
| `github.com/charmbracelet/bubbles` | Bubble Tea UI components (TUI) |
| `github.com/charmbracelet/bubbletea` | TUI framework (Elm architecture runtime) |
| `github.com/charmbracelet/lipgloss` | Terminal styling/theme for TUI |
| `github.com/leanovate/gopter` | Property-based testing |
| `github.com/ncw/gmp` | Optional GMP big integer backend (`gmp` build tag) |
| `github.com/rs/zerolog` | Structured logging (package-level Nop loggers by default) |
| `github.com/shirou/gopsutil/v4` | Host CPU/memory sampling, called directly from `internal/tui` |
| `golang.org/x/sys` | Low-level OS/CPU support (including CPU feature usage) |

**Notable indirect dependencies:** Charmbracelet ecosystem (x/ansi, x/cellbuf, x/term, colorprofile), fatih/color, go-ole (Windows COM), ebitengine/purego, tklauser/numcpus.

---

## 14) Architectural Decision Records (ADR)

> Les entrées ci-dessous (ADR-001..ADR-010) forment un **journal narratif interne à ce document**, avec sa propre numérotation à trois chiffres. Elles ne correspondent pas une à une aux fichiers de [`docs/adr/`](adr/) (registre formel `0001`..`0009`, numérotation à quatre chiffres et sujets distincts) ; consulter ce répertoire pour les ADR canoniques.

### ADR-001: Using `sync.Pool` for Calculation States
- **Context:** Fibonacci calculations for large N require numerous temporary `big.Int` objects.
- **Decision:** Use `sync.Pool` to recycle `CalculationState` and `matrixState` objects.
- **Guard:** Objects exceeding `MaxPooledBitLen` (50M bits / ~6.25 MB) are discarded to prevent memory bloat.
- **Results:** fewer allocations per calculation. No measurement of a speed-up from pooling alone exists in this repo; the only benchmark artifact tracked is `docs/audits/bench-baseline.txt`, which measures whole calculators, not this decision in isolation.

### ADR-002: Dynamic Multiplication Algorithm Selection
- **Context:** FFT multiplication has superior asymptotic complexity (O(n log n)) but significant overhead for small operands.
- **Decision:** 2-tier `smartMultiply` function: FFT (> FFTThreshold bits) or `math/big` Karatsuba (below).
- **Results:** Optimal performance across entire value range; configurable via threshold.

### ADR-003: Adaptive Parallelism
- **Context:** Parallelism has synchronization cost exceeding gains for small calculations.
- **Decision:** Enable parallelism only above `ParallelThreshold` (default: 4096 bits). Suppress during FFT (CPU saturation), re-enable above 5M bits.
- **Results:** Optimal performance by calculation size; avoids CPU over-subscription.

### ADR-004: Interface-Based Decoupling (Orchestration → Presentation)
- **Context:** Orchestration was importing CLI packages, violating Clean Architecture.
- **Decision:** Define `ProgressReporter` and `ResultPresenter` interfaces in orchestration; implement in CLI and TUI packages.
- **Results:** Clean dependency flow validated by TUI as second implementation; improved testability.

### ADR-005: Calculation Arena for Contiguous Allocation
- **Context:** Per-buffer GC tracking adds significant overhead for very large N.
- **Decision:** Pre-allocate contiguous block via `CalculationArena` for state big.Int backing arrays.
- **Results:** Reduced GC pressure, coexists with sync.Pool (pool recycles state objects, arena pre-sizes backing arrays).

### ADR-006: GC Control During Large Calculations
- **Context:** Go's default `GOGC=100` lets the live heap double before a cycle runs, so a large calculation carries roughly twice its working set.
- **Decision:** Disable GC during computation for N ≥ 1M (auto mode), with `debug.SetMemoryLimit` as OOM safety net.
- **Results:** No *GOGC-driven* cycle runs inside the guarded region; configurable via `--gc-control`. The region is not GC-free, though: the same `Begin()` installs `SetMemoryLimit(3 × Sys)`, which the runtime honours even with `GOGC=off` — by design, per the `DefaultMemoryLimitMultiplier` doc comment ("the Go runtime will trigger emergency GC"). See [PERFORMANCE.md](PERFORMANCE.md) §6. The repo carries no peak-RSS measurement for this decision — `memory.EstimateMemoryUsage` is a model, not an observation (see [PERFORMANCE.md](PERFORMANCE.md) §7).

### ADR-007: Observer Pattern for Progress Reporting
- **Context:** Progress reporting was tightly coupled to channel-based communication.
- **Decision:** Introduce `ProgressObserver` interface with `ProgressSubject` (observable). Use `Freeze()` for lock-free snapshot in hot loops.
- **Results:** Supports multiple concurrent observers; decouples progress from transport mechanism.

### ADR-008: Framework Pattern for Algorithm Loops
- **Context:** Fast Doubling and FFT-Based algorithms shared identical loop structures with different multiplication strategies.
- **Decision:** Extract `DoublingFramework` and `MatrixFramework` to own bit-iteration, progress reporting, and context checks, delegating operations to pluggable strategies.
- **Results:** Eliminated significant code duplication; new strategies can be added without modifying loop logic.

### ADR-009: Heuristique matérielle pour les seuils par défaut
- **Context:** Les seuils à 0 (auto) ne devaient pas dépendre uniquement de `runtime.NumCPU()` alors que les chemins FFT et multiplications larges bénéficient fortement des jeux d’instructions x86 (AVX2 / AVX-512).
- **Decision:** `internal/config/hardware.go` classifie l’hôte (`DetectHardwareHeuristic`) ; `thresholds.go` ajuste les estimations FFT / Strassen / parallélisme en conséquence. Le profil de calibration inclut `cpu_heuristic_key` (format v3) pour invalider un cache si la classe SIMD change.
- **Results:** Comportement documenté et testable via les variantes non exportées `estimate*ThresholdForHeuristic` (`internal/config/thresholds.go`), exercées par les tests du package `config` ; profils v2 obsolètes (`CurrentProfileVersion = 3`, `internal/calibration/profile.go:CurrentProfileVersion`).

### ADR-010: Backends arithmétiques hors GMP (décision recherche)
- **Context:** des bibliothèques externes (FLINT et autres) pourraient être évaluées pour comparaison recherche ; charge de build, licences et CI hétérogène.
- **Decision:** Pas d’intégration C/C++ supplémentaire dans la branche `main` tant qu’une matrice de build reproductible, une revue de licence et des tests d’équivalence sur un sous-ensemble de `N` ne sont pas bouclés. Point d’extension supporté : `Register` sur une fabrique construite via `fibonacci.NewDefaultFactory()` (même modèle que `RegisterGMPCalculator` sous le tag `gmp`).
- **Results:** Décision **no-go** pour un second backend obligatoire ; expérimentations possibles sur branche dédiée ou fork en suivant [docs/algorithms/GMP.md](algorithms/GMP.md) (section recherche).

---

## Appendix: Architectural Notes for New Engineers

- Start from `cmd/fibcalc/main.go` and trace into `internal/app`.
- For execution semantics, read `internal/orchestration` first.
- For algorithm internals, focus on:
  1. `internal/fibonacci/fastdoubling.go` + `doubling_framework.go`
  2. `internal/fibonacci/matrix.go` + `matrix_framework.go` + `matrix_ops.go`
  3. `internal/fibonacci/strategy.go` (understand the 2 strategies)
  4. `internal/fibonacci/fft.go` + `internal/bigfft`
- For user interaction, study `internal/cli` and `internal/tui` presenters.
- For operational tuning, use `docs/CALIBRATION.md`, `docs/PERFORMANCE.md`, and Makefile PGO targets.
- For in-depth architecture, see `docs/architecture/README.md` with C4 diagrams and flow charts.

This architecture intentionally emphasizes separation of concerns, algorithmic interchangeability, and performance-tuning hooks while keeping orchestration and presentation decoupled.