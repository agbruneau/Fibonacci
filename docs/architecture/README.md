# Fibonacci Calculator Architecture

## Overview

The Fibonacci Calculator is designed according to **Clean Architecture** principles, with strict separation of responsibilities and low coupling between modules. This architecture enables maximum testability, easy scalability, and simplified maintenance.

> **Ce document est la référence architecturale détaillée** avec diagrammes C4, flows Mermaid et index complet. Pour une vue d'ensemble rapide, voir **[docs/ARCH.md](../ARCH.md)**.

**Go Module**: `github.com/agbru/fibcalc` (Go 1.25.0)

**Codebase stats**: 20 Go packages | 105 source files | 89 test files | 20 documentation files

---

## Documentation Map

### Architecture

| Document | Description |
|----------|-------------|
| **[This file](README.md)** | Master index — package structure, interfaces, ADRs, data flow |
| [ARCH.md (quick overview)](../ARCH.md) | Architecture quick overview — layers, patterns, data flow, key decisions |
| [patterns/interface-hierarchy.mermaid](patterns/interface-hierarchy.mermaid) | Interface hierarchy diagram |

### C4 Diagrams (in `docs/architecture/`)

| Diagram | Description |
|---------|-------------|
| [system-context.mermaid](system-context.mermaid) | Level 1 — system context |
| [container-diagram.mermaid](container-diagram.mermaid) | Level 2 — container view |
| [component-diagram.mermaid](component-diagram.mermaid) | Level 3 — component view |
| [dependency-graph.mermaid](dependency-graph.mermaid) | Package dependency graph |

### Execution Flows

| Document | Description |
|----------|-------------|
| [flows/cli-flow.mermaid](flows/cli-flow.mermaid) | CLI mode execution flow |
| [flows/tui-flow.mermaid](flows/tui-flow.mermaid) | TUI mode execution flow |
| [flows/config-flow.mermaid](flows/config-flow.mermaid) | Configuration resolution flow |
| [flows/fastdoubling.mermaid](flows/fastdoubling.mermaid) | Fast Doubling execution flow |
| [flows/matrix.mermaid](flows/matrix.mermaid) | Matrix Exponentiation execution flow |
| [flows/fft-pipeline.mermaid](flows/fft-pipeline.mermaid) | FFT pipeline execution flow |

### Operational Guides (in `docs/`)

| Document | Description |
|----------|-------------|
| [../BUILD.md](../BUILD.md) | Build, install, and cross-compilation |
| [../TESTING.md](../TESTING.md) | Testing strategy, coverage, fuzz testing |
| [../PERFORMANCE.md](../PERFORMANCE.md) | Benchmarks, profiling, PGO |
| [../CALIBRATION.md](../CALIBRATION.md) | Hardware-adaptive calibration |
| [../TUI_GUIDE.md](../TUI_GUIDE.md) | TUI dashboard user guide |

### Algorithm Deep-Dives (in `docs/algorithms/`)

| Document | Description |
|----------|-------------|
| [../algorithms/FAST_DOUBLING.md](../algorithms/FAST_DOUBLING.md) | Fast Doubling algorithm |
| [../algorithms/MATRIX.md](../algorithms/MATRIX.md) | Matrix Exponentiation algorithm |
| [../algorithms/FFT.md](../algorithms/FFT.md) | FFT multiplication overview |
| [../algorithms/BIGFFT.md](../algorithms/BIGFFT.md) | BigFFT implementation details |
| [../algorithms/GMP.md](../algorithms/GMP.md) | GMP optional backend |
| [../algorithms/COMPARISON.md](../algorithms/COMPARISON.md) | Algorithm comparison |
| [../algorithms/PROGRESS_BAR_ALGORITHM.md](../algorithms/PROGRESS_BAR_ALGORITHM.md) | Progress bar algorithm |

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           ENTRY POINT                                   │
│                                                                         │
│                    ┌────────────────────────┐                           │
│                    │  cmd/fibcalc            │                          │
│                    │  cmd/generate-golden    │                          │
│                    └───────────┬─────────────┘                          │
└────────────────────────────────┼─────────────────────────────────────────┘
                                 │
