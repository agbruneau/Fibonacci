# FibGo / FibCalc Architecture

> **Ce document est la vue d'ensemble rapide** de l'architecture de FibCalc. Pour la référence détaillée (diagrammes C4, flows Mermaid, index complet de la documentation), voir **[docs/architecture/README.md](architecture/README.md)**.

> **Vue interactive** — Un dashboard navigable du graphe de connaissances (1 128 nœuds, 4 782 arêtes, 9 couches, tour guidé 12 étapes — comptages vérifiés le 2026-07-06 sur l'artefact régénéré au commit 6e3ec29 ; à re-vérifier à chaque régénération du dashboard) est publié sur **[agbruneau.github.io/FibGo/dashboard/](https://agbruneau.github.io/FibGo/dashboard/)**. Complément visuel à ce document statique. Source : [`docs/dashboard/knowledge-graph.json`](dashboard/knowledge-graph.json), build statique : [`docs/dashboard/`](dashboard/).

## 1) Project Overview

**FibGo** (module/library name: **FibCalc**) is a high-performance Fibonacci computation system implemented in Go.

- **Go module path:** `github.com/agbruneau/FibGo`
- **Go version:** 1.26.0+ (`go.mod` declares `go 1.26.0`, no `toolchain` directive)
- **Primary binary:** `cmd/fibcalc`
- **Codebase stats:** run `make stats` for the canonical Go-package and LOC counts (the totals drift on every refactor; encoding them statically here has historically caused divergence between this document, `CLAUDE.md` and reality).
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
| (algorithms)        (observer model)   (FFT arithmetic)                   |
|   fibonacci/memory   fibonacci/threshold                                   |
|   (arena, GC ctrl)   (dynamic tuning)                                      |
+----------------------------------+------------------------------------+
                                   |
                                   v
+-----------------------------------------------------------------------+
|                          Infrastructure Helpers                       |
|                                                                       |
| internal/metrics  metrics/system  internal/format  test/e2e, docs     |
+-----------------------------------------------------------------------+
```

### Dependency Rules

- **Interfaces layer** → imports Application + Use-Case layers. `internal/tui`
  consumes domain types through `orchestration.Calculator`/`Options` aliases
  (no direct `internal/fibonacci` import) — enforced by `internal/arch_test.go`.
- **Application layer** → imports Use-Case + Domain layers.
- **Use-Case layer** → imports Domain layer only.
- **Domain layer** → no imports from outer layers (self-contained).
  `internal/fibonacci/threshold` does **not** import `internal/config` ; the
  application layer wires `threshold.Tuning` via `threshold.SetTuning`.
- **Infrastructure** → utility packages with no upward dependencies.
  `internal/errors` ships its own byte-formatter (`formatBytesLocal`) instead
  of depending on `internal/format`.

These three arrows (`threshold → config`, `errors → format`, `tui →
fibonacci`) were upward leaks resolved during the May-2026 hardening sprint ;
`internal/arch_test.go` fails the build if any of them is reintroduced.

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
├── CHANGELOG.md                 # Version history
└── CLAUDE.md                    # AI assistant context
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
│   ├── scan.go                  # Fast base-10 string → big.Int (scanner)
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
│   ├── threshold/               # Dynamic threshold manager
│   └── fibonaccitest/           # Public test double for fibonacci.Calculator
├── format/                      # Duration/number/progress ETA formatting
├── metrics/                     # Runtime performance/memory indicators
│   └── system/                  # Host CPU/memory sampling (formerly internal/sysmon)
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
- **Key functions:** `ParseConfig`, `ApplyAdaptiveThresholds`, `DetectHardwareHeuristic`, `EstimateOptimalParallelThreshold`, `EstimateOptimalFFTThreshold`, `EstimateOptimalStrassenThreshold`, and `Estimate*ForHeuristic` for tests/diagnostics.
- **Precedence chain:** CLI flags > env vars > calibration profile > adaptive estimation (CPU cores + x86 SIMD tier via `golang.org/x/sys/cpu`) > static defaults.

## `internal/calibration`
- **Responsibility:** full/quick calibration, adaptive threshold candidate generation, micro-benchmarks, profile file persistence.
- **Key types:** `CalibrationProfile`, `CalibrationOptions`, `CalibrationResult`.
- **Key functions:** `RunCalibration`, `AutoCalibrate`, `AutoCalibrateWithProfile`, `LoadCachedCalibration`, `LoadOrCreateProfile`, `SaveProfile`, `QuickCalibrate`, `GenerateParallelThresholds`.
- **Three-tier calibration:** (1) cached profile → (2) quick micro-benchmarks (~100ms) → (3) full benchmark with adaptive threshold search.

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
- **Key types:** `DynamicThresholdManager`, `DynamicThresholdConfig`, `IterationMetric`, `ThresholdStats`.
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
  - Architecture-aware arithmetic via `go:linkname` to `math/big` internals
  - CPU feature probing (AVX2, etc.) on amd64

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
- **Key types:** `ConfigError`, `CalculationError`, `TimeoutError`, `ValidationError`, `MemoryError`.
- **Key helpers:** `WrapError`, `IsContextError`, `HandleCalculationError`, `ColorProvider` interface.

## `internal/metrics`, `internal/metrics/system`, `internal/format`, `internal/ui`, `internal/testutil`
- **Responsibility:** telemetry formatting, performance indicators (throughput, O(1) properties), host CPU/memory sampling (`internal/metrics/system`, formerly `internal/sysmon`), theming/color controls (`NO_COLOR` support), test helpers.

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
| **Bump Allocator** | `bigfft.bumpAllocator` | Batch temporary allocations with O(1) reset for FFT internals |
| **FFT Transform Cache** | `bigfft.fft_cache.go` | Caches FFT transforms for reuse across multiply/square operations within an iteration |
| **Dynamic Threshold Adjustment** | `threshold.DynamicThresholdManager` | Records per-iteration metrics and adjusts FFT/parallel thresholds mid-computation |
| **Zero-Copy Result Return** | `DoublingFramework.ExecuteDoublingLoop`, `MatrixFramework.ExecuteMatrixLoop` | "Steals" result pointer from pooled state instead of copying, saving O(n) copy for large results |
| **Generics with Pointer Constraints** | `executeTasks[T any, PT interface{*T; task}]` | Generic task execution eliminating code duplication between multiplication and squaring tasks |
| **GC Controller** | `memory.GCController` | Disables GC during large computations (N ≥ 1M), restores afterward; uses `debug.SetMemoryLimit` as safety net |

Additional notable engineering patterns include:
- **Runtime-configurable threshold heuristics** with adaptive estimation based on CPU count
- **Channel-based progress aggregation** with buffered channels (`ProgressBufferMultiplier = 5`)
- **Lock-free observer snapshots** via `ProgressSubject.Freeze()` for hot loop performance
- **Semaphore-based concurrency limiting** (`NumCPU` for Fibonacci tasks, `NumCPU` for FFT tasks — two separate semaphores)
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
│    ├─ config.Validate(availableAlgos) → semantic checks          │
│    └─ Algo normalization (strings.ToLower)                       │
├───────────────────────────────────────────────────────────────────┤
│ 3. THRESHOLD RESOLUTION                                           │
│    calibration.LoadCachedCalibration(cfg, profilePath)           │
│    ├─ IF cached profile valid → apply profile thresholds         │
│    └─ ELSE → config.ApplyAdaptiveThresholds(cfg)                 │
│         ├─ EstimateOptimalParallelThreshold() (CPU-based)        │
│         ├─ EstimateOptimalFFTThreshold() (CPU-based)             │
│         └─ EstimateOptimalStrassenThreshold() (CPU-based)        │
├───────────────────────────────────────────────────────────────────┤
│ 4. MODE DISPATCH (Application.Run)                                │
│    ├─ Completion mode → cli.GenerateCompletion → exit            │
│    ├─ Calibration mode → calibration.RunCalibration → exit       │
│    ├─ Auto-calibration → calibration.AutoCalibrate → update cfg  │
│    ├─ TUI mode → tui.Run(ctx, calculators, cfg, version)         │
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
│    ├─ algo="all" → factory.GetAll() → all registered calculators │
│    └─ algo=specific → factory.Get(algo) → single calculator     │
├───────────────────────────────────────────────────────────────────┤
│ 7. CONCURRENT EXECUTION                                           │
│    orchestration.ExecuteCalculations(ctx, calculators, n, opts)  │
│    ├─ Progress channel: make(chan, numCalcs * 5)                  │
│    ├─ Progress goroutine: reporter.DisplayProgress(wg, ch, ...)  │
│    ├─ Single calculator: direct call (no errgroup overhead)      │
│    └─ Multiple calculators: errgroup fan-out                     │
│         └─ Per calculator:                                        │
│             ├─ Calculator.Calculate(ctx, progressChan, idx, n)   │
│             ├─ → ProgressSubject + ChannelObserver registration   │
│             ├─ → CalculateWithObservers                           │
│             │    ├─ Small-N fast path (n ≤ 93 → iterative add)   │
│             │    ├─ configureFFTCache(opts)                       │
│             │    ├─ bigfft.EnsurePoolsWarmed(n)                   │
│             │    ├─ subject.Freeze(calcIndex) → lock-free reporter│
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
│         ├─ Per bit: ExecuteStep (3 multiplications)              │
│         │   ├─ shouldParallelizeMultiplication() decision         │
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
│    └─ Success → PresentResult(bestResult, n, verbose, details)   │
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
      └─ Controlled by shouldParallelizeMultiplication():
         ├─ Enabled when: operand > ParallelThreshold (default: 4096 bits)
         ├─ Suppressed when: FFT active (FFT saturates CPU cores)
         └─ Re-enabled when: operand > ParallelFFTThreshold (5M bits)
      └─ Semaphore: NumCPU concurrent goroutines max

Level 3: FFT internal parallelism
   └─ bigfft recursive decomposition: configurable goroutine limit
      └─ Semaphore: NumCPU concurrent goroutines max
      └─ Total system: up to NumCPU*2 simultaneous goroutines
         (mitigated by Level 2 suppression during FFT)
```

### Progress Propagation Flow

```text
Core Algorithm
   └─ ProgressCallback(float64)  [per-iteration, throttled to ≥1% change]
       └─ FrozenProgressSubject (lock-free snapshot)
           └─ ChannelObserver → progressChan (buffered)
               └─ ProgressReporter goroutine
                   ├─ CLI: spinner + ETA + progress percentage
                   └─ TUI: progress bar + sparklines + metrics panel
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
- **Result detachment:** `ReleaseStateWithResult` deep-copies the result out of the arena (~850 KB memcpy for F(10M), <0.01 % of runtime) so the arena can safely be reset and reused on the next acquisition. The previous "steal `s.FK`" zero-copy trick was dropped because it left the result aliasing pooled memory the next tenant would overwrite.

### B. Matrix Exponentiation (`MatrixExponentiationCalculator`)
- Uses binary exponentiation of Fibonacci Q-matrix: `[[1,1],[1,0]]^(n-1)`.
- `MatrixFramework` drives loop (LSB → MSB iteration).
- Matrix ops switch between naive 2×2 multiply and Strassen based on `StrassenThreshold`.
- Includes symmetric squaring optimization (exploits `[a,b; b,c]` symmetry to reduce multiplications).
- **Zero-copy result return:** steals `res.a` from matrix state. Matrix exponentiation does not use the state-bound arena, so the steal trick is still safe here.

### C. FFT-Based Doubling (`FFTBasedCalculator`)
- Same doubling loop model (via `DoublingFramework`), but strategy is `FFTOnlyStrategy`.
- Forces FFT multiplication/squaring for all operations regardless of operand size.
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
   └─ FFTOnlyStrategy:   always FFT (mulFFT/sqrFFT)
```

### `internal/bigfft` Role
- Provides efficient arithmetic primitives for huge operands (hundreds of millions of bits):
  - **Fermat ring arithmetic:** operations in Z/(2^k+1) for FFT kernel
  - **Recursive FFT decomposition** with configurable parallelism
  - **Transform caching:** reuses computed transforms across multiply/square in same step
  - **Size-class pools** with adaptive pre-warming based on estimated operand sizes
  - **Bump allocator** for batch temporary allocations with O(1) reset
  - **Architecture-aware:** `go:linkname` to `math/big` internal word operations, CPU feature detection (AVX2)
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

```text
CLI flags (highest priority)
   ↓ applyEnvOverrides(config, flagSet)
FIBCALC_* environment variables
   ↓ calibration.LoadCachedCalibration()
Cached calibration profile (~/.fibcalc_calibration.json)
   ↓ config.ApplyAdaptiveThresholds()
CPU-adaptive estimation (based on runtime.NumCPU, GOARCH, etc.)
   ↓
Static defaults (in constants.go)
```

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

1. CACHED PROFILE (fastest, ~0ms)
   LoadOrCreateProfile(path) → check IsValid() → apply thresholds

2. QUICK MICRO-BENCHMARKS (~100ms)
   QuickCalibrate(ctx) → parallel/FFT threshold tests
   → requires confidence ≥ 0.5 to accept results

3. FULL CALIBRATION (seconds to minutes)
   RunCalibration(ctx, out, registry, progressDisplay, colorProvider)
   ├─ GenerateParallelThresholds() → CPU-adaptive candidates
   ├─ For each threshold: Calculate(ctx, progressChan, 0, CalibrationN, opts)
   ├─ Find best parallel threshold by duration
   ├─ Find best FFT threshold
   ├─ Find best Strassen threshold (using matrix calculator)
   └─ SaveProfile(path) → persist for future runs
```

---

## 9) Configuration and Environment

### Core CLI flags (selected)

| Flag | Meaning |
|---|---|
| `-n` | Fibonacci index (default: 100,000,000) |
| `-algo` | `all`, `fast`, `matrix`, `fft` (and `gmp` if built/tagged) |
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

Implemented with precedence: **CLI flags > env vars > adaptive estimation > static defaults**.

Supported keys include:

- `FIBCALC_N`, `FIBCALC_ALGO`, `FIBCALC_TIMEOUT`
- `FIBCALC_THRESHOLD`, `FIBCALC_FFT_THRESHOLD`, `FIBCALC_STRASSEN_THRESHOLD`
- `FIBCALC_VERBOSE`, `FIBCALC_DETAILS`, `FIBCALC_QUIET`, `FIBCALC_CALCULATE`
- `FIBCALC_CALIBRATE`, `FIBCALC_AUTO_CALIBRATE`, `FIBCALC_CALIBRATION_PROFILE`
- `FIBCALC_OUTPUT`, `FIBCALC_MEMORY_LIMIT`, `FIBCALC_TUI`

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
| `TimeoutError` | Operation exceeded duration limit |
| `MemoryError` | Requested memory exceeds available/configured constraints |
| `ValidationError` | Structured field-level validation failures |
| `CalculationError` | Wraps underlying computation failure cause (with `Unwrap()`) |

Additional helpers: `WrapError` (contextual wrapping with `%w`), `IsContextError` (checks `context.Canceled`/`context.DeadlineExceeded`).

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
- **Spy/mock patterns:** orchestration spy tests ; pour le cœur d’algorithme, implémenter [`fibonacci.CoreCalculator`](../internal/fibonacci/calculator.go) (interface exportée) ou utiliser [`fibonacci/fibonaccitest`](../internal/fibonacci/fibonaccitest) pour un double minimal.

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

- **Build:** `build`, `build-all`, `build-linux`, `build-windows`, `build-darwin`
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
- Supported in Makefile for Linux/Windows/macOS (`amd64`, plus `arm64` for Darwin).

### GMP build tag
- Optional calculator in `internal/fibonacci/calculator_gmp.go`.
- Build with:

```bash
go build -tags=gmp -o fibcalc ./cmd/fibcalc
```

- Auto-registers a `gmp` algorithm at init time when tag is enabled.

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
| `github.com/shirou/gopsutil/v4` | Host/system metrics collection (sysmon) |
| `golang.org/x/sys` | Low-level OS/CPU support (including CPU feature usage) |

**Notable indirect dependencies:** Charmbracelet ecosystem (x/ansi, x/cellbuf, x/term, colorprofile), fatih/color, go-ole (Windows COM), ebitengine/purego, tklauser/numcpus.

---

## 14) Architectural Decision Records (ADR)

> Les entrées ci-dessous (ADR-001..ADR-010) forment un **journal narratif interne à ce document**, avec sa propre numérotation à trois chiffres. Elles ne correspondent pas une à une aux fichiers de [`docs/adr/`](adr/) (registre formel `0001`..`0008`, numérotation à quatre chiffres et sujets distincts) ; consulter ce répertoire pour les ADR canoniques.

### ADR-001: Using `sync.Pool` for Calculation States
- **Context:** Fibonacci calculations for large N require numerous temporary `big.Int` objects.
- **Decision:** Use `sync.Pool` to recycle `CalculationState` and `matrixState` objects.
- **Guard:** Objects exceeding `MaxPooledBitLen` (50M bits / ~6.25 MB) are discarded to prevent memory bloat.
- **Results:** 20-30% performance improvement, drastic allocation reduction.

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
- **Context:** Go's GC adds ~2× memory overhead for heap scanning.
- **Decision:** Disable GC during computation for N ≥ 1M (auto mode), with `debug.SetMemoryLimit` as OOM safety net.
- **Results:** Eliminates GC pauses, reduces peak memory ~50%; configurable via `--gc-control`.

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
- **Results:** Comportement documenté et testable via `Estimate*ForHeuristic` ; profils v2 obsolètes (version incrémentée).

### ADR-010: Backends arithmétiques hors GMP (décision recherche)
- **Context:** des bibliothèques externes (FLINT et autres) pourraient être évaluées pour comparaison recherche ; charge de build, licences et CI hétérogène.
- **Decision:** Pas d’intégration C/C++ supplémentaire dans la branche `main` tant qu’une matrice de build reproductible, une revue de licence et des tests d’équivalence sur un sous-ensemble de `N` ne sont pas bouclés. Point d’extension supporté : `fibonacci.RegisterCalculator` (même modèle que le tag `gmp`).
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