┌────────────────────────────────┼─────────────────────────────────────────┐
│                   ORCHESTRATION LAYER                                    │
│                                ▼                                         │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │                    internal/orchestration                         │   │
│  │  • ExecuteCalculations() — parallel algorithm execution           │   │
│  │  • AnalyzeComparisonResults() — analysis and comparison           │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                             │                                            │
│  ┌──────────────┐  ┌───────────────┐  ┌──────────────┐                  │
│  │    config     │  │  calibration  │  │     app      │                  │
│  │   Parsing     │  │    Tuning     │  │  Lifecycle   │                  │
│  └──────────────┘  └───────────────┘  └──────────────┘                  │
└────────────────────────────────┼─────────────────────────────────────────┘
                                 │
┌────────────────────────────────┼─────────────────────────────────────────┐
│                      BUSINESS LAYER                                      │
│                                ▼                                         │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │                    internal/fibonacci                             │   │
│  │  ┌──────────────────┐  ┌──────────────────┐  ┌────────────────┐ │   │
│  │  │  Fast Doubling   │  │     Matrix       │  │    FFT-Based   │ │   │
│  │  │  O(log n)        │  │  Exponentiation  │  │    Doubling    │ │   │
│  │  │  Parallel        │  │  O(log n)        │  │    O(log n)    │ │   │
│  │  │  Zero-Alloc      │  │  Strassen        │  │    FFT Mul     │ │   │
│  │  └──────────────────┘  └──────────────────┘  └────────────────┘ │   │
│  │                            │                                     │   │
│  │                            ▼                                     │   │
│  │  ┌──────────────────────────────────────────────────────────────┐│   │
│  │  │                    internal/bigfft                           ││   │
│  │  │  • FFT multiplication for very large numbers                ││   │
│  │  │  • Object pooling, bump allocation, SIMD dispatch           ││   │
│  │  └──────────────────────────────────────────────────────────────┘│   │
│  └──────────────────────────────────────────────────────────────────┘   │
└────────────────────────────────┼─────────────────────────────────────────┘
                                 │
┌────────────────────────────────┼─────────────────────────────────────────┐
│                   PRESENTATION LAYER                                     │
│                                ▼                                         │
│  ┌──────────────────────────────────┐  ┌──────────────────────────────┐ │
│  │         internal/cli             │  │       internal/ui            │ │
│  │  • Spinner and progress bar      │  │  • ANSI color functions     │ │
│  │  • Result formatting             │  │  • Theme system             │ │
│  │  • ETA estimation                │  │  • NO_COLOR support         │ │
│  │  • Shell completion              │  │                              │ │
│  └──────────────────────────────────┘  └──────────────────────────────┘ │
│  ┌──────────────────────────────────┐                                   │
│  │         internal/tui             │  Activated via --tui flag         │
│  │  • btop-style dashboard          │                                   │
│  │  • Real-time logs, metrics       │                                   │
│  │  • Progress bar with ETA         │                                   │
│  │  • Keyboard navigation           │                                   │
│  └──────────────────────────────────┘                                   │
└──────────────────────────────────────────────────────────────────────────┘
```

## Package Structure

### `cmd/fibcalc`

The main CLI entry point. A minimal wrapper that calls `app.New()` and `app.Run()`, providing:

- Version flag handling (`--version`)
- Command-line argument parsing via `internal/config`
- Application lifecycle (timeout + signal handling)

### `cmd/generate-golden`

Utility for generating Fibonacci golden test data used by the test suite.

- **`main.go`**: Entry point for golden file generation

### `internal/fibonacci`

Business core of the application. Contains algorithm implementations, the factory/registry system, multiplication strategies, and the observer pattern for progress reporting.

| File | Responsibility |
|------|---------------|
| `calculator.go` | `Calculator` and `coreCalculator` interfaces, `FibCalculator` decorator |
| `registry.go` | `CalculatorFactory` interface, `DefaultFactory` with lazy creation and caching |
| `strategy.go` | `Multiplier` (narrow) and `DoublingStepExecutor` (wide) interfaces; `AdaptiveStrategy`, `FFTOnlyStrategy`, `KaratsubaStrategy` |
| `progress_aliases.go` | Backward-compatible type aliases for `internal/progress` types |
| `options.go` | `Options` struct: `ParallelThreshold`, `FFTThreshold`, `StrassenThreshold`, FFT cache settings (`FFTCacheMinBitLen`, `FFTCacheMaxEntries`, `FFTCacheEnabled`), dynamic threshold settings (`EnableDynamicThresholds`, `DynamicAdjustmentInterval`); `normalizeOptions()` fills zero values with defaults |
| `constants.go` | Performance tuning constants: `DefaultParallelThreshold` (4096), `DefaultFFTThreshold` (500,000), `DefaultStrassenThreshold` (3072), `ParallelFFTThreshold` (5,000,000), `CalibrationN` (10,000,000), `ProgressReportThreshold` (0.01) |
| `fastdoubling.go` | `OptimizedFastDoubling` algorithm implementation, `CalculationState` type and pool |
| `doubling_framework.go` | `DoublingFramework` — shared iteration framework for doubling-based algorithms |
| `matrix.go` | `MatrixExponentiation` algorithm implementation |
| `matrix_framework.go` | `MatrixFramework` — shared framework for matrix-based algorithms |
| `matrix_ops.go` | Matrix multiplication and squaring operations, Strassen dispatch (`multiplyMatrices`, `multiplyMatrixStrassen`), runtime threshold control (`Set/GetDefaultStrassenThreshold`) |
| `matrix_types.go` | `matrix` type (2x2), `matrixState` pool type |
| `fft_based.go` | `FFTBasedCalculator` — forces FFT for all multiplications |
| `fft.go` | `smartMultiply` / `smartSquare` — 2-tier multiplication selection (FFT or standard math/big) |
| `common.go` | Task semaphore, `MaxPooledBitLen`, `executeTasks` generics, `executeMixedTasks` |
| `generator.go` | `SequenceGenerator` interface for Fibonacci sequence generation |
| `generator_iterative.go` | Iterative generator implementation |
| `testing.go` | Test helpers and utilities |
| `modular.go` | `FastDoublingMod` — modular fast doubling for `--last-digits` mode |
| `calculator_gmp.go` | GMP calculator, auto-registers via `init()` (build tag: `gmp`) |

### `internal/fibonacci/memory`

Memory management sub-package extracted from `internal/fibonacci`.

| File | Responsibility |
|------|---------------|
| `arena.go` | `CalculationArena` — contiguous bump allocator for state big.Int |
| `gc_control.go` | `GCController` — GC control during calculation (auto/aggressive/disabled) |
| `budget.go` | `EstimateMemoryUsage`, `ParseMemoryLimit` — pre-calculation memory validation |

### `internal/fibonacci/threshold`

Dynamic threshold management sub-package extracted from `internal/fibonacci`.

| File | Responsibility |
|------|---------------|
| `manager.go` | `DynamicThresholdManager` — runtime threshold adjustment logic |
| `types.go` | `IterationMetric`, `ThresholdStats`, `DynamicThresholdConfig` type definitions |

### `internal/progress`

Observer pattern and progress reporting, extracted from `internal/fibonacci`. Backward-compatible type aliases in `internal/fibonacci/progress_aliases.go`.

| File | Responsibility |
|------|---------------|
| `observer.go` | `ProgressObserver` interface, `ProgressSubject` (observable) |
| `observers.go` | Observer implementations: `ChannelObserver`, `LoggingObserver`, `NoOpObserver` |
| `progress.go` | `ProgressUpdate`, `ProgressCallback` types, utilities (`CalcTotalWork`, `ReportStepProgress`) |

### `internal/bigfft`

FFT multiplication for `big.Int`, with object pooling, memory management, and platform-specific SIMD dispatch.

| File | Responsibility |
|------|---------------|
| `fft.go` | Public API: `Mul`, `MulTo`, `Sqr`, `SqrTo` |
| `fft_core.go` | Core FFT algorithm implementation |
| `fft_recursion.go` | Recursive FFT decomposition with runtime-configurable parallelism (`FFTParallelismConfig`, `Set/GetFFTParallelismConfig`) |
| `fft_poly.go` | Polynomial operations for FFT |
| `fft_cache.go` | FFT transform caching |
| `fermat.go` | Fermat ring arithmetic: `fermat` type (Z/(2^k+1)), `Shift`, `ShiftHalf`, `Add`, `Sub`, `Mul`, `Sqr`, `norm`; `smallMulThreshold` for schoolbook/big.Int cutover |
| `pool.go` | `sync.Pool`-based object pools with size classes |
| `pool_warming.go` | Pool pre-warming for adaptive buffer pre-allocation |
| `allocator.go` | Memory allocator abstraction |
| `bump.go` | Bump allocator for batch allocations |
| `memory_est.go` | Memory estimation for pre-allocation |
| `scan.go` | Bit scanning utilities |
| `arith_amd64.go` | amd64 vector arithmetic wrappers |
| `arith_generic.go` | Non-amd64 vector arithmetic wrappers |
| `arith_decl.go` | `go:linkname` declarations to `math/big` internals |
| `cpu_amd64.go` | Runtime CPU feature detection |

### `internal/orchestration`

Concurrent execution management with Clean Architecture decoupling.

| File | Responsibility |
|------|---------------|
| `orchestrator.go` | `ExecuteCalculations()`, `AnalyzeComparisonResults()` — parallel execution via `errgroup` |
| `interfaces.go` | `ProgressReporter`, `ResultPresenter` interfaces, `NullProgressReporter` |
| `calculator_selection.go` | `GetCalculatorsToRun()` — calculator selection logic from config |
| `progress.go` | `ProgressAggregator` — multi-calculator progress aggregation |

### `internal/cli`

Command-line user interface and presentation layer.

| File | Responsibility |
|------|---------------|
| `output.go` | `Display*` / `Format*` / `Write*` functions for output |
| `presenter.go` | `CLIProgressReporter` and `CLIResultPresenter` implementations |
| `ui.go` | Spinner management and terminal interaction |
| `ui_display.go` | Display functions for progress reporting and result presentation |
| `calculate.go` | Calculation orchestration entry point for CLI |
| `completion.go` | Shell completion script generation (bash, zsh, fish, powershell) |
| `provider.go` | Dependency provider for CLI components |

### `internal/tui`

Interactive TUI dashboard (btop-style), activated via `--tui` flag or `FIBCALC_TUI=true`. Built on the Elm architecture (Model-Update-View) with Bubble Tea.

| File | Responsibility |
|------|---------------|
| `doc.go` | Package documentation |
| `messages.go` | Tea message types (`ProgressMsg`, `ResultMsg`, `TickMsg`, `MemStatsMsg`, etc.) |
| `styles.go` | Orange-dominant dark theme palette with lipgloss (rounded orange borders, warm color scheme) |
| `keymap.go` | Keyboard bindings (`q`, `space`, `r`, arrows, `pgup`/`pgdn`) |
| `bridge.go` | `TUIProgressReporter` and `TUIResultPresenter` — implements orchestration interfaces |
| `header.go` | Header sub-model (title, version, elapsed time using `FormatExecutionDuration`) |
| `logs.go` | Scrollable log panel sub-model (viewport, auto-scroll) |
| `metrics.go` | Runtime metrics sub-model (memory, heap, GC, goroutines, speed) |
| `chart.go` | Progress bar, ETA, CPU/MEM sparkline indicators sub-model |
| `sparkline.go` | Sparkline and braille chart visualization |
| `footer.go` | Footer sub-model (keyboard shortcuts, status indicator) |
| `model.go` | Root model, `Init()`/`Update()`/`View()`, `Run()` entry point, layout (60/40 split) |

### `internal/calibration`

Automatic calibration system for hardware-specific threshold tuning.

| File | Responsibility |
|------|---------------|
| `calibration.go` | Core calibration logic |
| `adaptive.go` | Adaptive threshold generation based on CPU |
| `microbench.go` | Micro-benchmarking routines |
| `io.go` | Calibration profile I/O |
| `profile.go` | Calibration profile data structures |
| `runner.go` | Calibration test runner |

### `internal/config`

Configuration management.

| File | Responsibility |
|------|---------------|
| `config.go` | `ParseConfig()`, `AppConfig` struct, flag parsing |
| `env.go` | Environment variable support (`FIBCALC_*` prefix) |
| `usage.go` | Help text and usage formatting |
| `thresholds.go` | `ApplyAdaptiveThresholds()`, `EstimateOptimal*Threshold()` — hardware-adaptive threshold estimation |

### `internal/errors`

Centralized error handling.

| File | Responsibility |
|------|---------------|
| `errors.go` | Custom error types: `ConfigError`, `CalculationError` |
| `handler.go` | Error handler with standardized exit codes (0=success, 1=generic, 2=timeout, 3=mismatch, 4=config, 130=canceled) |

### `internal/app`

Application lifecycle management.

| File | Responsibility |
|------|---------------|
| `app.go` | Application initialization and lifecycle (`SetupContext`, signal handling), DI via `WithFactory()` |
| `calculate.go` | Calculation dispatch logic (extracted from app.go) |
| `version.go` | Version information |
| `doc.go` | Package documentation |

### `internal/ui`

Terminal UI utilities.

| File | Responsibility |
|------|---------------|
| `colors.go` | ANSI color functions |
| `themes.go` | Theme system (dark, light, orange, none), `NO_COLOR` support |

### `internal/metrics`

Performance measurement utilities.

| File | Responsibility |
|------|---------------|
| `indicators.go` | Performance indicators (bits/s, digits/s, steps/s) |
| `memory.go` | `MemoryCollector`, `MemorySnapshot` — runtime memory statistics |

## Key Interfaces

### Calculator (public)

```go
type Calculator interface {
    Calculate(ctx context.Context, progressChan chan<- ProgressUpdate,
        calcIndex int, n uint64, opts Options) (*big.Int, error)
    Name() string
}
```

### coreCalculator (internal)

```go
type coreCalculator interface {
    CalculateCore(ctx context.Context, reporter ProgressCallback,
        n uint64, opts Options) (*big.Int, error)
    Name() string
}
```

### CalculatorFactory

```go
type CalculatorFactory interface {
    Create(name string) (Calculator, error)
    Get(name string) (Calculator, error)
    List() []string
    Register(name string, creator func() coreCalculator) error
    GetAll() map[string]Calculator
}
```

### Multiplier (narrow)

```go
type Multiplier interface {
    Multiply(z, x, y *big.Int, opts Options) (*big.Int, error)
    Square(z, x *big.Int, opts Options) (*big.Int, error)
    Name() string
}
```

### DoublingStepExecutor (wide)

```go
type DoublingStepExecutor interface {
    Multiplier
    ExecuteStep(ctx context.Context, s *CalculationState, opts Options, inParallel bool) error
}

```

### ProgressObserver

```go
type ProgressObserver interface {
    Update(calcIndex int, progress float64)
}
```

### ProgressReporter (orchestration)

```go
type ProgressReporter interface {
    DisplayProgress(wg *sync.WaitGroup, progressChan <-chan fibonacci.ProgressUpdate,
        numCalculators int, out io.Writer)
}
```

### ResultPresenter (orchestration)

```go
type ResultPresenter interface {
    PresentComparisonTable(results []CalculationResult, out io.Writer)
    PresentResult(result CalculationResult, n uint64, verbose, details, concise bool, out io.Writer)
    FormatDuration(d time.Duration) string
    HandleError(err error, duration time.Duration, out io.Writer) int
}
```

## Architecture Decision Records (ADR)

### ADR-001: Using `sync.Pool` for Calculation States

**Context**: Fibonacci calculations for large N require numerous temporary `big.Int` objects.

**Decision**: Use `sync.Pool` to recycle calculation states (`CalculationState`, matrix states).

**Consequences**:

- Drastic reduction in memory allocations
- Decreased GC pressure
- 20-30% performance improvement
- Increased code complexity

### ADR-002: Dynamic Multiplication Algorithm Selection

**Context**: FFT multiplication is more efficient than standard `math/big` for very large numbers, but has significant overhead for small numbers.

**Decision**: Implement a 2-tier `smartMultiply` function that selects the algorithm based on operand size: FFT (> 500K bits) or standard `math/big` (below). `math/big` internally uses Karatsuba for large operands.

**Consequences**:

- Optimal performance across the entire value range
- Configurable via `FFTThreshold` in `Options`
- Requires calibration for each architecture

### ADR-003: Adaptive Parallelism

**Context**: Parallelism has a synchronization cost that can exceed gains for small calculations.

**Decision**: Enable parallelism only above a configurable threshold (`ParallelThreshold`, default: 4096 bits).

**Consequences**:

- Optimal performance according to calculation size
- Avoids CPU saturation for small N
- Parallelism disabled when FFT is used (FFT already saturates CPU), re-enabled above 5M bits (`ParallelFFTThreshold`)

### ADR-004: Interface-Based Decoupling (Orchestration → CLI)

**Context**: The orchestration package was directly importing CLI packages, violating Clean Architecture principles where business logic should not depend on presentation.

**Decision**: Define `ProgressReporter` and `ResultPresenter` interfaces in the orchestration package, with implementations in the CLI package.

**Consequences**:

- Clean Architecture compliance: orchestration no longer imports CLI
- Improved testability: interfaces can be mocked for unit tests
- Flexibility: alternative presenters (JSON, TUI, GUI) can be easily added
- `NullProgressReporter` enables quiet mode without conditionals
- TUI dashboard (`internal/tui`) was added as a second implementation, validating this decoupling
- Slightly more complex initialization in the app layer

### ADR-005: Calculation Arena for Contiguous Allocation

**Context**: For very large N, per-buffer GC tracking adds significant memory overhead.

**Decision**: Pre-allocate a single contiguous block via `CalculationArena` for all 5 state `big.Int` backing arrays, falling back to heap when exhausted.

**Consequences**:

- Reduced GC pressure for large calculations
- O(1) bulk release via `Reset()`
- Coexists with existing `sync.Pool` (pool recycles state objects, arena pre-sizes their backing arrays)

### ADR-006: GC Control During Large Calculations

**Context**: Go's GC adds ~2× memory overhead for heap scanning during large calculations.

**Decision**: Disable GC during computation for N ≥ 1M (auto mode), with `debug.SetMemoryLimit` as OOM safety net.

**Consequences**:

- Eliminates GC pauses during computation
- Reduces peak memory by ~50% (no GC overhead)
- Small OOM risk mitigated by soft memory limit
- Configurable via `--gc-control` flag

## Data Flow

```
1. app.New(args) → config.ParseConfig() parses CLI flags + env vars → AppConfig
2. app.New() → calibration.LoadCachedCalibration() or config.ApplyAdaptiveThresholds()
3. app.Run() dispatches to: completion | calibration | auto-calibration | TUI | CLI
4. ui.InitTheme() initializes terminal color support (respects NO_COLOR)
5. orchestration.GetCalculatorsToRun() selects calculators from the injected CalculatorFactory
6. context.WithTimeout() + signal.NotifyContext() creates lifecycle context
7. orchestration.ExecuteCalculations() runs calculators concurrently via errgroup
   - Each Calculator.Calculate() creates ProgressSubject + ChannelObserver
   - GCController.Begin() disables GC for large N
   - FibCalculator.CalculateWithObservers(): small-N fast path, FFT cache config, pool warming
   - CalculateCore creates CalculationArena and pre-sizes state from arena
   - Core algorithm (DoublingFramework or MatrixFramework) executes the computation loop
   - GCController.End() restores GC and runs collection
   - ProgressSubject.Freeze() creates lock-free snapshot → ChannelObserver → progressChan
   - ProgressReporter (CLIProgressReporter or TUIProgressReporter) displays progress
8. orchestration.AnalyzeComparisonResults() sorts by duration, validates consistency
9. ResultPresenter (CLIResultPresenter or TUIResultPresenter) formats and displays output
```

For detailed flow diagrams, see [flows/](flows/).

## Design Patterns

This codebase employs 14 design patterns. See also: **[patterns/interface-hierarchy.mermaid](patterns/interface-hierarchy.mermaid)**.

Key patterns: Decorator (FibCalculator), Factory+Registry (DefaultFactory), Strategy+ISP (Multiplier/DoublingStepExecutor), Framework (DoublingFramework/MatrixFramework), Observer (ProgressSubject), Object Pooling, Bump Allocator, FFT Transform Cache, Dynamic Threshold Adjustment, Zero-Copy Result Return, Interface-Based Decoupling, Generics with Pointer Constraints, Calculation Arena, GC Controller.

## Performance Considerations

1. **Zero-Allocation**: Object pools avoid allocations in critical loops
2. **Smart Parallelism**: Enabled only when beneficial
3. **2-Tier Multiplication**: FFT or standard math/big selected by operand size
4. **Strassen**: Enabled for matrices with large elements
5. **Symmetric Squaring**: Specific optimization reducing multiplications

For benchmarks and profiling details, see [../PERFORMANCE.md](../PERFORMANCE.md).

## Extensibility

To add a new algorithm:

1. Implement the `coreCalculator` interface (`CalculateCore`, `Name`) in `internal/fibonacci/`
2. Register in `NewDefaultFactory()` in `registry.go`
3. Add corresponding tests (table-driven + golden file validation)
